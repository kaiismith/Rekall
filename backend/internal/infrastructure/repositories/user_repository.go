package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
)

// UserRepository implements ports.UserRepository using GORM.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a UserRepository backed by the given GORM DB.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create persists a new user row and returns the stored entity.
func (r *UserRepository) Create(ctx context.Context, user *entities.User) (*entities.User, error) {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID retrieves a non-deleted user by primary key.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("User", id.String())
		}
		return nil, err
	}
	return &user, nil
}

// FindByIDs retrieves a slice of non-deleted users by primary key.
func (r *UserRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.User, error) {
	if len(ids) == 0 {
		return []*entities.User{}, nil
	}
	var users []*entities.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetByEmail retrieves a non-deleted user by email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("User", email)
		}
		return nil, err
	}
	return &user, nil
}

// List returns a paginated slice of non-deleted users and the total count.
// GORM's soft-delete filter (WHERE deleted_at IS NULL) is applied automatically.
func (r *UserRepository) List(ctx context.Context, page, perPage int) ([]*entities.User, int, error) {
	var (
		users []*entities.User
		total int64
	)

	base := r.db.WithContext(ctx).Model(&entities.User{})

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := base.Order("created_at DESC").Limit(perPage).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, int(total), nil
}

// Update saves changes to an existing user row.
func (r *UserRepository) Update(ctx context.Context, user *entities.User) (*entities.User, error) {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// SoftDelete sets deleted_at on the matching user row.
// GORM handles this automatically when the model has a DeletedAt *time.Time field.
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entities.User{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound("User", id.String())
	}
	return nil
}

// SetEmailVerified updates the email_verified column for the given user.
func (r *UserRepository) SetEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	return r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("id = ?", id).
		Update("email_verified", verified).Error
}

// UpdatePassword replaces the password_hash column for the given user.
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	return r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("id = ?", id).
		Update("password_hash", hash).Error
}

// SetRoleByEmail sets the role column for the user with the given email.
// No rows are affected when the email is unknown — the caller treats that as
// a no-op (the AdminReconciler will then bootstrap-create the user if a
// password was supplied).
func (r *UserRepository) SetRoleByEmail(ctx context.Context, email, role string) error {
	return r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("email = ?", email).
		Update("role", role).Error
}

// DemoteAdminsExcept downgrades every current admin whose email is NOT in
// keepEmails to role="member". Returns the count of demoted users.
//
// keepEmails is matched case-insensitively (the env var is lowercased on
// load); this method does not lowercase incoming user.email so callers MUST
// store emails canonical-lowercased — which the auth service already does.
func (r *UserRepository) DemoteAdminsExcept(ctx context.Context, keepEmails []string) (int, error) {
	q := r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("role = ?", "admin")
	if len(keepEmails) > 0 {
		q = q.Where("email NOT IN ?", keepEmails)
	}
	res := q.Update("role", "member")
	return int(res.RowsAffected), res.Error
}
