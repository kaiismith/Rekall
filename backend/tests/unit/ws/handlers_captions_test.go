package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
)

// fakePersister records every RecordFinal/CloseSession call. Used to assert
// the WS hub fires the persistence path under the expected conditions.
type fakePersister struct {
	mu          sync.Mutex
	finals      []services.RecordFinalInput
	closes      []services.CloseSessionInput
	recordError error
}

func (f *fakePersister) RecordFinal(_ context.Context, in services.RecordFinalInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finals = append(f.finals, in)
	return f.recordError
}

func (f *fakePersister) CloseSession(_ context.Context, in services.CloseSessionInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closes = append(f.closes, in)
	return nil
}

func (f *fakePersister) finalsLen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.finals)
}

func (f *fakePersister) firstFinal() services.RecordFinalInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.finals[0]
}

// makeClientPairWithPersister mirrors makeClientPair but injects a persister.
func makeClientPairWithPersister(t *testing.T, userID uuid.UUID, p wsHub.TranscriptPersister) (*wsHub.Hub, *websocket.Conn, context.CancelFunc) {
	t.Helper()
	meetingID := uuid.New()
	hub := wsHub.NewHub(meetingID, userID, nil, p, nil, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
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

// waitFor polls the predicate up to 1s; returns true if it became true.
func waitFor(check func() bool) bool {
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return check()
}

func TestHub_CaptionChunk_PersistsFinalWithFullShape(t *testing.T) {
	p1ID := uuid.New()
	persister := &fakePersister{}

	hub, p1Conn, cancel := makeClientPairWithPersister(t, p1ID, persister)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	sid := uuid.New()
	idx := int32(0)
	startMs := int32(0)
	endMs := int32(1500)
	lang := "en"
	conf := float32(0.91)

	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:             wsHub.MsgTypeCaptionChunk,
		CaptionKind:      "final",
		CaptionText:      "hello world",
		CaptionSegmentID: "seg-0",
		ASRSessionID:     &sid,
		SegmentIndex:     &idx,
		StartMs:          &startMs,
		EndMs:            &endMs,
		Language:         &lang,
		Confidence:       &conf,
		Words: []entities.WordTiming{
			{Word: "hello", StartMs: 0, EndMs: 700, Probability: 0.93},
			{Word: "world", StartMs: 750, EndMs: 1500, Probability: 0.89},
		},
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	// Broadcast still happens.
	got := readUntil(t, p2Conn, wsHub.MsgTypeCaptionChunk)
	assert.Equal(t, "hello world", got.CaptionText)

	// Persistence runs off the broadcast critical path; poll briefly.
	require.True(t, waitFor(func() bool { return persister.finalsLen() == 1 }),
		"expected exactly one RecordFinal call after a final caption with full shape")

	rec := persister.firstFinal()
	assert.Equal(t, sid, rec.SessionID)
	assert.Equal(t, p1ID, rec.CallerUserID)
	assert.Equal(t, "hello world", rec.Text)
	assert.Equal(t, int32(1500), rec.EndMs)
	assert.Len(t, rec.Words, 2)
}

func TestHub_CaptionChunk_DoesNotPersistPartial(t *testing.T) {
	p1ID := uuid.New()
	persister := &fakePersister{}

	hub, p1Conn, cancel := makeClientPairWithPersister(t, p1ID, persister)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	sid := uuid.New()
	idx := int32(0)
	startMs := int32(0)
	endMs := int32(500)
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:             wsHub.MsgTypeCaptionChunk,
		CaptionKind:      "partial",
		CaptionText:      "hello",
		CaptionSegmentID: "seg-0",
		ASRSessionID:     &sid,
		SegmentIndex:     &idx,
		StartMs:          &startMs,
		EndMs:            &endMs,
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	// Broadcast still happens.
	readUntil(t, p2Conn, wsHub.MsgTypeCaptionChunk)

	// Give any spurious goroutine a beat to misfire, then assert no persistence.
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 0, persister.finalsLen(), "partials must NEVER be persisted")
}

func TestHub_CaptionChunk_DoesNotPersistLegacyShape(t *testing.T) {
	p1ID := uuid.New()
	persister := &fakePersister{}

	hub, p1Conn, cancel := makeClientPairWithPersister(t, p1ID, persister)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	// Legacy shape: no session_id / segment_index / start_ms / end_ms.
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:             wsHub.MsgTypeCaptionChunk,
		CaptionKind:      "final",
		CaptionText:      "hello world",
		CaptionSegmentID: "seg-0",
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))

	// Broadcast still happens — older clients keep working.
	got := readUntil(t, p2Conn, wsHub.MsgTypeCaptionChunk)
	assert.Equal(t, "hello world", got.CaptionText)

	// But persistence is skipped.
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 0, persister.finalsLen(), "legacy text-only shape must NOT trigger persistence")
}

