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
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/zjpiazza/nplb/internal/api/handlers"
	"github.com/zjpiazza/nplb/internal/api/routes"
	"github.com/zjpiazza/nplb/internal/config"
	"github.com/zjpiazza/nplb/internal/queue"
)

const version = "0.1.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize logger
	var zapLogger *zap.Logger
	if cfg.LogFormat == "json" {
		zapLogger, _ = zap.NewProduction()
	} else {
		zapLogger, _ = zap.NewDevelopment()
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting NPLB API",
		zap.String("version", version),
		zap.String("environment", cfg.AppEnv),
	)

	// Initialize Redis client for health checks
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		zapLogger.Warn("Redis connection failed - health checks will report degraded",
			zap.Error(err),
		)
	}
	cancel()

	// Initialize queue client (Cloudflare Queues)
	var queueClient queue.Queue
	if cfg.CloudflareAPIToken != "" {
		queueClient, err = queue.NewClient(cfg)
		if err != nil {
			zapLogger.Warn("Failed to initialize Cloudflare Queue client",
				zap.Error(err),
			)
		}
	}

	// If Cloudflare Queue is not available, use Asynq
	if queueClient == nil {
		zapLogger.Info("Using Asynq queue (Redis-based)")
		queueClient = queue.NewAsynqClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	}
	defer queueClient.Close()

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(redisClient, zapLogger, version)
	handler := handlers.NewHandler(queueClient, zapLogger)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		AppName:               fmt.Sprintf("NPLB API v%s", version),
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		DisableStartupMessage: cfg.IsProduction(),
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(cors.New())

	if cfg.IsDevelopment() {
		app.Use(logger.New())
	}

	// Setup routes
	routes.Setup(app, healthHandler, handler)

	// Start server in goroutine
	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	go func() {
		zapLogger.Info("API server listening", zap.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			zapLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	zapLogger.Info("Shutting down server gracefully...")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("Server stopped")
}
