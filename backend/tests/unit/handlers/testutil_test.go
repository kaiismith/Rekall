package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"

	"github.com/stretchr/testify/require"
)

const (
	testSecret = "test-secret-key"
	testIssuer = "rekall-test"
)

// ─── JWT helper ───────────────────────────────────────────────────────────────

func signToken(t *testing.T, userID uuid.UUID, role string) string {
	t.Helper()
	now := time.Now().UTC()
	claims := infraauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    testIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
		Email: "test@example.com",
		Role:  role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return signed
}

// injectClaims is a test middleware that plants JWT claims directly into the
// gin context, bypassing the actual JWT parse step. Use this for handler tests
// that need an authenticated context without going through Authenticate.
func injectClaims(userID uuid.UUID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := &infraauth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
			Email:            "test@example.com",
			Role:             role,
		}
		c.Set("auth_claims", claims)
		c.Next()
	}
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func doRequest(router *gin.Engine, method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	router.ServeHTTP(w, req)
	return w
}

func doRequestWithToken(t *testing.T, router *gin.Engine, method, path string, body *bytes.Buffer, userID uuid.UUID, role string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer "+signToken(t, userID, role))
	router.ServeHTTP(w, req)
	return w
}

// ─── Mock: UserRepository ─────────────────────────────────────────────────────

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, u *entities.User) (*entities.User, error) {
	args := m.Called(ctx, u)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}
func (m *mockUserRepo) List(ctx context.Context, page, perPage int) ([]*entities.User, int, error) {
	args := m.Called(ctx, page, perPage)
	return args.Get(0).([]*entities.User), args.Int(1), args.Error(2)
}
func (m *mockUserRepo) Update(ctx context.Context, u *entities.User) (*entities.User, error) {
	args := m.Called(ctx, u)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}
func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockUserRepo) SetEmailVerified(ctx context.Context, id uuid.UUID, v bool) error {
	return m.Called(ctx, id, v).Error(0)
}
func (m *mockUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	return m.Called(ctx, id, hash).Error(0)
}
func (m *mockUserRepo) SetRoleByEmail(ctx context.Context, email, role string) error {
	return m.Called(ctx, email, role).Error(0)
}
func (m *mockUserRepo) DemoteAdminsExcept(ctx context.Context, keep []string) (int, error) {
	args := m.Called(ctx, keep)
	return args.Int(0), args.Error(1)
}

// ─── Mock: TokenRepository ────────────────────────────────────────────────────

type mockTokenRepo struct{ mock.Mock }

