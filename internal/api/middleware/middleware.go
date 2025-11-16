package middleware

import "github.com/gofiber/fiber/v2"

// TODO: Implement any custom middleware here.
// For example, you could add middleware for:
// - Authentication/Authorization
// - Request logging
// - Rate limiting

// ExampleMiddleware is a placeholder for a custom middleware.
func ExampleMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// You can add logic here to execute before the request is handled.
		// For example, you could log the request or check an API key.
		return c.Next()
	}
}
