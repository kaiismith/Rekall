package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Mocks ────────────────────────────────────────────────────────────────────

type mockDeptRepo struct{ mock.Mock }

func (m *mockDeptRepo) Create(ctx context.Context, dept *entities.Department) (*entities.Department, error) {
	args := m.Called(ctx, dept)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Department, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.Department, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) Update(ctx context.Context, dept *entities.Department) (*entities.Department, error) {
	args := m.Called(ctx, dept)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type mockDeptMemberRepo struct{ mock.Mock }

func (m *mockDeptMemberRepo) Create(ctx context.Context, mem *entities.DepartmentMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *mockDeptMemberRepo) GetByDeptAndUser(ctx context.Context, deptID, userID uuid.UUID) (*entities.DepartmentMembership, error) {
	args := m.Called(ctx, deptID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DepartmentMembership), args.Error(1)
}
func (m *mockDeptMemberRepo) ListByDept(ctx context.Context, deptID uuid.UUID) ([]*entities.DepartmentMembership, error) {
	args := m.Called(ctx, deptID)
	return args.Get(0).([]*entities.DepartmentMembership), args.Error(1)
}
func (m *mockDeptMemberRepo) Update(ctx context.Context, mem *entities.DepartmentMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *mockDeptMemberRepo) Delete(ctx context.Context, deptID, userID uuid.UUID) error {
	return m.Called(ctx, deptID, userID).Error(0)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newDeptService(deptRepo *mockDeptRepo, deptMemberRepo *mockDeptMemberRepo, memberRepo *mockMemberRepo) *services.DepartmentService {
	// userRepo passed as nil for the legacy tests — they exercise per-org RBAC
	// only. Platform-admin fallthrough is covered by department_service_authz_test.go.
	return services.NewDepartmentService(deptRepo, deptMemberRepo, memberRepo, nil, zap.NewNop())
}

func orgMembership(orgID, userID uuid.UUID, role string) *entities.OrgMembership {
	return &entities.OrgMembership{OrgID: orgID, UserID: userID, Role: role}
}

func department(orgID uuid.UUID) *entities.Department {
	return &entities.Department{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      "Engineering",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func deptMembership(deptID, userID uuid.UUID, role string) *entities.DepartmentMembership {
	return &entities.DepartmentMembership{DepartmentID: deptID, UserID: userID, Role: role}
}

// ─── CreateDepartment ─────────────────────────────────────────────────────────

func TestCreateDepartment_AdminSuccess(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	created := &entities.Department{ID: uuid.New(), OrgID: orgID, Name: "Engineering"}

	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptRepo.On("Create", ctx, mock.AnythingOfType("*entities.Department")).Return(created, nil)

	dept, err := svc.CreateDepartment(ctx, orgID, adminID, "Engineering", "")

	require.NoError(t, err)
	assert.Equal(t, "Engineering", dept.Name)
}

func TestCreateDepartment_OwnerSuccess(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, ownerID := uuid.New(), uuid.New()
	created := &entities.Department{ID: uuid.New(), OrgID: orgID, Name: "Sales"}

	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(orgMembership(orgID, ownerID, "owner"), nil)
	deptRepo.On("Create", ctx, mock.AnythingOfType("*entities.Department")).Return(created, nil)

	dept, err := svc.CreateDepartment(ctx, orgID, ownerID, "Sales", "Sales team")

	require.NoError(t, err)
	assert.Equal(t, "Sales", dept.Name)
}

func TestCreateDepartment_MemberForbidden(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, memberID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, memberID).Return(orgMembership(orgID, memberID, "member"), nil)

	_, err := svc.CreateDepartment(ctx, orgID, memberID, "Engineering", "")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestCreateDepartment_EmptyName(t *testing.T) {
	svc := newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo))

	_, err := svc.CreateDepartment(context.Background(), uuid.New(), uuid.New(), "", "")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestCreateDepartment_NotOrgMember(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, outsiderID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, outsiderID).
		Return(nil, apperr.NotFound("OrgMembership", outsiderID.String()))

	_, err := svc.CreateDepartment(ctx, orgID, outsiderID, "Engineering", "")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

// ─── GetDepartment ────────────────────────────────────────────────────────────

func TestGetDepartment_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, requesterID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, requesterID).Return(orgMembership(orgID, requesterID, "member"), nil)

	result, err := svc.GetDepartment(ctx, dept.ID, requesterID)

	require.NoError(t, err)
	assert.Equal(t, dept.ID, result.ID)
}

func TestGetDepartment_NotOrgMember(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, outsiderID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, outsiderID).
		Return(nil, apperr.NotFound("OrgMembership", outsiderID.String()))

	_, err := svc.GetDepartment(ctx, dept.ID, outsiderID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestGetDepartment_NotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo))
	ctx := context.Background()

	deptID := uuid.New()
	deptRepo.On("GetByID", ctx, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	_, err := svc.GetDepartment(ctx, deptID, uuid.New())

	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

// ─── ListDepartments ──────────────────────────────────────────────────────────

func TestListDepartments_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, requesterID := uuid.New(), uuid.New()
	depts := []*entities.Department{department(orgID), department(orgID)}

	memberRepo.On("GetByOrgAndUser", ctx, orgID, requesterID).Return(orgMembership(orgID, requesterID, "member"), nil)
	deptRepo.On("ListByOrg", ctx, orgID).Return(depts, nil)

	result, err := svc.ListDepartments(ctx, orgID, requesterID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListDepartments_NotMember(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, outsiderID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, outsiderID).
		Return(nil, apperr.NotFound("OrgMembership", outsiderID.String()))

	_, err := svc.ListDepartments(ctx, orgID, outsiderID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

// ─── UpdateDepartment ─────────────────────────────────────────────────────────

// UpdateDepartment is now an org-head power; dept heads CANNOT rename their
// own department (RBAC spec Req 19.2). The legacy "head allowed" assertion
// is inverted to lock in the new contract.
func TestUpdateDepartment_DeptHeadForbidden(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, headID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	// Caller is a plain org member who happens to be the dept head — the new
	// rule treats that as insufficient for renaming.
	memberRepo.On("GetByOrgAndUser", ctx, orgID, headID).Return(orgMembership(orgID, headID, "member"), nil)

	_, err := svc.UpdateDepartment(ctx, dept.ID, headID, "Platform Engineering", "")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
	deptRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateDepartment_AdminAllowed(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	dept := department(orgID)
	updated := *dept
	updated.Name = "Data"

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptRepo.On("Update", ctx, mock.AnythingOfType("*entities.Department")).Return(&updated, nil)

	result, err := svc.UpdateDepartment(ctx, dept.ID, adminID, "Data", "")

	require.NoError(t, err)
	assert.Equal(t, "Data", result.Name)
}

func TestUpdateDepartment_PlainMemberForbidden(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, memberID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, memberID).Return(orgMembership(orgID, memberID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, memberID).
		Return(nil, apperr.NotFound("DepartmentMembership", memberID.String()))

	_, err := svc.UpdateDepartment(ctx, dept.ID, memberID, "New Name", "")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

// ─── DeleteDepartment ─────────────────────────────────────────────────────────

func TestDeleteDepartment_OwnerSuccess(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, ownerID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(orgMembership(orgID, ownerID, "owner"), nil)
	deptRepo.On("SoftDelete", ctx, dept.ID).Return(nil)

	require.NoError(t, svc.DeleteDepartment(ctx, dept.ID, ownerID))
}

func TestDeleteDepartment_AdminSuccess(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptRepo.On("SoftDelete", ctx, dept.ID).Return(nil)

	require.NoError(t, svc.DeleteDepartment(ctx, dept.ID, adminID))
}

func TestDeleteDepartment_DeptHeadForbidden(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, headID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, headID).Return(orgMembership(orgID, headID, "member"), nil)

	err := svc.DeleteDepartment(ctx, dept.ID, headID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestDeleteDepartment_DeptNotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo))
	ctx := context.Background()

	deptID := uuid.New()
	deptRepo.On("GetByID", ctx, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	err := svc.DeleteDepartment(ctx, deptID, uuid.New())
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestDeleteDepartment_SoftDeleteError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptRepo.On("SoftDelete", ctx, dept.ID).Return(assert.AnError)

	err := svc.DeleteDepartment(ctx, dept.ID, adminID)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

// ─── ListDeptMembers ──────────────────────────────────────────────────────────

func TestListDeptMembers_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, requesterID := uuid.New(), uuid.New()
	dept := department(orgID)
	members := []*entities.DepartmentMembership{
		deptMembership(dept.ID, uuid.New(), "head"),
		deptMembership(dept.ID, uuid.New(), "member"),
	}

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, requesterID).Return(orgMembership(orgID, requesterID, "member"), nil)
	deptMemberRepo.On("ListByDept", ctx, dept.ID).Return(members, nil)

	result, err := svc.ListDeptMembers(ctx, dept.ID, requesterID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "head", result[0].Role)
}

func TestListDeptMembers_DeptNotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo))
	ctx := context.Background()

	deptID := uuid.New()
	deptRepo.On("GetByID", ctx, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	_, err := svc.ListDeptMembers(ctx, deptID, uuid.New())
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestListDeptMembers_NotOrgMember(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, requesterID := uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, requesterID).Return(nil, apperr.NotFound("OrgMembership", ""))

	_, err := svc.ListDeptMembers(ctx, dept.ID, requesterID)
	require.Error(t, err)
}

// ─── AddDeptMember ────────────────────────────────────────────────────────────

func TestAddDeptMember_HeadCanAddMember(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, headID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, headID).Return(orgMembership(orgID, headID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, headID).Return(deptMembership(dept.ID, headID, "head"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(orgMembership(orgID, targetID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).
		Return(nil, apperr.NotFound("DepartmentMembership", targetID.String()))
	deptMemberRepo.On("Create", ctx, mock.AnythingOfType("*entities.DepartmentMembership")).Return(nil)

	require.NoError(t, svc.AddDeptMember(ctx, dept.ID, headID, targetID, "member"))
	deptMemberRepo.AssertExpectations(t)
}

func TestAddDeptMember_AdminCanAssignHeadRole(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(orgMembership(orgID, targetID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).
		Return(nil, apperr.NotFound("DepartmentMembership", targetID.String()))
	deptMemberRepo.On("Create", ctx, mock.AnythingOfType("*entities.DepartmentMembership")).Return(nil)

	require.NoError(t, svc.AddDeptMember(ctx, dept.ID, adminID, targetID, "head"))
}

func TestAddDeptMember_HeadCannotAssignHeadRole(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, headID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, headID).Return(orgMembership(orgID, headID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, headID).Return(deptMembership(dept.ID, headID, "head"), nil)

	err := svc.AddDeptMember(ctx, dept.ID, headID, targetID, "head")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestAddDeptMember_TargetNotOrgMember(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, outsiderID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, outsiderID).
		Return(nil, apperr.NotFound("OrgMembership", outsiderID.String()))

	err := svc.AddDeptMember(ctx, dept.ID, adminID, outsiderID, "member")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestAddDeptMember_AlreadyMember_UpdatesRole(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)
	existing := deptMembership(dept.ID, targetID, "member")

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(orgMembership(orgID, targetID, "member"), nil)
	// Target is already a member — Update, not Create
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).Return(existing, nil)
	deptMemberRepo.On("Update", ctx, existing).Return(nil)

	require.NoError(t, svc.AddDeptMember(ctx, dept.ID, adminID, targetID, "member"))
	deptMemberRepo.AssertNotCalled(t, "Create")
}

// ─── UpdateDeptMemberRole ─────────────────────────────────────────────────────

func TestUpdateDeptMemberRole_AdminCanPromoteToHead(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)
	targetMem := deptMembership(dept.ID, targetID, "member")

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).Return(targetMem, nil)
	deptMemberRepo.On("Update", ctx, targetMem).Return(nil)

	require.NoError(t, svc.UpdateDeptMemberRole(ctx, dept.ID, adminID, targetID, "head"))
	assert.Equal(t, "head", targetMem.Role)
}

func TestUpdateDeptMemberRole_HeadCannotPromoteToHead(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, headID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, headID).Return(orgMembership(orgID, headID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, headID).Return(deptMembership(dept.ID, headID, "head"), nil)

	err := svc.UpdateDeptMemberRole(ctx, dept.ID, headID, targetID, "head")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestCreateDepartment_NameTooLong(t *testing.T) {
	svc := newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo))

	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}
	_, err := svc.CreateDepartment(context.Background(), uuid.New(), uuid.New(), string(longName), "desc")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestCreateDepartment_CreateRepoError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptRepo.On("Create", ctx, mock.AnythingOfType("*entities.Department")).Return(nil, assert.AnError)

	_, err := svc.CreateDepartment(ctx, orgID, adminID, "Eng", "")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestUpdateDepartment_UpdateRepoError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptRepo.On("Update", ctx, mock.AnythingOfType("*entities.Department")).Return(nil, assert.AnError)

	_, err := svc.UpdateDepartment(ctx, dept.ID, adminID, "NewName", "desc")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestAddDeptMember_DeptNotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo))

	deptID := uuid.New()
	deptRepo.On("GetByID", mock.Anything, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	err := svc.AddDeptMember(context.Background(), deptID, uuid.New(), uuid.New(), "member")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestAddDeptMember_CreateError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(orgMembership(orgID, targetID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).Return(nil, apperr.NotFound("DeptMembership", ""))
	deptMemberRepo.On("Create", ctx, mock.AnythingOfType("*entities.DepartmentMembership")).Return(assert.AnError)

	err := svc.AddDeptMember(ctx, dept.ID, adminID, targetID, "member")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestAddDeptMember_UpdateError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(orgMembership(orgID, targetID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).
		Return(deptMembership(dept.ID, targetID, "member"), nil) // already a member
	deptMemberRepo.On("Update", ctx, mock.AnythingOfType("*entities.DepartmentMembership")).Return(assert.AnError)

	err := svc.AddDeptMember(ctx, dept.ID, adminID, targetID, "head")
	require.Error(t, err)
}

func TestUpdateDeptMemberRole_InvalidRole(t *testing.T) {
	svc := newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo))

	err := svc.UpdateDeptMemberRole(context.Background(), uuid.New(), uuid.New(), uuid.New(), "superadmin")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestUpdateDeptMemberRole_DeptNotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo))
	ctx := context.Background()

	deptID := uuid.New()
	deptRepo.On("GetByID", ctx, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	err := svc.UpdateDeptMemberRole(ctx, deptID, uuid.New(), uuid.New(), "member")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestUpdateDeptMemberRole_TargetNotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).Return(nil, apperr.NotFound("DepartmentMembership", ""))

	err := svc.UpdateDeptMemberRole(ctx, dept.ID, adminID, targetID, "head")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestUpdateDeptMemberRole_UpdateError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).Return(deptMembership(dept.ID, targetID, "member"), nil)
	deptMemberRepo.On("Update", ctx, mock.AnythingOfType("*entities.DepartmentMembership")).Return(assert.AnError)

	err := svc.UpdateDeptMemberRole(ctx, dept.ID, adminID, targetID, "head")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

