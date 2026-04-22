package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
	"gorm.io/gorm"
)

// InvitationRepository implements ports.InvitationRepository using GORM.
type InvitationRepository struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Upsert inserts a new invitation or updates the token_hash and expires_at of an existing
// pending (not yet accepted) invitation for the same (org_id, email) pair.
func (r *InvitationRepository) Upsert(ctx context.Context, inv *entities.Invitation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing entities.Invitation
		err := tx.Where("org_id = ? AND email = ? AND accepted_at IS NULL", inv.OrgID, inv.Email).
			First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(inv).Error
		}
		if err != nil {
			return err
		}
		// Refresh token and expiry on duplicate
		return tx.Model(&existing).Updates(map[string]interface{}{
			"token_hash": inv.TokenHash,
			"expires_at": inv.ExpiresAt,
			"invited_by": inv.InvitedBy,
			"role":       inv.Role,
		}).Error
	})
}

func (r *InvitationRepository) GetByTokenHash(ctx context.Context, hash string) (*entities.Invitation, error) {
	var inv entities.Invitation
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", hash).
		First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Invitation", hash)
	}
	return &inv, err
}

func (r *InvitationRepository) GetPendingByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*entities.Invitation, error) {
	var inv entities.Invitation
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND email = ? AND accepted_at IS NULL", orgID, email).
		First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Invitation", email)
	}
	return &inv, err
}

func (r *InvitationRepository) MarkAccepted(ctx context.Context, hash string) error {
	result := r.db.WithContext(ctx).
		Model(&entities.Invitation{}).
		Where("token_hash = ? AND accepted_at IS NULL", hash).
		Update("accepted_at", time.Now().UTC())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound("Invitation", hash)
	}
	return nil
}
