package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/entities"
)

// TranscriptSessionDTO is the wire shape of one transcript_sessions row.
// Engine snapshot fields are included so consumers can flag transcripts
// produced by specific models (e.g. a future "re-transcribe with v3" UX).
type TranscriptSessionDTO struct {
	ID                    uuid.UUID  `json:"id"`
	SpeakerUserID         uuid.UUID  `json:"speaker_user_id"`
	CallID                *uuid.UUID `json:"call_id,omitempty"`
	MeetingID             *uuid.UUID `json:"meeting_id,omitempty"`
	EngineMode            string     `json:"engine_mode"`
	ModelID               string     `json:"model_id"`
	LanguageRequested     *string    `json:"language_requested,omitempty"`
	Status                string     `json:"status"`
	StartedAt             time.Time  `json:"started_at"`
	EndedAt               *time.Time `json:"ended_at,omitempty"`
	FinalizedSegmentCount int32      `json:"finalized_segment_count"`
	AudioSecondsTotal     float64    `json:"audio_seconds_total"`
}

// TranscriptSegmentDTO is the wire shape of one transcript_segments row.
type TranscriptSegmentDTO struct {
	ID               uuid.UUID                 `json:"id"`
	SessionID        uuid.UUID                 `json:"session_id"`
	SegmentIndex     int32                     `json:"segment_index"`
	SpeakerUserID    uuid.UUID                 `json:"speaker_user_id"`
	Text             string                    `json:"text"`
	Language         *string                   `json:"language,omitempty"`
	Confidence       *float32                  `json:"confidence,omitempty"`
	StartMs          int32                     `json:"start_ms"`
	EndMs            int32                     `json:"end_ms"`
	Words            []TranscriptWordTimingDTO `json:"words,omitempty"`
	EngineMode       string                    `json:"engine_mode"`
	ModelID          string                    `json:"model_id"`
	SegmentStartedAt time.Time                 `json:"segment_started_at"`
}

// TranscriptPagination is the page-window metadata returned alongside
// paginated transcript reads. Lets the client decide when to stop fetching.
type TranscriptPagination struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasMore    bool `json:"has_more"`
}

// CallTranscriptResponse is the body of GET /api/v1/calls/:id/transcript.
type CallTranscriptResponse struct {
	Session    *TranscriptSessionDTO  `json:"session,omitempty"` // nil when no session has been opened yet
	Segments   []TranscriptSegmentDTO `json:"segments"`          // empty array, never null
	Pagination TranscriptPagination   `json:"pagination"`
}

// MeetingTranscriptResponse is the body of GET /api/v1/meetings/:id/transcript.
// Multi-speaker meetings have one session per participant; the response groups
// them so the consumer can merge or diarize.
type MeetingTranscriptResponse struct {
	Sessions   []TranscriptSessionDTO `json:"sessions"`
	Segments   []TranscriptSegmentDTO `json:"segments"`
	Pagination TranscriptPagination   `json:"pagination"`
}

// CallTranscriptEnvelope wraps CallTranscriptResponse in the standard success
// envelope used by other endpoints in this package.
type CallTranscriptEnvelope struct {
	Success bool                   `json:"success"`
	Data    CallTranscriptResponse `json:"data"`
}

// MeetingTranscriptEnvelope wraps MeetingTranscriptResponse.
type MeetingTranscriptEnvelope struct {
	Success bool                      `json:"success"`
	Data    MeetingTranscriptResponse `json:"data"`
}

// FromTranscriptSession projects an entity to its wire DTO.
func FromTranscriptSession(s *entities.TranscriptSession) TranscriptSessionDTO {
	return TranscriptSessionDTO{
		ID:                    s.ID,
		SpeakerUserID:         s.SpeakerUserID,
		CallID:                s.CallID,
		MeetingID:             s.MeetingID,
		EngineMode:            s.EngineMode,
		ModelID:               s.ModelID,
		LanguageRequested:     s.LanguageRequested,
		Status:                string(s.Status),
		StartedAt:             s.StartedAt,
		EndedAt:               s.EndedAt,
		FinalizedSegmentCount: s.FinalizedSegmentCount,
		AudioSecondsTotal:     s.AudioSecondsTotal,
	}
}

// FromTranscriptSegment projects an entity to its wire DTO.
func FromTranscriptSegment(s *entities.TranscriptSegment) TranscriptSegmentDTO {
	words := make([]TranscriptWordTimingDTO, 0, len(s.Words))
	for _, w := range s.Words {
		words = append(words, TranscriptWordTimingDTO{
			Word:        w.Word,
			StartMs:     w.StartMs,
			EndMs:       w.EndMs,
			Probability: w.Probability,
		})
	}
	return TranscriptSegmentDTO{
		ID:               s.ID,
		SessionID:        s.SessionID,
		SegmentIndex:     s.SegmentIndex,
		SpeakerUserID:    s.SpeakerUserID,
		Text:             s.Text,
		Language:         s.Language,
		Confidence:       s.Confidence,
		StartMs:          s.StartMs,
		EndMs:            s.EndMs,
		Words:            words,
		EngineMode:       s.EngineMode,
		ModelID:          s.ModelID,
		SegmentStartedAt: s.SegmentStartedAt,
	}
}
