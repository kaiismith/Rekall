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
	ID               uuid.UUID
	RunID            uuid.UUID
	MeetingID        *uuid.UUID
	CallID           *uuid.UUID
	WindowStartedAt  time.Time
	WindowEndedAt    time.Time
	SegmentIndexLo   int32
	SegmentIndexHi   int32
	Summary          string
	KeyPoints        []string
	OpenQuestions    []string
	ModelID          string
	PromptVersion    string
	PromptTokens     *int32
	CompletionTokens *int32
	LatencyMs        int32
	// Status carries the run outcome. v1 ring-buffer pushes only OK notes;
	// Errored is recorded in logs only (no persistence layer to write to).
	Status       KatNoteStatus
	ErrorCode    *string
	ErrorMessage *string
	CreatedAt    time.Time
}

// KatNoteStatus enumerates the run outcome.
type KatNoteStatus string

const (
	KatNoteStatusOK      KatNoteStatus = "ok"
	KatNoteStatusErrored KatNoteStatus = "errored"
)
