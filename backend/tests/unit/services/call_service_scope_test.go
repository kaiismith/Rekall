package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Membership mocks (scope tests only) ──────────────────────────────────────

type callMockOrgMemberRepo struct{ mock.Mock }

func (m *callMockOrgMemberRepo) Create(ctx context.Context, mem *entities.OrgMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *callMockOrgMemberRepo) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*entities.OrgMembership, error) {
	args := m.Called(ctx, orgID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.OrgMembership), args.Error(1)
}
func (m *callMockOrgMemberRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.OrgMembership, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*entities.OrgMembership), args.Error(1)
}
func (m *callMockOrgMemberRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*entities.OrgMembership, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entities.OrgMembership), args.Error(1)
}
func (m *callMockOrgMemberRepo) Update(ctx context.Context, mem *entities.OrgMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *callMockOrgMemberRepo) Delete(ctx context.Context, orgID, userID uuid.UUID) error {
	return m.Called(ctx, orgID, userID).Error(0)
}

type callMockDeptMemberRepo struct{ mock.Mock }

func (m *callMockDeptMemberRepo) Create(ctx context.Context, mem *entities.DepartmentMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *callMockDeptMemberRepo) GetByDeptAndUser(ctx context.Context, deptID, userID uuid.UUID) (*entities.DepartmentMembership, error) {
	args := m.Called(ctx, deptID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DepartmentMembership), args.Error(1)
}
func (m *callMockDeptMemberRepo) ListByDept(ctx context.Context, deptID uuid.UUID) ([]*entities.DepartmentMembership, error) {
	args := m.Called(ctx, deptID)
	return args.Get(0).([]*entities.DepartmentMembership), args.Error(1)
}
func (m *callMockDeptMemberRepo) Update(ctx context.Context, mem *entities.DepartmentMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *callMockDeptMemberRepo) Delete(ctx context.Context, deptID, userID uuid.UUID) error {
	return m.Called(ctx, deptID, userID).Error(0)
}

func newScopedCallService(callRepo *mockCallRepo, orgMem *callMockOrgMemberRepo, deptMem *callMockDeptMemberRepo) *services.CallService {
	return services.NewCallService(callRepo, orgMem, deptMem, zap.NewNop())
}

// ─── CreateCall — scope ───────────────────────────────────────────────────────

func TestCreateCall_OrgScope_NonMember_Forbidden(t *testing.T) {
	repo := new(mockCallRepo)
	orgMem := new(callMockOrgMemberRepo)
	svc := newScopedCallService(repo, orgMem, new(callMockDeptMemberRepo))

	userID := uuid.New()
	orgID := uuid.New()
	orgMem.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(nil, apperr.NotFound("OrgMembership", orgID.String()))

	_, err := svc.CreateCall(context.Background(), services.CreateCallInput{
		UserID:    userID,
		Title:     "Quarterly review",
		ScopeType: "organization",
		ScopeID:   &orgID,
	})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateCall_OrgScope_Member_Persists(t *testing.T) {
	repo := new(mockCallRepo)
	orgMem := new(callMockOrgMemberRepo)
	svc := newScopedCallService(repo, orgMem, new(callMockDeptMemberRepo))

	userID := uuid.New()
	orgID := uuid.New()
	orgMem.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(c *entities.Call) bool {
		return c.ScopeType != nil && *c.ScopeType == "organization" && c.ScopeID != nil && *c.ScopeID == orgID
	})).Return(pendingCall(userID), nil)

	_, err := svc.CreateCall(context.Background(), services.CreateCallInput{
		UserID:    userID,
		Title:     "Standup",
		ScopeType: "organization",
		ScopeID:   &orgID,
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreateCall_ScopeTypeWithoutID_BadRequest(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newScopedCallService(repo, new(callMockOrgMemberRepo), new(callMockDeptMemberRepo))

	_, err := svc.CreateCall(context.Background(), services.CreateCallInput{
		UserID:    uuid.New(),
		Title:     "Bad",
		ScopeType: "organization",
		// ScopeID intentionally nil
	})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

// ─── ListCalls — scope ────────────────────────────────────────────────────────

func TestListCalls_OrgScope_NonMember_Forbidden(t *testing.T) {
	repo := new(mockCallRepo)
	orgMem := new(callMockOrgMemberRepo)
	svc := newScopedCallService(repo, orgMem, new(callMockDeptMemberRepo))

	caller := uuid.New()
	orgID := uuid.New()
	orgMem.On("GetByOrgAndUser", mock.Anything, orgID, caller).
		Return(nil, apperr.NotFound("OrgMembership", orgID.String()))

	_, _, err := svc.ListCalls(context.Background(), caller,
		ports.ListCallsFilter{Scope: &ports.ScopeFilter{Kind: ports.ScopeKindOrganization, ID: orgID}},
		1, 20)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
	repo.AssertNotCalled(t, "List")
}

func TestListCalls_DeptScope_Member_PassesFilter(t *testing.T) {
	repo := new(mockCallRepo)
	deptMem := new(callMockDeptMemberRepo)
	svc := newScopedCallService(repo, new(callMockOrgMemberRepo), deptMem)

	caller := uuid.New()
	deptID := uuid.New()
	scope := &ports.ScopeFilter{Kind: ports.ScopeKindDepartment, ID: deptID}
	deptMem.On("GetByDeptAndUser", mock.Anything, deptID, caller).
		Return(&entities.DepartmentMembership{Role: "member"}, nil)
	repo.On("List", mock.Anything, ports.ListCallsFilter{Scope: scope}, 1, 20).
		Return([]*entities.Call{}, 0, nil)

	_, _, err := svc.ListCalls(context.Background(), caller,
		ports.ListCallsFilter{Scope: scope}, 1, 20)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListCalls_OpenScope_PinsToCallerUserID(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newScopedCallService(repo, new(callMockOrgMemberRepo), new(callMockDeptMemberRepo))

	caller := uuid.New()
	scope := &ports.ScopeFilter{Kind: ports.ScopeKindOpen}
	repo.On("List", mock.Anything, mock.MatchedBy(func(f ports.ListCallsFilter) bool {
		return f.UserID != nil && *f.UserID == caller && f.Scope == scope
	}), 1, 20).Return([]*entities.Call{}, 0, nil)

	_, _, err := svc.ListCalls(context.Background(), caller,
		ports.ListCallsFilter{Scope: scope}, 1, 20)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}
