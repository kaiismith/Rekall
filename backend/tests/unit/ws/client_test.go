package ws_test

import (
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

// ─── Send buffer full ────────────────────────────────────────────────────────

// TestClient_Send_BufferFull fills the 256-slot send channel and then issues
// one more Send, which should return false (drop signal).
func TestClient_Send_BufferFull(t *testing.T) {
	// Create a real WebSocket conn via a test server so NewClient has something
	// non-nil. We never Start() the client so writePump doesn't drain the buffer.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn
		// Keep the server-side conn open for the duration of the test.
		time.Sleep(500 * time.Millisecond)
		_ = conn.Close()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	hub := wsHub.NewHub(uuid.New(), uuid.New(), nil, nil, zap.NewNop())
	c := wsHub.NewClient(hub, clientConn, uuid.New(), "", "")

	// Fill the send buffer (sendBufferSize = 256).
	msg := wsHub.OutboundMessage{Type: wsHub.MsgTypePong}
	filled := 0
	for i := 0; i < 300; i++ {
		if c.Send(msg) {
			filled++
		} else {
			break
		}
	}
	assert.Equal(t, 256, filled, "should fill exactly 256 slots")

	// The next Send should fail (buffer full).
	assert.False(t, c.Send(msg))
}

// TestClient_Send_JSONMarshalError exercises the json.Marshal failure branch.
// OutboundMessage contains standard types that always marshal, so this branch
// is effectively unreachable through the public API — skipping as an inherent
// limitation of the serialisable-by-construction message type.
