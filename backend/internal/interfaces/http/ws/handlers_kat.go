package ws

// handleKatToggle updates the per-client "AI notes" preference and recomputes
// the meeting-wide aggregate. The Kat scheduler stops calling Foundry / OpenAI
// the moment the aggregate flips to false; transcript persistence and the
// captions broadcast are unaffected.
//
// Per-client, not per-meeting: each participant decides whether they want
// notes generated, but cost is shared (one Foundry call serves everyone). So
// the aggregate is "any client wants it" — the moment one user turns it on,
// everyone in the meeting starts seeing notes; only when ALL clients have it
// off does the scheduler go quiet.
//
// Stays inside the hub run goroutine — no locks needed for the clients map
// or the per-client katEnabled flag.
func handleKatToggle(h *Hub, from *Client, msg InboundMessage) {
	if msg.Enabled == nil {
		return
	}
	from.katEnabled = *msg.Enabled

	if h.kat == nil {
		return
	}

	// Aggregate: if ANY admitted client has Kat on, the scheduler runs.
	any := false
	for c := range h.clients {
		if c.katEnabled {
			any = true
			break
		}
	}
	h.kat.SetKatEnabled(h.meetingID, any)
}
