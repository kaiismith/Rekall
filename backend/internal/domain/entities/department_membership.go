package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/pkg/constants"
)

// DepartmentMembership links a user to a Department with a role of "head" or "member".
// A user may belong to multiple departments within the same org.
type DepartmentMembership struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DepartmentID uuid.UUID `gorm:"type:uuid;column:department_id;not null"        json:"department_id"`
	UserID       uuid.UUID `gorm:"type:uuid;column:user_id;not null"              json:"user_id"`
	Role         string    `gorm:"not null;default:member"                        json:"role"` // "head" | "member"
	JoinedAt     time.Time `gorm:"column:joined_at;autoCreateTime"                json:"joined_at"`
}

func (DepartmentMembership) TableName() string { return "department_memberships" }

// IsHead reports whether the member holds the head (leader) role.
func (m *DepartmentMembership) IsHead() bool { return m.Role == constants.DeptRoleHead }
