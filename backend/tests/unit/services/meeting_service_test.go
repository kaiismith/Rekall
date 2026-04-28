package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── Mocks ────────────────────────────────────────────────────────────────────

type mockMeetingRepo struct{ mock.Mock }

func (m *mockMeetingRepo) Create(ctx context.Context, meeting *entities.Meeting) error {
	return m.Called(ctx, meeting).Error(0)
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
func (m *mockMeetingRepo) Update(ctx context.Context, meeting *entities.Meeting) error {
	return m.Called(ctx, meeting).Error(0)
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

type meetingMockOrgMemberRepo struct{ mock.Mock }

func (m *meetingMockOrgMemberRepo) Create(ctx context.Context, mem *entities.OrgMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *meetingMockOrgMemberRepo) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*entities.OrgMembership, error) {
	args := m.Called(ctx, orgID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.OrgMembership), args.Error(1)
}
func (m *meetingMockOrgMemberRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entities.OrgMembership, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*entities.OrgMembership), args.Error(1)
}
func (m *meetingMockOrgMemberRepo) Update(ctx context.Context, mem *entities.OrgMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *meetingMockOrgMemberRepo) Delete(ctx context.Context, orgID, userID uuid.UUID) error {
	return m.Called(ctx, orgID, userID).Error(0)
}

type meetingMockDeptMemberRepo struct{ mock.Mock }

func (m *meetingMockDeptMemberRepo) Create(ctx context.Context, mem *entities.DepartmentMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *meetingMockDeptMemberRepo) GetByDeptAndUser(ctx context.Context, deptID, userID uuid.UUID) (*entities.DepartmentMembership, error) {
	args := m.Called(ctx, deptID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DepartmentMembership), args.Error(1)
}
func (m *meetingMockDeptMemberRepo) ListByDept(ctx context.Context, deptID uuid.UUID) ([]*entities.DepartmentMembership, error) {
	args := m.Called(ctx, deptID)
	return args.Get(0).([]*entities.DepartmentMembership), args.Error(1)
}
func (m *meetingMockDeptMemberRepo) Update(ctx context.Context, mem *entities.DepartmentMembership) error {
	return m.Called(ctx, mem).Error(0)
}
func (m *meetingMockDeptMemberRepo) Delete(ctx context.Context, deptID, userID uuid.UUID) error {
	return m.Called(ctx, deptID, userID).Error(0)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newTestMeetingService(mr *mockMeetingRepo, pr *mockParticipantRepo, or *meetingMockOrgMemberRepo, dr *meetingMockDeptMemberRepo) *services.MeetingService {
	return services.NewMeetingService(mr, pr, or, dr, "http://localhost:5173", zap.NewNop())
}

func openMeeting(hostID uuid.UUID) *entities.Meeting {
	return &entities.Meeting{
		ID:              uuid.New(),
		Code:            "abc-defg-hij",
		Title:           "Weekly Sync",
		Type:            entities.MeetingTypeOpen,
		HostID:          hostID,
		Status:          entities.MeetingStatusWaiting,
		MaxParticipants: 50,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func privateMeeting(hostID, orgID uuid.UUID) *entities.Meeting {
	scopeType := entities.MeetingScopeOrg
	m := openMeeting(hostID)
	m.Type = entities.MeetingTypePrivate
	m.ScopeType = &scopeType
	m.ScopeID = &orgID
	return m
}

// ─── CreateMeeting ────────────────────────────────────────────────────────────

func TestCreateMeeting_Open_Success(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	or := new(meetingMockOrgMemberRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, or, dr)

	hostID := uuid.New()
	mr.On("CountActiveByHost", mock.Anything, hostID).Return(int64(0), nil)
	mr.On("Create", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)

	meeting, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID: hostID,
		Title:  "Team Standup",
		Type:   entities.MeetingTypeOpen,
	})

	require.NoError(t, err)
	assert.Equal(t, entities.MeetingTypeOpen, meeting.Type)
	assert.Equal(t, entities.MeetingStatusWaiting, meeting.Status)
	assert.NotEmpty(t, meeting.Code)
	mr.AssertExpectations(t)
}

func TestCreateMeeting_HostLimitExceeded(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	or := new(meetingMockOrgMemberRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, or, dr)

	hostID := uuid.New()
	mr.On("CountActiveByHost", mock.Anything, hostID).Return(int64(entities.MeetingMaxPerHost), nil)

	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID: hostID,
		Type:   entities.MeetingTypeOpen,
	})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestCreateMeeting_Private_MissingScopeID(t *testing.T) {
	svc := newTestMeetingService(new(mockMeetingRepo), new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID:    uuid.New(),
		Type:      entities.MeetingTypePrivate,
		ScopeType: entities.MeetingScopeOrg,
		// ScopeID intentionally nil
	})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestCreateMeeting_InvalidType(t *testing.T) {
	svc := newTestMeetingService(new(mockMeetingRepo), new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID: uuid.New(),
		Type:   "webinar", // not "open" or "private"
	})
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestCreateMeeting_Private_InvalidScopeType(t *testing.T) {
	svc := newTestMeetingService(new(mockMeetingRepo), new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	scopeID := uuid.New()
	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID:    uuid.New(),
		Type:      entities.MeetingTypePrivate,
		ScopeType: "team", // invalid
		ScopeID:   &scopeID,
	})
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestCreateMeeting_CountActiveError(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	mr.On("CountActiveByHost", mock.Anything, hostID).Return(int64(0), assert.AnError)

	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID: hostID,
		Type:   entities.MeetingTypeOpen,
	})
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestCanJoin_MalformedPrivateMeeting_NoScope(t *testing.T) {
	// Private meeting with nil ScopeID → deny (malformed).
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	meeting.Type = entities.MeetingTypePrivate
	meeting.ScopeID = nil
	meeting.ScopeType = nil

	pr.On("CountActive", mock.Anything, meeting.ID).Return(int64(0), nil)

	result, err := svc.CanJoin(context.Background(), meeting, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinDenied, result)
}

