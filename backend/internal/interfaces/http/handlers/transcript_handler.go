package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	apperr "github.com/rekall/backend/pkg/errors"
)

// TranscriptHandler exposes the read endpoints for stored transcripts:
//
//	GET /api/v1/calls/:id/transcript
//	GET /api/v1/meetings/:code/transcript
//
// Authorization: Calls require call.user_id == caller. Meetings require the
// caller to have a meeting_participants row (join history; left_at not
// required to be NULL — past participants can still re-read the transcript).
type TranscriptHandler struct {
	transcriptRepo  ports.TranscriptRepository
	callRepo        ports.CallRepository
	meetingRepo     ports.MeetingRepository
	participantRepo ports.MeetingParticipantRepository
	logger          *zap.Logger
}

// NewTranscriptHandler wires the read handler.
func NewTranscriptHandler(
	transcriptRepo ports.TranscriptRepository,
	callRepo ports.CallRepository,
	meetingRepo ports.MeetingRepository,
	participantRepo ports.MeetingParticipantRepository,
	logger *zap.Logger,
) *TranscriptHandler {
	return &TranscriptHandler{
		transcriptRepo:  transcriptRepo,
		callRepo:        callRepo,
		meetingRepo:     meetingRepo,
		participantRepo: participantRepo,
		logger:          logger,
	}
}

// GetCallTranscript handles GET /api/v1/calls/:id/transcript.
//
// @Summary      Read a call's stored transcript
// @Description  Returns the latest ASR session bound to the call plus every persisted `final` segment ordered by (segment_started_at, segment_index). Caller must own the call.
// @Tags         Transcripts
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                       true  "Call UUID"
// @Success      200  {object}  dto.CallTranscriptEnvelope   "Transcript"
// @Failure      401  {object}  dto.ErrorResponse            "Unauthorized"
// @Failure      403  {object}  dto.ErrorResponse            "Forbidden"
// @Failure      404  {object}  dto.ErrorResponse            "Call not found"
// @Router       /api/v1/calls/{id}/transcript [get]
func (h *TranscriptHandler) GetCallTranscript(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	if claims == nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("authentication required"))
		return
	}
	callerID, err := claims.SubjectAsUUID()
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("invalid token subject"))
		return
	}

	callID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("invalid call id"))
		return
	}

	call, err := h.callRepo.GetByID(c.Request.Context(), callID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	if call.UserID != callerID {
		handlerhelpers.RespondError(c, h.logger,
			apperr.ForbiddenCode("CALL_ACCESS_DENIED", "caller does not own this call"))
		return
	}

	sessions, err := h.transcriptRepo.ListSessionsByCall(c.Request.Context(), callID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to load transcript sessions"))
		return
	}
	// Use a generous per-page so a typical call (< 5k segments) returns in one shot.
	segs, _, err := h.transcriptRepo.ListSegmentsByCall(c.Request.Context(), callID, 1, 5000)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to load transcript segments"))
		return
	}

	var sessionDTO *dto.TranscriptSessionDTO
	if len(sessions) > 0 {
		// Calls today produce at most one session per recording. If we ever
		// add multi-attempt recordings, this picks the most recent.
		latest := sessions[len(sessions)-1]
		s := dto.FromTranscriptSession(latest)
		sessionDTO = &s
	}

	segDTOs := make([]dto.TranscriptSegmentDTO, 0, len(segs))
	for _, s := range segs {
		segDTOs = append(segDTOs, dto.FromTranscriptSegment(s))
	}

	c.JSON(http.StatusOK, dto.CallTranscriptEnvelope{
		Success: true,
		Data: dto.CallTranscriptResponse{
			Session:  sessionDTO,
			Segments: segDTOs,
		},
	})
}

// GetMeetingTranscript handles GET /api/v1/meetings/:code/transcript.
//
// @Summary      Read a meeting's stored transcript
// @Description  Returns every ASR session bound to the meeting (one per participant who enabled captions) plus all persisted `final` segments. Caller must have been a participant of the meeting.
// @Tags         Transcripts
// @Produce      json
// @Security     BearerAuth
// @Param        code  path      string                          true  "Meeting code"
// @Success      200   {object}  dto.MeetingTranscriptEnvelope   "Transcript"
// @Failure      401   {object}  dto.ErrorResponse               "Unauthorized"
// @Failure      403   {object}  dto.ErrorResponse               "Forbidden"
// @Failure      404   {object}  dto.ErrorResponse               "Meeting not found"
// @Router       /api/v1/meetings/{code}/transcript [get]
func (h *TranscriptHandler) GetMeetingTranscript(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	if claims == nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("authentication required"))
		return
	}
	callerID, err := claims.SubjectAsUUID()
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("invalid token subject"))
		return
	}

	code := c.Param("code")
	meeting, err := h.meetingRepo.GetByCode(c.Request.Context(), code)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	// Past or present participation grants read. Hosts who never joined still
	// see the transcript via the host_id check.
	if meeting.HostID != callerID {
		part, err := h.participantRepo.GetByMeetingAndUser(c.Request.Context(), meeting.ID, callerID)
		if err != nil || part == nil {
			handlerhelpers.RespondError(c, h.logger,
				apperr.ForbiddenCode("MEETING_ACCESS_DENIED",
					"caller is not a participant of this meeting"))
			return
		}
	}

	sessions, err := h.transcriptRepo.ListSessionsByMeeting(c.Request.Context(), meeting.ID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to load transcript sessions"))
		return
	}
	segs, _, err := h.transcriptRepo.ListSegmentsByMeeting(c.Request.Context(), meeting.ID, 1, 10000)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to load transcript segments"))
		return
	}

	sessionDTOs := make([]dto.TranscriptSessionDTO, 0, len(sessions))
	for _, s := range sessions {
		sessionDTOs = append(sessionDTOs, dto.FromTranscriptSession(s))
	}
	segDTOs := make([]dto.TranscriptSegmentDTO, 0, len(segs))
	for _, s := range segs {
		segDTOs = append(segDTOs, dto.FromTranscriptSegment(s))
	}

	c.JSON(http.StatusOK, dto.MeetingTranscriptEnvelope{
		Success: true,
		Data: dto.MeetingTranscriptResponse{
			Sessions: sessionDTOs,
			Segments: segDTOs,
		},
	})
}
