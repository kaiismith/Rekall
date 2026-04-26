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
//	Platform admin — bypasses every org/dept membership requirement (org-level
//	                 fallthrough — e.g. can create a dept in any org without
//	                 being a member of it). Drives intervention paths for ops.
//	Org owner      — full control over departments and their members
//	Org admin      — create/update/delete departments; manage all dept members
//	Dept head      — update their own dept; add/update/remove members within their dept
//	Org member     — read-only access; can be added to depts by a head/admin/owner
type DepartmentService struct {
	deptRepo       ports.DepartmentRepository
	deptMemberRepo ports.DepartmentMembershipRepository
	orgMemberRepo  ports.OrgMembershipRepository
	userRepo       ports.UserRepository
	logger         *zap.Logger
}

// NewDepartmentService creates a DepartmentService with all required dependencies.
//
// userRepo is consulted to resolve the platform-admin role on the caller —
// pass nil only in test setups that do not exercise the platform-admin path
// (those callers will be treated as non-admin, the existing per-org rules
// still apply).
func NewDepartmentService(
	deptRepo ports.DepartmentRepository,
	deptMemberRepo ports.DepartmentMembershipRepository,
	orgMemberRepo ports.OrgMembershipRepository,
	userRepo ports.UserRepository,
	log *zap.Logger,
) *DepartmentService {
	return &DepartmentService{
		deptRepo:       deptRepo,
		deptMemberRepo: deptMemberRepo,
		orgMemberRepo:  orgMemberRepo,
		userRepo:       userRepo,
		logger:         applogger.WithComponent(log, "department_service"),
	}
}

// loadCallerCtx returns the caller's OrgMembership for the given org and —
// only when the membership alone wouldn't satisfy a manage-org check — also
// the caller's User row (for platform-admin fallthrough).
//
// Lazy loading the user keeps the call shape backward-compatible with tests
// that don't mock userRepo.GetByID for the common org-admin/owner path.
//
// Transport-layer failures (DB errors that aren't NotFound) are surfaced as
// Internal errors so the handler returns 5xx, not 403.
func (s *DepartmentService) loadCallerCtx(
	ctx context.Context,
	orgID, callerID uuid.UUID,
) (*entities.User, *entities.OrgMembership, error) {
	var orgMem *entities.OrgMembership
	m, err := s.orgMemberRepo.GetByOrgAndUser(ctx, orgID, callerID)
	if err == nil {
		orgMem = m
	} else if !apperr.IsNotFound(err) {
		return nil, nil, apperr.Internal("failed to load organization membership")
	}

	if orgMem != nil && orgMem.IsAdmin() {
		return nil, orgMem, nil
	}

	var caller *entities.User
	if s.userRepo != nil {
		u, err := s.userRepo.GetByID(ctx, callerID)
		if err == nil {
			caller = u
		} else if !apperr.IsNotFound(err) {
			return nil, nil, apperr.Internal("failed to load caller user")
		}
	}
	return caller, orgMem, nil
}

// loadCallerDeptMembership returns the caller's DepartmentMembership for the
// given department, or nil if they're not a member of it. NotFound is
// tolerated; transport-layer failures surface as Internal errors.
func (s *DepartmentService) loadCallerDeptMembership(
	ctx context.Context,
	deptID, callerID uuid.UUID,
) (*entities.DepartmentMembership, error) {
	m, err := s.deptMemberRepo.GetByDeptAndUser(ctx, deptID, callerID)
	if err == nil {
		return m, nil
	}
	if apperr.IsNotFound(err) {
		return nil, nil
	}
	return nil, apperr.Internal("failed to load department membership")
}

// ── Departments ────────────────────────────────────────────────────────────────

