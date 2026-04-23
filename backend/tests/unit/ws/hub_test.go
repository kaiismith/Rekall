package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// upgradeForTest creates a test WebSocket server and returns a client connection.
func upgradeForTest(t *testing.T, handler http.HandlerFunc) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

var testUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// makeClientPair creates a Hub with a connected test client. Returns the hub,
// the test-side conn, and a cancel function to stop the hub.
func makeClientPair(t *testing.T, userID uuid.UUID) (*wsHub.Hub, *websocket.Conn, context.CancelFunc) {
	t.Helper()

	meetingID := uuid.New()
	hub := wsHub.NewHub(meetingID, userID, nil, nil, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	var hubClient *wsHub.Client
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		hubClient = wsHub.NewClient(hub, conn, userID, "", "")
		hub.Register(hubClient, true, "")
		hubClient.Start()
	})

	clientConn := upgradeForTest(t, handler)
	// Give the hub goroutine time to process the register event.
	time.Sleep(50 * time.Millisecond)

	return hub, clientConn, cancel
}

func readJSON(t *testing.T, conn *websocket.Conn) wsHub.OutboundMessage {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	require.NoError(t, err)
	var msg wsHub.OutboundMessage
	require.NoError(t, json.Unmarshal(raw, &msg))
	return msg
}

// readUntil reads messages from conn until one matches the given type, returning it.
// Skips over any interleaved messages (e.g. room_state).
func readUntil(t *testing.T, conn *websocket.Conn, msgType string) wsHub.OutboundMessage {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		msg := readJSON(t, conn)
		if msg.Type == msgType {
			return msg
		}
	}
}

// drainAll reads and discards all pending messages until the deadline expires,
// then resets the deadline so subsequent reads work normally.
func drainAll(conn *websocket.Conn) {
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	conn.SetReadDeadline(time.Time{}) // clear deadline
}

// addAdmittedPeer adds an admitted participant to an existing hub and returns
// the test-side (dialer) connection. It drains the participant.joined message
// from the new peer's own conn and from every existingConns entry, so callers
// start with a clean read state regardless of how many peers are already in
// the hub.
func addAdmittedPeer(t *testing.T, hub *wsHub.Hub, existingConns ...*websocket.Conn) (peerID uuid.UUID, peerConn *websocket.Conn) {
	t.Helper()
	peerID = uuid.New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := wsHub.NewClient(hub, conn, peerID, "", "")
		hub.Register(c, true, "")
		c.Start()
	})
	peerConn = upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)

	// Drain participant.joined on the new peer's own conn (skip room_state etc.).
	readUntil(t, peerConn, wsHub.MsgTypeParticipantJoined)

	// Drain the participant.joined broadcast on every pre-existing conn.
	for _, ec := range existingConns {
		readUntil(t, ec, wsHub.MsgTypeParticipantJoined)
	}

	return peerID, peerConn
}

func TestHub_DirectJoin_BroadcastsParticipantJoined(t *testing.T) {
	userID := uuid.New()
	_, conn, cancel := makeClientPair(t, userID)
	defer cancel()

	// The client should receive a participant.joined message after registration.
	msg := readUntil(t, conn, wsHub.MsgTypeParticipantJoined)
	assert.Equal(t, wsHub.MsgTypeParticipantJoined, msg.Type)
	require.NotNil(t, msg.UserID)
	assert.Equal(t, userID, *msg.UserID)
}

func TestHub_Ping_ReturnsPong(t *testing.T) {
	userID := uuid.New()
	_, conn, cancel := makeClientPair(t, userID)
	defer cancel()

	// Drain until participant.joined is consumed.
	readUntil(t, conn, wsHub.MsgTypeParticipantJoined)

	// Send ping.
	ping, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypePing})
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, ping))

	msg := readUntil(t, conn, wsHub.MsgTypePong)
	assert.Equal(t, wsHub.MsgTypePong, msg.Type)
}

func TestHub_KnockFlow_ApproveAdmitsUser(t *testing.T) {
	// Create hub and an admitted participant (the one who will approve the knock).
	participantID := uuid.New()
	hub, participantConn, cancel := makeClientPair(t, participantID)
	defer cancel()

	// Drain until participant.joined for the admitted user.
	readUntil(t, participantConn, wsHub.MsgTypeParticipantJoined)

	// Now add a knocker.
	knockerID := uuid.New()
	knockID := "knock-test-123"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := wsHub.NewClient(hub, conn, knockerID, "", "")
		hub.Register(client, false, knockID)
		client.Start()
	})
	knockerConn := upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)

	// The admitted participant should receive a knock.requested message.
	knockMsg := readUntil(t, participantConn, wsHub.MsgTypeKnockRequested)
	assert.Equal(t, knockID, knockMsg.KnockID)

	// Approve the knock.
	approved := true
	respond, _ := json.Marshal(wsHub.InboundMessage{
		Type:     wsHub.MsgTypeKnockRespond,
		KnockID:  knockID,
		Approved: &approved,
	})
	require.NoError(t, participantConn.WriteMessage(websocket.TextMessage, respond))
	time.Sleep(50 * time.Millisecond)

	// Knocker should receive knock.approved.
	approvedMsg := readUntil(t, knockerConn, wsHub.MsgTypeKnockApproved)
	assert.Equal(t, wsHub.MsgTypeKnockApproved, approvedMsg.Type)

	// Hub should now have 2 clients.
	assert.Equal(t, 2, hub.ActiveCount())
}

