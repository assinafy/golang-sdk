package errors

import (
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
	cases := []struct {
		name string
		err  error
		code int
		want bool
	}{
		{"matches", &APIError{StatusCode: 400}, 400, true},
		{"mismatch", &APIError{StatusCode: 404}, 400, false},
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
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"429", &APIError{StatusCode: http.StatusTooManyRequests}, true},
		{"500", &APIError{StatusCode: http.StatusInternalServerError}, true},
		{"503", &APIError{StatusCode: http.StatusServiceUnavailable}, true},
		{"400", &APIError{StatusCode: http.StatusBadRequest}, false},
		{"404", &APIError{StatusCode: http.StatusNotFound}, false},
		{"network", &NetworkError{Err: errors.New("eof")}, true},
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
