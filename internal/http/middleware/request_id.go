package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the key used to store request ID in context locals
	RequestIDKey = "request_id"
)

// RequestID generates or extracts a request ID for each request
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get request ID from header
		requestID := c.Get(RequestIDHeader)
		
		// If not present, generate a new UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Store in context locals for use in handlers and other middlewares
		c.Locals(RequestIDKey, requestID)
		
		// Set response header
		c.Set(RequestIDHeader, requestID)
		
		return c.Next()
	}
}

// GetRequestID retrieves the request ID from context locals
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

