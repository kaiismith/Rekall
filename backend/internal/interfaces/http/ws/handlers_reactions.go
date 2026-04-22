package ws

// handleEmojiReaction broadcasts a floating emoji reaction to ALL admitted
// clients, including the sender (so the sender sees their own reaction).
func handleEmojiReaction(h *Hub, from *Client, msg InboundMessage) {
	if msg.Emoji == "" {
		return
	}
	uid := from.UserID
	h.broadcastClients(OutboundMessage{
		Type:  MsgTypeEmojiReaction,
		From:  &uid,
		Emoji: msg.Emoji,
	})
}

// handleHandRaise broadcasts a participant's raise/lower-hand state to all
// other admitted clients and updates the hub's mediaState snapshot.
func handleHandRaise(h *Hub, from *Client, msg InboundMessage) {
	if msg.Raised == nil {
		return
	}
	uid := from.UserID
	uidStr := uid.String()

	// Update snapshot.
	if ps, ok := h.mediaState[uidStr]; ok {
		ps.HandRaised = *msg.Raised
	}

	h.broadcastExcept(from, OutboundMessage{
		Type:   MsgTypeHandRaise,
		UserID: &uid,
		Raised: msg.Raised,
	})
}
