package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/helpers"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

// DepartmentService orchestrates department lifecycle and membership operations.
//
// RBAC hierarchy (most permissive → least):
//
//	Org owner  — full control over departments and their members
//	Org admin  — create/update/delete departments; manage all dept members
//	Dept head  — update their own dept; add/update/remove members within their dept
//	Org member — read-only access; can be added to depts by a head/admin/owner
type DepartmentService struct {
	deptRepo       ports.DepartmentRepository
	deptMemberRepo ports.DepartmentMembershipRepository
	orgMemberRepo  ports.OrgMembershipRepository
	logger         *zap.Logger
}

// NewDepartmentService creates a DepartmentService with all required dependencies.
func NewDepartmentService(
	deptRepo ports.DepartmentRepository,
	deptMemberRepo ports.DepartmentMembershipRepository,
	orgMemberRepo ports.OrgMembershipRepository,
	log *zap.Logger,
) *DepartmentService {
	return &DepartmentService{
		deptRepo:       deptRepo,
		deptMemberRepo: deptMemberRepo,
		orgMemberRepo:  orgMemberRepo,
		logger:         applogger.WithComponent(log, "department_service"),
	}
}

// ── Departments ────────────────────────────────────────────────────────────────

// CreateDepartment creates a new department within an org. Requires org admin or owner.
func (s *DepartmentService) CreateDepartment(ctx context.Context, orgID, requesterID uuid.UUID, name, description string) (*entities.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Unprocessable("department name is required", nil)
	}
	if len(name) > 100 {
		return nil, apperr.Unprocessable("department name must be 100 characters or fewer", nil)
	}

	orgMember, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, orgID, requesterID)
	if err != nil {
		return nil, err
	}
	if !orgMember.IsAdmin() {
		return nil, apperr.Forbidden("only org admins and owners can create departments")
	}

	now := time.Now().UTC()
	dept := &entities.Department{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        name,
		Description: strings.TrimSpace(description),
		CreatedBy:   requesterID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	created, err := s.deptRepo.Create(ctx, dept)
	if err != nil {
		return nil, apperr.Internal("failed to create department")
	}

	catalog.DeptCreated.Info(s.logger,
		zap.String("dept_id", created.ID.String()),
		zap.String("org_id", orgID.String()),
		zap.String("created_by", requesterID.String()),
	)
	return created, nil
}

// GetDepartment returns a department. Requires org membership.
func (s *DepartmentService) GetDepartment(ctx context.Context, deptID, requesterID uuid.UUID) (*entities.Department, error) {
	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return nil, err
	}
	if _, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, requesterID); err != nil {
		return nil, err
	}
	return dept, nil
}

// ListDepartments returns all non-deleted departments in an org. Requires org membership.
func (s *DepartmentService) ListDepartments(ctx context.Context, orgID, requesterID uuid.UUID) ([]*entities.Department, error) {
	if _, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, orgID, requesterID); err != nil {
		return nil, err
	}
	return s.deptRepo.ListByOrg(ctx, orgID)
}

// UpdateDepartment changes a department's name or description.
// Requires org admin/owner OR the department head.
func (s *DepartmentService) UpdateDepartment(ctx context.Context, deptID, requesterID uuid.UUID, name, description string) (*entities.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Unprocessable("department name is required", nil)
	}

	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return nil, err
	}

	if err := helpers.RequireDeptManager(ctx, s.orgMemberRepo, s.deptMemberRepo, dept, requesterID); err != nil {
		return nil, err
	}

	dept.Name = name
	dept.Description = strings.TrimSpace(description)
	dept.UpdatedAt = time.Now().UTC()

	updated, err := s.deptRepo.Update(ctx, dept)
	if err != nil {
		return nil, apperr.Internal("failed to update department")
	}

	catalog.DeptUpdated.Info(s.logger,
		zap.String("dept_id", deptID.String()),
		zap.String("requester_id", requesterID.String()),
	)
	return updated, nil
}

// DeleteDepartment soft-deletes a department. Requires org admin or owner.
func (s *DepartmentService) DeleteDepartment(ctx context.Context, deptID, requesterID uuid.UUID) error {
	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	orgMember, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, requesterID)
	if err != nil {
		return err
	}
	if !orgMember.IsAdmin() {
		return apperr.Forbidden("only org admins and owners can delete departments")
	}

	if err := s.deptRepo.SoftDelete(ctx, deptID); err != nil {
		return apperr.Internal("failed to delete department")
	}

	catalog.DeptDeleted.Info(s.logger,
		zap.String("dept_id", deptID.String()),
		zap.String("requester_id", requesterID.String()),
	)
	return nil
}

// ── Membership ─────────────────────────────────────────────────────────────────

// ListDeptMembers returns all memberships for a department. Requires org membership.
func (s *DepartmentService) ListDeptMembers(ctx context.Context, deptID, requesterID uuid.UUID) ([]*entities.DepartmentMembership, error) {
	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return nil, err
	}
	if _, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, requesterID); err != nil {
		return nil, err
	}
	return s.deptMemberRepo.ListByDept(ctx, deptID)
}

