package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	AppName    string
	AppVersion string
	AppEnv     string
	Host       string
	Port       string

	// Server timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Body limit in bytes
	BodyLimit int

	// Logging
	LogLevel string

	// Redis configuration
	RedisHost         string
	RedisPort         string
	RedisPassword     string
	RedisDB           int
	RedisPoolSize     int
	RedisMinIdleConns int

	// Rate limiting
	RateLimitMax      int
	RateLimitDuration time.Duration

	// External API configuration
	ExternalAPITimeout time.Duration // Default timeout for external API calls

	// Database configuration
	DatabaseURL        string
	DatabaseMaxConns   int
	DatabaseMaxIdleConns int
}

// Load loads configuration from environment variables
// It loads .env file if it exists, then reads environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		AppName:    getEnv("APP_NAME", "Galaxy Pay"),
		AppVersion: getEnv("APP_VERSION", "1.0.0"),
		AppEnv:     getEnv("APP_ENV", "development"),
		Host:       getEnv("HOST", "0.0.0.0"),
		Port:       getEnv("PORT", "8080"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
	}

	// Parse timeouts
	readTimeout := getEnvAsInt("READ_TIMEOUT", 15)
	writeTimeout := getEnvAsInt("WRITE_TIMEOUT", 15)
	idleTimeout := getEnvAsInt("IDLE_TIMEOUT", 60)

	cfg.ReadTimeout = time.Duration(readTimeout) * time.Second
	cfg.WriteTimeout = time.Duration(writeTimeout) * time.Second
	cfg.IdleTimeout = time.Duration(idleTimeout) * time.Second

	// Parse body limit (default 4MB)
	bodyLimitMB := getEnvAsInt("BODY_LIMIT_MB", 4)
	cfg.BodyLimit = bodyLimitMB * 1024 * 1024

	// Redis configuration
	cfg.RedisHost = getEnv("REDIS_HOST", "localhost")
	cfg.RedisPort = getEnv("REDIS_PORT", "6379")
	cfg.RedisPassword = getEnv("REDIS_PASSWORD", "")
	cfg.RedisDB = getEnvAsInt("REDIS_DB", 0)
	cfg.RedisPoolSize = getEnvAsInt("REDIS_POOL_SIZE", 10)
	cfg.RedisMinIdleConns = getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5)

	// Rate limiting
	cfg.RateLimitMax = getEnvAsInt("RATE_LIMIT_MAX", 100)
	rateLimitDurationSeconds := getEnvAsInt("RATE_LIMIT_DURATION", 60)
	cfg.RateLimitDuration = time.Duration(rateLimitDurationSeconds) * time.Second

	// External API timeout (default 30 seconds)
	externalAPITimeoutSeconds := getEnvAsInt("EXTERNAL_API_TIMEOUT", 30)
	cfg.ExternalAPITimeout = time.Duration(externalAPITimeoutSeconds) * time.Second

	// Database configuration
	cfg.DatabaseURL = getEnv("DATABASE_URL", "")
	cfg.DatabaseMaxConns = getEnvAsInt("DATABASE_MAX_CONNS", 25)
	cfg.DatabaseMaxIdleConns = getEnvAsInt("DATABASE_MAX_IDLE_CONNS", 5)

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetServerAddress returns the full server address (host:port)
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// IsProduction returns true if the app is running in production
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsDevelopment returns true if the app is running in development
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// GetRedisAddress returns the full Redis address (host:port)
func (c *Config) GetRedisAddress() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

