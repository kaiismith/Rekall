package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
)

// ─── Request DTOs ─────────────────────────────────────────────────────────────

// CreateMeetingRequest is the body for POST /api/v1/meetings.
type CreateMeetingRequest struct {
	Title                string     `json:"title"`
	Type                 string     `json:"type"       binding:"required,oneof=open private"`
	ScopeType            string     `json:"scope_type"`
	ScopeID              *uuid.UUID `json:"scope_id"`
	// TranscriptionEnabled allows the host to opt the meeting into the
	// live-captions / ASR feature at creation time. Defaults to false.
	TranscriptionEnabled bool `json:"transcription_enabled"`
}

// ─── Response DTOs ────────────────────────────────────────────────────────────

// ParticipantPreview is a lightweight user snapshot included in list responses.
type ParticipantPreview struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Initials string `json:"initials"`
}

// MeetingResponse is the outbound representation of a Meeting.
type MeetingResponse struct {
	ID                  uuid.UUID            `json:"id"`
	Code                string               `json:"code"`
	Title               string               `json:"title"`
	Type                string               `json:"type"`
	ScopeType           *string              `json:"scope_type,omitempty"`
	ScopeID             *uuid.UUID           `json:"scope_id,omitempty"`
	HostID              uuid.UUID            `json:"host_id"`
	Status              string               `json:"status"`
	MaxParticipants     int                  `json:"max_participants"`
	JoinURL             string               `json:"join_url"`
	StartedAt           *time.Time           `json:"started_at,omitempty"`
	EndedAt             *time.Time           `json:"ended_at,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	DurationSeconds     *int64               `json:"duration_seconds,omitempty"`
	ParticipantPreviews []ParticipantPreview `json:"participant_previews"`
	TranscriptionEnabled bool                `json:"transcription_enabled"`
}

// MeetingFromEntity builds a MeetingResponse from a Meeting entity.
// DurationSeconds and ParticipantPreviews are left as their zero values.
func MeetingFromEntity(m *entities.Meeting, baseURL string) MeetingResponse {
	return MeetingResponse{
		ID:                  m.ID,
		Code:                m.Code,
		Title:               m.Title,
		Type:                m.Type,
		ScopeType:           m.ScopeType,
		ScopeID:             m.ScopeID,
		HostID:              m.HostID,
		Status:              m.Status,
		MaxParticipants:     m.MaxParticipants,
		JoinURL:             m.JoinURL(baseURL),
		StartedAt:           m.StartedAt,
		EndedAt:             m.EndedAt,
		CreatedAt:           m.CreatedAt,
		ParticipantPreviews: []ParticipantPreview{},
		TranscriptionEnabled: m.TranscriptionEnabled,
	}
}

// MeetingFromListItem builds a MeetingResponse from a MeetingListItem,
// including the computed duration and participant previews.
func MeetingFromListItem(item *ports.MeetingListItem, baseURL string) MeetingResponse {
	previews := make([]ParticipantPreview, len(item.ParticipantPreviews))
	for i, p := range item.ParticipantPreviews {
		previews[i] = ParticipantPreview{
			UserID:   p.UserID.String(),
			FullName: p.FullName,
			Initials: p.Initials,
		}
	}
	resp := MeetingFromEntity(item.Meeting, baseURL)
	resp.DurationSeconds = item.DurationSeconds
	resp.ParticipantPreviews = previews
	return resp
}

// CanJoinResponse is returned by GET /api/v1/meetings/:code/can-join.
type CanJoinResponse struct {
	Result  string `json:"result"`  // "direct" | "knock" | "denied"
	KnockID string `json:"knock_id,omitempty"`
}

// MeetingResponseEnvelope wraps a single MeetingResponse in the standard envelope.
type MeetingResponseEnvelope struct {
	Success bool            `json:"success" example:"true"`
	Data    MeetingResponse `json:"data"`
}

// MeetingListResponse wraps a MeetingResponse slice.
type MeetingListResponse struct {
	Success bool              `json:"success" example:"true"`
	Data    []MeetingResponse `json:"data"`
}

// ─── Chat message DTOs ────────────────────────────────────────────────────────

// ChatMessage is the outbound representation of a MeetingMessage.
type ChatMessage struct {
	ID        uuid.UUID `json:"id"`
	MeetingID uuid.UUID `json:"meeting_id"`
	UserID    uuid.UUID `json:"user_id"`
	Body      string    `json:"body"`
	SentAt    time.Time `json:"sent_at"`
}

// ChatMessageListPayload is the data body of a chat history response.
// `HasMore` is true when additional older pages exist beyond the returned slice.
type ChatMessageListPayload struct {
	Messages []ChatMessage `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

// ChatMessageListResponse wraps a chat history page in the standard envelope.
type ChatMessageListResponse struct {
	Success bool                   `json:"success" example:"true"`
	Data    ChatMessageListPayload `json:"data"`
}

// ChatMessagesFromEntities converts a slice of MeetingMessage entities to
// their outbound DTO representation.
func ChatMessagesFromEntities(msgs []*entities.MeetingMessage) []ChatMessage {
	out := make([]ChatMessage, len(msgs))
	for i, m := range msgs {
		out[i] = ChatMessage{
			ID:        m.ID,
			MeetingID: m.MeetingID,
			UserID:    m.UserID,
			Body:      m.Body,
			SentAt:    m.SentAt,
		}
	}
	return out
}

// ─── WebSocket ticket DTOs ────────────────────────────────────────────────────

// WSTicketPayload is returned by POST /api/v1/meetings/:code/ws-ticket. The
// ticket authenticates a single WebSocket handshake and expires after 60s.
type WSTicketPayload struct {
	Ticket    string    `json:"ticket"`
	ExpiresAt time.Time `json:"expires_at"`
	WSURL     string    `json:"ws_url"`
}

// WSTicketResponse wraps WSTicketPayload in the standard envelope.
type WSTicketResponse struct {
	Success bool            `json:"success" example:"true"`
	Data    WSTicketPayload `json:"data"`
}
