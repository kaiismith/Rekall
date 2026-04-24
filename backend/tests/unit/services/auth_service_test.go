package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ─── Mocks ────────────────────────────────────────────────────────────────────

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

// ── TokenRepository mock ──────────────────────────────────────────────────────

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

// ── EmailSender mock ──────────────────────────────────────────────────────────

type mockMailer struct{ mock.Mock }

func (m *mockMailer) Send(ctx context.Context, msg ports.EmailMessage) error {
	return m.Called(ctx, msg).Error(0)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newAuthService(userRepo *mockUserRepo, tokenRepo *mockTokenRepo, mailer *mockMailer) *services.AuthService {
	return services.NewAuthService(
		userRepo, tokenRepo, mailer,
		"test-secret", "rekall-test", "http://localhost:5173",
		15*time.Minute, 7*24*time.Hour, time.Hour, 24*time.Hour,
		zap.NewNop(),
	)
}

func verifiedUser(id uuid.UUID, email, password string) *entities.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 4)
	return &entities.User{
		ID: id, Email: email, FullName: "Alice",
		PasswordHash: string(hash), EmailVerified: true, Role: "member",
	}
}

func validRefreshToken(userID uuid.UUID) *entities.RefreshToken {
	return &entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
}

// ─── Register ─────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, tokenRepo, mailer)
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "alice@example.com").Return(nil, apperr.NotFound("User", "alice@example.com"))
	userRepo.On("Create", ctx, mock.AnythingOfType("*entities.User")).Return(&entities.User{
		ID: uuid.New(), Email: "alice@example.com", FullName: "Alice", Role: "member",
	}, nil)
	tokenRepo.On("CreateVerificationToken", ctx, mock.AnythingOfType("*entities.EmailVerificationToken")).Return(nil)
	mailer.On("Send", ctx, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	user, err := svc.Register(ctx, "alice@example.com", "password1", "Alice")

	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
	userRepo.AssertExpectations(t)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	ctx := context.Background()

	existing := &entities.User{ID: uuid.New(), Email: "alice@example.com"}
	userRepo.On("GetByEmail", ctx, "alice@example.com").Return(existing, nil)

	_, err := svc.Register(ctx, "alice@example.com", "password1", "Alice")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 409, appErr.Status)
}

func TestRegister_InvalidEmail(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))

	_, err := svc.Register(context.Background(), "not-an-email", "password1", "Alice")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestRegister_WeakPassword(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))

	cases := []struct{ pw, desc string }{
		{"short1", "too short"},
		{"alllettters", "no digit"},
		{"12345678", "no letter"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := svc.Register(context.Background(), "a@b.com", tc.pw, "Name")
			require.Error(t, err)
			appErr, ok := apperr.AsAppError(err)
			require.True(t, ok)
			assert.Equal(t, 422, appErr.Status)
		})
	}
}

func TestRegister_MissingFullName(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))

	_, err := svc.Register(context.Background(), "a@b.com", "password1", "   ")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestRegister_UserRepoErrorOnLookup(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))

	// Non-NotFound error from GetByEmail.
	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(nil, assert.AnError)

	_, err := svc.Register(context.Background(), "alice@example.com", "Password1", "Alice")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestRegister_CreateError(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))

	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(nil, apperr.NotFound("User", "alice@example.com"))
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).Return(nil, assert.AnError)

	_, err := svc.Register(context.Background(), "alice@example.com", "Password1", "Alice")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestVerifyEmail_SetVerifiedError(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))

	userID := uuid.New()
	tokenRepo.On("GetVerificationToken", mock.Anything, mock.AnythingOfType("string")).Return(
		&entities.EmailVerificationToken{
			UserID:    userID,
			TokenHash: "h",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil)
	userRepo.On("SetEmailVerified", mock.Anything, userID, true).Return(assert.AnError)

	err := svc.VerifyEmail(context.Background(), "raw-token")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestVerifyEmail_MarkUsedError_SoftFail(t *testing.T) {
	// Marking the token used is non-fatal — verification still succeeds.
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))

	userID := uuid.New()
	tokenRepo.On("GetVerificationToken", mock.Anything, mock.Anything).Return(
		&entities.EmailVerificationToken{UserID: userID, TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	userRepo.On("SetEmailVerified", mock.Anything, userID, true).Return(nil)
	tokenRepo.On("MarkVerificationTokenUsed", mock.Anything, mock.Anything).Return(assert.AnError)

	require.NoError(t, svc.VerifyEmail(context.Background(), "raw-token"))
}

func TestResendVerification_RepoLookupError(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))

	userRepo.On("GetByEmail", mock.Anything, "a@b.com").Return(nil, assert.AnError)

	err := svc.ResendVerification(context.Background(), "a@b.com")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestForgotPassword_Success(t *testing.T) {
	// Full happy path: user exists → InvalidatePending → CreateToken → Send.
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, tokenRepo, mailer)

	userID := uuid.New()
	userRepo.On("GetByEmail", mock.Anything, "a@b.com").Return(&entities.User{
		ID: userID, Email: "a@b.com", FullName: "A",
	}, nil)
	tokenRepo.On("InvalidatePendingPasswordResetTokens", mock.Anything, userID).Return(nil)
	tokenRepo.On("CreatePasswordResetToken", mock.Anything, mock.AnythingOfType("*entities.PasswordResetToken")).Return(nil)
	mailer.On("Send", mock.Anything, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	require.NoError(t, svc.ForgotPassword(context.Background(), "a@b.com"))
	mailer.AssertExpectations(t)
}

