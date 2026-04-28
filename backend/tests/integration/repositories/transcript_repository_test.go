package repositories_test

// Integration tests for the transcript persistence repository.
// Require a running PostgreSQL instance with the spec's migrations applied.
//
// Run with: make test-integration  (starts docker-compose test services)
// Or pass TEST_DATABASE_URL directly:
//   TEST_DATABASE_URL="host=localhost user=rekall password=rekall_secret dbname=rekall_test port=5432 sslmode=disable" \
//   go test ./tests/integration/...

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
	"github.com/rekall/backend/internal/infrastructure/repositories"
)

func newTranscriptUser(t *testing.T, ctx context.Context, suffix string) *entities.User {
	t.Helper()
	db := testDB(t)
	userRepo := repositories.NewUserRepository(db)
	userSvc := services.NewUserService(userRepo, zap.NewNop())
	user, err := userSvc.CreateUser(ctx, "transcript-"+suffix+"@rekall.io", "Transcript Test "+suffix, "member")
	require.NoError(t, err)
	t.Cleanup(func() { _ = userRepo.SoftDelete(ctx, user.ID) })
	return user
}

func newTranscriptCall(t *testing.T, ctx context.Context, userID uuid.UUID) *entities.Call {
	t.Helper()
	db := testDB(t)
	callRepo := repositories.NewCallRepository(db)
	callSvc := services.NewCallService(callRepo, nil, nil, zap.NewNop())
	call, err := callSvc.CreateCall(ctx, services.CreateCallInput{
		UserID: userID,
		Title:  "Transcript Test Call",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = callRepo.SoftDelete(ctx, call.ID) })
	return call
}

func newTranscriptSession(callID, speaker uuid.UUID) *entities.TranscriptSession {
	now := time.Now().UTC()
	return &entities.TranscriptSession{
		ID:            uuid.New(),
		SpeakerUserID: speaker,
		CallID:        &callID,
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

func TestTranscriptRepository_CreateAndGetSession(t *testing.T) {
	db := testDB(t)
	repo := repositories.NewTranscriptRepository(db)
	ctx := context.Background()

	user := newTranscriptUser(t, ctx, "session1")
	call := newTranscriptCall(t, ctx, user.ID)
	sess := newTranscriptSession(call.ID, user.ID)

	require.NoError(t, repo.CreateSession(ctx, sess))

	got, err := repo.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)
	assert.Equal(t, user.ID, got.SpeakerUserID)
	assert.Equal(t, call.ID, *got.CallID)
	assert.Nil(t, got.MeetingID)
	assert.Equal(t, entities.TranscriptEngineModeOpenAI, got.EngineMode)
	assert.Equal(t, "whisper-1", got.ModelID)
	assert.Equal(t, entities.TranscriptSessionStatusActive, got.Status)
}

func TestTranscriptRepository_UpsertSegmentIdempotent(t *testing.T) {
	db := testDB(t)
	repo := repositories.NewTranscriptRepository(db)
	ctx := context.Background()

	user := newTranscriptUser(t, ctx, "upsert")
	call := newTranscriptCall(t, ctx, user.ID)
	sess := newTranscriptSession(call.ID, user.ID)
	require.NoError(t, repo.CreateSession(ctx, sess))

	// First insert: counter -> 1, audio_seconds_total -> 1.500
	conf := float32(0.91)
	lang := "en"
	seg := &entities.TranscriptSegment{
		SessionID:        sess.ID,
		SegmentIndex:     0,
		SpeakerUserID:    user.ID,
		CallID:           &call.ID,
		Text:             "hello world",
		Language:         &lang,
		Confidence:       &conf,
		StartMs:          0,
		EndMs:            1500,
		Words:            entities.WordTimings{{Word: "hello", StartMs: 0, EndMs: 700, Probability: 0.93}},
		EngineMode:       sess.EngineMode,
		ModelID:          sess.ModelID,
		SegmentStartedAt: sess.StartedAt,
	}
	require.NoError(t, repo.UpsertSegment(ctx, seg))

	got, err := repo.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, got.FinalizedSegmentCount)
	assert.InDelta(t, 1.500, got.AudioSecondsTotal, 0.001)

	// Second insert with the same (session_id, segment_index) but different text:
	// counter must NOT advance, but text must be updated.
	seg.Text = "hello world updated"
	require.NoError(t, repo.UpsertSegment(ctx, seg))

	got, err = repo.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, got.FinalizedSegmentCount, "counter must not double-count duplicates")
	assert.InDelta(t, 1.500, got.AudioSecondsTotal, 0.001)

	segs, err := repo.ListSegmentsBySession(ctx, sess.ID)
	require.NoError(t, err)
	require.Len(t, segs, 1)
	assert.Equal(t, "hello world updated", segs[0].Text)

	// Third insert with a NEW segment_index: counter -> 2.
	seg2 := *seg
	seg2.ID = uuid.Nil
	seg2.SegmentIndex = 1
	seg2.Text = "second segment"
	seg2.StartMs = 1500
	seg2.EndMs = 3500
	require.NoError(t, repo.UpsertSegment(ctx, &seg2))

	got, err = repo.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, got.FinalizedSegmentCount)
	assert.InDelta(t, 3.500, got.AudioSecondsTotal, 0.001)
}

