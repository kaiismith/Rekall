package ws

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/ports"
)

// HubManager is a process-wide singleton that owns one Hub per active meeting.
// All map access is guarded by an RWMutex; Hub.Run goroutines own their own
// internal state exclusively.
//
// The manager owns a background context derived from the server lifecycle
// (NOT from any HTTP request) and feeds it to every hub's Run loop. This
// ensures hubs survive past the request that created them — without it, the
// HTTP handler returns, Gin cancels its request context, and the hub exits
// immediately, leaving the WS open with no goroutine to service it.
type HubManager struct {
	mu        sync.RWMutex
	hubs      map[uuid.UUID]*Hub
	chatRepo  ports.MeetingMessageRepository
	persister TranscriptPersister
	kat       KatLifecycle // optional; attached to every hub created via GetOrCreate
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewHubManager creates an empty HubManager. chatRepo is forwarded to every
// hub created via GetOrCreate so the chat handler can persist messages.
// persister, when non-nil, is also forwarded so the caption handler can
// record `final` segments to transcript_segments.
func NewHubManager(
	chatRepo ports.MeetingMessageRepository,
	persister TranscriptPersister,
	logger *zap.Logger,
) *HubManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &HubManager{
		hubs:      make(map[uuid.UUID]*Hub),
		chatRepo:  chatRepo,
		persister: persister,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// SetKat installs a KatLifecycle hook on the manager; subsequent
// GetOrCreate calls will attach it to every new hub. Safe to call once at
// boot before any hub exists. nil clears the hook (no Kat).
func (m *HubManager) SetKat(k KatLifecycle) {
	m.mu.Lock()
	m.kat = k
	m.mu.Unlock()
}

// Shutdown cancels the manager's background context, signalling every
// running hub to terminate. Safe to call multiple times.
func (m *HubManager) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

// GetOrCreate returns the existing Hub for meetingID, or starts a new one.
// hostID is stored in the hub so it can validate force_mute senders.
// onEnd is called (in a goroutine) when the hub detects the meeting has ended.
//
// The supplied ctx is unused for the hub's lifetime — the manager's own
// background context drives the Run loop. Past callers passed in the HTTP
// request context, which would cancel as soon as the handler returned and
// kill the hub. The parameter is kept for backwards compatibility.
func (m *HubManager) GetOrCreate(_ context.Context, meetingID, hostID uuid.UUID, onEnd func(uuid.UUID)) *Hub {
	m.mu.RLock()
	if h, ok := m.hubs[meetingID]; ok {
		m.mu.RUnlock()
		return h
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring the write lock.
	if h, ok := m.hubs[meetingID]; ok {
		return h
	}

	h := NewHub(meetingID, hostID, m.chatRepo, m.persister, func(id uuid.UUID) {
		m.remove(id)
		if onEnd != nil {
			onEnd(id)
		}
	}, m.logger)
	if m.kat != nil {
		h.SetKat(m.kat)
	}

	m.hubs[meetingID] = h
	go h.Run(m.ctx)
	m.logger.Info("hub created", zap.String("meeting_id", meetingID.String()))
	return h
}

// Get returns the Hub for meetingID, or nil if no hub exists.
func (m *HubManager) Get(meetingID uuid.UUID) *Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[meetingID]
}

// remove deletes the hub entry. Safe to call from any goroutine.
func (m *HubManager) remove(meetingID uuid.UUID) {
	m.mu.Lock()
	delete(m.hubs, meetingID)
	m.mu.Unlock()
	m.logger.Info("hub removed", zap.String("meeting_id", meetingID.String()))
}

// ActiveMeetingCount returns the number of meetings with live hubs.
func (m *HubManager) ActiveMeetingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.hubs)
}

// allHubs returns a snapshot copy of every live hub. Used by adapters that
// need to fan out to every hub (e.g. the Kat WSBroadcaster's SendToUser).
func (m *HubManager) allHubs() []*Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Hub, 0, len(m.hubs))
	for _, h := range m.hubs {
		out = append(out, h)
	}
	return out
}
