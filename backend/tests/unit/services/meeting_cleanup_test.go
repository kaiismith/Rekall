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
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestCleanupJob(mr *mockMeetingRepo, pr *mockParticipantRepo) *services.MeetingCleanupJob {
	return services.NewMeetingCleanupJob(mr, pr, services.MeetingCleanupConfig{
		Interval:       time.Hour, // not used in unit tests (RunOnce bypasses the ticker)
		WaitingTimeout: 10 * time.Minute,
		MaxDuration:    8 * time.Hour,
	}, zap.NewNop())
}

func cleanupActiveMeeting() *entities.Meeting {
	now := time.Now().UTC()
	return &entities.Meeting{
		ID:        uuid.New(),
		Code:      "aaa-bbbb-ccc",
		Type:      entities.MeetingTypeOpen,
		Status:    entities.MeetingStatusActive,
		HostID:    uuid.New(),
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func cleanupWaitingMeeting() *entities.Meeting {
	m := cleanupActiveMeeting()
	m.Status = entities.MeetingStatusWaiting
	m.StartedAt = nil
	return m
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCleanupJob_EndsStaleWaitingMeetings(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	stale := []*entities.Meeting{cleanupWaitingMeeting(), cleanupWaitingMeeting()}

	mr.On("FindStaleWaiting", mock.Anything, 10*time.Minute).Return(stale, nil)
	mr.On("FindStaleActive", mock.Anything, 8*time.Hour).Return([]*entities.Meeting{}, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, mock.Anything).Return(nil)

	services.NewMeetingCleanupJob(mr, pr, services.MeetingCleanupConfig{
		Interval:       time.Hour,
		WaitingTimeout: 10 * time.Minute,
		MaxDuration:    8 * time.Hour,
	}, zap.NewNop()).RunOnce(context.Background())

	// Update + MarkAllLeft called once per stale meeting.
	mr.AssertNumberOfCalls(t, "Update", 2)
	pr.AssertNumberOfCalls(t, "MarkAllLeft", 2)
	for _, m := range stale {
		assert.Equal(t, entities.MeetingStatusEnded, m.Status)
		assert.NotNil(t, m.EndedAt)
	}
}

func TestCleanupJob_EndsStaleActiveMeetings(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	stale := []*entities.Meeting{cleanupActiveMeeting()}

	mr.On("FindStaleWaiting", mock.Anything, 10*time.Minute).Return([]*entities.Meeting{}, nil)
	mr.On("FindStaleActive", mock.Anything, 8*time.Hour).Return(stale, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, mock.Anything).Return(nil)

	newTestCleanupJob(mr, pr).RunOnce(context.Background())

	mr.AssertNumberOfCalls(t, "Update", 1)
	pr.AssertNumberOfCalls(t, "MarkAllLeft", 1)
	assert.Equal(t, entities.MeetingStatusEnded, stale[0].Status)
}

func TestCleanupJob_EndsAbandonedMeetings(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	abandoned := []*entities.Meeting{cleanupActiveMeeting()}

	mr.On("FindStaleWaiting", mock.Anything, 10*time.Minute).Return([]*entities.Meeting{}, nil)
	mr.On("FindStaleActive", mock.Anything, 8*time.Hour).Return([]*entities.Meeting{}, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return(abandoned, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, mock.Anything).Return(nil)

	newTestCleanupJob(mr, pr).RunOnce(context.Background())

	mr.AssertNumberOfCalls(t, "Update", 1)
	assert.Equal(t, entities.MeetingStatusEnded, abandoned[0].Status)
}

func TestCleanupJob_ErrorInOnePhase_OtherPhasesStillRun(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	abandoned := []*entities.Meeting{cleanupActiveMeeting()}

	// Phases 1 and 2 return DB errors.
	mr.On("FindStaleWaiting", mock.Anything, 10*time.Minute).
		Return([]*entities.Meeting{}, errors.New("db timeout"))
	mr.On("FindStaleActive", mock.Anything, 8*time.Hour).
		Return([]*entities.Meeting{}, errors.New("db timeout"))
	// Phase 3 succeeds.
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return(abandoned, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, mock.Anything).Return(nil)

	require.NotPanics(t, func() {
		newTestCleanupJob(mr, pr).RunOnce(context.Background())
	})
	// Phase 3 still ran despite phases 1 and 2 failing.
	mr.AssertNumberOfCalls(t, "Update", 1)
}

func TestCleanupJob_AbandonedFindError(t *testing.T) {
	// Error in FindActiveWithNoParticipants — phase 3 errors don't crash.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	mr.On("FindStaleWaiting", mock.Anything, mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("FindStaleActive", mock.Anything, mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return([]*entities.Meeting{}, errors.New("db failure"))

	require.NotPanics(t, func() {
		newTestCleanupJob(mr, pr).RunOnce(context.Background())
	})
	mr.AssertNotCalled(t, "Update")
}

func TestCleanupJob_AutoEndUpdateError(t *testing.T) {
	// Covers autoEnd's meetingRepo.Update error branch.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	stale := []*entities.Meeting{cleanupWaitingMeeting()}
	mr.On("FindStaleWaiting", mock.Anything, mock.Anything).Return(stale, nil)
	mr.On("FindStaleActive", mock.Anything, mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).
		Return(errors.New("update failed"))

	// Should not panic; autoEnd returns err and cleanup continues.
	require.NotPanics(t, func() {
		newTestCleanupJob(mr, pr).RunOnce(context.Background())
	})
	// MarkAllLeft never called because Update failed first.
	pr.AssertNotCalled(t, "MarkAllLeft")
}

func TestCleanupJob_AutoEndMarkAllLeftError(t *testing.T) {
	// Covers autoEnd's participantRepo.MarkAllLeft error branch.
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	stale := []*entities.Meeting{cleanupWaitingMeeting()}
	mr.On("FindStaleWaiting", mock.Anything, mock.Anything).Return(stale, nil)
	mr.On("FindStaleActive", mock.Anything, mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, mock.Anything).Return(errors.New("mark failed"))

	require.NotPanics(t, func() {
		newTestCleanupJob(mr, pr).RunOnce(context.Background())
	})
	mr.AssertNumberOfCalls(t, "Update", 1)
	pr.AssertNumberOfCalls(t, "MarkAllLeft", 1)
}

func TestCleanupJob_NothingStale_NoUpdates(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	mr.On("FindStaleWaiting", mock.Anything, 10*time.Minute).Return([]*entities.Meeting{}, nil)
	mr.On("FindStaleActive", mock.Anything, 8*time.Hour).Return([]*entities.Meeting{}, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return([]*entities.Meeting{}, nil)

	newTestCleanupJob(mr, pr).RunOnce(context.Background())

	mr.AssertNotCalled(t, "Update")
	pr.AssertNotCalled(t, "MarkAllLeft")
}

// ─── Run (ticker loop) ────────────────────────────────────────────────────────

func TestCleanupJob_Run_FiresTicksAndExitsOnContextCancel(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	// All three Find methods are called every tick; return empty slices.
	mr.On("FindStaleWaiting", mock.Anything, mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("FindStaleActive", mock.Anything, mock.Anything).Return([]*entities.Meeting{}, nil)
	mr.On("FindActiveWithNoParticipants", mock.Anything).Return([]*entities.Meeting{}, nil)

	// Short interval so we actually see a tick fire before cancelling.
	job := services.NewMeetingCleanupJob(mr, pr, services.MeetingCleanupConfig{
		Interval:       20 * time.Millisecond,
		WaitingTimeout: 10 * time.Minute,
		MaxDuration:    8 * time.Hour,
	}, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		job.Run(ctx)
		close(done)
	}()

	// Let the ticker fire at least once.
	time.Sleep(60 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not exit after context cancellation")
	}

	// Verify the tick actually executed (Find methods were called).
	mr.AssertCalled(t, "FindStaleWaiting", mock.Anything, mock.Anything)
}

func TestCleanupJob_Run_ExitsImmediatelyIfContextAlreadyCancelled(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	job := services.NewMeetingCleanupJob(mr, pr, services.MeetingCleanupConfig{
		Interval:       time.Hour,
		WaitingTimeout: 10 * time.Minute,
		MaxDuration:    8 * time.Hour,
	}, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Run starts

	done := make(chan struct{})
	go func() {
		job.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not exit immediately when ctx was already cancelled")
	}
}
