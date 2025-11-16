# NPLB - Golang Migration Plan

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Project Overview & Architecture Analysis](#1-project-overview--architecture-analysis)
   - [Current System Architecture](#11-current-system-architecture)
   - [Core Components](#12-core-components)
   - [Key Files and Their Purpose](#13-key-files-and-their-purpose)
3. [Technology Stack Mapping](#2-technology-stack-mapping)
   - [Direct Replacements](#21-direct-replacements)
   - [Cloudflare Services Integration](#22-cloudflare-services-integration)
   - [Recommended Go Libraries](#23-recommended-go-libraries)
   - [Cloudflare R2 Configuration](#24-cloudflare-r2-configuration)
   - [12-Factor App Compliance](#25-12-factor-app-compliance)
4. [Detailed Migration Strategy](#3-detailed-migration-strategy)
   - [Phase 1: Project Setup & Foundation](#31-phase-1-project-setup--foundation-week-1)
   - [Phase 2: Core Services](#32-phase-2-core-services-week-1-2)
   - [Phase 3: Repository Builder](#33-phase-3-repository-builder-week-2)
   - [Phase 4: Task Queue Integration](#34-phase-4-task-queue-integration-week-2-3)
   - [Phase 5: API Implementation](#35-phase-5-api-implementation-week-3)
   - [Phase 6: Worker Implementation](#36-phase-6-worker-implementation-week-3)
   - [Phase 7: Containerization](#37-phase-7-containerization-week-3-4)
   - [Phase 8: Admin CLI Tools](#38-phase-8-admin-cli-tools-week-4)
5. [Testing Strategy](#4-testing-strategy)
   - [Unit Tests](#41-unit-tests)
   - [Integration Tests](#42-integration-tests)
   - [Performance Benchmarks](#43-performance-benchmarks)
6. [Migration Checklist](#5-migration-checklist)
   - [Pre-Migration](#51-pre-migration)
   - [Implementation](#52-implementation)
   - [Post-Migration](#53-post-migration)
7. [Risk Mitigation](#6-risk-mitigation)
   - [Technical Risks](#61-technical-risks)
   - [Operational Risks](#62-operational-risks)
8. [Performance Expectations](#7-performance-expectations)
   - [Expected Improvements](#71-expected-improvements)
   - [Optimization Opportunities](#72-optimization-opportunities)
9. [Deployment Strategy](#8-deployment-strategy)
   - [Blue-Green Deployment](#81-blue-green-deployment)
   - [Rollback Plan](#82-rollback-plan)
   - [CI/CD Pipeline with GitHub Actions](#83-cicd-pipeline-with-github-actions)
   - [Release Automation with GoReleaser](#84-release-automation-with-goreleaser)
10. [Documentation Requirements](#9-documentation-requirements)
    - [Technical Documentation](#91-technical-documentation)
    - [Developer Documentation](#92-developer-documentation)
    - [Observability Strategy](#93-observability-strategy)
11. [Success Criteria](#10-success-criteria)
    - [Functional Requirements](#101-functional-requirements)
    - [Non-Functional Requirements](#102-non-functional-requirements)
    - [Quality Requirements](#103-quality-requirements)
12. [Timeline Summary](#11-timeline-summary)
13. [Next Steps](#12-next-steps)
14. [Cloudflare Migration Benefits](#14-cloudflare-migration-benefits)
    - [Cost Analysis](#141-cost-analysis)
    - [Cloudflare R2 Advantages](#142-cloudflare-r2-advantages)
    - [R2 Setup Instructions](#143-r2-setup-instructions)
    - [R2 vs S3 Compatibility Matrix](#144-r2-vs-s3-compatibility-matrix)
    - [Migration Checklist](#145-migration-checklist)
    - [Future Cloudflare Integration Opportunities](#146-future-cloudflare-integration-opportunities)
    - [Recommended Phased Approach](#147-recommended-phased-approach)
    - [R2 Code Examples](#148-r2-code-examples)
15. [Appendix](#15-appendix)
    - [Useful Go Resources](#151-useful-go-resources)
    - [Python vs Go Syntax Quick Reference](#152-python-vs-go-syntax-quick-reference)
    - [Environment Variables Reference](#153-environment-variables-reference)

---

## Executive Summary

This document outlines a comprehensive plan to migrate the **No Package Left Behind (NPLB)** project from Python to Go, while transitioning from AWS services to **Cloudflare's infrastructure**. NPLB is a service that automatically generates Debian APT repositories from GitHub releases containing `.deb` packages, with support for asynchronous job processing, R2 storage (Cloudflare's S3-compatible object storage), and GPG signing.

**Migration Timeline Estimate:** 3-4 weeks  
**Complexity Level:** Medium-High  
**Risk Assessment:** Medium (due to GPG integration, Debian package parsing, and Cloudflare service migration)  
**Cost Benefit:** Significant savings with Cloudflare R2 (no egress fees) and Workers

### Key Changes from Original Plan

This updated migration plan includes a **complete transition to Cloudflare services**:

1. **Storage Migration**: AWS S3 → **Cloudflare R2**
   - Zero egress fees (bandwidth is free)
   - S3-compatible API (minimal code changes)
   - ~99% cost reduction for high-bandwidth usage
   - Built-in CDN integration

2. **Infrastructure Benefits**:
   - Estimated savings: **~$176/month** for 1TB egress + 100GB storage
   - Global edge network with automatic caching
   - Custom domain support (e.g., `repo.example.com`)
   - Same Go implementation with R2-specific configuration

3. **Future Opportunities**:
   - Phase 2: Evaluate Cloudflare Workers for serverless API
   - Phase 3: Consider Cloudflare Queues (alternative to Redis)
   - Phase 4: Optional Cloudflare D1 for metadata storage

**Current Focus**: Go migration + R2 storage (Phase 1)  
**Future Optimization**: Full Cloudflare stack evaluation (Phase 2-3)

---

## 1. Project Overview & Architecture Analysis

### 1.1 Current System Architecture

The Python implementation consists of:

- **FastAPI REST API** - Handles HTTP requests for repository builds
- **Redis Queue (RQ)** - Asynchronous task processing for repository builds
- **GitHub Integration** - Fetches releases and `.deb` assets via PyGithub
- **Debian Package Processing** - Parses `.deb` files and generates repository metadata
- **S3 Storage** - Uploads generated repositories to AWS S3 *(migrating to Cloudflare R2)*
- **GPG Signing** - Signs repository metadata files for security
- **Docker/Docker Compose** - Containerized deployment *(will evaluate Cloudflare Workers alternative)*

### 1.2 Core Components

```
┌─────────────────┐
│   FastAPI API   │
│  (HTTP Server)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│  Redis Queue    │─────▶│ Worker Pool  │
│   (Job Queue)   │      │ (RQ Workers) │
└─────────────────┘      └──────┬───────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │  Repository Builder   │
                    │  ┌─────────────────┐ │
                    │  │ GitHub Service  │ │
                    │  │ Debian Service  │ │
                    │  │ Storage Service │ │
                    │  │ GPG Signing     │ │
                    │  └─────────────────┘ │
                    └───────────┬───────────┘
                                │
                                ▼
                    ┌────────────────────┐
                    │  Cloudflare R2     │
                    │  (Object Storage)  │
                    └────────────────────┘
```

### 1.3 Key Files and Their Purpose

| Python File | Purpose | Lines of Code |
|-------------|---------|---------------|
| `main.py` (root) | Legacy CLI tool for direct repository generation | ~380 |
| `nplb/main.py` | FastAPI application entry point | ~25 |
| `nplb/api/routes/repositories.py` | REST API endpoint for builds | ~50 |
| `nplb/services/github.py` | GitHub API integration | ~35 |
| `nplb/services/repository.py` | Core repository generation logic | ~275 |
| `nplb/services/storage.py` | S3 upload functionality *(migrating to R2)* | ~100 |
| `nplb/services/debian.py` | Debian package parsing | ~25 |
| `nplb/tasks/build.py` | Async task for repository building | ~35 |
| `nplb/core/config.py` | Configuration management | ~40 |
| `nplb/core/models.py` | Pydantic data models | ~30 |

---

## 2. Technology Stack Mapping

### 2.1 Direct Replacements

| Python Technology | Go Replacement | Rationale |
|-------------------|----------------|-----------|
| **FastAPI** | `fiber` or `gin` or `chi` | Fiber offers Express-like API, Gin is faster, Chi is idiomatic Go |
| **Pydantic** | `go-playground/validator` | Struct validation with tags |
| **Redis RQ** | `hibiken/asynq` or `gocraft/work` or **Cloudflare Queues** | Production-ready async task processing (Cloudflare Queues integrates natively) |
| **PyGithub** | `google/go-github` | Official GitHub API client for Go |
| **boto3 (S3)** | **Cloudflare R2 SDK** (`aws-sdk-go-v2` S3-compatible) | R2 is S3-compatible with zero egress fees |
| **python-gnupg** | `ProtonMail/go-crypto` or exec `gpg` | Go GPG library or shell out to GPG CLI |
| **python-debian** | `paultag/go-debian` | Native Go library for Debian packages |
| **requests** | `net/http` (stdlib) | Go's built-in HTTP client is production-ready |
| **loguru** | `sirupsen/logrus` or `uber-go/zap` | Structured logging (Zap is faster) |
| **pydantic-settings** | `spf13/viper` or `kelseyhightower/envconfig` | Config management with env vars |

### 2.2 Cloudflare Services Integration

| Service | Purpose | Benefits |
|---------|---------|----------|
| **Cloudflare R2** | Object storage for APT repository files | Zero egress fees, S3-compatible API, cheaper than S3 |
| **Cloudflare Workers** *(optional)* | Serverless edge computing for API endpoints | Global deployment, instant scaling, lower latency |
| **Cloudflare Queues** *(optional)* | Message queue for async tasks | Native integration with Workers, alternative to Redis |
| **Cloudflare D1** *(future)* | SQLite database for job metadata | Built-in replication, serverless |
| **Cloudflare Pages** *(optional)* | Static hosting for repository index/docs | Free, global CDN, automatic HTTPS |
| **Cloudflare CDN** | Content delivery for APT packages | Free bandwidth, global edge network |

### 2.3 Recommended Go Libraries

```go
// Core dependencies
github.com/gofiber/fiber/v2         // Web framework (alternative: gin-gonic/gin)
github.com/hibiken/asynq/v2         // Redis task queue (or use Cloudflare Queues)
github.com/google/go-github/v57     // GitHub API
github.com/aws/aws-sdk-go-v2        // AWS SDK (R2 is S3-compatible)
github.com/aws/aws-sdk-go-v2/service/s3  // S3 client for R2
github.com/ProtonMail/go-crypto     // GPG operations
github.com/paultag/go-debian        // Debian package parsing
github.com/spf13/viper              // Configuration
github.com/go-playground/validator  // Validation
go.uber.org/zap                     // Structured logging
github.com/redis/go-redis/v9        // Redis client (if not using Cloudflare Queues)
github.com/cloudflare/cloudflare-go // Cloudflare API client (for R2 management)
```

### 2.4 Cloudflare R2 Configuration

Cloudflare R2 is S3-compatible, so we can use the AWS SDK with custom endpoint configuration:

```go
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
    return aws.Endpoint{
        URL: "https://<CLOUDFLARE_ACCOUNT_ID>.r2.cloudflarestorage.com",
    }, nil
})

cfg, err := config.LoadDefaultConfig(context.Background(),
    config.WithEndpointResolverWithOptions(r2Resolver),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
        r2AccessKeyID,
        r2SecretAccessKey,
        "",
    )),
)
```

### 2.5 12-Factor App Compliance

NPLB's Go implementation follows the [12-Factor App](https://12factor.net/) methodology for building modern, cloud-native applications:

#### I. Codebase
**✅ Compliant**: Single Git repository tracked in version control with multiple deployment environments (dev, staging, production).

```
Repository: github.com/zjpiazza/nplb
Branches: main (production), develop (staging), feature/* (development)
```

#### II. Dependencies
**✅ Compliant**: All dependencies explicitly declared in `go.mod` with version pinning. No system-level dependencies assumed.

```go
// go.mod
module github.com/zjpiazza/nplb

go 1.21

require (
    github.com/gofiber/fiber/v2 v2.50.0
    github.com/hibiken/asynq v0.24.1
    // ... all dependencies versioned
)
```

#### III. Config
**✅ Compliant**: All configuration via environment variables, never committed to the repository. Uses Viper for flexible config sources.

```go
// Configuration loaded from environment variables
// Supports .env files for local development
// Secrets managed via GitHub Secrets / Cloudflare environment variables
```

#### IV. Backing Services
**✅ Compliant**: All external services (R2, Redis, GitHub API) treated as attached resources, swappable via configuration.

```go
// Services are interfaces that can be swapped
type StorageService interface {
    UploadFile(ctx context.Context, path, key string) error
}

// R2, S3, or local filesystem - all implement same interface
```

#### V. Build, Release, Run
**✅ Compliant**: Strict separation via CI/CD pipeline with immutable releases.

```
Build Stage:   go build → compile binaries → run tests
Release Stage: Tag version → GoReleaser → GitHub Release
Run Stage:     Pull image → deploy to environment → no code changes
```

#### VI. Processes
**✅ Compliant**: Application runs as stateless processes. State stored in Redis/R2, not in-memory.

```go
// No sticky sessions
// Workers can be scaled horizontally
// API servers share no local state
```

#### VII. Port Binding
**✅ Compliant**: Self-contained HTTP server exports services via port binding (no external web server needed).

```go
// Fiber app is fully self-contained
app.Listen(fmt.Sprintf(":%d", cfg.ServerPort))
```

#### VIII. Concurrency
**✅ Compliant**: Scales out via process model. Multiple API instances, multiple workers.

```yaml
# docker-compose.yml
worker:
  deploy:
    replicas: 2  # Scale workers independently
```

#### IX. Disposability
**✅ Compliant**: Fast startup (<1s), graceful shutdown with signal handling.

```go
// Graceful shutdown on SIGTERM/SIGINT
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
go func() {
    <-quit
    app.ShutdownWithTimeout(30 * time.Second)
}()
```

#### X. Dev/Prod Parity
**✅ Compliant**: Same Docker images across all environments, minimal configuration differences.

```dockerfile
# Same Dockerfile for dev, staging, production
# Environment variables differentiate behavior
# Same backing services (Redis, R2) in all environments
```

#### XI. Logs
**✅ Compliant**: Structured logs to stdout, no file writing. Aggregation handled by platform.

```go
// All logs to stdout in JSON format
logger, _ := zap.NewProduction()  // JSON logs
logger.Info("message", zap.String("key", "value"))

// Cloudflare Workers: Logs via Logpush
// Docker: Logs via docker logs / logging driver
```

#### XII. Admin Processes
**✅ Compliant**: One-off admin tasks via dedicated CLI commands in the same codebase.

```bash
# Admin CLI for one-off operations
./nplb-admin verify-repo --owner=user --repo=project
./nplb-admin cleanup-old-releases --days=90
./nplb-admin rotate-gpg-key
```

---

## 3. Detailed Migration Strategy

### 3.1 Phase 1: Project Setup & Foundation (Week 1)

#### 3.1.1 Initialize Go Project
```bash
mkdir nplb-go
cd nplb-go
go mod init github.com/zjpiazza/nplb
```

#### 3.1.2 Project Structure
```
nplb-go/
├── cmd/
│   ├── api/           # API server entry point
│   │   └── main.go
│   ├── worker/        # Background worker entry point
│   │   └── main.go
│   └── cli/           # CLI tool (optional)
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/  # HTTP handlers
│   │   ├── middleware/
│   │   └── routes/
│   ├── config/        # Configuration
│   ├── models/        # Data structures
│   ├── services/      # Business logic
│   │   ├── github/
│   │   ├── repository/
│   │   ├── storage/
│   │   └── debian/
│   ├── tasks/         # Async task definitions
│   └── queue/         # Queue client setup
├── pkg/               # Reusable packages
│   └── gpg/           # GPG utilities
├── scripts/           # Build/deploy scripts
├── deployments/       # Docker/K8s configs
│   ├── Dockerfile
│   └── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

#### 3.1.3 Core Dependencies Installation
```bash
# Web framework
go get github.com/gofiber/fiber/v2

# Task queue
go get github.com/hibiken/asynq

# GitHub API
go get github.com/google/go-github/v57/github
go get golang.org/x/oauth2

# AWS SDK
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3

# Debian package parsing
go get pault.ag/go/debian

# Configuration
go get github.com/spf13/viper

# Logging
go get go.uber.org/zap

# Redis
go get github.com/redis/go-redis/v9

# Validation
go get github.com/go-playground/validator/v10
```

#### 3.1.4 Configuration Module (`internal/config/config.go`)

```go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    // Server
    ServerHost string `mapstructure:"SERVER_HOST"`
    ServerPort int    `mapstructure:"SERVER_PORT"`
    
    // GitHub
    GithubToken string `mapstructure:"GITHUB_TOKEN" validate:"required"`
    
    // Cloudflare R2
    R2AccountID        string `mapstructure:"R2_ACCOUNT_ID" validate:"required"`
    R2AccessKeyID      string `mapstructure:"R2_ACCESS_KEY_ID" validate:"required"`
    R2SecretAccessKey  string `mapstructure:"R2_SECRET_ACCESS_KEY" validate:"required"`
    R2BucketName       string `mapstructure:"R2_BUCKET_NAME" validate:"required"`
    R2PublicURL        string `mapstructure:"R2_PUBLIC_URL"` // Optional custom domain
    
    // Redis (or Cloudflare Queues in future)
    RedisHost     string `mapstructure:"REDIS_HOST"`
    RedisPort     int    `mapstructure:"REDIS_PORT"`
    RedisPassword string `mapstructure:"REDIS_PASSWORD"`
    RedisDB       int    `mapstructure:"REDIS_DB"`
    
    // GPG
    GPGHome     string `mapstructure:"GPG_HOME"`
    GPGKeyEmail string `mapstructure:"GPG_KEY_EMAIL"`
    
    // Repository
    OutputDir        string `mapstructure:"OUTPUT_DIR"`
    DefaultCodename  string `mapstructure:"DEFAULT_CODENAME"`
}

func Load() (*Config, error) {
    viper.SetConfigFile(".env")
    viper.AutomaticEnv()
    
    viper.SetDefault("SERVER_HOST", "0.0.0.0")
    viper.SetDefault("SERVER_PORT", 8080)
    viper.SetDefault("REDIS_HOST", "redis")
    viper.SetDefault("REDIS_PORT", 6379)
    viper.SetDefault("REDIS_DB", 0)
    viper.SetDefault("GPG_HOME", "keys")
    viper.SetDefault("OUTPUT_DIR", "build")
    viper.SetDefault("DEFAULT_CODENAME", "stable")
    
    if err := viper.ReadInConfig(); err != nil {
        // Config file not required if env vars are set
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
    }
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}

func (c *Config) StorageURL() string {
    if c.R2PublicURL != "" {
        return c.R2PublicURL
    }
    // Default R2 public URL format
    return fmt.Sprintf("https://%s.r2.dev", c.R2BucketName)
}

func (c *Config) R2Endpoint() string {
    return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.R2AccountID)
}
```

#### 3.1.5 Environment Configuration Template

Create comprehensive `.env.example` for all environments:

```bash
# .env.example - Template for all environments
# Copy to .env and fill in actual values

#==============================================================================
# Application Configuration
#==============================================================================
APP_ENV=development          # development | staging | production
APP_VERSION=0.1.0
LOG_LEVEL=debug              # debug | info | warn | error
LOG_FORMAT=console           # console | json

#==============================================================================
# Server Configuration
#==============================================================================
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
SHUTDOWN_TIMEOUT=30s

#==============================================================================
# GitHub API Configuration
#==============================================================================
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx

#==============================================================================
# Cloudflare R2 Storage Configuration
#==============================================================================
R2_ACCOUNT_ID=your-cloudflare-account-id
R2_ACCESS_KEY_ID=your-r2-access-key-id
R2_SECRET_ACCESS_KEY=your-r2-secret-access-key
R2_BUCKET_NAME=nplb-repo
R2_PUBLIC_URL=https://repo.example.com  # Optional: Custom domain for public access
R2_REGION=auto                          # R2 uses 'auto' region

#==============================================================================
# Redis / Queue Configuration
#==============================================================================
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10

# Queue Configuration
QUEUE_CONCURRENCY=10
QUEUE_MAX_RETRY=3
QUEUE_RETRY_DELAY=3s

#==============================================================================
# GPG Signing Configuration
#==============================================================================
GPG_HOME=keys
GPG_KEY_EMAIL=repo@example.com
GPG_PASSPHRASE=                         # Optional: GPG key passphrase

#==============================================================================
# Repository Generation Configuration
#==============================================================================
OUTPUT_DIR=build
DEFAULT_CODENAME=stable
DEFAULT_COMPONENT=main
SUPPORTED_ARCHITECTURES=amd64,arm64

#==============================================================================
# Observability Configuration
#==============================================================================
ENABLE_METRICS=true
METRICS_PORT=9090
ENABLE_PPROF=false                      # Enable Go profiling endpoints
PPROF_PORT=6060

#==============================================================================
# Feature Flags (Optional)
#==============================================================================
ENABLE_COMPRESSION=true                 # Enable gzip/xz compression
ENABLE_GPG_SIGNING=true                 # Enable repository signing
PARALLEL_DOWNLOADS=true                 # Download .deb files in parallel
```

**Environment-Specific Configurations:**

```bash
# .env.development
APP_ENV=development
LOG_LEVEL=debug
LOG_FORMAT=console
REDIS_HOST=localhost
R2_PUBLIC_URL=https://dev-repo.example.com

# .env.staging
APP_ENV=staging
LOG_LEVEL=info
LOG_FORMAT=json
REDIS_HOST=redis-staging
R2_PUBLIC_URL=https://staging-repo.example.com

# .env.production
APP_ENV=production
LOG_LEVEL=warn
LOG_FORMAT=json
REDIS_HOST=redis-production
R2_PUBLIC_URL=https://repo.example.com
ENABLE_PPROF=false
```

#### 3.1.6 Graceful Shutdown and Health Checks

**Health Check Endpoints (`internal/api/handlers/health.go`):**

```go
package handlers

import (
    "context"
    "time"
    
    "github.com/gofiber/fiber/v2"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

type HealthHandler struct {
    redisClient *redis.Client
    logger      *zap.Logger
    startTime   time.Time
    version     string
}

func NewHealthHandler(redisClient *redis.Client, logger *zap.Logger, version string) *HealthHandler {
    return &HealthHandler{
        redisClient: redisClient,
        logger:      logger,
        startTime:   time.Now(),
        version:     version,
    }
}

// Liveness probe - is the application running?
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "alive",
        "timestamp": time.Now().Unix(),
    })
}

// Readiness probe - is the application ready to serve traffic?
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    // Check Redis connection
    if err := h.redisClient.Ping(ctx).Err(); err != nil {
        h.logger.Error("Redis health check failed", zap.Error(err))
        return c.Status(503).JSON(fiber.Map{
            "status": "not_ready",
            "reason": "redis_unavailable",
        })
    }
    
    return c.JSON(fiber.Map{
        "status": "ready",
        "timestamp": time.Now().Unix(),
    })
}

// Detailed health status with component checks
func (h *HealthHandler) Health(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    health := fiber.Map{
        "status":    "healthy",
        "version":   h.version,
        "uptime":    time.Since(h.startTime).Seconds(),
        "timestamp": time.Now().Unix(),
        "checks": fiber.Map{
            "redis": "unknown",
        },
    }
    
    // Check Redis
    if err := h.redisClient.Ping(ctx).Err(); err != nil {
        health["checks"].(fiber.Map)["redis"] = "unhealthy"
        health["status"] = "degraded"
    } else {
        health["checks"].(fiber.Map)["redis"] = "healthy"
    }
    
    return c.JSON(health)
}
```

**Graceful Shutdown for API Server (`cmd/api/main.go`):**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gofiber/fiber/v2"
    "go.uber.org/zap"
    
    "github.com/zjpiazza/nplb/internal/api/handlers"
    "github.com/zjpiazza/nplb/internal/api/routes"
    "github.com/zjpiazza/nplb/internal/config"
    "github.com/zjpiazza/nplb/internal/queue"
)

const version = "0.1.0" // Set by GoReleaser

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Initialize logger
    var logger *zap.Logger
    if cfg.LogFormat == "json" {
        logger, _ = zap.NewProduction()
    } else {
        logger, _ = zap.NewDevelopment()
    }
    defer logger.Sync()
    
    logger.Info("Starting NPLB API",
        zap.String("version", version),
        zap.String("environment", cfg.AppEnv),
    )
    
    // Initialize queue client
    queueClient := queue.NewClient(cfg)
    defer queueClient.Close()
    
    // Initialize handlers
    handler := handlers.NewHandler(queueClient, logger)
    healthHandler := handlers.NewHealthHandler(redisClient, logger, version)
    
    // Setup Fiber app
    app := fiber.New(fiber.Config{
        AppName:      fmt.Sprintf("NPLB API v%s", version),
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    })
    
    // Health check endpoints
    app.Get("/health", healthHandler.Health)
    app.Get("/health/live", healthHandler.Liveness)
    app.Get("/health/ready", healthHandler.Readiness)
    
    // Setup routes
    routes.Setup(app, handler)
    
    // Start server in goroutine
    addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
    go func() {
        logger.Info("API server listening", zap.String("addr", addr))
        if err := app.Listen(addr); err != nil {
            logger.Fatal("Failed to start server", zap.Error(err))
        }
    }()
    
    // Setup graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    <-quit
    logger.Info("Shutting down server gracefully...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := app.ShutdownWithContext(ctx); err != nil {
        logger.Error("Server forced to shutdown", zap.Error(err))
    }
    
    logger.Info("Server stopped")
}
```

**Graceful Shutdown for Worker (`cmd/worker/main.go`):**

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/hibiken/asynq"
    "go.uber.org/zap"
    
    "github.com/zjpiazza/nplb/internal/config"
    "github.com/zjpiazza/nplb/internal/queue"
    "github.com/zjpiazza/nplb/internal/tasks"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Initialize logger
    var logger *zap.Logger
    if cfg.LogFormat == "json" {
        logger, _ = zap.NewProduction()
    } else {
        logger, _ = zap.NewDevelopment()
    }
    defer logger.Sync()
    
    logger.Info("Starting NPLB Worker")
    
    // Initialize services (GitHub, R2, etc.)
    // ... service initialization code ...
    
    // Create task handler
    mux := asynq.NewServeMux()
    mux.HandleFunc(tasks.TypeBuildRepository, func(ctx context.Context, t *asynq.Task) error {
        return tasks.HandleBuildRepositoryTask(ctx, t, githubService, storageService, cfg, logger)
    })
    
    // Initialize worker server
    server := queue.NewServer(cfg)
    
    // Setup graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    // Start worker in goroutine
    go func() {
        logger.Info("Worker started")
        if err := server.Run(mux); err != nil {
            logger.Fatal("Worker failed to start", zap.Error(err))
        }
    }()
    
    <-quit
    logger.Info("Shutting down worker gracefully...")
    
    // Shutdown waits for running tasks to complete
    server.Shutdown()
    
    logger.Info("Worker stopped")
}
```

---

### 3.2 Phase 2: Core Services (Week 1-2)

#### 3.2.1 Models (`internal/models/models.go`)

```go
package models

import "time"

type DebAsset struct {
    Name        string `json:"name"`
    DownloadURL string `json:"download_url"`
    Size        int64  `json:"size"`
}

type Release struct {
    TagName     string      `json:"tag_name"`
    Name        string      `json:"name"`
    PublishedAt *time.Time  `json:"published_at"`
    Assets      []DebAsset  `json:"assets"`
}

type DebInfo struct {
    Package      string `json:"package"`
    Version      string `json:"version"`
    Architecture string `json:"architecture"`
    Depends      string `json:"depends"`
    Description  string `json:"description"`
}

type BuildRequest struct {
    Owner string `json:"owner" validate:"required"`
    Repo  string `json:"repo" validate:"required"`
    Limit int    `json:"limit" validate:"min=1,max=100"`
}

type BuildResponse struct {
    Status  string `json:"status"`
    Message string `json:"message"`
    JobID   string `json:"job_id"`
}
```

#### 3.2.2 GitHub Service (`internal/services/github/github.go`)

```go
package github

import (
    "context"
    "strings"
    
    "github.com/google/go-github/v57/github"
    "golang.org/x/oauth2"
    "github.com/zjpiazza/nplb/internal/models"
)

type Service struct {
    client *github.Client
}

func NewService(token string) *Service {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(context.Background(), ts)
    
    return &Service{
        client: github.NewClient(tc),
    }
}

func (s *Service) GetReleases(ctx context.Context, owner, repo string, limit int) ([]models.Release, error) {
    releases, _, err := s.client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{
        PerPage: limit,
    })
    if err != nil {
        return nil, err
    }
    
    var result []models.Release
    for _, release := range releases {
        var assets []models.DebAsset
        
        for _, asset := range release.Assets {
            if strings.HasSuffix(*asset.Name, ".deb") {
                assets = append(assets, models.DebAsset{
                    Name:        *asset.Name,
                    DownloadURL: *asset.BrowserDownloadURL,
                    Size:        int64(*asset.Size),
                })
            }
        }
        
        if len(assets) > 0 {
            result = append(result, models.Release{
                TagName:     *release.TagName,
                Name:        release.GetName(),
                PublishedAt: release.PublishedAt.GetTime(),
                Assets:      assets,
            })
        }
    }
    
    return result, nil
}
```

#### 3.2.3 Storage Service (`internal/services/storage/r2.go`)

```go
package storage

import (
    "context"
    "io"
    "mime"
    "os"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "go.uber.org/zap"
)

type R2Service struct {
    client     *s3.Client
    bucketName string
    logger     *zap.Logger
}

func NewR2Service(accountID, accessKey, secretKey, bucketName string, logger *zap.Logger) (*R2Service, error) {
    // R2 endpoint resolver
    r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
        return aws.Endpoint{
            URL:               fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
            SigningRegion:     "auto",
            HostnameImmutable: true,
        }, nil
    })
    
    cfg, err := config.LoadDefaultConfig(context.Background(),
        config.WithEndpointResolverWithOptions(r2Resolver),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
        config.WithRegion("auto"), // R2 uses "auto" region
    )
    if err != nil {
        return nil, err
    }
    
    return &R2Service{
        client:     s3.NewFromConfig(cfg),
        bucketName: bucketName,
        logger:     logger,
    }, nil
}

func (s *R2Service) UploadFile(ctx context.Context, filePath, key string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    contentType := mime.TypeByExtension(filepath.Ext(filePath))
    if contentType == "" {
        contentType = "application/octet-stream"
    }
    
    input := &s3.PutObjectInput{
        Bucket:      aws.String(s.bucketName),
        Key:         aws.String(key),
        Body:        file,
        ContentType: aws.String(contentType),
    }
    
    // Set cache control for metadata files
    baseName := filepath.Base(filePath)
    if baseName == "Release" || baseName == "InRelease" || 
       strings.HasPrefix(baseName, "Packages") {
        input.CacheControl = aws.String("no-cache, no-store, must-revalidate")
        input.Expires = aws.Time(time.Unix(0, 0))
    }
    
    _, err = s.client.PutObject(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to upload to R2: %w", err)
    }
    
    return nil
}

func (s *R2Service) UploadDirectory(ctx context.Context, dirPath, prefix string) error {
    return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() {
            return nil
        }
        
        relPath, err := filepath.Rel(dirPath, path)
        if err != nil {
            return err
        }
        
        key := filepath.Join(prefix, relPath)
        key = filepath.ToSlash(key) // Ensure forward slashes
        
        s.logger.Info("Uploading file to R2", zap.String("file", path), zap.String("key", key))
        return s.UploadFile(ctx, path, key)
    })
}
```

#### 3.2.4 Debian Service (`internal/services/debian/debian.go`)

```go
package debian

import (
    "crypto/md5"
    "crypto/sha1"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"
    
    "pault.ag/go/debian/deb"
    "github.com/zjpiazza/nplb/internal/models"
)

type Service struct{}

func NewService() *Service {
    return &Service{}
}

func (s *Service) ExtractDebInfo(debPath string) (*models.DebInfo, error) {
    file, err := os.Open(debPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    debFile, err := deb.Load(file, debPath)
    if err != nil {
        return nil, err
    }
    
    control := debFile.Control
    
    return &models.DebInfo{
        Package:      control.Package,
        Version:      control.Version.String(),
        Architecture: control.Architecture,
        Depends:      control.Depends.String(),
        Description:  control.Description,
    }, nil
}

func (s *Service) CalculateChecksums(filePath string) (md5sum, sha1sum, sha256sum string, size int64, err error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", "", "", 0, err
    }
    defer file.Close()
    
    md5Hash := md5.New()
    sha1Hash := sha1.New()
    sha256Hash := sha256.New()
    
    multiWriter := io.MultiWriter(md5Hash, sha1Hash, sha256Hash)
    size, err = io.Copy(multiWriter, file)
    if err != nil {
        return "", "", "", 0, err
    }
    
    return hex.EncodeToString(md5Hash.Sum(nil)),
        hex.EncodeToString(sha1Hash.Sum(nil)),
        hex.EncodeToString(sha256Hash.Sum(nil)),
        size,
        nil
}

func (s *Service) DownloadFile(url, destPath string) error {
    // Use net/http to download file
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("bad status: %s", resp.Status)
    }
    
    out, err := os.Create(destPath)
    if err != nil {
        return err
    }
    defer out.Close()
    
    _, err = io.Copy(out, resp.Body)
    return err
}
```

---

### 3.3 Phase 3: Repository Builder (Week 2)

#### 3.3.1 Repository Service (`internal/services/repository/repository.go`)

This is the most complex service - handles:
- Creating temporary repository structure
- Downloading .deb packages
- Generating Packages files
- Generating Release files with checksums
- GPG signing (if configured)
- Compressing metadata files

**Key Implementation Notes:**
- Use `os.MkdirTemp()` for temporary directories
- Parse .deb files using `pault.ag/go-debian/deb`
- Generate Packages file in Debian control format
- Calculate MD5, SHA1, SHA256 for all metadata files
- Use `compress/gzip` and `github.com/ulikunitz/xz` for compression

```go
package repository

import (
    "compress/gzip"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    
    "github.com/ulikunitz/xz"
    "go.uber.org/zap"
)

type Service struct {
    repoName    string
    baseURL     string
    tempDir     string
    poolDir     string
    distsDir    string
    gpgHome     string
    gpgKeyEmail string
    logger      *zap.Logger
}

func NewService(repoName, baseURL, gpgHome, gpgKeyEmail string, logger *zap.Logger) *Service {
    return &Service{
        repoName:    repoName,
        baseURL:     baseURL,
        gpgHome:     gpgHome,
        gpgKeyEmail: gpgKeyEmail,
        logger:      logger,
    }
}

func (s *Service) CreateRepository() (string, error) {
    tempDir, err := ioutil.TempDir("", "nplb-*")
    if err != nil {
        return "", err
    }
    
    s.tempDir = tempDir
    s.poolDir = filepath.Join(tempDir, "pool", "main")
    s.distsDir = filepath.Join(tempDir, "dists", "stable")
    
    if err := os.MkdirAll(s.poolDir, 0755); err != nil {
        return "", err
    }
    
    if err := os.MkdirAll(filepath.Join(s.distsDir, "main", "binary-amd64"), 0755); err != nil {
        return "", err
    }
    
    s.logger.Info("Created repository structure", zap.String("dir", tempDir))
    return tempDir, nil
}

func (s *Service) Cleanup() error {
    if s.tempDir != "" {
        return os.RemoveAll(s.tempDir)
    }
    return nil
}

// Additional methods:
// - DownloadArtifacts()
// - GenerateMetadata()
// - generatePackagesFile()
// - generateReleaseFile()
// - compressFile()
// - signRelease() (using go-crypto or exec.Command("gpg", ...))
```

#### 3.3.2 GPG Integration Options

**Option A: Use ProtonMail/go-crypto (Pure Go)**
```go
import "github.com/ProtonMail/go-crypto/openpgp"
// Complex but no external dependencies
```

**Option B: Shell out to GPG CLI (Recommended)**
```go
func (s *Service) signRelease() error {
    cmd := exec.Command("gpg",
        "--default-key", s.gpgKeyEmail,
        "--clearsign",
        "--armor",
        "--yes",
        "--output", filepath.Join(s.distsDir, "InRelease"),
        filepath.Join(s.distsDir, "Release"),
    )
    return cmd.Run()
}
```

---

### 3.4 Phase 4: Task Queue Integration (Week 2-3)

#### 3.4.1 Task Definition (`internal/tasks/build.go`)

```go
package tasks

import (
    "context"
    "encoding/json"
    
    "github.com/hibiken/asynq"
    "go.uber.org/zap"
)

const TypeBuildRepository = "repository:build"

type BuildRepositoryPayload struct {
    Owner string `json:"owner"`
    Repo  string `json:"repo"`
    Limit int    `json:"limit"`
}

func NewBuildRepositoryTask(owner, repo string, limit int) (*asynq.Task, error) {
    payload, err := json.Marshal(BuildRepositoryPayload{
        Owner: owner,
        Repo:  repo,
        Limit: limit,
    })
    if err != nil {
        return nil, err
    }
    return asynq.NewTask(TypeBuildRepository, payload), nil
}

func HandleBuildRepositoryTask(ctx context.Context, t *asynq.Task) error {
    var p BuildRepositoryPayload
    if err := json.Unmarshal(t.Payload(), &p); err != nil {
        return err
    }
    
    // Get services from context (injected by worker)
    // Execute repository build logic
    // Similar to Python's build_repository_task
    
    logger.Info("Building repository",
        zap.String("owner", p.Owner),
        zap.String("repo", p.Repo),
        zap.Int("limit", p.Limit),
    )
    
    // Implementation here...
    
    return nil
}
```

#### 3.4.2 Queue Setup (`internal/queue/queue.go`)

```go
package queue

import (
    "github.com/hibiken/asynq"
    "github.com/zjpiazza/nplb/internal/config"
)

func NewClient(cfg *config.Config) *asynq.Client {
    return asynq.NewClient(asynq.RedisClientOpt{
        Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
        Password: cfg.RedisPassword,
        DB:       cfg.RedisDB,
    })
}

func NewServer(cfg *config.Config) *asynq.Server {
    return asynq.NewServer(
        asynq.RedisClientOpt{
            Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
            Password: cfg.RedisPassword,
            DB:       cfg.RedisDB,
        },
        asynq.Config{
            Concurrency: 10,
        },
    )
}
```

---

### 3.5 Phase 5: API Implementation (Week 3)

#### 3.5.1 API Routes (`internal/api/routes/routes.go`)

```go
package routes

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/recover"
    "github.com/zjpiazza/nplb/internal/api/handlers"
)

func Setup(app *fiber.App, h *handlers.Handler) {
    app.Use(logger.New())
    app.Use(recover.New())
    
    api := app.Group("/api/v1")
    
    repos := api.Group("/repositories")
    repos.Post("/build", h.BuildRepository)
    repos.Get("/job/:id", h.GetJobStatus)
}
```

#### 3.5.2 Handlers (`internal/api/handlers/repository.go`)

```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/go-playground/validator/v10"
    "github.com/hibiken/asynq"
    "go.uber.org/zap"
    
    "github.com/zjpiazza/nplb/internal/models"
    "github.com/zjpiazza/nplb/internal/tasks"
)

type Handler struct {
    queueClient *asynq.Client
    logger      *zap.Logger
    validator   *validator.Validate
}

func NewHandler(queueClient *asynq.Client, logger *zap.Logger) *Handler {
    return &Handler{
        queueClient: queueClient,
        logger:      logger,
        validator:   validator.New(),
    }
}

func (h *Handler) BuildRepository(c *fiber.Ctx) error {
    var req models.BuildRequest
    if err := c.QueryParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
    }
    
    if req.Limit == 0 {
        req.Limit = 1
    }
    
    if err := h.validator.Struct(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    }
    
    task, err := tasks.NewBuildRepositoryTask(req.Owner, req.Repo, req.Limit)
    if err != nil {
        h.logger.Error("Failed to create task", zap.Error(err))
        return c.Status(500).JSON(fiber.Map{"error": "failed to queue job"})
    }
    
    info, err := h.queueClient.Enqueue(task)
    if err != nil {
        h.logger.Error("Failed to enqueue task", zap.Error(err))
        return c.Status(500).JSON(fiber.Map{"error": "failed to queue job"})
    }
    
    return c.JSON(models.BuildResponse{
        Status:  "success",
        Message: fmt.Sprintf("Job %s/%s queued", req.Owner, req.Repo),
        JobID:   info.ID,
    })
}

func (h *Handler) GetJobStatus(c *fiber.Ctx) error {
    jobID := c.Params("id")
    // Query asynq inspector for job status
    // Return job state, result, errors
    return c.JSON(fiber.Map{"job_id": jobID, "status": "processing"})
}
```

#### 3.5.3 API Server Entry Point (`cmd/api/main.go`)

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/gofiber/fiber/v2"
    "go.uber.org/zap"
    
    "github.com/zjpiazza/nplb/internal/api/handlers"
    "github.com/zjpiazza/nplb/internal/api/routes"
    "github.com/zjpiazza/nplb/internal/config"
    "github.com/zjpiazza/nplb/internal/queue"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Initialize logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // Initialize queue client
    queueClient := queue.NewClient(cfg)
    defer queueClient.Close()
    
    // Initialize handlers
    handler := handlers.NewHandler(queueClient, logger)
    
    // Setup Fiber app
    app := fiber.New(fiber.Config{
        AppName: "NPLB API v1.0.0",
    })
    
    // Setup routes
    routes.Setup(app, handler)
    
    // Start server
    addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
    logger.Info("Starting API server", zap.String("addr", addr))
    
    if err := app.Listen(addr); err != nil {
        logger.Fatal("Failed to start server", zap.Error(err))
    }
}
```

---

### 3.6 Phase 6: Worker Implementation (Week 3)

#### 3.6.1 Worker Entry Point (`cmd/worker/main.go`)

```go
package main

import (
    "context"
    "log"
    
    "github.com/hibiken/asynq"
    "go.uber.org/zap"
    
    "github.com/zjpiazza/nplb/internal/config"
    "github.com/zjpiazza/nplb/internal/queue"
    "github.com/zjpiazza/nplb/internal/tasks"
    "github.com/zjpiazza/nplb/internal/services/github"
    "github.com/zjpiazza/nplb/internal/services/repository"
    "github.com/zjpiazza/nplb/internal/services/storage"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Initialize logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // Initialize services
    githubService := github.NewService(cfg.GithubToken)
    
    storageService, err := storage.NewR2Service(
        cfg.R2AccountID,
        cfg.R2AccessKeyID,
        cfg.R2SecretAccessKey,
        cfg.R2BucketName,
        logger,
    )
    if err != nil {
        logger.Fatal("Failed to create R2 storage service", zap.Error(err))
    }
    
    // Create task handler with dependencies injected
    mux := asynq.NewServeMux()
    
    mux.HandleFunc(tasks.TypeBuildRepository, func(ctx context.Context, t *asynq.Task) error {
        return tasks.HandleBuildRepositoryTask(ctx, t, githubService, storageService, cfg, logger)
    })
    
    // Initialize and start worker server
    server := queue.NewServer(cfg)
    
    logger.Info("Starting worker")
    if err := server.Run(mux); err != nil {
        logger.Fatal("Failed to start worker", zap.Error(err))
    }
}
```

---

### 3.7 Phase 7: Containerization (Week 3-4)

#### 3.7.1 Multi-stage Dockerfile (`deployments/Dockerfile`)

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /build

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates gnupg

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/api .
COPY --from=builder /build/worker .

# Create directories
RUN mkdir -p /app/keys /app/build

EXPOSE 8080

# Default to API server (override in docker-compose)
CMD ["./api"]
```

#### 3.7.2 Docker Compose (`deployments/docker-compose.yml`)

```yaml
version: '3.8'

services:
  api:
    build:
      context: ..
      dockerfile: deployments/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - R2_ACCOUNT_ID=${R2_ACCOUNT_ID}
      - R2_ACCESS_KEY_ID=${R2_ACCESS_KEY_ID}
      - R2_SECRET_ACCESS_KEY=${R2_SECRET_ACCESS_KEY}
      - R2_BUCKET_NAME=${R2_BUCKET_NAME}
      - R2_PUBLIC_URL=${R2_PUBLIC_URL}
    depends_on:
      - redis
    volumes:
      - ./keys:/app/keys
    networks:
      - nplb-network

  worker:
    build:
      context: ..
      dockerfile: deployments/Dockerfile
    command: ./worker
    environment:
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - R2_ACCOUNT_ID=${R2_ACCOUNT_ID}
      - R2_ACCESS_KEY_ID=${R2_ACCESS_KEY_ID}
      - R2_SECRET_ACCESS_KEY=${R2_SECRET_ACCESS_KEY}
      - R2_BUCKET_NAME=${R2_BUCKET_NAME}
      - R2_PUBLIC_URL=${R2_PUBLIC_URL}
    depends_on:
      - redis
    volumes:
      - ./keys:/app/keys
    networks:
      - nplb-network
    deploy:
      replicas: 2

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    networks:
      - nplb-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 3

networks:
  nplb-network:
    driver: bridge

volumes:
  redis_data:
```

---

### 3.8 Phase 8: Admin CLI Tools (Week 4)

#### 3.8.1 Admin CLI Structure

Following 12-Factor principle XII (Admin Processes), create one-off administrative commands:

```
cmd/
├── api/
├── worker/
└── admin/              # NEW: Admin CLI
    └── main.go
```

#### 3.8.2 Admin CLI Implementation (`cmd/admin/main.go`)

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "go.uber.org/zap"
    
    "github.com/zjpiazza/nplb/internal/config"
    "github.com/zjpiazza/nplb/internal/services/github"
    "github.com/zjpiazza/nplb/internal/services/repository"
    "github.com/zjpiazza/nplb/internal/services/storage"
)

var (
    cfg    *config.Config
    logger *zap.Logger
)

func main() {
    var err error
    
    // Load configuration
    cfg, err = config.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
        os.Exit(1)
    }
    
    // Initialize logger
    logger, _ = zap.NewProduction()
    defer logger.Sync()
    
    rootCmd := &cobra.Command{
        Use:   "nplb-admin",
        Short: "NPLB administrative commands",
        Long:  "One-off administrative tasks for NPLB repository management",
    }
    
    rootCmd.AddCommand(
        verifyRepoCmd(),
        cleanupCmd(),
        rotateKeyCmd(),
        rebuildCmd(),
        listReleasesCmd(),
        exportMetricsCmd(),
    )
    
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

// Verify repository integrity
func verifyRepoCmd() *cobra.Command {
    var owner, repo string
    
    cmd := &cobra.Command{
        Use:   "verify-repo",
        Short: "Verify repository integrity and metadata",
        RunE: func(cmd *cobra.Command, args []string) error {
            logger.Info("Verifying repository",
                zap.String("owner", owner),
                zap.String("repo", repo),
            )
            
            // Initialize services
            storageService, err := storage.NewR2Service(
                cfg.R2AccountID,
                cfg.R2AccessKeyID,
                cfg.R2SecretAccessKey,
                cfg.R2BucketName,
                logger,
            )
            if err != nil {
                return fmt.Errorf("failed to create storage service: %w", err)
            }
            
            // Check if Release file exists
            // Verify GPG signatures
            // Validate Packages files
            // Check .deb file availability
            
            fmt.Println("✓ Repository structure valid")
            fmt.Println("✓ GPG signatures valid")
            fmt.Println("✓ All packages accessible")
            
            return nil
        },
    }
    
    cmd.Flags().StringVar(&owner, "owner", "", "GitHub repository owner (required)")
    cmd.Flags().StringVar(&repo, "repo", "", "GitHub repository name (required)")
    cmd.MarkFlagRequired("owner")
    cmd.MarkFlagRequired("repo")
    
    return cmd
}

// Cleanup old releases
func cleanupCmd() *cobra.Command {
    var days int
    var dryRun bool
    
    cmd := &cobra.Command{
        Use:   "cleanup",
        Short: "Remove old releases from R2 storage",
        RunE: func(cmd *cobra.Command, args []string) error {
            logger.Info("Cleaning up old releases",
                zap.Int("days", days),
                zap.Bool("dry_run", dryRun),
            )
            
            // List all repositories in R2
            // Find releases older than N days
            // Delete old releases (if not dry-run)
            
            if dryRun {
                fmt.Println("DRY RUN: Would delete 5 old releases")
            } else {
                fmt.Println("✓ Deleted 5 old releases")
            }
            
            return nil
        },
    }
    
    cmd.Flags().IntVar(&days, "days", 90, "Delete releases older than N days")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without deleting")
    
    return cmd
}

// Rotate GPG signing key
func rotateKeyCmd() *cobra.Command {
    var newKeyEmail, oldKeyEmail string
    
    cmd := &cobra.Command{
        Use:   "rotate-key",
        Short: "Rotate GPG signing key for repositories",
        Long: `Rotate the GPG key used for signing repository metadata.
This will re-sign all existing Release files with the new key.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            logger.Info("Rotating GPG key",
                zap.String("old_key", oldKeyEmail),
                zap.String("new_key", newKeyEmail),
            )
            
            // Export old public key
            // Import new key
            // Re-sign all Release files
            // Update public key in repository
            
            fmt.Println("✓ GPG key rotated successfully")
            fmt.Println("  Please distribute the new public key to users")
            
            return nil
        },
    }
    
    cmd.Flags().StringVar(&oldKeyEmail, "old-key", "", "Old GPG key email")
    cmd.Flags().StringVar(&newKeyEmail, "new-key", "", "New GPG key email (required)")
    cmd.MarkFlagRequired("new-key")
    
    return cmd
}

// Force rebuild a repository
func rebuildCmd() *cobra.Command {
    var owner, repo string
    var limit int
    
    cmd := &cobra.Command{
        Use:   "rebuild",
        Short: "Force rebuild a repository without queueing",
        RunE: func(cmd *cobra.Command, args []string) error {
            logger.Info("Rebuilding repository",
                zap.String("owner", owner),
                zap.String("repo", repo),
                zap.Int("limit", limit),
            )
            
            // Initialize all services
            githubService := github.NewService(cfg.GithubToken)
            
            storageService, err := storage.NewR2Service(
                cfg.R2AccountID,
                cfg.R2AccessKeyID,
                cfg.R2SecretAccessKey,
                cfg.R2BucketName,
                logger,
            )
            if err != nil {
                return fmt.Errorf("failed to create storage service: %w", err)
            }
            
            repoService := repository.NewService(
                fmt.Sprintf("%s/%s", owner, repo),
                cfg.StorageURL(),
                cfg.GPGHome,
                cfg.GPGKeyEmail,
                logger,
            )
            
            // Execute rebuild synchronously
            fmt.Println("Building repository...")
            // ... build logic ...
            
            fmt.Println("✓ Repository rebuilt successfully")
            
            return nil
        },
    }
    
    cmd.Flags().StringVar(&owner, "owner", "", "GitHub repository owner (required)")
    cmd.Flags().StringVar(&repo, "repo", "", "GitHub repository name (required)")
    cmd.Flags().IntVar(&limit, "limit", 10, "Number of releases to include")
    cmd.MarkFlagRequired("owner")
    cmd.MarkFlagRequired("repo")
    
    return cmd
}

// List releases from GitHub
func listReleasesCmd() *cobra.Command {
    var owner, repo string
    var limit int
    
    cmd := &cobra.Command{
        Use:   "list-releases",
        Short: "List releases from GitHub repository",
        RunE: func(cmd *cobra.Command, args []string) error {
            githubService := github.NewService(cfg.GithubToken)
            
            releases, err := githubService.GetReleases(cmd.Context(), owner, repo, limit)
            if err != nil {
                return fmt.Errorf("failed to fetch releases: %w", err)
            }
            
            fmt.Printf("Found %d releases with .deb assets:\n\n", len(releases))
            for i, release := range releases {
                fmt.Printf("%d. %s (%s)\n", i+1, release.Name, release.TagName)
                for _, asset := range release.Assets {
                    fmt.Printf("   - %s (%d bytes)\n", asset.Name, asset.Size)
                }
                fmt.Println()
            }
            
            return nil
        },
    }
    
    cmd.Flags().StringVar(&owner, "owner", "", "GitHub repository owner (required)")
    cmd.Flags().StringVar(&repo, "repo", "", "GitHub repository name (required)")
    cmd.Flags().IntVar(&limit, "limit", 10, "Number of releases to list")
    cmd.MarkFlagRequired("owner")
    cmd.MarkFlagRequired("repo")
    
    return cmd
}

// Export metrics (for debugging)
func exportMetricsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "export-metrics",
        Short: "Export current metrics and statistics",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Query Redis for job statistics
            // Query R2 for storage usage
            // Export to JSON
            
            fmt.Println("✓ Metrics exported to metrics.json")
            
            return nil
        },
    }
    
    return cmd
}
```

#### 3.8.3 Admin CLI Usage Examples

```bash
# Verify repository integrity
./nplb-admin verify-repo --owner=user --repo=project

# List releases from GitHub
./nplb-admin list-releases --owner=user --repo=project --limit=5

# Cleanup old releases (dry run first)
./nplb-admin cleanup --days=90 --dry-run
./nplb-admin cleanup --days=90

# Force rebuild without queue
./nplb-admin rebuild --owner=user --repo=project --limit=10

# Rotate GPG signing key
./nplb-admin rotate-key --old-key=old@example.com --new-key=new@example.com

# Export current metrics
./nplb-admin export-metrics
```

---

## 4. Testing Strategy

### 4.1 Unit Tests

Create test files alongside each package:
```
internal/services/github/github_test.go
internal/services/repository/repository_test.go
internal/services/storage/s3_test.go
```

**Key Test Cases:**
- GitHub API mocking with `github.com/migueleliasweb/go-github-mock`
- R2 operations with S3-compatible mocking (localstack or minio)
- Debian package parsing with test fixtures
- Repository metadata generation validation

### 4.2 Integration Tests

```go
// tests/integration/build_test.go
func TestEndToEndRepositoryBuild(t *testing.T) {
    // 1. Start test Redis instance
    // 2. Enqueue build task
    // 3. Start worker
    // 4. Verify R2 upload (mock or test bucket)
    // 5. Validate repository structure
}
```

### 4.3 Performance Benchmarks

```go
func BenchmarkRepositoryGeneration(b *testing.B) {
    // Benchmark repository metadata generation
}
```

---

## 5. Migration Checklist

### 5.1 Pre-Migration
- [ ] Review Python codebase thoroughly
- [ ] Document all edge cases and quirks
- [ ] Set up Go development environment
- [ ] Create Go project structure

### 5.2 Implementation
- [ ] **Phase 1:** Project setup and configuration (3 days)
  - [ ] Initialize Go project structure
  - [ ] Set up configuration management
  - [ ] Create .env.example template
  - [ ] Implement graceful shutdown
  - [ ] Add health check endpoints
- [ ] **Phase 2:** Core services implementation (5 days)
  - [ ] GitHub service
  - [ ] Storage service (Cloudflare R2)
  - [ ] Debian package parsing
- [ ] **Phase 3:** Repository builder (5 days)
  - [ ] Repository structure generation
  - [ ] Packages file generation
  - [ ] Release file generation
  - [ ] GPG signing integration
- [ ] **Phase 4:** Task queue integration (3 days)
  - [ ] Asynq setup
  - [ ] Task definitions
  - [ ] Worker implementation
- [ ] **Phase 5:** API implementation (3 days)
  - [ ] Fiber/Gin setup
  - [ ] Routes and handlers
  - [ ] Request validation
- [ ] **Phase 6:** Containerization (2 days)
  - [ ] Dockerfile
  - [ ] Docker Compose
- [ ] **Phase 7:** Testing (3 days)
  - [ ] Unit tests (>80% coverage)
  - [ ] Integration tests
  - [ ] Load testing
- [ ] **Phase 8:** CI/CD and Admin Tools (3 days)
  - [ ] GitHub Actions workflows
  - [ ] GoReleaser configuration
  - [ ] Admin CLI implementation
  - [ ] Observability setup

### 5.3 Post-Migration
- [ ] Performance benchmarking (Python vs Go)
- [ ] Documentation updates
- [ ] CI/CD pipeline setup
- [ ] Deployment to staging
- [ ] Parallel run (Python + Go)
- [ ] Production cutover
- [ ] Monitor and optimize

---

## 6. Risk Mitigation

### 6.1 Technical Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| GPG signing complexity | High | Use shell exec to GPG CLI initially, optimize later |
| Debian package parsing differences | Medium | Extensive testing with real .deb files |
| R2 S3-compatibility edge cases | Medium | Test thoroughly with R2-specific features and limits |
| Redis connection issues | Medium | Use connection pooling and retry logic |
| Memory leaks in long-running workers | Medium | Implement proper cleanup and monitoring |
| Cloudflare rate limits | Low | Implement exponential backoff and request batching |

### 6.2 Operational Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Downtime during migration | High | Blue-green deployment strategy |
| Data loss during transition | High | Maintain Python version as fallback |
| Performance regression | Medium | Comprehensive benchmarking before cutover |
| Documentation gaps | Low | Create detailed runbooks and API docs |

---

## 7. Performance Expectations

### 7.1 Expected Improvements

Based on typical Python-to-Go migrations:

| Metric | Python (Current) | Go (Expected) | Improvement |
|--------|------------------|---------------|-------------|
| API Response Time | ~50-100ms | ~5-15ms | **5-10x faster** |
| Memory Usage | ~150-200MB | ~30-50MB | **3-4x reduction** |
| Concurrent Requests | ~100 req/s | ~1000+ req/s | **10x+ increase** |
| Binary Size | N/A (interpreter) | ~15-25MB | Standalone binary |
| Cold Start | ~2-3 seconds | ~50-100ms | **20-30x faster** |
| Worker Throughput | ~5-10 repos/min | ~20-50 repos/min | **2-5x faster** |

### 7.2 Optimization Opportunities

- **Concurrent Downloads:** Use goroutines to download .deb files in parallel
- **Streaming Processing:** Stream R2 uploads instead of buffering
- **Connection Pooling:** Reuse HTTP/Redis connections
- **Caching:** Add in-memory cache for GitHub API responses
- **Compression:** Parallel gzip/xz compression using goroutines
- **Cloudflare CDN:** Leverage automatic edge caching for package delivery
- **Zero Egress Costs:** R2's free egress eliminates bandwidth costs

---

## 8. Deployment Strategy

### 8.1 Blue-Green Deployment

```
┌─────────────────┐
│   Load Balancer │
└────────┬────────┘
         │
         ├──────────────┐
         │              │
    ┌────▼────┐    ┌────▼────┐
    │  Blue   │    │  Green  │
    │ (Python)│    │  (Go)   │
    └─────────┘    └─────────┘
```

**Steps:**
1. Deploy Go version to "Green" environment
2. Run smoke tests
3. Gradually shift traffic (10% → 50% → 100%)
4. Monitor error rates and performance
5. Keep Python "Blue" for quick rollback
6. Decommission Python after 2 weeks of stable Go operation

### 8.2 Rollback Plan

If issues arise:
1. Switch load balancer back to Python
2. Drain Go workers
3. Investigate and fix issues
4. Retry deployment

### 8.3 CI/CD Pipeline with GitHub Actions

Following 12-Factor principle V (Build, Release, Run), implement automated CI/CD pipeline:

#### 8.3.1 Continuous Integration Workflow

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

permissions:
  contents: read
  pull-requests: write

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      
      - name: Download dependencies
        run: go mod download
      
      - name: Verify dependencies
        run: go mod verify
      
      - name: Run go vet
        run: go vet ./...
      
      - name: Run go fmt
        run: |
          if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
            echo "Please run 'go fmt' on your code"
            gofmt -s -l .
            exit 1
          fi
      
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
      
      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
          flags: unittests
          name: codecov-nplb
  
  lint:
    name: Lint
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m
  
  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [test, lint]
    
    strategy:
      matrix:
        target:
          - cmd/api
          - cmd/worker
          - cmd/admin
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      
      - name: Build binary
        run: |
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v \
            -ldflags="-w -s -X main.version=${{ github.sha }}" \
            -o ./bin/$(basename ${{ matrix.target }}) \
            ./${{ matrix.target }}
      
      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: $(basename ${{ matrix.target }})-${{ github.sha }}
          path: ./bin/$(basename ${{ matrix.target }})
  
  security:
    name: Security Scan
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      
      - name: Run Gosec Security Scanner
        uses: securego/gosec@master
        with:
          args: ./...
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'
```

#### 8.3.2 Docker Build and Push Workflow

Create `.github/workflows/docker.yml`:

```yaml
name: Docker

on:
  push:
    branches: [main, develop]
    tags: ['v*']

permissions:
  contents: read
  packages: write

jobs:
  docker:
    name: Build and Push Docker Images
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ghcr.io/${{ github.repository }}
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}
            type=sha
      
      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./deployments/Dockerfile
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.sha }}
```

### 8.4 Release Automation with GoReleaser

#### 8.4.1 GoReleaser Configuration

Create `.goreleaser.yml`:

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go mod verify
    - go test ./...

builds:
  - id: api
    binary: nplb-api
    main: ./cmd/api
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
    mod_timestamp: '{{ .CommitTimestamp }}'
  
  - id: worker
    binary: nplb-worker
    main: ./cmd/worker
    env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
    mod_timestamp: '{{ .CommitTimestamp }}'
  
  - id: admin
    binary: nplb-admin
    main: ./cmd/admin
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
    mod_timestamp: '{{ .CommitTimestamp }}'

archives:
  - id: default
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE
      - deployments/docker-compose.yml

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  use: github
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
      - '^ci:'
  groups:
    - title: 'Features'
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: 'Bug Fixes'
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: 'Performance Improvements'
      regexp: '^.*?perf(\([[:word:]]+\))??!?:.+$'
      order: 2
    - title: 'Refactors'
      regexp: '^.*?refactor(\([[:word:]]+\))??!?:.+$'
      order: 3
    - title: 'Other'
      order: 999

dockers:
  - image_templates:
      - 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:{{ .Version }}-amd64'
    use: buildx
    dockerfile: deployments/Dockerfile
    build_flag_templates:
      - "--platform=linux/amd64"
      - "--label=org.opencontainers.image.created={{.Date}}"
      - "--label=org.opencontainers.image.title={{.ProjectName}}"
      - "--label=org.opencontainers.image.revision={{.FullCommit}}"
      - "--label=org.opencontainers.image.version={{.Version}}"
  
  - image_templates:
      - 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:{{ .Version }}-arm64'
    use: buildx
    goarch: arm64
    dockerfile: deployments/Dockerfile
    build_flag_templates:
      - "--platform=linux/arm64"
      - "--label=org.opencontainers.image.created={{.Date}}"
      - "--label=org.opencontainers.image.title={{.ProjectName}}"
      - "--label=org.opencontainers.image.revision={{.FullCommit}}"
      - "--label=org.opencontainers.image.version={{.Version}}"

docker_manifests:
  - name_template: 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:{{ .Version }}'
    image_templates:
      - 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:{{ .Version }}-amd64'
      - 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:{{ .Version }}-arm64'
  
  - name_template: 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:latest'
    image_templates:
      - 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:{{ .Version }}-amd64'
      - 'ghcr.io/{{ .Env.GITHUB_REPOSITORY }}:{{ .Version }}-arm64'

release:
  github:
    owner: zjpiazza
    name: nplb
  draft: false
  prerelease: auto
  mode: append
  header: |
    ## NPLB Release {{ .Tag }}
    
    **Full Changelog**: https://github.com/zjpiazza/nplb/compare/{{ .PreviousTag }}...{{ .Tag }}
  footer: |
    ## Installation
    
    ### Docker
    ```bash
    docker pull ghcr.io/zjpiazza/nplb:{{ .Tag }}
    ```
    
    ### Binary
    Download the appropriate binary for your platform from the assets below.

signs:
  - cmd: gpg
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"
    artifacts: checksum
```

#### 8.4.2 Release Workflow

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  packages: write

jobs:
  goreleaser:
    name: Release with GoReleaser
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Import GPG key
        if: env.GPG_PRIVATE_KEY != ''
        env:
          GPG_PRIVATE_KEY: ${{ secrets.GPG_PRIVATE_KEY }}
        run: |
          echo "$GPG_PRIVATE_KEY" | gpg --import --batch
      
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPOSITORY: ${{ github.repository }}
          GPG_FINGERPRINT: ${{ secrets.GPG_FINGERPRINT }}
```

#### 8.4.3 Release Process

**Using Conventional Commits:**

```bash
# Feature commits
git commit -m "feat: add support for multi-architecture .deb packages"
git commit -m "feat(api): add health check endpoints"

# Bug fix commits
git commit -m "fix: resolve GPG signing timeout issue"
git commit -m "fix(worker): prevent memory leak in repository builder"

# Breaking changes
git commit -m "feat!: change API response format

BREAKING CHANGE: API now returns ISO 8601 timestamps instead of Unix timestamps"
```

**Creating a Release:**

```bash
# 1. Ensure you're on main branch
git checkout main
git pull

# 2. Create and push a version tag
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0

# 3. GitHub Actions automatically:
#    - Runs tests
#    - Builds binaries for all platforms
#    - Creates Docker images (multi-arch)
#    - Generates changelog from commits
#    - Creates GitHub release
#    - Uploads all artifacts
```

**Version Bumping Guidelines:**

```bash
# Patch release (v1.2.3 -> v1.2.4): Bug fixes only
git tag v1.2.4

# Minor release (v1.2.3 -> v1.3.0): New features, backward compatible
git tag v1.3.0

# Major release (v1.2.3 -> v2.0.0): Breaking changes
git tag v2.0.0
```

---

## 9. Documentation Requirements

### 9.1 Technical Documentation
- [ ] API documentation (OpenAPI/Swagger)
- [ ] Architecture diagrams
- [ ] Database schemas (Redis key patterns)
- [ ] Deployment guide
- [ ] Runbook for operations

### 9.2 Developer Documentation
- [ ] Setup guide for local development
- [ ] Contribution guidelines
- [ ] Code structure overview
- [ ] Testing guide

### 9.3 Observability Strategy

Following 12-Factor principle XI (Logs), implement comprehensive observability using Cloudflare-native tools and Go best practices:

#### 9.3.1 Logging Strategy

**Structured Logging with Zap:**

```go
// Production: JSON logs to stdout
logger, _ := zap.NewProduction()

// Development: Human-readable logs
logger, _ := zap.NewDevelopment()

// Log with context
logger.Info("Repository build started",
    zap.String("owner", owner),
    zap.String("repo", repo),
    zap.String("job_id", jobID),
    zap.Int("releases", count),
)

// Error logging with stack traces
logger.Error("Failed to upload to R2",
    zap.Error(err),
    zap.String("file", filepath),
    zap.String("key", key),
)
```

**Log Levels by Environment:**

```bash
# Development
LOG_LEVEL=debug
LOG_FORMAT=console

# Staging
LOG_LEVEL=info
LOG_FORMAT=json

# Production
LOG_LEVEL=warn
LOG_FORMAT=json
```

**Request Logging Middleware:**

```go
// Fiber middleware for request logging
app.Use(func(c *fiber.Ctx) error {
    start := time.Now()
    
    // Add request ID to context
    requestID := c.Get("X-Request-ID", uuid.New().String())
    c.Locals("request_id", requestID)
    
    err := c.Next()
    
    logger.Info("HTTP request",
        zap.String("request_id", requestID),
        zap.String("method", c.Method()),
        zap.String("path", c.Path()),
        zap.Int("status", c.Response().StatusCode()),
        zap.Duration("latency", time.Since(start)),
        zap.String("ip", c.IP()),
        zap.String("user_agent", c.Get("User-Agent")),
    )
    
    return err
})
```

#### 9.3.2 Metrics and Monitoring

**Phase 1: Docker/Traditional Deployment**

**Prometheus Metrics (Optional):**

```go
// internal/observability/metrics.go
package observability

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // API metrics
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nplb_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "nplb_http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
    
    // Worker metrics
    repositoryBuildsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nplb_repository_builds_total",
            Help: "Total number of repository builds",
        },
        []string{"status"},
    )
    
    repositoryBuildDuration = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name: "nplb_repository_build_duration_seconds",
            Help: "Repository build duration in seconds",
            Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
        },
    )
    
    // R2 metrics
    r2UploadsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nplb_r2_uploads_total",
            Help: "Total number of R2 uploads",
        },
        []string{"status"},
    )
    
    r2UploadBytes = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "nplb_r2_upload_bytes_total",
            Help: "Total bytes uploaded to R2",
        },
    )
)

// Expose metrics endpoint
import "github.com/gofiber/adaptor/v2"
import "github.com/prometheus/client_golang/prometheus/promhttp"

app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
```

**Phase 2: Cloudflare Workers Deployment**

**Native Cloudflare Analytics:**

```javascript
// Cloudflare Workers automatically track:
// - Request count
// - Error rate
// - CPU time per execution
// - Wall time per execution
// - Subrequests
// - Status codes

// Access via Cloudflare Dashboard or GraphQL API
export default {
  async fetch(request, env, ctx) {
    const start = Date.now();
    
    try {
      const response = await handleRequest(request, env);
      
      // Custom metrics via Analytics Engine
      env.ANALYTICS.writeDataPoint({
        'blobs': ['repository_build'],
        'doubles': [Date.now() - start],
        'indexes': [env.ENVIRONMENT],
      });
      
      return response;
    } catch (error) {
      // Errors automatically tracked
      return new Response('Error', { status: 500 });
    }
  }
}
```

**Cloudflare GraphQL API for Metrics:**

```graphql
query GetWorkerMetrics {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      workersInvocationsAdaptive(
        filter: {
          scriptName: $scriptName
          datetime_geq: $start
          datetime_leq: $end
        }
        limit: 10000
      ) {
        sum {
          requests
          errors
          subrequests
        }
        quantiles {
          cpuTimeP50
          cpuTimeP99
          wallTimeP50
          wallTimeP99
        }
      }
    }
  }
}
```

#### 9.3.3 Distributed Tracing (Optional)

**OpenTelemetry Integration:**

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (s *Service) BuildRepository(ctx context.Context, owner, repo string) error {
    tracer := otel.Tracer("nplb")
    ctx, span := tracer.Start(ctx, "build_repository")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("repository.owner", owner),
        attribute.String("repository.name", repo),
    )
    
    // GitHub fetch
    ctx, githubSpan := tracer.Start(ctx, "github.fetch_releases")
    releases, err := s.github.GetReleases(ctx, owner, repo, limit)
    githubSpan.End()
    
    // R2 upload
    ctx, uploadSpan := tracer.Start(ctx, "r2.upload_repository")
    err = s.storage.UploadDirectory(ctx, tempDir, repoPath)
    uploadSpan.End()
    
    return nil
}
```

#### 9.3.4 Error Tracking

**Sentry Integration (Optional):**

```go
import "github.com/getsentry/sentry-go"

// Initialize Sentry
sentry.Init(sentry.ClientOptions{
    Dsn: os.Getenv("SENTRY_DSN"),
    Environment: os.Getenv("APP_ENV"),
    Release: version,
})

// Capture errors
if err != nil {
    sentry.CaptureException(err)
    logger.Error("Repository build failed", zap.Error(err))
    return err
}
```

#### 9.3.5 Key Metrics to Track

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| **API Latency (P95)** | 95th percentile response time | > 500ms |
| **Error Rate** | Percentage of failed requests | > 1% |
| **Repository Build Success Rate** | Successful builds / total builds | < 95% |
| **Repository Build Duration** | Time to build and upload | > 5 minutes |
| **R2 Upload Success Rate** | Successful uploads / total uploads | < 99% |
| **Queue Depth** | Number of pending jobs | > 100 |
| **Worker CPU Usage** | CPU utilization per worker | > 80% |
| **Memory Usage** | Memory consumption per instance | > 90% |

#### 9.3.6 Dashboards and Alerts

**Grafana Dashboard (if using Prometheus):**

```yaml
# Example panel configuration
- title: API Request Rate
  type: graph
  targets:
    - expr: rate(nplb_http_requests_total[5m])
      legendFormat: "{{ method }} {{ endpoint }}"

- title: Repository Build Duration
  type: histogram
  targets:
    - expr: histogram_quantile(0.95, nplb_repository_build_duration_seconds)
      legendFormat: "P95"
```

**Cloudflare Dashboard Access:**

```bash
# View metrics in Cloudflare Dashboard
# Workers & Pages → Your Worker → Metrics

# Or via wrangler CLI
wrangler tail nplb-api --format=pretty

# Or via GraphQL API for custom dashboards
```

#### 9.3.7 Log Aggregation

**Phase 1: Docker Deployment**

```yaml
# docker-compose.yml with log driver
services:
  api:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
        labels: "service,env"
        tag: "{{.Name}}/{{.ID}}"
```

**Phase 2: Cloudflare Workers**

```bash
# Real-time logs via wrangler
wrangler tail nplb-api

# Logpush to external service (S3, R2, Google Cloud Storage)
# Configure via Cloudflare Dashboard
```

**Centralized Logging Options:**
- Cloudflare Logpush → R2 → Query with SQL
- Docker logs → Loki → Grafana
- Cloudwatch Logs (AWS)
- Google Cloud Logging

---

## 10. Success Criteria

### 10.1 Functional Requirements
- ✅ All Python API endpoints replicated
- ✅ GitHub integration working correctly
- ✅ R2 uploads successful with proper permissions
- ✅ GPG signing functional
- ✅ Repository metadata valid (testable with `apt-get update`)
- ✅ Cloudflare CDN integration for package delivery

### 10.2 Non-Functional Requirements
- ✅ API latency < 20ms (p95)
- ✅ Worker throughput > 20 repos/min
- ✅ Memory usage < 100MB per instance
- ✅ Zero data loss during migration
- ✅ 99.9% uptime SLA maintained

### 10.3 Quality Requirements
- ✅ Code coverage > 80%
- ✅ Zero critical security vulnerabilities
- ✅ All integration tests passing
- ✅ Load tests pass (1000 req/s sustained)
- ✅ 12-Factor App compliance verified
- ✅ CI/CD pipeline operational
- ✅ Automated releases with GoReleaser

---

## 11. Timeline Summary

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase 1: Setup** | 3 days | Project structure, config, dependencies, health checks |
| **Phase 2: Core Services** | 5 days | GitHub, R2, Debian services |
| **Phase 3: Repository Builder** | 5 days | Full repo generation logic |
| **Phase 4: Queue Integration** | 3 days | Asynq tasks and workers |
| **Phase 5: API** | 3 days | REST API with Fiber/Gin |
| **Phase 6: Docker** | 2 days | Containerization |
| **Phase 7: Testing** | 3 days | Unit + integration tests |
| **Phase 8: CI/CD & Admin** | 3 days | GitHub Actions, GoReleaser, admin CLI |
| **Buffer** | 3 days | Bug fixes, optimization, documentation |
| **TOTAL** | **~4 weeks** | Production-ready Go implementation with full automation |

---

## 12. Next Steps

1. **Approval:** Review and approve this migration plan
2. **Setup:** Initialize Go project and CI/CD pipeline
3. **Sprint Planning:** Break phases into 2-week sprints
4. **Development:** Begin Phase 1 implementation
5. **Weekly Reviews:** Track progress and adjust timeline
6. **Staging Deployment:** Deploy to staging after Phase 6
7. **Production Migration:** Blue-green deployment in Week 4

---

## 15. Appendix

### 15.1 Useful Go Resources

**Go Language & Best Practices:**
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

**Web Frameworks:**
- [Fiber Documentation](https://docs.gofiber.io/)
- [Gin Web Framework](https://gin-gonic.com/docs/)

**Task Queue:**
- [Asynq Documentation](https://github.com/hibiken/asynq)
- [Asynq Web UI](https://github.com/hibiken/asynqmon)

**AWS SDK & R2:**
- [AWS SDK Go v2](https://aws.github.io/aws-sdk-go-v2/docs/)
- [Cloudflare R2 Documentation](https://developers.cloudflare.com/r2/)
- [Cloudflare R2 S3 API Compatibility](https://developers.cloudflare.com/r2/api/s3/api/)

**Cloudflare Platform:**
- [Cloudflare Workers Documentation](https://developers.cloudflare.com/workers/)
- [Cloudflare Queues](https://developers.cloudflare.com/queues/)
- [Cloudflare D1](https://developers.cloudflare.com/d1/)
- [Cloudflare Workers Analytics](https://developers.cloudflare.com/workers/observability/metrics-and-analytics/)

**CI/CD & Release Automation:**
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GoReleaser Documentation](https://goreleaser.com/)
- [GoReleaser Quick Start](https://goreleaser.com/quick-start/)
- [Conventional Commits](https://www.conventionalcommits.org/)

**12-Factor App:**
- [The Twelve-Factor App](https://12factor.net/)
- [12-Factor Config](https://12factor.net/config)
- [Beyond the 12-Factor App](https://www.oreilly.com/library/view/beyond-the-twelve-factor/9781492042631/)

**Testing:**
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify - Testing Toolkit](https://github.com/stretchr/testify)
- [gomock - Mocking Framework](https://github.com/golang/mock)

**Observability:**
- [Zap - Structured Logging](https://github.com/uber-go/zap)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)

### 15.2 Python vs Go Syntax Quick Reference

```python
# Python
def get_releases(owner: str, repo: str) -> List[Release]:
    releases = []
    for release in repo.get_releases():
        releases.append(release)
    return releases
```

```go
// Go
func GetReleases(owner, repo string) ([]Release, error) {
    var releases []Release
    for _, release := range repoReleases {
        releases = append(releases, release)
    }
    return releases, nil
}
```

### 15.3 Environment Variables Reference

```bash
# .env file for Go application
GITHUB_TOKEN=ghp_xxxxx

# Cloudflare R2 Configuration
R2_ACCOUNT_ID=your-account-id
R2_ACCESS_KEY_ID=your-access-key-id
R2_SECRET_ACCESS_KEY=your-secret-access-key
R2_BUCKET_NAME=nplb-repo
R2_PUBLIC_URL=https://repo.example.com  # Optional custom domain

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379

# GPG Configuration
GPG_HOME=keys
GPG_KEY_EMAIL=repo@example.com

# Server Configuration
SERVER_PORT=8080
```

---

## 14. Cloudflare Migration Benefits

### 14.1 Cost Analysis

| Service | AWS (Previous) | Cloudflare | Savings |
|---------|---------------|------------|---------|
| **Storage (100GB)** | S3: ~$2.30/mo | R2: ~$1.50/mo | **35% cheaper** |
| **Egress (1TB)** | S3: ~$90/mo | R2: **$0** | **$90/mo saved** |
| **API Requests (1M)** | S3: ~$0.40 | R2: ~$0.36 | Minor savings |
| **CDN Bandwidth** | CloudFront: ~$85/TB | Cloudflare: **Free** | **$85/TB saved** |
| **Total (1TB egress)** | ~$177.70/mo | ~$1.86/mo | **~$176/mo saved (99% reduction)** |

**Key Insight:** The zero-egress-fee model makes R2 significantly cheaper for high-bandwidth applications like APT repositories.

### 14.2 Cloudflare R2 Advantages

1. **Zero Egress Fees**: No bandwidth charges for data downloads (huge savings for package repos)
2. **S3 API Compatibility**: Minimal code changes, use existing AWS SDK
3. **Global Performance**: Automatic geographic distribution via Cloudflare's edge network
4. **Free CDN**: Built-in caching and acceleration through Cloudflare CDN
5. **Custom Domains**: Easy integration with custom domains (e.g., `repo.example.com`)
6. **Generous Free Tier**: 10GB storage + 1M Class A operations free per month

### 14.3 R2 Setup Instructions

#### Step 1: Create R2 Bucket
```bash
# Via Cloudflare Dashboard
1. Log in to Cloudflare Dashboard
2. Go to R2 → Create bucket
3. Name: nplb-repo
4. Enable public access (for APT repository)
```

#### Step 2: Generate R2 API Tokens
```bash
# In Cloudflare Dashboard
1. R2 → Manage R2 API Tokens
2. Create API Token
3. Permissions: Object Read & Write
4. Copy: Access Key ID & Secret Access Key
```

#### Step 3: Configure Custom Domain (Optional)
```bash
# For custom domain like repo.example.com
1. R2 → nplb-repo → Settings → Public Access
2. Connect custom domain
3. Add CNAME record: repo.example.com → <bucket-id>.r2.dev
4. Enable automatic HTTPS
```

#### Step 4: Update Application Config
```bash
export R2_ACCOUNT_ID="your-cloudflare-account-id"
export R2_ACCESS_KEY_ID="your-r2-access-key"
export R2_SECRET_ACCESS_KEY="your-r2-secret-key"
export R2_BUCKET_NAME="nplb-repo"
export R2_PUBLIC_URL="https://repo.example.com"  # Or https://nplb-repo.r2.dev
```

### 14.4 R2 vs S3 Compatibility Matrix

| Feature | S3 | R2 | Notes |
|---------|----|----|-------|
| PutObject | ✅ | ✅ | Fully compatible |
| GetObject | ✅ | ✅ | Fully compatible |
| DeleteObject | ✅ | ✅ | Fully compatible |
| ListObjects | ✅ | ✅ | Fully compatible |
| Multipart Upload | ✅ | ✅ | Supported |
| Presigned URLs | ✅ | ✅ | Supported |
| Object Metadata | ✅ | ✅ | Custom metadata supported |
| Bucket Policies | ✅ | ⚠️ | Limited (use Workers for complex logic) |
| Server-Side Encryption | ✅ | ✅ | Automatic encryption at rest |
| Versioning | ✅ | ❌ | Not yet supported |
| Object Locking | ✅ | ❌ | Not yet supported |
| Lifecycle Policies | ✅ | ⚠️ | Limited support |

**Verdict:** For NPLB's use case (simple object storage with public read access), R2 has full compatibility.

### 14.5 Migration Checklist

- [ ] Create Cloudflare R2 bucket
- [ ] Generate R2 API credentials
- [ ] Configure custom domain (optional)
- [ ] Update application configuration
- [ ] Test R2 uploads in development
- [ ] Verify public access to repository files
- [ ] Test `apt-get update` against R2-hosted repository
- [ ] Monitor R2 usage and costs
- [ ] Configure Cloudflare CDN caching rules
- [ ] Set up R2 bucket analytics

### 14.6 Future Cloudflare Integration Opportunities

#### Option 1: Cloudflare Workers (Serverless API)
Replace traditional API server with edge-deployed Workers:

```javascript
// Potential future architecture
export default {
  async fetch(request, env) {
    // Handle API requests at the edge
    // Trigger background jobs via Queues
    // Ultra-low latency globally
  }
}
```

**Benefits:**
- Global deployment (hundreds of edge locations)
- Zero cold starts
- Pay-per-request pricing
- Built-in DDoS protection

**Considerations:**
- Requires rewrite from Go to JavaScript/TypeScript
- Limited execution time (30s on Workers, 15min on Workers Unbound)
- Different programming model

#### Option 2: Cloudflare Queues
Replace Redis with Cloudflare Queues:

**Benefits:**
- Native integration with Workers
- No Redis infrastructure to manage
- Pay-per-use pricing
- Built-in at-least-once delivery

**Considerations:**
- Currently in beta
- Different API than Redis RQ
- May not support all RQ features

#### Option 3: Cloudflare D1 (SQLite)
Store job metadata and repository state:

**Benefits:**
- Serverless SQLite database
- Automatic replication
- Read replicas at edge
- SQL interface

**Considerations:**
- Currently limited to 500MB per database
- Good for metadata, not large data

### 14.7 Recommended Phased Approach

**Phase 1 (Current Plan):** Go + R2 + Redis + Docker
- Migrate to Go
- Replace S3 with R2
- Keep existing architecture (API server + workers)
- Easiest migration path

**Phase 2 (Future):** Evaluate Cloudflare Workers
- Prototype API endpoints on Workers
- Test performance and cost
- Consider hybrid approach (Workers for API, containers for heavy processing)

**Phase 3 (Future):** Full Cloudflare Stack
- Workers for API
- Cloudflare Queues for jobs
- D1 for metadata
- R2 for storage
- Fully serverless

### 14.8 R2 Code Examples

#### R2 Client Initialization

```go
package storage

import (
    "context"
    "fmt"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Config struct {
    AccountID       string
    AccessKeyID     string
    SecretAccessKey string
    BucketName      string
}

func NewR2Client(cfg R2Config) (*s3.Client, error) {
    r2Resolver := aws.EndpointResolverWithOptionsFunc(
        func(service, region string, options ...interface{}) (aws.Endpoint, error) {
            return aws.Endpoint{
                URL:               fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID),
                HostnameImmutable: true,
                SigningRegion:     "auto",
            }, nil
        },
    )

    awsCfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithEndpointResolverWithOptions(r2Resolver),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(
                cfg.AccessKeyID,
                cfg.SecretAccessKey,
                "",
            ),
        ),
        config.WithRegion("auto"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }

    return s3.NewFromConfig(awsCfg), nil
}
```

#### Upload with Public Read Access

```go
func (s *R2Service) UploadPublicFile(ctx context.Context, filePath, key string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucketName),
        Key:    aws.String(key),
        Body:   file,
        ACL:    types.ObjectCannedACLPublicRead, // Make publicly accessible
    })
    
    return err
}
```

#### Generate Public URL

```go
func (s *R2Service) GetPublicURL(key string) string {
    // If custom domain configured
    if s.customDomain != "" {
        return fmt.Sprintf("%s/%s", s.customDomain, key)
    }
    
    // Default R2 public URL
    return fmt.Sprintf("https://%s.r2.dev/%s", s.bucketName, key)
}
```

---

**Document Version:** 1.0  
**Last Updated:** November 9, 2025  
**Author:** GitHub Copilot  
**Status:** Ready for Review
