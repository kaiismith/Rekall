package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the canonical error type for the Rekall API.
// It carries an HTTP status, a machine-readable code, a human-readable
// message, and optional structured details (e.g. validation field errors).
type AppError struct {
	Status  int         `json:"-"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ─── Constructors ─────────────────────────────────────────────────────────────

// NotFound creates a 404 AppError.
func NotFound(resource, id string) *AppError {
	return &AppError{
		Status:  http.StatusNotFound,
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s with ID %q not found", resource, id),
	}
}

// BadRequest creates a 400 AppError.
func BadRequest(message string) *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Code:    "BAD_REQUEST",
		Message: message,
	}
}

// Unprocessable creates a 422 AppError, typically for validation failures.
// details should contain field-level validation errors.
func Unprocessable(message string, details interface{}) *AppError {
	return &AppError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Details: details,
	}
}

// Unauthorized creates a 401 AppError.
func Unauthorized(message string) *AppError {
	return &AppError{
		Status:  http.StatusUnauthorized,
		Code:    "UNAUTHORIZED",
		Message: message,
	}
}

// Forbidden creates a 403 AppError.
func Forbidden(message string) *AppError {
	return &AppError{
		Status:  http.StatusForbidden,
		Code:    "FORBIDDEN",
		Message: message,
	}
}

// Conflict creates a 409 AppError.
func Conflict(message string) *AppError {
	return &AppError{
		Status:  http.StatusConflict,
		Code:    "CONFLICT",
		Message: message,
	}
}

// Internal creates a 500 AppError.
// The raw cause is intentionally NOT exposed in the response; log it separately.
func Internal(message string) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: message,
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// AsAppError unwraps err as *AppError. Returns (nil, false) when err is not an AppError.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// IsNotFound returns true when err is a 404 AppError.
func IsNotFound(err error) bool {
	appErr, ok := AsAppError(err)
	return ok && appErr.Status == http.StatusNotFound
}

// IsForbidden returns true when err is a 403 AppError.
func IsForbidden(err error) bool {
	appErr, ok := AsAppError(err)
	return ok && appErr.Status == http.StatusForbidden
}
