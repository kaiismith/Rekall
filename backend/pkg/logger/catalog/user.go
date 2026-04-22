package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── User domain events ───────────────────────────────────────────────────────
//
// Each event covers one distinct outcome of a user-service operation.
// Success paths log at Info (audit trail); infrastructure failures log at Error;
// business-rule rejections (conflict, not-found) log at Warn.

var (
	// ── Create ───────────────────────────────────────────────────────────────

	// UserCreated is an audit event logged when a user account is persisted successfully.
	UserCreated = logger.LogEvent{
		Code:    "USER_CREATED",
		Message: "user account created successfully",
	}

	// UserCreateFailed is logged when the repository cannot persist the new user.
	UserCreateFailed = logger.LogEvent{
		Code:    "USER_CREATE_FAILED",
		Message: "repository error while persisting new user account",
	}

	// UserEmailCheckFailed is logged when the duplicate-email look-up itself errors
	// before the create attempt is made.
	UserEmailCheckFailed = logger.LogEvent{
		Code:    "USER_EMAIL_CHECK_FAILED",
		Message: "repository error while checking for existing email address",
	}

	// UserEmailConflict is logged at Warn when a creation request uses an email
	// that is already registered to an active account.
	UserEmailConflict = logger.LogEvent{
		Code:    "USER_EMAIL_CONFLICT",
		Message: "user creation rejected — email address is already registered",
	}

	// ── Read ─────────────────────────────────────────────────────────────────

	// UserFetched is logged at Debug level when a user record is retrieved successfully.
	UserFetched = logger.LogEvent{
		Code:    "USER_FETCHED",
		Message: "user record retrieved from repository",
	}

	// UserNotFound is logged at Warn when a requested user ID does not exist.
	UserNotFound = logger.LogEvent{
		Code:    "USER_NOT_FOUND",
		Message: "user record not found for the given ID",
	}

	// UserGetFailed is logged when the repository returns an unexpected error
	// while fetching a user (as opposed to a clean not-found).
	UserGetFailed = logger.LogEvent{
		Code:    "USER_GET_FAILED",
		Message: "repository error while fetching user record by ID",
	}

	// ── List ─────────────────────────────────────────────────────────────────

	// UsersListed is logged at Debug level after a successful paginated list query.
	UsersListed = logger.LogEvent{
		Code:    "USERS_LISTED",
		Message: "user records listed from repository",
	}

	// UserListFailed is logged when the repository cannot return a paginated result set.
	UserListFailed = logger.LogEvent{
		Code:    "USER_LIST_FAILED",
		Message: "repository error while listing user records",
	}

	// ── Delete ───────────────────────────────────────────────────────────────

	// UserDeleted is an audit event logged when a user account is soft-deleted successfully.
	UserDeleted = logger.LogEvent{
		Code:    "USER_DELETED",
		Message: "user account soft-deleted successfully",
	}

	// UserDeleteFailed is logged when the repository cannot soft-delete a user.
	UserDeleteFailed = logger.LogEvent{
		Code:    "USER_DELETE_FAILED",
		Message: "repository error while soft-deleting user account",
	}
)