// CreateDepartment creates a new department within an org. Requires org
// admin/owner OR a platform admin (who may create departments in any org).
func (s *DepartmentService) CreateDepartment(ctx context.Context, orgID, requesterID uuid.UUID, name, description string) (*entities.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Unprocessable("department name is required", nil)
	}
	if len(name) > 100 {
		return nil, apperr.Unprocessable("department name must be 100 characters or fewer", nil)
	}

	caller, orgMember, err := s.loadCallerCtx(ctx, orgID, requesterID)
	if err != nil {
		return nil, err
	}
	if !helpers.CanManageOrg(caller, orgMember) {
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
// Requires org admin/owner OR a platform admin. Per the RBAC spec, dept
// heads CANNOT rename their own department — that's an org-head power.
func (s *DepartmentService) UpdateDepartment(ctx context.Context, deptID, requesterID uuid.UUID, name, description string) (*entities.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Unprocessable("department name is required", nil)
	}

	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return nil, err
	}

	caller, orgMember, err := s.loadCallerCtx(ctx, dept.OrgID, requesterID)
	if err != nil {
		return nil, err
	}
	if !helpers.CanManageDepartment(caller, orgMember) {
		return nil, apperr.Forbidden("only org admins and owners can update departments")
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

// DeleteDepartment soft-deletes a department. Requires org admin/owner OR
// a platform admin.
func (s *DepartmentService) DeleteDepartment(ctx context.Context, deptID, requesterID uuid.UUID) error {
	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	caller, orgMember, err := s.loadCallerCtx(ctx, dept.OrgID, requesterID)
	if err != nil {
		return err
	}
	if !helpers.CanManageOrg(caller, orgMember) {
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
// Requires platform admin OR org admin/owner OR dept head. The target user
// must be an org member. Promotion to "head" is gated higher — only platform
// admins or org admins/owners may assign that role (a dept head cannot
// promote anyone).
func (s *DepartmentService) AddDeptMember(ctx context.Context, deptID, requesterID, targetUserID uuid.UUID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "head" && role != "member" {
		role = "member"
	}

	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	caller, orgMember, err := s.loadCallerCtx(ctx, dept.OrgID, requesterID)
	if err != nil {
		return err
	}
	// Short-circuit when the caller already has org-management rights — saves
	// the dept-membership round-trip and matches the legacy call shape.
	if !helpers.CanManageOrg(caller, orgMember) {
		deptMem, err := s.loadCallerDeptMembership(ctx, deptID, requesterID)
		if err != nil {
			return err
		}
		if !helpers.CanManageDepartmentMembers(caller, orgMember, deptMem) {
			return apperr.Forbidden("only org admins/owners or the department head can perform this action")
		}
		// Only org admin / platform admin may assign the head role; a dept
		// head reaching this branch by definition cannot.
		if role == "head" {
			return apperr.Forbidden("only org admins and owners can assign the head role")
		}
	}

	// Target must be an org member (platform-admin caller still requires the
	// target to be in the org — adding non-members is out of scope here).
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
// Requires platform admin OR org admin/owner OR dept head; promotion to
// "head" is reserved for platform admins / org admins.
func (s *DepartmentService) UpdateDeptMemberRole(ctx context.Context, deptID, requesterID, targetUserID uuid.UUID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "head" && role != "member" {
		return apperr.Unprocessable("role must be 'head' or 'member'", nil)
	}

	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	caller, orgMember, err := s.loadCallerCtx(ctx, dept.OrgID, requesterID)
	if err != nil {
		return err
	}
	if !helpers.CanManageOrg(caller, orgMember) {
		deptMem, err := s.loadCallerDeptMembership(ctx, deptID, requesterID)
		if err != nil {
			return err
		}
		if !helpers.CanManageDepartmentMembers(caller, orgMember, deptMem) {
			return apperr.Forbidden("only org admins/owners or the department head can perform this action")
		}
		// Only platform admin / org admin / owner may promote to head; a dept
		// head reaching this branch by definition cannot.
		if role == "head" {
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
// Platform admin / org admin/owner can remove anyone; dept head can remove
// members (not other heads); users can always remove themselves.
func (s *DepartmentService) RemoveDeptMember(ctx context.Context, deptID, requesterID, targetUserID uuid.UUID) error {
	dept, err := s.deptRepo.GetByID(ctx, deptID)
	if err != nil {
		return err
	}

	caller, orgMember, err := s.loadCallerCtx(ctx, dept.OrgID, requesterID)
	if err != nil {
		return err
	}

	isSelf := requesterID == targetUserID
	if !isSelf {
		// Short-circuit when caller can manage the org outright.
		if !helpers.CanManageOrg(caller, orgMember) {
			deptMem, err := s.loadCallerDeptMembership(ctx, deptID, requesterID)
			if err != nil {
				return err
			}
			if !helpers.CanManageDepartmentMembers(caller, orgMember, deptMem) {
				return apperr.Forbidden("only org admins/owners or the department head can perform this action")
			}
		}
	} else if orgMember == nil && !helpers.IsPlatformAdmin(caller) {
		// Self-removal still requires org membership (or platform admin).
		return apperr.Forbidden("you are not a member of this organization")
	}

	target, err := s.deptMemberRepo.GetByDeptAndUser(ctx, deptID, targetUserID)
	if apperr.IsNotFound(err) {
		return apperr.NotFound("DepartmentMembership", targetUserID.String())
	}
	if err != nil {
		return apperr.Internal("failed to retrieve department membership")
	}

	// Removing another head is reserved for platform admin / org admins.
	if target.IsHead() && !isSelf && !helpers.CanManageOrg(caller, orgMember) {
		return apperr.Forbidden("only org admins and owners can remove a department head")
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

