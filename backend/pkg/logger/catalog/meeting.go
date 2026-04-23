package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Meeting domain events ────────────────────────────────────────────────────
//
// Lifecycle events (created, started, ended) log at Info for audit.
// Business rejections (limit exceeded, access denied, not found) log at Warn.
// Infrastructure / hub failures log at Error.

var (
	// ── Lifecycle ─────────────────────────────────────────────────────────────

	MeetingCreated = logger.LogEvent{
		Code:    "MEETING_CREATED",
		Message: "meeting created and waiting for participants",
	}

	MeetingStarted = logger.LogEvent{
		Code:    "MEETING_STARTED",
		Message: "meeting transitioned to active state on first participant join",
	}

	MeetingEnded = logger.LogEvent{
		Code:    "MEETING_ENDED",
		Message: "meeting ended and all participants marked as left",
	}

	MeetingEndedByHost = logger.LogEvent{
		Code:    "MEETING_ENDED_BY_HOST",
		Message: "meeting ended by host request",
	}

	// ── Participants ──────────────────────────────────────────────────────────

	ParticipantJoined = logger.LogEvent{
		Code:    "MEETING_PARTICIPANT_JOINED",
		Message: "participant joined the meeting",
	}

	ParticipantLeft = logger.LogEvent{
		Code:    "MEETING_PARTICIPANT_LEFT",
		Message: "participant left the meeting",
	}

	// ── Waiting room / knock flow ─────────────────────────────────────────────

	KnockRequested = logger.LogEvent{
		Code:    "MEETING_KNOCK_REQUESTED",
		Message: "user knocked on private meeting — waiting for admission",
	}

	KnockApproved = logger.LogEvent{
		Code:    "MEETING_KNOCK_APPROVED",
		Message: "knock approved — user admitted to meeting",
	}

	KnockDenied = logger.LogEvent{
		Code:    "MEETING_KNOCK_DENIED",
		Message: "knock denied — user not admitted to meeting",
	}

	KnockTimeout = logger.LogEvent{
		Code:    "MEETING_KNOCK_TIMEOUT",
		Message: "knock auto-denied after waiting-room timeout",
	}

	// ── Business rejections ───────────────────────────────────────────────────

	MeetingHostLimitExceeded = logger.LogEvent{
		Code:    "MEETING_HOST_LIMIT_EXCEEDED",
		Message: "meeting creation rejected — host has reached the active-meeting limit",
	}

	MeetingParticipantLimitExceeded = logger.LogEvent{
		Code:    "MEETING_PARTICIPANT_LIMIT_EXCEEDED",
		Message: "join rejected — meeting has reached maximum participant capacity",
	}

	MeetingAccessDenied = logger.LogEvent{
		Code:    "MEETING_ACCESS_DENIED",
		Message: "join rejected — user is not a member of the meeting scope",
	}

	// ── Cleanup job ───────────────────────────────────────────────────────────

	CleanupJobStarted = logger.LogEvent{
		Code:    "MEETING_CLEANUP_JOB_STARTED",
		Message: "stale-meeting cleanup job tick started",
	}

	CleanupJobEnded = logger.LogEvent{
		Code:    "MEETING_CLEANUP_JOB_ENDED",
		Message: "stale-meeting cleanup job tick completed",
	}

	CleanupMeetingEnded = logger.LogEvent{
		Code:    "MEETING_CLEANUP_ENDED",
		Message: "stale meeting auto-ended by cleanup job",
	}

	CleanupJobError = logger.LogEvent{
		Code:    "MEETING_CLEANUP_ERROR",
		Message: "cleanup job encountered an error while processing stale meetings",
	}

	// ── WebSocket hub ─────────────────────────────────────────────────────────

	HubClientConnected = logger.LogEvent{
		Code:    "MEETING_HUB_CLIENT_CONNECTED",
		Message: "WebSocket client connected to meeting hub",
	}

	HubClientDisconnected = logger.LogEvent{
		Code:    "MEETING_HUB_CLIENT_DISCONNECTED",
		Message: "WebSocket client disconnected from meeting hub",
	}

	HubError = logger.LogEvent{
		Code:    "MEETING_HUB_ERROR",
		Message: "meeting hub encountered an unrecoverable error",
	}

	// ── Chat ─────────────────────────────────────────────────────────────────

	ChatPersistFailed = logger.LogEvent{
		Code:    "MEETING_CHAT_PERSIST_FAILED",
		Message: "chat message could not be persisted — not broadcast to peers",
	}

	ChatRateLimited = logger.LogEvent{
		Code:    "MEETING_CHAT_RATE_LIMITED",
		Message: "chat message dropped — sender exceeded server-side rate limit",
	}

	ChatHistoryFetchFailed = logger.LogEvent{
		Code:    "MEETING_CHAT_HISTORY_FETCH_FAILED",
		Message: "failed to load chat history for a meeting",
	}
)