func TestHubManager_GetOrCreate_ReturnsSameHub(t *testing.T) {
	manager := wsHub.NewHubManager(nil, zap.NewNop())
	meetingID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hostID := uuid.New()
	h1 := manager.GetOrCreate(ctx, meetingID, hostID, nil)
	h2 := manager.GetOrCreate(ctx, meetingID, hostID, nil)
	assert.Equal(t, h1, h2)
	assert.Equal(t, 1, manager.ActiveMeetingCount())
}

func TestHub_KnockFlow_DenyClosesKnocker(t *testing.T) {
	participantID := uuid.New()
	hub, participantConn, cancel := makeClientPair(t, participantID)
	defer cancel()
	readUntil(t, participantConn, wsHub.MsgTypeParticipantJoined) // drain participant.joined

	knockerID := uuid.New()
	knockID := "knock-deny-test"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := wsHub.NewClient(hub, conn, knockerID, "", "")
		hub.Register(client, false, knockID)
		client.Start()
	})
	knockerConn := upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)

	// Drain knock.requested on the participant side.
	readUntil(t, participantConn, wsHub.MsgTypeKnockRequested)

	// Participant denies.
	denied := false
	respond, _ := json.Marshal(wsHub.InboundMessage{
		Type:     wsHub.MsgTypeKnockRespond,
		KnockID:  knockID,
		Approved: &denied,
	})
	require.NoError(t, participantConn.WriteMessage(websocket.TextMessage, respond))
	time.Sleep(50 * time.Millisecond)

	// Knocker should receive knock.denied text message, then a close frame 4003.
	deniedMsg := readUntil(t, knockerConn, wsHub.MsgTypeKnockDenied)
	assert.Equal(t, wsHub.MsgTypeKnockDenied, deniedMsg.Type)

	// Next read should be the close frame (code 4003).
	knockerConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := knockerConn.ReadMessage()
	require.Error(t, err)
	closeErr, ok := err.(*websocket.CloseError)
	require.True(t, ok, "expected WebSocket close error, got %T: %v", err, err)
	assert.Equal(t, 4003, closeErr.Code)

	// Hub still has only 1 active participant (the original one).
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.ActiveCount())
}

func TestHub_KnockRace_SecondApproveDiscarded(t *testing.T) {
	// Two admitted participants both click Admit for the same knocker.
	// Only the first response should take effect; the second is silently discarded.
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined) // drain participant.joined for p1

	// Add a second admitted participant.
	p2ID := uuid.New()
	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		c := wsHub.NewClient(hub, conn, p2ID, "", "")
		hub.Register(c, true, "")
		c.Start()
	})
	upgradeForTest(t, handler2)
	time.Sleep(50 * time.Millisecond)
	// Drain participant.joined for p2 from p1's perspective.
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	// Add a knocker.
	knockID := "knock-race-test"
	knockerID := uuid.New()
	handler3 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		c := wsHub.NewClient(hub, conn, knockerID, "", "")
		hub.Register(c, false, knockID)
		c.Start()
	})
	upgradeForTest(t, handler3)
	time.Sleep(50 * time.Millisecond)

	// Drain knock.requested from p1.
	readUntil(t, p1Conn, wsHub.MsgTypeKnockRequested)

	// Both participants approve — only the first should take effect.
	approved := true
	respond, _ := json.Marshal(wsHub.InboundMessage{
		Type:     wsHub.MsgTypeKnockRespond,
		KnockID:  knockID,
		Approved: &approved,
	})
	// Send both approvals rapidly.
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, respond))
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, respond))
	time.Sleep(100 * time.Millisecond)

	// 3 clients total: p1, p2, knocker (admitted).
	assert.Equal(t, 3, hub.ActiveCount())
}