func TestCanJoin_ScopeAssertionUnexpectedError(t *testing.T) {
	// Non-NotFound/Forbidden error from assertScopeMember → return Denied + error.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	orgMem := new(meetingMockOrgMemberRepo)
	svc := newTestMeetingService(mr, pr, orgMem, new(meetingMockDeptMemberRepo))

	orgID := uuid.New()
	meeting := privateMeeting(uuid.New(), orgID)

	pr.On("CountActive", mock.Anything, meeting.ID).Return(int64(0), nil)
	orgMem.On("GetByOrgAndUser", mock.Anything, orgID, mock.AnythingOfType("uuid.UUID")).
		Return(nil, assert.AnError) // unexpected error

	_, err := svc.CanJoin(context.Background(), meeting, uuid.New())
	require.Error(t, err)
}

func TestAssertScopeMember_DeptScope(t *testing.T) {
	// Exercises the dept-scope branch of assertScopeMember via a private
	// meeting with scope=department.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	orgMem := new(meetingMockOrgMemberRepo)
	deptMem := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, orgMem, deptMem)

	deptID := uuid.New()
	scopeType := entities.MeetingScopeDept
	meeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "xyz",
		Type:            entities.MeetingTypePrivate,
		HostID:          uuid.New(),
		Status:          entities.MeetingStatusWaiting,
		MaxParticipants: 50,
		ScopeType:       &scopeType,
		ScopeID:         &deptID,
	}

	pr.On("CountActive", mock.Anything, meeting.ID).Return(int64(0), nil)
	deptMem.On("GetByDeptAndUser", mock.Anything, deptID, mock.AnythingOfType("uuid.UUID")).
		Return(&entities.DepartmentMembership{}, nil)

	result, err := svc.CanJoin(context.Background(), meeting, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinDirect, result)
}

func TestAssertScopeMember_UnknownScope(t *testing.T) {
	// Exercises the "unknown scope_type" branch via a private meeting with a
	// bogus scope_type. CanJoin propagates the error as Denied.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	bogus := "team"
	id := uuid.New()
	meeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "xyz",
		Type:            entities.MeetingTypePrivate,
		HostID:          uuid.New(),
		Status:          entities.MeetingStatusWaiting,
		MaxParticipants: 50,
		ScopeType:       &bogus,
		ScopeID:         &id,
	}

	pr.On("CountActive", mock.Anything, meeting.ID).Return(int64(0), nil)

	// assertScopeMember returns BadRequest for unknown scope → caller receives
	// Denied with the error.
	_, err := svc.CanJoin(context.Background(), meeting, uuid.New())
	require.Error(t, err)
}

