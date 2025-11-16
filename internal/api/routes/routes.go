package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zjpiazza/nplb/internal/api/handlers"
)

// Setup configures the routes for the application.
func Setup(app *fiber.App, healthHandler *handlers.HealthHandler) {
	// Group API routes
	api := app.Group("/api")

	// Version 1 of the API
	v1 := api.Group("/v1")

	// Health check endpoints
	health := v1.Group("/health")
	health.Get("/", healthHandler.Health)
	health.Get("/live", healthHandler.Liveness)
	health.Get("/ready", healthHandler.Readiness)

	// You can add more routes here as needed.
}
