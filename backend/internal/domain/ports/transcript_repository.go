package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/entities"
)

// TranscriptRepository abstracts persistence for ASR session and segment rows.
// Infrastructure implementations must satisfy this interface.
type TranscriptRepository interface {
	// CreateSession inserts a new transcript_sessions row. The caller must set
	// s.ID to the session_id issued by the ASR service (StartSessionOutput).
	CreateSession(ctx context.Context, s *entities.TranscriptSession) error

	// GetSession returns the session row by its (ASR-issued) id.
	// Returns an error wrapping errors.NotFound when no row exists.
	GetSession(ctx context.Context, sessionID uuid.UUID) (*entities.TranscriptSession, error)

	// UpdateSessionStatus flips the lifecycle state. errCode and errMsg are
	// only honoured when status == TranscriptSessionStatusErrored. ended_at is
	// stamped to NOW() for any terminal state (ended | errored | expired).
	UpdateSessionStatus(
		ctx context.Context,
		sessionID uuid.UUID,
		status entities.TranscriptSessionStatus,
		errCode, errMsg *string,
	) error

	// UpsertSegment writes a transcript_segments row, treating duplicates on
	// (session_id, segment_index) as an in-place update (ON CONFLICT DO UPDATE).
	// On a true insert (not an update) the parent session's
	// finalized_segment_count is incremented by 1 and audio_seconds_total is
	// increased by (end_ms - start_ms) / 1000. Both writes happen atomically.
	UpsertSegment(ctx context.Context, seg *entities.TranscriptSegment) error

	// ListSegmentsBySession returns every segment for a session, ordered by
	// (segment_started_at, segment_index).
	ListSegmentsBySession(ctx context.Context, sessionID uuid.UUID) ([]*entities.TranscriptSegment, error)

	// ListSegmentsByCall returns segments belonging to all sessions bound to
	// the given call, paginated. Returns (segments, total, error).
	ListSegmentsByCall(ctx context.Context, callID uuid.UUID, page, perPage int) ([]*entities.TranscriptSegment, int, error)

	// ListSegmentsByMeeting returns segments belonging to all sessions bound
	// to the given meeting, paginated. Returns (segments, total, error).
	ListSegmentsByMeeting(ctx context.Context, meetingID uuid.UUID, page, perPage int) ([]*entities.TranscriptSegment, int, error)

	// ListSegmentsByMeetingInRange returns segments belonging to a meeting
	// whose segment_started_at falls within [fromTs, toTs) (inclusive lower,
	// exclusive upper), ordered by (segment_started_at, segment_index).
	// Used by the Kat live-notes scheduler to load a sliding window of recent
	// segments. Returns an empty slice (not nil) when nothing matches.
	ListSegmentsByMeetingInRange(ctx context.Context, meetingID uuid.UUID, fromTs, toTs time.Time) ([]*entities.TranscriptSegment, error)

	// ListSegmentsByCallInRange is the solo-call analogue of
	// ListSegmentsByMeetingInRange.
	ListSegmentsByCallInRange(ctx context.Context, callID uuid.UUID, fromTs, toTs time.Time) ([]*entities.TranscriptSegment, error)

	// ListSessionsByCall returns the session rows bound to the given call
	// ordered by started_at ASC.
	ListSessionsByCall(ctx context.Context, callID uuid.UUID) ([]*entities.TranscriptSession, error)

	// ListSessionsByMeeting returns the session rows bound to the given meeting
	// ordered by started_at ASC.
	ListSessionsByMeeting(ctx context.Context, meetingID uuid.UUID) ([]*entities.TranscriptSession, error)

	// ListSpeakerUserIDsByMeeting returns the distinct user ids that opened a
	// transcript session against the given meeting, ordered deterministically
	// by user id ASC. Used by the records detail handler to resolve speaker
	// display names for the timeline.
	ListSpeakerUserIDsByMeeting(ctx context.Context, meetingID uuid.UUID) ([]uuid.UUID, error)

	// FindExpiredActive returns active sessions whose expires_at has passed,
	// up to limit rows, oldest first. Used by the cleanup job.
	FindExpiredActive(ctx context.Context, limit int) ([]*entities.TranscriptSession, error)

	// StitchSession concatenates a single session's segments into plain text.
	StitchSession(ctx context.Context, sessionID uuid.UUID) (string, error)

	// StitchCall concatenates every session bound to the call into plain text.
	// Single-speaker (no speaker prefixes).
	StitchCall(ctx context.Context, callID uuid.UUID) (string, error)

	// StitchMeeting concatenates every session bound to the meeting, prefixing
	// each segment with the speaker's initials and collapsing consecutive
	// same-speaker segments.
	StitchMeeting(ctx context.Context, meetingID uuid.UUID) (string, error)
}
