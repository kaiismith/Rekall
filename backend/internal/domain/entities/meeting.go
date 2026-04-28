package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	MeetingTypeOpen    = "open"
	MeetingTypePrivate = "private"

	MeetingStatusWaiting = "waiting"
	MeetingStatusActive  = "active"
	MeetingStatusEnded   = "ended"

	MeetingScopeOrg  = "organization"
	MeetingScopeDept = "department"

	MeetingMaxParticipants = 50
	MeetingMaxPerHost      = 5
)

// Meeting represents a real-time video/audio session. Open meetings allow any
// authenticated user to join directly; private meetings restrict direct access
// to org/dept members and route outsiders through a waiting-room knock flow.
type Meeting struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code            string     `gorm:"not null;uniqueIndex"                           json:"code"`
	Title           string     `gorm:"not null;default:''"                            json:"title"`
	Type            string     `gorm:"not null;default:'open'"                        json:"type"`
	ScopeType       *string    `gorm:"column:scope_type"                              json:"scope_type,omitempty"`
	ScopeID         *uuid.UUID `gorm:"type:uuid;column:scope_id"                      json:"scope_id,omitempty"`
	HostID          uuid.UUID  `gorm:"type:uuid;column:host_id;not null"              json:"host_id"`
	Status          string     `gorm:"not null;default:'waiting'"                     json:"status"`
	MaxParticipants int        `gorm:"column:max_participants;not null;default:50"    json:"max_participants"`
	// TranscriptionEnabled gates the live-captions / ASR feature for this
	// meeting. When false, the /meetings/:code/asr-session endpoint returns
	// 403 TRANSCRIPTION_DISABLED. Set at creation time by the host (or later
	// via PATCH /meetings/:code — out of scope for v1). Defaults to false so
	// existing rows and unset clients are unaffected.
	TranscriptionEnabled bool       `gorm:"column:transcription_enabled;not null;default:false" json:"transcription_enabled"`
	StartedAt            *time.Time `gorm:"column:started_at"                              json:"started_at,omitempty"`
	EndedAt              *time.Time `gorm:"column:ended_at"                                json:"ended_at,omitempty"`
	CreatedAt            time.Time  `gorm:"autoCreateTime"                                 json:"created_at"`
	UpdatedAt            time.Time  `gorm:"autoUpdateTime"                                 json:"updated_at"`
}

func (Meeting) TableName() string { return "meetings" }

func (m *Meeting) IsActive() bool  { return m.Status == MeetingStatusActive }
func (m *Meeting) IsEnded() bool   { return m.Status == MeetingStatusEnded }
func (m *Meeting) IsWaiting() bool { return m.Status == MeetingStatusWaiting }

// JoinURL returns the full shareable URL for this meeting.
func (m *Meeting) JoinURL(baseURL string) string {
	return fmt.Sprintf("%s/meeting/%s", baseURL, m.Code)
}

// IsPrivate reports whether the meeting restricts direct access to scope members.
func (m *Meeting) IsPrivate() bool { return m.Type == MeetingTypePrivate }
