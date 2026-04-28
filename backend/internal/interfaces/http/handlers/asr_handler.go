package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ASRHandler exposes the call-scoped ASR endpoints. The issuer pointer may be
// nil — in that case the handler returns ASR_NOT_CONFIGURED for every call.
// This lets the wider repo build and serve without the C++ binary deployed.
//
// persister, when non-nil, enables the per-segment HTTP write path
// (POST /calls/:id/asr-session/:sid/segments). Solo Calls have no meeting WS
// hub, so the browser POSTs each `final` event directly through this handler.
type ASRHandler struct {
	issuer    *services.ASRTokenIssuer
	persister *services.TranscriptPersister
	logger    *zap.Logger
}

// NewASRHandler returns a handler. Pass nil for `issuer` to disable the
// feature without removing the routes. persister may also be nil — the
// per-segment endpoint then returns 503 ASR_NOT_CONFIGURED.
func NewASRHandler(
	issuer *services.ASRTokenIssuer,
	persister *services.TranscriptPersister,
	logger *zap.Logger,
) *ASRHandler {
	return &ASRHandler{issuer: issuer, persister: persister, logger: logger}
}

// Request handles POST /api/v1/calls/:id/asr-session.
//
// @Summary      Issue an ASR session token
// @Description  Registers an ASR session for the supplied call (which must be owned by the caller) and returns a short-lived, single-use JWT plus the WebSocket URL the browser uses to stream audio. The full Session_Token never appears in logs.
// @Tags         ASR
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                          true  "Call UUID"
// @Param        body  body      dto.ASRSessionRequest           false "Optional model/language/TTL preferences"
// @Success      201   {object}  dto.ASRSessionResponseEnvelope  "Session token issued"
// @Failure      400   {object}  dto.ErrorResponse               "Invalid call id or TTL"
// @Failure      401   {object}  dto.ErrorResponse               "Missing or invalid bearer"
// @Failure      403   {object}  dto.ErrorResponse               "Caller does not own the call"
// @Failure      409   {object}  dto.ErrorResponse               "Call has been finalised"
// @Failure      503   {object}  dto.ErrorResponse               "ASR service offline / at capacity / not configured"
// @Router       /api/v1/calls/{id}/asr-session [post]
func (h *ASRHandler) Request(c *gin.Context) {
	if h.issuer == nil {
		handlerhelpers.RespondError(c, h.logger,
			apperr.ServiceUnavailable("ASR_NOT_CONFIGURED",
				"asr service is not enabled in this environment", 0))
		return
	}

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

	var req dto.ASRSessionRequest
	if c.Request.ContentLength > 0 {
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			handlerhelpers.RespondError(c, h.logger, apperr.BadRequest(bindErr.Error()))
			return
		}
	}

	payload, err := h.issuer.Request(c.Request.Context(), services.RequestInput{
		CallerID:     callerID,
		CallID:       callID,
		ModelID:      req.ModelID,
		Language:     req.Language,
		RequestedTTL: time.Duration(req.TTLSeconds) * time.Second,
	})
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.JSON(http.StatusCreated, dto.ASRSessionResponseEnvelope{
		Success: true,
		Data: dto.ASRSessionPayload{
			SessionID:    payload.SessionID,
			SessionToken: payload.SessionToken,
			WsURL:        payload.WsURL,
			ExpiresAt:    payload.ExpiresAt,
			ModelID:      payload.ModelID,
			SampleRate:   payload.SampleRate,
			FrameFormat:  payload.FrameFormat,
		},
	})
}