// Compile-time check: services.TranscriptPersister structurally satisfies the
// WS-side TranscriptPersister port. Catches signature drift between the two.
var _ wsHub.TranscriptPersister = (*services.TranscriptPersister)(nil)

func (f *fakePersister) closesLen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.closes)
}

func (f *fakePersister) firstClose() services.CloseSessionInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closes[0]
}

// Disconnect path: when the WS drops AFTER a final segment with the
// persistence shape arrived, the hub MUST trigger CloseSession so the
// transcript_sessions row doesn't sit `active` until the cleanup tick.
func TestHub_Disconnect_ClosesActiveASRSession(t *testing.T) {
	p1ID := uuid.New()
	persister := &fakePersister{}

	hub, p1Conn, cancel := makeClientPairWithPersister(t, p1ID, persister)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)

	// Need a peer so the hub doesn't call onEnd → unregister; we want to
	// observe the disconnect path while the meeting stays open.
	_, p2Conn := addAdmittedPeer(t, hub, p1Conn)

	sid := uuid.New()
	idx := int32(0)
	startMs := int32(0)
	endMs := int32(1500)
	msg, _ := json.Marshal(wsHub.InboundMessage{
		Type:             wsHub.MsgTypeCaptionChunk,
		CaptionKind:      "final",
		CaptionText:      "hello world",
		CaptionSegmentID: "seg-0",
		ASRSessionID:     &sid,
		SegmentIndex:     &idx,
		StartMs:          &startMs,
		EndMs:            &endMs,
	})
	require.NoError(t, p1Conn.WriteMessage(websocket.TextMessage, msg))
	require.True(t, waitFor(func() bool { return persister.finalsLen() == 1 }))

	// Drop the WS without a graceful End call.
	require.NoError(t, p1Conn.Close())

	// Persister.CloseSession must fire with the captured session_id +
	// the disconnecting client's user id.
	require.True(t, waitFor(func() bool { return persister.closesLen() == 1 }),
		"expected CloseSession to fire on WS disconnect")
	got := persister.firstClose()
	assert.Equal(t, sid, got.SessionID)
	assert.Equal(t, p1ID, got.CallerUserID)
	assert.Equal(t, entities.TranscriptSessionStatusEnded, got.Status)

	// Drain the participant.left broadcast so the test cleanup is quiet.
	_ = p2Conn
}

// Disconnecting a client that NEVER sent a persistence-shape final
// (e.g. captions never enabled) must NOT call CloseSession.
func TestHub_Disconnect_NoActiveASRSession_Skips(t *testing.T) {
	p1ID := uuid.New()
	persister := &fakePersister{}

	hub, p1Conn, cancel := makeClientPairWithPersister(t, p1ID, persister)
	defer cancel()
	readUntil(t, p1Conn, wsHub.MsgTypeParticipantJoined)
	_, _ = addAdmittedPeer(t, hub, p1Conn)

	require.NoError(t, p1Conn.Close())

	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 0, persister.closesLen(),
		"clients with no captured ASR session must not trigger CloseSession")
}
