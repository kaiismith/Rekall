package services

import (
	"context"
	"time"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

// MeetingCleanupConfig controls thresholds for the background cleanup job.
type MeetingCleanupConfig struct {
	// Interval is how often the cleanup job runs.
	Interval time.Duration
	// WaitingTimeout is the maximum time a meeting may stay in the 'waiting'
	// state (no participants have joined yet) before being auto-ended.
	WaitingTimeout time.Duration
	// MaxDuration is the maximum wall-clock duration of an 'active' meeting
	// before the cleanup job forcibly ends it.
	MaxDuration time.Duration
}

// MeetingCleanupJob runs on a ticker and ends stale meetings.
type MeetingCleanupJob struct {
	meetingRepo     ports.MeetingRepository
	participantRepo ports.MeetingParticipantRepository
	cfg             MeetingCleanupConfig
	logger          *zap.Logger
}

func NewMeetingCleanupJob(
	meetingRepo ports.MeetingRepository,
	participantRepo ports.MeetingParticipantRepository,
	cfg MeetingCleanupConfig,
	logger *zap.Logger,
) *MeetingCleanupJob {
	return &MeetingCleanupJob{
		meetingRepo:     meetingRepo,
		participantRepo: participantRepo,
		cfg:             cfg,
		logger:          applogger.WithComponent(logger, "meeting_cleanup"),
	}
}

// Run blocks until ctx is cancelled, firing a cleanup tick every cfg.Interval.
func (j *MeetingCleanupJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.tick(ctx)
		}
	}
}

// RunOnce executes a single cleanup tick synchronously. Exposed for testing.
func (j *MeetingCleanupJob) RunOnce(ctx context.Context) {
	j.tick(ctx)
}

func (j *MeetingCleanupJob) tick(ctx context.Context) {
	catalog.CleanupJobStarted.Info(j.logger)

	ended := 0
	errs := 0

	// 1. Waiting meetings that never attracted a participant.
	staleWaiting, err := j.meetingRepo.FindStaleWaiting(ctx, j.cfg.WaitingTimeout)
	if err != nil {
		catalog.CleanupJobError.Error(j.logger, zap.Error(err), zap.String("phase", "stale_waiting"))
		errs++
	} else {
		for _, m := range staleWaiting {
			if e := j.autoEnd(ctx, m); e != nil {
				errs++
			} else {
				ended++
			}
		}
	}

	// 2. Active meetings that exceeded the maximum duration.
	staleActive, err := j.meetingRepo.FindStaleActive(ctx, j.cfg.MaxDuration)
	if err != nil {
		catalog.CleanupJobError.Error(j.logger, zap.Error(err), zap.String("phase", "stale_active"))
		errs++
	} else {
		for _, m := range staleActive {
			if e := j.autoEnd(ctx, m); e != nil {
				errs++
			} else {
				ended++
			}
		}
	}

	// 3. Active meetings with zero remaining participants (e.g. after a server
	//    restart that did not flush in-memory hub state to the DB).
	abandoned, err := j.meetingRepo.FindActiveWithNoParticipants(ctx)
	if err != nil {
		catalog.CleanupJobError.Error(j.logger, zap.Error(err), zap.String("phase", "abandoned"))
		errs++
	} else {
		for _, m := range abandoned {
			if e := j.autoEnd(ctx, m); e != nil {
				errs++
			} else {
				ended++
			}
		}
	}

	catalog.CleanupJobEnded.Info(j.logger,
		zap.Int("ended", ended),
		zap.Int("errors", errs),
	)
}

// autoEnd marks a meeting as ended and evicts all active participants.
func (j *MeetingCleanupJob) autoEnd(ctx context.Context, m *entities.Meeting) error {
	now := time.Now().UTC()
	m.Status = entities.MeetingStatusEnded
	m.EndedAt = &now
	m.UpdatedAt = now

	if err := j.meetingRepo.Update(ctx, m); err != nil {
		catalog.CleanupJobError.Error(j.logger,
			zap.Error(err),
			zap.String("meeting_id", m.ID.String()),
			zap.String("phase", "update_meeting"),
		)
		return err
	}
	if err := j.participantRepo.MarkAllLeft(ctx, m.ID); err != nil {
		catalog.CleanupJobError.Error(j.logger,
			zap.Error(err),
			zap.String("meeting_id", m.ID.String()),
			zap.String("phase", "mark_all_left"),
		)
		return err
	}

	catalog.CleanupMeetingEnded.Info(j.logger,
		zap.String("meeting_id", m.ID.String()),
		zap.String("status_was", m.Status),
	)
	return nil
}
