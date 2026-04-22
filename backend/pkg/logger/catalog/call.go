package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Call domain events ───────────────────────────────────────────────────────
//
// Each event covers one distinct outcome of a call-service operation.
// Success paths log at Info (audit trail); infrastructure failures log at Error;
// business-rule rejections (not-found, validation) log at Warn.

var (
	// ── Create ───────────────────────────────────────────────────────────────

	// CallCreated is an audit event logged when a call record is persisted successfully.
	CallCreated = logger.LogEvent{
		Code:    "CALL_CREATED",
		Message: "call record created successfully",
	}

	// CallCreateFailed is logged when the repository cannot persist the new call.
	// Indicates a database or serialisation error, not a validation failure.
	CallCreateFailed = logger.LogEvent{
		Code:    "CALL_CREATE_FAILED",
		Message: "repository error while persisting new call record",
	}

	// ── Read ─────────────────────────────────────────────────────────────────

	// CallFetched is logged at Debug level when a call is retrieved successfully.
	// High-frequency; kept at Debug to avoid noise in production.
	CallFetched = logger.LogEvent{
		Code:    "CALL_FETCHED",
		Message: "call record retrieved from repository",
	}

	// CallNotFound is logged at Warn when a requested call ID does not exist.
	// Not an error — could be a stale client reference or an invalid ID.
	CallNotFound = logger.LogEvent{
		Code:    "CALL_NOT_FOUND",
		Message: "call record not found for the given ID",
	}

	// CallGetFailed is logged when the repository returns an unexpected error
	// while fetching a call (as opposed to a clean not-found).
	CallGetFailed = logger.LogEvent{
		Code:    "CALL_GET_FAILED",
		Message: "repository error while fetching call record by ID",
	}

	// ── List ─────────────────────────────────────────────────────────────────

	// CallsListed is logged at Debug level after a successful paginated list query.
	CallsListed = logger.LogEvent{
		Code:    "CALLS_LISTED",
		Message: "call records listed from repository",
	}

	// CallListFailed is logged when the repository cannot return a paginated result set.
	CallListFailed = logger.LogEvent{
		Code:    "CALL_LIST_FAILED",
		Message: "repository error while listing call records",
	}

	// ── Update ───────────────────────────────────────────────────────────────

	// CallUpdated is an audit event logged when call fields are saved successfully.
	CallUpdated = logger.LogEvent{
		Code:    "CALL_UPDATED",
		Message: "call record updated successfully",
	}

	// CallUpdateFailed is logged when the repository cannot save changes to an existing call.
	CallUpdateFailed = logger.LogEvent{
		Code:    "CALL_UPDATE_FAILED",
		Message: "repository error while saving call record updates",
	}

	// ── Delete ───────────────────────────────────────────────────────────────

	// CallDeleted is an audit event logged when a call is soft-deleted successfully.
	CallDeleted = logger.LogEvent{
		Code:    "CALL_DELETED",
		Message: "call record soft-deleted successfully",
	}

	// CallDeleteFailed is logged when the repository cannot soft-delete a call.
	CallDeleteFailed = logger.LogEvent{
		Code:    "CALL_DELETE_FAILED",
		Message: "repository error while soft-deleting call record",
	}

	// ── Validation ───────────────────────────────────────────────────────────

	// CallValidationFailed is logged at Warn when service-level input validation fails
	// before any repository call is made.
	CallValidationFailed = logger.LogEvent{
		Code:    "CALL_VALIDATION_FAILED",
		Message: "call request rejected due to invalid or missing input fields",
	}
)
