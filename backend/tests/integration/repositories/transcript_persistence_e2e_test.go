package repositories_test

// End-to-end persistence flow for the transcript-persistence spec.
// Exercises the persister + repo + cleanup job against a real PostgreSQL.
// Skipped automatically when TEST_DATABASE_URL is not set.

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

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newPersistenceFixture(t *testing.T) (
	*repositories.TranscriptRepository,
	*repositories.CallRepository,
	*repositories.MeetingRepository,
	*services.TranscriptPersister,
) {
	t.Helper()
	db := testDB(t)
	tr := repositories.NewTranscriptRepository(db)
	cr := repositories.NewCallRepository(db)
	mr := repositories.NewMeetingRepository(db)
	p := services.NewTranscriptPersister(tr, cr, mr, zap.NewNop())
	return tr, cr, mr, p
}

func makeCallForTranscript(t *testing.T, ctx context.Context, owner uuid.UUID) *entities.Call {
	t.Helper()
	db := testDB(t)
	cr := repositories.NewCallRepository(db)
	cs := services.NewCallService(cr, nil, nil, zap.NewNop())
	c, err := cs.CreateCall(ctx, services.CreateCallInput{
		UserID: owner,
		Title:  "E2E Transcript Call",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cr.SoftDelete(ctx, c.ID) })
	return c
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestTranscriptPersistence_CallE2E walks the full Call lifecycle through the
// real persister + Postgres repo: open → 5 finals → close with stitch.
func TestTranscriptPersistence_CallE2E(t *testing.T) {
	tr, cr, _, persister := newPersistenceFixture(t)
	ctx := context.Background()

	user := createTestUser(t, ctx, repositories.NewUserRepository(testDB(t)), "transcript-e2e@rekall.io", "Transcript E2E")
	call := makeCallForTranscript(t, ctx, user.ID)

	sessionID := uuid.New()
	require.NoError(t, persister.OpenSession(ctx, services.OpenSessionInput{
		SessionID:     sessionID,
		SpeakerUserID: user.ID,
		CallID:        &call.ID,
		Engine: services.EngineSnapshot{
			Mode:    entities.TranscriptEngineModeOpenAI,
			Target:  "https://api.openai.com/v1/",
			ModelID: "whisper-1",
		},
		SampleRate:  16000,
		FrameFormat: "pcm_s16le_mono",
		ExpiresAt:   time.Now().UTC().Add(15 * time.Minute),
	}))

	// Five sequential finals, each 1.5 s of "audio".
	texts := []string{
		"hello world",
		"this is the second segment",
		"persistence verified",
		"per-word timing checks below",
		"end of test fixture",
	}
	conf := float32(0.9)
	lang := "en"
	for i, text := range texts {
		require.NoError(t, persister.RecordFinal(ctx, services.RecordFinalInput{
			SessionID:    sessionID,
			CallerUserID: user.ID,
			SegmentIndex: int32(i),
			Text:         text,
			Language:     &lang,
			Confidence:   &conf,
			StartMs:      int32(i) * 1500,
			EndMs:        int32(i+1) * 1500,
			Words: []entities.WordTiming{
				{Word: "hello", StartMs: 0, EndMs: 700, Probability: 0.92},
			},
		}))
	}

	// Read back: segments are ordered by (segment_started_at, segment_index)
	// and carry the denormalised engine snapshot.
	segs, err := tr.ListSegmentsBySession(ctx, sessionID)
	require.NoError(t, err)
	require.Len(t, segs, 5)
	for i, s := range segs {
		assert.EqualValues(t, i, s.SegmentIndex)
		assert.Equal(t, texts[i], s.Text)
		assert.Equal(t, entities.TranscriptEngineModeOpenAI, s.EngineMode)
		assert.Equal(t, "whisper-1", s.ModelID)
		assert.Equal(t, user.ID, s.SpeakerUserID)
		require.NotNil(t, s.CallID)
		assert.Equal(t, call.ID, *s.CallID)
	}

	// Counters reflect exactly 5 inserts and 7.5 s of audio.
	got, err := tr.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.EqualValues(t, 5, got.FinalizedSegmentCount)
	assert.InDelta(t, 7.5, got.AudioSecondsTotal, 0.001)

	// Close with stitch: calls.transcript becomes the space-joined text.
	require.NoError(t, persister.CloseSession(ctx, services.CloseSessionInput{
		SessionID:    sessionID,
		CallerUserID: user.ID,
		Status:       entities.TranscriptSessionStatusEnded,
		StitchInto:   &call.ID,
	}))

	closed, err := tr.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, entities.TranscriptSessionStatusEnded, closed.Status)
	require.NotNil(t, closed.EndedAt)

	updated, err := cr.GetByID(ctx, call.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.Transcript)
	assert.Equal(t,
		"hello world this is the second segment persistence verified per-word timing checks below end of test fixture",
		*updated.Transcript,
	)
}

