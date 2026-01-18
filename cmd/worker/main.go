package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/zjpiazza/nplb/internal/config"
	"github.com/zjpiazza/nplb/internal/queue"
	"github.com/zjpiazza/nplb/internal/services/debian"
	"github.com/zjpiazza/nplb/internal/services/github"
	"github.com/zjpiazza/nplb/internal/services/repository"
	"github.com/zjpiazza/nplb/internal/services/storage"
	"github.com/zjpiazza/nplb/internal/tasks"
	"github.com/zjpiazza/nplb/pkg/gpg"
)

const version = "0.1.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Validate worker configuration
	if err := cfg.ValidateWorker(); err != nil {
		log.Fatal("Invalid worker configuration:", err)
	}

	// Initialize logger
	var logger *zap.Logger
	if cfg.LogFormat == "json" {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	logger.Info("Starting NPLB Worker",
		zap.String("version", version),
		zap.String("environment", cfg.AppEnv),
	)

	// Initialize services
	githubSvc := github.NewService(cfg.GitHubToken, logger)
	debianSvc := debian.NewService(logger)

	storageSvc, err := storage.NewR2Storage(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		cfg.R2BucketName,
		cfg.R2PublicURL,
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to initialize storage service", zap.Error(err))
	}

	// Initialize GPG signer
	gpgSigner := gpg.NewSigner(cfg.GPGHome, cfg.GPGKeyEmail, logger)

	// Ensure GPG home directory exists
	if err := gpg.EnsureGPGHome(cfg.GPGHome); err != nil {
		logger.Warn("Failed to create GPG home directory", zap.Error(err))
	}

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		logger.Fatal("Failed to create output directory", zap.Error(err))
	}

	// Initialize repository builder
	builder := repository.NewBuilder(
		githubSvc,
		debianSvc,
		storageSvc,
		gpgSigner,
		logger,
		cfg.OutputDir,
		cfg.DefaultCodename,
	)

	// Create task handler
	taskHandler := tasks.NewTaskHandler(builder, logger)

	// Create Asynq server
	server := queue.NewAsynqServer(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, 10)

	// Create mux and register handlers
	mux := asynq.NewServeMux()
	taskHandler.RegisterHandlers(mux)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start worker in goroutine
	go func() {
		logger.Info("Worker started",
			zap.String("redis_addr", cfg.RedisAddr),
		)
		if err := server.Run(mux); err != nil {
			logger.Fatal("Worker failed to start", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("Shutting down worker gracefully...")

	server.Shutdown()

	logger.Info("Worker stopped")
}
