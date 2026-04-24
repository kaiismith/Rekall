package dto

import "time"

// ── Auth request DTOs ─────────────────────────────────────────────────────────

// RegisterRequest is the body for POST /api/v1/auth/register.
type RegisterRequest struct {
	Email    string `json:"email"     binding:"required"          example:"alice@example.com"`
	Password string `json:"password"  binding:"required"          example:"P@ssw0rd123"`
	FullName string `json:"full_name" binding:"required"          example:"Alice Nguyen"`
}

// LoginRequest is the body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required" example:"alice@example.com"`
	Password string `json:"password" binding:"required" example:"P@ssw0rd123"`
}

// ResendVerificationRequest is the body for POST /api/v1/auth/verify/resend.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required" example:"alice@example.com"`
}

// ForgotPasswordRequest is the body for POST /api/v1/auth/password/forgot.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required" example:"alice@example.com"`
}

// ResetPasswordRequest is the body for POST /api/v1/auth/password/reset.
type ResetPasswordRequest struct {
	Token    string `json:"token"    binding:"required" example:"a3f2c1d4e5b6..."`
	Password string `json:"password" binding:"required" example:"NewP@ssw0rd123"`
}

// UpdateMeRequest is the body for PATCH /api/v1/auth/me.
// Only full_name is editable via this endpoint; email, role, and verified
// status require different flows.
type UpdateMeRequest struct {
	FullName string `json:"full_name" binding:"required,min=1,max=100" example:"Alice Nguyen"`
}

// ChangePasswordRequest is the body for POST /api/v1/auth/password/change.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"OldP@ssw0rd123"`
	NewPassword     string `json:"new_password"     binding:"required" example:"NewP@ssw0rd123"`
}

// ChangePasswordPayload is returned by POST /api/v1/auth/password/change.
// A rotated refresh cookie is set on the response alongside this body so the
// current session stays signed in; every OTHER refresh token for the user is
// revoked by this operation.
type ChangePasswordPayload struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ── Auth response DTOs ────────────────────────────────────────────────────────

// UserResponse is the public representation of a user (no sensitive fields).
type UserResponse struct {
	ID            string    `json:"id"             example:"00000000-0000-0000-0000-000000000001"`
	Email         string    `json:"email"          example:"alice@example.com"`
	FullName      string    `json:"full_name"      example:"Alice Nguyen"`
	Role          string    `json:"role"           example:"member"  enums:"admin,member"`
	EmailVerified bool      `json:"email_verified" example:"true"`
	CreatedAt     time.Time `json:"created_at"     example:"2026-01-15T09:00:00Z"`
}

// LoginResponse is returned by POST /api/v1/auth/login and POST /api/v1/auth/refresh.
type LoginResponse struct {
	AccessToken string       `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEifQ.signature"`
	User        UserResponse `json:"user"`
}