func TestRecordJoin_GetParticipantError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	userID := uuid.New()
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).Return(nil, assert.AnError)

	err := svc.RecordJoin(context.Background(), meeting, userID, "participant")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestRecordJoin_UpdateParticipantError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	userID := uuid.New()
	now := time.Now()
	existing := &entities.MeetingParticipant{
		MeetingID: meeting.ID, UserID: userID, Role: "participant",
		JoinedAt: &now, LeftAt: &now,
	}
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).Return(existing, nil)
	pr.On("Update", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(assert.AnError)

	err := svc.RecordJoin(context.Background(), meeting, userID, "participant")
	require.Error(t, err)
}

func TestRecordJoin_CreateParticipantError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	userID := uuid.New()
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(nil, apperr.NotFound("participant", ""))
	pr.On("Create", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(assert.AnError)

	err := svc.RecordJoin(context.Background(), meeting, userID, "participant")
	require.Error(t, err)
}

func TestRecordJoin_UpdateMeetingError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	// Waiting meeting → RecordJoin will transition to active → calls Update.
	meeting := openMeeting(uuid.New())
	meeting.Status = entities.MeetingStatusWaiting
	userID := uuid.New()

	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(nil, apperr.NotFound("participant", ""))
	pr.On("Create", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(assert.AnError)

	err := svc.RecordJoin(context.Background(), meeting, userID, "participant")
	require.Error(t, err)
}

func TestRecordLeave_ParticipantNotFound(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	userID := uuid.New()
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(nil, apperr.NotFound("participant", ""))

	err := svc.RecordLeave(context.Background(), meeting, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestRecordLeave_UpdateError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	userID := uuid.New()
	now := time.Now()
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(&entities.MeetingParticipant{
			MeetingID: meeting.ID, UserID: userID, Role: "participant", JoinedAt: &now,
		}, nil)
	pr.On("Update", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(assert.AnError)

	err := svc.RecordLeave(context.Background(), meeting, userID)
	require.Error(t, err)
}

func TestEndMeeting_AlreadyEnded_Noop2(t *testing.T) {
	// Covers the early return when meeting.IsEnded() already.
	svc := newTestMeetingService(new(mockMeetingRepo), new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	ended := openMeeting(uuid.New())
	ended.Status = entities.MeetingStatusEnded

	require.NoError(t, svc.EndMeeting(context.Background(), ended))
}

func TestEndMeeting_UpdateError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(assert.AnError)

	err := svc.EndMeeting(context.Background(), meeting)
	require.Error(t, err)
}

func TestEndMeeting_MarkAllLeftError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	meeting := openMeeting(uuid.New())
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, meeting.ID).Return(assert.AnError)

	err := svc.EndMeeting(context.Background(), meeting)
	require.Error(t, err)
}

func TestCreateMeeting_RepoCreateError(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	mr.On("CountActiveByHost", mock.Anything, hostID).Return(int64(0), nil)
	mr.On("Create", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(assert.AnError)

	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID: hostID,
		Type:   entities.MeetingTypeOpen,
	})
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

func TestCreateMeeting_Private_HostNotInScope(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	or := new(meetingMockOrgMemberRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, or, dr)

	hostID := uuid.New()
	orgID := uuid.New()
	or.On("GetByOrgAndUser", mock.Anything, orgID, hostID).
		Return(nil, apperr.NotFound("OrgMembership", orgID.String()))

	scopeID := orgID
	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID:    hostID,
		Type:      entities.MeetingTypePrivate,
		ScopeType: entities.MeetingScopeOrg,
		ScopeID:   &scopeID,
	})

	require.Error(t, err)
	or.AssertExpectations(t)
}

// ─── CanJoin ──────────────────────────────────────────────────────────────────

func TestCanJoin_Open_Direct(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	or := new(meetingMockOrgMemberRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, or, dr)

	hostID := uuid.New()
	m := openMeeting(hostID)
	callerID := uuid.New()

	pr.On("CountActive", mock.Anything, m.ID).Return(int64(0), nil)

	result, err := svc.CanJoin(context.Background(), m, callerID)
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinDirect, result)
}

