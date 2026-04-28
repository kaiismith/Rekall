package services

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// TranscriptCleanupConfig controls the orphaned-session reaper.
type TranscriptCleanupConfig struct {
	// Interval is how often the job runs.
	Interval time.Duration
	// BatchSize is the maximum number of expired sessions reaped per tick.
	// Bounded so a long backlog (e.g. after a server outage) is processed in
	// digestible chunks rather than one mega-transaction.
	BatchSize int
}

// TranscriptCleanupJob walks transcript_sessions where status='active' AND
// expires_at < NOW() and transitions each to 'expired'. For sessions bound to
// a Call, the partial stitched text is written into calls.transcript so the
// legacy denormalised cache stays coherent — a partial transcript is more
// useful than none.
//
// This is the safety net for sessions whose owning browser disconnected
// without an explicit End call. The graceful-disconnect path already calls
// CloseSession from the WS hub.
type TranscriptCleanupJob struct {
	repo      ports.TranscriptRepository
	persister *TranscriptPersister
	cfg       TranscriptCleanupConfig
	logger    *zap.Logger
}

// NewTranscriptCleanupJob wires the job. persister is the same instance the
// rest of the backend uses; the job is only the trigger, not a duplicate
// state machine.
func NewTranscriptCleanupJob(
	repo ports.TranscriptRepository,
	persister *TranscriptPersister,
	cfg TranscriptCleanupConfig,
	logger *zap.Logger,
) *TranscriptCleanupJob {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	return &TranscriptCleanupJob{
		repo:      repo,
		persister: persister,
		cfg:       cfg,
		logger:    applogger.WithComponent(logger, "transcript_cleanup"),
	}
}

// Run blocks until ctx is cancelled, firing a cleanup tick every cfg.Interval.
func (j *TranscriptCleanupJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.RunOnce(ctx)
		}
	}
}

// RunOnce executes a single cleanup tick synchronously. Exposed for tests
// and for run-on-startup catch-up scenarios.
func (j *TranscriptCleanupJob) RunOnce(ctx context.Context) {
	expired, err := j.repo.FindExpiredActive(ctx, j.cfg.BatchSize)
	if err != nil {
		j.logger.Error("transcript cleanup: find_expired_active failed", zap.Error(err))
		return
	}
	if len(expired) == 0 {
		return
	}

	reaped := 0
	for _, s := range expired {
		var stitchInto = s.CallID // pointer; nil for meeting sessions
		err := j.persister.CloseSession(ctx, CloseSessionInput{
			SessionID:  s.ID,
			Status:     entities.TranscriptSessionStatusExpired,
			StitchInto: stitchInto,
		})
		if err != nil {
			j.logger.Warn("transcript cleanup: close_session failed",
				zap.Error(err),
				zap.String("session_id", s.ID.String()),
			)
			continue
		}
		reaped++
	}

	catalog.TranscriptSessionsExpired.Info(j.logger,
		zap.Int("reaped", reaped),
		zap.Int("found", len(expired)),
	)
}