func TestForgotPassword_CreateTokenError_SilentlySucceeds(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))

	userID := uuid.New()
	userRepo.On("GetByEmail", mock.Anything, "a@b.com").Return(&entities.User{
		ID: userID, Email: "a@b.com", FullName: "A",
	}, nil)
	tokenRepo.On("InvalidatePendingPasswordResetTokens", mock.Anything, userID).Return(nil)
	tokenRepo.On("CreatePasswordResetToken", mock.Anything, mock.Anything).Return(assert.AnError)

	// Token creation error is swallowed to prevent user enumeration.
	require.NoError(t, svc.ForgotPassword(context.Background(), "a@b.com"))
}

func TestLogin_RepoError(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))

	// Non-NotFound error from GetByEmail — should return 500.
	userRepo.On("GetByEmail", mock.Anything, "a@b.com").Return(nil, assert.AnError)

	_, _, _, err := svc.Login(context.Background(), "a@b.com", "Password1")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestLogin_CreateRefreshTokenError(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))

	userID := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 4)
	userRepo.On("GetByEmail", mock.Anything, "a@b.com").Return(&entities.User{
		ID: userID, Email: "a@b.com", FullName: "A", Role: "member",
		PasswordHash: string(hash), EmailVerified: true,
	}, nil)
	tokenRepo.On("CreateRefreshToken", mock.Anything, mock.AnythingOfType("*entities.RefreshToken")).Return(assert.AnError)

	_, _, _, err := svc.Login(context.Background(), "a@b.com", "Password1!")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestResetPassword_UserLookupError(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))

	userID := uuid.New()
	tokenRepo.On("GetPasswordResetToken", mock.Anything, mock.Anything).Return(
		&entities.PasswordResetToken{UserID: userID, TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	userRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(assert.AnError)

	err := svc.ResetPassword(context.Background(), "raw-token", "NewPassword1")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestRegister_VerificationEmailFailure_StillSucceeds(t *testing.T) {
	// Verification email failures should NOT fail registration — the user
	// exists and can resend the verification later.
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, tokenRepo, mailer)

	userID := uuid.New()
	userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return(nil, apperr.NotFound("User", "alice@example.com"))
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).Return(&entities.User{
		ID: userID, Email: "alice@example.com", FullName: "Alice", Role: "member",
	}, nil)
	// Verification token creation fails.
	tokenRepo.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(assert.AnError)

	user, err := svc.Register(context.Background(), "alice@example.com", "Password1", "Alice")
	require.NoError(t, err) // still succeeds
	assert.Equal(t, userID, user.ID)
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	user := verifiedUser(userID, "alice@example.com", "password1")

	userRepo.On("GetByEmail", ctx, "alice@example.com").Return(user, nil)
	tokenRepo.On("CreateRefreshToken", ctx, mock.AnythingOfType("*entities.RefreshToken")).Return(nil)

	gotUser, accessToken, rawRefresh, err := svc.Login(ctx, "alice@example.com", "password1")

	require.NoError(t, err)
	assert.Equal(t, userID, gotUser.ID)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, rawRefresh)
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	ctx := context.Background()

	user := verifiedUser(uuid.New(), "alice@example.com", "correctpass1")
	userRepo.On("GetByEmail", ctx, "alice@example.com").Return(user, nil)

	_, _, _, err := svc.Login(ctx, "alice@example.com", "wrongpass1")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}

