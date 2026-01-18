package debian

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/blakesmith/ar"
	"go.uber.org/zap"

	"github.com/zjpiazza/nplb/internal/models"
)

// Service provides methods for parsing Debian packages and generating repository metadata.
type Service struct {
	logger *zap.Logger
}

// NewService creates a new Debian service.
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger: logger,
	}
}

// ParseDebFile parses a .deb file and extracts its control information.
func (s *Service) ParseDebFile(filePath string) (*models.DebInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open deb file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat deb file: %w", err)
	}

	// Calculate checksums
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read deb file: %w", err)
	}

	md5sum := fmt.Sprintf("%x", md5.Sum(content))
	sha256sum := fmt.Sprintf("%x", sha256.Sum256(content))

	// Reset file position for ar reading
	file.Seek(0, 0)

	// Read the ar archive
	arReader := ar.NewReader(file)

	var controlData []byte

	for {
		header, err := arReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read ar archive: %w", err)
		}

		// Look for control.tar.gz, control.tar.xz, or control.tar.zst
		name := strings.TrimSuffix(header.Name, "/")
		if strings.HasPrefix(name, "control.tar") {
			data, err := io.ReadAll(arReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read control archive: %w", err)
			}

			controlData, err = extractControlFile(name, data)
			if err != nil {
				return nil, fmt.Errorf("failed to extract control file: %w", err)
			}
			break
		}
	}

	if controlData == nil {
		return nil, fmt.Errorf("control file not found in deb package")
	}

	// Parse the control file
	info, err := parseControlFile(controlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse control file: %w", err)
	}

	// Add file metadata
	info.Size = fileInfo.Size()
	info.MD5sum = md5sum
	info.SHA256 = sha256sum
	info.Filename = filepath.Base(filePath)

	return info, nil
}

// extractControlFile extracts the control file from a compressed tar archive.
func extractControlFile(archiveName string, data []byte) ([]byte, error) {
	var reader io.Reader = bytes.NewReader(data)
	var err error

	// Handle different compression formats
	if strings.HasSuffix(archiveName, ".gz") {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
	} else if strings.HasSuffix(archiveName, ".xz") {
		// For xz, we'd need an external library like github.com/ulikunitz/xz
		// For now, fall through and try to read as uncompressed
		return nil, fmt.Errorf("xz compression not yet supported")
	} else if strings.HasSuffix(archiveName, ".zst") {
		return nil, fmt.Errorf("zstd compression not yet supported")
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for the control file
		name := strings.TrimPrefix(header.Name, "./")
		if name == "control" {
			return io.ReadAll(tarReader)
		}
	}

	return nil, fmt.Errorf("control file not found in tar archive")
}

// parseControlFile parses the Debian control file format.
func parseControlFile(data []byte) (*models.DebInfo, error) {
	info := &models.DebInfo{}
	fields := make(map[string]string)

	lines := strings.Split(string(data), "\n")
	var currentKey string
	var currentValue strings.Builder

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Continuation line (starts with space or tab)
		if line[0] == ' ' || line[0] == '\t' {
			if currentKey != "" {
				currentValue.WriteString("\n")
				currentValue.WriteString(strings.TrimLeft(line, " \t"))
			}
			continue
		}

		// Save previous field
		if currentKey != "" {
			fields[currentKey] = currentValue.String()
		}

		// Parse new field
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			currentKey = strings.TrimSpace(parts[0])
			currentValue.Reset()
			currentValue.WriteString(strings.TrimSpace(parts[1]))
		}
	}

	// Save last field
	if currentKey != "" {
		fields[currentKey] = currentValue.String()
	}

	// Map fields to struct
	info.Package = fields["Package"]
	info.Version = fields["Version"]
	info.Architecture = fields["Architecture"]
	info.Maintainer = fields["Maintainer"]
	info.Description = fields["Description"]
	info.Depends = fields["Depends"]
	info.Section = fields["Section"]
	info.Priority = fields["Priority"]
	info.Homepage = fields["Homepage"]

	if info.Package == "" {
		return nil, fmt.Errorf("package name not found in control file")
	}

	return info, nil
}

