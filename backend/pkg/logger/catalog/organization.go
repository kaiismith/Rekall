package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Organization domain events ───────────────────────────────────────────────
//
// Mutations (create, update, delete, member changes) log at Info for audit trail.
// Access denials log at Warn. Infrastructure errors log at Error.

var (
	// ── Organization lifecycle ────────────────────────────────────────────────

	OrgCreated = logger.LogEvent{
		Code:    "ORG_CREATED",
		Message: "organization created",
	}

	OrgUpdated = logger.LogEvent{
		Code:    "ORG_UPDATED",
		Message: "organization updated",
	}

	OrgDeleted = logger.LogEvent{
		Code:    "ORG_DELETED",
		Message: "organization soft-deleted",
	}

	// ── Membership ────────────────────────────────────────────────────────────

	MemberAdded = logger.LogEvent{
		Code:    "MEMBER_ADDED",
		Message: "user added to organization",
	}

	MemberUpdated = logger.LogEvent{
		Code:    "MEMBER_UPDATED",
		Message: "organization member role updated",
	}

	MemberRemoved = logger.LogEvent{
		Code:    "MEMBER_REMOVED",
		Message: "user removed from organization",
	}

	// ── Invitations ───────────────────────────────────────────────────────────

	InvitationSent = logger.LogEvent{
		Code:    "INVITATION_SENT",
		Message: "organization invitation email dispatched",
	}

	InvitationAccepted = logger.LogEvent{
		Code:    "INVITATION_ACCEPTED",
		Message: "organization invitation accepted — membership created",
	}

	InvitationInvalid = logger.LogEvent{
		Code:    "INVITATION_INVALID",
		Message: "invitation token invalid, expired, or already accepted",
	}

	// ── Departments ───────────────────────────────────────────────────────────

	DeptCreated = logger.LogEvent{
		Code:    "DEPT_CREATED",
		Message: "department created",
	}

	DeptUpdated = logger.LogEvent{
		Code:    "DEPT_UPDATED",
		Message: "department updated",
	}

	DeptDeleted = logger.LogEvent{
		Code:    "DEPT_DELETED",
		Message: "department soft-deleted",
	}

	DeptMemberAdded = logger.LogEvent{
		Code:    "DEPT_MEMBER_ADDED",
		Message: "user added to department",
	}

	DeptMemberUpdated = logger.LogEvent{
		Code:    "DEPT_MEMBER_UPDATED",
		Message: "department member role updated",
	}

	DeptMemberRemoved = logger.LogEvent{
		Code:    "DEPT_MEMBER_REMOVED",
		Message: "user removed from department",
	}

	// ── Infrastructure failures ───────────────────────────────────────────────

	OwnerMembershipFailed = logger.LogEvent{
		Code:    "ORG_OWNER_MEMBERSHIP_FAILED",
		Message: "org created but failed to create owner membership — manual repair may be needed",
	}

	InvitationMarkAcceptedFailed = logger.LogEvent{
		Code:    "ORG_INVITATION_MARK_ACCEPTED_FAILED",
		Message: "invitation accepted but failed to mark token used — token remains valid until expiry",
	}
)
