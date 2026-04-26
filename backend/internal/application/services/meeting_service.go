package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

// MeetingService orchestrates business logic for meeting management.
type MeetingService struct {
	meetingRepo     ports.MeetingRepository
	participantRepo ports.MeetingParticipantRepository
	orgMemberRepo   ports.OrgMembershipRepository
	deptMemberRepo  ports.DepartmentMembershipRepository
	baseURL         string
	logger          *zap.Logger
}

func NewMeetingService(
	meetingRepo ports.MeetingRepository,
	participantRepo ports.MeetingParticipantRepository,
	orgMemberRepo ports.OrgMembershipRepository,
	deptMemberRepo ports.DepartmentMembershipRepository,
	baseURL string,
	logger *zap.Logger,
) *MeetingService {
	return &MeetingService{
		meetingRepo:     meetingRepo,
		participantRepo: participantRepo,
		orgMemberRepo:   orgMemberRepo,
		deptMemberRepo:  deptMemberRepo,
		baseURL:         baseURL,
		logger:          applogger.WithComponent(logger, "meeting_service"),
	}
}

// CreateMeetingInput holds the data required to create a new meeting.
type CreateMeetingInput struct {
	HostID    uuid.UUID
	Title     string
	Type      string // "open" | "private"
	ScopeType string // "organization" | "department" | ""
	ScopeID   *uuid.UUID
	// TranscriptionEnabled opts the meeting into the live-captions / ASR
	// feature; defaults to false when the host doesn't request it.
	TranscriptionEnabled bool
}

// CanJoinResult describes how a user may enter a meeting.
type CanJoinResult string

const (
	CanJoinDirect CanJoinResult = "direct"
	CanJoinKnock  CanJoinResult = "knock"
	CanJoinDenied CanJoinResult = "denied"
)