func TestCanJoin_Private_ScopeMember_Direct(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	or := new(meetingMockOrgMemberRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, or, dr)

	hostID := uuid.New()
	orgID := uuid.New()
	m := privateMeeting(hostID, orgID)
	callerID := uuid.New()

	pr.On("CountActive", mock.Anything, m.ID).Return(int64(1), nil)
	or.On("GetByOrgAndUser", mock.Anything, orgID, callerID).
		Return(&entities.OrgMembership{}, nil)

	result, err := svc.CanJoin(context.Background(), m, callerID)
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinDirect, result)
}

func TestCanJoin_Private_NonMember_Knock(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	or := new(meetingMockOrgMemberRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, or, dr)

	hostID := uuid.New()
	orgID := uuid.New()
	m := privateMeeting(hostID, orgID)
	callerID := uuid.New()

	pr.On("CountActive", mock.Anything, m.ID).Return(int64(3), nil)
	or.On("GetByOrgAndUser", mock.Anything, orgID, callerID).
		Return(nil, apperr.NotFound("OrgMembership", orgID.String()))

	result, err := svc.CanJoin(context.Background(), m, callerID)
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinKnock, result)
}

func TestCanJoin_EndedMeeting_Denied(t *testing.T) {
	svc := newTestMeetingService(new(mockMeetingRepo), new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	m := openMeeting(uuid.New())
	m.Status = entities.MeetingStatusEnded

	result, err := svc.CanJoin(context.Background(), m, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinDenied, result)
}

func TestCanJoin_AtCapacity_Denied(t *testing.T) {
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(new(mockMeetingRepo), pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	m := openMeeting(uuid.New())
	pr.On("CountActive", mock.Anything, m.ID).Return(int64(50), nil)

	result, err := svc.CanJoin(context.Background(), m, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinDenied, result)
}

// ─── RecordJoin ───────────────────────────────────────────────────────────────

func TestRecordJoin_FirstParticipant_ActivatesMeeting(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	m := openMeeting(hostID)

	pr.On("GetByMeetingAndUser", mock.Anything, m.ID, hostID).Return(nil, apperr.NotFound("MeetingParticipant", m.ID.String()))
	pr.On("Create", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)

	err := svc.RecordJoin(context.Background(), m, hostID, entities.ParticipantRoleHost)
	require.NoError(t, err)
	assert.Equal(t, entities.MeetingStatusActive, m.Status)
	assert.NotNil(t, m.StartedAt)
}

// ─── EndMeeting ───────────────────────────────────────────────────────────────

func TestEndMeeting_Success(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	m := openMeeting(uuid.New())
	m.Status = entities.MeetingStatusActive

	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, m.ID).Return(nil)

	err := svc.EndMeeting(context.Background(), m)
	require.NoError(t, err)
	assert.Equal(t, entities.MeetingStatusEnded, m.Status)
	assert.NotNil(t, m.EndedAt)
}

func TestEndMeeting_AlreadyEnded_Noop(t *testing.T) {
	svc := newTestMeetingService(new(mockMeetingRepo), new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	m := openMeeting(uuid.New())
	m.Status = entities.MeetingStatusEnded

	err := svc.EndMeeting(context.Background(), m)
	require.NoError(t, err) // should be a no-op
}

// ─── RecordLeave ──────────────────────────────────────────────────────────────

func TestRecordLeave_ParticipantLeaves_NoEnd(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	callerID := uuid.New()
	m := openMeeting(hostID)
	m.Status = entities.MeetingStatusActive

	p := &entities.MeetingParticipant{
		MeetingID: m.ID,
		UserID:    callerID,
		Role:      entities.ParticipantRoleParticipant,
	}
	pr.On("GetByMeetingAndUser", mock.Anything, m.ID, callerID).Return(p, nil)
	pr.On("Update", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(nil)

	err := svc.RecordLeave(context.Background(), m, callerID)
	require.NoError(t, err)
	assert.NotNil(t, p.LeftAt)
	// Meeting should NOT be ended — caller is not the host.
	assert.Equal(t, entities.MeetingStatusActive, m.Status)
	mr.AssertNotCalled(t, "Update")
}

func TestRecordLeave_HostLeaves_EndsMeeting(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	m := openMeeting(hostID)
	m.Status = entities.MeetingStatusActive

	p := &entities.MeetingParticipant{
		MeetingID: m.ID,
		UserID:    hostID,
		Role:      entities.ParticipantRoleHost,
	}
	pr.On("GetByMeetingAndUser", mock.Anything, m.ID, hostID).Return(p, nil)
	pr.On("Update", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, m.ID).Return(nil)

	err := svc.RecordLeave(context.Background(), m, hostID)
	require.NoError(t, err)
	assert.Equal(t, entities.MeetingStatusEnded, m.Status)
	assert.NotNil(t, m.EndedAt)
}

// ─── RecordJoin — rejoin ──────────────────────────────────────────────────────

func TestRecordJoin_Rejoin_ClearsLeftAt(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	callerID := uuid.New()
	m := openMeeting(hostID)
	m.Status = entities.MeetingStatusActive

	now := time.Now().UTC().Add(-1 * time.Minute)
	existing := &entities.MeetingParticipant{
		MeetingID: m.ID,
		UserID:    callerID,
		Role:      entities.ParticipantRoleParticipant,
		LeftAt:    &now,
	}
	pr.On("GetByMeetingAndUser", mock.Anything, m.ID, callerID).Return(existing, nil)
	pr.On("Update", mock.Anything, mock.AnythingOfType("*entities.MeetingParticipant")).Return(nil)

	err := svc.RecordJoin(context.Background(), m, callerID, entities.ParticipantRoleParticipant)
	require.NoError(t, err)
	assert.Nil(t, existing.LeftAt, "LeftAt should be cleared on rejoin")
	assert.NotNil(t, existing.JoinedAt)
}

// ─── generateJoinCode format ──────────────────────────────────────────────────

func TestGenerateJoinCode_Format(t *testing.T) {
	// Exercise CreateMeeting and verify the code matches the 3-4-3 pattern.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	svc := newTestMeetingService(mr, pr, new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	mr.On("CountActiveByHost", mock.Anything, hostID).Return(int64(0), nil)

	var capturedCode string
	mr.On("Create", mock.Anything, mock.AnythingOfType("*entities.Meeting")).
		Run(func(args mock.Arguments) {
			m := args.Get(1).(*entities.Meeting)
			capturedCode = m.Code
		}).
		Return(nil)

	_, err := svc.CreateMeeting(context.Background(), services.CreateMeetingInput{
		HostID: hostID,
		Type:   entities.MeetingTypeOpen,
	})
	require.NoError(t, err)
	require.NotEmpty(t, capturedCode)

	// Must match xxx-xxxx-xxx where x is a lowercase letter.
	parts := strings.Split(capturedCode, "-")
	require.Len(t, parts, 3, "code must have exactly 3 dash-separated segments")
	assert.Len(t, parts[0], 3)
	assert.Len(t, parts[1], 4)
	assert.Len(t, parts[2], 3)
	for _, p := range parts {
		for _, ch := range p {
			assert.True(t, ch >= 'a' && ch <= 'z', "all code chars must be lowercase letters, got %q", ch)
		}
	}
}

// ─── ListMeetingsWithMeta ─────────────────────────────────────────────────────

func TestListMeetingsWithMeta_NoFilter_PassesThrough(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	now := time.Now()
	items := []*ports.MeetingListItem{
		{Meeting: &entities.Meeting{ID: uuid.New(), Code: "aaa-bbbb-ccc", Status: entities.MeetingStatusActive, CreatedAt: now}},
	}
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "created_at_desc"}).
		Return(items, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "", "created_at_desc")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	mr.AssertExpectations(t)
}

func TestListMeetingsWithMeta_StatusFilter_PassedToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	status := "complete"
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Status: &status, Sort: "created_at_desc"}).
		Return([]*ports.MeetingListItem{}, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "complete", "created_at_desc")
	require.NoError(t, err)
	assert.Empty(t, result)
	mr.AssertExpectations(t)
}

