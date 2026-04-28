package services_test

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
)

func TestTranscriptCleanupJob_FlipsExpiredToExpired(t *testing.T) {
	repo := newFakeTranscriptRepo()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())

	speaker := uuid.New()
	mid := uuid.New()
	expired := &entities.TranscriptSession{
		ID:            uuid.New(),
		SpeakerUserID: speaker,
		MeetingID:     &mid,
		EngineMode:    entities.TranscriptEngineModeOpenAI,
		EngineTarget:  "https://api.openai.com/v1/",
		ModelID:       "whisper-1",
		SampleRate:    16000,
		FrameFormat:   "pcm_s16le_mono",
		Status:        entities.TranscriptSessionStatusActive,
		StartedAt:     time.Now().UTC().Add(-30 * time.Minute),
		ExpiresAt:     time.Now().UTC().Add(-1 * time.Minute),
	}
	require.NoError(t, repo.CreateSession(context.Background(), expired))

	fresh := *expired
	fresh.ID = uuid.New()
	fresh.ExpiresAt = time.Now().UTC().Add(15 * time.Minute)
	require.NoError(t, repo.CreateSession(context.Background(), &fresh))

	job := services.NewTranscriptCleanupJob(
		repo, persister,
		services.TranscriptCleanupConfig{Interval: time.Hour, BatchSize: 100},
		zap.NewNop(),
	)
	job.RunOnce(context.Background())

	got, err := repo.GetSession(context.Background(), expired.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.TranscriptSessionStatusExpired, got.Status,
		"expired session should be transitioned to 'expired'")

	gotFresh, err := repo.GetSession(context.Background(), fresh.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.TranscriptSessionStatusActive, gotFresh.Status,
		"fresh session must remain active")
}

func TestTranscriptCleanupJob_NoOpOnEmptyBatch(t *testing.T) {
	repo := newFakeTranscriptRepo()
	persister := services.NewTranscriptPersister(repo, new(mockCallRepo), new(mockMeetingRepo), zap.NewNop())
	job := services.NewTranscriptCleanupJob(
		repo, persister,
		services.TranscriptCleanupConfig{Interval: time.Hour, BatchSize: 100},
		zap.NewNop(),
	)
	// Run with no rows in the repo. Must not panic / error.
	job.RunOnce(context.Background())
}
