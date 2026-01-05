package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devgugga/galaxy-pay/internal/config"
	"github.com/devgugga/galaxy-pay/internal/http/middleware"
	"github.com/devgugga/galaxy-pay/internal/infrastructure/cache"
	"github.com/devgugga/galaxy-pay/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	json "github.com/goccy/go-json"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	if err := logger.Init(cfg.AppEnv, cfg.LogLevel); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Log.Info("Application starting",
		logger.String("app_name", cfg.AppName),
		logger.String("version", cfg.AppVersion),
		logger.String("env", cfg.AppEnv),
	)

	// Initialize Redis (required - application will exit if connection fails)
	logger.Log.Info("Connecting to Redis...",
		logger.String("address", cfg.GetRedisAddress()),
	)
	if err := cache.Init(cfg); err != nil {
		logger.Log.Fatal("Failed to initialize Redis - application cannot start without Redis",
			logger.Error(err),
		)
	}
	logger.Log.Info("Redis connected successfully")
	defer cache.Close()

	// Create idempotency store (will be used when payment routes are added)
	// var idempotencyStore *cache.IdempotencyStore
	// if cache.Client != nil {
	// 	idempotencyStore = cache.NewIdempotencyStore(cache.Client, 24*time.Hour)
	// }

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
	// 1. Recover (first - catches panics)
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			requestID := middleware.GetRequestID(c)
			logger.WithRequestID(requestID).Error("Panic recovered",
				logger.String("panic", fmt.Sprintf("%v", e)),
			)
		},
	}))

	// 2. Request ID (early - needed by other middlewares)
	app.Use(middleware.RequestID())

	// 3. Zap Logger (uses Request ID)
	app.Use(middleware.ZapLogger())

	// 4. Rate Limiting
	app.Use(middleware.RateLimit(cfg))

	// 5. Idempotency (only for payment routes - will be added later)
	// app.Use(middleware.Idempotency(idempotencyStore))

	// Setup routes
	setupRoutes(app, cfg)

	// Graceful shutdown
	go func() {
		address := cfg.GetServerAddress()
		logger.Log.Info("Server starting",
			logger.String("address", address),
		)
		if err := app.Listen(address); err != nil {
			logger.Log.Fatal("Server failed to start",
				logger.Error(err),
			)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Server shutting down...")
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		logger.Log.Fatal("Shutdown failed",
			logger.Error(err),
		)
	}

	logger.Log.Info("Server exited")
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

	// Get request ID
	requestID := middleware.GetRequestID(c)

	// Log error
	logger.WithRequestID(requestID).Error("HTTP Error",
		logger.Method(c.Method()),
		logger.Path(c.Path()),
		logger.Status(code),
		logger.Error(err),
	)

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
