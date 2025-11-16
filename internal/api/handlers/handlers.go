package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/zjpiazza/nplb/internal/models"
	"github.com/zjpiazza/nplb/internal/queue"
	"github.com/zjpiazza/nplb/internal/tasks"
)

// HealthHandler provides handlers for health checks.
type HealthHandler struct {
	redisClient *redis.Client
	logger      *zap.Logger
	startTime   time.Time
	version     string
}

// NewHealthHandler creates a new HealthHandler.
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
	// This is a simple check to see if the server is up.
	return c.JSON(fiber.Map{
		"status":    "alive",
		"timestamp": time.Now().Unix(),
	})
}

// Readiness probe - is the application ready to serve traffic?
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	// This check should verify dependencies.
	// Since we are using Cloudflare Queues, we can't easily "ping" it.
	// A better check would be to see if we can communicate with the Cloudflare API.
	// For now, we'll just return ready.
	return c.JSON(fiber.Map{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	})
}

// Health provides a detailed health status.
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	// This handler can provide more detailed information about the
	// health of the service and its dependencies.
	health := fiber.Map{
		"status":    "healthy",
		"version":   h.version,
		"uptime":    time.Since(h.startTime).Seconds(),
		"timestamp": time.Now().Unix(),
		"checks": fiber.Map{
			"cloudflare_api": "unknown",
		},
	}

	// TODO: Implement a check to the Cloudflare API
	// For example, you could use `api.VerifyAPIToken`.
	health["checks"].(fiber.Map)["cloudflare_api"] = "healthy" // Placeholder

	return c.JSON(health)
}

type Handler struct {
	queueClient queue.Queue
	logger      *zap.Logger
}

func NewHandler(queueClient queue.Queue, logger *zap.Logger) *Handler {
	return &Handler{
		queueClient: queueClient,
		logger:      logger,
	}
}

func (h *Handler) CreateBuild(c *fiber.Ctx) error {
	// 1. Parse and validate the request body
	var req models.BuildRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse request body",
		})
	}

	// TODO: Add validation for the request
	// validate := validator.New()
	// if err := validate.Struct(&req); err != nil {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": err.Error(),
	// 	})
	// }

	// Set default limit if not provided
	if req.Limit == 0 {
		req.Limit = 10
	}

	// 2. Create a new build task payload
	taskPayload := tasks.BuildRepositoryPayload{
		Owner: req.Owner,
		Repo:  req.Repo,
		Limit: req.Limit,
	}

	// 3. Enqueue the task
	err := h.queueClient.Enqueue(c.Context(), tasks.TypeBuildRepository, taskPayload)
	if err != nil {
		h.logger.Error("failed to enqueue task", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to enqueue build task",
		})
	}

	// 4. Return a response
	// In a real application, the queue would return a job ID.
	// For now, we'll just return a success message.
	return c.Status(fiber.StatusAccepted).JSON(models.BuildResponse{
		Status:  "accepted",
		Message: fmt.Sprintf("repository build for %s/%s has been queued", req.Owner, req.Repo),
		JobID:   "not-implemented", // Placeholder
	})
}
