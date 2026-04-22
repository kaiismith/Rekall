package entities

import (
	"time"

	"github.com/google/uuid"
)

// Invitation is an email-based invite to join an organization.
// The raw token is emailed to the recipient; only the SHA-256 hash is persisted.
type Invitation struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrgID      uuid.UUID  `gorm:"type:uuid;column:org_id;not null"               json:"org_id"`
	Email      string     `gorm:"not null"                                       json:"email"`
	TokenHash  string     `gorm:"column:token_hash;uniqueIndex;not null"         json:"-"`
	Role       string     `gorm:"not null;default:member"                        json:"role"`
	InvitedBy  uuid.UUID  `gorm:"type:uuid;column:invited_by;not null"           json:"invited_by"`
	ExpiresAt  time.Time  `gorm:"not null"                                       json:"expires_at"`
	AcceptedAt *time.Time `gorm:"column:accepted_at"                             json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `gorm:"autoCreateTime"                                 json:"created_at"`
}

func (Invitation) TableName() string { return "invitations" }

func (i *Invitation) IsExpired() bool  { return time.Now().UTC().After(i.ExpiresAt) }
func (i *Invitation) IsAccepted() bool { return i.AcceptedAt != nil }
func (i *Invitation) IsValid() bool    { return !i.IsExpired() && !i.IsAccepted() }
