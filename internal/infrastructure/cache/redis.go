package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/devgugga/galaxy-pay/internal/config"
	"github.com/devgugga/galaxy-pay/pkg/logger"
	"github.com/redis/go-redis/v9"
)

var (
	// Client is the global Redis client instance
	Client *redis.Client
)

// Init initializes the Redis client with connection pool and retry logic
func Init(cfg *config.Config) error {
	const maxRetries = 5
	const retryDelay = 2 * time.Second

	Client = redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddress(),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		// Disable automatic reconnection to prevent flooding
		MaxRetries:      0,
		MinRetryBackoff: 0,
		MaxRetryBackoff: 0,
	})

	// Retry connection with exponential backoff
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create a new context for each attempt
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		logger.Log.Info("Attempting to connect to Redis",
			logger.Int("attempt", attempt),
			logger.Int("max_attempts", maxRetries),
			logger.String("address", cfg.GetRedisAddress()),
		)
		
		pingErr := Client.Ping(ctx).Err()
		cancel()
		
		if pingErr == nil {
			// Connection successful
			logger.Log.Info("Redis connection established",
				logger.Int("attempt", attempt),
			)
			return nil
		}

		lastErr = pingErr
		if attempt < maxRetries {
			waitTime := retryDelay * time.Duration(attempt)
			logger.Log.Warn("Redis connection failed, retrying...",
				logger.Int("attempt", attempt),
				logger.String("error", pingErr.Error()),
				logger.String("retry_in", waitTime.String()),
			)
			time.Sleep(waitTime)
		}
	}

	// Close client if all retries failed
	if Client != nil {
		_ = Client.Close()
		Client = nil
	}

	return fmt.Errorf("failed to connect to Redis after %d attempts: %w", maxRetries, lastErr)
}

// Close closes the Redis client connection
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

// HealthCheck checks if Redis is healthy
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("Redis client is not initialized")
	}
	return Client.Ping(ctx).Err()
}

