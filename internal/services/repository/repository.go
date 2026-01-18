package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/zjpiazza/nplb/internal/models"
	"github.com/zjpiazza/nplb/internal/services/debian"
	"github.com/zjpiazza/nplb/internal/services/github"
	"github.com/zjpiazza/nplb/internal/services/storage"
	"github.com/zjpiazza/nplb/pkg/gpg"
)

// Builder orchestrates the building of Debian APT repositories.
type Builder struct {
	github    *github.Service
	debian    *debian.Service
	storage   storage.Storage
	gpg       *gpg.Signer
	logger    *zap.Logger
	outputDir string
	codename  string
}

// NewBuilder creates a new repository builder.
func NewBuilder(
	githubSvc *github.Service,
	debianSvc *debian.Service,
	storageSvc storage.Storage,
	gpgSigner *gpg.Signer,
	logger *zap.Logger,
	outputDir string,
	codename string,
) *Builder {
	return &Builder{
		github:    githubSvc,
		debian:    debianSvc,
		storage:   storageSvc,
		gpg:       gpgSigner,
		logger:    logger,
		outputDir: outputDir,
		codename:  codename,
	}
}

// BuildOptions contains options for building a repository.
type BuildOptions struct {
	Owner       string
	Repo        string
	Limit       int
	Component   string
	Origin      string
	Label       string
	Description string
}

// BuildResult contains the result of a repository build.
type BuildResult struct {
	PackagesBuilt int
	Architectures []string
	RepoURL       string
}

