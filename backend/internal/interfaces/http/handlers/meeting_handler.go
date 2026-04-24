package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	apperr "github.com/rekall/backend/pkg/errors"
	"go.uber.org/zap"
)

const wsTicketTTL = 60 * time.Second

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// MeetingHandler handles HTTP and WebSocket requests for the /meetings resource.
type MeetingHandler struct {
	service     *services.MeetingService
	chatService *services.ChatMessageService
	userService *services.UserService
	hubManager  *wsHub.HubManager
	ticketStore ports.WSTicketStore
	baseURL     string
	logger      *zap.Logger
}

// NewMeetingHandler creates a MeetingHandler with its required dependencies.
// chatService and userService may be nil — the chat endpoints will degrade
// gracefully (500 / missing names) rather than panic in test harnesses that
// don't wire them. ticketStore must be supplied when WebSocket routes are
// registered; nil is acceptable only for tests that do not exercise the WS
// upgrade or ticket endpoints.
func NewMeetingHandler(
	service *services.MeetingService,
	chatService *services.ChatMessageService,
	userService *services.UserService,
	hubManager *wsHub.HubManager,
	ticketStore ports.WSTicketStore,
	baseURL string,
	logger *zap.Logger,
) *MeetingHandler {
	return &MeetingHandler{
		service:     service,
		chatService: chatService,
		userService: userService,
		hubManager:  hubManager,
		ticketStore: ticketStore,
		baseURL:     baseURL,
		logger:      logger,
	}
}

// Create handles POST /api/v1/meetings.
//
// @Summary      Create a meeting
// @Description  Creates a new meeting room. Open meetings allow any authenticated user to join directly; private meetings restrict access to org/department members.
// @Tags         Meetings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateMeetingRequest          true  "Meeting creation parameters"
// @Success      201   {object}  dto.MeetingResponseEnvelope       "Meeting created"
// @Failure      400   {object}  dto.ErrorResponse                 "Invalid request body or host meeting limit reached"
// @Failure      401   {object}  dto.ErrorResponse                 "Missing or invalid token"
// @Failure      500   {object}  dto.ErrorResponse                 "Internal server error"
// @Router       /api/v1/meetings [post]
func (h *MeetingHandler) Create(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	if claims == nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("authentication required"))
		return
	}
	hostID, err := claims.SubjectAsUUID()
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("invalid token subject"))
		return
	}

	var req dto.CreateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.BadRequest(err.Error()))
		return
	}

	input := services.CreateMeetingInput{
		HostID:    hostID,
		Title:     req.Title,
		Type:      req.Type,
		ScopeType: req.ScopeType,
		ScopeID:   req.ScopeID,
	}

	meeting, err := h.service.CreateMeeting(c.Request.Context(), input)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusCreated, dto.OK(dto.MeetingFromEntity(meeting, h.baseURL)))
}

// GetByCode handles GET /api/v1/meetings/:code.
//
// @Summary      Get a meeting
// @Description  Returns a single meeting by its short code (e.g. abc-defg-hij). Does not require authentication.
// @Tags         Meetings
// @Produce      json
// @Param        code  path      string                      true  "Meeting code"  example(abc-defg-hij)
// @Success      200   {object}  dto.MeetingResponseEnvelope "Meeting record"
// @Failure      404   {object}  dto.ErrorResponse           "Meeting not found"
// @Failure      500   {object}  dto.ErrorResponse           "Internal server error"
// @Router       /api/v1/meetings/{code} [get]
func (h *MeetingHandler) GetByCode(c *gin.Context) {
	meeting, err := h.service.GetMeetingByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MeetingFromEntity(meeting, h.baseURL)))
}

