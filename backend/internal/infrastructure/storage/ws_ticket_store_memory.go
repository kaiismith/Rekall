package storage

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/ports"
)

const (
	sweepInterval   = 30 * time.Second
	defaultCapacity = 100_000
)

// MemoryWSTicketStore is an in-memory implementation of ports.WSTicketStore.
// It is safe for concurrent use; Consume is atomic via sync.Map.LoadAndDelete.
// A background sweeper removes expired entries every 30 seconds. When the
// store approaches capacity, Issue triggers an opportunistic sweep before
// inserting a new entry to keep memory bounded.
type MemoryWSTicketStore struct {
	data     sync.Map // map[string]*ports.WSTicket
	size     atomic.Int64
	capacity int64
	cancel   context.CancelFunc
	logger   *zap.Logger
}

// NewMemoryWSTicketStore constructs the in-memory store and starts its sweeper.
func NewMemoryWSTicketStore(logger *zap.Logger) *MemoryWSTicketStore {
	ctx, cancel := context.WithCancel(context.Background())
	s := &MemoryWSTicketStore{
		capacity: defaultCapacity,
		cancel:   cancel,
		logger:   logger,
	}
	go s.sweepLoop(ctx)
	return s
}

// Close stops the background sweeper.
func (s *MemoryWSTicketStore) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

// Len returns the approximate number of outstanding tickets.
func (s *MemoryWSTicketStore) Len() int64 {
	return s.size.Load()
}

// Issue mints a new cryptographically random ticket, stores it with the
// supplied meeting code and user ID, and returns the value plus expiry.
func (s *MemoryWSTicketStore) Issue(
	ctx context.Context,
	meetingCode string,
	userID uuid.UUID,
	ttl time.Duration,
) (string, time.Time, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	ticket := base64.RawURLEncoding.EncodeToString(buf)
	expiresAt := time.Now().Add(ttl)

	if s.size.Load() >= s.capacity {
		s.sweepExpired(time.Now())
	}

	s.data.Store(ticket, &ports.WSTicket{
		Ticket:      ticket,
		MeetingCode: meetingCode,
		UserID:      userID,
		ExpiresAt:   expiresAt,
	})
	s.size.Add(1)
	return ticket, expiresAt, nil
}

// Consume atomically fetches and deletes the ticket. Returns ErrTicketInvalid
// on miss or expiry; under concurrent callers for the same ticket value,
// exactly one caller receives the payload and the rest receive ErrTicketInvalid.
func (s *MemoryWSTicketStore) Consume(ctx context.Context, ticket string) (*ports.WSTicket, error) {
	raw, ok := s.data.LoadAndDelete(ticket)
	if !ok {
		return nil, ports.ErrTicketInvalid
	}
	s.size.Add(-1)
	t := raw.(*ports.WSTicket)
	if time.Now().After(t.ExpiresAt) {
		return nil, ports.ErrTicketInvalid
	}
	return t, nil
}

func (s *MemoryWSTicketStore) sweepLoop(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.sweepExpired(now)
		}
	}
}

func (s *MemoryWSTicketStore) sweepExpired(now time.Time) {
	s.data.Range(func(k, v any) bool {
		entry, ok := v.(*ports.WSTicket)
		if !ok {
			return true
		}
		if entry.ExpiresAt.Before(now) {
			if _, loaded := s.data.LoadAndDelete(k); loaded {
				s.size.Add(-1)
			}
		}
		return true
	})
}
