package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Auth domain events ───────────────────────────────────────────────────────
//
// Success paths (registration, login, password reset) log at Info for audit.
// Business rejections (bad credentials, unverified email) log at Warn.
// Infrastructure failures log at Error.
// Token values MUST NEVER appear in log fields.

var (
	// ── Registration ─────────────────────────────────────────────────────────

	UserRegistered = logger.LogEvent{
		Code:    "USER_REGISTERED",
		Message: "new user account registered — verification email dispatched",
	}

	// ── Email verification ────────────────────────────────────────────────────

	EmailVerified = logger.LogEvent{
		Code:    "EMAIL_VERIFIED",
		Message: "user email address verified successfully",
	}

	EmailAlreadyVerified = logger.LogEvent{
		Code:    "EMAIL_ALREADY_VERIFIED",
		Message: "resend-verification requested but email is already verified",
	}

	VerificationTokenInvalid = logger.LogEvent{
		Code:    "VERIFICATION_TOKEN_INVALID",
		Message: "email verification failed — token invalid or expired",
	}

	// ── Login ─────────────────────────────────────────────────────────────────

	LoginSuccess = logger.LogEvent{
		Code:    "LOGIN_SUCCESS",
		Message: "user authenticated successfully",
	}

	LoginFailedBadCredentials = logger.LogEvent{
		Code:    "LOGIN_FAILED_BAD_CREDENTIALS",
		Message: "login rejected — invalid email or password",
	}

	LoginFailedUnverified = logger.LogEvent{
		Code:    "LOGIN_FAILED_UNVERIFIED",
		Message: "login rejected — email address not yet verified",
	}

	// ── Token lifecycle ───────────────────────────────────────────────────────

	TokenRefreshed = logger.LogEvent{
		Code:    "TOKEN_REFRESHED",
		Message: "access token refreshed via rotation",
	}

	TokenInvalid = logger.LogEvent{
		Code:    "TOKEN_INVALID",
		Message: "JWT validation failed",
	}

	TokenMissing = logger.LogEvent{
		Code:    "TOKEN_MISSING",
		Message: "request arrived without an Authorization header",
	}

	// ── Logout ────────────────────────────────────────────────────────────────

	Logout = logger.LogEvent{
		Code:    "LOGOUT",
		Message: "refresh token revoked — user logged out",
	}

	// ── Password reset ────────────────────────────────────────────────────────

	PasswordResetRequested = logger.LogEvent{
		Code:    "PASSWORD_RESET_REQUESTED",
		Message: "password reset email dispatched",
	}

	PasswordResetCompleted = logger.LogEvent{
		Code:    "PASSWORD_RESET_COMPLETED",
		Message: "password updated and all sessions revoked",
	}

	PasswordResetTokenInvalid = logger.LogEvent{
		Code:    "PASSWORD_RESET_TOKEN_INVALID",
		Message: "password reset failed — token invalid or expired",
	}

	// ── Infrastructure failures ───────────────────────────────────────────────

	VerificationEmailFailed = logger.LogEvent{
		Code:    "AUTH_VERIFICATION_EMAIL_FAILED",
		Message: "failed to dispatch verification email — user registered but may not receive link",
	}

	VerificationTokenMarkFailed = logger.LogEvent{
		Code:    "AUTH_VERIFICATION_TOKEN_MARK_FAILED",
		Message: "failed to mark email verification token as used — token remains reusable until expiry",
	}
)
