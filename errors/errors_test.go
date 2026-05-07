package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Message:    "bad request",
	}

	if err.Error() != "bad request" {
		t.Errorf("expected 'bad request' but got '%s'", err.Error())
	}
}

func TestValidationError(t *testing.T) {
	t.Run("empty errors", func(t *testing.T) {
		err := &ValidationError{}
		if err.Error() != "validation error" {
			t.Errorf("expected 'validation error' but got '%s'", err.Error())
		}
	})

	t.Run("with field errors", func(t *testing.T) {
		err := &ValidationError{
			Errors: []ValidationFieldError{
				{Field: "email", Message: "required"},
				{Field: "name", Message: "too short"},
			},
		}
		expected := "email: required; name: too short; "
		if err.Error() != expected {
			t.Errorf("expected '%s' but got '%s'", expected, err.Error())
		}
	})
}

func TestNetworkError(t *testing.T) {
	origErr := errors.New("connection refused")
	err := &NetworkError{OriginalError: origErr}

	if err.Error() != "network error: connection refused" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	t.Run("nil original error", func(t *testing.T) {
		nilErr := &NetworkError{}
		if nilErr.Error() != "network error" {
			t.Errorf("unexpected error message: %s", nilErr.Error())
		}
	})
}

func TestIsStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     int
		expected bool
	}{
		{
			name:     "matches 400",
			err:      &APIError{StatusCode: 400},
			code:     400,
			expected: true,
		},
		{
			name:     "does not match 400",
			err:      &APIError{StatusCode: 404},
			code:     400,
			expected: false,
		},
		{
			name:     "non-API error",
			err:      errors.New("other"),
			code:     400,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStatusCode(tt.err, tt.code)
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "429 Too Many Requests",
			err:      &APIError{StatusCode: http.StatusTooManyRequests},
			expected: true,
		},
		{
			name:     "500 Internal Error",
			err:      &APIError{StatusCode: http.StatusInternalServerError},
			expected: true,
		},
		{
			name:     "400 Bad Request",
			err:      &APIError{StatusCode: http.StatusBadRequest},
			expected: false,
		},
		{
			name:     "404 Not Found",
			err:      &APIError{StatusCode: http.StatusNotFound},
			expected: false,
		},
		{
			name:     "non-API error",
			err:      errors.New("other"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}
