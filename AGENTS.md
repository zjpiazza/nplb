# AGENTS.md - AI Coding Agent Instructions

This document provides guidelines for AI coding agents working on the NPLB (No Package Left Behind) codebase.

## Project Overview

NPLB is a Go service that generates Debian APT repositories from GitHub releases containing `.deb` packages. It monitors GitHub releases and creates proper APT repositories hosted on Cloudflare R2.

**Tech Stack:**
- Language: Go 1.24+
- Web Framework: Fiber v2
- Task Queue: Asynq (Redis-based) + Cloudflare Queues
- Storage: AWS SDK v2 (S3-compatible for Cloudflare R2)
- Configuration: Viper
- Logging: Zap (structured logging)

## Build Commands

```bash
# Build all binaries
go build ./...

# Build specific binaries
go build ./cmd/api
go build ./cmd/worker
go build ./cmd/cli

# Run directly without building
go run cmd/api/main.go
go run cmd/worker/main.go

# Docker build
docker build -f deployments/Dockerfile -t nplb .

# Docker Compose (runs api + worker)
docker-compose -f deployments/docker-compose.yml up
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test by name
go test -run TestFunctionName ./path/to/package

# Run tests for a specific package
go test ./internal/services/github/...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Lint/Format Commands

```bash
# Format code (always run before committing)
go fmt ./...
gofmt -s -w .

# Vet code for common issues
go vet ./...

# Tidy dependencies
go mod tidy
```

## Project Structure

```
cmd/                    # Application entry points
  api/                  # HTTP API server
  worker/               # Background worker
  cli/                  # CLI tool
internal/               # Private application code
  api/handlers/         # HTTP request handlers
  api/middleware/       # Fiber middleware
  api/routes/           # Route definitions
  config/               # Viper configuration
  models/               # Data structures/types
  queue/                # Queue client abstraction
  services/             # Business logic
    debian/             # Debian package parsing
    github/             # GitHub API integration
    repository/         # APT repository builder
    storage/            # R2/S3 storage client
  tasks/                # Asynq task definitions
pkg/                    # Public library code
  gpg/                  # GPG signing utilities
deployments/            # Docker and deployment configs
```

## Code Style Guidelines

### Imports

Order imports in three groups separated by blank lines:
1. Standard library
2. Third-party packages
3. Internal packages (github.com/zjpiazza/nplb/...)

```go
import (
    "context"
    "fmt"

    "github.com/gofiber/fiber/v2"
    "go.uber.org/zap"

    "github.com/zjpiazza/nplb/internal/models"
    "github.com/zjpiazza/nplb/internal/queue"
)
```

### Naming Conventions

- **Files:** lowercase with underscores (`github.go`, `queue_client.go`)
- **Packages:** lowercase, single word preferred (`handlers`, `models`, `queue`)
- **Types:** PascalCase (`BuildRequest`, `DebAsset`, `CloudflareQueue`)
- **Interfaces:** PascalCase, often ending in `-er` (`Queue`, `Storage`)
- **Functions/Methods:** PascalCase for exported, camelCase for unexported
- **Variables:** camelCase (`queueClient`, `httpClient`)
- **Constants:** PascalCase for exported, camelCase for unexported

### Struct Tags

Use appropriate struct tags for serialization and validation:
```go
type BuildRequest struct {
    Owner string `json:"owner" validate:"required"`
    Repo  string `json:"repo" validate:"required"`
    Limit int    `json:"limit" validate:"min=1,max=100"`
}
```

For configuration structs, use `mapstructure` tags:
```go
type Config struct {
    ServerPort int `mapstructure:"SERVER_PORT"`
}
```

### Comments

- Use `// TypeName description.` for type comments
- Use `// FunctionName description.` for function comments
- Comments should be complete sentences ending with periods

```go
// Service provides methods for interacting with the GitHub API.
type Service struct { ... }

// NewService creates a new GitHub service.
func NewService(token string) *Service { ... }
```

### Error Handling

- Always check and handle errors explicitly
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return errors early (guard clauses)
- Use structured logging for error context

```go
if err != nil {
    return nil, fmt.Errorf("failed to marshal message: %w", err)
}
```

### Constructor Pattern

Use `NewXxx` functions as constructors that return pointers:
```go
func NewService(token string) *Service {
    return &Service{
        client: github.NewClient(tc),
    }
}
```

### Interface Design

Define interfaces for abstractions to enable testing and flexibility:
```go
type Queue interface {
    Enqueue(ctx context.Context, taskType string, payload interface{}) error
    Close() error
}
```

### Context Usage

- Pass `context.Context` as the first parameter to functions that do I/O
- Use `context.Background()` only at application boundaries
- Respect context cancellation in long-running operations

### HTTP Handlers (Fiber)

Return appropriate status codes and JSON responses:
```go
func (h *Handler) CreateBuild(c *fiber.Ctx) error {
    var req models.BuildRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "cannot parse request body",
        })
    }
    // ... handle request
    return c.Status(fiber.StatusAccepted).JSON(response)
}
```

### Logging

Use Zap structured logging with appropriate levels:
```go
h.logger.Error("failed to enqueue task", zap.Error(err))
h.logger.Info("task completed", zap.String("job_id", jobID))
```

## Configuration

Environment variables are loaded via Viper. See `.env.example` for all options.
Always provide sensible defaults in the config loader.

## Dependencies

Key dependencies to be aware of:
- `github.com/gofiber/fiber/v2` - HTTP framework
- `github.com/hibiken/asynq` - Redis task queue
- `go.uber.org/zap` - Structured logging
- `github.com/spf13/viper` - Configuration
- `github.com/google/go-github/v57` - GitHub API client
- `pault.ag/go/debian` - Debian package parsing
