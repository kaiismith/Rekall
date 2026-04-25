package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/helpers"
	apputils "github.com/rekall/backend/internal/application/utils"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	infraemail "github.com/rekall/backend/internal/infrastructure/email"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

// OrganizationService orchestrates organization lifecycle operations:
// create, update, delete, member management, and invitation flow.
type OrganizationService struct {
	orgRepo    ports.OrganizationRepository
	memberRepo ports.OrgMembershipRepository
	inviteRepo ports.InvitationRepository
	userRepo   ports.UserRepository
	mailer     ports.EmailSender
	appBaseURL  string
	inviteTTL  time.Duration
	logger     *zap.Logger
}

// NewOrganizationService creates an OrganizationService with all required dependencies.
func NewOrganizationService(
	orgRepo ports.OrganizationRepository,
	memberRepo ports.OrgMembershipRepository,
	inviteRepo ports.InvitationRepository,
	userRepo ports.UserRepository,
	mailer ports.EmailSender,
	appBaseURL string,
	inviteTTL time.Duration,
	log *zap.Logger,
) *OrganizationService {
	return &OrganizationService{
		orgRepo:    orgRepo,
		memberRepo: memberRepo,
		inviteRepo: inviteRepo,
		userRepo:   userRepo,
		mailer:     mailer,
		appBaseURL: appBaseURL,
		inviteTTL:  inviteTTL,
		logger:     applogger.WithComponent(log, "organization_service"),
	}
}

// CreateOrganization creates a new org and adds the designated owner as a
// member with role="owner". When ownerEmail is the empty string, the caller
// (callerID) becomes the owner — admin self-service path. When ownerEmail is
// supplied (platform admin creating on behalf of another user), the org's
// OwnerID is resolved from that email instead. Unknown emails return 422.
func (s *OrganizationService) CreateOrganization(
	ctx context.Context,
	callerID uuid.UUID,
	name, ownerEmail string,
) (*entities.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Unprocessable("organization name is required", nil)
	}
	if len(name) > 100 {
		return nil, apperr.Unprocessable("organization name must be 100 characters or fewer", nil)
	}

	ownerID := callerID
	if ownerEmail != "" {
		owner, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(ownerEmail)))
		if err != nil {
			if apperr.IsNotFound(err) {
				return nil, apperr.Unprocessable("owner_email: no user with that email", nil)
			}
			return nil, apperr.Internal("failed to resolve owner_email")
		}
		ownerID = owner.ID
	}

	slug := apputils.GenerateSlug(name)

	// Ensure slug is unique; append a short suffix if taken.
	slug, err := helpers.UniqueSlug(ctx, s.orgRepo, slug)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	org := &entities.Organization{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		OwnerID:   ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, err := s.orgRepo.Create(ctx, org)
	if err != nil {
		return nil, apperr.Internal("failed to create organization")
	}

	// Add the designated owner as owner-level member.
	membership := &entities.OrgMembership{
		ID:       uuid.New(),
		OrgID:    created.ID,
		UserID:   ownerID,
		Role:     "owner",
		JoinedAt: now,
	}
	if err := s.memberRepo.Create(ctx, membership); err != nil {
		// The org was created; log but do not fail — the caller can retry membership.
		catalog.OwnerMembershipFailed.Error(s.logger,
			zap.String("org_id", created.ID.String()),
			zap.String("owner_id", ownerID.String()),
			zap.Error(err),
		)
	}

	catalog.OrgCreated.Info(s.logger,
		zap.String("org_id", created.ID.String()),
		zap.String("owner_id", ownerID.String()),
		zap.String("created_by", callerID.String()),
	)
	return created, nil
}

// GetOrganization returns the org if the requesting user is a member.
func (s *OrganizationService) GetOrganization(ctx context.Context, orgID, requesterID uuid.UUID) (*entities.Organization, error) {
	if _, err := helpers.RequireOrgMembership(ctx, s.memberRepo, orgID, requesterID); err != nil {
		return nil, err
	}
	return s.orgRepo.GetByID(ctx, orgID)
}

