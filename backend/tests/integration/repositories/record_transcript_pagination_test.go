package repositories_test

// Integration test for the records/transcripts paginated read.
// Drives the TranscriptRepository pagination + speaker resolution against a
// real Postgres instance. Skipped automatically when TEST_DATABASE_URL is unset.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/infrastructure/repositories"
)

func newTranscriptMeeting(t *testing.T, ctx context.Context, hostID uuid.UUID) *entities.Meeting {
	t.Helper()
	db := testDB(t)
	meetingRepo := repositories.NewMeetingRepository(db)
	now := time.Now().UTC()
	m := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "recpag-" + uuid.New().String()[:8],
		Title:           "Records Pagination Meeting",
		Type:            entities.MeetingTypeOpen,
		HostID:          hostID,
		Status:          entities.MeetingStatusEnded,
		MaxParticipants: 10,
		StartedAt:       timePtr(now.Add(-1 * time.Hour)),
		EndedAt:         timePtr(now.Add(-30 * time.Minute)),
	}
	require.NoError(t, meetingRepo.Create(ctx, m))
	t.Cleanup(func() { _ = meetingRepo.Delete(ctx, m.ID) })
	return m
}

// TestTranscriptRepository_ListSegmentsByMeeting_Paginates seeds 10 segments
// across two participants and walks them in pages of 4, asserting unique
// segments, ordering, total/has_more, and pagination boundaries.
func TestTranscriptRepository_ListSegmentsByMeeting_Paginates(t *testing.T) {
	db := testDB(t)
	transcriptRepo := repositories.NewTranscriptRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hostA := createTestUser(t, ctx, userRepo, "recpag-a@rekall.io", "Alice Anderson")
	hostB := createTestUser(t, ctx, userRepo, "recpag-b@rekall.io", "Bob Brown")
	meeting := newTranscriptMeeting(t, ctx, hostA.ID)

	// One session per speaker.
	now := time.Now().UTC()
	mkSession := func(speaker uuid.UUID) *entities.TranscriptSession {
		return &entities.TranscriptSession{
			ID:            uuid.New(),
			SpeakerUserID: speaker,
			MeetingID:     &meeting.ID,
			EngineMode:    entities.TranscriptEngineModeOpenAI,
			EngineTarget:  "https://api.openai.com/v1/",
			ModelID:       "whisper-1",
			SampleRate:    16000,
			FrameFormat:   "pcm_s16le_mono",
			Status:        entities.TranscriptSessionStatusActive,
			StartedAt:     now,
			ExpiresAt:     now.Add(15 * time.Minute),
		}
	}
	sessA := mkSession(hostA.ID)
	sessB := mkSession(hostB.ID)
	require.NoError(t, transcriptRepo.CreateSession(ctx, sessA))
	require.NoError(t, transcriptRepo.CreateSession(ctx, sessB))

	// 10 segments alternating speakers, strictly-increasing segment_started_at.
	for i := 0; i < 10; i++ {
		sess := sessA
		if i%2 == 1 {
			sess = sessB
		}
		seg := &entities.TranscriptSegment{
			SessionID:        sess.ID,
			SegmentIndex:     int32(i),
			SpeakerUserID:    sess.SpeakerUserID,
			MeetingID:        &meeting.ID,
			Text:             "segment text",
			StartMs:          int32(i * 1000),
			EndMs:            int32(i*1000 + 500),
			EngineMode:       entities.TranscriptEngineModeOpenAI,
			ModelID:          "whisper-1",
			SegmentStartedAt: now.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, transcriptRepo.UpsertSegment(ctx, seg))
	}

	// Walk page-by-page. perPage=4, total=10 → pages 1, 2, 3 with sizes 4, 4, 2.
	seen := map[uuid.UUID]bool{}
	var allOrdered []*entities.TranscriptSegment

	page1, total1, err := transcriptRepo.ListSegmentsByMeeting(ctx, meeting.ID, 1, 4)
	require.NoError(t, err)
	assert.Equal(t, 10, total1)
	require.Len(t, page1, 4)

	page2, total2, err := transcriptRepo.ListSegmentsByMeeting(ctx, meeting.ID, 2, 4)
	require.NoError(t, err)
	assert.Equal(t, 10, total2)
	require.Len(t, page2, 4)

	page3, total3, err := transcriptRepo.ListSegmentsByMeeting(ctx, meeting.ID, 3, 4)
	require.NoError(t, err)
	assert.Equal(t, 10, total3)
	require.Len(t, page3, 2, "last page returns the tail")

	for _, seg := range append(append(page1, page2...), page3...) {
		assert.False(t, seen[seg.ID], "segment %s appears across multiple pages", seg.ID)
		seen[seg.ID] = true
		allOrdered = append(allOrdered, seg)
	}

	// Ordering across all pages is non-decreasing on (segment_started_at, segment_index).
	for i := 1; i < len(allOrdered); i++ {
		prev, cur := allOrdered[i-1], allOrdered[i]
		assert.True(
			t,
			prev.SegmentStartedAt.Before(cur.SegmentStartedAt) ||
				(prev.SegmentStartedAt.Equal(cur.SegmentStartedAt) && prev.SegmentIndex <= cur.SegmentIndex),
			"segments must be ordered by (segment_started_at, segment_index)",
		)
	}
	assert.Len(t, allOrdered, 10, "every segment retrieved exactly once across pages")
}

