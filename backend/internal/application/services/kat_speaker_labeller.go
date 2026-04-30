package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/rekall/backend/internal/domain/ports"
)

// KatSpeakerLabeller resolves a user_id to a privacy-aware human-readable
// label ("Sarah K.") for the Kat prompt seam. Used in place of UUIDs so the
// prompt stays deterministic and the model never sees raw user identifiers.
//
// Lifetime is tied to the cohort entry it serves — when the cohort is
// removed the labeller is dropped along with it (no global cache, no spill
// to disk; this matches the no-persistence stance for Kat overall).
type KatSpeakerLabeller struct {
	users ports.UserRepository

	mu       sync.Mutex
	cache    map[uuid.UUID]string
	fallback map[uuid.UUID]int // deterministic 1-based index for unresolved users
	nextIdx  int
}

// NewKatSpeakerLabeller constructs a labeller backed by the user repository.
func NewKatSpeakerLabeller(users ports.UserRepository) *KatSpeakerLabeller {
	return &KatSpeakerLabeller{
		users:    users,
		cache:    make(map[uuid.UUID]string),
		fallback: make(map[uuid.UUID]int),
	}
}

// Label returns a label for userID, computing it on first sight. On a user
// lookup failure the labeller falls back to "Speaker N" with N stable for the
// labeller's lifetime — same user always gets the same fallback label.
func (l *KatSpeakerLabeller) Label(ctx context.Context, userID uuid.UUID) string {
	l.mu.Lock()
	if cached, ok := l.cache[userID]; ok {
		l.mu.Unlock()
		return cached
	}
	l.mu.Unlock()

	label, err := l.lookup(ctx, userID)
	if err != nil || label == "" {
		label = l.fallbackLabel(userID)
	}

	l.mu.Lock()
	l.cache[userID] = label
	l.mu.Unlock()
	return label
}

func (l *KatSpeakerLabeller) lookup(ctx context.Context, userID uuid.UUID) (string, error) {
	u, err := l.users.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return formatSpeakerLabel(u.FullName), nil
}

func (l *KatSpeakerLabeller) fallbackLabel(userID uuid.UUID) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if idx, ok := l.fallback[userID]; ok {
		return fmt.Sprintf("Speaker %d", idx)
	}
	l.nextIdx++
	l.fallback[userID] = l.nextIdx
	return fmt.Sprintf("Speaker %d", l.nextIdx)
}

// formatSpeakerLabel produces "<First> <L>." from a full name, falling back
// to the first whole word when only one word is present.
func formatSpeakerLabel(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return ""
	}
	first := parts[0]
	if len(parts) == 1 {
		return first
	}
	last := parts[len(parts)-1]
	initial := string([]rune(last)[:1])
	return first + " " + strings.ToUpper(initial) + "."
}