// UpdateOrganization changes the org name (and re-derives the slug). Only admins/owners may call this.
func (s *OrganizationService) UpdateOrganization(ctx context.Context, orgID, requesterID uuid.UUID, name string) (*entities.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Unprocessable("organization name is required", nil)
	}

	m, err := helpers.RequireOrgMembership(ctx, s.memberRepo, orgID, requesterID)
	if err != nil {
		return nil, err
	}
	if !m.IsAdmin() {
		return nil, apperr.Forbidden("only admins and owners can update organization settings")
	}

	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	org.Name = name
	org.UpdatedAt = time.Now().UTC()

	updated, err := s.orgRepo.Update(ctx, org)
	if err != nil {
		return nil, apperr.Internal("failed to update organization")
	}

	catalog.OrgUpdated.Info(s.logger,
		zap.String("org_id", orgID.String()),
		zap.String("requester_id", requesterID.String()),
	)
	return updated, nil
}

// DeleteOrganization soft-deletes an org. Only the owner may call this.
func (s *OrganizationService) DeleteOrganization(ctx context.Context, orgID, requesterID uuid.UUID) error {
	m, err := helpers.RequireOrgMembership(ctx, s.memberRepo, orgID, requesterID)
	if err != nil {
		return err
	}
	if !m.IsOwner() {
		return apperr.Forbidden("only the owner can delete an organization")
	}

	if err := s.orgRepo.SoftDelete(ctx, orgID); err != nil {
		return apperr.Internal("failed to delete organization")
	}

	catalog.OrgDeleted.Info(s.logger,
		zap.String("org_id", orgID.String()),
		zap.String("requester_id", requesterID.String()),
	)
	return nil
}

// ListOrganizations returns all orgs the requesting user belongs to.
func (s *OrganizationService) ListOrganizations(ctx context.Context, userID uuid.UUID) ([]*entities.Organization, error) {
	return s.orgRepo.ListByUserID(ctx, userID)
}

// ListMembers returns all memberships for the given org. Requires membership.
func (s *OrganizationService) ListMembers(ctx context.Context, orgID, requesterID uuid.UUID) ([]*entities.OrgMembership, error) {
	if _, err := helpers.RequireOrgMembership(ctx, s.memberRepo, orgID, requesterID); err != nil {
		return nil, err
	}
	return s.memberRepo.ListByOrg(ctx, orgID)
}

// UpdateMemberRole changes a member's role. Owners may set any role; admins may only set "member".
func (s *OrganizationService) UpdateMemberRole(ctx context.Context, orgID, requesterID, targetUserID uuid.UUID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "admin" && role != "member" {
		return apperr.Unprocessable("role must be 'admin' or 'member'", nil)
	}

	requester, err := helpers.RequireOrgMembership(ctx, s.memberRepo, orgID, requesterID)
	if err != nil {
		return err
	}
	if !requester.CanManageMembers() {
		return apperr.Forbidden("only admins and owners can update member roles")
	}
	// Only an owner can promote someone to admin.
	if role == "admin" && !requester.IsOwner() {
		return apperr.Forbidden("only the owner can grant admin role")
	}

	target, err := s.memberRepo.GetByOrgAndUser(ctx, orgID, targetUserID)
	if apperr.IsNotFound(err) {
		return apperr.NotFound("OrgMembership", targetUserID.String())
	}
	if err != nil {
		return apperr.Internal("failed to retrieve membership")
	}
	// Protect the owner seat.
	if target.IsOwner() {
		return apperr.Forbidden("owner role cannot be changed via this endpoint")
	}

	target.Role = role
	if err := s.memberRepo.Update(ctx, target); err != nil {
		return apperr.Internal("failed to update member role")
	}

	catalog.MemberUpdated.Info(s.logger,
		zap.String("org_id", orgID.String()),
		zap.String("target_user_id", targetUserID.String()),
		zap.String("new_role", role),
	)
	return nil
}

// RemoveMember removes a user from an org. Admins/owners can remove members; users can remove themselves.
func (s *OrganizationService) RemoveMember(ctx context.Context, orgID, requesterID, targetUserID uuid.UUID) error {
	requester, err := helpers.RequireOrgMembership(ctx, s.memberRepo, orgID, requesterID)
	if err != nil {
		return err
	}

	isSelf := requesterID == targetUserID
	if !isSelf && !requester.CanManageMembers() {
		return apperr.Forbidden("only admins and owners can remove other members")
	}

	target, err := s.memberRepo.GetByOrgAndUser(ctx, orgID, targetUserID)
	if apperr.IsNotFound(err) {
		return apperr.NotFound("OrgMembership", targetUserID.String())
	}
	if err != nil {
		return apperr.Internal("failed to retrieve membership")
	}
	if target.IsOwner() {
		return apperr.Forbidden("the owner cannot be removed; transfer ownership first")
	}

	if err := s.memberRepo.Delete(ctx, orgID, targetUserID); err != nil {
		return apperr.Internal("failed to remove member")
	}

	catalog.MemberRemoved.Info(s.logger,
		zap.String("org_id", orgID.String()),
		zap.String("user_id", targetUserID.String()),
	)
	return nil
}