// RequestForMeeting handles POST /api/v1/meetings/:code/asr-session.
//
// @Summary      Issue an ASR session token for a meeting
// @Description  Registers an ASR session for the meeting (which must have transcription_enabled=true and the caller must be an active participant) and returns a short-lived single-use JWT plus the WebSocket URL.
// @Tags         ASR
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        code  path      string                          true  "Meeting code (abc-defg-hij)"
// @Param        body  body      dto.ASRSessionRequest           false "Optional model/language/TTL preferences"
// @Success      201   {object}  dto.ASRSessionResponseEnvelope  "Session token issued"
// @Failure      403   {object}  dto.ErrorResponse               "Caller not in meeting OR transcription disabled for this meeting"
// @Failure      404   {object}  dto.ErrorResponse               "Meeting not found"
// @Failure      410   {object}  dto.ErrorResponse               "Meeting has ended"
// @Failure      503   {object}  dto.ErrorResponse               "ASR service offline / at capacity / not configured"
// @Router       /api/v1/meetings/{code}/asr-session [post]
func (h *ASRHandler) RequestForMeeting(c *gin.Context) {
	if h.issuer == nil {
		handlerhelpers.RespondError(c, h.logger,
			apperr.ServiceUnavailable("ASR_NOT_CONFIGURED",
				"asr service is not enabled in this environment", 0))
		return
	}
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
	if code == "" {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("missing meeting code"))
		return
	}
	var req dto.ASRSessionRequest
	if c.Request.ContentLength > 0 {
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			handlerhelpers.RespondError(c, h.logger, apperr.BadRequest(bindErr.Error()))
			return
		}
	}

	payload, err := h.issuer.RequestForMeeting(c.Request.Context(), services.MeetingRequestInput{
		CallerID:     callerID,
		MeetingCode:  code,
		ModelID:      req.ModelID,
		Language:     req.Language,
		RequestedTTL: time.Duration(req.TTLSeconds) * time.Second,
	})
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.JSON(http.StatusCreated, dto.ASRSessionResponseEnvelope{
		Success: true,
		Data: dto.ASRSessionPayload{
			SessionID:    payload.SessionID,
			SessionToken: payload.SessionToken,
			WsURL:        payload.WsURL,
			ExpiresAt:    payload.ExpiresAt,
			ModelID:      payload.ModelID,
			SampleRate:   payload.SampleRate,
			FrameFormat:  payload.FrameFormat,
		},
	})
}

// EndForMeeting handles POST /api/v1/meetings/:code/asr-session/end.
//
// @Summary      End an ASR session for a meeting
// @Tags         ASR
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        code  path      string                              true "Meeting code"
// @Param        body  body      dto.ASRSessionEndRequest            true "Session id"
// @Success      200   {object}  dto.ASRSessionEndResponseEnvelope   "Final transcript"
// @Router       /api/v1/meetings/{code}/asr-session/end [post]
func (h *ASRHandler) EndForMeeting(c *gin.Context) {
	if h.issuer == nil {
		handlerhelpers.RespondError(c, h.logger,
			apperr.ServiceUnavailable("ASR_NOT_CONFIGURED",
				"asr service is not enabled in this environment", 0))
		return
	}
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
	var body dto.ASRSessionEndRequest
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest(bindErr.Error()))
		return
	}
	sid, err := uuid.Parse(body.SessionID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("invalid session_id"))
		return
	}
	out, err := h.issuer.EndForMeeting(c.Request.Context(), callerID, code, sid)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, dto.ASRSessionEndResponseEnvelope{
		Success: true,
		Data: dto.ASRSessionEndPayload{
			FinalTranscript: out.FinalTranscript,
			FinalCount:      out.FinalCount,
		},
	})
}

// End handles POST /api/v1/calls/:id/asr-session/end.
//
// @Summary      End an ASR session
// @Description  Tells the ASR service to terminate the session and returns the stitched final transcript so the frontend can persist it.
// @Tags         ASR
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                              true "Call UUID"
// @Param        body  body      dto.ASRSessionEndRequest            true "Session id"
// @Success      200   {object}  dto.ASRSessionEndResponseEnvelope   "Final transcript"
// @Failure      400   {object}  dto.ErrorResponse                   "Invalid id"
// @Failure      401   {object}  dto.ErrorResponse                   "Unauthorized"
// @Failure      403   {object}  dto.ErrorResponse                   "Forbidden"
// @Failure      503   {object}  dto.ErrorResponse                   "ASR offline"
// @Router       /api/v1/calls/{id}/asr-session/end [post]
func (h *ASRHandler) End(c *gin.Context) {
	if h.issuer == nil {
		handlerhelpers.RespondError(c, h.logger,
			apperr.ServiceUnavailable("ASR_NOT_CONFIGURED",
				"asr service is not enabled in this environment", 0))
		return
	}
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
	var body dto.ASRSessionEndRequest
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest(bindErr.Error()))
		return
	}
	sid, err := uuid.Parse(body.SessionID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("invalid session_id"))
		return
	}
	out, err := h.issuer.End(c.Request.Context(), callerID, callID, sid)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, dto.ASRSessionEndResponseEnvelope{
		Success: true,
		Data: dto.ASRSessionEndPayload{
			FinalTranscript: out.FinalTranscript,
			FinalCount:      out.FinalCount,
		},
	})
}

