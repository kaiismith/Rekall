package helpers_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/helpers"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ─── Mocks ───────────────────────────────────────────────────────────────────

type mockOrgRepo struct{ mock.Mock }

func (m *mockOrgRepo) Create(ctx context.Context, org *entities.Organization) (*entities.Organization, error) {
	args := m.Called(ctx, org)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.Organization), args.Error(1)
}
func (m *mockOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.Organization), args.Error(1)
}
func (m *mockOrgRepo) GetBySlug(ctx context.Context, slug string) (*entities.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.Organization), args.Error(1)
}
func (m *mockOrgRepo) Update(ctx context.Context, org *entities.Organization) (*entities.Organization, error) {
	args := m.Called(ctx, org)
	if args.Get(0) == nil { return nil, args.Error(1) }
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
	if args.Get(0) == nil { return nil, args.Error(1) }
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

type mockDeptMemberRepo struct{ mock.Mock }

func (m *mockDeptMemberRepo) Create(ctx context.Context, mem *entities.DepartmentMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *mockDeptMemberRepo) GetByDeptAndUser(ctx context.Context, deptID, userID uuid.UUID) (*entities.DepartmentMembership, error) {
	args := m.Called(ctx, deptID, userID)
	if args.Get(0) == nil { return nil, args.Error(1) }
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

type mockTokenRepo struct{ mock.Mock }

func (m *mockTokenRepo) CreateRefreshToken(ctx context.Context, t *entities.RefreshToken) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTokenRepo) GetRefreshToken(ctx context.Context, hash string) (*entities.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.RefreshToken), args.Error(1)
}
func (m *mockTokenRepo) RevokeRefreshToken(ctx context.Context, hash string) error {
	return m.Called(ctx, hash).Error(0)
}
func (m *mockTokenRepo) RevokeAllRefreshTokens(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockTokenRepo) CreateVerificationToken(ctx context.Context, t *entities.EmailVerificationToken) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTokenRepo) GetVerificationToken(ctx context.Context, hash string) (*entities.EmailVerificationToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.EmailVerificationToken), args.Error(1)
}
func (m *mockTokenRepo) MarkVerificationTokenUsed(ctx context.Context, hash string) error {
	return m.Called(ctx, hash).Error(0)
}
func (m *mockTokenRepo) InvalidatePendingVerificationTokens(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockTokenRepo) CreatePasswordResetToken(ctx context.Context, t *entities.PasswordResetToken) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTokenRepo) GetPasswordResetToken(ctx context.Context, hash string) (*entities.PasswordResetToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.PasswordResetToken), args.Error(1)
}
func (m *mockTokenRepo) MarkPasswordResetTokenUsed(ctx context.Context, hash string) error {
	return m.Called(ctx, hash).Error(0)
}
func (m *mockTokenRepo) InvalidatePendingPasswordResetTokens(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type mockMailer struct{ mock.Mock }

func (m *mockMailer) Send(ctx context.Context, msg ports.EmailMessage) error {
	return m.Called(ctx, msg).Error(0)
}

type mockDeptRepo struct{ mock.Mock }

func (m *mockDeptRepo) Create(ctx context.Context, dept *entities.Department) (*entities.Department, error) {
	args := m.Called(ctx, dept)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Department, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.Department, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) Update(ctx context.Context, dept *entities.Department) (*entities.Department, error) {
	args := m.Called(ctx, dept)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*entities.Department), args.Error(1)
}
func (m *mockDeptRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// ─── UniqueSlug ──────────────────────────────────────────────────────────────

func TestUniqueSlug_BaseAvailable(t *testing.T) {
	repo := new(mockOrgRepo)
	repo.On("GetBySlug", mock.Anything, "acme").Return(nil, apperr.NotFound("Organization", "acme"))

	slug, err := helpers.UniqueSlug(context.Background(), repo, "acme")
	require.NoError(t, err)
	assert.Equal(t, "acme", slug)
}

func TestUniqueSlug_AddsSuffixWhenTaken(t *testing.T) {
	repo := new(mockOrgRepo)
	// First two are taken, third is available.
	existing := &entities.Organization{Name: "taken"}
	repo.On("GetBySlug", mock.Anything, "acme").Return(existing, nil)
	repo.On("GetBySlug", mock.Anything, "acme-2").Return(existing, nil)
	repo.On("GetBySlug", mock.Anything, "acme-3").Return(nil, apperr.NotFound("Organization", "acme-3"))

	slug, err := helpers.UniqueSlug(context.Background(), repo, "acme")
	require.NoError(t, err)
	assert.Equal(t, "acme-3", slug)
}

func TestUniqueSlug_FallsBackToUUIDWhenAllTaken(t *testing.T) {
	repo := new(mockOrgRepo)
	existing := &entities.Organization{Name: "taken"}
	// Base + all 9 suffixes (i=2..10) are taken.
	repo.On("GetBySlug", mock.Anything, mock.Anything).Return(existing, nil)

	slug, err := helpers.UniqueSlug(context.Background(), repo, "acme")
	require.NoError(t, err)
	// The UUID suffix branch returns base-<8 chars>.
	assert.Regexp(t, `^acme-[0-9a-f]{8}$`, slug)
}

func TestUniqueSlug_PropagatesRepoError(t *testing.T) {
	repo := new(mockOrgRepo)
	repo.On("GetBySlug", mock.Anything, "acme").Return(nil, assert.AnError)

	_, err := helpers.UniqueSlug(context.Background(), repo, "acme")
	require.Error(t, err)
}

// ─── NewRefreshToken ─────────────────────────────────────────────────────────

func TestNewRefreshToken_Success(t *testing.T) {
	userID := uuid.New()
	raw, tok, err := helpers.NewRefreshToken(userID, time.Hour)

	require.NoError(t, err)
	assert.NotEmpty(t, raw)
	assert.Equal(t, userID, tok.UserID)
	assert.NotEmpty(t, tok.TokenHash)
	assert.WithinDuration(t, time.Now().Add(time.Hour), tok.ExpiresAt, 5*time.Second)
}

// ─── SendVerificationEmail ───────────────────────────────────────────────────

func TestSendVerificationEmail_Success(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	user := &entities.User{ID: uuid.New(), Email: "alice@example.com", FullName: "Alice"}

	tokenRepo.On("CreateVerificationToken", mock.Anything, mock.AnythingOfType("*entities.EmailVerificationToken")).Return(nil)
	mailer.On("Send", mock.Anything, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	err := helpers.SendVerificationEmail(context.Background(), user, tokenRepo, 24*time.Hour, "http://app", mailer)
	require.NoError(t, err)

	mailer.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
}

func TestSendVerificationEmail_CreateTokenError(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	user := &entities.User{ID: uuid.New(), Email: "alice@example.com", FullName: "Alice"}

	tokenRepo.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(assert.AnError)

	err := helpers.SendVerificationEmail(context.Background(), user, tokenRepo, 24*time.Hour, "http://app", mailer)
	require.Error(t, err)
	mailer.AssertNotCalled(t, "Send")
}

func TestSendVerificationEmail_MailerError(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	user := &entities.User{ID: uuid.New(), Email: "a@b.com", FullName: "A"}

	tokenRepo.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(nil)
	mailer.On("Send", mock.Anything, mock.Anything).Return(assert.AnError)

	err := helpers.SendVerificationEmail(context.Background(), user, tokenRepo, 24*time.Hour, "http://app", mailer)
	require.Error(t, err)
}

// ─── RequireOrgMembership ────────────────────────────────────────────────────

func TestRequireOrgMembership_Success(t *testing.T) {
	repo := new(mockMemberRepo)
	orgID, userID := uuid.New(), uuid.New()
	m := &entities.OrgMembership{OrgID: orgID, UserID: userID, Role: "member"}

	repo.On("GetByOrgAndUser", mock.Anything, orgID, userID).Return(m, nil)

	result, err := helpers.RequireOrgMembership(context.Background(), repo, orgID, userID)
	require.NoError(t, err)
	assert.Equal(t, "member", result.Role)
}

func TestRequireOrgMembership_NotFound_ReturnsForbidden(t *testing.T) {
	repo := new(mockMemberRepo)
	orgID, userID := uuid.New(), uuid.New()
	repo.On("GetByOrgAndUser", mock.Anything, orgID, userID).Return(nil, apperr.NotFound("OrgMembership", ""))

	_, err := helpers.RequireOrgMembership(context.Background(), repo, orgID, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestRequireOrgMembership_OtherError_Propagates(t *testing.T) {
	repo := new(mockMemberRepo)
	orgID, userID := uuid.New(), uuid.New()
	repo.On("GetByOrgAndUser", mock.Anything, orgID, userID).Return(nil, assert.AnError)

	_, err := helpers.RequireOrgMembership(context.Background(), repo, orgID, userID)
	require.Error(t, err)
	assert.False(t, apperr.IsForbidden(err))
}

// ─── RequireDeptManager ──────────────────────────────────────────────────────

func TestRequireDeptManager_OrgAdminPasses(t *testing.T) {
	orgRepo := new(mockMemberRepo)
	deptRepo := new(mockDeptMemberRepo)

	orgID := uuid.New()
	userID := uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	orgRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(&entities.OrgMembership{OrgID: orgID, UserID: userID, Role: "admin"}, nil)

	err := helpers.RequireDeptManager(context.Background(), orgRepo, deptRepo, dept, userID)
	require.NoError(t, err)
	// Dept lookup not needed when org admin/owner.
	deptRepo.AssertNotCalled(t, "GetByDeptAndUser")
}

func TestRequireDeptManager_DeptHeadPasses(t *testing.T) {
	orgRepo := new(mockMemberRepo)
	deptRepo := new(mockDeptMemberRepo)

	orgID := uuid.New()
	userID := uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	orgRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(&entities.OrgMembership{OrgID: orgID, UserID: userID, Role: "member"}, nil)
	deptRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, userID).
		Return(&entities.DepartmentMembership{DepartmentID: dept.ID, UserID: userID, Role: "head"}, nil)

	err := helpers.RequireDeptManager(context.Background(), orgRepo, deptRepo, dept, userID)
	require.NoError(t, err)
}

func TestRequireDeptManager_PlainMember_Forbidden(t *testing.T) {
	orgRepo := new(mockMemberRepo)
	deptRepo := new(mockDeptMemberRepo)

	orgID := uuid.New()
	userID := uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	orgRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(&entities.OrgMembership{OrgID: orgID, UserID: userID, Role: "member"}, nil)
	deptRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, userID).
		Return(&entities.DepartmentMembership{DepartmentID: dept.ID, UserID: userID, Role: "member"}, nil)

	err := helpers.RequireDeptManager(context.Background(), orgRepo, deptRepo, dept, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestRequireDeptManager_NotOrgMember_Forbidden(t *testing.T) {
	orgRepo := new(mockMemberRepo)
	deptRepo := new(mockDeptMemberRepo)

	orgID := uuid.New()
	userID := uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	orgRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(nil, apperr.NotFound("OrgMembership", ""))

	err := helpers.RequireDeptManager(context.Background(), orgRepo, deptRepo, dept, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}

func TestRequireDeptManager_NotDeptMember_Forbidden(t *testing.T) {
	orgRepo := new(mockMemberRepo)
	deptRepo := new(mockDeptMemberRepo)

	orgID := uuid.New()
	userID := uuid.New()
	dept := &entities.Department{ID: uuid.New(), OrgID: orgID}

	orgRepo.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(&entities.OrgMembership{OrgID: orgID, UserID: userID, Role: "member"}, nil)
	deptRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, userID).
		Return(nil, apperr.NotFound("DeptMembership", ""))

	err := helpers.RequireDeptManager(context.Background(), orgRepo, deptRepo, dept, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsForbidden(err))
}
