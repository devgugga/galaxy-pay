package database

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/devgugga/galaxy-pay/ent"
	"github.com/devgugga/galaxy-pay/internal/config"
	"github.com/devgugga/galaxy-pay/pkg/logger"
	_ "github.com/lib/pq"
)

var (
	// Client is the global Ent client instance
	Client *ent.Client
)

// InitEntClient initializes the Ent client with PostgreSQL connection
func InitEntClient(cfg *config.Config) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	// Open database connection
	drv, err := entsql.Open(dialect.Postgres, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed opening database connection: %w", err)
	}

	// Configure connection pool
	db := drv.DB()
	db.SetMaxOpenConns(cfg.DatabaseMaxConns)
	db.SetMaxIdleConns(cfg.DatabaseMaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 10)

	// Create Ent client
	Client = ent.NewClient(ent.Driver(drv))

	// Run auto migration in development
	if cfg.IsDevelopment() {
		logger.Log.Info("Running database migrations (development mode)...")
		if err := Client.Schema.Create(context.Background()); err != nil {
			return fmt.Errorf("failed creating schema resources: %w", err)
		}
		logger.Log.Info("Database migrations completed")
	}

	// Test connection by querying the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Database connection established",
		logger.String("max_conns", fmt.Sprintf("%d", cfg.DatabaseMaxConns)),
		logger.String("max_idle_conns", fmt.Sprintf("%d", cfg.DatabaseMaxIdleConns)),
	)

	return nil
}

// Close closes the database connection
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

// HealthCheck checks if the database is healthy
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("database client is not initialized")
	}
	// Use a simple query to check database health
	_, err := Client.IdempotencyKey.Query().Limit(1).All(ctx)
	return err
}

