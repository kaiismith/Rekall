package services

import (
	"time"

	"github.com/google/uuid"
)

// KatNote is a VALUE TYPE — not an entity. It exists only in process memory
// for the duration of a meeting / call cohort entry. Holding KatNote in
// `application/services` (rather than `domain/entities`) is deliberate: the
// `entities/` package is reserved for durable record types, and Kat must
// never accidentally pick up a TableName / GORM tag.
//
// See ../../../.kiro/specs/kat-live-notes/requirements.md Requirement 1
// (Ephemerality, No Persistence) for the design contract.
type KatNote struct {
	ID               uuid.UUID  `json:"id"`
	RunID            uuid.UUID  `json:"run_id"`
	MeetingID        *uuid.UUID `json:"meeting_id,omitempty"`
	CallID           *uuid.UUID `json:"call_id,omitempty"`
	WindowStartedAt  time.Time  `json:"window_started_at"`
	WindowEndedAt    time.Time  `json:"window_ended_at"`
	SegmentIndexLo   int32      `json:"segment_index_lo"`
	SegmentIndexHi   int32      `json:"segment_index_hi"`
	Summary          string     `json:"summary"`
	KeyPoints        []string   `json:"key_points"`
	OpenQuestions    []string   `json:"open_questions"`
	ModelID          string     `json:"model_id"`
	PromptVersion    string     `json:"prompt_version"`
	PromptTokens     *int32     `json:"prompt_tokens,omitempty"`
	CompletionTokens *int32     `json:"completion_tokens,omitempty"`
	LatencyMs        int32      `json:"latency_ms"`
	// Status carries the run outcome. v1 ring-buffer pushes only OK notes;
	// Errored is recorded in logs only (no persistence layer to write to).
	Status       KatNoteStatus `json:"status"`
	ErrorCode    *string       `json:"error_code,omitempty"`
	ErrorMessage *string       `json:"error_message,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
}

// KatNoteStatus enumerates the run outcome.
type KatNoteStatus string

const (
	KatNoteStatusOK      KatNoteStatus = "ok"
	KatNoteStatusErrored KatNoteStatus = "errored"
	// KatNoteStatusEmptyWindow signals the frontend that the most recent
	// tick found no transcript segments in the window. The panel renders a
	// "Nothing to take notes" empty state instead of the warming-up
	// placeholder. NOT pushed to the ring buffer — purely transient.
	KatNoteStatusEmptyWindow KatNoteStatus = "empty_window"
	// KatNoteStatusStreaming carries an in-flight partial response from the
	// LLM. The frontend renders the partial text progressively while the
	// stream completes; on stream end a final 'ok' note replaces it. NOT
	// pushed to the ring buffer.
	KatNoteStatusStreaming KatNoteStatus = "streaming"
)
