package middleware

import (
	"time"

	"github.com/devgugga/galaxy-pay/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

// ZapLogger is a middleware that logs HTTP requests using Zap
func ZapLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		// Process request
		err := c.Next()
		
		// Calculate latency
		latency := time.Since(start)
		
		// Get request ID
		requestID := GetRequestID(c)
		
		// Log based on status code
		status := c.Response().StatusCode()
		log := logger.WithRequestID(requestID)
		
		switch {
		case status >= 500:
			log.Error("HTTP Request",
				logger.Method(c.Method()),
				logger.Path(c.Path()),
				logger.Status(status),
				logger.Latency(latency.String()),
				logger.IP(c.IP()),
			)
		case status >= 400:
			log.Warn("HTTP Request",
				logger.Method(c.Method()),
				logger.Path(c.Path()),
				logger.Status(status),
				logger.Latency(latency.String()),
				logger.IP(c.IP()),
			)
		default:
			log.Info("HTTP Request",
				logger.Method(c.Method()),
				logger.Path(c.Path()),
				logger.Status(status),
				logger.Latency(latency.String()),
				logger.IP(c.IP()),
			)
		}
		
		return err
	}
}

