package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	httputils "github.com/rekall/backend/internal/interfaces/http/utils"
	apperr "github.com/rekall/backend/pkg/errors"
)

// CallHandler handles HTTP requests for the /calls resource.
type CallHandler struct {
	service *services.CallService
	logger  *zap.Logger
}

// NewCallHandler creates a CallHandler with its required dependencies.
func NewCallHandler(service *services.CallService, logger *zap.Logger) *CallHandler {
	return &CallHandler{service: service, logger: logger}
}

// List handles GET /api/v1/calls.
//
// @Summary      List calls
// @Description  Returns a paginated list of call records sorted by creation date descending. Optionally filter by status or user ID.
// @Tags         Calls
// @Produce      json
// @Security     BearerAuth
// @Param        page                query     int     false  "Page number (1-based)"                     minimum(1)  default(1)
// @Param        per_page            query     int     false  "Number of items per page"                  minimum(1)  maximum(100) default(20)
// @Param        status              query     string  false  "Filter by call status"                     Enums(pending,processing,done,failed)
// @Param        user_id             query     string  false  "Filter by user UUID"                       format(uuid)
// @Param        filter[scope_type]  query     string  false  "Filter by scope"                           Enums(organization,department,open)
// @Param        filter[scope_id]    query     string  false  "UUID of the org or dept; required when scope_type is organization or department"
// @Success      200  {object}  dto.CallListResponse  "Paginated list of calls"
// @Failure      400  {object}  dto.ErrorResponse     "Invalid query parameter (e.g. malformed UUID or scope params)"
// @Failure      401  {object}  dto.ErrorResponse     "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse     "Caller is not a member of the requested scope"
// @Failure      500  {object}  dto.ErrorResponse     "Internal server error"
// @Router       /api/v1/calls [get]
func (h *CallHandler) List(c *gin.Context) {
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

	page := httputils.QueryInt(c, "page", 1)
	perPage := httputils.QueryInt(c, "per_page", 20)

	var filter ports.ListCallsFilter

	if uidStr := c.Query("user_id"); uidStr != "" {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("invalid user_id format"))
			return
		}
		filter.UserID = &uid
	}

	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	scope, err := dto.ParseScopeFilter(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	filter.Scope = scope

	calls, total, err := h.service.ListCalls(c.Request.Context(), callerID, filter, page, perPage)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated(calls, page, perPage, total))
}

// Get handles GET /api/v1/calls/:id.
//
// @Summary      Get a call
// @Description  Returns a single call record by its UUID.
// @Tags         Calls
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                    true  "Call UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000002)
// @Success      200  {object}  dto.CallResponseEnvelope  "Call record"
// @Failure      400  {object}  dto.ErrorResponse         "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      404  {object}  dto.ErrorResponse         "Call not found (CALL_NOT_FOUND)"
// @Failure      500  {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/calls/{id} [get]
func (h *CallHandler) Get(c *gin.Context) {
	id, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	call, err := h.service.GetCall(c.Request.Context(), id)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(call))
}

// Create handles POST /api/v1/calls.
//
// @Summary      Create a call
// @Description  Creates a new call record in `pending` status. The call can be updated later with a recording URL, transcript, and timing once processing completes.
// @Tags         Calls
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateCallRequest     true  "Call payload"
// @Success      201   {object}  dto.CallResponseEnvelope  "Call created"
// @Failure      401   {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      422   {object}  dto.ErrorResponse         "Validation error"
// @Failure      500   {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/calls [post]
func (h *CallHandler) Create(c *gin.Context) {
	var req dto.CreateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	call, err := h.service.CreateCall(c.Request.Context(), services.CreateCallInput{
		UserID:    req.UserID,
		Title:     req.Title,
		Metadata:  req.Metadata,
		ScopeType: req.ScopeType,
		ScopeID:   req.ScopeID,
	})
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusCreated, dto.OK(call))
}

// Update handles PATCH /api/v1/calls/:id.
//
// @Summary      Update a call
// @Description  Partially updates a call record. All fields are optional — only the fields present in the request body are modified.
// @Tags         Calls
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "Call UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000002)
// @Param        body  body      dto.UpdateCallRequest     true  "Fields to update (all optional)"
// @Success      200   {object}  dto.CallResponseEnvelope  "Updated call record"
// @Failure      400   {object}  dto.ErrorResponse         "Invalid UUID format"
// @Failure      401   {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      404   {object}  dto.ErrorResponse         "Call not found (CALL_NOT_FOUND)"
// @Failure      422   {object}  dto.ErrorResponse         "Validation error"
// @Failure      500   {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/calls/{id} [patch]
func (h *CallHandler) Update(c *gin.Context) {
	id, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.UpdateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	call, err := h.service.UpdateCall(c.Request.Context(), id, services.UpdateCallInput{
		Title:        req.Title,
		Status:       req.Status,
		RecordingURL: req.RecordingURL,
		Transcript:   req.Transcript,
		StartedAt:    req.StartedAt,
		EndedAt:      req.EndedAt,
		Metadata:     req.Metadata,
	})
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(call))
}

// Delete handles DELETE /api/v1/calls/:id.
//
// @Summary      Delete a call
// @Description  Soft-deletes a call record. The record is retained in the database but excluded from all future list and get queries.
// @Tags         Calls
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Call UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000002)
// @Success      204  "Call deleted"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      404  {object}  dto.ErrorResponse  "Call not found (CALL_NOT_FOUND)"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/calls/{id} [delete]
func (h *CallHandler) Delete(c *gin.Context) {
	id, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.DeleteCall(c.Request.Context(), id); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
