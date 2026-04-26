package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Exercises the platform-admin RBAC matrix from .kiro/specs/org-scoped-navigation/
// design.md. Each test wires up the four mock repos (incl. userRepo so the
// platform-admin fallthrough actually fires) and asserts the expected
// allow/deny per row.

func newAuthzDeptService(
	deptRepo *mockDeptRepo,
	deptMemberRepo *mockDeptMemberRepo,
	memberRepo *mockMemberRepo,
	userRepo *mockUserRepo,
) *services.DepartmentService {
	return services.NewDepartmentService(deptRepo, deptMemberRepo, memberRepo, userRepo, zap.NewNop())
}

// ─── CreateDepartment ─────────────────────────────────────────────────────────

func TestRBAC_CreateDept_OwnerAllowed(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newAuthzDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo, new(mockUserRepo))

	orgID, ownerID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, ownerID).
		Return(&entities.OrgMembership{Role: "owner"}, nil)
	deptRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Department")).
		Return(&entities.Department{ID: uuid.New(), OrgID: orgID}, nil)

	_, err := svc.CreateDepartment(context.Background(), orgID, ownerID, "Engineering", "")
	require.NoError(t, err)
}

func TestRBAC_CreateDept_AdminAllowed(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newAuthzDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo, new(mockUserRepo))

	orgID, adminID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, adminID).
		Return(&entities.OrgMembership{Role: "admin"}, nil)
	deptRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Department")).
		Return(&entities.Department{ID: uuid.New(), OrgID: orgID}, nil)

	_, err := svc.CreateDepartment(context.Background(), orgID, adminID, "Engineering", "")
	require.NoError(t, err)
}

func TestRBAC_CreateDept_MemberForbidden(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo, userRepo)

	orgID, memberID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, memberID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, memberID).
		Return(&entities.User{Role: "member"}, nil)

	_, err := svc.CreateDepartment(context.Background(), orgID, memberID, "Engineering", "")
	require.Error(t, err)
	appErr, _ := apperr.AsAppError(err)
	assert.Equal(t, 403, appErr.Status)
	deptRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestRBAC_CreateDept_PlatformAdminBypassesMembership(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo, userRepo)

	orgID, adminID := uuid.New(), uuid.New()
	// Platform admin is NOT a member of this org.
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, adminID).
		Return(nil, apperr.NotFound("OrgMembership", ""))
	userRepo.On("GetByID", mock.Anything, adminID).
		Return(&entities.User{Role: "admin"}, nil)
	deptRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Department")).
		Return(&entities.Department{ID: uuid.New(), OrgID: orgID}, nil)

	_, err := svc.CreateDepartment(context.Background(), orgID, adminID, "Ops", "")
	require.NoError(t, err)
}

// ─── UpdateDepartment ─────────────────────────────────────────────────────────

func TestRBAC_UpdateDept_DeptHeadForbidden(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, userRepo)

	orgID, headID := uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, headID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, headID).
		Return(&entities.User{Role: "member"}, nil)

	_, err := svc.UpdateDepartment(context.Background(), dept.ID, headID, "Renamed", "")
	require.Error(t, err)
	appErr, _ := apperr.AsAppError(err)
	assert.Equal(t, 403, appErr.Status)
	deptRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	deptMemberRepo.AssertNotCalled(t, "GetByDeptAndUser", mock.Anything, mock.Anything, mock.Anything)
}

// ─── AddDeptMember ────────────────────────────────────────────────────────────

func TestRBAC_AddDeptMember_HeadAllowed_MemberRole(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, userRepo)

	orgID, headID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	// Caller is plain org member but dept head.
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, headID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, headID).
		Return(&entities.User{Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, headID).
		Return(&entities.DepartmentMembership{Role: "head"}, nil)
	// Target is an org member.
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, targetID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	// Target not yet in dept → Create called.
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, targetID).
		Return(nil, apperr.NotFound("DepartmentMembership", ""))
	deptMemberRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.DepartmentMembership")).
		Return(nil)

	require.NoError(t, svc.AddDeptMember(context.Background(), dept.ID, headID, targetID, "member"))
}

func TestRBAC_AddDeptMember_HeadCannotPromoteToHead(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, userRepo)

	orgID, headID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, headID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, headID).
		Return(&entities.User{Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, headID).
		Return(&entities.DepartmentMembership{Role: "head"}, nil)

	err := svc.AddDeptMember(context.Background(), dept.ID, headID, targetID, "head")
	require.Error(t, err)
	appErr, _ := apperr.AsAppError(err)
	assert.Equal(t, 403, appErr.Status)
}

func TestRBAC_AddDeptMember_PlainMemberForbidden(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, userRepo)

	orgID, callerID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{Role: "member"}, nil)
	// Caller is also a plain dept member (not head).
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, callerID).
		Return(&entities.DepartmentMembership{Role: "member"}, nil)

	err := svc.AddDeptMember(context.Background(), dept.ID, callerID, targetID, "member")
	require.Error(t, err)
	appErr, _ := apperr.AsAppError(err)
	assert.Equal(t, 403, appErr.Status)
}

func TestRBAC_AddDeptMember_PlatformAdminBypass(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, userRepo)

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	// Platform admin not a member of the org.
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, adminID).
		Return(nil, apperr.NotFound("OrgMembership", ""))
	userRepo.On("GetByID", mock.Anything, adminID).
		Return(&entities.User{Role: "admin"}, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, targetID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, targetID).
		Return(nil, apperr.NotFound("DepartmentMembership", ""))
	deptMemberRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.DepartmentMembership")).
		Return(nil)

	require.NoError(t, svc.AddDeptMember(context.Background(), dept.ID, adminID, targetID, "head"))
}

// ─── UpdateDeptMemberRole ─────────────────────────────────────────────────────

func TestRBAC_PromoteToHead_OrgAdminAllowed(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, new(mockUserRepo))

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, adminID).
		Return(&entities.OrgMembership{Role: "admin"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, targetID).
		Return(&entities.DepartmentMembership{Role: "member"}, nil)
	deptMemberRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.DepartmentMembership")).
		Return(nil)

	require.NoError(t, svc.UpdateDeptMemberRole(context.Background(), dept.ID, adminID, targetID, "head"))
}

// ─── RemoveDeptMember ─────────────────────────────────────────────────────────

func TestRBAC_RemoveDeptMember_HeadCannotRemoveAnotherHead(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, userRepo)

	orgID, head1ID, head2ID := uuid.New(), uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, head1ID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, head1ID).
		Return(&entities.User{Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, head1ID).
		Return(&entities.DepartmentMembership{Role: "head"}, nil)
	// Target is also a head.
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, head2ID).
		Return(&entities.DepartmentMembership{Role: "head"}, nil)

	err := svc.RemoveDeptMember(context.Background(), dept.ID, head1ID, head2ID)
	require.Error(t, err)
	appErr, _ := apperr.AsAppError(err)
	assert.Equal(t, 403, appErr.Status)
	deptMemberRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
}

func TestRBAC_RemoveDeptMember_SelfLeaveAllowed(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newAuthzDeptService(deptRepo, deptMemberRepo, memberRepo, userRepo)

	orgID, userID := uuid.New(), uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, userID).
		Return(&entities.User{Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, userID).
		Return(&entities.DepartmentMembership{Role: "member"}, nil)
	deptMemberRepo.On("Delete", mock.Anything, dept.ID, userID).Return(nil)

	require.NoError(t, svc.RemoveDeptMember(context.Background(), dept.ID, userID, userID))
}