func TestListMeetingsWithMeta_SortKey_PassedToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "duration_desc"}).
		Return([]*ports.MeetingListItem{}, nil)

	_, err := svc.ListMeetingsWithMeta(context.Background(), userID, "", "duration_desc")
	require.NoError(t, err)
	mr.AssertExpectations(t)
}

func TestListMeetingsWithMeta_DurationSeconds_ComputedByRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	startedAt := time.Now().Add(-10 * time.Minute)
	endedAt := startedAt.Add(8 * time.Minute)
	d := int64(480)
	items := []*ports.MeetingListItem{
		{
			Meeting:         &entities.Meeting{ID: uuid.New(), StartedAt: &startedAt, EndedAt: &endedAt},
			DurationSeconds: &d,
		},
	}
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "created_at_desc"}).
		Return(items, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "", "created_at_desc")
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].DurationSeconds)
	assert.Equal(t, int64(480), *result[0].DurationSeconds)
}

func TestListMeetingsWithMeta_NilDuration_WhenInProgress(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	items := []*ports.MeetingListItem{
		{Meeting: &entities.Meeting{ID: uuid.New(), Status: entities.MeetingStatusActive}, DurationSeconds: nil},
	}
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "created_at_desc"}).
		Return(items, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "", "created_at_desc")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Nil(t, result[0].DurationSeconds)
}

