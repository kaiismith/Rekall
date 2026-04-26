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

type mockOrgRepo struct{ mock.Mock }

func (m *mockOrgRepo) Create(ctx context.Context, org *entities.Organization) (*entities.Organization, error) {
	args := m.Called(ctx, org)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Organization), args.Error(1)
}
func (m *mockOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Organization), args.Error(1)
}
func (m *mockOrgRepo) GetBySlug(ctx context.Context, slug string) (*entities.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Organization), args.Error(1)
}
func (m *mockOrgRepo) Update(ctx context.Context, org *entities.Organization) (*entities.Organization, error) {
	args := m.Called(ctx, org)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Organization), args.Error(1)
}
func (m *mockOrgRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockOrgRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Organization, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entities.Organization), args.Error(1)
}

type mockMemberRepo struct{ mock.Mock }

func (m *mockMemberRepo) Create(ctx context.Context, mem *entities.OrgMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *mockMemberRepo) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*entities.OrgMembership, error) {
	args := m.Called(ctx, orgID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.OrgMembership), args.Error(1)
}
func (m *mockMemberRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.OrgMembership, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*entities.OrgMembership), args.Error(1)
}
func (m *mockMemberRepo) Update(ctx context.Context, mem *entities.OrgMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *mockMemberRepo) Delete(ctx context.Context, orgID, userID uuid.UUID) error {
	return m.Called(ctx, orgID, userID).Error(0)
}

type mockInviteRepo struct{ mock.Mock }

func (m *mockInviteRepo) Upsert(ctx context.Context, inv *entities.Invitation) error {
	return m.Called(ctx, inv).Error(0)
}
func (m *mockInviteRepo) GetByTokenHash(ctx context.Context, hash string) (*entities.Invitation, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Invitation), args.Error(1)
}
func (m *mockInviteRepo) GetPendingByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*entities.Invitation, error) {
	args := m.Called(ctx, orgID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Invitation), args.Error(1)
}
func (m *mockInviteRepo) MarkAccepted(ctx context.Context, hash string) error {
	return m.Called(ctx, hash).Error(0)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newOrgService(
	orgRepo *mockOrgRepo,
	memberRepo *mockMemberRepo,
	inviteRepo *mockInviteRepo,
	userRepo *mockUserRepo,
	mailer *mockMailer,
) *services.OrganizationService {
	return services.NewOrganizationService(
		orgRepo, memberRepo, inviteRepo, userRepo, mailer,
		"http://localhost:5173", 7*24*time.Hour, zap.NewNop(),
	)
}

func membership(orgID, userID uuid.UUID, role string) *entities.OrgMembership {
	return &entities.OrgMembership{OrgID: orgID, UserID: userID, Role: role}
}

func org(ownerID uuid.UUID) *entities.Organization {
	return &entities.Organization{
		ID: uuid.New(), Name: "Acme Corp", Slug: "acme-corp", OwnerID: ownerID,
	}
}

// ─── CreateOrganization ───────────────────────────────────────────────────────

func TestCreateOrganization_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	ownerID := uuid.New()
	created := &entities.Organization{ID: uuid.New(), Name: "Acme Corp", Slug: "acme-corp", OwnerID: ownerID}

	orgRepo.On("GetBySlug", ctx, "acme-corp").Return(nil, apperr.NotFound("Organization", "acme-corp"))
	orgRepo.On("Create", ctx, mock.AnythingOfType("*entities.Organization")).Return(created, nil)
	memberRepo.On("Create", ctx, mock.AnythingOfType("*entities.OrgMembership")).Return(nil)

	o, err := svc.CreateOrganization(ctx, ownerID, "Acme Corp", "")

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", o.Name)
	memberRepo.AssertExpectations(t)
}

