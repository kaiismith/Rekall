package ws

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// KatBroadcaster adapts the meeting WS HubManager onto the
// ports.WSBroadcaster interface used by the Kat scheduler. It only knows how
// to fan out to a Hub — solo-call routing is delegated to a separate
// per-call sender callback set by the wiring in main.go.
//
// The adapter is intentionally thin: it does not see KatNote shapes, it does
// not know about prompt versions, it just wraps the data into an
// OutboundMessage with the right Type and lets the hub do its existing
// fan-out.
type KatBroadcaster struct {
	hubs       *HubManager
	soloSender func(callID uuid.UUID, msgType string, data any)
	log        *zap.Logger
}

// NewKatBroadcaster constructs the adapter.
//
// soloSender, when non-nil, handles the solo-call routing. Pass nil if solo
// calls aren't going through Kat in your deployment; calls without a meeting
// hub will be silently dropped (with a debug log).
func NewKatBroadcaster(
	hubs *HubManager,
	soloSender func(callID uuid.UUID, msgType string, data any),
	log *zap.Logger,
) *KatBroadcaster {
	return &KatBroadcaster{hubs: hubs, soloSender: soloSender, log: log}
}

// BroadcastToMeeting fans the message out to every admitted client of the
// meeting hub. When no hub exists (the meeting wound down between the
// scheduler's load-segments call and the broadcast), the message is dropped
// silently. The Kat ring buffer still holds the note for late-join replay.
//
// The ws.Hub does not currently expose a public broadcast method outside its
// run goroutine; we use Send-on-clients via the hub's existing helper by
// acquiring the manager's read lock. To keep the broadcast off the run loop
// goroutine we walk the clients map under the manager's read lock — this is
// safe because the hub's run loop only mutates the clients map under its own
// goroutine and we accept eventually-consistent reads (a note arriving in the
// instant a client connects/disconnects may be missed; the next tick covers it).
func (k *KatBroadcaster) BroadcastToMeeting(meetingID uuid.UUID, msgType string, data any) {
	hub := k.hubs.Get(meetingID)
	if hub == nil {
		// Solo-call code path piggybacks the same port; route via the call sender.
		if k.soloSender != nil {
			k.soloSender(meetingID, msgType, data)
		}
		return
	}
	hub.EnqueueBroadcast(OutboundMessage{Type: msgType, Data: data})
}

// SendToUser routes a message to a specific user via every hub the user is
// currently in. (In v1 a user is in at most one meeting at a time; we walk
// all hubs to stay independent of that invariant.)
func (k *KatBroadcaster) SendToUser(userID uuid.UUID, msgType string, data any) {
	msg := OutboundMessage{Type: msgType, Data: data}
	for _, hub := range k.hubs.allHubs() {
		hub.EnqueueSendToUser(userID, msg)
	}
}
