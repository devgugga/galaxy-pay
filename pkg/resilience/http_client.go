package resilience

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// ResilientHTTPClient is an HTTP client with circuit breaker, retry and timeouts
type ResilientHTTPClient struct {
	httpClient     *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
	retryConfig    RetryConfig
	defaultTimeout time.Duration
}

// ResilientHTTPClientConfig holds configuration for the resilient HTTP client
type ResilientHTTPClientConfig struct {
	CircuitBreaker *gobreaker.CircuitBreaker
	RetryConfig    RetryConfig
	DefaultTimeout time.Duration
	Transport      http.RoundTripper
}

// NewResilientHTTPClient creates a new resilient HTTP client
func NewResilientHTTPClient(cfg ResilientHTTPClientConfig) *ResilientHTTPClient {
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout:   cfg.DefaultTimeout,
		Transport: cfg.Transport,
	}

	if cfg.Transport == nil {
		// Configure optimized transport
		transport := &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   false,
		}
		httpClient.Transport = transport
	}

	// Use default retry config if not provided
	retryCfg := cfg.RetryConfig
	if retryCfg.MaxAttempts == 0 {
		retryCfg = DefaultRetryConfig()
	}

	return &ResilientHTTPClient{
		httpClient:     httpClient,
		circuitBreaker: cfg.CircuitBreaker,
		retryConfig:    retryCfg,
		defaultTimeout: cfg.DefaultTimeout,
	}
}

// Do executes an HTTP request with circuit breaker, retry and timeout
func (c *ResilientHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Apply timeout from context or use default
	timeout := c.defaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = c.defaultTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req = req.WithContext(ctx)

	var resp *http.Response
	var err error

	// Execute with circuit breaker
	if c.circuitBreaker != nil {
		result, cbErr := c.circuitBreaker.Execute(func() (interface{}, error) {
			// Inside circuit breaker, apply retry
			retryErr := RetryWithBackoff(ctx, c.retryConfig, func() error {
				var retryErr error
				resp, retryErr = c.httpClient.Do(req)
				
				// Check if response indicates retryable error
				if retryErr == nil && resp != nil {
					if IsHTTPRetryable(resp.StatusCode) {
						// Read body before closing
						bodyBytes, _ := io.ReadAll(resp.Body)
						resp.Body.Close()
						
						retryErr = NewHTTPError(resp.StatusCode, http.StatusText(resp.StatusCode), bodyBytes)
						resp = nil
					}
				}
				
				return retryErr
			})
			return resp, retryErr
		})

		if cbErr != nil {
			return nil, cbErr
		}

		if result == nil {
			return nil, nil
		}

		resp = result.(*http.Response)
	} else {
		// Without circuit breaker, just retry
		err = RetryWithBackoff(ctx, c.retryConfig, func() error {
			var retryErr error
			resp, retryErr = c.httpClient.Do(req)
			
			// Check if response indicates retryable error
			if retryErr == nil && resp != nil {
				if IsHTTPRetryable(resp.StatusCode) {
					// Read body before closing
					bodyBytes, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					
					retryErr = NewHTTPError(resp.StatusCode, http.StatusText(resp.StatusCode), bodyBytes)
					resp = nil
				}
			}
			
			return retryErr
		})

		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