// GeneratePackagesFile generates the Packages file content for a list of packages.
func (s *Service) GeneratePackagesFile(packages []*models.DebInfo, poolPath string) string {
	var buf strings.Builder

	for _, pkg := range packages {
		buf.WriteString(fmt.Sprintf("Package: %s\n", pkg.Package))
		buf.WriteString(fmt.Sprintf("Version: %s\n", pkg.Version))
		buf.WriteString(fmt.Sprintf("Architecture: %s\n", pkg.Architecture))

		if pkg.Maintainer != "" {
			buf.WriteString(fmt.Sprintf("Maintainer: %s\n", pkg.Maintainer))
		}
		if pkg.Depends != "" {
			buf.WriteString(fmt.Sprintf("Depends: %s\n", pkg.Depends))
		}
		if pkg.Section != "" {
			buf.WriteString(fmt.Sprintf("Section: %s\n", pkg.Section))
		}
		if pkg.Priority != "" {
			buf.WriteString(fmt.Sprintf("Priority: %s\n", pkg.Priority))
		}
		if pkg.Homepage != "" {
			buf.WriteString(fmt.Sprintf("Homepage: %s\n", pkg.Homepage))
		}

		buf.WriteString(fmt.Sprintf("Filename: %s/%s\n", poolPath, pkg.Filename))
		buf.WriteString(fmt.Sprintf("Size: %d\n", pkg.Size))
		buf.WriteString(fmt.Sprintf("MD5sum: %s\n", pkg.MD5sum))
		buf.WriteString(fmt.Sprintf("SHA256: %s\n", pkg.SHA256))

		if pkg.Description != "" {
			// Format description: first line, then subsequent lines with space prefix
			lines := strings.Split(pkg.Description, "\n")
			buf.WriteString(fmt.Sprintf("Description: %s\n", lines[0]))
			for _, line := range lines[1:] {
				if line == "" {
					buf.WriteString(" .\n")
				} else {
					buf.WriteString(fmt.Sprintf(" %s\n", line))
				}
			}
		}

		buf.WriteString("\n")
	}

	return buf.String()
}

// GenerateReleaseFile generates the Release file content.
func (s *Service) GenerateReleaseFile(opts ReleaseOptions, files []ReleaseFileInfo) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("Origin: %s\n", opts.Origin))
	buf.WriteString(fmt.Sprintf("Label: %s\n", opts.Label))
	buf.WriteString(fmt.Sprintf("Suite: %s\n", opts.Suite))
	buf.WriteString(fmt.Sprintf("Codename: %s\n", opts.Codename))
	buf.WriteString(fmt.Sprintf("Architectures: %s\n", strings.Join(opts.Architectures, " ")))
	buf.WriteString(fmt.Sprintf("Components: %s\n", strings.Join(opts.Components, " ")))
	buf.WriteString(fmt.Sprintf("Date: %s\n", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 MST")))

	if opts.Description != "" {
		buf.WriteString(fmt.Sprintf("Description: %s\n", opts.Description))
	}

	// Sort files for consistent output
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// MD5Sum section
	buf.WriteString("MD5Sum:\n")
	for _, f := range files {
		buf.WriteString(fmt.Sprintf(" %s %16d %s\n", f.MD5, f.Size, f.Path))
	}

	// SHA256 section
	buf.WriteString("SHA256:\n")
	for _, f := range files {
		buf.WriteString(fmt.Sprintf(" %s %16d %s\n", f.SHA256, f.Size, f.Path))
	}

	return buf.String()
}

// ReleaseOptions contains options for generating a Release file.
type ReleaseOptions struct {
	Origin        string
	Label         string
	Suite         string
	Codename      string
	Architectures []string
	Components    []string
	Description   string
}

// ReleaseFileInfo contains information about a file in the Release file.
type ReleaseFileInfo struct {
	Path   string
	Size   int64
	MD5    string
	SHA256 string
}

// CalculateFileHashes calculates MD5 and SHA256 hashes for file content.
func CalculateFileHashes(content []byte) (md5sum, sha256sum string) {
	md5sum = fmt.Sprintf("%x", md5.Sum(content))
	sha256sum = fmt.Sprintf("%x", sha256.Sum256(content))
	return
}

// CompressGzip compresses data using gzip.
func CompressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
