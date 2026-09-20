package errors

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 400, Message: "bad request"}
	if !strings.Contains(err.Error(), "bad request") {
		t.Errorf("expected message to contain 'bad request', got %q", err.Error())
	}

	empty := &APIError{StatusCode: 500}
	if !strings.Contains(empty.Error(), "500") {
		t.Errorf("expected status code in error, got %q", empty.Error())
	}
}

func TestNetworkError(t *testing.T) {
	orig := errors.New("connection refused")
	err := &NetworkError{Err: orig}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("unexpected message: %q", err.Error())
	}
	if !errors.Is(err, orig) {
		t.Error("NetworkError should unwrap to the original error")
	}

	nilErr := &NetworkError{}
	if !strings.Contains(nilErr.Error(), "network error") {
		t.Errorf("unexpected message: %q", nilErr.Error())
	}
}

func TestIsStatusCode(t *testing.T) {
	var nilAPI *APIError
	cases := []struct {
		name string
		err  error
		code int
		want bool
	}{
		{"matches", &APIError{StatusCode: 400}, 400, true},
		{"mismatch", &APIError{StatusCode: 404}, 400, false},
		{"typed nil", error(nilAPI), 400, false},
		{"non-api", errors.New("other"), 400, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsStatusCode(tc.err, tc.code); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	var nilAPI *APIError
	var nilNetwork *NetworkError
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"429", &APIError{StatusCode: http.StatusTooManyRequests}, true},
		{"500", &APIError{StatusCode: http.StatusInternalServerError}, true},
		{"503", &APIError{StatusCode: http.StatusServiceUnavailable}, true},
		{"600", &APIError{StatusCode: 600}, false},
		{"400", &APIError{StatusCode: http.StatusBadRequest}, false},
		{"404", &APIError{StatusCode: http.StatusNotFound}, false},
		{"network", &NetworkError{Err: errors.New("eof")}, true},
		{"canceled", &NetworkError{Err: context.Canceled}, false},
		{"deadline", &NetworkError{Err: context.DeadlineExceeded}, false},
		{"unsafe redirect", &NetworkError{Err: ErrUnsafeRedirect}, false},
		{"typed nil API", error(nilAPI), false},
		{"typed nil network", error(nilNetwork), false},
		{"other", errors.New("other"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsRetryable(tc.err); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInsufficientScope(t *testing.T) {
	challenge := func(value string) http.Header {
		return http.Header{"Www-Authenticate": []string{value}}
	}

	cases := []struct {
		name  string
		err   error
		scope string
		ok    bool
	}{
		{
			name: "names the missing scope",
			err: &APIError{StatusCode: http.StatusForbidden, Headers: challenge(
				`Bearer error="insufficient_scope", scope="documents:write", ` +
					`resource_metadata="https://api.assinafy.com.br/.well-known/oauth-protected-resource"`)},
			scope: "documents:write",
			ok:    true,
		},
		{
			// A challenge without a scope still identifies the cause.
			name:  "challenge without a scope",
			err:   &APIError{StatusCode: http.StatusForbidden, Headers: challenge(`Bearer error="insufficient_scope"`)},
			scope: "",
			ok:    true,
		},
		{
			// A 403 for another reason: a different workspace, the user's role,
			// or an area OAuth tokens can never reach.
			name: "other forbidden",
			err:  &APIError{StatusCode: http.StatusForbidden},
			ok:   false,
		},
		{
			name: "unauthenticated challenge",
			err: &APIError{StatusCode: http.StatusUnauthorized, Headers: challenge(
				`Bearer resource_metadata="https://api.assinafy.com.br/.well-known/oauth-protected-resource"`)},
			ok: false,
		},
		{"malformed challenge", &APIError{Headers: challenge("Bearer error=insufficient_scope")}, "", false},
		{"not an API error", errors.New("other"), "", false},
		{"typed nil", error((*APIError)(nil)), "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scope, ok := InsufficientScope(tc.err)
			if scope != tc.scope || ok != tc.ok {
				t.Errorf("got (%q, %v), want (%q, %v)", scope, ok, tc.scope, tc.ok)
			}
		})
	}
}

func TestInsufficientScopeReadsEveryChallenge(t *testing.T) {
	err := &APIError{StatusCode: http.StatusForbidden, Headers: http.Header{"Www-Authenticate": []string{
		`Basic realm="other"`,
		`Bearer error="insufficient_scope", scope="templates:write"`,
	}}}
	if scope, ok := InsufficientScope(err); scope != "templates:write" || !ok {
		t.Fatalf("got (%q, %v)", scope, ok)
	}
}