// ListMine handles GET /api/v1/meetings/mine.
//
// @Summary      List my meetings
// @Description  Returns all meetings the authenticated user hosted or participated in, enriched with computed duration and up to 3 participant previews. Supports filtering by status and sorting by multiple criteria.
// @Tags         Meetings
// @Produce      json
// @Security     BearerAuth
// @Param        filter[status]  query     string  false  "Filter by status"  Enums(in_progress,complete,processing,failed)
// @Param        sort            query     string  false  "Sort order"        Enums(created_at_desc,created_at_asc,duration_desc,duration_asc,title_asc,title_desc)  default(created_at_desc)
// @Success      200  {object}  dto.MeetingListResponse  "List of meetings"
// @Failure      401  {object}  dto.ErrorResponse        "Missing or invalid token"
// @Failure      500  {object}  dto.ErrorResponse        "Internal server error"
// @Router       /api/v1/meetings/mine [get]
func (h *MeetingHandler) ListMine(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	if claims == nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("authentication required"))
		return
	}
	userID, err := claims.SubjectAsUUID()
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("invalid token subject"))
		return
	}

	statusFilter := c.Query("filter[status]")
	sort := c.DefaultQuery("sort", "created_at_desc")

	items, err := h.service.ListMeetingsWithMeta(c.Request.Context(), userID, statusFilter, sort)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	resp := make([]dto.MeetingResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, dto.MeetingFromListItem(item, h.baseURL))
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// End handles DELETE /api/v1/meetings/:code (host ends meeting).
//
// @Summary      End a meeting
// @Description  Ends an active meeting. Only the host may call this. Marks all participants as left and sets ended_at.
// @Tags         Meetings
// @Produce      json
// @Security     BearerAuth
// @Param        code  path      string             true  "Meeting code"  example(abc-defg-hij)
// @Success      200   {object}  dto.MessageResponse  "Meeting ended"
// @Failure      401   {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403   {object}  dto.ErrorResponse  "Caller is not the host"
// @Failure      404   {object}  dto.ErrorResponse  "Meeting not found"
// @Failure      500   {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/meetings/{code} [delete]
func (h *MeetingHandler) End(c *gin.Context) {
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

	meeting, err := h.service.GetMeetingByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	if meeting.HostID != callerID {
		handlerhelpers.RespondError(c, h.logger, apperr.Forbidden("only the host can end the meeting"))
		return
	}

	if err := h.service.EndMeeting(c.Request.Context(), meeting); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(map[string]string{"message": "meeting ended"}))
}

// IssueWSTicket handles POST /api/v1/meetings/:code/ws-ticket.
//
// @Summary      Issue a WebSocket ticket for a meeting
// @Description  Exchanges the caller's bearer token for a short-lived (60s), single-use ticket that authenticates the meeting WebSocket handshake. The ticket is bound to the calling user and the meeting code. Returns the ticket value and a fully-qualified ws_url path.
// @Tags         Meetings
// @Produce      json
// @Security     BearerAuth
// @Param        code  path      string                true  "Meeting code"  example(abc-defg-hij)
// @Success      201   {object}  dto.WSTicketResponse  "Ticket issued"
// @Failure      401   {object}  dto.ErrorResponse     "Missing or invalid bearer token"
// @Failure      404   {object}  dto.ErrorResponse     "Meeting not found (MEETING_NOT_FOUND)"
// @Failure      410   {object}  dto.ErrorResponse     "Meeting has ended (MEETING_ENDED)"
// @Failure      500   {object}  dto.ErrorResponse     "Internal server error"
// @Router       /api/v1/meetings/{code}/ws-ticket [post]
func (h *MeetingHandler) IssueWSTicket(c *gin.Context) {
	if h.ticketStore == nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("ws ticket store not configured"))
		return
	}
	claims := middleware.ClaimsFromContext(c)
	if claims == nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("authentication required"))
		return
	}
	userID, err := claims.SubjectAsUUID()
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unauthorized("invalid token subject"))
		return
	}

	code := c.Param("code")
	meeting, err := h.service.GetMeetingByCode(c.Request.Context(), code)
	if err != nil {
		if apperr.IsNotFound(err) {
			handlerhelpers.RespondError(c, h.logger, apperr.NotFoundCode("MEETING_NOT_FOUND", "meeting not found"))
			return
		}
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	if meeting.IsEnded() {
		handlerhelpers.RespondError(c, h.logger, apperr.Gone("MEETING_ENDED", "meeting has ended"))
		return
	}

	ticket, expiresAt, err := h.ticketStore.Issue(c.Request.Context(), code, userID, wsTicketTTL)
	if err != nil {
		h.logger.Error("ws ticket issue failed", zap.Error(err))
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("failed to issue ws ticket"))
		return
	}

	h.logger.Info("ws ticket issued",
		zap.String("user_id", userID.String()),
		zap.String("meeting_code", code),
		zap.String("ticket_prefix", safePrefix(ticket, 8)))

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, dto.WSTicketResponse{
		Success: true,
		Data: dto.WSTicketPayload{
			Ticket:    ticket,
			ExpiresAt: expiresAt,
			WSURL:     fmt.Sprintf("/api/v1/meetings/%s/ws?ticket=%s", code, ticket),
		},
	})
}

