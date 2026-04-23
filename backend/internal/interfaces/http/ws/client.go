package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 64 * 1024 // 64 KiB

	// Send buffer size per client.
	sendBufferSize = 256

	// Maximum length (in runes) of a single chat message body.
	maxChatMessageLength = 2000
)

// Message types sent over the WebSocket connection.
const (
	// WebRTC signaling
	MsgTypeOffer        = "offer"
	MsgTypeAnswer       = "answer"
	MsgTypeIceCandidate = "ice_candidate"

	// Voice activity detection
	MsgTypeSpeakingState = "speaking_state"

	// Keepalive
	MsgTypePing = "ping"
	MsgTypePong = "pong"

	// Knock flow (waiting room)
	MsgTypeKnockRequested = "knock.requested"
	MsgTypeKnockRespond   = "knock.respond"
	MsgTypeKnockApproved  = "knock.approved"
	MsgTypeKnockDenied    = "knock.denied"
	MsgTypeKnockResolved  = "knock.resolved"
	MsgTypeKnockCancelled = "knock.cancelled"

	// Participant presence
	MsgTypeParticipantJoined = "participant.joined"
	MsgTypeParticipantLeft   = "participant.left"

	// Meeting lifecycle
	MsgTypeMeetingEnded = "meeting.ended"

	// In-room controls (new)
	MsgTypeMediaState    = "media_state"    // audio/video enabled toggle
	MsgTypeForceMute     = "force_mute"     // host mutes a participant
	MsgTypeEmojiReaction = "emoji_reaction" // floating emoji animation
	MsgTypeHandRaise     = "hand_raise"     // raise / lower hand
	MsgTypeLaserMove     = "laser_move"     // laser pointer position
	MsgTypeLaserStop     = "laser_stop"     // laser pointer deactivated
	MsgTypeRoomState     = "room_state"     // full snapshot sent on join

	// Chat
	MsgTypeChatMessage = "chat_message" // persistent in-room text message
)

// InboundMessage is a generic envelope for messages received from a client.
type InboundMessage struct {
	Type     string          `json:"type"`
	To       *uuid.UUID      `json:"to,omitempty"`        // peer target for WebRTC relay
	KnockID  string          `json:"knock_id,omitempty"`  // knock.respond
	Approved *bool           `json:"approved,omitempty"`  // knock.respond
	Payload  json.RawMessage `json:"payload,omitempty"`
	// In-room controls
	TargetID *uuid.UUID `json:"target_id,omitempty"` // force_mute
	Audio    *bool      `json:"audio,omitempty"`     // media_state
	Video    *bool      `json:"video,omitempty"`     // media_state
	Raised   *bool      `json:"raised,omitempty"`    // hand_raise
	X        *float64   `json:"x,omitempty"`         // laser_move
	Y        *float64   `json:"y,omitempty"`         // laser_move
	Emoji    string     `json:"emoji,omitempty"`     // emoji_reaction
	// Chat
	Body     string `json:"body,omitempty"`      // chat_message
	ClientID string `json:"client_id,omitempty"` // chat_message (echoed back)
}

// RoomStateParticipant is a snapshot of one participant's ephemeral state.
type RoomStateParticipant struct {
	UserID      string `json:"user_id"`
	FullName    string `json:"full_name,omitempty"`
	Initials    string `json:"initials,omitempty"`
	Audio       bool   `json:"audio"`
	Video       bool   `json:"video"`
	HandRaised  bool   `json:"hand_raised"`
	LaserActive bool   `json:"laser_active"`
}

