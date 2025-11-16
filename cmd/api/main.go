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
	// "github.com/zjpiazza/nplb/internal/api/handlers"
	// "github.com/zjpiazza/nplb/internal/api/routes"
	// "github.com/zjpiazza/nplb/internal/config"
	// "github.com/zjpiazza/nplb/internal/queue"
)

const version = "0.1.0" // Set by GoReleaser

func main() {
	// TODO: Implement config loading
	// cfg, err := config.Load()
	// if err != nil {
	// 	log.Fatal("Failed to load config:", err)
	// }

	// TODO: Initialize logger
	// var logger *zap.Logger
	// if cfg.LogFormat == "json" {
	// 	logger, _ = zap.NewProduction()
	// } else {
	// 	logger, _ = zap.NewDevelopment()
	// }
	// defer logger.Sync()

	// logger.Info("Starting NPLB API",
	// 	zap.String("version", version),
	// 	zap.String("environment", cfg.AppEnv),
	// )

	// TODO: Initialize queue client
	// queueClient := queue.NewClient(cfg)
	// defer queueClient.Close()

	// TODO: Initialize handlers
	// handler := handlers.NewHandler(queueClient, logger)
	// healthHandler := handlers.NewHealthHandler(redisClient, logger, version)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		AppName:      fmt.Sprintf("NPLB API v%s", version),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	// TODO: Add health check endpoints
	// app.Get("/health", healthHandler.Health)
	// app.Get("/health/live", healthHandler.Liveness)
	// app.Get("/health/ready", healthHandler.Readiness)

	// TODO: Setup routes
	// routes.Setup(app, handler)

	// Start server in goroutine
	// addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	addr := ":8080" // Placeholder
	go func() {
		log.Println("API server listening on", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Println("Server forced to shutdown:", err)
	}

	log.Println("Server stopped")
}
