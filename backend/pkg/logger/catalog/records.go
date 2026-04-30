package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Records read events ──────────────────────────────────────────────────────
//
// Emitted by handlers serving the records UI (paginated transcript reads,
// meeting detail with speakers). Mostly security signals so SOC tooling can
// pick up access-denial patterns.

var (
	// RecordTranscriptAccessDenied is logged at Warn when a caller is rejected
	// from reading a meeting's or call's stored transcript. Provides the
	// `meeting_id`/`call_id`, `caller_id`, and `reason` for triage.
	RecordTranscriptAccessDenied = logger.LogEvent{
		Code:    "RECORD_TRANSCRIPT_ACCESS_DENIED",
		Message: "record transcript access denied",
	}
)
