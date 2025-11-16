package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	// "github.com/zjpiazza/nplb/internal/config"
	// "github.com/zjpiazza/nplb/internal/queue"
	// "github.com/zjpiazza/nplb/internal/tasks"
)

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

	// logger.Info("Starting NPLB Worker")

	// TODO: Initialize services (GitHub, R2, etc.)
	// ... service initialization code ...

	// Create task handler
	mux := asynq.NewServeMux()
	// mux.HandleFunc(tasks.TypeBuildRepository, func(ctx context.Context, t *asynq.Task) error {
	// 	return tasks.HandleBuildRepositoryTask(ctx, t, githubService, storageService, cfg, logger)
	// })

	// TODO: Initialize worker server
	// server := queue.NewServer(cfg)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// TODO: Start worker in goroutine
	// go func() {
	// 	logger.Info("Worker started")
	// 	if err := server.Run(mux); err != nil {
	// 		logger.Fatal("Worker failed to start", zap.Error(err))
	// 	}
	// }()

	<-quit
	log.Println("Shutting down worker gracefully...")

	// TODO: Shutdown worker
	// server.Shutdown()

	log.Println("Worker stopped")
}
