package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Transcript persistence events ────────────────────────────────────────────
//
// Emitted by services.TranscriptPersister and the WS / HTTP wire-up that
// records ASR `final` segments to PostgreSQL. Codes mirror the C++-side
// convention so a single Loki/Datadog query joins both surfaces by
// event_code or session_id.

var (
	// TranscriptSessionOpened is logged when a transcript_sessions row has been
	// inserted (one per ASR session_id, immediately after StartSession returns).
	TranscriptSessionOpened = logger.LogEvent{
		Code:    "TRANSCRIPT_SESSION_OPENED",
		Message: "transcript session opened",
	}

	// TranscriptSessionOpenFailed covers any failure to insert the session row.
	// The token issuance still succeeds — persistence is degraded, captions UX
	// continues working.
	TranscriptSessionOpenFailed = logger.LogEvent{
		Code:    "TRANSCRIPT_SESSION_OPEN_FAILED",
		Message: "transcript session open failed",
	}

	// TranscriptSegmentPersisted is logged at Debug per successful upsert.
	TranscriptSegmentPersisted = logger.LogEvent{
		Code:    "TRANSCRIPT_SEGMENT_PERSISTED",
		Message: "transcript segment persisted",
	}

	// TranscriptPersistFailed is logged at Warn for transient write failures
	// (DB hiccup, transaction abort). The broadcast/response is unaffected.
	TranscriptPersistFailed = logger.LogEvent{
		Code:    "TRANSCRIPT_PERSIST_FAILED",
		Message: "transcript segment persist failed",
	}

	// TranscriptSessionClosed is logged at Info when a session row transitions
	// to a terminal state (ended | errored | expired).
	TranscriptSessionClosed = logger.LogEvent{
		Code:    "TRANSCRIPT_SESSION_CLOSED",
		Message: "transcript session closed",
	}

	// TranscriptSessionNotOwned is a security signal: a caller attempted to
	// write segments under a session whose speaker_user_id is someone else.
	TranscriptSessionNotOwned = logger.LogEvent{
		Code:    "TRANSCRIPT_SESSION_NOT_OWNED",
		Message: "transcript session ownership mismatch",
	}

	// TranscriptSessionNotFound is logged when a write arrives for a
	// session_id that has no row (typically because OpenSession failed).
	TranscriptSessionNotFound = logger.LogEvent{
		Code:    "TRANSCRIPT_SESSION_NOT_FOUND",
		Message: "transcript session not found",
	}

	// TranscriptStitchFailed is logged at Error when CloseSession failed to
	// rebuild calls.transcript from the persisted segments.
	TranscriptStitchFailed = logger.LogEvent{
		Code:    "TRANSCRIPT_STITCH_FAILED",
		Message: "transcript stitch into calls.transcript failed",
	}

	// TranscriptSessionsExpired is logged once per cleanup-job batch with
	// the number of orphaned active sessions transitioned to 'expired'.
	TranscriptSessionsExpired = logger.LogEvent{
		Code:    "TRANSCRIPT_SESSIONS_EXPIRED",
		Message: "expired transcript sessions reaped",
	}
)
