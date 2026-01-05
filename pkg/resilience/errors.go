package resilience

import (
	"fmt"
	"net/http"
)

// HTTPError represents an HTTP error with status code and message
type HTTPError struct {
	StatusCode int
	Message    string
	Body       []byte
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("HTTP %d", e.StatusCode)
}

// IsRetryable determines if the HTTP error should be retried
// Returns true for:
// - 429 (Too Many Requests)
// - 5xx (Server Errors)
func (e *HTTPError) IsRetryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || 
	       (e.StatusCode >= http.StatusInternalServerError && e.StatusCode < 600)
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(statusCode int, message string, body []byte) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Body:       body,
	}
}