// safePrefix returns the first n chars of s, or s if it is shorter. Used to
// log a correlation slice of a ticket without leaking the full value.
func safePrefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

// Connect handles GET /api/v1/meetings/:code/ws — upgrades to WebSocket.
//
// @Summary      Join meeting via WebSocket
// @Description  Upgrades the connection to WebSocket and places the caller into the meeting hub. Callers authenticate by presenting a short-lived ticket obtained from POST /meetings/:code/ws-ticket; the ticket is single-use and is consumed atomically at upgrade time. Callers who are scope members join directly; others enter the waiting room (knock flow).
// @Tags         Meetings
// @Produce      json
// @Param        code    path      string             true  "Meeting code"   example(abc-defg-hij)
// @Param        ticket  query     string             true  "Short-lived WS ticket (see /ws-ticket)"
// @Success      101     {string}  string             "Switching Protocols — WebSocket connection established"
// @Failure      401     {object}  dto.ErrorResponse  "Missing (TICKET_REQUIRED) or invalid (TICKET_INVALID) or code-mismatched (TICKET_MISMATCH) ticket"
// @Failure      403     {object}  dto.ErrorResponse  "Access denied (meeting ended or at capacity)"
// @Failure      404     {object}  dto.ErrorResponse  "Meeting not found"
// @Failure      500     {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/meetings/{code}/ws [get]
func (h *MeetingHandler) Connect(c *gin.Context) {
	if h.ticketStore == nil {
		c.JSON(http.StatusInternalServerError, apperr.Internal("ws ticket store not configured"))
		return
	}

	ticket := c.Query("ticket")
	if ticket == "" {
		c.JSON(http.StatusUnauthorized, apperr.UnauthorizedCode("TICKET_REQUIRED", "ws ticket required"))
		return
	}

	payload, err := h.ticketStore.Consume(c.Request.Context(), ticket)
	if err != nil {
		h.logger.Info("ws ticket consume failed",
			zap.String("ticket_prefix", safePrefix(ticket, 8)),
			zap.Error(err))
		if errors.Is(err, ports.ErrTicketInvalid) {
			c.JSON(http.StatusUnauthorized, apperr.UnauthorizedCode("TICKET_INVALID", "ticket is invalid or has expired"))
			return
		}
		c.JSON(http.StatusInternalServerError, apperr.Internal("failed to consume ws ticket"))
		return
	}

	code := c.Param("code")
	if payload.MeetingCode != code {
		h.logger.Warn("ws ticket meeting-code mismatch",
			zap.String("ticket_prefix", safePrefix(ticket, 8)),
			zap.String("ticket_code", payload.MeetingCode),
			zap.String("url_code", code))
		c.JSON(http.StatusUnauthorized, apperr.UnauthorizedCode("TICKET_MISMATCH", "ticket is invalid or has expired"))
		return
	}

	callerID := payload.UserID

	meeting, err := h.service.GetMeetingByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	result, err := h.service.CanJoin(c.Request.Context(), meeting, callerID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}
	if result == services.CanJoinDenied {
		c.JSON(http.StatusForbidden, apperr.Forbidden("access denied"))
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("ws upgrade failed", zap.Error(err))
		return
	}
	h.logger.Info("ws connect: upgraded",
		zap.String("user_id", callerID.String()),
		zap.String("meeting_code", c.Param("code")),
		zap.String("can_join", string(result)),
	)

	hub := h.hubManager.GetOrCreate(c.Request.Context(), meeting.ID, meeting.HostID, h.onMeetingEmpty)

	isDirect := result == services.CanJoinDirect
	knockID := ""
	if !isDirect {
		knockID = fmt.Sprintf("knock-%s", uuid.New().String())
	}

	// Persist the join record for directly admitted users.
	if isDirect {
		role := entities.ParticipantRoleParticipant
		if meeting.HostID == callerID {
			role = entities.ParticipantRoleHost
		}
		if err := h.service.RecordJoin(c.Request.Context(), meeting, callerID, role); err != nil {
			h.logger.Error("ws connect: RecordJoin failed",
				zap.String("user_id", callerID.String()),
				zap.Error(err),
			)
			conn.Close()
			return
		}
		h.logger.Info("ws connect: RecordJoin OK",
			zap.String("user_id", callerID.String()),
			zap.String("role", role),
		)
	}

	// Resolve the caller's display info so peers see real names in
	// participant.joined and room_state broadcasts. A lookup failure is
	// non-fatal: we still admit the client, just with empty name/initials.
	var fullName, initials string
	if h.userService != nil {
		if u, err := h.userService.GetUser(c.Request.Context(), callerID); err == nil && u != nil {
			fullName = u.FullName
			initials = userInitials(u.FullName)
		}
	}

	client := wsHub.NewClient(hub, conn, callerID, fullName, initials)
	// Start the read/write pumps BEFORE Register — Register dispatches into the
	// hub's run loop, which may immediately call client.Send (room_state on
	// admitDirect). If writePump isn't running yet that send still queues fine,
	// but starting first eliminates any window where the hub could see the
	// client before it's drainable.
	client.Start()
	hub.Register(client, isDirect, knockID)
	h.logger.Info("ws connect: registered with hub",
		zap.String("user_id", callerID.String()),
		zap.Bool("is_direct", isDirect),
	)
}