// TestTranscriptRepository_ListSpeakerUserIDsByMeeting_Distinct seeds 2
// sessions per speaker and asserts the helper returns exactly 2 ids.
func TestTranscriptRepository_ListSpeakerUserIDsByMeeting_Distinct(t *testing.T) {
	db := testDB(t)
	transcriptRepo := repositories.NewTranscriptRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hostA := createTestUser(t, ctx, userRepo, "recpag-spk-a@rekall.io", "Alice Anderson")
	hostB := createTestUser(t, ctx, userRepo, "recpag-spk-b@rekall.io", "Bob Brown")
	meeting := newTranscriptMeeting(t, ctx, hostA.ID)

	now := time.Now().UTC()
	mkSession := func(speaker uuid.UUID) *entities.TranscriptSession {
		return &entities.TranscriptSession{
			ID:            uuid.New(),
			SpeakerUserID: speaker,
			MeetingID:     &meeting.ID,
			EngineMode:    entities.TranscriptEngineModeOpenAI,
			EngineTarget:  "https://api.openai.com/v1/",
			ModelID:       "whisper-1",
			SampleRate:    16000,
			FrameFormat:   "pcm_s16le_mono",
			Status:        entities.TranscriptSessionStatusEnded,
			StartedAt:     now,
			ExpiresAt:     now.Add(15 * time.Minute),
		}
	}
	require.NoError(t, transcriptRepo.CreateSession(ctx, mkSession(hostA.ID)))
	require.NoError(t, transcriptRepo.CreateSession(ctx, mkSession(hostA.ID))) // duplicate speaker → still one
	require.NoError(t, transcriptRepo.CreateSession(ctx, mkSession(hostB.ID)))

	ids, err := transcriptRepo.ListSpeakerUserIDsByMeeting(ctx, meeting.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 2, "distinct speakers")
	assert.Contains(t, ids, hostA.ID)
	assert.Contains(t, ids, hostB.ID)
}

// TestTranscriptRepository_ListSpeakerUserIDsByMeeting_EmptyForNoSessions
// confirms the helper returns an empty (non-nil) slice when no sessions exist.
func TestTranscriptRepository_ListSpeakerUserIDsByMeeting_EmptyForNoSessions(t *testing.T) {
	db := testDB(t)
	transcriptRepo := repositories.NewTranscriptRepository(db)
	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	host := createTestUser(t, ctx, userRepo, "recpag-empty@rekall.io", "Empty Host")
	meeting := newTranscriptMeeting(t, ctx, host.ID)

	ids, err := transcriptRepo.ListSpeakerUserIDsByMeeting(ctx, meeting.ID)
	require.NoError(t, err)
	require.NotNil(t, ids)
	assert.Empty(t, ids)
}
