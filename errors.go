package dotenv

import (
	"fmt"
	"net/http"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Response *http.Response
	Message  string            `json:"message"`
	Errors   map[string]string `json:"errors,omitempty"`
	Code     string            `json:"code,omitempty"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v",
		e.Response.Request.Method, e.Response.Request.URL,
		e.Response.StatusCode, e.Message)
}

// Common error types
type (
	// ErrNotFound indicates a resource was not found
	ErrNotFound struct {
		Resource string
		ID       string
	}

	// ErrUnauthorized indicates authentication failed
	ErrUnauthorized struct {
		Message string
	}

	// ErrRateLimited indicates rate limit exceeded
	ErrRateLimited struct {
		RetryAfter int
	}

	// ErrValidation indicates validation failed
	ErrValidation struct {
		Errors map[string]string
	}
)

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s with ID %s not found", e.Resource, e.ID)
}

func (e ErrUnauthorized) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "unauthorized"
}

func (e ErrRateLimited) Error() string {
	return fmt.Sprintf("rate limited, retry after %d seconds", e.RetryAfter)
}

func (e ErrValidation) Error() string {
	return fmt.Sprintf("validation failed: %v", e.Errors)
}

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	_, ok := err.(*ErrNotFound)
	return ok
}

// IsUnauthorized checks if error is unauthorized
func IsUnauthorized(err error) bool {
	_, ok := err.(*ErrUnauthorized)
	return ok
}

// IsRateLimited checks if error is rate limited
func IsRateLimited(err error) bool {
	_, ok := err.(*ErrRateLimited)
	return ok
}

// IsValidation checks if error is validation error
func IsValidation(err error) bool {
	_, ok := err.(*ErrValidation)
	return ok
}
