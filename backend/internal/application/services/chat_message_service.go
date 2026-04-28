package services

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// errNotAParticipant is the machine-readable 403 returned when a non-host,
// non-participant user asks for chat history. The specific code is exposed
// to clients so they can distinguish "you were never here" from a generic
// "forbidden".
var errNotAParticipant = &apperr.AppError{
	Status:  http.StatusForbidden,
	Code:    "NOT_A_PARTICIPANT",
	Message: "you are not a participant of this meeting",
}

// ChatMessageService orchestrates read-side access to meeting chat history.
// Writes flow through the WebSocket hub directly (handleChatMessage) because
// the hub is the serialisation point for in-room events and needs to persist
// the row before fanning out to peers.
type ChatMessageService struct {
	meetingRepo     ports.MeetingRepository
	participantRepo ports.MeetingParticipantRepository
	messageRepo     ports.MeetingMessageRepository
	logger          *zap.Logger
}

func NewChatMessageService(
	meetingRepo ports.MeetingRepository,
	participantRepo ports.MeetingParticipantRepository,
	messageRepo ports.MeetingMessageRepository,
	logger *zap.Logger,
) *ChatMessageService {
	return &ChatMessageService{
		meetingRepo:     meetingRepo,
		participantRepo: participantRepo,
		messageRepo:     messageRepo,
		logger:          applogger.WithComponent(logger, "chat_message_service"),
	}
}

// ListHistoryInput carries the parameters for a chat history read.
type ListHistoryInput struct {
	MeetingCode string
	RequesterID uuid.UUID
	Before      *time.Time
	Limit       int
}

// ListHistory returns chat messages for the identified meeting, enforcing
// the access rule: the requester must be the host, or must have an existing
// meeting_participants row (past or present).
func (s *ChatMessageService) ListHistory(
	ctx context.Context,
	input ListHistoryInput,
) ([]*entities.MeetingMessage, bool, error) {
	meeting, err := s.meetingRepo.GetByCode(ctx, input.MeetingCode)
	if err != nil {
		return nil, false, err
	}

	if meeting.HostID != input.RequesterID {
		p, err := s.participantRepo.GetByMeetingAndUser(ctx, meeting.ID, input.RequesterID)
		if err != nil {
			if apperr.IsNotFound(err) {
				return nil, false, errNotAParticipant
			}
			return nil, false, apperr.Internal("failed to check participant record")
		}
		if p == nil || p.JoinedAt == nil {
			return nil, false, errNotAParticipant
		}
	}

	msgs, hasMore, err := s.messageRepo.ListByMeeting(ctx, meeting.ID, input.Before, input.Limit)
	if err != nil {
		// Swallow the raw DB error — surface a generic 500 AppError so the
		// handler returns a clean payload without implementation details.
		catalog.ChatHistoryFetchFailed.Error(s.logger,
			zap.String("meeting_id", meeting.ID.String()),
			zap.String("requester_id", input.RequesterID.String()),
			zap.Error(err),
		)
		return nil, false, apperr.Internal("failed to load chat history")
	}
	return msgs, hasMore, nil
}
