package ws

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/ports"
	"go.uber.org/zap"
)

const (
	// knockTimeout is the maximum time a knock waits for a response before
	// being auto-denied.
	knockTimeout = 120 * time.Second

	// closeCodeDenied is sent to a knocked client when their knock is denied.
	closeCodeDenied = 4003
)

// handlerFn is the signature for message type handlers registered in the hub.
// Each handler is a package-level function in a dedicated file (handlers_*.go).
type handlerFn func(h *Hub, from *Client, msg InboundMessage)

// ParticipantState holds ephemeral in-room state for one admitted participant.
// Owned exclusively by the hub's run goroutine — no mutex needed.
type ParticipantState struct {
	AudioEnabled bool
	VideoEnabled bool
	HandRaised   bool
}

// KnockRequest represents a user waiting in the waiting room.
type KnockRequest struct {
	ID     string
	Client *Client
	UserID uuid.UUID
	Timer  *time.Timer
}

// inboundEnvelope pairs an inbound message with its sender.
type inboundEnvelope struct {
	client *Client
	msg    InboundMessage
}

// identity pairs a user's display name + initials. Kept alongside ephemeral
// mediaState so participant.joined and room_state broadcasts can surface
// human-readable names to chat consumers without per-message DB lookups.
type identity struct {
	FullName string
	Initials string
}

// Hub manages all clients for a single meeting. One goroutine (run) owns all
// map mutations; no locks are needed on clients/pending/knocks/mediaState.
type Hub struct {
	meetingID uuid.UUID
	hostID    uuid.UUID // used to validate force_mute senders

	// clients holds connected, admitted participants.
	clients map[*Client]struct{}

	// pending holds clients in the waiting room (knocked, awaiting decision).
	pending map[*Client]*KnockRequest

	// knocks indexes pending requests by KnockRequest.ID for O(1) lookup
	// on knock.respond messages.
	knocks map[string]*KnockRequest

	// mediaState holds ephemeral in-room state for each admitted participant,
	// keyed by userID string. Populated on admit; deleted on unregister.
	mediaState map[string]*ParticipantState

	// identities maps userID string → display name/initials for every
	// admitted participant. Populated at admit time from the Client struct.
	identities map[string]identity

	// handlers is a registry of message-type → handler function.
	// Adding a new in-room message type only requires registering here.
	handlers map[string]handlerFn

	// chatRepo persists chat messages before they are broadcast. Nil-safe:
	// when unset (e.g. in WS unit tests that don't exercise chat), the chat
	// handler silently drops inbound messages.
	chatRepo ports.MeetingMessageRepository

	// channels
	register   chan registerRequest
	unregister chan *Client
	inbound    chan inboundEnvelope

	// done is closed when the hub's run loop exits.
	done chan struct{}

	onEnd func(meetingID uuid.UUID) // called when the meeting ends

	logger *zap.Logger
}

type registerRequest struct {
	client  *Client
	direct  bool   // true = direct join, false = knock
	knockID string // pre-generated ID for the knock request
}

// NewHub creates a Hub for the given meeting. chatRepo may be nil — handlers
// that depend on it (handleChatMessage) will no-op when it is unset.
func NewHub(
	meetingID, hostID uuid.UUID,
	chatRepo ports.MeetingMessageRepository,
	onEnd func(uuid.UUID),
	logger *zap.Logger,
) *Hub {
	h := &Hub{
		meetingID:  meetingID,
		hostID:     hostID,
		clients:    make(map[*Client]struct{}),
		pending:    make(map[*Client]*KnockRequest),
		knocks:     make(map[string]*KnockRequest),
		mediaState: make(map[string]*ParticipantState),
		identities: make(map[string]identity),
		chatRepo:   chatRepo,
		register:   make(chan registerRequest, 8),
		unregister: make(chan *Client, 8),
		inbound:    make(chan inboundEnvelope, 64),
		done:       make(chan struct{}),
		onEnd:      onEnd,
		logger:     logger.With(zap.String("meeting_id", meetingID.String())),
	}
	h.handlers = map[string]handlerFn{
		MsgTypeMediaState:    handleMediaState,
		MsgTypeForceMute:     handleForceMute,
		MsgTypeEmojiReaction: handleEmojiReaction,
		MsgTypeHandRaise:     handleHandRaise,
		MsgTypeChatMessage:   handleChatMessage,
		MsgTypeCaptionChunk:  handleCaptionChunk,
	}
	return h
}

// Run is the hub's single-goroutine event loop. It must be started in a
// goroutine and exits when ctx is cancelled or the meeting ends.
func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)
	for {
		select {
		case <-ctx.Done():
			h.broadcastAll(OutboundMessage{Type: MsgTypeMeetingEnded})
			return

		case req := <-h.register:
			if req.direct {
				h.admitDirect(req.client)
			} else {
				h.addKnocker(req.client, req.knockID)
			}

		case c := <-h.unregister:
			h.handleUnregister(c)

		case env := <-h.inbound:
			h.handleInbound(env)
		}
	}
}