// PostCallSegment handles POST /api/v1/calls/:id/asr-session/:session_id/segments.
//
// Solo Calls have no meeting WS hub, so the browser POSTs each `final` event
// directly here. The handler delegates validation + ownership checks to the
// TranscriptPersister; this layer only translates HTTP <-> service calls.
//
// @Summary      Persist a final transcript segment for a solo call
// @Description  Records one ASR `final` TranscriptEvent in transcript_segments. Idempotent on (session_id, segment_index) — a duplicate POST updates the row in place. Caller must own the call AND own the ASR session.
// @Tags         ASR
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id          path  string                       true  "Call UUID"
// @Param        session_id  path  string                       true  "ASR session UUID"
// @Param        body        body  dto.TranscriptSegmentRequest true  "Segment payload"
// @Success      204         "Segment persisted"
// @Failure      400         {object} dto.ErrorResponse "Invalid id, session_id, or body"
// @Failure      401         {object} dto.ErrorResponse "Unauthorized"
// @Failure      403         {object} dto.ErrorResponse "Caller does not own the call OR session"
// @Failure      404         {object} dto.ErrorResponse "Call or session not found"
// @Failure      409         {object} dto.ErrorResponse "Session not active"
// @Failure      503         {object} dto.ErrorResponse "Persistence not configured"
// @Router       /api/v1/calls/{id}/asr-session/{session_id}/segments [post]
func (h *ASRHandler) PostCallSegment(c *gin.Context) {
	if h.persister == nil {
		handlerhelpers.RespondError(c, h.logger,
			apperr.ServiceUnavailable("ASR_NOT_CONFIGURED",
				"transcript persistence is not enabled in this environment", 0))
		return
	}

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

	if _, err := uuid.Parse(c.Param("id")); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("invalid call id"))
		return
	}
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("invalid session_id"))
		return
	}

	var body dto.TranscriptSegmentRequest
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest(bindErr.Error()))
		return
	}

	words := make([]entities.WordTiming, 0, len(body.Words))
	for _, w := range body.Words {
		words = append(words, entities.WordTiming{
			Word:        w.Word,
			StartMs:     w.StartMs,
			EndMs:       w.EndMs,
			Probability: w.Probability,
		})
	}

	err = h.persister.RecordFinal(c.Request.Context(), services.RecordFinalInput{
		SessionID:    sessionID,
		CallerUserID: callerID,
		SegmentIndex: body.SegmentIndex,
		Text:         body.Text,
		Language:     body.Language,
		Confidence:   body.Confidence,
		StartMs:      body.StartMs,
		EndMs:        body.EndMs,
		Words:        words,
	})
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, services.ErrTranscriptInvalidSegment):
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("invalid segment payload"))
	case errors.Is(err, services.ErrTranscriptSessionNotFound):
		handlerhelpers.RespondError(c, h.logger, apperr.NotFound("TranscriptSession", sessionID.String()))
	case errors.Is(err, services.ErrTranscriptSessionNotOwned):
		handlerhelpers.RespondError(c, h.logger,
			apperr.ForbiddenCode("ASR_ACCESS_DENIED", "caller does not own this asr session"))
	case errors.Is(err, services.ErrTranscriptSessionClosed):
		handlerhelpers.RespondError(c, h.logger,
			apperr.ConflictCode("ASR_SESSION_CLOSED", "asr session is no longer active"))
	default:
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to persist segment"))
	}
}
