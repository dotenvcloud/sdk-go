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

	// ErrForbidden indicates access is denied to a resource
	ErrForbidden struct {
		Resource string
		ID       string
		Action   string // e.g., "access", "modify", "delete"
	}

	// ErrRateLimited indicates rate limit exceeded
	ErrRateLimited struct {
		RetryAfter int
	}

	// ErrValidation indicates validation failed
	ErrValidation struct {
		Errors map[string]string
	}

	// ErrConflict indicates a resource conflict (e.g., duplicate)
	ErrConflict struct {
		Resource string
		Field    string
		Value    string
	}
)

func (e ErrNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s '%s' not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

func (e ErrUnauthorized) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "unauthorized"
}

func (e ErrForbidden) Error() string {
	if e.Action != "" && e.ID != "" {
		return fmt.Sprintf("forbidden: cannot %s %s '%s'", e.Action, e.Resource, e.ID)
	}
	if e.ID != "" {
		return fmt.Sprintf("access denied to %s '%s'", e.Resource, e.ID)
	}
	return fmt.Sprintf("access denied to %s", e.Resource)
}

func (e ErrRateLimited) Error() string {
	return fmt.Sprintf("rate limited, retry after %d seconds", e.RetryAfter)
}

func (e ErrValidation) Error() string {
	return fmt.Sprintf("validation failed: %v", e.Errors)
}

func (e ErrConflict) Error() string {
	if e.Field != "" && e.Value != "" {
		return fmt.Sprintf("%s already exists with %s '%s'", e.Resource, e.Field, e.Value)
	}
	return fmt.Sprintf("%s already exists", e.Resource)
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

// IsForbidden checks if error is forbidden
func IsForbidden(err error) bool {
	_, ok := err.(*ErrForbidden)
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

// IsConflict checks if error is conflict error
func IsConflict(err error) bool {
	_, ok := err.(*ErrConflict)
	return ok
}
