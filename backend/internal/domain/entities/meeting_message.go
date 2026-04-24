package entities

import (
	"time"

	"github.com/google/uuid"
)

// MeetingMessage is a single text chat message sent during a meeting.
// Messages are persisted for late-joiner history and reconnect resync.
type MeetingMessage struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MeetingID uuid.UUID  `gorm:"type:uuid;column:meeting_id;not null"           json:"meeting_id"`
	UserID    uuid.UUID  `gorm:"type:uuid;column:user_id;not null"              json:"user_id"`
	Body      string     `gorm:"type:text;not null"                             json:"body"`
	SentAt    time.Time  `gorm:"column:sent_at;not null;default:NOW()"          json:"sent_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"                              json:"deleted_at,omitempty"`
}

func (MeetingMessage) TableName() string { return "meeting_messages" }
