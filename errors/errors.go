// Package errors defines the error types returned by the Assinafy SDK.
package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
)

// APIError represents a non-success HTTP response or API-envelope status.
type APIError struct {
	// StatusCode is the HTTP or API-envelope status code.
	StatusCode int `json:"status"`
	// Message is the human-readable error message returned by the API.
	Message string `json:"message"`
	// Data contains optional structured error details from the response envelope.
	Data any `json:"data,omitempty"`
	// Headers contains the HTTP response headers, including Retry-After and
	// provider request IDs when supplied.
	Headers http.Header `json:"-"`
}

// Error formats the API message and status code.
func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("assinafy: api error (status %d)", e.StatusCode)
	}
	return fmt.Sprintf("assinafy: %s (status %d)", e.Message, e.StatusCode)
}

// NetworkError wraps transport-level failures.
type NetworkError struct {
	// Err is the wrapped transport or context error.
	Err error
}

// Error formats the wrapped network failure.
func (e *NetworkError) Error() string {
	if e.Err == nil {
		return "assinafy: network error"
	}
	return "assinafy: network error: " + e.Err.Error()
}

// Unwrap returns the underlying transport or context error.
func (e *NetworkError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsStatusCode reports whether err is an APIError with the given status code.
func IsStatusCode(err error, code int) bool {
	var apiErr *APIError
	return stderrors.As(err, &apiErr) && apiErr != nil && apiErr.StatusCode == code
}

// IsRetryable reports whether err represents a transient failure (HTTP 429 or 5xx).
func IsRetryable(err error) bool {
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr *APIError
	if stderrors.As(err, &apiErr) && apiErr != nil {
		return apiErr.StatusCode == http.StatusTooManyRequests ||
			(apiErr.StatusCode >= http.StatusInternalServerError && apiErr.StatusCode < 600)
	}
	var netErr *NetworkError
	return stderrors.As(err, &netErr) && netErr != nil
}
