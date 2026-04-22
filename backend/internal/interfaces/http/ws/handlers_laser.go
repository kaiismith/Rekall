package ws

// handleLaserMove relays the laser pointer position to all other admitted
// clients. If a different participant already holds the laser, a laser_stop is
// broadcast for the previous owner first (implicit laser takeover).
func handleLaserMove(h *Hub, from *Client, msg InboundMessage) {
	if msg.X == nil || msg.Y == nil {
		return
	}
	uid := from.UserID
	uidStr := uid.String()

	// If a different participant holds the laser, revoke it.
	if h.laserOwner != "" && h.laserOwner != uidStr {
		prevUID := h.laserOwner
		// Find the previous owner's *uuid.UUID for the outbound message.
		for c := range h.clients {
			if c.UserID.String() == prevUID {
				prev := c.UserID
				h.broadcastClients(OutboundMessage{Type: MsgTypeLaserStop, UserID: &prev})
				if ps, ok := h.mediaState[prevUID]; ok {
					ps.LaserActive = false
				}
				break
			}
		}
	}

	h.laserOwner = uidStr
	if ps, ok := h.mediaState[uidStr]; ok {
		ps.LaserActive = true
	}

	h.broadcastExcept(from, OutboundMessage{
		Type:   MsgTypeLaserMove,
		UserID: &uid,
		X:      msg.X,
		Y:      msg.Y,
	})
}

// handleLaserStop clears the laser pointer for the sender and notifies all
// other admitted clients.
func handleLaserStop(h *Hub, from *Client, msg InboundMessage) {
	uid := from.UserID
	uidStr := uid.String()

	if h.laserOwner == uidStr {
		h.laserOwner = ""
	}
	if ps, ok := h.mediaState[uidStr]; ok {
		ps.LaserActive = false
	}

	h.broadcastExcept(from, OutboundMessage{
		Type:   MsgTypeLaserStop,
		UserID: &uid,
	})
}
