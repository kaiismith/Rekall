package ws

// handleMediaState broadcasts a participant's audio/video enabled state to all
// other admitted clients and updates the hub's mediaState snapshot.
func handleMediaState(h *Hub, from *Client, msg InboundMessage) {
	uid := from.UserID
	uidStr := uid.String()

	// Update snapshot so newly joining participants receive current state.
	if ps, ok := h.mediaState[uidStr]; ok {
		if msg.Audio != nil {
			ps.AudioEnabled = *msg.Audio
		}
		if msg.Video != nil {
			ps.VideoEnabled = *msg.Video
		}
	}

	h.broadcastExcept(from, OutboundMessage{
		Type:   MsgTypeMediaState,
		UserID: &uid,
		Audio:  msg.Audio,
		Video:  msg.Video,
	})
}

// handleForceMute allows the host to mute a specific participant.
// The message is forwarded point-to-point to the target; non-host senders are
// silently discarded.
func handleForceMute(h *Hub, from *Client, msg InboundMessage) {
	// Only the host may force-mute others.
	if from.UserID != h.hostID {
		return
	}
	if msg.TargetID == nil {
		return
	}
	for c := range h.clients {
		if c.UserID == *msg.TargetID {
			c.Send(OutboundMessage{Type: MsgTypeForceMute})
			return
		}
	}
}
