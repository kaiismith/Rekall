package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/domain/entities"
	apperr "github.com/rekall/backend/pkg/errors"
)

// TranscriptRepository implements ports.TranscriptRepository using GORM.
type TranscriptRepository struct {
	db *gorm.DB
}

// NewTranscriptRepository creates a TranscriptRepository backed by the given GORM DB.
func NewTranscriptRepository(db *gorm.DB) *TranscriptRepository {
	return &TranscriptRepository{db: db}
}

// CreateSession persists a new transcript_sessions row.
func (r *TranscriptRepository) CreateSession(ctx context.Context, s *entities.TranscriptSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// GetSession retrieves a session by its ASR-issued id.
func (r *TranscriptRepository) GetSession(ctx context.Context, sessionID uuid.UUID) (*entities.TranscriptSession, error) {
	var s entities.TranscriptSession
	err := r.db.WithContext(ctx).First(&s, "id = ?", sessionID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("TranscriptSession", sessionID.String())
		}
		return nil, err
	}
	return &s, nil
}

// UpdateSessionStatus flips the lifecycle state, stamping ended_at on any
// terminal state.
func (r *TranscriptRepository) UpdateSessionStatus(
	ctx context.Context,
	sessionID uuid.UUID,
	status entities.TranscriptSessionStatus,
	errCode, errMsg *string,
) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	if status != entities.TranscriptSessionStatusActive {
		updates["ended_at"] = time.Now().UTC()
	}
	if status == entities.TranscriptSessionStatusErrored {
		updates["error_code"] = errCode
		updates["error_message"] = errMsg
	}

	res := r.db.WithContext(ctx).
		Model(&entities.TranscriptSession{}).
		Where("id = ?", sessionID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.NotFound("TranscriptSession", sessionID.String())
	}
	return nil
}

