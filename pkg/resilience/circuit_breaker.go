package resilience

import (
	"time"

	"github.com/devgugga/galaxy-pay/pkg/logger"
	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name          string
	MaxRequests   uint32        // Number of requests allowed when half-open
	Interval      time.Duration // Interval to reset counters
	Timeout       time.Duration // Timeout to try closing from half-open
	ReadyToTrip   func(counts gobreaker.Counts) bool
	OnStateChange func(name string, from, to gobreaker.State)
}

// DefaultCircuitBreakerConfig returns a default configuration optimized for payment APIs
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 3, // Allow 3 requests when half-open
		Interval:    60 * time.Second, // Reset counters every 60s
		Timeout:     30 * time.Second,  // Wait 30s before trying to close
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit if:
			// - More than 5 consecutive failures OR
			// - Failure rate >= 50% with at least 10 requests
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.ConsecutiveFailures > 5 ||
				(counts.Requests >= 10 && failureRatio >= 0.5)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Log.Warn("Circuit breaker state changed",
				logger.String("name", name),
				logger.String("from_state", from.String()),
				logger.String("to_state", to.String()),
			)
		},
	}
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: cfg.ReadyToTrip,
	}

	if cfg.OnStateChange != nil {
		settings.OnStateChange = func(name string, from, to gobreaker.State) {
			cfg.OnStateChange(name, from, to)
		}
	}

	return gobreaker.NewCircuitBreaker(settings)
}

