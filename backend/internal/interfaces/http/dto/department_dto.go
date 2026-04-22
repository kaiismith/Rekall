package dto

import "time"

// ── Department request DTOs ───────────────────────────────────────────────────

// CreateDeptRequest is the body for POST /api/v1/organizations/:id/departments.
type CreateDeptRequest struct {
	Name        string `json:"name"        binding:"required" example:"Engineering"`
	Description string `json:"description"                   example:"Backend and frontend engineers"`
}

// UpdateDeptRequest is the body for PATCH /api/v1/departments/:id.
type UpdateDeptRequest struct {
	Name        string `json:"name"        binding:"required" example:"Engineering (updated)"`
	Description string `json:"description"                   example:"Backend, frontend, and platform engineers"`
}

// AddDeptMemberRequest is the body for POST /api/v1/departments/:id/members.
type AddDeptMemberRequest struct {
	UserID string `json:"user_id" binding:"required"                    example:"00000000-0000-0000-0000-000000000001"`
	Role   string `json:"role"    binding:"omitempty,oneof=head member"  example:"member" enums:"head,member"`
}

// UpdateDeptMemberRoleRequest is the body for PATCH /api/v1/departments/:id/members/:userID.
type UpdateDeptMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=head member" example:"head" enums:"head,member"`
}

// ── Department response DTOs ──────────────────────────────────────────────────

// DeptResponse is the public representation of a department.
type DeptResponse struct {
	ID          string    `json:"id"          example:"00000000-0000-0000-0000-000000000020"`
	OrgID       string    `json:"org_id"      example:"00000000-0000-0000-0000-000000000010"`
	Name        string    `json:"name"        example:"Engineering"`
	Description string    `json:"description" example:"Backend and frontend engineers"`
	CreatedBy   string    `json:"created_by"  example:"00000000-0000-0000-0000-000000000001"`
	CreatedAt   time.Time `json:"created_at"  example:"2026-01-15T09:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at"  example:"2026-01-15T09:00:00Z"`
}

// DeptMemberResponse is the public representation of a department membership.
type DeptMemberResponse struct {
	UserID       string    `json:"user_id"       example:"00000000-0000-0000-0000-000000000001"`
	DepartmentID string    `json:"department_id" example:"00000000-0000-0000-0000-000000000020"`
	Role         string    `json:"role"          example:"member" enums:"head,member"`
	JoinedAt     time.Time `json:"joined_at"     example:"2026-01-15T09:00:00Z"`
}
