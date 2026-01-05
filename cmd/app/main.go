package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devgugga/galaxy-pay/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	json "github.com/goccy/go-json"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Fiber with performance config
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName + " v" + cfg.AppVersion,
		ServerHeader: "Fiber",

		// Custom JSON encoder (performance)
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,

		// Error handling
		ErrorHandler: customErrorHandler,

		// Performance tuning
		Prefork:       false, // Set true for multi-process
		CaseSensitive: true,
		StrictRouting: false,

		// Timeouts from config
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,

		// Body limits from config
		BodyLimit: cfg.BodyLimit,

		// Disable startup message in production
		DisableStartupMessage: cfg.IsProduction(),
	})

	// Setup middleware (order matters!)
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Printf("Panic recovered: %v", e)
		},
	}))
	app.Use(logger.New())

	// Setup routes
	setupRoutes(app, cfg)

	// Graceful shutdown
	go func() {
		address := cfg.GetServerAddress()
		log.Printf("Server starting on %s", address)
		if err := app.Listen(address); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}

	log.Println("Server exited")
}

// customErrorHandler handles errors in a consistent way
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Extract Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log error
	log.Printf("[ERROR] %d - %s %s: %v", code, c.Method(), c.Path(), err)

	return c.Status(code).JSON(fiber.Map{
		"error":   true,
		"message": message,
		"status":  code,
		"path":    c.Path(),
	})
}

// setupRoutes configures all application routes
func setupRoutes(app *fiber.App, cfg *config.Config) {
	// Health check endpoint
	app.Get("/health", healthHandler)

	// Ready endpoint
	app.Get("/ready", readyHandler)

	// Root endpoint
	app.Get("/", rootHandler(cfg))
}

// healthHandler handles health check requests
func healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
	})
}

// readyHandler handles readiness check requests
func readyHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ready",
	})
}

// rootHandler handles root requests
func rootHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "galaxy-pay",
			"version": cfg.AppVersion,
		})
	}
}