// UpsertSegment writes a segment row with ON CONFLICT DO UPDATE on
// (session_id, segment_index). The parent session's counters are incremented
// only on a true insert (xmax = 0 trick), so duplicate retransmissions don't
// double-count.
func (r *TranscriptRepository) UpsertSegment(ctx context.Context, seg *entities.TranscriptSegment) error {
	if seg.ID == uuid.Nil {
		seg.ID = uuid.New()
	}

	wordsJSON, err := seg.Words.Value()
	if err != nil {
		return fmt.Errorf("transcript segment: encode words: %w", err)
	}

	durationSec := float64(seg.EndMs-seg.StartMs) / 1000.0

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Upsert and detect insert-vs-update via the Postgres `xmax = 0` trick:
		// for a freshly-inserted tuple xmax is 0; for an UPDATE-via-ON-CONFLICT
		// it carries the updating xid.
		row := tx.Raw(`
			INSERT INTO transcript_segments
			    (id, session_id, segment_index, speaker_user_id, call_id, meeting_id,
			     text, language, confidence, start_ms, end_ms, words,
			     engine_mode, model_id, segment_started_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
			ON CONFLICT (session_id, segment_index) DO UPDATE SET
			    text                = EXCLUDED.text,
			    language            = EXCLUDED.language,
			    confidence          = EXCLUDED.confidence,
			    start_ms            = EXCLUDED.start_ms,
			    end_ms              = EXCLUDED.end_ms,
			    words               = EXCLUDED.words,
			    segment_started_at  = EXCLUDED.segment_started_at
			RETURNING (xmax = 0) AS inserted
		`,
			seg.ID, seg.SessionID, seg.SegmentIndex, seg.SpeakerUserID,
			seg.CallID, seg.MeetingID,
			seg.Text, seg.Language, seg.Confidence, seg.StartMs, seg.EndMs, wordsJSON,
			seg.EngineMode, seg.ModelID, seg.SegmentStartedAt,
		).Row()

		var inserted bool
		if err := row.Scan(&inserted); err != nil {
			return err
		}

		if inserted {
			if err := tx.Exec(`
				UPDATE transcript_sessions
				   SET finalized_segment_count = finalized_segment_count + 1,
				       audio_seconds_total     = audio_seconds_total + ?,
				       updated_at              = NOW()
				 WHERE id = ?
			`, durationSec, seg.SessionID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListSegmentsBySession returns every segment for a session, ordered by
// (segment_started_at, segment_index).
func (r *TranscriptRepository) ListSegmentsBySession(
	ctx context.Context,
	sessionID uuid.UUID,
) ([]*entities.TranscriptSegment, error) {
	var rows []*entities.TranscriptSegment
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("segment_started_at ASC, segment_index ASC").
		Find(&rows).Error
	return rows, err
}

// ListSegmentsByCall returns paginated segments for a call.
func (r *TranscriptRepository) ListSegmentsByCall(
	ctx context.Context,
	callID uuid.UUID,
	page, perPage int,
) ([]*entities.TranscriptSegment, int, error) {
	return r.listSegmentsByParent(ctx, "call_id = ?", callID, page, perPage)
}

// ListSegmentsByMeeting returns paginated segments for a meeting.
func (r *TranscriptRepository) ListSegmentsByMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	page, perPage int,
) ([]*entities.TranscriptSegment, int, error) {
	return r.listSegmentsByParent(ctx, "meeting_id = ?", meetingID, page, perPage)
}

func (r *TranscriptRepository) listSegmentsByParent(
	ctx context.Context,
	whereClause string,
	parentID uuid.UUID,
	page, perPage int,
) ([]*entities.TranscriptSegment, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}

	var (
		rows  []*entities.TranscriptSegment
		total int64
	)

	base := r.db.WithContext(ctx).Model(&entities.TranscriptSegment{}).Where(whereClause, parentID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := base.
		Order("segment_started_at ASC, segment_index ASC").
		Limit(perPage).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, int(total), nil
}

// ListSessionsByCall returns sessions bound to a call, ordered by started_at ASC.
func (r *TranscriptRepository) ListSessionsByCall(
	ctx context.Context,
	callID uuid.UUID,
) ([]*entities.TranscriptSession, error) {
	var rows []*entities.TranscriptSession
	err := r.db.WithContext(ctx).
		Where("call_id = ?", callID).
		Order("started_at ASC").
		Find(&rows).Error
	return rows, err
}

// ListSessionsByMeeting returns sessions bound to a meeting, ordered by started_at ASC.
func (r *TranscriptRepository) ListSessionsByMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
) ([]*entities.TranscriptSession, error) {
	var rows []*entities.TranscriptSession
	err := r.db.WithContext(ctx).
		Where("meeting_id = ?", meetingID).
		Order("started_at ASC").
		Find(&rows).Error
	return rows, err
}

// FindExpiredActive returns active sessions whose expires_at has passed.
// Used by the cleanup job; oldest-first so the longest-stale rows close first.
func (r *TranscriptRepository) FindExpiredActive(
	ctx context.Context,
	limit int,
) ([]*entities.TranscriptSession, error) {
	if limit < 1 {
		limit = 100
	}
	var rows []*entities.TranscriptSession
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at < NOW()", entities.TranscriptSessionStatusActive).
		Order("expires_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// StitchSession joins one session's segments into plain text.
func (r *TranscriptRepository) StitchSession(
	ctx context.Context,
	sessionID uuid.UUID,
) (string, error) {
	segs, err := r.ListSegmentsBySession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(segs))
	for _, s := range segs {
		if s.Text != "" {
			parts = append(parts, s.Text)
		}
	}
	return strings.Join(parts, " "), nil
}

// StitchCall joins every session bound to the call into single-speaker plain
// text. Calls today are single-speaker by construction (one user, one ASR
// session) so no speaker prefix is added.
func (r *TranscriptRepository) StitchCall(
	ctx context.Context,
	callID uuid.UUID,
) (string, error) {
	segs, _, err := r.ListSegmentsByCall(ctx, callID, 1, 100000)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(segs))
	for _, s := range segs {
		if s.Text != "" {
			parts = append(parts, s.Text)
		}
	}
	return strings.Join(parts, " "), nil
}

// StitchMeeting joins every session bound to the meeting into multi-speaker
// plain text, prefixing each segment with the speaker's initials. Consecutive
// same-speaker segments are collapsed under a single prefix.
func (r *TranscriptRepository) StitchMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
) (string, error) {
	type row struct {
		Text          string
		SpeakerUserID uuid.UUID
		FullName      string
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("transcript_segments AS ts").
		Select("ts.text AS text, ts.speaker_user_id AS speaker_user_id, u.full_name AS full_name").
		Joins("LEFT JOIN users u ON u.id = ts.speaker_user_id").
		Where("ts.meeting_id = ?", meetingID).
		Order("ts.segment_started_at ASC, ts.segment_index ASC").
		Scan(&rows).Error
	if err != nil {
		return "", err
	}

	var (
		parts       []string
		prevSpeaker uuid.UUID
		buf         strings.Builder
		flush       = func() {
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
		}
	)

	for _, r := range rows {
		if r.Text == "" {
			continue
		}
		if r.SpeakerUserID != prevSpeaker {
			flush()
			buf.WriteString(transcriptInitials(r.FullName))
			buf.WriteString(": ")
			buf.WriteString(r.Text)
			prevSpeaker = r.SpeakerUserID
		} else {
			buf.WriteByte(' ')
			buf.WriteString(r.Text)
		}
	}
	flush()
	return strings.Join(parts, "\n"), nil
}

// transcriptInitials returns up to two uppercase initials from a full name.
// Mirrors meetingInitials in meeting_repository.go.
func transcriptInitials(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "?"
	}
	first := strings.ToUpper(string([]rune(parts[0])[:1]))
	if len(parts) == 1 {
		return first
	}
	return first + strings.ToUpper(string([]rune(parts[len(parts)-1])[:1]))
}