// Build builds a Debian repository from GitHub releases.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	b.logger.Info("starting repository build",
		zap.String("owner", opts.Owner),
		zap.String("repo", opts.Repo),
		zap.Int("limit", opts.Limit),
	)

	// Set defaults
	if opts.Component == "" {
		opts.Component = "main"
	}
	if opts.Origin == "" {
		opts.Origin = fmt.Sprintf("%s/%s", opts.Owner, opts.Repo)
	}
	if opts.Label == "" {
		opts.Label = opts.Repo
	}

	// Create temporary directory for build
	tmpDir, err := os.MkdirTemp(b.outputDir, "build-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 1: Fetch releases from GitHub
	b.logger.Info("fetching releases from GitHub")
	releases, err := b.github.GetReleases(ctx, opts.Owner, opts.Repo, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found with .deb assets")
	}

	b.logger.Info("found releases", zap.Int("count", len(releases)))

	// Step 2: Download and parse all .deb files
	packagesByArch := make(map[string][]*models.DebInfo)
	poolDir := filepath.Join(tmpDir, "pool", opts.Component)
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create pool directory: %w", err)
	}

	for _, release := range releases {
		for _, asset := range release.Assets {
			b.logger.Debug("processing asset",
				zap.String("name", asset.Name),
				zap.String("release", release.TagName),
			)

			// Download the .deb file
			debPath, err := b.github.DownloadAsset(ctx, asset, poolDir)
			if err != nil {
				b.logger.Warn("failed to download asset",
					zap.String("name", asset.Name),
					zap.Error(err),
				)
				continue
			}

			// Parse the .deb file
			debInfo, err := b.debian.ParseDebFile(debPath)
			if err != nil {
				b.logger.Warn("failed to parse deb file",
					zap.String("name", asset.Name),
					zap.Error(err),
				)
				continue
			}

			// Group by architecture
			arch := debInfo.Architecture
			if arch == "" {
				arch = "all"
			}
			packagesByArch[arch] = append(packagesByArch[arch], debInfo)

			b.logger.Debug("parsed package",
				zap.String("package", debInfo.Package),
				zap.String("version", debInfo.Version),
				zap.String("arch", arch),
			)
		}
	}

	if len(packagesByArch) == 0 {
		return nil, fmt.Errorf("no valid packages found")
	}

	// Step 3: Generate repository structure and upload
	architectures := make([]string, 0, len(packagesByArch))
	for arch := range packagesByArch {
		architectures = append(architectures, arch)
	}

	// Track files for Release file
	var releaseFiles []debian.ReleaseFileInfo

	// Step 4: Generate and upload Packages files for each architecture
	for arch, packages := range packagesByArch {
		b.logger.Info("generating Packages file",
			zap.String("arch", arch),
			zap.Int("packages", len(packages)),
		)

		// Pool path relative to repo root
		poolPath := fmt.Sprintf("pool/%s", opts.Component)
		packagesContent := b.debian.GeneratePackagesFile(packages, poolPath)

		// Upload Packages file (uncompressed)
		packagesPath := fmt.Sprintf("dists/%s/%s/binary-%s/Packages",
			b.codename, opts.Component, arch)

		if err := b.storage.Upload(ctx, packagesPath, []byte(packagesContent), "text/plain"); err != nil {
			return nil, fmt.Errorf("failed to upload Packages: %w", err)
		}

		md5, sha256 := debian.CalculateFileHashes([]byte(packagesContent))
		releaseFiles = append(releaseFiles, debian.ReleaseFileInfo{
			Path:   fmt.Sprintf("%s/binary-%s/Packages", opts.Component, arch),
			Size:   int64(len(packagesContent)),
			MD5:    md5,
			SHA256: sha256,
		})

		// Upload compressed Packages.gz
		packagesGz, err := debian.CompressGzip([]byte(packagesContent))
		if err != nil {
			return nil, fmt.Errorf("failed to compress Packages: %w", err)
		}

		packagesGzPath := packagesPath + ".gz"
		if err := b.storage.Upload(ctx, packagesGzPath, packagesGz, "application/gzip"); err != nil {
			return nil, fmt.Errorf("failed to upload Packages.gz: %w", err)
		}

		md5Gz, sha256Gz := debian.CalculateFileHashes(packagesGz)
		releaseFiles = append(releaseFiles, debian.ReleaseFileInfo{
			Path:   fmt.Sprintf("%s/binary-%s/Packages.gz", opts.Component, arch),
			Size:   int64(len(packagesGz)),
			MD5:    md5Gz,
			SHA256: sha256Gz,
		})
	}

	// Step 5: Upload .deb files to pool
	for arch, packages := range packagesByArch {
		for _, pkg := range packages {
			localPath := filepath.Join(poolDir, pkg.Filename)
			remotePath := fmt.Sprintf("pool/%s/%s", opts.Component, pkg.Filename)

			content, err := os.ReadFile(localPath)
			if err != nil {
				b.logger.Warn("failed to read deb file",
					zap.String("file", pkg.Filename),
					zap.Error(err),
				)
				continue
			}

			if err := b.storage.Upload(ctx, remotePath, content, "application/vnd.debian.binary-package"); err != nil {
				b.logger.Warn("failed to upload deb file",
					zap.String("file", pkg.Filename),
					zap.Error(err),
				)
				continue
			}

			b.logger.Debug("uploaded package",
				zap.String("package", pkg.Package),
				zap.String("arch", arch),
			)
		}
	}

	// Step 6: Generate Release file
	releaseOpts := debian.ReleaseOptions{
		Origin:        opts.Origin,
		Label:         opts.Label,
		Suite:         b.codename,
		Codename:      b.codename,
		Architectures: architectures,
		Components:    []string{opts.Component},
		Description:   opts.Description,
	}

	releaseContent := b.debian.GenerateReleaseFile(releaseOpts, releaseFiles)

	releasePath := fmt.Sprintf("dists/%s/Release", b.codename)
	if err := b.storage.Upload(ctx, releasePath, []byte(releaseContent), "text/plain"); err != nil {
		return nil, fmt.Errorf("failed to upload Release: %w", err)
	}

	// Step 7: Sign Release file (if GPG is configured)
	if b.gpg != nil && b.gpg.KeyExists() {
		b.logger.Info("signing Release file")

		// Create Release.gpg (detached signature)
		releaseGpg, err := b.gpg.SignDetachedToBytes([]byte(releaseContent))
		if err != nil {
			b.logger.Warn("failed to create detached signature", zap.Error(err))
		} else {
			releaseGpgPath := fmt.Sprintf("dists/%s/Release.gpg", b.codename)
			if err := b.storage.Upload(ctx, releaseGpgPath, releaseGpg, "application/pgp-signature"); err != nil {
				b.logger.Warn("failed to upload Release.gpg", zap.Error(err))
			}
		}

		// Create InRelease (clearsigned)
		inRelease, err := b.gpg.SignClearToBytes([]byte(releaseContent))
		if err != nil {
			b.logger.Warn("failed to create clearsigned Release", zap.Error(err))
		} else {
			inReleasePath := fmt.Sprintf("dists/%s/InRelease", b.codename)
			if err := b.storage.Upload(ctx, inReleasePath, inRelease, "text/plain"); err != nil {
				b.logger.Warn("failed to upload InRelease", zap.Error(err))
			}
		}

		// Export and upload public key
		pubKey, err := b.gpg.ExportPublicKey()
		if err != nil {
			b.logger.Warn("failed to export public key", zap.Error(err))
		} else {
			pubKeyPath := "public.gpg"
			if err := b.storage.Upload(ctx, pubKeyPath, pubKey, "application/pgp-keys"); err != nil {
				b.logger.Warn("failed to upload public key", zap.Error(err))
			}
		}
	} else {
		b.logger.Warn("GPG signing skipped - no key configured")
	}

	// Count total packages
	totalPackages := 0
	for _, packages := range packagesByArch {
		totalPackages += len(packages)
	}

	result := &BuildResult{
		PackagesBuilt: totalPackages,
		Architectures: architectures,
		RepoURL:       b.storage.GetPublicURL(fmt.Sprintf("dists/%s", b.codename)),
	}

	b.logger.Info("repository build completed",
		zap.Int("packages", result.PackagesBuilt),
		zap.Strings("architectures", result.Architectures),
		zap.String("url", result.RepoURL),
	)

	return result, nil
}

// GenerateAptSourcesLine generates the apt sources.list line for this repository.
func (b *Builder) GenerateAptSourcesLine(repoURL, component string) string {
	// Remove trailing slash and path components after the base URL
	baseURL := strings.TrimSuffix(repoURL, "/")
	idx := strings.Index(baseURL, "/dists/")
	if idx != -1 {
		baseURL = baseURL[:idx]
	}

	return fmt.Sprintf("deb [signed-by=/etc/apt/keyrings/%s.gpg] %s %s %s",
		b.codename, baseURL, b.codename, component)
}
