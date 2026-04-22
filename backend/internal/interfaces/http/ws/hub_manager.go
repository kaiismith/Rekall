package ws

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HubManager is a process-wide singleton that owns one Hub per active meeting.
// All map access is guarded by an RWMutex; Hub.Run goroutines own their own
// internal state exclusively.
type HubManager struct {
	mu     sync.RWMutex
	hubs   map[uuid.UUID]*Hub
	logger *zap.Logger
}

// NewHubManager creates an empty HubManager.
func NewHubManager(logger *zap.Logger) *HubManager {
	return &HubManager{
		hubs:   make(map[uuid.UUID]*Hub),
		logger: logger,
	}
}

// GetOrCreate returns the existing Hub for meetingID, or starts a new one.
// hostID is stored in the hub so it can validate force_mute senders.
// onEnd is called (in a goroutine) when the hub detects the meeting has ended.
func (m *HubManager) GetOrCreate(ctx context.Context, meetingID, hostID uuid.UUID, onEnd func(uuid.UUID)) *Hub {
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

	h := NewHub(meetingID, hostID, func(id uuid.UUID) {
		m.remove(id)
		if onEnd != nil {
			onEnd(id)
		}
	}, m.logger)

	m.hubs[meetingID] = h
	go h.Run(ctx)
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
