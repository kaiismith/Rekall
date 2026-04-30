package ports

import "github.com/google/uuid"

// WSBroadcaster is the Kat-side port over the WebSocket hub. Implementations
// live in backend/internal/interfaces/http/ws as a thin adapter over the
// existing Hub. The application service depends only on this interface so it
// can be tested without spinning up a hub.
type WSBroadcaster interface {
	// BroadcastToMeeting fan-outs msg to every client connected to the meeting.
	// A no-op when the meeting has no listeners.
	BroadcastToMeeting(meetingID uuid.UUID, msgType string, data any)

	// SendToUser delivers msg to the specific user's WS client(s). Used for
	// late-join replay where only the joining client should receive the
	// historical notes from the in-memory ring buffer.
	SendToUser(userID uuid.UUID, msgType string, data any)
}
