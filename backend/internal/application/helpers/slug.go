package helpers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
)

// UniqueSlug returns base if unused, otherwise base-<short-suffix>.
func UniqueSlug(ctx context.Context, orgRepo ports.OrganizationRepository, base string) (string, error) {
	candidate := base
	for i := 2; i <= 10; i++ {
		_, err := orgRepo.GetBySlug(ctx, candidate)
		if apperr.IsNotFound(err) {
			return candidate, nil
		}
		if err != nil {
			return "", apperr.Internal("failed to check slug availability")
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	// Fall back to UUID suffix to guarantee uniqueness.
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:8]), nil
}