func TestCreateOrganization_EmptyName(t *testing.T) {
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	_, err := svc.CreateOrganization(context.Background(), uuid.New(), "", "")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestCreateOrganization_NameTooLong(t *testing.T) {
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}

	_, err := svc.CreateOrganization(context.Background(), uuid.New(), string(longName), "")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestCreateOrganization_RepoCreateError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	svc := newOrgService(orgRepo, new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgRepo.On("GetBySlug", mock.Anything, mock.Anything).Return(nil, apperr.NotFound("Organization", "x"))
	orgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Organization")).Return(nil, assert.AnError)

	_, err := svc.CreateOrganization(context.Background(), uuid.New(), "Acme", "")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestCreateOrganization_MembershipFailure_StillSucceeds(t *testing.T) {
	// If membership creation fails after org is created, the org is still
	// returned — the error is logged for the caller to retry separately.
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgRepo.On("GetBySlug", mock.Anything, mock.Anything).Return(nil, apperr.NotFound("Organization", "x"))
	orgID := uuid.New()
	orgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Organization")).Return(&entities.Organization{
		ID: orgID, Name: "Acme", Slug: "acme",
	}, nil)
	memberRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.OrgMembership")).Return(assert.AnError)

	org, err := svc.CreateOrganization(context.Background(), uuid.New(), "Acme", "")
	require.NoError(t, err)
	assert.Equal(t, orgID, org.ID)
}

func TestCreateOrganization_SlugCollision(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	ownerID := uuid.New()
	existing := &entities.Organization{ID: uuid.New(), Slug: "acme-corp"}
	unique := &entities.Organization{ID: uuid.New(), Slug: "acme-corp-2"}

	orgRepo.On("GetBySlug", ctx, "acme-corp").Return(existing, nil)
	orgRepo.On("GetBySlug", ctx, "acme-corp-2").Return(nil, apperr.NotFound("Organization", "acme-corp-2"))
	orgRepo.On("Create", ctx, mock.AnythingOfType("*entities.Organization")).Return(unique, nil)
	memberRepo.On("Create", ctx, mock.AnythingOfType("*entities.OrgMembership")).Return(nil)

	o, err := svc.CreateOrganization(ctx, ownerID, "Acme Corp", "")

	require.NoError(t, err)
	assert.Equal(t, "acme-corp-2", o.Slug)
}

// ─── GetOrganization ──────────────────────────────────────────────────────────

func TestGetOrganization_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	ownerID := uuid.New()
	o := org(ownerID)

	memberRepo.On("GetByOrgAndUser", ctx, o.ID, ownerID).Return(membership(o.ID, ownerID, "owner"), nil)
	orgRepo.On("GetByID", ctx, o.ID).Return(o, nil)

	result, err := svc.GetOrganization(ctx, o.ID, ownerID)

	require.NoError(t, err)
	assert.Equal(t, o.ID, result.ID)
}

func TestGetOrganization_NotMember(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))
	ctx := context.Background()

	orgID, outsiderID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, outsiderID).
		Return(nil, apperr.NotFound("OrgMembership", outsiderID.String()))
	// Outsider is not a platform admin — userRepo.GetByID returns a plain
	// member User, so the platform-admin fallthrough does NOT engage and
	// the call is forbidden.
	userRepo.On("GetByID", ctx, outsiderID).
		Return(&entities.User{ID: outsiderID, Role: "member"}, nil)

	_, err := svc.GetOrganization(ctx, orgID, outsiderID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

// ─── UpdateOrganization ───────────────────────────────────────────────────────

func TestUpdateOrganization_AdminSuccess(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	adminID := uuid.New()
	o := org(uuid.New())
	updated := *o
	updated.Name = "Renamed Corp"

	memberRepo.On("GetByOrgAndUser", ctx, o.ID, adminID).Return(membership(o.ID, adminID, "admin"), nil)
	orgRepo.On("GetByID", ctx, o.ID).Return(o, nil)
	orgRepo.On("Update", ctx, mock.AnythingOfType("*entities.Organization")).Return(&updated, nil)

	result, err := svc.UpdateOrganization(ctx, o.ID, adminID, "Renamed Corp")

	require.NoError(t, err)
	assert.Equal(t, "Renamed Corp", result.Name)
}

func TestUpdateOrganization_MemberForbidden(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))
	ctx := context.Background()

	orgID, memberID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, memberID).Return(membership(orgID, memberID, "member"), nil)
	// Plain member at the platform level — no fallthrough.
	userRepo.On("GetByID", ctx, memberID).
		Return(&entities.User{ID: memberID, Role: "member"}, nil)

	_, err := svc.UpdateOrganization(ctx, orgID, memberID, "New Name")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