// CreateMeeting validates limits and persists a new meeting.
func (s *MeetingService) CreateMeeting(ctx context.Context, input CreateMeetingInput) (*entities.Meeting, error) {
	// Validate meeting type.
	if input.Type != entities.MeetingTypeOpen && input.Type != entities.MeetingTypePrivate {
		return nil, apperr.BadRequest("type must be 'open' or 'private'")
	}

	// Private meetings must declare a scope.
	if input.Type == entities.MeetingTypePrivate {
		if input.ScopeID == nil {
			return nil, apperr.BadRequest("scope_id is required for private meetings")
		}
		if input.ScopeType != entities.MeetingScopeOrg && input.ScopeType != entities.MeetingScopeDept {
			return nil, apperr.BadRequest("scope_type must be 'organization' or 'department'")
		}
		// Verify the host is a member of the scope they are claiming.
		if err := s.assertScopeMember(ctx, input.ScopeType, *input.ScopeID, input.HostID); err != nil {
			return nil, err
		}
	}

	// Enforce per-host active-meeting limit.
	count, err := s.meetingRepo.CountActiveByHost(ctx, input.HostID)
	if err != nil {
		return nil, apperr.Internal("failed to count active meetings")
	}
	if count >= entities.MeetingMaxPerHost {
		catalog.MeetingHostLimitExceeded.Warn(s.logger,
			zap.String("host_id", input.HostID.String()),
			zap.Int64("current_count", count),
		)
		return nil, apperr.BadRequest(fmt.Sprintf("maximum of %d active meetings allowed per host", entities.MeetingMaxPerHost))
	}

	code, err := generateJoinCode()
	if err != nil {
		return nil, apperr.Internal("failed to generate meeting code")
	}

	now := time.Now().UTC()
	m := &entities.Meeting{
		Code:                 code,
		Title:                input.Title,
		Type:                 input.Type,
		HostID:               input.HostID,
		Status:               entities.MeetingStatusWaiting,
		MaxParticipants:      entities.MeetingMaxParticipants,
		TranscriptionEnabled: input.TranscriptionEnabled,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if input.ScopeType != "" {
		m.ScopeType = &input.ScopeType
	}
	if input.ScopeID != nil {
		m.ScopeID = input.ScopeID
	}

	if err := s.meetingRepo.Create(ctx, m); err != nil {
		return nil, apperr.Internal("failed to create meeting")
	}

	catalog.MeetingCreated.Info(s.logger,
		zap.String("meeting_id", m.ID.String()),
		zap.String("host_id", m.HostID.String()),
		zap.String("code", m.Code),
		zap.String("type", m.Type),
	)
	return m, nil
}

// GetMeeting retrieves a meeting by ID.
func (s *MeetingService) GetMeeting(ctx context.Context, id uuid.UUID) (*entities.Meeting, error) {
	return s.meetingRepo.GetByID(ctx, id)
}

// GetMeetingByCode retrieves a meeting by its join code.
func (s *MeetingService) GetMeetingByCode(ctx context.Context, code string) (*entities.Meeting, error) {
	return s.meetingRepo.GetByCode(ctx, code)
}

// ListMyMeetings returns meetings hosted by the caller.
func (s *MeetingService) ListMyMeetings(ctx context.Context, hostID uuid.UUID, status string) ([]*entities.Meeting, error) {
	return s.meetingRepo.ListByHost(ctx, hostID, status)
}

// ListMeetingsWithMeta returns meetings where the caller is host or participant,
// enriched with duration and participant previews, filtered and sorted per the
// given parameters. statusFilter and sort correspond directly to query params
// from the API (e.g. "in_progress", "complete"; "created_at_desc", "title_asc").
func (s *MeetingService) ListMeetingsWithMeta(ctx context.Context, userID uuid.UUID, statusFilter, sort string) ([]*ports.MeetingListItem, error) {
	filter := ports.ListMeetingsFilter{Sort: sort}
	if statusFilter != "" {
		filter.Status = &statusFilter
	}
	return s.meetingRepo.ListByUser(ctx, userID, filter)
}

// ListMeetingsInScope returns meetings attached to the given scope (organization,
// department, or open items), enriched with duration and participant previews.
//
// Membership is enforced here — callers who are not members of the scope receive
// Forbidden and never see the list. The "open" scope has no membership check
// (every authenticated user can see their own open-item list); the repository
// restricts to rows where scope_type IS NULL and the caller is host or participant.
func (s *MeetingService) ListMeetingsInScope(
	ctx context.Context,
	userID uuid.UUID,
	scope *ports.ScopeFilter,
	statusFilter, sort string,
) ([]*ports.MeetingListItem, error) {
	if scope == nil {
		return s.ListMeetingsWithMeta(ctx, userID, statusFilter, sort)
	}

	switch scope.Kind {
	case ports.ScopeKindOrganization:
		if err := s.assertScopeMember(ctx, entities.MeetingScopeOrg, scope.ID, userID); err != nil {
			if apperr.IsNotFound(err) {
				return nil, apperr.Forbidden("caller is not a member of the organization")
			}
			return nil, err
		}
	case ports.ScopeKindDepartment:
		if err := s.assertScopeMember(ctx, entities.MeetingScopeDept, scope.ID, userID); err != nil {
			if apperr.IsNotFound(err) {
				return nil, apperr.Forbidden("caller is not a member of the department")
			}
			return nil, err
		}
	case ports.ScopeKindOpen:
		// Open items are visible to the caller as host or participant only —
		// the repository layer already encodes this when scope is nil. We need
		// a narrower query here: scope_type IS NULL AND (host_id=? OR participant).
		// Model this by passing the filter to the repo with the Open kind; the
		// repo applies both constraints.
	default:
		return nil, apperr.BadRequest("invalid scope kind")
	}

	filter := ports.ListMeetingsFilter{Sort: sort, Scope: scope}
	if statusFilter != "" {
		filter.Status = &statusFilter
	}
	return s.meetingRepo.ListByUser(ctx, userID, filter)
}

// CanJoin determines whether userID can enter the given meeting and how.
//
//   - CanJoinDirect  — user may join immediately (open meeting, or private + scope member)
//   - CanJoinKnock   — meeting is private, user is not in scope; must knock
//   - CanJoinDenied  — meeting is ended or at capacity
func (s *MeetingService) CanJoin(ctx context.Context, meeting *entities.Meeting, userID uuid.UUID) (CanJoinResult, error) {
	if meeting.IsEnded() {
		return CanJoinDenied, nil
	}

	// The host always joins directly — no knock, no scope check, no capacity
	// guard. They created the meeting; making them wait for an admitter
	// (often nobody else is there) would be a UX dead-end.
	if meeting.HostID == userID {
		return CanJoinDirect, nil
	}

	// Check capacity (only counts current active participants from DB; the hub
	// may have an in-flight count — that is fine as a secondary guard).
	count, err := s.participantRepo.CountActive(ctx, meeting.ID)
	if err != nil {
		return CanJoinDenied, apperr.Internal("failed to count participants")
	}
	if count >= int64(meeting.MaxParticipants) {
		catalog.MeetingParticipantLimitExceeded.Warn(s.logger,
			zap.String("meeting_id", meeting.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return CanJoinDenied, nil
	}

	if meeting.Type == entities.MeetingTypeOpen {
		return CanJoinDirect, nil
	}

	// Private meeting — check scope membership.
	if meeting.ScopeID == nil || meeting.ScopeType == nil {
		// Malformed private meeting; deny.
		return CanJoinDenied, nil
	}

	err = s.assertScopeMember(ctx, *meeting.ScopeType, *meeting.ScopeID, userID)
	if err == nil {
		return CanJoinDirect, nil
	}
	if apperr.IsNotFound(err) || apperr.IsForbidden(err) {
		return CanJoinKnock, nil
	}
	return CanJoinDenied, err
}

// RecordJoin creates or updates a participant row to mark a user as joined.
// If this is the first participant, the meeting transitions to 'active'.
func (s *MeetingService) RecordJoin(ctx context.Context, meeting *entities.Meeting, userID uuid.UUID, role string) error {
	now := time.Now().UTC()

	existing, err := s.participantRepo.GetByMeetingAndUser(ctx, meeting.ID, userID)
	if err != nil && !apperr.IsNotFound(err) {
		return apperr.Internal("failed to check participant record")
	}

	if existing != nil {
		// Rejoin — clear left_at.
		existing.JoinedAt = &now
		existing.LeftAt = nil
		if err := s.participantRepo.Update(ctx, existing); err != nil {
			return apperr.Internal("failed to update participant")
		}
	} else {
		p := &entities.MeetingParticipant{
			MeetingID: meeting.ID,
			UserID:    userID,
			Role:      role,
			JoinedAt:  &now,
		}
		if err := s.participantRepo.Create(ctx, p); err != nil {
			return apperr.Internal("failed to create participant record")
		}
	}

	// Transition meeting to active on first join.
	if meeting.IsWaiting() {
		meeting.Status = entities.MeetingStatusActive
		meeting.StartedAt = &now
		meeting.UpdatedAt = now
		if err := s.meetingRepo.Update(ctx, meeting); err != nil {
			return apperr.Internal("failed to activate meeting")
		}
		catalog.MeetingStarted.Info(s.logger,
			zap.String("meeting_id", meeting.ID.String()),
			zap.String("user_id", userID.String()),
		)
	}

	catalog.ParticipantJoined.Info(s.logger,
		zap.String("meeting_id", meeting.ID.String()),
		zap.String("user_id", userID.String()),
		zap.String("role", role),
	)
	return nil
}

// RecordLeave sets left_at for the participant. If no active participants remain
// AND the caller was the host, the meeting is ended.
func (s *MeetingService) RecordLeave(ctx context.Context, meeting *entities.Meeting, userID uuid.UUID) error {
	p, err := s.participantRepo.GetByMeetingAndUser(ctx, meeting.ID, userID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	p.LeftAt = &now
	if err := s.participantRepo.Update(ctx, p); err != nil {
		return apperr.Internal("failed to record leave")
	}

	catalog.ParticipantLeft.Info(s.logger,
		zap.String("meeting_id", meeting.ID.String()),
		zap.String("user_id", userID.String()),
	)

	// If host leaves, end the meeting.
	if p.IsHost() {
		return s.EndMeeting(ctx, meeting)
	}
	return nil
}

// EndMeeting marks the meeting as ended and evicts all remaining participants.
func (s *MeetingService) EndMeeting(ctx context.Context, meeting *entities.Meeting) error {
	if meeting.IsEnded() {
		return nil
	}
	now := time.Now().UTC()
	meeting.Status = entities.MeetingStatusEnded
	meeting.EndedAt = &now
	meeting.UpdatedAt = now
	if err := s.meetingRepo.Update(ctx, meeting); err != nil {
		return apperr.Internal("failed to end meeting")
	}
	if err := s.participantRepo.MarkAllLeft(ctx, meeting.ID); err != nil {
		return apperr.Internal("failed to mark participants as left")
	}
	catalog.MeetingEnded.Info(s.logger,
		zap.String("meeting_id", meeting.ID.String()),
	)
	return nil
}

// assertScopeMember returns nil if userID is a member of the given scope, or
// an apperr.NotFound / apperr.Forbidden error if they are not.
func (s *MeetingService) assertScopeMember(ctx context.Context, scopeType string, scopeID, userID uuid.UUID) error {
	switch scopeType {
	case entities.MeetingScopeOrg:
		_, err := s.orgMemberRepo.GetByOrgAndUser(ctx, scopeID, userID)
		return err
	case entities.MeetingScopeDept:
		_, err := s.deptMemberRepo.GetByDeptAndUser(ctx, scopeID, userID)
		return err
	default:
		return apperr.BadRequest("unknown scope_type: " + scopeType)
	}
}

// generateJoinCode returns a Google Meet–style code: "abc-defg-hij".
// Uses crypto/rand for unpredictability.
func generateJoinCode() (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	segments := []int{3, 4, 3}
	parts := make([]byte, 0, 12)

	for i, n := range segments {
		if i > 0 {
			parts = append(parts, '-')
		}
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for j := range buf {
			buf[j] = letters[int(buf[j])%len(letters)]
		}
		parts = append(parts, buf...)
	}
	return string(parts), nil
}
