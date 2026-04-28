package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// UserService orchestrates business logic for user management.
type UserService struct {
	repo   ports.UserRepository
	logger *zap.Logger
}

// NewUserService creates a UserService with its required dependencies.
// The logger is tagged with component="user_service" so every log line emitted
// here is automatically identified without repeating the tag at each call site.
func NewUserService(repo ports.UserRepository, logger *zap.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: applogger.WithComponent(logger, "user_service"),
	}
}

// CreateUser validates the input and persists a new user.
func (s *UserService) CreateUser(ctx context.Context, email, fullName, role string) (*entities.User, error) {
	if email == "" {
		return nil, apperr.BadRequest("email is required")
	}
	if fullName == "" {
		return nil, apperr.BadRequest("full_name is required")
	}
	if role == "" {
		role = "member"
	}

	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil && !apperr.IsNotFound(err) {
		catalog.UserEmailCheckFailed.Error(s.logger,
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, apperr.Internal("failed to create user")
	}
	if existing != nil {
		catalog.UserEmailConflict.Warn(s.logger,
			zap.String("email", email),
			zap.String("existing_user_id", existing.ID.String()),
		)
		return nil, apperr.Conflict("a user with this email already exists")
	}

	now := time.Now().UTC()
	user := &entities.User{
		ID:        uuid.New(),
		Email:     email,
		FullName:  fullName,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		catalog.UserCreateFailed.Error(s.logger,
			zap.Error(err),
			zap.String("email", email),
			zap.String("role", role),
		)
		return nil, apperr.Internal("failed to create user")
	}

	catalog.UserCreated.Info(s.logger,
		zap.String("user_id", created.ID.String()),
		zap.String("email", created.Email),
		zap.String("role", created.Role),
	)
	return created, nil
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if apperr.IsNotFound(err) {
			catalog.UserNotFound.Warn(s.logger,
				zap.String("user_id", id.String()),
			)
			return nil, apperr.NotFound("User", id.String())
		}
		catalog.UserGetFailed.Error(s.logger,
			zap.Error(err),
			zap.String("user_id", id.String()),
		)
		return nil, apperr.Internal("failed to retrieve user")
	}

	catalog.UserFetched.Debug(s.logger,
		zap.String("user_id", user.ID.String()),
		zap.String("role", user.Role),
	)
	return user, nil
}

// ListUsers returns a paginated list of users.
func (s *UserService) ListUsers(ctx context.Context, page, perPage int) ([]*entities.User, int, error) {
	users, total, err := s.repo.List(ctx, page, perPage)
	if err != nil {
		catalog.UserListFailed.Error(s.logger,
			zap.Error(err),
			zap.Int("page", page),
			zap.Int("per_page", perPage),
		)
		return nil, 0, apperr.Internal("failed to list users")
	}

	catalog.UsersListed.Debug(s.logger,
		zap.Int("count", len(users)),
		zap.Int("total", total),
		zap.Int("page", page),
		zap.Int("per_page", perPage),
	)
	return users, total, nil
}

// DeleteUser soft-deletes a user by ID.
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		catalog.UserDeleteFailed.Error(s.logger,
			zap.Error(err),
			zap.String("user_id", id.String()),
			zap.String("email", user.Email),
		)
		return apperr.Internal("failed to delete user")
	}

	catalog.UserDeleted.Info(s.logger,
		zap.String("user_id", id.String()),
		zap.String("email", user.Email),
		zap.String("role", user.Role),
	)
	return nil
}
