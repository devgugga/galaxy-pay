package middleware

import (
	"context"
	"time"

	"github.com/devgugga/galaxy-pay/internal/infrastructure/cache"
	"github.com/devgugga/galaxy-pay/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

const (
	// IdempotencyKeyHeader is the header name for idempotency key
	IdempotencyKeyHeader = "Idempotency-Key"
)

// Idempotency creates a middleware that handles idempotency for payment operations
func Idempotency(store *cache.IdempotencyStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only apply to methods that modify state
		if c.Method() != "POST" && c.Method() != "PUT" && c.Method() != "PATCH" {
			return c.Next()
		}

		// Get idempotency key from header
		idempotencyKey := c.Get(IdempotencyKeyHeader)
		if idempotencyKey == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   true,
				"message": "Idempotency-Key header is required for this operation",
			})
		}

		// Validate idempotency key format (should be a UUID)
		if len(idempotencyKey) < 32 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid Idempotency-Key format",
			})
		}

		requestID := GetRequestID(c)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Check if response exists (Cache-Aside: Redis -> DB)
		cachedResponse, err := store.WithRequestID(requestID).Get(ctx, idempotencyKey)
		if err != nil {
			logger.WithRequestID(requestID).Error("Failed to check idempotency key",
				logger.Error(err),
				logger.String("idempotency_key", idempotencyKey),
			)
			// Continue processing if Redis error (fail open)
			return c.Next()
		}

		// If cached response exists, return it
		if cachedResponse != nil {
			logger.WithRequestID(requestID).Info("Returning cached idempotent response",
				logger.String("idempotency_key", idempotencyKey),
				logger.Int("status", cachedResponse.StatusCode),
			)

			// Set headers
			for key, value := range cachedResponse.Headers {
				c.Set(key, value)
			}

			// Return cached response
			c.Status(cachedResponse.StatusCode)
			return c.Send(cachedResponse.Body)
		}

		// No cached response, process request and capture response
		// Process request
		err = c.Next()
		if err != nil {
			return err
		}

		// Capture response after processing
		statusCode := c.Response().StatusCode()
		responseBody := make([]byte, len(c.Response().Body()))
		copy(responseBody, c.Response().Body())

		// Store response in cache
		response := &cache.IdempotencyResponse{
			StatusCode: statusCode,
			Headers:    make(map[string]string),
			Body:       responseBody,
		}

		// Copy headers (excluding some that shouldn't be cached)
		c.Response().Header.VisitAll(func(key, value []byte) {
			keyStr := string(key)
			// Skip headers that shouldn't be cached
			if keyStr != "X-Request-ID" && keyStr != "Date" && keyStr != "Server" {
				response.Headers[keyStr] = string(value)
			}
		})

		// Store response (Write-Through: DB first, then Redis async)
		// This is done synchronously to ensure DB persistence, Redis is async
		if err := store.WithRequestID(requestID).Set(ctx, idempotencyKey, response); err != nil {
			logger.WithRequestID(requestID).Error("Failed to save idempotent response",
				logger.Error(err),
				logger.String("idempotency_key", idempotencyKey),
			)
			// Continue even if save fails (fail-open)
		}

		return nil
	}
}