func TestLogin_UnverifiedEmail(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password1"), 4)
	user := &entities.User{
		ID: uuid.New(), Email: "alice@example.com",
		PasswordHash: string(hash), EmailVerified: false,
	}
	userRepo.On("GetByEmail", ctx, "alice@example.com").Return(user, nil)

	_, _, _, err := svc.Login(ctx, "alice@example.com", "password1")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestLogin_UnknownEmail(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "nobody@example.com").Return(nil, apperr.NotFound("User", "nobody@example.com"))

	_, _, _, err := svc.Login(ctx, "nobody@example.com", "password1")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	// Must return 401, not 404, to prevent user enumeration.
	assert.Equal(t, 401, appErr.Status)
}

// ─── RefreshTokens ────────────────────────────────────────────────────────────

func TestRefreshTokens_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	user := verifiedUser(userID, "alice@example.com", "password1")
	tok := validRefreshToken(userID)

	tokenRepo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(tok, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	tokenRepo.On("RevokeRefreshToken", ctx, mock.AnythingOfType("string")).Return(nil)
	tokenRepo.On("CreateRefreshToken", ctx, mock.AnythingOfType("*entities.RefreshToken")).Return(nil)

	newAccess, newRefresh, err := svc.RefreshTokens(ctx, "raw-valid-token")

	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	tokenRepo.AssertCalled(t, "RevokeRefreshToken", ctx, mock.AnythingOfType("string"))
}

func TestRefreshTokens_ExpiredToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	ctx := context.Background()

	expired := &entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "hash",
		ExpiresAt: time.Now().Add(-time.Hour), // expired
	}
	tokenRepo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(expired, nil)

	_, _, err := svc.RefreshTokens(ctx, "raw-expired-token")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}

func TestRefreshTokens_RevokedToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	ctx := context.Background()

	now := time.Now()
	revoked := &entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "hash",
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &now,
	}
	tokenRepo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(revoked, nil)

	_, _, err := svc.RefreshTokens(ctx, "raw-revoked-token")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}

func TestRefreshTokens_TokenNotFound(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	ctx := context.Background()

	tokenRepo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).
		Return(nil, apperr.NotFound("RefreshToken", "hash"))

	_, _, err := svc.RefreshTokens(ctx, "unknown-token")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}

func TestRefreshTokens_UserNotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	tokenRepo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(&entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	userRepo.On("GetByID", ctx, userID).Return(nil, apperr.NotFound("User", userID.String()))

	_, _, err := svc.RefreshTokens(ctx, "raw-token")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 401, appErr.Status)
}

func TestRefreshTokens_RevokeError(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	tokenRepo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(&entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	userRepo.On("GetByID", ctx, userID).Return(&entities.User{
		ID: userID, Email: "a@b.com", FullName: "A", Role: "member",
	}, nil)
	tokenRepo.On("RevokeRefreshToken", ctx, mock.AnythingOfType("string")).Return(assert.AnError)

	_, _, err := svc.RefreshTokens(ctx, "raw-token")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestRefreshTokens_CreateNewError(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	tokenRepo.On("GetRefreshToken", ctx, mock.AnythingOfType("string")).Return(&entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	userRepo.On("GetByID", ctx, userID).Return(&entities.User{
		ID: userID, Email: "a@b.com", FullName: "A", Role: "member",
	}, nil)
	tokenRepo.On("RevokeRefreshToken", ctx, mock.AnythingOfType("string")).Return(nil)
	tokenRepo.On("CreateRefreshToken", ctx, mock.AnythingOfType("*entities.RefreshToken")).Return(assert.AnError)

	_, _, err := svc.RefreshTokens(ctx, "raw-token")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestLogout_RevokesToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	ctx := context.Background()

	tokenRepo.On("RevokeRefreshToken", ctx, mock.AnythingOfType("string")).Return(nil)

	err := svc.Logout(ctx, "some-raw-token")

	require.NoError(t, err)
	tokenRepo.AssertCalled(t, "RevokeRefreshToken", ctx, mock.AnythingOfType("string"))
}

func TestLogout_EmptyToken_NoOp(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))

	err := svc.Logout(context.Background(), "")

	require.NoError(t, err)
	tokenRepo.AssertNotCalled(t, "RevokeRefreshToken")
}

// ─── VerifyEmail ──────────────────────────────────────────────────────────────

func TestVerifyEmail_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	tok := &entities.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tokenRepo.On("GetVerificationToken", ctx, mock.AnythingOfType("string")).Return(tok, nil)
	userRepo.On("SetEmailVerified", ctx, userID, true).Return(nil)
	tokenRepo.On("MarkVerificationTokenUsed", ctx, mock.AnythingOfType("string")).Return(nil)

	err := svc.VerifyEmail(ctx, "raw-token-value")

	require.NoError(t, err)
	userRepo.AssertCalled(t, "SetEmailVerified", ctx, userID, true)
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	ctx := context.Background()

	tokenRepo.On("GetVerificationToken", ctx, mock.AnythingOfType("string")).
		Return(nil, apperr.NotFound("EmailVerificationToken", "hash"))

	err := svc.VerifyEmail(ctx, "bad-token")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	ctx := context.Background()

	expired := &entities.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	tokenRepo.On("GetVerificationToken", ctx, mock.AnythingOfType("string")).Return(expired, nil)

	err := svc.VerifyEmail(ctx, "expired-token")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

// ─── ResendVerification ───────────────────────────────────────────────────────

func TestResendVerification_UnknownEmail_SilentlySucceeds(t *testing.T) {
	userRepo := new(mockUserRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, new(mockTokenRepo), mailer)
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "nobody@example.com").
		Return(nil, apperr.NotFound("User", "nobody@example.com"))

	err := svc.ResendVerification(ctx, "nobody@example.com")

	// Must not reveal whether the address exists.
	require.NoError(t, err)
	mailer.AssertNotCalled(t, "Send")
}