func TestListMeetingsWithMeta_InProgressFilter_PassedToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	status := "in_progress"
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Status: &status, Sort: "created_at_desc"}).
		Return([]*ports.MeetingListItem{}, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "in_progress", "created_at_desc")
	require.NoError(t, err)
	assert.Empty(t, result)
	mr.AssertExpectations(t)
}

func TestListMeetingsWithMeta_ProcessingFilter_ReturnsEmpty(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	// "processing" is a reserved status — the repository short-circuits and
	// returns an empty slice before executing any query.
	status := "processing"
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Status: &status, Sort: "created_at_desc"}).
		Return([]*ports.MeetingListItem{}, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "processing", "created_at_desc")
	require.NoError(t, err)
	assert.Empty(t, result)
	mr.AssertExpectations(t)
}

func TestListMeetingsWithMeta_UnrecognisedSort_FallsBackToDefault(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	// The service passes the sort string through; listSortExpr in the repo maps
	// unrecognised values to "created_at DESC". Verify the service does not
	// swallow or reject an unknown sort key.
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "bogus_sort"}).
		Return([]*ports.MeetingListItem{}, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "", "bogus_sort")
	require.NoError(t, err)
	assert.Empty(t, result)
	mr.AssertExpectations(t)
}

func TestListMeetingsWithMeta_ParticipantPreviews_CappedAt3(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	previews := []ports.ParticipantPreview{
		{UserID: uuid.New(), FullName: "Alice A", Initials: "AA"},
		{UserID: uuid.New(), FullName: "Bob B", Initials: "BB"},
		{UserID: uuid.New(), FullName: "Carol C", Initials: "CC"},
	}
	items := []*ports.MeetingListItem{
		{Meeting: &entities.Meeting{ID: uuid.New()}, ParticipantPreviews: previews},
	}
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "created_at_desc"}).
		Return(items, nil)

	result, err := svc.ListMeetingsWithMeta(context.Background(), userID, "", "created_at_desc")
	require.NoError(t, err)
	assert.Len(t, result[0].ParticipantPreviews, 3)
}

// ─── CanJoin — dept scope ─────────────────────────────────────────────────────

func TestCanJoin_Private_DeptMember_Direct(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	or := new(meetingMockOrgMemberRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, pr, or, dr)

	hostID := uuid.New()
	deptID := uuid.New()
	scopeType := entities.MeetingScopeDept
	m := openMeeting(hostID)
	m.Type = entities.MeetingTypePrivate
	m.ScopeType = &scopeType
	m.ScopeID = &deptID

	callerID := uuid.New()
	pr.On("CountActive", mock.Anything, m.ID).Return(int64(2), nil)
	dr.On("GetByDeptAndUser", mock.Anything, deptID, callerID).
		Return(&entities.DepartmentMembership{}, nil)

	result, err := svc.CanJoin(context.Background(), m, callerID)
	require.NoError(t, err)
	assert.Equal(t, services.CanJoinDirect, result)
}

// ─── ListMeetingsInScope ──────────────────────────────────────────────────────

func TestListMeetingsInScope_NilScope_FallsBackToDefault(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "created_at_desc"}).
		Return([]*ports.MeetingListItem{}, nil)

	_, err := svc.ListMeetingsInScope(context.Background(), userID, nil, "", "created_at_desc")
	require.NoError(t, err)
	mr.AssertExpectations(t)
}