// Register enqueues a client for admission. direct=true for scope members /
// open-meeting users; false for knockers.
func (h *Hub) Register(c *Client, direct bool, knockID string) {
	h.register <- registerRequest{client: c, direct: direct, knockID: knockID}
}

// Done returns a channel that closes when the hub's run loop exits.
func (h *Hub) Done() <-chan struct{} { return h.done }

// ─── internal event handlers ──────────────────────────────────────────────────

func (h *Hub) admitDirect(c *Client) {
	h.clients[c] = struct{}{}
	h.logger.Info("admitDirect: entered", zap.String("user_id", c.UserID.String()))
	h.promoteToActive(c)
	uid := c.UserID
	ok := h.broadcastAllReport(OutboundMessage{
		Type:     MsgTypeParticipantJoined,
		UserID:   &uid,
		FullName: c.FullName,
		Initials: c.Initials,
	})
	h.logger.Info("admitDirect: broadcast participant.joined",
		zap.String("user_id", uid.String()),
		zap.Bool("send_ok_for_self", ok),
	)
}

// promoteToActive sends a room_state snapshot to the newly admitted client and
// initialises their mediaState entry. Called before the participant.joined
// broadcast so the new client knows existing participants' states immediately.
func (h *Hub) promoteToActive(c *Client) {
	// Build snapshot of all currently admitted participants (excluding c, who
	// is already in h.clients but not yet in h.mediaState).
	snapshot := h.buildRoomStateSnapshot()
	ok := c.Send(OutboundMessage{Type: MsgTypeRoomState, Participants: snapshot})
	h.logger.Info("promoteToActive: sent room_state",
		zap.String("user_id", c.UserID.String()),
		zap.Int("snapshot_size", len(snapshot)),
		zap.Bool("send_ok", ok),
	)

	// Initialise this participant's state (defaults: audio on, video on).
	uidStr := c.UserID.String()
	h.mediaState[uidStr] = &ParticipantState{
		AudioEnabled: true,
		VideoEnabled: true,
	}
	h.identities[uidStr] = identity{FullName: c.FullName, Initials: c.Initials}
}

// broadcastAllReport is broadcastAll plus a return flag indicating whether
// the named client (the joiner themselves) successfully received the broadcast.
// Used only for diagnostic logging in admitDirect; production callers can keep
// using broadcastAll.
func (h *Hub) broadcastAllReport(msg OutboundMessage) bool {
	selfOk := false
	uid := msg.UserID
	for c := range h.clients {
		ok := c.Send(msg)
		if uid != nil && c.UserID == *uid {
			selfOk = ok
		}
		if !ok {
			delete(h.clients, c)
			c.conn.Close()
		}
	}
	for c := range h.pending {
		c.Send(msg)
	}
	return selfOk
}

// buildRoomStateSnapshot returns the current state of all admitted participants
// whose mediaState entry exists (i.e. everyone already in the room).
func (h *Hub) buildRoomStateSnapshot() []RoomStateParticipant {
	result := make([]RoomStateParticipant, 0, len(h.mediaState))
	for uid, ps := range h.mediaState {
		id := h.identities[uid]
		result = append(result, RoomStateParticipant{
			UserID:     uid,
			FullName:   id.FullName,
			Initials:   id.Initials,
			Audio:      ps.AudioEnabled,
			Video:      ps.VideoEnabled,
			HandRaised: ps.HandRaised,
		})
	}
	return result
}

func (h *Hub) addKnocker(c *Client, knockID string) {
	kr := &KnockRequest{
		ID:     knockID,
		Client: c,
		UserID: c.UserID,
	}
	kr.Timer = time.AfterFunc(knockTimeout, func() {
		h.inbound <- inboundEnvelope{
			client: c,
			msg:    InboundMessage{Type: "__knock_timeout__", KnockID: knockID},
		}
	})
	h.pending[c] = kr
	h.knocks[knockID] = kr

	// Notify all active participants.
	uid := c.UserID
	h.broadcastClients(OutboundMessage{
		Type:    MsgTypeKnockRequested,
		KnockID: knockID,
		UserID:  &uid,
	})
	h.logger.Info("knock registered", zap.String("user_id", uid.String()), zap.String("knock_id", knockID))
}

func (h *Hub) handleUnregister(c *Client) {
	// If client was in the waiting room, cancel their knock.
	if kr, ok := h.pending[c]; ok {
		kr.Timer.Stop()
		delete(h.pending, c)
		delete(h.knocks, kr.ID)
		h.broadcastClients(OutboundMessage{Type: MsgTypeKnockCancelled, KnockID: kr.ID, UserID: &kr.UserID})
		h.logger.Info("knocker disconnected", zap.String("user_id", kr.UserID.String()))
		return
	}

	// Client was an active participant.
	if _, ok := h.clients[c]; ok {
		uid := c.UserID
		uidStr := uid.String()

		// Clean up ephemeral state.
		delete(h.mediaState, uidStr)
		delete(h.identities, uidStr)
		delete(h.clients, c)

		h.broadcastAll(OutboundMessage{Type: MsgTypeParticipantLeft, UserID: &uid})
		h.logger.Info("participant left", zap.String("user_id", uidStr))

		// No participants left — meeting is over.
		if len(h.clients) == 0 {
			if h.onEnd != nil {
				h.onEnd(h.meetingID)
			}
		}
	}
}