func TestResendVerification_AlreadyVerified(t *testing.T) {
	userRepo := new(mockUserRepo)
	svc := newAuthService(userRepo, new(mockTokenRepo), new(mockMailer))
	ctx := context.Background()

	user := &entities.User{ID: uuid.New(), Email: "alice@example.com", EmailVerified: true}
	userRepo.On("GetByEmail", ctx, "alice@example.com").Return(user, nil)

	err := svc.ResendVerification(ctx, "alice@example.com")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

// ─── ForgotPassword ───────────────────────────────────────────────────────────

func TestForgotPassword_UnknownEmail_SilentlySucceeds(t *testing.T) {
	userRepo := new(mockUserRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, new(mockTokenRepo), mailer)
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "nobody@example.com").
		Return(nil, apperr.NotFound("User", "nobody@example.com"))

	err := svc.ForgotPassword(ctx, "nobody@example.com")

	// Must not reveal whether the address exists.
	require.NoError(t, err)
	mailer.AssertNotCalled(t, "Send")
}

func TestForgotPassword_KnownEmail_SendsEmail(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	mailer := new(mockMailer)
	svc := newAuthService(userRepo, tokenRepo, mailer)
	ctx := context.Background()

	userID := uuid.New()
	user := &entities.User{ID: userID, Email: "alice@example.com", FullName: "Alice"}
	userRepo.On("GetByEmail", ctx, "alice@example.com").Return(user, nil)
	tokenRepo.On("InvalidatePendingPasswordResetTokens", ctx, userID).Return(nil)
	tokenRepo.On("CreatePasswordResetToken", ctx, mock.AnythingOfType("*entities.PasswordResetToken")).Return(nil)
	mailer.On("Send", ctx, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	err := svc.ForgotPassword(ctx, "alice@example.com")

	require.NoError(t, err)
	mailer.AssertCalled(t, "Send", ctx, mock.AnythingOfType("ports.EmailMessage"))
}

// ─── ResetPassword ────────────────────────────────────────────────────────────

func TestResetPassword_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(userRepo, tokenRepo, new(mockMailer))
	ctx := context.Background()

	userID := uuid.New()
	tok := &entities.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tokenRepo.On("GetPasswordResetToken", ctx, mock.AnythingOfType("string")).Return(tok, nil)
	userRepo.On("UpdatePassword", ctx, userID, mock.AnythingOfType("string")).Return(nil)
	tokenRepo.On("MarkPasswordResetTokenUsed", ctx, mock.AnythingOfType("string")).Return(nil)
	tokenRepo.On("RevokeAllRefreshTokens", ctx, userID).Return(nil)

	err := svc.ResetPassword(ctx, "raw-reset-token", "newpass123")

	require.NoError(t, err)
	userRepo.AssertCalled(t, "UpdatePassword", ctx, userID, mock.AnythingOfType("string"))
	tokenRepo.AssertCalled(t, "RevokeAllRefreshTokens", ctx, userID)
}

func TestResetPassword_WeakPassword(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockTokenRepo), new(mockMailer))

	err := svc.ResetPassword(context.Background(), "token", "weak")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 422, appErr.Status)
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	tokenRepo := new(mockTokenRepo)
	svc := newAuthService(new(mockUserRepo), tokenRepo, new(mockMailer))
	ctx := context.Background()

	expired := &entities.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	tokenRepo.On("GetPasswordResetToken", ctx, mock.AnythingOfType("string")).Return(expired, nil)

	err := svc.ResetPassword(ctx, "expired-token", "newpass123")

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}
