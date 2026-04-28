package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── mockMessageRepo ──────────────────────────────────────────────────────────

type mockMessageRepo struct{ mock.Mock }

func (m *mockMessageRepo) Create(ctx context.Context, msg *entities.MeetingMessage) error {
	return m.Called(ctx, msg).Error(0)
}

func (m *mockMessageRepo) ListByMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	before *time.Time,
	limit int,
) ([]*entities.MeetingMessage, bool, error) {
	args := m.Called(ctx, meetingID, before, limit)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]*entities.MeetingMessage), args.Bool(1), args.Error(2)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newChatService(mr *mockMeetingRepo, pr *mockParticipantRepo, msgRepo *mockMessageRepo) *services.ChatMessageService {
	return services.NewChatMessageService(mr, pr, msgRepo, zap.NewNop())
}

func joinedParticipant(meetingID, userID uuid.UUID) *entities.MeetingParticipant {
	now := time.Now().UTC()
	return &entities.MeetingParticipant{
		ID:        uuid.New(),
		MeetingID: meetingID,
		UserID:    userID,
		Role:      entities.ParticipantRoleParticipant,
		JoinedAt:  &now,
	}
}

func formerParticipant(meetingID, userID uuid.UUID) *entities.MeetingParticipant {
	joined := time.Now().Add(-time.Hour).UTC()
	left := time.Now().UTC()
	return &entities.MeetingParticipant{
		ID:        uuid.New(),
		MeetingID: meetingID,
		UserID:    userID,
		Role:      entities.ParticipantRoleParticipant,
		JoinedAt:  &joined,
		LeftAt:    &left,
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestChatService_ListHistory_HostAllowed(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	msgRepo := new(mockMessageRepo)

	hostID := uuid.New()
	meeting := &entities.Meeting{ID: uuid.New(), Code: "abc-xyz-pqr", HostID: hostID}
	mr.On("GetByCode", mock.Anything, meeting.Code).Return(meeting, nil)
	expected := []*entities.MeetingMessage{{ID: uuid.New(), MeetingID: meeting.ID, Body: "hi"}}
	msgRepo.On("ListByMeeting", mock.Anything, meeting.ID, (*time.Time)(nil), 50).
		Return(expected, false, nil)

	svc := newChatService(mr, pr, msgRepo)
	got, hasMore, err := svc.ListHistory(context.Background(), services.ListHistoryInput{
		MeetingCode: meeting.Code,
		RequesterID: hostID,
		Limit:       50,
	})

	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Len(t, got, 1)
	// Participant repo must NOT have been consulted — the host shortcut applies.
	pr.AssertNotCalled(t, "GetByMeetingAndUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestChatService_ListHistory_ActiveParticipantAllowed(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	msgRepo := new(mockMessageRepo)

	hostID, userID := uuid.New(), uuid.New()
	meeting := &entities.Meeting{ID: uuid.New(), Code: "abc-xyz-pqr", HostID: hostID}
	mr.On("GetByCode", mock.Anything, meeting.Code).Return(meeting, nil)
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(joinedParticipant(meeting.ID, userID), nil)
	msgRepo.On("ListByMeeting", mock.Anything, meeting.ID, (*time.Time)(nil), 50).
		Return([]*entities.MeetingMessage{}, false, nil)

	svc := newChatService(mr, pr, msgRepo)
	_, _, err := svc.ListHistory(context.Background(), services.ListHistoryInput{
		MeetingCode: meeting.Code, RequesterID: userID, Limit: 50,
	})

	require.NoError(t, err)
}

func TestChatService_ListHistory_FormerParticipantAllowed(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	msgRepo := new(mockMessageRepo)

	hostID, userID := uuid.New(), uuid.New()
	meeting := &entities.Meeting{ID: uuid.New(), Code: "abc-xyz-pqr", HostID: hostID}
	mr.On("GetByCode", mock.Anything, meeting.Code).Return(meeting, nil)
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(formerParticipant(meeting.ID, userID), nil)
	msgRepo.On("ListByMeeting", mock.Anything, meeting.ID, (*time.Time)(nil), 50).
		Return([]*entities.MeetingMessage{}, false, nil)

	svc := newChatService(mr, pr, msgRepo)
	_, _, err := svc.ListHistory(context.Background(), services.ListHistoryInput{
		MeetingCode: meeting.Code, RequesterID: userID, Limit: 50,
	})

	require.NoError(t, err)
}

func TestChatService_ListHistory_NonParticipantForbidden(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	msgRepo := new(mockMessageRepo)

	hostID, userID := uuid.New(), uuid.New()
	meeting := &entities.Meeting{ID: uuid.New(), Code: "abc-xyz-pqr", HostID: hostID}
	mr.On("GetByCode", mock.Anything, meeting.Code).Return(meeting, nil)
	// Repo returns NotFound — user never participated.
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(nil, apperr.NotFound("MeetingParticipant", meeting.ID.String()))

	svc := newChatService(mr, pr, msgRepo)
	_, _, err := svc.ListHistory(context.Background(), services.ListHistoryInput{
		MeetingCode: meeting.Code, RequesterID: userID, Limit: 50,
	})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "NOT_A_PARTICIPANT", appErr.Code)

	msgRepo.AssertNotCalled(t, "ListByMeeting", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestChatService_ListHistory_MeetingNotFound(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	msgRepo := new(mockMessageRepo)

	mr.On("GetByCode", mock.Anything, "missing").Return(nil, apperr.NotFound("Meeting", "missing"))

	svc := newChatService(mr, pr, msgRepo)
	_, _, err := svc.ListHistory(context.Background(), services.ListHistoryInput{
		MeetingCode: "missing", RequesterID: uuid.New(), Limit: 50,
	})

	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestChatService_ListHistory_ParticipantRepoInternalError(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	msgRepo := new(mockMessageRepo)

	hostID, userID := uuid.New(), uuid.New()
	meeting := &entities.Meeting{ID: uuid.New(), Code: "abc-xyz-pqr", HostID: hostID}
	mr.On("GetByCode", mock.Anything, meeting.Code).Return(meeting, nil)
	pr.On("GetByMeetingAndUser", mock.Anything, meeting.ID, userID).
		Return(nil, errors.New("db down"))

	svc := newChatService(mr, pr, msgRepo)
	_, _, err := svc.ListHistory(context.Background(), services.ListHistoryInput{
		MeetingCode: meeting.Code, RequesterID: userID, Limit: 50,
	})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", appErr.Code)
}

func TestChatService_ListHistory_ForwardsBeforeAndLimit(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	msgRepo := new(mockMessageRepo)

	hostID := uuid.New()
	meeting := &entities.Meeting{ID: uuid.New(), Code: "abc-xyz-pqr", HostID: hostID}
	before := time.Now().Add(-time.Hour).UTC()

	mr.On("GetByCode", mock.Anything, meeting.Code).Return(meeting, nil)
	msgRepo.On("ListByMeeting", mock.Anything, meeting.ID, &before, 17).
		Return([]*entities.MeetingMessage{}, true, nil)

	svc := newChatService(mr, pr, msgRepo)
	_, hasMore, err := svc.ListHistory(context.Background(), services.ListHistoryInput{
		MeetingCode: meeting.Code, RequesterID: hostID, Before: &before, Limit: 17,
	})

	require.NoError(t, err)
	assert.True(t, hasMore)
	msgRepo.AssertExpectations(t)
}