// InviteUser creates or refreshes an invitation and sends an invitation email.
// Requires the requester to be an admin or owner.
func (s *OrganizationService) InviteUser(ctx context.Context, orgID, requesterID uuid.UUID, email, role string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.ToLower(strings.TrimSpace(role))
	if !emailRegexp.MatchString(email) {
		return apperr.Unprocessable("invalid email address", nil)
	}
	if role != "admin" && role != "member" {
		role = "member"
	}

	requester, err := helpers.RequireOrgMembership(ctx, s.memberRepo, orgID, requesterID)
	if err != nil {
		return err
	}
	if !requester.CanManageMembers() {
		return apperr.Forbidden("only admins and owners can invite users")
	}

	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}

	inviter, err := s.userRepo.GetByID(ctx, requesterID)
	if err != nil {
		return apperr.Internal("failed to retrieve inviter")
	}

	raw, err := infraauth.GenerateRawToken()
	if err != nil {
		return apperr.Internal("failed to generate invitation token")
	}

	inv := &entities.Invitation{
		ID:        uuid.New(),
		OrgID:     orgID,
		InvitedBy: requesterID,
		Email:     email,
		TokenHash: infraauth.HashToken(raw),
		Role:      role,
		ExpiresAt: time.Now().UTC().Add(s.inviteTTL),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.inviteRepo.Upsert(ctx, inv); err != nil {
		return apperr.Internal("failed to save invitation")
	}

	acceptURL := fmt.Sprintf("%s/invitations/accept?token=%s", s.appBaseURL, raw)
	_ = s.mailer.Send(ctx, ports.EmailMessage{
		To:      email,
		Subject: infraemail.InvitationEmailSubject(org.Name),
		Body:    infraemail.InvitationEmailBody(org.Name, inviter.FullName, role, acceptURL),
	})

	catalog.InvitationSent.Info(s.logger,
		zap.String("org_id", orgID.String()),
		zap.String("invited_email", email),
		zap.String("invited_by", requesterID.String()),
	)
	return nil
}

// AcceptInvitation validates the token, creates a membership, and marks the invitation accepted.
// If the user is already a member, the invitation is marked accepted and no error is returned.
func (s *OrganizationService) AcceptInvitation(ctx context.Context, userID uuid.UUID, rawToken string) (*entities.Organization, error) {
	hash := infraauth.HashToken(rawToken)
	inv, err := s.inviteRepo.GetByTokenHash(ctx, hash)
	if err != nil || !inv.IsValid() {
		catalog.InvitationInvalid.Warn(s.logger)
		return nil, apperr.BadRequest("invitation link is invalid, expired, or already accepted")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperr.Unauthorized("user not found")
	}

	// Invitation email must match the authenticated user's email.
	if !strings.EqualFold(inv.Email, user.Email) {
		return nil, apperr.Forbidden("this invitation was sent to a different email address")
	}

	org, err := s.orgRepo.GetByID(ctx, inv.OrgID)
	if err != nil {
		return nil, apperr.Internal("failed to retrieve organization")
	}

	// Idempotent: skip membership creation if already a member.
	_, existingErr := s.memberRepo.GetByOrgAndUser(ctx, inv.OrgID, userID)
	if apperr.IsNotFound(existingErr) {
		membership := &entities.OrgMembership{
			ID:       uuid.New(),
			OrgID:    inv.OrgID,
			UserID:   userID,
			Role:     inv.Role,
			JoinedAt: time.Now().UTC(),
		}
		if err := s.memberRepo.Create(ctx, membership); err != nil {
			return nil, apperr.Internal("failed to create membership")
		}
		catalog.MemberAdded.Info(s.logger,
			zap.String("org_id", inv.OrgID.String()),
			zap.String("user_id", userID.String()),
			zap.String("role", inv.Role),
		)
	}

	if err := s.inviteRepo.MarkAccepted(ctx, hash); err != nil {
		catalog.InvitationMarkAcceptedFailed.Error(s.logger, zap.Error(err))
	}

	catalog.InvitationAccepted.Info(s.logger,
		zap.String("org_id", inv.OrgID.String()),
		zap.String("user_id", userID.String()),
	)
	return org, nil
}

