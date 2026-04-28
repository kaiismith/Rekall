package entities

import (
	"time"

	"github.com/google/uuid"

	"github.com/rekall/backend/pkg/constants"
)

// OrgMembership links a user to an organization with a specific role.
type OrgMembership struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrgID    uuid.UUID `gorm:"type:uuid;column:org_id;not null"               json:"org_id"`
	UserID   uuid.UUID `gorm:"type:uuid;column:user_id;not null"               json:"user_id"`
	Role     string    `gorm:"not null;default:member"                        json:"role"` // "owner" | "admin" | "member"
	JoinedAt time.Time `gorm:"column:joined_at;autoCreateTime"                json:"joined_at"`
}

func (OrgMembership) TableName() string { return "org_memberships" }

// IsOwner reports whether the member holds the owner role.
func (m *OrgMembership) IsOwner() bool { return m.Role == constants.OrgRoleOwner }

// IsAdmin reports whether the member holds admin or owner role.
func (m *OrgMembership) IsAdmin() bool {
	return m.Role == constants.OrgRoleAdmin || m.Role == constants.OrgRoleOwner
}

// CanManageMembers reports whether the member can invite, update, or remove other members.
func (m *OrgMembership) CanManageMembers() bool { return m.IsAdmin() }

// CanManageOrg is an alias for IsAdmin scoped to org-level admin operations
// (create/rename/delete department, invite users, etc.). Kept as a separate
// predicate so future divergence between "manage-members" and "manage-org"
// stays surgical.
func (m *OrgMembership) CanManageOrg() bool { return m.IsAdmin() }
