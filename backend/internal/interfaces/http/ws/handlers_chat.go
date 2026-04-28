package ws

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// chatInsertTimeout bounds how long the hub goroutine may wait on the DB
// before giving up on a chat message. Longer than the median insert latency
// but short enough that a stalled DB cannot block in-room events.
const chatInsertTimeout = 5 * time.Second

// handleChatMessage persists an inbound chat_message and fans it out to all
// admitted clients (including the sender so they can reconcile their
// optimistic entry by client_id).
//
// The message is silently dropped in any of the following cases:
//   - Sender is not an admitted client (never echoed, never persisted)
//   - Body is empty after trimming
//   - Body exceeds maxChatMessageLength runes
//   - Hub has no chatRepo configured (test harness path)
//   - Persist fails or times out — a Warn is logged, no broadcast occurs
func handleChatMessage(h *Hub, from *Client, msg InboundMessage) {
	if _, ok := h.clients[from]; !ok {
		return
	}
	if h.chatRepo == nil {
		return
	}

	body := strings.TrimSpace(msg.Body)
	if body == "" {
		return
	}
	if len([]rune(body)) > maxChatMessageLength {
		return
	}

	// Server-side rate limit — independent of the client's own 3/2s throttle.
	// Catches misbehaving clients without tripping the well-behaved ones.
	if !from.allowChat(time.Now()) {
		catalog.ChatRateLimited.Warn(h.logger,
			zap.String("user_id", from.UserID.String()),
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), chatInsertTimeout)
	defer cancel()

	record := &entities.MeetingMessage{
		MeetingID: h.meetingID,
		UserID:    from.UserID,
		Body:      body,
	}
	if err := h.chatRepo.Create(ctx, record); err != nil {
		catalog.ChatPersistFailed.Warn(h.logger,
			zap.String("user_id", from.UserID.String()),
			zap.Error(err),
		)
		return
	}

	uid := from.UserID
	h.broadcastClients(OutboundMessage{
		Type:     MsgTypeChatMessage,
		ID:       &record.ID,
		ClientID: msg.ClientID,
		UserID:   &uid,
		Body:     record.Body,
		SentAt:   &record.SentAt,
	})
}