// ─── DeleteOrganization ───────────────────────────────────────────────────────

func TestDeleteOrganization_OnlyOwnerCanDelete(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(membership(orgID, adminID, "admin"), nil)

	err := svc.DeleteOrganization(ctx, orgID, adminID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestDeleteOrganization_OwnerSuccess(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, ownerID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	orgRepo.On("SoftDelete", ctx, orgID).Return(nil)

	err := svc.DeleteOrganization(ctx, orgID, ownerID)

	require.NoError(t, err)
}

// ─── ListOrganizations ────────────────────────────────────────────────────────

func TestListOrganizations_ReturnsUserOrgs(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	svc := newOrgService(orgRepo, new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	orgs := []*entities.Organization{
		{ID: uuid.New(), Name: "Org A"},
		{ID: uuid.New(), Name: "Org B"},
	}
	orgRepo.On("ListByUserID", ctx, userID).Return(orgs, nil)

	result, err := svc.ListOrganizations(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

// ─── ListMembers ──────────────────────────────────────────────────────────────

func TestListMembers_Success(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, requesterID := uuid.New(), uuid.New()
	members := []*entities.OrgMembership{
		{OrgID: orgID, UserID: uuid.New(), Role: "owner"},
		{OrgID: orgID, UserID: requesterID, Role: "member"},
	}
	memberRepo.On("GetByOrgAndUser", ctx, orgID, requesterID).Return(membership(orgID, requesterID, "member"), nil)
	memberRepo.On("ListByOrg", ctx, orgID).Return(members, nil)

	result, err := svc.ListMembers(ctx, orgID, requesterID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListMembers_NotMember(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, outsiderID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, outsiderID).
		Return(nil, apperr.NotFound("OrgMembership", outsiderID.String()))

	_, err := svc.ListMembers(ctx, orgID, outsiderID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

// ─── UpdateMemberRole ─────────────────────────────────────────────────────────

func TestUpdateMemberRole_AdminCannotPromoteToAdmin(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(membership(orgID, adminID, "admin"), nil)

	err := svc.UpdateMemberRole(ctx, orgID, adminID, targetID, "admin")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestUpdateMemberRole_OwnerCanPromoteToAdmin(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, ownerID, targetID := uuid.New(), uuid.New(), uuid.New()
	targetMem := membership(orgID, targetID, "member")

	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(targetMem, nil)
	memberRepo.On("Update", ctx, targetMem).Return(nil)

	err := svc.UpdateMemberRole(ctx, orgID, ownerID, targetID, "admin")

	require.NoError(t, err)
	assert.Equal(t, "admin", targetMem.Role)
}

func TestUpdateMemberRole_CannotChangeOwnerRole(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, ownerID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(membership(orgID, targetID, "owner"), nil)

	err := svc.UpdateMemberRole(ctx, orgID, ownerID, targetID, "member")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestUpdateMemberRole_InvalidRole(t *testing.T) {
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	err := svc.UpdateMemberRole(context.Background(), uuid.New(), uuid.New(), uuid.New(), "bogus")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestUpdateMemberRole_NotManager_Forbidden(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))
	ctx := context.Background()

	orgID, callerID := uuid.New(), uuid.New()
	// Caller is a plain "member" — cannot manage members.
	memberRepo.On("GetByOrgAndUser", ctx, orgID, callerID).Return(membership(orgID, callerID, "member"), nil)
	userRepo.On("GetByID", ctx, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	err := svc.UpdateMemberRole(ctx, orgID, callerID, uuid.New(), "member")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestUpdateMemberRole_TargetNotFound(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, ownerID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(nil, apperr.NotFound("OrgMembership", ""))

	err := svc.UpdateMemberRole(ctx, orgID, ownerID, targetID, "admin")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestRemoveMember_SelfRemoval_AsPlainMember(t *testing.T) {
	// Plain members can still remove themselves (isSelf branch bypasses CanManageOrg).
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))

	orgID, userID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).Return(membership(orgID, userID, "member"), nil)
	// Plain member at the platform level — lazy user-fetch fires.
	userRepo.On("GetByID", mock.Anything, userID).
		Return(&entities.User{ID: userID, Role: "member"}, nil)
	memberRepo.On("Delete", mock.Anything, orgID, userID).Return(nil)

	require.NoError(t, svc.RemoveMember(context.Background(), orgID, userID, userID))
}

func TestRemoveMember_NotManager_Forbidden(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))

	orgID, callerID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(membership(orgID, callerID, "member"), nil)
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	err := svc.RemoveMember(context.Background(), orgID, callerID, targetID)
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestRemoveMember_TargetNotFound(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgID, ownerID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, targetID).Return(nil, apperr.NotFound("OrgMembership", ""))

	err := svc.RemoveMember(context.Background(), orgID, ownerID, targetID)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestUpdateMemberRole_TargetRepoError(t *testing.T) {
	// Non-NotFound error from target GetByOrgAndUser → 500.
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, ownerID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(nil, assert.AnError)

	err := svc.UpdateMemberRole(ctx, orgID, ownerID, targetID, "admin")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestUpdateMemberRole_AdminCannotGrantAdmin(t *testing.T) {
	// Admins cannot promote others to admin — only the owner can.
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(membership(orgID, adminID, "admin"), nil)

	err := svc.UpdateMemberRole(ctx, orgID, adminID, targetID, "admin")
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestRemoveMember_DeleteError(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgID, ownerID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, targetID).Return(membership(orgID, targetID, "member"), nil)
	memberRepo.On("Delete", mock.Anything, orgID, targetID).Return(assert.AnError)

	err := svc.RemoveMember(context.Background(), orgID, ownerID, targetID)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestInviteUser_NotManager_Forbidden(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "member"), nil)
	userRepo.On("GetByID", mock.Anything, reqID).
		Return(&entities.User{ID: reqID, Role: "member"}, nil)

	err := svc.InviteUser(context.Background(), orgID, reqID, "new@x.com", "member")
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestInviteUser_OrgNotFound(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "owner"), nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(nil, apperr.NotFound("Organization", orgID.String()))

	err := svc.InviteUser(context.Background(), orgID, reqID, "x@y.z", "member")
	require.Error(t, err)
}

func TestInviteUser_InviterNotFound(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "owner"), nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(&entities.Organization{ID: orgID, Name: "Acme"}, nil)
	userRepo.On("GetByID", mock.Anything, reqID).Return(nil, assert.AnError)

	err := svc.InviteUser(context.Background(), orgID, reqID, "x@y.z", "member")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestInviteUser_UpsertError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	inviteRepo := new(mockInviteRepo)
	svc := newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "owner"), nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(&entities.Organization{ID: orgID, Name: "Acme"}, nil)
	userRepo.On("GetByID", mock.Anything, reqID).Return(&entities.User{ID: reqID, Email: "o@x.y", FullName: "O"}, nil)
	inviteRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entities.Invitation")).Return(assert.AnError)

	err := svc.InviteUser(context.Background(), orgID, reqID, "x@y.z", "member")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestInviteUser_DefaultRoleWhenInvalid(t *testing.T) {
	// Invalid role passed → defaults to "member" and succeeds.
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	inviteRepo := new(mockInviteRepo)
	mailer := new(mockMailer)
	svc := newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, mailer)

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "owner"), nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(&entities.Organization{ID: orgID, Name: "Acme"}, nil)
	userRepo.On("GetByID", mock.Anything, reqID).Return(&entities.User{ID: reqID, Email: "o@x.y", FullName: "O"}, nil)
	inviteRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(inv *entities.Invitation) bool {
		return inv.Role == "member" // defaulted
	})).Return(nil)
	mailer.On("Send", mock.Anything, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	require.NoError(t, svc.InviteUser(context.Background(), orgID, reqID, "x@y.z", "supreme-leader"))
}

func TestAcceptInvitation_InvalidToken(t *testing.T) {
	inviteRepo := new(mockInviteRepo)
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), inviteRepo, new(mockUserRepo), new(mockMailer))

	inviteRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, apperr.NotFound("Invitation", "x"))

	_, err := svc.AcceptInvitation(context.Background(), uuid.New(), "bad-token")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestAcceptInvitation_UserNotFound(t *testing.T) {
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), inviteRepo, userRepo, new(mockMailer))

	userID := uuid.New()
	inv := &entities.Invitation{
		ID: uuid.New(), OrgID: uuid.New(), Email: "a@x.y", TokenHash: "h",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	inviteRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(inv, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError)

	_, err := svc.AcceptInvitation(context.Background(), userID, "t")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}

func TestAcceptInvitation_CreateMembershipError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, new(mockMailer))

	userID := uuid.New()
	orgID := uuid.New()
	inv := &entities.Invitation{
		ID: uuid.New(), OrgID: orgID, Email: "a@x.y", TokenHash: "h",
		Role: "member", ExpiresAt: time.Now().Add(time.Hour),
	}
	inviteRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(inv, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(&entities.User{ID: userID, Email: "a@x.y"}, nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(&entities.Organization{ID: orgID}, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).Return(nil, apperr.NotFound("OrgMembership", ""))
	memberRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.OrgMembership")).Return(assert.AnError)

	_, err := svc.AcceptInvitation(context.Background(), userID, "t")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestAcceptInvitation_MarkAcceptedError_StillSucceeds(t *testing.T) {
	// MarkAccepted failure is non-fatal — org is still returned.
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, new(mockMailer))

	userID := uuid.New()
	orgID := uuid.New()
	inv := &entities.Invitation{
		ID: uuid.New(), OrgID: orgID, Email: "a@x.y", TokenHash: "h",
		Role: "member", ExpiresAt: time.Now().Add(time.Hour),
	}
	inviteRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(inv, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(&entities.User{ID: userID, Email: "a@x.y"}, nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(&entities.Organization{ID: orgID}, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(&entities.OrgMembership{OrgID: orgID, UserID: userID, Role: "member"}, nil) // already member
	inviteRepo.On("MarkAccepted", mock.Anything, mock.Anything).Return(assert.AnError)

	org, err := svc.AcceptInvitation(context.Background(), userID, "t")
	require.NoError(t, err)
	assert.Equal(t, orgID, org.ID)
}

func TestAcceptInvitation_OrgNotFound(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(orgRepo, new(mockMemberRepo), inviteRepo, userRepo, new(mockMailer))

	userID := uuid.New()
	orgID := uuid.New()
	inv := &entities.Invitation{
		ID: uuid.New(), OrgID: orgID, Email: "a@x.y", TokenHash: "h",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	inviteRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(inv, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(&entities.User{ID: userID, Email: "a@x.y"}, nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(nil, assert.AnError)

	_, err := svc.AcceptInvitation(context.Background(), userID, "t")
	require.Error(t, err)
}

func TestUpdateOrganization_EmptyName(t *testing.T) {
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	_, err := svc.UpdateOrganization(context.Background(), uuid.New(), uuid.New(), "   ")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestUpdateOrganization_OrgGetByIDError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "admin"), nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(nil, apperr.NotFound("Organization", orgID.String()))

	_, err := svc.UpdateOrganization(context.Background(), orgID, reqID, "New Name")
	require.Error(t, err)
}

func TestUpdateOrganization_UpdateRepoError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	org := &entities.Organization{ID: orgID, Name: "Old", OwnerID: reqID}
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "admin"), nil)
	orgRepo.On("GetByID", mock.Anything, orgID).Return(org, nil)
	orgRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.Organization")).Return(nil, assert.AnError)

	_, err := svc.UpdateOrganization(context.Background(), orgID, reqID, "New")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestDeleteOrganization_NotOwner(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "admin"), nil)

	err := svc.DeleteOrganization(context.Background(), orgID, reqID)
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestDeleteOrganization_SoftDeleteError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))

	orgID, reqID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, reqID).Return(membership(orgID, reqID, "owner"), nil)
	orgRepo.On("SoftDelete", mock.Anything, orgID).Return(assert.AnError)

	err := svc.DeleteOrganization(context.Background(), orgID, reqID)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestUpdateMemberRole_UpdateError(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, ownerID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(membership(orgID, targetID, "member"), nil)
	memberRepo.On("Update", ctx, mock.AnythingOfType("*entities.OrgMembership")).Return(assert.AnError)

	err := svc.UpdateMemberRole(ctx, orgID, ownerID, targetID, "admin")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

// ─── RemoveMember ─────────────────────────────────────────────────────────────

func TestRemoveMember_AdminCanRemoveMember(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, adminID, targetID := uuid.New(), uuid.New(), uuid.New()
	targetMem := membership(orgID, targetID, "member")

	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(membership(orgID, adminID, "admin"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, targetID).Return(targetMem, nil)
	memberRepo.On("Delete", ctx, orgID, targetID).Return(nil)

	err := svc.RemoveMember(ctx, orgID, adminID, targetID)

	require.NoError(t, err)
}

func TestRemoveMember_SelfRemoval(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))
	ctx := context.Background()

	orgID, memberID := uuid.New(), uuid.New()
	mem := membership(orgID, memberID, "member")

	memberRepo.On("GetByOrgAndUser", ctx, orgID, memberID).Return(mem, nil)
	userRepo.On("GetByID", ctx, memberID).
		Return(&entities.User{ID: memberID, Role: "member"}, nil)
	memberRepo.On("Delete", ctx, orgID, memberID).Return(nil)

	err := svc.RemoveMember(ctx, orgID, memberID, memberID)

	require.NoError(t, err)
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, adminID, ownerID := uuid.New(), uuid.New(), uuid.New()

	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(membership(orgID, adminID, "admin"), nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, ownerID).Return(membership(orgID, ownerID, "owner"), nil)

	err := svc.RemoveMember(ctx, orgID, adminID, ownerID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestRemoveMember_PlainMemberCannotRemoveOther(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))
	ctx := context.Background()

	orgID, requesterID, targetID := uuid.New(), uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, requesterID).Return(membership(orgID, requesterID, "member"), nil)
	userRepo.On("GetByID", ctx, requesterID).
		Return(&entities.User{ID: requesterID, Role: "member"}, nil)

	err := svc.RemoveMember(ctx, orgID, requesterID, targetID)

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