func TestHub_LastParticipantLeaves_TriggersOnEnd(t *testing.T) {
	meetingID := uuid.New()
	userID := uuid.New()
	onEndCalled := make(chan uuid.UUID, 1)

	hub := wsHub.NewHub(meetingID, userID, nil, func(id uuid.UUID) {
		onEndCalled <- id
	}, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		c := wsHub.NewClient(hub, conn, userID, "", "")
		hub.Register(c, true, "")
		c.Start()
	})
	conn := upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)
	readUntil(t, conn, wsHub.MsgTypeParticipantJoined) // drain participant.joined

	// Close the connection to trigger unregister.
	conn.Close()

	select {
	case id := <-onEndCalled:
		assert.Equal(t, meetingID, id)
	case <-time.After(2 * time.Second):
		t.Fatal("onEnd was not called after last participant left")
	}
}

func TestHub_SpeakingState_BroadcastToPeers(t *testing.T) {
	userID := uuid.New()
	hub, speakerConn, cancel := makeClientPair(t, userID)
	defer cancel()
	readUntil(t, speakerConn, wsHub.MsgTypeParticipantJoined) // drain speaker's own joined

	// Add a listener.
	_, listenerConn := addAdmittedPeer(t, hub, speakerConn)

	// Speaker sends speaking_state.
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:    wsHub.MsgTypeSpeakingState,
		Payload: json.RawMessage(`{"speaking":true}`),
	})
	require.NoError(t, speakerConn.WriteMessage(websocket.TextMessage, msg))

	// Listener should receive it (readUntil skips any interleaved messages).
	received := readUntil(t, listenerConn, wsHub.MsgTypeSpeakingState)
	assert.Equal(t, wsHub.MsgTypeSpeakingState, received.Type)
	require.NotNil(t, received.From)
	assert.Equal(t, userID, *received.From)
}

func TestHub_WebRTC_RelayToTargetPeerOnly(t *testing.T) {
	// Three admitted participants. P1 sends an offer addressed to P2.
	// P2 should receive it; P3 should NOT.
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined) // drain p1's own participant.joined

	// When p2 joins: p1Conn and p2Conn each get participant.joined → drained.
	p2ID, p2Conn := addAdmittedPeer(t, hub, p1Conn)
	// When p3 joins: p1Conn, p2Conn, and p3Conn each get participant.joined → drained.
	_, p3Conn := addAdmittedPeer(t, hub, p1Conn, p2Conn)

	// P1 sends an offer targeted at P2.
	offer, _ := json.Marshal(wsHub.InboundMessage{
		Type:    wsHub.MsgTypeOffer,
		To:      &p2ID,
		Payload: json.RawMessage(`{"sdp":"v=0..."}`),
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, offer))

	// P2 should receive the offer with From = P1 (readUntil skips buffered messages).
	received := readUntil(t, p2Conn, wsHub.MsgTypeOffer)
	require.NotNil(t, received.From)
	assert.Equal(t, p1ID, *received.From)

	// P3 must NOT receive anything (no message within deadline).
	p3Conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err := p3Conn.ReadMessage()
	require.Error(t, err, "P3 should not have received the offer")
}

func TestHub_CtxCancel_BroadcastsMeetingEnded(t *testing.T) {
	// Two admitted participants. Cancelling the hub context should broadcast
	// meeting.ended to both before the hub exits.
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	readJSON(t, p1Conn) // drain participant.joined

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	// Cancel the hub context — this triggers the ctx.Done() branch in Run.
	cancel()

	// Both connections should receive meeting.ended (may be preceded by buffered messages).
	p1Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		msg := readJSON(t, p1Conn)
		if msg.Type == wsHub.MsgTypeMeetingEnded {
			break
		}
	}

	p2Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		msg := readJSON(t, p2Conn)
		if msg.Type == wsHub.MsgTypeMeetingEnded {
			break
		}
	}

	// Hub's done channel should close.
	select {
	case <-hub.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("hub.Done() was not closed after context cancellation")
	}
}

func TestHub_ParticipantLeaves_BroadcastsParticipantLeft(t *testing.T) {
	// Two admitted participants. When one disconnects the other should receive
	// participant.left (not trigger onEnd, since one participant remains).
	onEndCalled := make(chan struct{}, 1)
	meetingID := uuid.New()
	hostID := uuid.New()
	hub := wsHub.NewHub(meetingID, hostID, nil, func(uuid.UUID) { onEndCalled <- struct{}{} }, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Register p1.
	p1ID := uuid.New()
	var p1ServerConn *websocket.Conn
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		p1ServerConn = conn
		c := wsHub.NewClient(hub, conn, p1ID, "", "")
		hub.Register(c, true, "")
		c.Start()
	})
	p1Conn := upgradeForTest(t, h1)
	time.Sleep(50 * time.Millisecond)
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined) // drain p1's own participant.joined

	// Register p2.
	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)
	_ = p1ServerConn

	// P1 disconnects.
	p1Conn.Close()
	time.Sleep(100 * time.Millisecond)

	// P2 should receive participant.left for P1.
	// There may be a room_state message before participant.left, so read until we find it.
	p2Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var leftMsg wsHub.OutboundMessage
	for i := 0; i < 5; i++ {
		msg := readJSON(t, p2Conn)
		if msg.Type == wsHub.MsgTypeParticipantLeft {
			leftMsg = msg
			break
		}
	}
	assert.Equal(t, wsHub.MsgTypeParticipantLeft, leftMsg.Type)
	require.NotNil(t, leftMsg.UserID)
	assert.Equal(t, p1ID, *leftMsg.UserID)

	// onEnd must NOT have been called (p2 is still in the hub).
	select {
	case <-onEndCalled:
		t.Fatal("onEnd should not be called while participants remain")
	default:
	}
	assert.Equal(t, 1, hub.ActiveCount())
}