// TestTranscriptPersistence_MeetingTwoParticipants verifies that two parallel
// ASR sessions in the same meeting attribute their segments to the right
// speaker_user_id and never cross-contaminate.
func TestTranscriptPersistence_MeetingTwoParticipants(t *testing.T) {
	tr, _, mr, persister := newPersistenceFixture(t)
	ctx := context.Background()

	userRepo := repositories.NewUserRepository(testDB(t))
	alice := createTestUser(t, ctx, userRepo, "alice-transcript@rekall.io", "Alice")
	bob := createTestUser(t, ctx, userRepo, "bob-transcript@rekall.io", "Bob")

	meeting := createTestMeeting(t, ctx, mr, &entities.Meeting{
		ID:              uuid.New(),
		Code:            "trx-" + uuid.New().String()[:8],
		Title:           "Two-Speaker E2E",
		Type:            entities.MeetingTypeOpen,
		HostID:          alice.ID,
		Status:          entities.MeetingStatusActive,
		MaxParticipants: 50,
		StartedAt:       timePtr(time.Now().UTC()),
	})

	// Two sessions, one per participant.
	openMeeting := func(speaker uuid.UUID) uuid.UUID {
		sid := uuid.New()
		require.NoError(t, persister.OpenSession(ctx, services.OpenSessionInput{
			SessionID:     sid,
			SpeakerUserID: speaker,
			MeetingID:     &meeting.ID,
			Engine: services.EngineSnapshot{
				Mode: entities.TranscriptEngineModeLocal, Target: "/models/whisper.bin", ModelID: "whisper-large-v3",
			},
			SampleRate: 16000, FrameFormat: "pcm_s16le_mono",
			ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
		}))
		return sid
	}
	aSess := openMeeting(alice.ID)
	bSess := openMeeting(bob.ID)

	// Interleave 3 segments per speaker.
	for i := int32(0); i < 3; i++ {
		require.NoError(t, persister.RecordFinal(ctx, services.RecordFinalInput{
			SessionID: aSess, CallerUserID: alice.ID, SegmentIndex: i,
			Text: "alice " + string(rune('A'+i)), StartMs: i * 1000, EndMs: (i + 1) * 1000,
		}))
		require.NoError(t, persister.RecordFinal(ctx, services.RecordFinalInput{
			SessionID: bSess, CallerUserID: bob.ID, SegmentIndex: i,
			Text: "bob " + string(rune('A'+i)), StartMs: i * 1000, EndMs: (i + 1) * 1000,
		}))
	}

	// Each session's segments belong to the right speaker.
	aSegs, err := tr.ListSegmentsBySession(ctx, aSess)
	require.NoError(t, err)
	require.Len(t, aSegs, 3)
	for _, s := range aSegs {
		assert.Equal(t, alice.ID, s.SpeakerUserID)
		require.NotNil(t, s.MeetingID)
		assert.Equal(t, meeting.ID, *s.MeetingID)
		assert.Nil(t, s.CallID)
	}

	bSegs, err := tr.ListSegmentsBySession(ctx, bSess)
	require.NoError(t, err)
	require.Len(t, bSegs, 3)
	for _, s := range bSegs {
		assert.Equal(t, bob.ID, s.SpeakerUserID)
	}

	// The meeting-wide listing returns all 6 segments.
	all, _, err := tr.ListSegmentsByMeeting(ctx, meeting.ID, 1, 100)
	require.NoError(t, err)
	assert.Len(t, all, 6)

	// Cross-session ownership check: Alice cannot post under Bob's session.
	err = persister.RecordFinal(ctx, services.RecordFinalInput{
		SessionID: bSess, CallerUserID: alice.ID, SegmentIndex: 99,
		Text: "spoofed", StartMs: 99000, EndMs: 100000,
	})
	assert.ErrorIs(t, err, services.ErrTranscriptSessionNotOwned)

	// StitchMeeting prefixes initials and collapses consecutive same-speaker.
	stitched, err := tr.StitchMeeting(ctx, meeting.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, stitched)
}

// TestTranscriptPersistence_SessionExpiry verifies the cleanup job transitions
// orphaned active sessions to 'expired' and stitches the partial transcript
// into calls.transcript for the bound Call.
func TestTranscriptPersistence_SessionExpiry(t *testing.T) {
	tr, cr, _, persister := newPersistenceFixture(t)
	ctx := context.Background()

	user := createTestUser(t, ctx, repositories.NewUserRepository(testDB(t)),
		"expiry-e2e@rekall.io", "Expiry E2E")
	call := makeCallForTranscript(t, ctx, user.ID)

	// Open a session whose expires_at is already in the past.
	sid := uuid.New()
	require.NoError(t, tr.CreateSession(ctx, &entities.TranscriptSession{
		ID:            sid,
		SpeakerUserID: user.ID,
		CallID:        &call.ID,
		EngineMode:    entities.TranscriptEngineModeLocal,
		EngineTarget:  "/models/whisper.bin",
		ModelID:       "whisper-large-v3",
		SampleRate:    16000,
		FrameFormat:   "pcm_s16le_mono",
		Status:        entities.TranscriptSessionStatusActive,
		StartedAt:     time.Now().UTC().Add(-30 * time.Minute),
		ExpiresAt:     time.Now().UTC().Add(-1 * time.Minute), // already expired
	}))

	// One persisted partial transcript.
	require.NoError(t, persister.RecordFinal(ctx, services.RecordFinalInput{
		SessionID:    sid,
		CallerUserID: user.ID,
		SegmentIndex: 0,
		Text:         "this segment was captured before the session timed out",
		StartMs:      0,
		EndMs:        2000,
	}))

	// Run the cleanup job once.
	job := services.NewTranscriptCleanupJob(
		tr, persister,
		services.TranscriptCleanupConfig{Interval: time.Hour, BatchSize: 100},
		zap.NewNop(),
	)
	job.RunOnce(ctx)

	// Status flips to expired and calls.transcript gets the partial text.
	closed, err := tr.GetSession(ctx, sid)
	require.NoError(t, err)
	assert.Equal(t, entities.TranscriptSessionStatusExpired, closed.Status)
	require.NotNil(t, closed.EndedAt)

	updated, err := cr.GetByID(ctx, call.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.Transcript)
	assert.Contains(t, *updated.Transcript, "this segment was captured before")
}
