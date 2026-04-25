package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AdminReconciler keeps the platform-admin role on the users table in sync
// with the PLATFORM_ADMIN_EMAILS environment variable.
//
// On every server boot the reconciler:
//
//  1. (Optional) bootstrap-creates a User row for any listed email that does
//     not yet have one, using PLATFORM_ADMIN_BOOTSTRAP_PASSWORD as the
//     initial password. This is a first-run convenience — once the user
//     exists, the password is NOT re-applied.
//  2. Sets role="admin" for every user whose email is in the list.
//  3. Demotes every user with role="admin" whose email is NOT in the list
//     back to role="member".
//
// The env var is the source of truth — re-deploying with a different list is
// the supported way to rotate platform admins.
type AdminReconciler struct {
	userRepo        ports.UserRepository
	emails          []string
	bootstrapPwd    string
	logger          *zap.Logger
}

// NewAdminReconciler builds a reconciler. emails should already be
// trimmed/lowercased/de-duplicated by the config loader.
func NewAdminReconciler(
	userRepo ports.UserRepository,
	emails []string,
	bootstrapPwd string,
	logger *zap.Logger,
) *AdminReconciler {
	return &AdminReconciler{
		userRepo:     userRepo,
		emails:       emails,
		bootstrapPwd: bootstrapPwd,
		logger:       applogger.WithComponent(logger, "admin_reconciler"),
	}
}

// ReconcileResult summarises what changed during a single reconciliation pass.
type ReconcileResult struct {
	Created  int
	Promoted int
	Demoted  int
}

// Reconcile runs one full pass. Idempotent — running it N times is
// indistinguishable from running it once.
func (r *AdminReconciler) Reconcile(ctx context.Context) (ReconcileResult, error) {
	var result ReconcileResult

	// Step 1 — bootstrap missing users when a password is supplied.
	if r.bootstrapPwd != "" {
		for _, email := range r.emails {
			_, err := r.userRepo.GetByEmail(ctx, email)
			if err == nil {
				continue
			}
			if !apperr.IsNotFound(err) {
				r.logger.Warn("admin reconciler: lookup failed", zap.String("email", email), zap.Error(err))
				continue
			}
			if err := r.bootstrapCreate(ctx, email); err != nil {
				r.logger.Warn("admin reconciler: bootstrap failed", zap.String("email", email), zap.Error(err))
				continue
			}
			result.Created++
		}
	}

	// Step 2 — promote every listed email (including just-created ones).
	for _, email := range r.emails {
		// Read the user first so we only count actual transitions.
		u, err := r.userRepo.GetByEmail(ctx, email)
		if err != nil {
			// NotFound here is expected when bootstrap was disabled and the
			// user has not registered yet — skip silently.
			if !apperr.IsNotFound(err) {
				r.logger.Warn("admin reconciler: lookup failed", zap.String("email", email), zap.Error(err))
			}
			continue
		}
		if u.Role == "admin" {
			continue
		}
		if err := r.userRepo.SetRoleByEmail(ctx, email, "admin"); err != nil {
			r.logger.Warn("admin reconciler: promotion failed", zap.String("email", email), zap.Error(err))
			continue
		}
		result.Promoted++
	}

	// Step 3 — demote any current admin not on the list.
	demoted, err := r.userRepo.DemoteAdminsExcept(ctx, r.emails)
	if err != nil {
		r.logger.Warn("admin reconciler: bulk demote failed", zap.Error(err))
	} else {
		result.Demoted = demoted
	}

	r.logger.Info("admin reconciler done",
		zap.Int("created", result.Created),
		zap.Int("promoted", result.Promoted),
		zap.Int("demoted", result.Demoted),
		zap.Int("listed_emails", len(r.emails)),
	)
	return result, nil
}

func (r *AdminReconciler) bootstrapCreate(ctx context.Context, email string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(r.bootstrapPwd), bcryptCost)
	if err != nil {
		return apperr.Internal("failed to hash bootstrap password")
	}
	now := time.Now().UTC()
	_, err = r.userRepo.Create(ctx, &entities.User{
		ID:            uuid.New(),
		Email:         email,
		FullName:      deriveBootstrapName(email),
		Role:          "admin",
		PasswordHash:  string(hash),
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	return err
}

// deriveBootstrapName turns an email's local-part into a human-ish display
// name ("travis.duong" → "Travis Duong"). Only used for bootstrap; users can
// edit it later through the profile page.
func deriveBootstrapName(email string) string {
	at := strings.IndexByte(email, '@')
	local := email
	if at > 0 {
		local = email[:at]
	}
	parts := strings.FieldsFunc(local, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return local
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, strings.ToUpper(p[:1])+p[1:])
	}
	return strings.Join(out, " ")
}
