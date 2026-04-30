package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/rekall/backend/pkg/logger/catalog"
)

const (
	defaultTranscriptPerPage = 50
	maxTranscriptPerPage     = 200
)

// parseTranscriptPagination reads `page` and `per_page` query params with
// silent fallback to defaults on missing or unparseable values. `per_page` is
// clamped to [1, maxTranscriptPerPage]; `page` is clamped to >= 1.
func parseTranscriptPagination(c *gin.Context) (page, perPage int) {
	page = 1
	perPage = defaultTranscriptPerPage
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			switch {
			case n < 1:
				perPage = defaultTranscriptPerPage
			case n > maxTranscriptPerPage:
				perPage = maxTranscriptPerPage
			default:
				perPage = n
			}
		}
	}
	return
}

// buildTranscriptPagination computes the response pagination block from
// (page, perPage, total). When perPage is zero (defensive — never happens
// with parseTranscriptPagination), TotalPages is zero and HasMore is false.
func buildTranscriptPagination(page, perPage, total int) dto.TranscriptPagination {
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	return dto.TranscriptPagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

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
// @Description  Returns the latest ASR session bound to the call plus the page of persisted `final` segments specified by the `page` / `per_page` query parameters, ordered by (segment_started_at, segment_index). Caller must own the call. Default `per_page` is 50; max is 200.
// @Tags         Transcripts
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      string                       true   "Call UUID"
// @Param        page      query     int                          false  "Page number (1-indexed). Defaults to 1; unparseable values silently treated as 1."
// @Param        per_page  query     int                          false  "Page size. Defaults to 50; clamped to [1, 200]."
// @Success      200  {object}  dto.CallTranscriptEnvelope   "Transcript page"
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
		catalog.RecordTranscriptAccessDenied.Warn(h.logger,
			zap.String("call_id", callID.String()),
			zap.String("caller_id", callerID.String()),
			zap.String("reason", "not_owner"),
		)
		handlerhelpers.RespondError(c, h.logger,
			apperr.ForbiddenCode("CALL_ACCESS_DENIED", "caller does not own this call"))
		return
	}

	page, perPage := parseTranscriptPagination(c)

	sessions, err := h.transcriptRepo.ListSessionsByCall(c.Request.Context(), callID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to load transcript sessions"))
		return
	}
	segs, total, err := h.transcriptRepo.ListSegmentsByCall(c.Request.Context(), callID, page, perPage)
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
			Session:    sessionDTO,
			Segments:   segDTOs,
			Pagination: buildTranscriptPagination(page, perPage, total),
		},
	})
}

// GetMeetingTranscript handles GET /api/v1/meetings/:code/transcript.
//
// @Summary      Read a meeting's stored transcript
// @Description  Returns every ASR session bound to the meeting (one per participant who enabled captions) plus the page of persisted `final` segments specified by the `page` / `per_page` query parameters. Caller must have been a participant of the meeting. Sessions are returned in full on every page; only `segments` is paginated.
// @Tags         Transcripts
// @Produce      json
// @Security     BearerAuth
// @Param        code      path      string                          true   "Meeting code"
// @Param        page      query     int                             false  "Page number (1-indexed). Defaults to 1; unparseable values silently treated as 1."
// @Param        per_page  query     int                             false  "Page size. Defaults to 50; clamped to [1, 200]."
// @Success      200   {object}  dto.MeetingTranscriptEnvelope   "Transcript page"
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
			catalog.RecordTranscriptAccessDenied.Warn(h.logger,
				zap.String("meeting_id", meeting.ID.String()),
				zap.String("caller_id", callerID.String()),
				zap.String("reason", "not_host_not_participant"),
			)
			handlerhelpers.RespondError(c, h.logger,
				apperr.ForbiddenCode("MEETING_ACCESS_DENIED",
					"caller is not a participant of this meeting"))
			return
		}
	}

	page, perPage := parseTranscriptPagination(c)

	sessions, err := h.transcriptRepo.ListSessionsByMeeting(c.Request.Context(), meeting.ID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to load transcript sessions"))
		return
	}
	segs, total, err := h.transcriptRepo.ListSegmentsByMeeting(c.Request.Context(), meeting.ID, page, perPage)
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
			Sessions:   sessionDTOs,
			Segments:   segDTOs,
			Pagination: buildTranscriptPagination(page, perPage, total),
		},
	})
}