func TestTranscriptRepository_UpdateSessionStatus(t *testing.T) {
	db := testDB(t)
	repo := repositories.NewTranscriptRepository(db)
	ctx := context.Background()

	user := newTranscriptUser(t, ctx, "status")
	call := newTranscriptCall(t, ctx, user.ID)
	sess := newTranscriptSession(call.ID, user.ID)
	require.NoError(t, repo.CreateSession(ctx, sess))

	require.NoError(t, repo.UpdateSessionStatus(ctx, sess.ID, entities.TranscriptSessionStatusEnded, nil, nil))

	got, err := repo.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.TranscriptSessionStatusEnded, got.Status)
	require.NotNil(t, got.EndedAt)
}

func TestTranscriptRepository_StitchCall(t *testing.T) {
	db := testDB(t)
	repo := repositories.NewTranscriptRepository(db)
	ctx := context.Background()

	user := newTranscriptUser(t, ctx, "stitch")
	call := newTranscriptCall(t, ctx, user.ID)
	sess := newTranscriptSession(call.ID, user.ID)
	require.NoError(t, repo.CreateSession(ctx, sess))

	for i, text := range []string{"first", "second", "third"} {
		require.NoError(t, repo.UpsertSegment(ctx, &entities.TranscriptSegment{
			SessionID:        sess.ID,
			SegmentIndex:     int32(i),
			SpeakerUserID:    user.ID,
			CallID:           &call.ID,
			Text:             text,
			StartMs:          int32(i) * 1000,
			EndMs:            int32(i+1) * 1000,
			EngineMode:       sess.EngineMode,
			ModelID:          sess.ModelID,
			SegmentStartedAt: sess.StartedAt.Add(time.Duration(i) * time.Second),
		}))
	}

	stitched, err := repo.StitchCall(ctx, call.ID)
	require.NoError(t, err)
	assert.Equal(t, "first second third", stitched)
}

func TestTranscriptRepository_FindExpiredActive(t *testing.T) {
	db := testDB(t)
	repo := repositories.NewTranscriptRepository(db)
	ctx := context.Background()

	user := newTranscriptUser(t, ctx, "expired")
	call := newTranscriptCall(t, ctx, user.ID)

	expired := newTranscriptSession(call.ID, user.ID)
	expired.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)
	require.NoError(t, repo.CreateSession(ctx, expired))

	fresh := newTranscriptSession(call.ID, user.ID)
	fresh.ExpiresAt = time.Now().UTC().Add(15 * time.Minute)
	require.NoError(t, repo.CreateSession(ctx, fresh))

	rows, err := repo.FindExpiredActive(ctx, 100)
	require.NoError(t, err)

	var foundExpired, foundFresh bool
	for _, r := range rows {
		if r.ID == expired.ID {
			foundExpired = true
		}
		if r.ID == fresh.ID {
			foundFresh = true
		}
	}
	assert.True(t, foundExpired, "expired session should be returned")
	assert.False(t, foundFresh, "fresh session should NOT be returned")
}

func TestTranscriptRepository_CallXorMeetingCheckRejects(t *testing.T) {
	db := testDB(t)
	repo := repositories.NewTranscriptRepository(db)
	ctx := context.Background()

	user := newTranscriptUser(t, ctx, "checkfail")
	call := newTranscriptCall(t, ctx, user.ID)

	bogusMeetingID := uuid.New()
	sess := newTranscriptSession(call.ID, user.ID)
	sess.MeetingID = &bogusMeetingID // both call_id AND meeting_id set → CHECK should reject

	err := repo.CreateSession(ctx, sess)
	assert.Error(t, err, "CHECK constraint must reject session with both call_id and meeting_id set")
}
