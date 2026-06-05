// Package errors defines the error types returned by the Assinafy SDK.
package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

// APIError represents a non-2xx response from the Assinafy API.
type APIError struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
	Data       any    `json:"data,omitempty"`
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("assinafy: api error (status %d)", e.StatusCode)
	}
	return fmt.Sprintf("assinafy: %s (status %d)", e.Message, e.StatusCode)
}

// NetworkError wraps transport-level failures.
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string {
	if e.Err == nil {
		return "assinafy: network error"
	}
	return "assinafy: network error: " + e.Err.Error()
}

func (e *NetworkError) Unwrap() error { return e.Err }

// IsStatusCode reports whether err is an APIError with the given status code.
func IsStatusCode(err error, code int) bool {
	var apiErr *APIError
	return stderrors.As(err, &apiErr) && apiErr.StatusCode == code
}

// IsRetryable reports whether err represents a transient failure (HTTP 429 or 5xx).
func IsRetryable(err error) bool {
	var apiErr *APIError
	if stderrors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
	}
	var netErr *NetworkError
	return stderrors.As(err, &netErr)
}
