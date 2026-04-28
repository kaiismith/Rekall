package ws

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// transcriptPersistTimeout caps how long the off-path persistence goroutine
// will spend on a single segment write. Short enough that goroutines don't
// pile up unbounded if the DB is slow; long enough to absorb a normal
// transient hiccup.
const transcriptPersistTimeout = 5 * time.Second

// handleCaptionChunk relays a speaker's caption chunk to every other admitted
// participant in the meeting. Captions are also persisted (best-effort, off
// the broadcast critical path) when the inbound message carries the
// persistence-shape fields (session_id, segment_index, start_ms, end_ms) and
// kind == "final".
//
// We exclude the sender from the broadcast to avoid echoing their own text
// back. Persistence failures are logged at warn but never block or fail the
// broadcast — the captions UX continues working even when persistence is
// degraded.
func handleCaptionChunk(h *Hub, from *Client, msg InboundMessage) {
	if _, ok := h.clients[from]; !ok {
		return
	}
	if msg.CaptionKind != "partial" && msg.CaptionKind != "final" {
		return
	}
	if msg.CaptionText == "" || msg.CaptionSegmentID == "" {
		return
	}

	uid := from.UserID
	h.broadcastExcept(from, OutboundMessage{
		Type:             MsgTypeCaptionChunk,
		UserID:           &uid,
		FullName:         from.FullName,
		Initials:         from.Initials,
		CaptionKind:      msg.CaptionKind,
		CaptionText:      msg.CaptionText,
		CaptionSegmentID: msg.CaptionSegmentID,
		CaptionTimestamp: msg.CaptionTimestamp,
	})

	// Persistence: only `final` segments, only when the client provides the
	// full persistence-shape payload (session_id + segment_index + timing).
	// Older clients sending only text continue to be relayed — we just skip
	// persistence for them.
	if msg.CaptionKind != "final" {
		return
	}
	if h.persister == nil {
		return
	}
	if msg.ASRSessionID == nil || msg.SegmentIndex == nil ||
		msg.StartMs == nil || msg.EndMs == nil {
		return
	}

	// Remember the session so the disconnect path can close it immediately.
	// Owned by the hub goroutine — same goroutine that calls handleUnregister.
	from.activeASRSessionID = msg.ASRSessionID

	// Detach: a slow DB write must not stall the hub's run loop or other
	// participants. The goroutine carries its own context with a tight
	// timeout so a hung DB doesn't leak goroutines.
	in := services.RecordFinalInput{
		SessionID:    *msg.ASRSessionID,
		CallerUserID: from.UserID,
		SegmentIndex: *msg.SegmentIndex,
		Text:         msg.CaptionText,
		Language:     msg.Language,
		Confidence:   msg.Confidence,
		StartMs:      *msg.StartMs,
		EndMs:        *msg.EndMs,
		Words:        msg.Words,
	}
	persister := h.persister
	logger := h.logger
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), transcriptPersistTimeout)
		defer cancel()
		if err := persister.RecordFinal(ctx, in); err != nil {
			// All sentinel rejections are already logged in the persister
			// (TRANSCRIPT_SESSION_NOT_OWNED at warn for the security signal).
			// "Not found" is benign — the session predates persistence rollout
			// or OpenSession failed at issue time. Demote to Debug so we don't
			// flood the warn channel for every segment of an un-tracked session.
			if errors.Is(err, services.ErrTranscriptSessionNotFound) {
				logger.Debug("transcript segment skipped: no session row",
					zap.String("session_id", in.SessionID.String()),
					zap.Int32("segment_index", in.SegmentIndex),
				)
				return
			}
			// Catch-all warn for genuine infra errors.
			catalog.TranscriptPersistFailed.Warn(logger,
				zap.Error(err),
				zap.String("session_id", in.SessionID.String()),
				zap.Int32("segment_index", in.SegmentIndex),
			)
		}
	}()
}
