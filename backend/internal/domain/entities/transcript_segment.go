package entities

import (
	"time"

	"github.com/google/uuid"
)

// TranscriptSegment is one row per ASR `final` TranscriptEvent. Stored
// verbatim (text + per-word timings JSONB + engine snapshot) so downstream
// insight extraction can iterate without re-running ASR.
//
// (SessionID, SegmentIndex) is UNIQUE — duplicate writes UPSERT in place
// (see infrastructure/repositories/transcript_repository.go). EngineMode and
// ModelID are denormalised from the parent TranscriptSession for fast
// per-segment provenance queries without a join.
type TranscriptSegment struct {
	ID               uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID        uuid.UUID   `gorm:"type:uuid;column:session_id;not null"           json:"session_id"`
	SegmentIndex     int32       `gorm:"column:segment_index;not null"                  json:"segment_index"`
	SpeakerUserID    uuid.UUID   `gorm:"type:uuid;column:speaker_user_id;not null"      json:"speaker_user_id"`
	CallID           *uuid.UUID  `gorm:"type:uuid;column:call_id"                       json:"call_id,omitempty"`
	MeetingID        *uuid.UUID  `gorm:"type:uuid;column:meeting_id"                    json:"meeting_id,omitempty"`
	Text             string      `gorm:"column:text;not null"                           json:"text"`
	Language         *string     `gorm:"column:language"                                json:"language,omitempty"`
	Confidence       *float32    `gorm:"column:confidence"                              json:"confidence,omitempty"`
	StartMs          int32       `gorm:"column:start_ms;not null"                       json:"start_ms"`
	EndMs            int32       `gorm:"column:end_ms;not null"                         json:"end_ms"`
	Words            WordTimings `gorm:"column:words;type:jsonb"                        json:"words,omitempty"`
	EngineMode       string      `gorm:"column:engine_mode;not null"                    json:"engine_mode"`
	ModelID          string      `gorm:"column:model_id;not null"                       json:"model_id"`
	SegmentStartedAt time.Time   `gorm:"column:segment_started_at;not null"             json:"segment_started_at"`
	CreatedAt        time.Time   `gorm:"column:created_at;not null;default:NOW()"       json:"created_at"`
}

// TableName tells GORM which table to use for this model.
func (TranscriptSegment) TableName() string { return "transcript_segments" }

// DurationMs returns the segment length in milliseconds.
func (s *TranscriptSegment) DurationMs() int32 { return s.EndMs - s.StartMs }
