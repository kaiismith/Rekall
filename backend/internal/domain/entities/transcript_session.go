package entities

import (
	"time"

	"github.com/google/uuid"
)

// TranscriptSessionStatus is the lifecycle state of a single ASR session.
type TranscriptSessionStatus string

const (
	// TranscriptSessionStatusActive is the initial state; segments may be written.
	TranscriptSessionStatusActive TranscriptSessionStatus = "active"
	// TranscriptSessionStatusEnded is the normal terminal state set by the End handler.
	TranscriptSessionStatusEnded TranscriptSessionStatus = "ended"
	// TranscriptSessionStatusErrored is set when ASR or persistence reported a fatal error.
	TranscriptSessionStatusErrored TranscriptSessionStatus = "errored"
	// TranscriptSessionStatusExpired is set by the cleanup job for sessions whose
	// expires_at passed without an explicit End call.
	TranscriptSessionStatusExpired TranscriptSessionStatus = "expired"
)

const (
	// TranscriptEngineModeLocal is the in-process whisper.cpp engine.
	TranscriptEngineModeLocal = "local"
	// TranscriptEngineModeOpenAI is the OpenAI cloud transcription engine.
	TranscriptEngineModeOpenAI = "openai"
	// TranscriptEngineModeLegacy tags rows synthesised by the one-shot backfill
	// from the legacy calls.transcript TEXT column.
	TranscriptEngineModeLegacy = "legacy"
)

// TranscriptSession is one row per ASR session lifecycle. The PK is the
// session_id issued by the C++ ASR service so backend logs, ASR logs, OpenAI
// logs, and DB rows all share a single id.
//
// Exactly one of CallID and MeetingID is set (CHECK enforced in SQL).
// (EngineMode, EngineTarget, ModelID) is a snapshot taken at session-open time
// from asrClient.Health(); a future model rollover does not retroactively
// rewrite the audit trail.
type TranscriptSession struct {
	ID                    uuid.UUID               `gorm:"type:uuid;primaryKey"                                json:"id"`
	SpeakerUserID         uuid.UUID               `gorm:"type:uuid;column:speaker_user_id;not null"           json:"speaker_user_id"`
	CallID                *uuid.UUID              `gorm:"type:uuid;column:call_id"                            json:"call_id,omitempty"`
	MeetingID             *uuid.UUID              `gorm:"type:uuid;column:meeting_id"                         json:"meeting_id,omitempty"`
	ScopeType             *string                 `gorm:"column:scope_type"                                   json:"scope_type,omitempty"`
	ScopeID               *uuid.UUID              `gorm:"type:uuid;column:scope_id"                           json:"scope_id,omitempty"`
	EngineMode            string                  `gorm:"column:engine_mode;not null"                         json:"engine_mode"`
	EngineTarget          string                  `gorm:"column:engine_target;not null"                       json:"engine_target"`
	ModelID               string                  `gorm:"column:model_id;not null"                            json:"model_id"`
	LanguageRequested     *string                 `gorm:"column:language_requested"                           json:"language_requested,omitempty"`
	SampleRate            int32                   `gorm:"column:sample_rate;not null"                         json:"sample_rate"`
	FrameFormat           string                  `gorm:"column:frame_format;not null"                        json:"frame_format"`
	CorrelationID         *string                 `gorm:"column:correlation_id"                               json:"correlation_id,omitempty"`
	Status                TranscriptSessionStatus `gorm:"column:status;not null;default:active"               json:"status"`
	StartedAt             time.Time               `gorm:"column:started_at;not null;default:NOW()"            json:"started_at"`
	EndedAt               *time.Time              `gorm:"column:ended_at"                                     json:"ended_at,omitempty"`
	ExpiresAt             time.Time               `gorm:"column:expires_at;not null"                          json:"expires_at"`
	FinalizedSegmentCount int32                   `gorm:"column:finalized_segment_count;not null;default:0"   json:"finalized_segment_count"`
	AudioSecondsTotal     float64                 `gorm:"column:audio_seconds_total;type:numeric(10,3);not null;default:0" json:"audio_seconds_total"`
	ErrorCode             *string                 `gorm:"column:error_code"                                   json:"error_code,omitempty"`
	ErrorMessage          *string                 `gorm:"column:error_message"                                json:"error_message,omitempty"`
	CreatedAt             time.Time               `gorm:"column:created_at;not null;default:NOW()"            json:"created_at"`
	UpdatedAt             time.Time               `gorm:"column:updated_at;not null;default:NOW()"            json:"updated_at"`
}

// TableName tells GORM which table to use for this model.
func (TranscriptSession) TableName() string { return "transcript_sessions" }

// IsActive reports whether the session can still accept segment writes.
func (s *TranscriptSession) IsActive() bool {
	return s.Status == TranscriptSessionStatusActive
}
