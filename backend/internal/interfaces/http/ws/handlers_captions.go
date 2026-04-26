package ws

// handleCaptionChunk relays a speaker's caption chunk to every other admitted
// participant in the meeting. Captions are NOT persisted: each chunk is a
// best-effort message, and partials are routinely superseded by finals on the
// receiving side. We exclude the sender to avoid echoing their own text back.
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
}
