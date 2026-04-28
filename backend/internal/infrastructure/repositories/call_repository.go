package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	repohelpers "github.com/rekall/backend/internal/infrastructure/repositories/helpers"
	apperr "github.com/rekall/backend/pkg/errors"
)

// CallRepository implements ports.CallRepository using GORM.
type CallRepository struct {
	db *gorm.DB
}

// NewCallRepository creates a CallRepository backed by the given GORM DB.
func NewCallRepository(db *gorm.DB) *CallRepository {
	return &CallRepository{db: db}
}

// Create persists a new call row and returns the stored entity.
func (r *CallRepository) Create(ctx context.Context, call *entities.Call) (*entities.Call, error) {
	if err := r.db.WithContext(ctx).Create(call).Error; err != nil {
		return nil, err
	}
	return call, nil
}

// GetByID retrieves a non-deleted call by primary key.
// GORM's soft-delete filter is applied automatically.
func (r *CallRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Call, error) {
	var call entities.Call
	err := r.db.WithContext(ctx).First(&call, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Call", id.String())
		}
		return nil, err
	}
	return &call, nil
}

// List returns a paginated slice of non-deleted calls matching the filter and the total count.
func (r *CallRepository) List(
	ctx context.Context,
	filter ports.ListCallsFilter,
	page, perPage int,
) ([]*entities.Call, int, error) {
	var (
		calls []*entities.Call
		total int64
	)

	base := r.db.WithContext(ctx).Model(&entities.Call{})
	base = repohelpers.ApplyCallFilter(base, filter)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := base.Order("created_at DESC").Limit(perPage).Offset(offset).Find(&calls).Error; err != nil {
		return nil, 0, err
	}

	return calls, int(total), nil
}

// Update saves changes to an existing call row.
func (r *CallRepository) Update(ctx context.Context, call *entities.Call) (*entities.Call, error) {
	if err := r.db.WithContext(ctx).Save(call).Error; err != nil {
		return nil, err
	}
	return call, nil
}

// SoftDelete sets deleted_at on the matching call row.
// GORM handles this automatically when the model has a DeletedAt *time.Time field.
func (r *CallRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&entities.Call{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound("Call", id.String())
	}
	return nil
}
