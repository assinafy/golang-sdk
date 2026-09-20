// Package errors defines the error types returned by the Assinafy SDK.
package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ErrInvalidInput identifies a request rejected locally before any network
// call because a required path, file, or other input is invalid.
var ErrInvalidInput = stderrors.New("assinafy: invalid input")

// ErrUnsafeRedirect identifies an unsafe request redirected across origins and
// rejected before the destination received it.
var ErrUnsafeRedirect = stderrors.New("assinafy: refused unsafe cross-origin redirect")

// DeletionRestriction describes one account-deletion blocker returned by the API.
type DeletionRestriction struct {
	// Code is ActivePaidSubscription or PendingDocuments.
	Code string `json:"code"`
	// Message explains how the restriction blocks deletion.
	Message string `json:"message"`
	// AccountIDs lists the affected account IDs.
	AccountIDs []string `json:"account_ids"`
}

// APIError represents a non-success HTTP response or API-envelope status.
type APIError struct {
	// StatusCode is the HTTP or API-envelope status code.
	StatusCode int `json:"status"`
	// Message is the human-readable error message returned by the API.
	Message string `json:"message"`
	// Data contains optional structured error details from the response envelope.
	Data any `json:"data,omitempty"`
	// Restrictions contains account-deletion blockers when the API returns them.
	Restrictions []DeletionRestriction `json:"restrictions,omitempty"`
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

// InsufficientScope reports the OAuth scope an API call was rejected for, and
// whether the rejection was in fact a missing-scope one.
//
// The API answers a call an OAuth token lacks permission for with 403 and a
// challenge naming the scope:
//
//	WWW-Authenticate: Bearer error="insufficient_scope", scope="documents:write", …
//
// Treat it as a prompt to send the user through the authorization flow again
// with that scope added, not as a request to retry. A 403 without this header
// has another cause: a different workspace, the user's own role, or an area
// OAuth tokens can never reach, such as billing or credential management.
func InsufficientScope(err error) (string, bool) {
	var apiErr *APIError
	if !stderrors.As(err, &apiErr) || apiErr == nil || apiErr.Headers == nil {
		return "", false
	}
	for _, challenge := range apiErr.Headers.Values("WWW-Authenticate") {
		params := authParams(challenge)
		if params["error"] != "insufficient_scope" {
			continue
		}
		return params["scope"], true
	}
	return "", false
}

// authParams pulls the key="value" pairs out of one WWW-Authenticate challenge.
// Unquoted values and the leading scheme token are ignored.
func authParams(challenge string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(challenge, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		if unquoted, err := strconv.Unquote(strings.TrimSpace(value)); err == nil {
			// A challenge starts with the scheme, as in `Bearer error="…"`, so the
			// first key carries it; keep only the parameter name.
			if _, name, hasScheme := strings.Cut(strings.TrimSpace(key), " "); hasScheme {
				key = name
			}
			params[strings.TrimSpace(key)] = unquoted
		}
	}
	return params
}

// IsStatusCode reports whether err is an APIError with the given status code.
func IsStatusCode(err error, code int) bool {
	var apiErr *APIError
	return stderrors.As(err, &apiErr) && apiErr != nil && apiErr.StatusCode == code
}

// IsRetryable reports whether err represents a transient failure (HTTP 429 or 5xx).
func IsRetryable(err error) bool {
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) || stderrors.Is(err, ErrUnsafeRedirect) {
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