// AddDeptMember adds a user to a department with the given role.
// Requires org admin/owner OR dept head. The target user must be an org member.
func (s *DepartmentService) AddDeptMember(ctx context.Context, deptID, requesterID, targetUserID uuid.UUID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "head" && role != "member" {
		role = "member"
	}

	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	if err := helpers.RequireDeptManager(ctx, s.orgMemberRepo, s.deptMemberRepo, dept, requesterID); err != nil {
		return err
	}

	// Only org admin/owner can assign the head role.
	if role == "head" {
		orgMember, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, requesterID)
		if err != nil {
			return err
		}
		if !orgMember.IsAdmin() {
			return apperr.Forbidden("only org admins and owners can assign the head role")
		}
	}

	// Target must be an org member.
	if _, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, targetUserID); err != nil {
		return apperr.Unprocessable("target user is not a member of this organization", nil)
	}

	// Idempotent: update role if already a dept member.
	existing, existingErr := s.deptMemberRepo.GetByDeptAndUser(ctx, deptID, targetUserID)
	if existingErr == nil {
		existing.Role = role
		if err := s.deptMemberRepo.Update(ctx, existing); err != nil {
			return apperr.Internal("failed to update department membership")
		}
		catalog.DeptMemberUpdated.Info(s.logger,
			zap.String("dept_id", deptID.String()),
			zap.String("user_id", targetUserID.String()),
			zap.String("new_role", role),
		)
		return nil
	}

	m := &entities.DepartmentMembership{
		ID:           uuid.New(),
		DepartmentID: deptID,
		UserID:       targetUserID,
		Role:         role,
		JoinedAt:     time.Now().UTC(),
	}
	if err := s.deptMemberRepo.Create(ctx, m); err != nil {
		return apperr.Internal("failed to add department member")
	}

	catalog.DeptMemberAdded.Info(s.logger,
		zap.String("dept_id", deptID.String()),
		zap.String("user_id", targetUserID.String()),
		zap.String("role", role),
	)
	return nil
}

// UpdateDeptMemberRole changes a dept member's role.
// Requires org admin/owner OR dept head (dept head cannot promote to head).
func (s *DepartmentService) UpdateDeptMemberRole(ctx context.Context, deptID, requesterID, targetUserID uuid.UUID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "head" && role != "member" {
		return apperr.Unprocessable("role must be 'head' or 'member'", nil)
	}

	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	if err := helpers.RequireDeptManager(ctx, s.orgMemberRepo, s.deptMemberRepo, dept, requesterID); err != nil {
		return err
	}

	// Only org admin/owner can assign head role.
	if role == "head" {
		orgMember, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, requesterID)
		if err != nil {
			return err
		}
		if !orgMember.IsAdmin() {
			return apperr.Forbidden("only org admins and owners can assign the head role")
		}
	}

	target, err := s.deptMemberRepo.GetByDeptAndUser(ctx, deptID, targetUserID)
	if apperr.IsNotFound(err) {
		return apperr.NotFound("DepartmentMembership", targetUserID.String())
	}
	if err != nil {
		return apperr.Internal("failed to retrieve department membership")
	}

	target.Role = role
	if err := s.deptMemberRepo.Update(ctx, target); err != nil {
		return apperr.Internal("failed to update department member role")
	}

	catalog.DeptMemberUpdated.Info(s.logger,
		zap.String("dept_id", deptID.String()),
		zap.String("user_id", targetUserID.String()),
		zap.String("new_role", role),
	)
	return nil
}

// RemoveDeptMember removes a user from a department.
// Org admin/owner can remove anyone; dept head can remove members (not other heads);
// users can remove themselves.
func (s *DepartmentService) RemoveDeptMember(ctx context.Context, deptID, requesterID, targetUserID uuid.UUID) error {
	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	isSelf := requesterID == targetUserID
	if !isSelf {
		if err := helpers.RequireDeptManager(ctx, s.orgMemberRepo, s.deptMemberRepo, dept, requesterID); err != nil {
			return err
		}
	} else {
		// Self-removal: just verify requester is an org member (they could be leaving).
		if _, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, requesterID); err != nil {
			return err
		}
	}

	target, err := s.deptMemberRepo.GetByDeptAndUser(ctx, deptID, targetUserID)
	if apperr.IsNotFound(err) {
		return apperr.NotFound("DepartmentMembership", targetUserID.String())
	}
	if err != nil {
		return apperr.Internal("failed to retrieve department membership")
	}

	// Dept head (not org admin) cannot remove another head.
	if target.IsHead() && !isSelf {
		orgMember, err := helpers.RequireOrgMembership(ctx, s.orgMemberRepo, dept.OrgID, requesterID)
		if err != nil {
			return err
		}
		if !orgMember.IsAdmin() {
			return apperr.Forbidden("only org admins and owners can remove a department head")
		}
	}

	if err := s.deptMemberRepo.Delete(ctx, deptID, targetUserID); err != nil {
		return apperr.Internal("failed to remove department member")
	}

	catalog.DeptMemberRemoved.Info(s.logger,
		zap.String("dept_id", deptID.String()),
		zap.String("user_id", targetUserID.String()),
	)
	return nil
}

