package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the canonical error type for the Rekall API.
// It carries an HTTP status, a machine-readable code, a human-readable
// message, and optional structured details (e.g. validation field errors).
//
// RetryAfterSeconds, when non-zero, is rendered by the HTTP helper as a
// `Retry-After` response header. Use it for 503/429 responses where the
// client should back off for a known duration.
type AppError struct {
	Status            int         `json:"-"`
	Code              string      `json:"code"`
	Message           string      `json:"message"`
	Details           interface{} `json:"details,omitempty"`
	RetryAfterSeconds int         `json:"-"`
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

// Gone creates a 410 AppError with a caller-supplied machine code.
// Use this for resources that existed but have been permanently retired
// (e.g. an ended meeting).
func Gone(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusGone,
		Code:    code,
		Message: message,
	}
}

// Unauthorized with a caller-specified machine code.
// Useful where the default "UNAUTHORIZED" code is too broad (e.g.
// TICKET_REQUIRED, TICKET_INVALID, TICKET_MISMATCH).
func UnauthorizedCode(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusUnauthorized,
		Code:    code,
		Message: message,
	}
}

// NotFoundCode creates a 404 AppError with a caller-specified machine code.
func NotFoundCode(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusNotFound,
		Code:    code,
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

// ConflictCode creates a 409 AppError with a caller-supplied machine code.
func ConflictCode(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusConflict,
		Code:    code,
		Message: message,
	}
}

// ForbiddenCode creates a 403 AppError with a caller-supplied machine code.
func ForbiddenCode(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusForbidden,
		Code:    code,
		Message: message,
	}
}

// ServiceUnavailable creates a 503 AppError with a caller-supplied machine
// code. RetryAfterSeconds is propagated as a Retry-After header by the HTTP
// helper; 0 omits the header.
func ServiceUnavailable(code, message string, retryAfterSeconds int) *AppError {
	return &AppError{
		Status:            http.StatusServiceUnavailable,
		Code:              code,
		Message:           message,
		RetryAfterSeconds: retryAfterSeconds,
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