func TestHub_WebRTC_NilOrUnknownTo_SilentlyDropped(t *testing.T) {
	// offer with nil To → silently dropped (sender receives nothing back).
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined) // drain participant.joined

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	// Drain any buffered messages before the test.
	time.Sleep(100 * time.Millisecond)
	drainAll(p1Conn)
	drainAll(p2Conn)

	// Case 1: To is omitted (nil).
	nilTo, _ := json.Marshal(wsHub.InboundMessage{
		Type:    wsHub.MsgTypeOffer,
		Payload: json.RawMessage(`{"sdp":"v=0..."}`),
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, nilTo))

	// Case 2: To references a userID not in the hub.
	unknown := uuid.New()
	unknownTo, _ := json.Marshal(wsHub.InboundMessage{
		Type:    wsHub.MsgTypeOffer,
		To:      &unknown,
		Payload: json.RawMessage(`{"sdp":"v=0..."}`),
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, unknownTo))

	time.Sleep(100 * time.Millisecond)

	// Neither p2 nor p1 should receive anything.
	p2Conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "p2 should not receive a message for nil/unknown To")

	p1Conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = p1Conn.ReadMessage()
	require.Error(t, err, "p1 should not receive an echo")
}

func TestHubManager_RemovesHubAfterLastParticipantLeaves(t *testing.T) {
	manager := wsHub.NewHubManager(nil, zap.NewNop())
	meetingID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hostID := uuid.New()
	onEndCalled := make(chan uuid.UUID, 1)
	hub := manager.GetOrCreate(ctx, meetingID, hostID, func(id uuid.UUID) { onEndCalled <- id })
	assert.Equal(t, 1, manager.ActiveMeetingCount())

	// Connect the sole participant via the manager's hub.
	userID := uuid.New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		c := wsHub.NewClient(hub, conn, userID, "", "")
		hub.Register(c, true, "")
		c.Start()
	})
	conn := upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)
	readUntil(t, conn, wsHub.MsgTypeParticipantJoined) // drain participant.joined

	// Disconnect — triggers onEnd → manager.remove.
	conn.Close()

	select {
	case id := <-onEndCalled:
		assert.Equal(t, meetingID, id)
	case <-time.After(2 * time.Second):
		t.Fatal("onEnd was not called after last participant left")
	}

	// Hub entry must have been removed from the manager.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, manager.ActiveMeetingCount())
	assert.Nil(t, manager.Get(meetingID))
}

func TestHub_KnockerDisconnects_BroadcastsCancelled(t *testing.T) {
	participantID := uuid.New()
	hub, participantConn, cancel := makeClientPair(t, participantID)
	defer cancel()
	readUntil(t, participantConn, wsHub.MsgTypeParticipantJoined) // drain participant.joined

	knockerID := uuid.New()
	knockID := "knock-cancel-test"
	var knockerConn *websocket.Conn

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		knockerConn = conn
		c := wsHub.NewClient(hub, conn, knockerID, "", "")
		hub.Register(c, false, knockID)
		c.Start()
	})
	upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)

	// Drain messages until we find knock.requested.
	participantConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		msg := readJSON(t, participantConn)
		if msg.Type == wsHub.MsgTypeKnockRequested {
			break
		}
	}

	// Knocker disconnects before being admitted.
	knockerConn.Close()
	time.Sleep(100 * time.Millisecond)

	// Participant should receive knock.cancelled (may be preceded by other messages).
	participantConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var cancelledMsg wsHub.OutboundMessage
	for i := 0; i < 5; i++ {
		m := readJSON(t, participantConn)
		if m.Type == wsHub.MsgTypeKnockCancelled {
			cancelledMsg = m
			break
		}
	}
	assert.Equal(t, wsHub.MsgTypeKnockCancelled, cancelledMsg.Type)
	assert.Equal(t, knockID, cancelledMsg.KnockID)
}