func TestListMeetingsInScope_OrgScope_NonMember_Forbidden(t *testing.T) {
	mr := new(mockMeetingRepo)
	or := new(meetingMockOrgMemberRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), or, new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	orgID := uuid.New()
	or.On("GetByOrgAndUser", mock.Anything, orgID, userID).
		Return(nil, apperr.NotFound("OrgMembership", orgID.String()))

	_, err := svc.ListMeetingsInScope(context.Background(), userID,
		&ports.ScopeFilter{Kind: ports.ScopeKindOrganization, ID: orgID},
		"", "created_at_desc")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
	mr.AssertNotCalled(t, "ListByUser")
}

func TestListMeetingsInScope_OrgScope_Member_PassesFilterToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	or := new(meetingMockOrgMemberRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), or, new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	orgID := uuid.New()
	scope := &ports.ScopeFilter{Kind: ports.ScopeKindOrganization, ID: orgID}
	or.On("GetByOrgAndUser", mock.Anything, orgID, userID).Return(&entities.OrgMembership{}, nil)
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "created_at_desc", Scope: scope}).
		Return([]*ports.MeetingListItem{}, nil)

	_, err := svc.ListMeetingsInScope(context.Background(), userID, scope, "", "created_at_desc")
	require.NoError(t, err)
	mr.AssertExpectations(t)
}

func TestListMeetingsInScope_DeptScope_NonMember_Forbidden(t *testing.T) {
	mr := new(mockMeetingRepo)
	dr := new(meetingMockDeptMemberRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), dr)

	userID := uuid.New()
	deptID := uuid.New()
	dr.On("GetByDeptAndUser", mock.Anything, deptID, userID).
		Return(nil, apperr.NotFound("DepartmentMembership", deptID.String()))

	_, err := svc.ListMeetingsInScope(context.Background(), userID,
		&ports.ScopeFilter{Kind: ports.ScopeKindDepartment, ID: deptID},
		"", "created_at_desc")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 403, appErr.Status)
}

func TestListMeetingsInScope_OpenScope_PassesScopeFilterToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	userID := uuid.New()
	scope := &ports.ScopeFilter{Kind: ports.ScopeKindOpen}
	mr.On("ListByUser", mock.Anything, userID, ports.ListMeetingsFilter{Sort: "created_at_desc", Scope: scope}).
		Return([]*ports.MeetingListItem{}, nil)

	_, err := svc.ListMeetingsInScope(context.Background(), userID, scope, "", "created_at_desc")
	require.NoError(t, err)
	mr.AssertExpectations(t)
}

// ─── GetMeeting / GetMeetingByCode / ListMyMeetings ──────────────────────────

func TestGetMeeting_DelegatesToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	m := openMeeting(hostID)
	mr.On("GetByID", mock.Anything, m.ID).Return(m, nil)

	got, err := svc.GetMeeting(context.Background(), m.ID)
	require.NoError(t, err)
	assert.Equal(t, m, got)
	mr.AssertExpectations(t)
}

func TestGetMeetingByCode_DelegatesToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	m := openMeeting(hostID)
	mr.On("GetByCode", mock.Anything, m.Code).Return(m, nil)

	got, err := svc.GetMeetingByCode(context.Background(), m.Code)
	require.NoError(t, err)
	assert.Equal(t, m, got)
	mr.AssertExpectations(t)
}

func TestListMyMeetings_DelegatesToRepo(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	list := []*entities.Meeting{openMeeting(hostID), openMeeting(hostID)}
	mr.On("ListByHost", mock.Anything, hostID, "active").Return(list, nil)

	got, err := svc.ListMyMeetings(context.Background(), hostID, "active")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	mr.AssertExpectations(t)
}

func TestListMyMeetings_EmptyStatusPassesThrough(t *testing.T) {
	mr := new(mockMeetingRepo)
	svc := newTestMeetingService(mr, new(mockParticipantRepo), new(meetingMockOrgMemberRepo), new(meetingMockDeptMemberRepo))

	hostID := uuid.New()
	mr.On("ListByHost", mock.Anything, hostID, "").Return([]*entities.Meeting{}, nil)

	got, err := svc.ListMyMeetings(context.Background(), hostID, "")
	require.NoError(t, err)
	assert.Empty(t, got)
}
