package entities

import (
	"time"

	"github.com/google/uuid"
)

// Organization is a named workspace that groups users around a shared set of calls.
// A user may belong to zero or more organizations; zero means personal workspace.
type Organization struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string     `gorm:"not null"                                       json:"name"`
	Slug      string     `gorm:"uniqueIndex;not null"                           json:"slug"`
	OwnerID   uuid.UUID  `gorm:"type:uuid;column:owner_id;not null"             json:"owner_id"`
	CreatedAt time.Time  `gorm:"autoCreateTime"                                 json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime"                                 json:"updated_at"`
	DeletedAt *time.Time `gorm:"index"                                          json:"deleted_at,omitempty"`
}

func (Organization) TableName() string { return "organizations" }

func (o *Organization) IsDeleted() bool { return o.DeletedAt != nil }
