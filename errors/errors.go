package errors

import (
	"fmt"
	"net/http"
)

type ErrorCode int

const (
	ErrCodeBadRequest      ErrorCode = 400
	ErrCodeUnauthorized    ErrorCode = 401
	ErrCodeForbidden       ErrorCode = 403
	ErrCodeNotFound        ErrorCode = 404
	ErrCodeTooManyRequests ErrorCode = 429
	ErrCodeInternal        ErrorCode = 500
)

type APIError struct {
	StatusCode int         `json:"status"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
}

func (e *APIError) Error() string {
	return e.Message
}

type ValidationError struct {
	Errors []ValidationFieldError
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation error"
	}
	msg := ""
	for _, err := range e.Errors {
		msg += fmt.Sprintf("%s: %s; ", err.Field, err.Message)
	}
	return msg
}

type ValidationFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type NetworkError struct {
	OriginalError error
}

func (e *NetworkError) Error() string {
	if e.OriginalError != nil {
		return fmt.Sprintf("network error: %v", e.OriginalError)
	}
	return "network error"
}

func IsStatusCode(err error, code int) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == code
	}
	return false
}

func IsRetryable(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
	}
	return false
}
