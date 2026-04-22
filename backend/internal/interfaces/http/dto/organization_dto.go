package dto

import "time"

// ── Organization request DTOs ─────────────────────────────────────────────────

// CreateOrgRequest is the body for POST /api/v1/organizations.
type CreateOrgRequest struct {
	Name string `json:"name" binding:"required" example:"Acme Corp"`
}

// UpdateOrgRequest is the body for PATCH /api/v1/organizations/:id.
type UpdateOrgRequest struct {
	Name string `json:"name" binding:"required" example:"Acme Corp (renamed)"`
}

// InviteUserRequest is the body for POST /api/v1/organizations/:id/invitations.
type InviteUserRequest struct {
	Email string `json:"email" binding:"required"              example:"bob@example.com"`
	Role  string `json:"role"  binding:"omitempty,oneof=admin member" example:"member" enums:"admin,member"`
}

// UpdateMemberRoleRequest is the body for PATCH /api/v1/organizations/:id/members/:userID.
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member" example:"admin" enums:"admin,member"`
}

// AcceptInvitationRequest is the body for POST /api/v1/invitations/accept.
type AcceptInvitationRequest struct {
	Token string `json:"token" binding:"required" example:"a3f2c1d4e5b6c7d8e9f0a1b2c3d4e5f6..."`
}

// ── Organization response DTOs ────────────────────────────────────────────────

// OrgResponse is the public representation of an organization.
type OrgResponse struct {
	ID        string    `json:"id"         example:"00000000-0000-0000-0000-000000000010"`
	Name      string    `json:"name"       example:"Acme Corp"`
	Slug      string    `json:"slug"       example:"acme-corp"`
	OwnerID   string    `json:"owner_id"   example:"00000000-0000-0000-0000-000000000001"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T09:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T09:00:00Z"`
}

// MemberResponse is the public representation of an org membership.
type MemberResponse struct {
	UserID   string    `json:"user_id"   example:"00000000-0000-0000-0000-000000000001"`
	OrgID    string    `json:"org_id"    example:"00000000-0000-0000-0000-000000000010"`
	Role     string    `json:"role"      example:"member" enums:"owner,admin,member"`
	JoinedAt time.Time `json:"joined_at" example:"2026-01-15T09:00:00Z"`
}
