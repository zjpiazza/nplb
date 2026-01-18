package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/zjpiazza/nplb/internal/models"
)

// Service provides methods for interacting with the GitHub API.
type Service struct {
	client     *github.Client
	httpClient *http.Client
	logger     *zap.Logger
}

// NewService creates a new GitHub service.
func NewService(token string, logger *zap.Logger) *Service {
	var client *github.Client
	var httpClient *http.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(context.Background(), ts)
		client = github.NewClient(httpClient)
	} else {
		httpClient = http.DefaultClient
		client = github.NewClient(nil)
	}

	return &Service{
		client:     client,
		httpClient: httpClient,
		logger:     logger,
	}
}

// GetReleases fetches releases for a given repository with proper pagination.
func (s *Service) GetReleases(ctx context.Context, owner, repo string, limit int) ([]models.Release, error) {
	var allReleases []models.Release
	opts := &github.ListOptions{
		PerPage: 100, // Max per page
	}

	for {
		releases, resp, err := s.client.Repositories.ListReleases(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases: %w", err)
		}

		for _, release := range releases {
			// Skip drafts and prereleases if needed
			if release.GetDraft() {
				continue
			}

			var assets []models.DebAsset

			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.GetName(), ".deb") {
					assets = append(assets, models.DebAsset{
						Name:        asset.GetName(),
						DownloadURL: asset.GetBrowserDownloadURL(),
						Size:        int64(asset.GetSize()),
					})
				}
			}

			// Only include releases with .deb assets
			if len(assets) > 0 {
				allReleases = append(allReleases, models.Release{
					TagName:     release.GetTagName(),
					Name:        release.GetName(),
					PublishedAt: release.PublishedAt.GetTime(),
					Assets:      assets,
				})
			}

			// Check if we've reached the limit
			if limit > 0 && len(allReleases) >= limit {
				return allReleases[:limit], nil
			}
		}

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allReleases, nil
}

// GetLatestRelease fetches the latest release for a repository.
func (s *Service) GetLatestRelease(ctx context.Context, owner, repo string) (*models.Release, error) {
	release, _, err := s.client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	var assets []models.DebAsset
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.GetName(), ".deb") {
			assets = append(assets, models.DebAsset{
				Name:        asset.GetName(),
				DownloadURL: asset.GetBrowserDownloadURL(),
				Size:        int64(asset.GetSize()),
			})
		}
	}

	return &models.Release{
		TagName:     release.GetTagName(),
		Name:        release.GetName(),
		PublishedAt: release.PublishedAt.GetTime(),
		Assets:      assets,
	}, nil
}

// DownloadAsset downloads a release asset to a local file.
func (s *Service) DownloadAsset(ctx context.Context, asset models.DebAsset, destDir string) (string, error) {
	destPath := filepath.Join(destDir, asset.Name)

	s.logger.Debug("downloading asset",
		zap.String("name", asset.Name),
		zap.String("url", asset.DownloadURL),
		zap.Int64("size", asset.Size),
	)

	// Create the destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Download the file
	req, err := http.NewRequestWithContext(ctx, "GET", asset.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Copy the response body to the file
	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	s.logger.Debug("downloaded asset",
		zap.String("name", asset.Name),
		zap.Int64("bytes", written),
	)

	return destPath, nil
}

// DownloadAssetToBytes downloads a release asset and returns its content as bytes.
func (s *Service) DownloadAssetToBytes(ctx context.Context, asset models.DebAsset) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", asset.DownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ValidateRepository checks if a repository exists and is accessible.
func (s *Service) ValidateRepository(ctx context.Context, owner, repo string) error {
	_, _, err := s.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("repository not found or not accessible: %w", err)
	}
	return nil
}
