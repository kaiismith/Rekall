package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── media_state ──────────────────────────────────────────────────────────────

func TestHub_MediaState_BroadcastsToOthers(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	// p1 toggles audio off and video off.
	audioOff := false
	videoOff := false
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:  wsHub.MsgTypeMediaState,
		Audio: &audioOff,
		Video: &videoOff,
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	// p2 should receive media_state with p1's UserID and audio/video false.
	received := readUntil(t, p2Conn, wsHub.MsgTypeMediaState)
	require.NotNil(t, received.UserID)
	assert.Equal(t, p1ID, *received.UserID)
	require.NotNil(t, received.Audio)
	assert.False(t, *received.Audio)
	require.NotNil(t, received.Video)
	assert.False(t, *received.Video)
}

func TestHub_MediaState_OnlyAudio(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	audioOff := false
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:  wsHub.MsgTypeMediaState,
		Audio: &audioOff,
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	received := readUntil(t, p2Conn, wsHub.MsgTypeMediaState)
	require.NotNil(t, received.Audio)
	assert.False(t, *received.Audio)
	// Video was not provided → nil in outbound
	assert.Nil(t, received.Video)
}

// ─── force_mute ───────────────────────────────────────────────────────────────

func TestHub_ForceMute_HostSendsToTarget(t *testing.T) {
	// Host is p1, target is p2.
	hostID := uuid.New()
	hub, hostConn, cancel := makeClientPair(t, hostID)
	defer cancel()
	readUntil(t, hostConn, wsHub.MsgTypeParticipantJoined)

	targetID, targetConn := addAdmittedPeer(t, hub, hostConn)

	// Host sends force_mute targeting the other participant.
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:     wsHub.MsgTypeForceMute,
		TargetID: &targetID,
	})
	require.NoError(t, hostConn.WriteMessage(websocket.TextMessage, msg))

	// Target should receive force_mute.
	received := readUntil(t, targetConn, wsHub.MsgTypeForceMute)
	assert.Equal(t, wsHub.MsgTypeForceMute, received.Type)
}

func TestHub_ForceMute_NonHostIgnored(t *testing.T) {
	// Create a hub where hostID is NOT the client's userID. The client is a
	// regular participant, so force_mute should be silently dropped.
	hostID := uuid.New()
	clientID := uuid.New()
	meetingID := uuid.New()
	hub := wsHub.NewHub(meetingID, hostID, nil, nil, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	var clientConn *websocket.Conn
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		c := wsHub.NewClient(hub, conn, clientID, "", "")
		hub.Register(c, true, "")
		c.Start()
	})
	clientConn = upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)
	readUntil(t, clientConn, wsHub.MsgTypeParticipantJoined)

	otherID, otherConn := addAdmittedPeer(t, hub, clientConn)

	// Non-host tries to force_mute — should be dropped.
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:     wsHub.MsgTypeForceMute,
		TargetID: &otherID,
	})
	require.NoError(t, clientConn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	otherConn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := otherConn.ReadMessage()
	require.Error(t, err, "non-host force_mute should be silently dropped")
}

func TestHub_ForceMute_NilTargetIgnored(t *testing.T) {
	// Even the host is a no-op when TargetID is nil.
	hostID := uuid.New()
	hub, hostConn, cancel := makeClientPair(t, hostID)
	defer cancel()
	readUntil(t, hostConn, wsHub.MsgTypeParticipantJoined)

	_, otherConn := addAdmittedPeer(t, hub, hostConn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeForceMute})
	require.NoError(t, hostConn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	otherConn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := otherConn.ReadMessage()
	require.Error(t, err)
}

// ─── emoji_reaction ───────────────────────────────────────────────────────────

func TestHub_EmojiReaction_BroadcastsToAllIncludingSender(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	// p1 sends emoji_reaction.
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:  wsHub.MsgTypeEmojiReaction,
		Emoji: "🎉",
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	// Both p1 (sender) AND p2 should receive it.
	received2 := readUntil(t, p2Conn, wsHub.MsgTypeEmojiReaction)
	assert.Equal(t, "🎉", received2.Emoji)
	require.NotNil(t, received2.From)
	assert.Equal(t, p1ID, *received2.From)

	received1 := readUntil(t, p1Conn, wsHub.MsgTypeEmojiReaction)
	assert.Equal(t, "🎉", received1.Emoji)
}

func TestHub_EmojiReaction_EmptyStringIgnored(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeEmojiReaction})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "empty emoji should be ignored")
}

// ─── hand_raise ───────────────────────────────────────────────────────────────

func TestHub_HandRaise_Broadcast(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	raised := true
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:   wsHub.MsgTypeHandRaise,
		Raised: &raised,
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	received := readUntil(t, p2Conn, wsHub.MsgTypeHandRaise)
	require.NotNil(t, received.UserID)
	assert.Equal(t, p1ID, *received.UserID)
	require.NotNil(t, received.Raised)
	assert.True(t, *received.Raised)
}

func TestHub_HandRaise_NilRaisedIgnored(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeHandRaise})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "nil Raised field should be ignored")
}

// ─── laser_move / laser_stop ──────────────────────────────────────────────────