// OutboundMessage is a generic envelope sent to a client.
type OutboundMessage struct {
	Type     string      `json:"type"`
	From     *uuid.UUID  `json:"from,omitempty"`
	KnockID  string      `json:"knock_id,omitempty"`
	Approved *bool       `json:"approved,omitempty"`
	UserID   *uuid.UUID  `json:"user_id,omitempty"`
	// Display info for participant.joined / chat broadcasts — populated from
	// the Client struct at join time. Omitempty so other message types stay
	// unchanged.
	FullName string      `json:"full_name,omitempty"`
	Initials string      `json:"initials,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
	// In-room controls
	Audio        *bool                  `json:"audio,omitempty"`
	Video        *bool                  `json:"video,omitempty"`
	Raised       *bool                  `json:"raised,omitempty"`
	X            *float64               `json:"x,omitempty"`
	Y            *float64               `json:"y,omitempty"`
	Emoji        string                 `json:"emoji,omitempty"`
	Participants []RoomStateParticipant `json:"participants,omitempty"`
	// Chat
	ID       *uuid.UUID `json:"id,omitempty"`        // chat_message (server-assigned)
	ClientID string     `json:"client_id,omitempty"` // chat_message (echoed back)
	Body     string     `json:"body,omitempty"`      // chat_message
	SentAt   *time.Time `json:"sent_at,omitempty"`   // chat_message
}

// closeSignal carries a WebSocket close frame to be sent by writePump.
// Routing the close frame through the send pipeline serialises all writes and
// eliminates concurrent-write races.
type closeSignal struct {
	code   int
	reason string
}

// Client represents a single WebSocket connection in a meeting hub.
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	closing chan closeSignal // signals writePump to send a close frame then exit
	UserID  uuid.UUID

	// Display name + initials — populated at connection time from the users
	// table. Included in participant.joined and room_state broadcasts so peers
	// can render sender names for chat messages without a per-user fetch.
	FullName string
	Initials string

	// chatSendTimestamps is a ring of recent chat_message send timestamps used
	// for server-side rate limiting. Owned exclusively by the hub run goroutine
	// (handleChatMessage) — no mutex needed.
	chatSendTimestamps []time.Time
}

// ChatRateLimit constants. Server-side enforcement is deliberately lenient so
// well-behaved clients (who enforce 3/2s locally) never trigger it; it only
// catches bad actors bypassing the UI.
const (
	ChatRateLimitWindow   = 10 * time.Second
	ChatRateLimitMessages = 10
)

// allowChat reports whether this client may send another chat_message right
// now. It prunes expired timestamps and records the attempt on allow.
func (c *Client) allowChat(now time.Time) bool {
	cutoff := now.Add(-ChatRateLimitWindow)
	// Prune in place (preserves capacity, no allocation in the steady state).
	j := 0
	for _, t := range c.chatSendTimestamps {
		if t.After(cutoff) {
			c.chatSendTimestamps[j] = t
			j++
		}
	}
	c.chatSendTimestamps = c.chatSendTimestamps[:j]

	if len(c.chatSendTimestamps) >= ChatRateLimitMessages {
		return false
	}
	c.chatSendTimestamps = append(c.chatSendTimestamps, now)
	return true
}

// NewClient creates a new Client and starts its read/write pumps.
// fullName and initials are optional display info surfaced to peers in
// participant.joined and room_state broadcasts.
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, fullName, initials string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, sendBufferSize),
		closing:  make(chan closeSignal, 1),
		UserID:   userID,
		FullName: fullName,
		Initials: initials,
	}
}

// Start launches the readPump and writePump goroutines.
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// Send enqueues a message for delivery to this client.
// Returns false if the send buffer is full (client will be dropped).
func (c *Client) Send(msg OutboundMessage) bool {
	b, err := json.Marshal(msg)
	if err != nil {
		return false
	}
	select {
	case c.send <- b:
		return true
	default:
		return false
	}
}

// readPump reads messages from the WebSocket and dispatches them to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg InboundMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		// Ping → respond with pong directly (no hub dispatch needed).
		if msg.Type == MsgTypePing {
			c.Send(OutboundMessage{Type: MsgTypePong})
			continue
		}

		c.hub.inbound <- inboundEnvelope{client: c, msg: msg}
	}
}

// writePump drains the send channel and writes messages to the WebSocket.
// All writes to the connection go through this goroutine, so there is never
// more than one writer — no mutex required.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case sig := <-c.closing:
			// Drain any buffered text messages so they arrive before the close frame.
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		drain:
			for {
				select {
				case msg, ok := <-c.send:
					if ok {
						c.conn.WriteMessage(websocket.TextMessage, msg) //nolint:errcheck
					}
				default:
					break drain
				}
			}
			// Send the close frame with the requested code, then exit.
			frame := websocket.FormatCloseMessage(sig.code, sig.reason)
			c.conn.WriteMessage(websocket.CloseMessage, frame) //nolint:errcheck
			return

		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// closeWith enqueues a WebSocket close frame for writePump to send. Safe to
// call from any goroutine — never writes to conn directly. writePump drains
// any buffered text messages before the close frame so they are all delivered.
func (c *Client) closeWith(code int, reason string) {
	select {
	case c.closing <- closeSignal{code: code, reason: reason}:
	default:
		// A close signal is already pending; just close the underlying conn to
		// unblock readPump so it can call unregister.
		c.conn.Close()
	}
}
