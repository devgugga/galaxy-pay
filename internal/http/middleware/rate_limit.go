package middleware

import (
	"github.com/devgugga/galaxy-pay/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimit creates a rate limiting middleware
func RateLimit(cfg *config.Config) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        cfg.RateLimitMax,
		Expiration: cfg.RateLimitDuration,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use IP address as key for rate limiting
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   true,
				"message": "Rate limit exceeded. Please try again later.",
				"status":  fiber.StatusTooManyRequests,
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
		Storage:                nil, // Uses in-memory storage by default
	})
}

