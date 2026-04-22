package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateCallRequest is the body expected for POST /api/v1/calls.
type CreateCallRequest struct {
	UserID   uuid.UUID              `json:"user_id" binding:"required"           example:"00000000-0000-0000-0000-000000000001"`
	Title    string                 `json:"title"   binding:"required,min=1,max=255" example:"Q1 Sales Call — ACME Corp"`
	Metadata map[string]interface{} `json:"metadata" swaggertype:"object"`
}

// UpdateCallRequest is the body expected for PATCH /api/v1/calls/:id.
type UpdateCallRequest struct {
	Title        *string                `json:"title"         binding:"omitempty,min=1,max=255" example:"Q1 Sales Call — ACME Corp (updated)"`
	Status       *string                `json:"status"        binding:"omitempty,oneof=pending processing done failed" example:"done" enums:"pending,processing,done,failed"`
	RecordingURL *string                `json:"recording_url" example:"https://storage.example.com/calls/rec-001.mp4"`
	Transcript   *string                `json:"transcript"    example:"Alice: Hello, this is Alice from Rekall..."`
	StartedAt    *time.Time             `json:"started_at"    example:"2026-01-15T09:00:00Z"`
	EndedAt      *time.Time             `json:"ended_at"      example:"2026-01-15T09:30:23Z"`
	Metadata     map[string]interface{} `json:"metadata"      swaggertype:"object"`
}

// CallResponse is the shape returned for a single call.
type CallResponse struct {
	ID           uuid.UUID              `json:"id"                     example:"00000000-0000-0000-0000-000000000002"`
	UserID       uuid.UUID              `json:"user_id"                example:"00000000-0000-0000-0000-000000000001"`
	Title        string                 `json:"title"                  example:"Q1 Sales Call — ACME Corp"`
	DurationSec  int                    `json:"duration_sec"           example:"1823"`
	Status       string                 `json:"status"                 example:"done" enums:"pending,processing,done,failed"`
	RecordingURL *string                `json:"recording_url,omitempty" example:"https://storage.example.com/calls/rec-001.mp4"`
	Transcript   *string                `json:"transcript,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"               swaggertype:"object"`
	StartedAt    *time.Time             `json:"started_at,omitempty"   example:"2026-01-15T09:00:00Z"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"     example:"2026-01-15T09:30:23Z"`
	CreatedAt    time.Time              `json:"created_at"             example:"2026-01-15T09:00:00Z"`
	UpdatedAt    time.Time              `json:"updated_at"             example:"2026-01-15T09:30:23Z"`
}
