package storage_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newStore(t *testing.T) *storage.MemoryWSTicketStore {
	t.Helper()
	s := storage.NewMemoryWSTicketStore(zap.NewNop())
	t.Cleanup(s.Close)
	return s
}

func TestIssueConsumeSuccess(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	uid := uuid.New()

	ticket, expiresAt, err := s.Issue(ctx, "abc-defg-hij", uid, time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, ticket)
	assert.True(t, expiresAt.After(time.Now()))

	payload, err := s.Consume(ctx, ticket)
	require.NoError(t, err)
	assert.Equal(t, "abc-defg-hij", payload.MeetingCode)
	assert.Equal(t, uid, payload.UserID)
	assert.Equal(t, ticket, payload.Ticket)
}

func TestConsumeTwiceReturnsInvalid(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	ticket, _, err := s.Issue(ctx, "abc", uuid.New(), time.Minute)
	require.NoError(t, err)

	_, err = s.Consume(ctx, ticket)
	require.NoError(t, err)

	_, err = s.Consume(ctx, ticket)
	assert.ErrorIs(t, err, ports.ErrTicketInvalid)
}

func TestConsumeExpiredReturnsInvalid(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	ticket, _, err := s.Issue(ctx, "abc", uuid.New(), 1*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = s.Consume(ctx, ticket)
	assert.ErrorIs(t, err, ports.ErrTicketInvalid)
}

func TestConsumeUnknownReturnsInvalid(t *testing.T) {
	s := newStore(t)
	_, err := s.Consume(context.Background(), "never-issued")
	assert.ErrorIs(t, err, ports.ErrTicketInvalid)
}

// TestConsumeConcurrentExactlyOne spawns 100 goroutines racing to consume the
// same ticket; exactly one SHALL succeed.
func TestConsumeConcurrentExactlyOne(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	ticket, _, err := s.Issue(ctx, "abc", uuid.New(), time.Minute)
	require.NoError(t, err)

	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Consume(ctx, ticket); err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ports.ErrTicketInvalid) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), successes.Load(), "exactly one goroutine must succeed")
}

func TestIssueGeneratesUniqueTickets(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		ticket, _, err := s.Issue(ctx, "abc", uuid.New(), time.Minute)
		require.NoError(t, err)
		_, dup := seen[ticket]
		assert.False(t, dup, "ticket collision at iteration %d", i)
		seen[ticket] = struct{}{}
	}
}

func TestLenTracksOutstandingTickets(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	assert.Equal(t, int64(0), s.Len())

	t1, _, _ := s.Issue(ctx, "a", uuid.New(), time.Minute)
	t2, _, _ := s.Issue(ctx, "b", uuid.New(), time.Minute)
	assert.Equal(t, int64(2), s.Len())

	_, _ = s.Consume(ctx, t1)
	assert.Equal(t, int64(1), s.Len())

	_, _ = s.Consume(ctx, t2)
	assert.Equal(t, int64(0), s.Len())
}
