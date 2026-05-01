package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/repositories"
)

// createTestUser is a helper that creates a user and registers a cleanup to delete it.
func createTestUser(t *testing.T, ctx context.Context, userRepo *repositories.UserRepository, email, name string) *entities.User {
	t.Helper()
	userSvc := services.NewUserService(userRepo, zap.NewNop())
	user, err := userSvc.CreateUser(ctx, email, name, "member")
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, user.ID) })
	return user
}

// createTestMeeting inserts a meeting directly and registers cleanup.
func createTestMeeting(t *testing.T, ctx context.Context, meetingRepo *repositories.MeetingRepository, m *entities.Meeting) *entities.Meeting {
	t.Helper()
	require.NoError(t, meetingRepo.Create(ctx, m))
	t.Cleanup(func() { _ = meetingRepo.Delete(ctx, m.ID) })
	return m
}

// ptr helpers
func strPtr(s string) *string        { return &s }
func timePtr(t time.Time) *time.Time { return &t }

// TestMeetingRepository_ListByUser_StatusFilter verifies that passing
// filter[status]=complete returns only meetings with status "ended".
func TestMeetingRepository_ListByUser_StatusFilter(t *testing.T) {
	db := testDB(t)
	meetingRepo := repositories.NewMeetingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	host := createTestUser(t, ctx, userRepo, "filter-host@rekall.io", "Filter Host")

	now := time.Now().UTC()
	endedMeeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "filter-ended-" + uuid.New().String()[:8],
		Title:           "Ended Meeting",
		Type:            entities.MeetingTypeOpen,
		HostID:          host.ID,
		Status:          entities.MeetingStatusEnded,
		MaxParticipants: 50,
		StartedAt:       timePtr(now.Add(-1 * time.Hour)),
		EndedAt:         timePtr(now.Add(-30 * time.Minute)),
	}
	createTestMeeting(t, ctx, meetingRepo, endedMeeting)

	activeMeeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "filter-active-" + uuid.New().String()[:8],
		Title:           "Active Meeting",
		Type:            entities.MeetingTypeOpen,
		HostID:          host.ID,
		Status:          entities.MeetingStatusActive,
		MaxParticipants: 50,
		StartedAt:       timePtr(now.Add(-10 * time.Minute)),
	}
	createTestMeeting(t, ctx, meetingRepo, activeMeeting)

	filter := ports.ListMeetingsFilter{Status: strPtr("complete")}
	items, _, err := meetingRepo.ListByUser(ctx, host.ID, filter)
	require.NoError(t, err)

	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = item.Meeting.ID
	}

	assert.Contains(t, ids, endedMeeting.ID, "ended meeting should be returned")
	for _, item := range items {
		assert.Equal(t, entities.MeetingStatusEnded, item.Meeting.Status,
			"only ended meetings should be returned for status=complete")
	}
	assert.NotContains(t, ids, activeMeeting.ID, "active meeting should not be returned for status=complete")
}