// ─── RemoveDeptMember ─────────────────────────────────────────────────────────

func TestRemoveDeptMember_SelfRemoval(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, memberID := uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, memberID).Return(orgMembership(orgID, memberID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, memberID).Return(deptMembership(dept.ID, memberID, "member"), nil)
	deptMemberRepo.On("Delete", ctx, dept.ID, memberID).Return(nil)

	require.NoError(t, svc.RemoveDeptMember(ctx, dept.ID, memberID, memberID))
}

func TestRemoveDeptMember_AdminCanRemoveHead(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, headID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, adminID).Return(deptMembership(dept.ID, adminID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, headID).Return(deptMembership(dept.ID, headID, "head"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptMemberRepo.On("Delete", ctx, dept.ID, headID).Return(nil)

	require.NoError(t, svc.RemoveDeptMember(ctx, dept.ID, adminID, headID))
}

func TestRemoveDeptMember_HeadCannotRemoveAnotherHead(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, headID, otherHeadID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)

	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, headID).Return(orgMembership(orgID, headID, "member"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, headID).Return(deptMembership(dept.ID, headID, "head"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, otherHeadID).Return(deptMembership(dept.ID, otherHeadID, "head"), nil)

	err := svc.RemoveDeptMember(ctx, dept.ID, headID, otherHeadID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestRemoveDeptMember_DeptNotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo))
	ctx := context.Background()

	deptID := uuid.New()
	deptRepo.On("GetByID", ctx, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	err := svc.RemoveDeptMember(ctx, deptID, uuid.New(), uuid.New())
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestRemoveDeptMember_SelfRemoval_NotOrgMember(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo)
	ctx := context.Background()

	orgID, userID := uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, userID).Return(nil, apperr.NotFound("OrgMembership", ""))

	err := svc.RemoveDeptMember(ctx, dept.ID, userID, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestRemoveDeptMember_TargetNotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).Return(nil, apperr.NotFound("DepartmentMembership", ""))

	err := svc.RemoveDeptMember(ctx, dept.ID, adminID, targetID)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestRemoveDeptMember_DeleteError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	svc := newDeptService(deptRepo, deptMemberRepo, memberRepo)
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	dept := department(orgID)
	deptRepo.On("GetByID", ctx, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(orgMembership(orgID, adminID, "admin"), nil)
	deptMemberRepo.On("GetByDeptAndUser", ctx, dept.ID, targetID).Return(deptMembership(dept.ID, targetID, "member"), nil)
	deptMemberRepo.On("Delete", ctx, dept.ID, targetID).Return(assert.AnError)

	err := svc.RemoveDeptMember(ctx, dept.ID, adminID, targetID)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}
