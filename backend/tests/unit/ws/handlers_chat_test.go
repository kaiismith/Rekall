package ws_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rekall/backend/internal/domain/entities"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Test chatRepo implementations ────────────────────────────────────────────

type recordingChatRepo struct {
	mu       sync.Mutex
	created  []*entities.MeetingMessage
	idToStub uuid.UUID
	timeStub time.Time
}

func newRecordingChatRepo() *recordingChatRepo {
	return &recordingChatRepo{
		idToStub: uuid.New(),
		timeStub: time.Date(2026, 4, 23, 14, 3, 17, 0, time.UTC),
	}
}

func (r *recordingChatRepo) Create(_ context.Context, m *entities.MeetingMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m.ID = r.idToStub
	m.SentAt = r.timeStub
	r.created = append(r.created, m)
	return nil
}

func (r *recordingChatRepo) ListByMeeting(_ context.Context, _ uuid.UUID, _ *time.Time, _ int) ([]*entities.MeetingMessage, bool, error) {
	return nil, false, nil
}

func (r *recordingChatRepo) CreatedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.created)
}

type failingChatRepo struct{}

func (f *failingChatRepo) Create(_ context.Context, _ *entities.MeetingMessage) error {
	return errors.New("db unavailable")
}
func (f *failingChatRepo) ListByMeeting(_ context.Context, _ uuid.UUID, _ *time.Time, _ int) ([]*entities.MeetingMessage, bool, error) {
	return nil, false, nil
}

type slowChatRepo struct{}

func (s *slowChatRepo) Create(ctx context.Context, _ *entities.MeetingMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return nil
	}
}
func (s *slowChatRepo) ListByMeeting(_ context.Context, _ uuid.UUID, _ *time.Time, _ int) ([]*entities.MeetingMessage, bool, error) {
	return nil, false, nil
}

// ─── Helper: hub + client with a custom chatRepo ─────────────────────────────

// makeClientPairWithChatRepo mirrors makeClientPair but lets the test supply
// a chatRepo. Returns the hub, test-side conn, and cancel function.
func makeClientPairWithChatRepo(
	t *testing.T,
	userID uuid.UUID,
	chatRepo interface {
		Create(context.Context, *entities.MeetingMessage) error
		ListByMeeting(context.Context, uuid.UUID, *time.Time, int) ([]*entities.MeetingMessage, bool, error)
	},
) (*wsHub.Hub, *websocket.Conn, context.CancelFunc) {
	t.Helper()

	meetingID := uuid.New()
	hub := wsHub.NewHub(meetingID, userID, chatRepo, nil, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := wsHub.NewClient(hub, conn, userID, "", "")
		hub.Register(c, true, "")
		c.Start()
	})
	clientConn := upgradeForTest(t, handler)
	time.Sleep(50 * time.Millisecond)
	return hub, clientConn, cancel
}

// ─── Happy path ──────────────────────────────────────────────────────────────

func TestHub_ChatMessage_PersistsAndBroadcasts(t *testing.T) {
	repo := newRecordingChatRepo()
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	clientID := "client-" + uuid.NewString()
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:     wsHub.MsgTypeChatMessage,
		ClientID: clientID,
		Body:     "hello team",
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	// Both sender and peer receive the broadcast.
	for _, conn := range []*websocket.Conn{p1Conn, p2Conn} {
		received := readUntil(t, conn, wsHub.MsgTypeChatMessage)
		require.NotNil(t, received.ID)
		assert.Equal(t, repo.idToStub, *received.ID)
		assert.Equal(t, clientID, received.ClientID)
		require.NotNil(t, received.UserID)
		assert.Equal(t, p1ID, *received.UserID)
		assert.Equal(t, "hello team", received.Body)
		require.NotNil(t, received.SentAt)
	}

	// DB write was recorded once.
	assert.Equal(t, 1, repo.CreatedCount())
	require.NotEmpty(t, repo.created)
	assert.Equal(t, "hello team", repo.created[0].Body)
	assert.Equal(t, p1ID, repo.created[0].UserID)
}

// ─── Validation ──────────────────────────────────────────────────────────────

func TestHub_ChatMessage_EmptyBodyDropped(t *testing.T) {
	repo := newRecordingChatRepo()
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeChatMessage, Body: ""})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "empty body should be silently dropped")
	assert.Equal(t, 0, repo.CreatedCount())
}

func TestHub_ChatMessage_WhitespaceBodyDropped(t *testing.T) {
	repo := newRecordingChatRepo()
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeChatMessage, Body: "   \t\n  "})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "whitespace-only body should be silently dropped")
	assert.Equal(t, 0, repo.CreatedCount())
}