// TestMeetingRepository_ListByUser_SortByDuration verifies that sort=duration_desc
// returns ended meetings longest-first, with meetings that have no duration last.
func TestMeetingRepository_ListByUser_SortByDuration(t *testing.T) {
	db := testDB(t)
	meetingRepo := repositories.NewMeetingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	host := createTestUser(t, ctx, userRepo, "sort-host@rekall.io", "Sort Host")

	now := time.Now().UTC()

	// Short meeting: 10 minutes.
	shortMeeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "sort-short-" + uuid.New().String()[:8],
		Title:           "Short Meeting",
		Type:            entities.MeetingTypeOpen,
		HostID:          host.ID,
		Status:          entities.MeetingStatusEnded,
		MaxParticipants: 50,
		StartedAt:       timePtr(now.Add(-2 * time.Hour)),
		EndedAt:         timePtr(now.Add(-2*time.Hour + 10*time.Minute)),
	}
	createTestMeeting(t, ctx, meetingRepo, shortMeeting)

	// Long meeting: 45 minutes.
	longMeeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "sort-long-" + uuid.New().String()[:8],
		Title:           "Long Meeting",
		Type:            entities.MeetingTypeOpen,
		HostID:          host.ID,
		Status:          entities.MeetingStatusEnded,
		MaxParticipants: 50,
		StartedAt:       timePtr(now.Add(-3 * time.Hour)),
		EndedAt:         timePtr(now.Add(-3*time.Hour + 45*time.Minute)),
	}
	createTestMeeting(t, ctx, meetingRepo, longMeeting)

	// Waiting meeting (no duration — should sort NULLS LAST).
	waitingMeeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "sort-waiting-" + uuid.New().String()[:8],
		Title:           "Waiting Meeting",
		Type:            entities.MeetingTypeOpen,
		HostID:          host.ID,
		Status:          entities.MeetingStatusWaiting,
		MaxParticipants: 50,
	}
	createTestMeeting(t, ctx, meetingRepo, waitingMeeting)

	filter := ports.ListMeetingsFilter{Sort: "duration_desc"}
	items, _, err := meetingRepo.ListByUser(ctx, host.ID, filter)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(items), 3, "expected at least 3 meetings")

	// Find positions of our test meetings in the result.
	posOf := func(id uuid.UUID) int {
		for i, item := range items {
			if item.Meeting.ID == id {
				return i
			}
		}
		return -1
	}

	posLong := posOf(longMeeting.ID)
	posShort := posOf(shortMeeting.ID)
	posWaiting := posOf(waitingMeeting.ID)

	require.NotEqual(t, -1, posLong, "long meeting not found in results")
	require.NotEqual(t, -1, posShort, "short meeting not found in results")
	require.NotEqual(t, -1, posWaiting, "waiting meeting not found in results")

	assert.Less(t, posLong, posShort, "long meeting should come before short meeting (duration_desc)")
	assert.Less(t, posShort, posWaiting, "meetings with no duration should be sorted NULLS LAST")
}

// TestMeetingRepository_ListByUser_ParticipantPreviewsCappedAt3 verifies that even
// when a meeting has more than 3 participants, only 3 previews are returned.
func TestMeetingRepository_ListByUser_ParticipantPreviewsCappedAt3(t *testing.T) {
	db := testDB(t)
	meetingRepo := repositories.NewMeetingRepository(db)
	participantRepo := repositories.NewMeetingParticipantRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	host := createTestUser(t, ctx, userRepo, "previews-host@rekall.io", "Previews Host")

	now := time.Now().UTC()
	meeting := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "previews-meeting-" + uuid.New().String()[:8],
		Title:           "Big Meeting",
		Type:            entities.MeetingTypeOpen,
		HostID:          host.ID,
		Status:          entities.MeetingStatusEnded,
		MaxParticipants: 50,
		StartedAt:       timePtr(now.Add(-1 * time.Hour)),
		EndedAt:         timePtr(now.Add(-30 * time.Minute)),
	}
	createTestMeeting(t, ctx, meetingRepo, meeting)

	// Create 4 participants with staggered join times so ordering is deterministic.
	for i := 0; i < 4; i++ {
		suffix := uuid.New().String()[:8]
		u := createTestUser(t, ctx, userRepo,
			"participant-"+suffix+"@rekall.io",
			"Participant "+suffix,
		)
		joinedAt := now.Add(time.Duration(i) * time.Minute)
		p := &entities.MeetingParticipant{
			ID:        uuid.New(),
			MeetingID: meeting.ID,
			UserID:    u.ID,
			Role:      entities.ParticipantRoleParticipant,
			JoinedAt:  timePtr(joinedAt),
		}
		require.NoError(t, participantRepo.Create(ctx, p))
		pID := p.ID // capture loop variable
		t.Cleanup(func() {
			db.WithContext(ctx).Delete(&entities.MeetingParticipant{}, "id = ?", pID)
		})
	}

	filter := ports.ListMeetingsFilter{}
	items, _, err := meetingRepo.ListByUser(ctx, host.ID, filter)
	require.NoError(t, err)

	var found *ports.MeetingListItem
	for _, item := range items {
		if item.Meeting.ID == meeting.ID {
			found = item
			break
		}
	}
	require.NotNil(t, found, "meeting not found in ListByUser results")
	assert.LessOrEqual(t, len(found.ParticipantPreviews), 3,
		"participant_previews must be capped at 3")
	assert.Equal(t, 3, len(found.ParticipantPreviews),
		"expected exactly 3 participant previews for a 4-participant meeting")
}
