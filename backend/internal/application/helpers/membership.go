package helpers

import (
	"context"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
)

// RequireOrgMembership returns the OrgMembership if userID is an active member of orgID.
func RequireOrgMembership(
	ctx context.Context,
	repo ports.OrgMembershipRepository,
	orgID, userID uuid.UUID,
) (*entities.OrgMembership, error) {
	m, err := repo.GetByOrgAndUser(ctx, orgID, userID)
	if apperr.IsNotFound(err) {
		return nil, apperr.Forbidden("you are not a member of this organization")
	}
	if err != nil {
		return nil, apperr.Internal("failed to verify organization membership")
	}
	return m, nil
}

// RequireDeptManager checks that requesterID can manage dept.
// Passes for org admin/owner OR a dept head of that specific department.
func RequireDeptManager(
	ctx context.Context,
	orgMemberRepo ports.OrgMembershipRepository,
	deptMemberRepo ports.DepartmentMembershipRepository,
	dept *entities.Department,
	requesterID uuid.UUID,
) error {
	orgMember, err := RequireOrgMembership(ctx, orgMemberRepo, dept.OrgID, requesterID)
	if err != nil {
		return err
	}
	if orgMember.IsAdmin() {
		return nil
	}
	// Check dept-level head role.
	deptMember, err := deptMemberRepo.GetByDeptAndUser(ctx, dept.ID, requesterID)
	if err == nil && deptMember.IsHead() {
		return nil
	}
	return apperr.Forbidden("only org admins/owners or the department head can perform this action")
}