func (m *mockTokenRepo) CreateRefreshToken(ctx context.Context, t *entities.RefreshToken) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTokenRepo) GetRefreshToken(ctx context.Context, hash string) (*entities.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.PasswordResetToken), args.Error(1)
}
func (m *mockTokenRepo) MarkPasswordResetTokenUsed(ctx context.Context, hash string) error {
	return m.Called(ctx, hash).Error(0)
}
func (m *mockTokenRepo) InvalidatePendingPasswordResetTokens(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// ─── Mock: EmailSender ────────────────────────────────────────────────────────

type mockMailer struct{ mock.Mock }

func (m *mockMailer) Send(ctx context.Context, msg ports.EmailMessage) error {
	return m.Called(ctx, msg).Error(0)
}

// ─── Mock: OrganizationRepository ────────────────────────────────────────────

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

// ─── Mock: OrgMembershipRepository ───────────────────────────────────────────

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

// ─── Mock: InvitationRepository ──────────────────────────────────────────────

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

// ─── Mock: CallRepository ─────────────────────────────────────────────────────

type mockCallRepo struct{ mock.Mock }

func (m *mockCallRepo) Create(ctx context.Context, call *entities.Call) (*entities.Call, error) {
	args := m.Called(ctx, call)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Call), args.Error(1)
}
func (m *mockCallRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Call, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Call), args.Error(1)
}
func (m *mockCallRepo) List(ctx context.Context, filter ports.ListCallsFilter, page, perPage int) ([]*entities.Call, int, error) {
	args := m.Called(ctx, filter, page, perPage)
	return args.Get(0).([]*entities.Call), args.Int(1), args.Error(2)
}
func (m *mockCallRepo) Update(ctx context.Context, call *entities.Call) (*entities.Call, error) {
	args := m.Called(ctx, call)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Call), args.Error(1)
}
func (m *mockCallRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// ─── Mock: DepartmentRepository ──────────────────────────────────────────────

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

// ─── Mock: MeetingRepository ─────────────────────────────────────────────────

type mockMeetingRepo struct{ mock.Mock }

func (m *mockMeetingRepo) Create(ctx context.Context, mtg *entities.Meeting) error {
	return m.Called(ctx, mtg).Error(0)
}
func (m *mockMeetingRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Meeting, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Meeting), args.Error(1)
}
func (m *mockMeetingRepo) GetByCode(ctx context.Context, code string) (*entities.Meeting, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Meeting), args.Error(1)
}
func (m *mockMeetingRepo) Update(ctx context.Context, mtg *entities.Meeting) error {
	return m.Called(ctx, mtg).Error(0)
}
func (m *mockMeetingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockMeetingRepo) ListByHost(ctx context.Context, hostID uuid.UUID, status string) ([]*entities.Meeting, error) {
	args := m.Called(ctx, hostID, status)
	return args.Get(0).([]*entities.Meeting), args.Error(1)
}
func (m *mockMeetingRepo) CountActiveByHost(ctx context.Context, hostID uuid.UUID) (int64, error) {
	args := m.Called(ctx, hostID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockMeetingRepo) FindStaleWaiting(ctx context.Context, timeout time.Duration) ([]*entities.Meeting, error) {
	args := m.Called(ctx, timeout)
	return args.Get(0).([]*entities.Meeting), args.Error(1)
}
func (m *mockMeetingRepo) FindStaleActive(ctx context.Context, maxDuration time.Duration) ([]*entities.Meeting, error) {
	args := m.Called(ctx, maxDuration)
	return args.Get(0).([]*entities.Meeting), args.Error(1)
}
func (m *mockMeetingRepo) FindActiveWithNoParticipants(ctx context.Context) ([]*entities.Meeting, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Meeting), args.Error(1)
}
func (m *mockMeetingRepo) ListByUser(ctx context.Context, userID uuid.UUID, filter ports.ListMeetingsFilter) ([]*ports.MeetingListItem, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ports.MeetingListItem), args.Error(1)
}

// ─── Mock: MeetingParticipantRepository ──────────────────────────────────────

type mockParticipantRepo struct{ mock.Mock }

func (m *mockParticipantRepo) Create(ctx context.Context, p *entities.MeetingParticipant) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockParticipantRepo) GetByMeetingAndUser(ctx context.Context, meetingID, userID uuid.UUID) (*entities.MeetingParticipant, error) {
	args := m.Called(ctx, meetingID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.MeetingParticipant), args.Error(1)
}
func (m *mockParticipantRepo) Update(ctx context.Context, p *entities.MeetingParticipant) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockParticipantRepo) ListActive(ctx context.Context, meetingID uuid.UUID) ([]*entities.MeetingParticipant, error) {
	args := m.Called(ctx, meetingID)
	return args.Get(0).([]*entities.MeetingParticipant), args.Error(1)
}
func (m *mockParticipantRepo) CountActive(ctx context.Context, meetingID uuid.UUID) (int64, error) {
	args := m.Called(ctx, meetingID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockParticipantRepo) MarkAllLeft(ctx context.Context, meetingID uuid.UUID) error {
	return m.Called(ctx, meetingID).Error(0)
}

// ─── Mock: DepartmentMembershipRepository ────────────────────────────────────

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
