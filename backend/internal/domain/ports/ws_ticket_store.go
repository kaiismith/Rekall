package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrTicketInvalid is returned by WSTicketStore.Consume for any failure mode
// (not-found, expired, or already-consumed). The distinct reasons are
// deliberately collapsed into one sentinel so the HTTP layer cannot leak
// which specific condition occurred.
var ErrTicketInvalid = errors.New("ws ticket invalid or expired")

// WSTicket is the stored payload for a meeting WebSocket ticket.
type WSTicket struct {
	Ticket      string
	MeetingCode string
	UserID      uuid.UUID
	ExpiresAt   time.Time
}

// WSTicketStore issues and consumes short-lived, single-use tickets used to
// authenticate the meeting WebSocket handshake. Implementations MUST make
// Consume atomic — exactly one caller may succeed for any given ticket value.
type WSTicketStore interface {
	Issue(ctx context.Context, meetingCode string, userID uuid.UUID, ttl time.Duration) (ticket string, expiresAt time.Time, err error)
	Consume(ctx context.Context, ticket string) (*WSTicket, error)
}