// userInitials returns up to two uppercase letters from the user's full name
// (first letter of the first word + first letter of the last word). Falls back
// to "?" if the name is empty or contains no letters.
func userInitials(fullName string) string {
	fields := strings.Fields(fullName)
	if len(fields) == 0 {
		return "?"
	}
	first := firstLetter(fields[0])
	if len(fields) == 1 {
		if first == "" {
			return "?"
		}
		return first
	}
	last := firstLetter(fields[len(fields)-1])
	return first + last
}

func firstLetter(word string) string {
	for _, r := range word {
		if unicode.IsLetter(r) {
			return strings.ToUpper(string(r))
		}
	}
	return ""
}

// ListMessages handles GET /api/v1/meetings/:code/messages.
//
// @Summary      List chat messages for a meeting
// @Description  Returns the chat history for a meeting, ordered by sent_at ascending. Supports cursor-based pagination via the `before` query parameter (messages strictly older than the cursor are returned). Accessible to the host and to anyone who has ever been an admitted participant.
// @Tags         Meetings
// @Produce      json
// @Security     BearerAuth
// @Param        code    path      string  true   "Meeting code"                         example(abc-defg-hij)
// @Param        before  query     string  false  "RFC3339 cursor — returns messages strictly older"  example(2026-04-23T14:03:17Z)
// @Param        limit   query     int     false  "Page size (default 50, max 100)"      default(50)
// @Success      200  {object}  dto.ChatMessageListResponse  "Chat history page"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid `before` or `limit`"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Requester is not a participant (NOT_A_PARTICIPANT)"
// @Failure      404  {object}  dto.ErrorResponse  "Meeting not found"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/meetings/{code}/messages [get]
func (h *MeetingHandler) ListMessages(c *gin.Context) {
	if h.chatService == nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Internal("chat service not configured"))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("before must be an RFC3339 timestamp"))
			return
		}
		before = &t
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			handlerhelpers.RespondError(c, h.logger, apperr.BadRequest("limit must be a positive integer"))
			return
		}
		if n > 100 {
			n = 100
		}
		limit = n
	}

	msgs, hasMore, err := h.chatService.ListHistory(c.Request.Context(), services.ListHistoryInput{
		MeetingCode: c.Param("code"),
		RequesterID: callerID,
		Before:      before,
		Limit:       limit,
	})
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(dto.ChatMessageListPayload{
		Messages: dto.ChatMessagesFromEntities(msgs),
		HasMore:  hasMore,
	}))
}

// onMeetingEmpty is called by the hub when the last participant leaves.
// It ends the meeting in the database.
func (h *MeetingHandler) onMeetingEmpty(meetingID uuid.UUID) {
	ctx := context.Background()
	m, err := h.service.GetMeeting(ctx, meetingID)
	if err != nil {
		h.logger.Error("onMeetingEmpty: failed to load meeting",
			zap.Error(err),
			zap.String("meeting_id", meetingID.String()),
		)
		return
	}
	if err := h.service.EndMeeting(ctx, m); err != nil {
		h.logger.Error("onMeetingEmpty: failed to end meeting",
			zap.Error(err),
			zap.String("meeting_id", meetingID.String()),
		)
	}
}