func (h *Hub) handleInbound(env inboundEnvelope) {
	msg := env.msg
	c := env.client

	// Check the handler registry first (new in-room control message types).
	if fn, ok := h.handlers[msg.Type]; ok {
		fn(h, c, msg)
		return
	}

	// Legacy switch for pre-existing message types.
	switch msg.Type {
	case MsgTypeKnockRespond:
		h.handleKnockRespond(c, msg)

	case "__knock_timeout__":
		h.handleKnockTimeout(msg.KnockID)

	case MsgTypeOffer, MsgTypeAnswer, MsgTypeIceCandidate:
		// Relay WebRTC signaling to the target peer.
		if msg.To == nil {
			return
		}
		uid := c.UserID
		for peer := range h.clients {
			if peer.UserID == *msg.To {
				peer.Send(OutboundMessage{
					Type:    msg.Type,
					From:    &uid,
					Payload: msg.Payload,
				})
				return
			}
		}

	case MsgTypeSpeakingState:
		// Broadcast VAD speaking state to all other participants.
		uid := c.UserID
		out := OutboundMessage{
			Type:    MsgTypeSpeakingState,
			From:    &uid,
			Payload: msg.Payload,
		}
		for peer := range h.clients {
			if peer != c {
				peer.Send(out)
			}
		}
	}
}

func (h *Hub) handleKnockRespond(responder *Client, msg InboundMessage) {
	// Responder must be an admitted participant.
	if _, ok := h.clients[responder]; !ok {
		return
	}
	kr, ok := h.knocks[msg.KnockID]
	if !ok {
		// Already resolved (race: two people clicked admit simultaneously).
		return
	}

	// Stop the auto-deny timer.
	kr.Timer.Stop()
	delete(h.knocks, kr.ID)
	delete(h.pending, kr.Client)

	approved := msg.Approved != nil && *msg.Approved

	if approved {
		// Admit the knocker.
		h.clients[kr.Client] = struct{}{}
		h.promoteToActive(kr.Client)
		uid := kr.UserID
		t := true
		kr.Client.Send(OutboundMessage{Type: MsgTypeKnockApproved, KnockID: kr.ID})
		h.broadcastAll(OutboundMessage{
			Type:     MsgTypeParticipantJoined,
			UserID:   &uid,
			FullName: kr.Client.FullName,
			Initials: kr.Client.Initials,
		})
		h.broadcastAll(OutboundMessage{Type: MsgTypeKnockResolved, KnockID: kr.ID, Approved: &t, UserID: &uid})
		h.logger.Info("knock approved", zap.String("user_id", uid.String()))
	} else {
		f := false
		uid := kr.UserID
		kr.Client.Send(OutboundMessage{Type: MsgTypeKnockDenied, KnockID: kr.ID})
		kr.Client.closeWith(closeCodeDenied, "knock denied")
		h.broadcastClients(OutboundMessage{Type: MsgTypeKnockResolved, KnockID: kr.ID, Approved: &f, UserID: &uid})
		h.logger.Info("knock denied", zap.String("user_id", uid.String()))
	}
}

func (h *Hub) handleKnockTimeout(knockID string) {
	kr, ok := h.knocks[knockID]
	if !ok {
		return
	}
	delete(h.knocks, knockID)
	delete(h.pending, kr.Client)

	f := false
	uid := kr.UserID
	kr.Client.Send(OutboundMessage{Type: MsgTypeKnockDenied, KnockID: kr.ID})
	kr.Client.closeWith(closeCodeDenied, "knock timed out")
	h.broadcastClients(OutboundMessage{Type: MsgTypeKnockResolved, KnockID: kr.ID, Approved: &f, UserID: &uid})
	h.logger.Info("knock timed out", zap.String("user_id", uid.String()))
}

// ─── broadcast helpers ────────────────────────────────────────────────────────

// broadcastClients sends to admitted clients only (not pending knockers).
func (h *Hub) broadcastClients(msg OutboundMessage) {
	for c := range h.clients {
		if !c.Send(msg) {
			// Buffer full — drop client.
			delete(h.clients, c)
			c.conn.Close()
		}
	}
}

// broadcastAll sends to admitted clients AND pending knockers.
func (h *Hub) broadcastAll(msg OutboundMessage) {
	h.broadcastClients(msg)
	for c := range h.pending {
		c.Send(msg)
	}
}

// broadcastExcept sends to all admitted clients except the sender.
func (h *Hub) broadcastExcept(sender *Client, msg OutboundMessage) {
	for c := range h.clients {
		if c != sender {
			if !c.Send(msg) {
				delete(h.clients, c)
				c.conn.Close()
			}
		}
	}
}

// ActiveCount returns the number of admitted participants (snapshot; called
// from outside the run goroutine only during shutdown/stats).
func (h *Hub) ActiveCount() int {
	return len(h.clients)
}

