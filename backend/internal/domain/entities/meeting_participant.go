package entities

import (
	"time"

	"github.com/google/uuid"
)

const (
	ParticipantRoleHost        = "host"
	ParticipantRoleParticipant = "participant"
)

// MeetingParticipant tracks a user's presence in a meeting. A row is created
// when the user joins (directly or after being admitted from the waiting room)
// and left_at is set when they leave or are removed.
type MeetingParticipant struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MeetingID uuid.UUID  `gorm:"type:uuid;column:meeting_id;not null"           json:"meeting_id"`
	UserID    uuid.UUID  `gorm:"type:uuid;column:user_id;not null"              json:"user_id"`
	Role      string     `gorm:"not null;default:'participant'"                 json:"role"`
	InvitedBy *uuid.UUID `gorm:"type:uuid;column:invited_by"                    json:"invited_by,omitempty"`
	JoinedAt  *time.Time `gorm:"column:joined_at"                               json:"joined_at,omitempty"`
	LeftAt    *time.Time `gorm:"column:left_at"                                 json:"left_at,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime"                                 json:"created_at"`
}

func (MeetingParticipant) TableName() string { return "meeting_participants" }

// IsActive reports whether the participant is currently in the meeting
// (has joined and has not yet left).
func (p *MeetingParticipant) IsActive() bool {
	return p.JoinedAt != nil && p.LeftAt == nil
}

// IsHost reports whether the participant holds the host role.
func (p *MeetingParticipant) IsHost() bool { return p.Role == ParticipantRoleHost }