func TestHub_LaserMove_BroadcastsPosition(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	x, y := 0.5, 0.25
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type: wsHub.MsgTypeLaserMove,
		X:    &x,
		Y:    &y,
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	received := readUntil(t, p2Conn, wsHub.MsgTypeLaserMove)
	require.NotNil(t, received.UserID)
	assert.Equal(t, p1ID, *received.UserID)
	require.NotNil(t, received.X)
	assert.InDelta(t, 0.5, *received.X, 0.001)
	require.NotNil(t, received.Y)
	assert.InDelta(t, 0.25, *received.Y, 0.001)
}

func TestHub_LaserMove_NilCoordsIgnored(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	// Missing X and Y → drop.
	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeLaserMove})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err)
}

func TestHub_LaserMove_TakeoverFromPreviousOwner(t *testing.T) {
	// p1 owns laser → p2 takes over → p1 should receive laser_stop broadcast,
	// and p2's laser_move should also be broadcast.
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	x, y := 0.1, 0.1
	// p1 claims the laser.
	msg1, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeLaserMove, X: &x, Y: &y})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg1))
	// Drain p2's laser_move from p1.
	readUntil(t, p2Conn, wsHub.MsgTypeLaserMove)

	// p2 takes over.
	x2, y2 := 0.8, 0.9
	msg2, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeLaserMove, X: &x2, Y: &y2})
	require.NoError(t, p2Conn.WriteMessage(websocket.TextMessage, msg2))

	// p1 should now receive: laser_stop (for p1's own laser) then laser_move (from p2)
	received := readUntil(t, p1Conn, wsHub.MsgTypeLaserStop)
	assert.Equal(t, wsHub.MsgTypeLaserStop, received.Type)
}

// ─── broadcastAll with pending knocker ───────────────────────────────────────

func TestHub_BroadcastAll_NotifiesPendingKnockers(t *testing.T) {
	// Setup: one admitted participant, one pending knocker. When a third
	// party joins (or leaves), broadcastAll fires. The knocker — who is
	// pending (not admitted) — should still receive the message because
	// broadcastAll iterates over h.pending in addition to h.clients.
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	// Add a knocker — they sit in pending state.
	knockerID := uuid.New()
	knockID := "knock-bcall"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := testUpgrader.Upgrade(w, r, nil)
		c := wsHub.NewClient(hub, conn, knockerID, "", "")
		hub.Register(c, false, knockID)
		c.Start()
	})
	knockerConn := upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)

	// Drain knock.requested from p1.
	readUntil(t, p1Conn, wsHub.MsgTypeKnockRequested)

	// Now add a SECOND admitted participant. When they join, the hub
	// broadcasts participant.joined to ALL (admitted + pending).
	_, _ = addAdmittedPeer(t, hub, p1Conn)

	// The knocker (pending) should see the participant.joined broadcast.
	knockerConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	found := false
	for i := 0; i < 5; i++ {
		msg := readJSON(t, knockerConn)
		if msg.Type == wsHub.MsgTypeParticipantJoined {
			found = true
			break
		}
	}
	assert.True(t, found, "pending knocker should receive broadcastAll messages")
}

// ─── handleKnockTimeout ──────────────────────────────────────────────────────

func TestHub_KnockTimeout_TriggeredByInternalMessage(t *testing.T) {
	// The internal __knock_timeout__ message type can be processed via the
	// inbound channel (the same way addKnocker's timer fires). We simulate
	// this by sending the message from the participant side.
	participantID := uuid.New()
	hub, participantConn, cancel := makeClientPair(t, participantID)
	defer cancel()
	readUntil(t, participantConn, wsHub.MsgTypeParticipantJoined)

	// Register a knocker.
	knockerID := uuid.New()
	knockID := "knock-timeout-test"
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

	// Drain knock.requested on the participant side.
	readUntil(t, participantConn, wsHub.MsgTypeKnockRequested)

	// Send the internal __knock_timeout__ message from the participant.
	// The hub's handleInbound switches on msg.Type and calls handleKnockTimeout.
	timeoutMsg, _ := json.Marshal(wsHub.InboundMessage{
		Type:    "__knock_timeout__",
		KnockID: knockID,
	})
	require.NoError(t, participantConn.WriteMessage(websocket.TextMessage, timeoutMsg))

	// The knocker gets denied + closed; we don't assert what it reads since
	// the race between Send() and closeWith() can close before the flush
	// reaches our side. Silence "unused" lint.
	_ = knockerConn

	// Participant should get knock.resolved broadcast — this is the visible
	// side-effect of handleKnockTimeout running.
	resolved := readUntil(t, participantConn, wsHub.MsgTypeKnockResolved)
	assert.Equal(t, knockID, resolved.KnockID)
	require.NotNil(t, resolved.Approved)
	assert.False(t, *resolved.Approved)

	// Hub should no longer have the knocker pending.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.ActiveCount()) // only the participant remains
}

func TestHub_LaserStop_ClearsOwnership(t *testing.T) {
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	x, y := 0.5, 0.5
	// p1 claims the laser.
	mv, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeLaserMove, X: &x, Y: &y})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, mv))
	readUntil(t, p2Conn, wsHub.MsgTypeLaserMove)

	// p1 stops.
	stopMsg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeLaserStop})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, stopMsg))

	received := readUntil(t, p2Conn, wsHub.MsgTypeLaserStop)
	require.NotNil(t, received.UserID)
	assert.Equal(t, p1ID, *received.UserID)
}

