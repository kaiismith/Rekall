package entities

import (
	"time"

	"github.com/google/uuid"
)

// Department is a named sub-group within an Organization (e.g. Engineering, Sales).
// Departments have a designated head (dept-level manager) and ordinary members.
type Department struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrgID       uuid.UUID  `gorm:"type:uuid;column:org_id;not null"              json:"org_id"`
	Name        string     `gorm:"not null"                                       json:"name"`
	Description string     `gorm:"not null;default:''"                            json:"description"`
	CreatedBy   uuid.UUID  `gorm:"type:uuid;column:created_by;not null"           json:"created_by"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"                                 json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"                                 json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index"                                          json:"deleted_at,omitempty"`
}

func (Department) TableName() string { return "departments" }

func (d *Department) IsDeleted() bool { return d.DeletedAt != nil }