func TestHub_ChatMessage_OverlongBodyDropped(t *testing.T) {
	repo := newRecordingChatRepo()
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	// 2001 runes — just over the limit.
	body := strings.Repeat("a", 2001)
	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeChatMessage, Body: body})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "2001-char body should be silently dropped")
	assert.Equal(t, 0, repo.CreatedCount())
}

func TestHub_ChatMessage_MaxLengthBodyAccepted(t *testing.T) {
	repo := newRecordingChatRepo()
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	body := strings.Repeat("b", 2000) // exactly on the limit
	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeChatMessage, Body: body})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	received := readUntil(t, p2Conn, wsHub.MsgTypeChatMessage)
	assert.Equal(t, body, received.Body)
	assert.Equal(t, 1, repo.CreatedCount())
}

// ─── DB failure paths ────────────────────────────────────────────────────────

func TestHub_ChatMessage_PersistFailureNoBroadcast(t *testing.T) {
	repo := &failingChatRepo{}
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeChatMessage, Body: "hi"})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(200 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "persist failure should prevent broadcast")
}

// ─── Nil repo (test harness path) ────────────────────────────────────────────

func TestHub_ChatMessage_NilRepoDropped(t *testing.T) {
	// No chat repo configured — handler must no-op.
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPair(t, p1ID)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeChatMessage, Body: "hi"})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	p2Conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "nil chatRepo should silently drop the message")
}

// ─── Non-admitted sender ─────────────────────────────────────────────────────

func TestHub_ChatMessage_PendingKnockerDropped(t *testing.T) {
	// A knocker (pending, not admitted) must not be able to send chat.
	repo := newRecordingChatRepo()
	hostID := uuid.New()
	hub, hostConn, cancel := makeClientPairWithChatRepo(t, hostID, repo)
	defer cancel()
	readUntil(t, hostConn, wsHub.MsgTypeParticipantJoined)

	knockerID := uuid.New()
	knockID := "knock-chat-test"
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
	readUntil(t, hostConn, wsHub.MsgTypeKnockRequested)

	// Knocker tries to chat — must be dropped.
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type: wsHub.MsgTypeChatMessage,
		Body: "hi from waiting room",
	})
	require.NoError(t, knockerConn.WriteMessage(websocket.TextMessage, msg))

	time.Sleep(150 * time.Millisecond)
	hostConn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	_, _, err := hostConn.ReadMessage()
	require.Error(t, err, "knocker's chat should not reach the host")
	assert.Equal(t, 0, repo.CreatedCount())
}

// ─── Server-side rate limit ──────────────────────────────────────────────────

func TestHub_ChatMessage_ServerSideRateLimit(t *testing.T) {
	// The server rate limit is ChatRateLimitMessages within ChatRateLimitWindow.
	// Sending > that count in quick succession should drop the overflow before
	// the DB insert.
	repo := newRecordingChatRepo()
	p1ID := uuid.New()
	_, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	// Fire ChatRateLimitMessages + 3 messages in rapid succession.
	total := wsHub.ChatRateLimitMessages + 3
	for i := 0; i < total; i++ {
		msg, _ := json.Marshal(wsHub.InboundMessage{
			Type:     wsHub.MsgTypeChatMessage,
			ClientID: uuid.NewString(),
			Body:     "spam",
		})
		require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))
	}

	// Let the hub drain.
	time.Sleep(300 * time.Millisecond)

	// Exactly the rate-limit count should have been persisted — the rest
	// are silently dropped.
	assert.Equal(t, wsHub.ChatRateLimitMessages, repo.CreatedCount())
}

// ─── Timeout path ────────────────────────────────────────────────────────────
//
// The handler uses a 5-second timeout. We don't actually wait that long; we
// verify that a sufficiently slow repo eventually times out by checking that
// no broadcast occurs within a short window. (The real timeout path is
// exercised by ctx.Deadline in handlers_chat.go and the handler's failure
// branch is already covered by failingChatRepo; this test documents the
// behavioural expectation under a slow DB.)
func TestHub_ChatMessage_SlowRepoDoesNotBroadcastImmediately(t *testing.T) {
	repo := &slowChatRepo{}
	p1ID := uuid.New()
	hub, p1Conn, cancel := makeClientPairWithChatRepo(t, p1ID, repo)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	msg, _ := json.Marshal(wsHub.InboundMessage{Type: wsHub.MsgTypeChatMessage, Body: "slow"})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	// Within 200 ms nothing should arrive.
	p2Conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err := p2Conn.ReadMessage()
	require.Error(t, err, "slow DB should not produce an immediate broadcast")
}