// ─── InviteUser ───────────────────────────────────────────────────────────────

func TestInviteUser_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	mailer := new(mockMailer)
	svc := newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, mailer)
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	o := &entities.Organization{ID: orgID, Name: "Acme Corp"}
	inviter := &entities.User{ID: adminID, FullName: "Admin Alice"}

	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(membership(orgID, adminID, "admin"), nil)
	orgRepo.On("GetByID", ctx, orgID).Return(o, nil)
	userRepo.On("GetByID", ctx, adminID).Return(inviter, nil)
	inviteRepo.On("Upsert", ctx, mock.AnythingOfType("*entities.Invitation")).Return(nil)
	mailer.On("Send", ctx, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	err := svc.InviteUser(ctx, orgID, adminID, "newuser@example.com", "member")

	require.NoError(t, err)
	inviteRepo.AssertExpectations(t)
	mailer.AssertCalled(t, "Send", ctx, mock.AnythingOfType("ports.EmailMessage"))
}

func TestInviteUser_NonAdminForbidden(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), userRepo, new(mockMailer))
	ctx := context.Background()

	orgID, memberID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, memberID).Return(membership(orgID, memberID, "member"), nil)
	userRepo.On("GetByID", ctx, memberID).
		Return(&entities.User{ID: memberID, Role: "member"}, nil)

	err := svc.InviteUser(ctx, orgID, memberID, "someone@example.com", "member")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestInviteUser_InvalidEmail(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	svc := newOrgService(new(mockOrgRepo), memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	orgID, adminID := uuid.New(), uuid.New()
	memberRepo.On("GetByOrgAndUser", ctx, orgID, adminID).Return(membership(orgID, adminID, "admin"), nil)

	err := svc.InviteUser(ctx, orgID, adminID, "not-an-email", "member")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

// ─── AcceptInvitation ─────────────────────────────────────────────────────────

func TestAcceptInvitation_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	inv := &entities.Invitation{
		ID:        uuid.New(),
		OrgID:     orgID,
		Email:     "alice@example.com",
		Role:      "member",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	user := &entities.User{ID: userID, Email: "alice@example.com"}
	o := &entities.Organization{ID: orgID, Name: "Acme"}

	inviteRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(inv, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(o, nil)
	memberRepo.On("GetByOrgAndUser", ctx, orgID, userID).Return(nil, apperr.NotFound("OrgMembership", userID.String()))
	memberRepo.On("Create", ctx, mock.AnythingOfType("*entities.OrgMembership")).Return(nil)
	inviteRepo.On("MarkAccepted", ctx, mock.AnythingOfType("string")).Return(nil)

	result, err := svc.AcceptInvitation(ctx, userID, "raw-token")

	require.NoError(t, err)
	assert.Equal(t, orgID, result.ID)
	memberRepo.AssertExpectations(t)
}

func TestAcceptInvitation_EmailMismatch(t *testing.T) {
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), inviteRepo, userRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	inv := &entities.Invitation{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Email:     "alice@example.com",
		Role:      "member",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	bob := &entities.User{ID: userID, Email: "bob@example.com"}

	inviteRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(inv, nil)
	userRepo.On("GetByID", ctx, userID).Return(bob, nil)

	_, err := svc.AcceptInvitation(ctx, userID, "raw-token")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestAcceptInvitation_ExpiredToken(t *testing.T) {
	inviteRepo := new(mockInviteRepo)
	svc := newOrgService(new(mockOrgRepo), new(mockMemberRepo), inviteRepo, new(mockUserRepo), new(mockMailer))
	ctx := context.Background()

	expired := &entities.Invitation{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Email:     "alice@example.com",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	inviteRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(expired, nil)

	_, err := svc.AcceptInvitation(ctx, uuid.New(), "raw-token")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestAcceptInvitation_AlreadyMember_Idempotent(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	svc := newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	inv := &entities.Invitation{
		ID:        uuid.New(),
		OrgID:     orgID,
		Email:     "alice@example.com",
		Role:      "member",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	inviteRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(inv, nil)
	userRepo.On("GetByID", ctx, userID).Return(&entities.User{ID: userID, Email: "alice@example.com"}, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(&entities.Organization{ID: orgID, Name: "Acme"}, nil)
	// User is already a member — no Create called
	memberRepo.On("GetByOrgAndUser", ctx, orgID, userID).Return(membership(orgID, userID, "member"), nil)
	inviteRepo.On("MarkAccepted", ctx, mock.AnythingOfType("string")).Return(nil)

	_, err := svc.AcceptInvitation(ctx, userID, "raw-token")

	require.NoError(t, err)
	memberRepo.AssertNotCalled(t, "Create")
}
