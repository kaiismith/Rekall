package catalog

import "github.com/rekall/backend/pkg/logger"

// ─── Intellikat publisher events ──────────────────────────────────────────────
//
// Emitted by infrastructure/messaging/IntellikatPublisher when a
// transcript_sessions row reaches a terminal state and the backend posts
// a reference message to Service Bus for the Python intellikat consumer.

var (
	// IntellikatPublishOK is logged at Info on a successful publish. Carries
	// transcript_session_id + job_id so log search can join the backend side
	// to the intellikat side by either id.
	IntellikatPublishOK = logger.LogEvent{
		Code:    "INTELLIKAT_PUBLISH_OK",
		Message: "intellikat reference message published",
	}

	// IntellikatPublishFailed is logged at Warn when the publish fails. It
	// is non-fatal: the DB close already committed, and a future operator
	// reprocess command can recover any missed messages.
	IntellikatPublishFailed = logger.LogEvent{
		Code:    "INTELLIKAT_PUBLISH_FAILED",
		Message: "intellikat reference message publish failed",
	}
)
