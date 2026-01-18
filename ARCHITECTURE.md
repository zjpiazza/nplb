# NPLB Architecture Plan

## Overview

NPLB (No Package Left Behind) is a service that generates Debian APT repositories from GitHub releases containing `.deb` packages. Users can point to a GitHub repo with `.deb` releases, and NPLB will create a proper APT repository hosted on Cloudflare R2.

**Example:** A project like [ghostty-ubuntu](https://github.com/mkasberg/ghostty-ubuntu) releases `.deb` files. NPLB monitors releases and creates an APT repo users can add with:
```bash
curl -fsSL https://repo.example.com/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/example.gpg
echo "deb [signed-by=/etc/apt/keyrings/example.gpg] https://repo.example.com/ghostty stable main" | sudo tee /etc/apt/sources.list.d/example.list
sudo apt update && sudo apt install ghostty
```

---

## Current State

The codebase has scaffolding in place but core logic is incomplete:

| Component | Status |
|-----------|--------|
| Project structure | Done |
| Config (Viper) | Done |
| GitHub service | Done |
| Storage client | Stub |
| Debian parsing | Stub |
| Repository builder | Stub |
| GPG signing | Stub |
| API handlers | Partial |
| Worker | Stub |
| Tests | Missing |

---

## Architecture

```
                                    ┌─────────────────────────────────┐
                                    │         Cloudflare R2           │
                                    │  (APT Repository Storage)       │
                                    │                                 │
                                    │  /owner/repo/                   │
                                    │    ├── dists/stable/            │
                                    │    │   ├── Release              │
                                    │    │   ├── Release.gpg          │
                                    │    │   ├── InRelease            │
                                    │    │   └── main/binary-amd64/   │
                                    │    │       ├── Packages         │
                                    │    │       ├── Packages.gz      │
                                    │    │       └── Release          │
                                    │    ├── pool/main/               │
                                    │    │   └── *.deb                │
                                    │    └── gpg.key                  │
                                    └─────────────────────────────────┘
                                                    ▲
                                                    │ Upload
                                                    │
┌──────────────┐    POST /build     ┌──────────────┴──────────────┐
│   Client     │ ─────────────────▶ │         API Server          │
│  (webhook    │                    │        (Fiber HTTP)         │
│   or manual) │ ◀───────────────── │                             │
└──────────────┘    202 Accepted    └──────────────┬──────────────┘
                    + job ID                       │
                                                   │ Enqueue
                                                   ▼
                                    ┌─────────────────────────────────┐
                                    │            Redis                │
                                    │     (Asynq Task Queue)          │
                                    └─────────────────────────────────┘
                                                   │
                                                   │ Dequeue
                                                   ▼
                                    ┌─────────────────────────────────┐
                                    │           Worker                │
                                    │     (Asynq Processor)           │
                                    │                                 │
                                    │  1. Fetch releases from GitHub  │
                                    │  2. Download .deb files         │
                                    │  3. Parse control metadata      │
                                    │  4. Generate Packages index     │
                                    │  5. Generate Release file       │
                                    │  6. GPG sign Release            │
                                    │  7. Upload to R2                │
                                    └─────────────────────────────────┘
```

---

## Core Components

### 1. API Server (`cmd/api`)

Minimal REST API for triggering builds:

```
POST /api/v1/build
  Body: { "owner": "mkasberg", "repo": "ghostty-ubuntu" }
  Response: { "job_id": "uuid", "status": "queued" }

GET /api/v1/build/:id
  Response: { "job_id": "uuid", "status": "completed|failed|processing", "error": "..." }

GET /health
  Response: { "status": "ok" }

GET /ready
  Response: { "status": "ok", "redis": "connected", "r2": "connected" }
```

### 2. Worker (`cmd/worker`)

Processes repository build jobs from Redis queue:

```go
// Task payload
type BuildTask struct {
    Owner string `json:"owner"`
    Repo  string `json:"repo"`
}

// Processing steps:
// 1. Fetch all releases from GitHub API
// 2. For each release, find .deb assets
// 3. Download .deb files to temp directory
// 4. Parse control file from each .deb
// 5. Generate Packages file (index of all packages)
// 6. Generate Release file (metadata + checksums)
// 7. GPG sign: Release.gpg (detached) and InRelease (inline)
// 8. Upload everything to R2
// 9. Cleanup temp files
```

### 3. Services (`internal/services`)

#### GitHub Service (Done)
- Fetch releases for a repo
- Filter assets ending in `.deb`
- Download asset content

#### Debian Service (TODO)
```go
type DebInfo struct {
    Package      string
    Version      string
    Architecture string
    Maintainer   string
    Description  string
    Depends      string
    Size         int64
    MD5sum       string
    SHA256       string
    Filename     string
}

// ParseDeb extracts control file from .deb archive
func ParseDeb(reader io.Reader) (*DebInfo, error)

// GeneratePackages creates the Packages index file
func GeneratePackages(packages []DebInfo) ([]byte, error)

// GenerateRelease creates the Release metadata file
func GenerateRelease(suite, codename string, files []FileHash) ([]byte, error)
```

#### Storage Service (TODO)
```go
// Upload file to R2
func (s *Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error

// Download file from R2
func (s *Storage) Download(ctx context.Context, key string) (io.ReadCloser, error)

// List files with prefix
func (s *Storage) List(ctx context.Context, prefix string) ([]string, error)

// Delete file
func (s *Storage) Delete(ctx context.Context, key string) error
```

#### GPG Service (TODO)
```go
// Sign creates detached signature (Release.gpg)
func (g *GPG) Sign(data []byte) ([]byte, error)

// ClearSign creates inline signature (InRelease)
func (g *GPG) ClearSign(data []byte) ([]byte, error)

// ExportPublicKey exports ASCII-armored public key
func (g *GPG) ExportPublicKey() ([]byte, error)
```

### 4. Repository Builder (`internal/services/repository`)

Orchestrates the full build process:

```go
type Builder struct {
    github  *github.Service
    debian  *debian.Service
    storage *storage.Service
    gpg     *gpg.Service
    logger  *zap.Logger
}

func (b *Builder) Build(ctx context.Context, owner, repo string) error {
    // 1. Create temp directory
    // 2. Fetch releases and download .debs
    // 3. Parse each .deb, collect metadata
    // 4. Generate Packages, Packages.gz
    // 5. Generate Release
    // 6. Sign Release -> Release.gpg, InRelease
    // 7. Upload pool/*.deb files
    // 8. Upload dists/stable/* files
    // 9. Upload gpg.key
    // 10. Cleanup
}
```

---

## APT Repository Structure

Generated repository follows standard APT layout:

```
/{owner}/{repo}/
├── gpg.key                           # Public GPG key for verification
├── dists/
│   └── stable/                       # Suite name
│       ├── Release                   # Repository metadata
│       ├── Release.gpg               # Detached GPG signature
│       ├── InRelease                 # Inline signed Release
│       └── main/                     # Component
│           └── binary-{arch}/        # Architecture (amd64, arm64, etc.)
│               ├── Packages          # Package index
│               ├── Packages.gz       # Compressed index
│               └── Release           # Component metadata
└── pool/
    └── main/
        ├── package_1.0.0_amd64.deb
        ├── package_1.0.1_amd64.deb
        └── ...
```

### Release File Format
```
Origin: NPLB
Label: {owner}/{repo}
Suite: stable
Codename: stable
Architectures: amd64 arm64
Components: main
Date: {RFC 1123 date}
MD5Sum:
 {md5} {size} main/binary-amd64/Packages
 {md5} {size} main/binary-amd64/Packages.gz
SHA256:
 {sha256} {size} main/binary-amd64/Packages
 {sha256} {size} main/binary-amd64/Packages.gz
```

### Packages File Format
```
Package: ghostty
Version: 1.0.0
Architecture: amd64
Maintainer: Example <example@example.com>
Installed-Size: 12345
Depends: libc6 (>= 2.17)
Filename: pool/main/ghostty_1.0.0_amd64.deb
Size: 5678901
MD5sum: abc123...
SHA256: def456...
Description: A terminal emulator
 Long description goes here.
```

---

## Implementation Plan

### Phase 1: Core Services (Priority: High)

1. **Debian Service** - Parse `.deb` files and generate index files
   - Extract control file from `.deb` (ar archive -> tar.gz -> control)
   - Parse control file key-value format
   - Generate Packages file
   - Generate Release file with checksums

2. **Storage Service** - R2 upload/download
   - Implement `Upload()` with proper content types
   - Implement `Download()` for reading existing packages
   - Implement `List()` for checking existing files

3. **GPG Service** - Signing
   - Load private key from file or environment
   - Create detached signature (Release.gpg)
   - Create clearsign (InRelease)
   - Export public key

### Phase 2: Repository Builder (Priority: High)

4. **Repository Builder** - Orchestration
   - Wire up all services
   - Implement full build flow
   - Handle multiple architectures
   - Handle multiple releases/versions

### Phase 3: API & Worker (Priority: Medium)

5. **API Handlers** - HTTP endpoints
   - POST /build endpoint
   - GET /build/:id status endpoint
   - Input validation

6. **Worker** - Task processing
   - Wire up Asynq handlers
   - Call repository builder
   - Error handling and retries

### Phase 4: Polish (Priority: Low)

7. **CLI Tool** - Manual operations
   - Trigger builds manually
   - Check job status
   - List repositories

8. **Testing** - Unit and integration tests
9. **CI/CD** - GitHub Actions workflow
10. **Documentation** - Usage guide

---

## Configuration

Environment variables (see `.env.example`):

```bash
# Server
HOST=0.0.0.0
PORT=8080
ENV=development
LOG_LEVEL=debug

# GitHub
GITHUB_TOKEN=ghp_xxx              # For private repos or higher rate limits

# Cloudflare R2
R2_ACCOUNT_ID=xxx
R2_ACCESS_KEY_ID=xxx
R2_ACCESS_KEY_SECRET=xxx
R2_BUCKET_NAME=apt-repos
R2_PUBLIC_URL=https://repo.example.com

# Redis (for Asynq)
REDIS_URL=redis://localhost:6379

# GPG
GPG_PRIVATE_KEY_PATH=/path/to/private.key
GPG_PASSPHRASE=xxx
GPG_KEY_ID=ABC123
```

---

## Tech Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Language | Go 1.24 | |
| Web Framework | Fiber v2 | Fast, Express-like API |
| Task Queue | Asynq | Redis-based, reliable |
| GitHub API | go-github v57 | Official client |
| Debian Parsing | go-debian | Native Go library |
| Storage | AWS SDK v2 | S3-compatible for R2 |
| GPG | go-crypto or exec | ProtonMail library or shell |
| Logging | Zap | Structured, fast |
| Config | Viper | Env vars, files |

---

## Getting Started

```bash
# 1. Copy environment config
cp .env.example .env
# Edit .env with your credentials

# 2. Start Redis
docker run -d -p 6379:6379 redis:alpine

# 3. Run API server
go run cmd/api/main.go

# 4. Run worker (separate terminal)
go run cmd/worker/main.go

# 5. Trigger a build
curl -X POST http://localhost:8080/api/v1/build \
  -H "Content-Type: application/json" \
  -d '{"owner": "mkasberg", "repo": "ghostty-ubuntu"}'
```

---

## Next Steps

1. [ ] Implement Debian service (parsing + index generation)
2. [ ] Implement Storage service (R2 uploads)
3. [ ] Implement GPG service (signing)
4. [ ] Implement Repository builder
5. [ ] Wire up API handlers
6. [ ] Wire up Worker handlers
7. [ ] Add tests
8. [ ] Create GitHub Actions CI
