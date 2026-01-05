package resilience

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// RetryConfig holds configuration for retry with exponential backoff
type RetryConfig struct {
	MaxAttempts         int           // Maximum number of attempts
	InitialInterval     time.Duration // Initial interval
	MaxInterval         time.Duration // Maximum interval
	Multiplier          float64       // Exponential multiplier
	RandomizationFactor float64      // Randomization factor (jitter) - 0.0 to 1.0
	MaxElapsedTime      time.Duration // Maximum total elapsed time
}

// DefaultRetryConfig returns a default configuration for payment APIs
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:         5,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         10 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.5, // 50% jitter to avoid thundering herd
		MaxElapsedTime:      30 * time.Second,
	}
}

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network/timeout errors are retryable
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return true
	}

	// HTTP errors that are retryable
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.IsRetryable()
	}

	// Check for temporary network errors
	var netErr interface {
		Timeout() bool
		Temporary() bool
	}
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

// RetryWithBackoff executes a function with retry and exponential backoff
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func() error) error {
	// Create exponential backoff
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = cfg.InitialInterval
	b.MaxInterval = cfg.MaxInterval
	b.Multiplier = cfg.Multiplier
	b.RandomizationFactor = cfg.RandomizationFactor
	b.MaxElapsedTime = cfg.MaxElapsedTime
	b.Reset()

	// Add context support and max retries
	// MaxInterval is already set in the ExponentialBackOff, so we don't need WithCappedDuration
	withContext := backoff.WithContext(
		backoff.WithMaxRetries(
			b,
			uint64(cfg.MaxAttempts-1), // -1 because first attempt doesn't count
		),
		ctx,
	)

	operation := func() error {
		err := fn()
		if err != nil && !IsRetryableError(err) {
			// Non-retryable error, stop immediately
			return backoff.Permanent(err)
		}
		return err
	}

	return backoff.Retry(operation, withContext)
}

// IsHTTPRetryable checks if an HTTP status code should be retried
func IsHTTPRetryable(statusCode int) bool {
	// 429 (Too Many Requests) and 5xx (Server Errors) are retryable
	return statusCode == http.StatusTooManyRequests ||
		(statusCode >= http.StatusInternalServerError && statusCode < 600)
}

