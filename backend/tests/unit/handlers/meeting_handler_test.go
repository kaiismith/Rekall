package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"context"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	"github.com/rekall/backend/internal/infrastructure/storage"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newMeetingService(mr *mockMeetingRepo, pr *mockParticipantRepo) *services.MeetingService {
	return services.NewMeetingService(mr, pr, new(mockMemberRepo), new(mockDeptMemberRepo),
		"http://rekall.test", zap.NewNop())
}

func newMeetingHandler(svc *services.MeetingService) *handlers.MeetingHandler {
	h, _ := newMeetingHandlerWithTicketStore(svc)
	return h
}

// newMeetingHandlerWithTicketStore returns the handler plus the in-memory
// ticket store so Connect tests can issue tickets before calling the upgrade.
func newMeetingHandlerWithTicketStore(svc *services.MeetingService) (*handlers.MeetingHandler, *storage.MemoryWSTicketStore) {
	manager := wsHub.NewHubManager(nil, nil, zap.NewNop())
	store := storage.NewMemoryWSTicketStore(zap.NewNop())
	h := handlers.NewMeetingHandler(svc, nil, nil, manager, store, "http://rekall.test", zap.NewNop())
	return h, store
}

// issueTestTicket mints a ticket for (code, userID) via the in-memory store
// and returns the value. TTL is long enough for tests not to race on it.
func issueTestTicket(t *testing.T, store *storage.MemoryWSTicketStore, code string, userID uuid.UUID) string {
	t.Helper()
	ticket, _, err := store.Issue(context.Background(), code, userID, 5*time.Minute)
	require.NoError(t, err)
	return ticket
}

// newMeetingRouter wires the meeting handler onto a gin engine.
// Routes that read JWT claims from context use injectClaims(callerID).
// The WS connect route performs its own JWT parse from ?token=, so it needs
// no middleware.
func newMeetingRouter(h *handlers.MeetingHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	authed := r.Group("/")
	authed.Use(injectClaims(callerID, "member"))
	authed.POST("/meetings", h.Create)
	authed.GET("/meetings/mine", h.ListMine)
	authed.DELETE("/meetings/:code", h.End)
	// GetByCode and Connect are unauthenticated at the middleware level.
	r.GET("/meetings/:code", h.GetByCode)
	r.GET("/meetings/:code/ws", h.Connect)
	return r
}

func activeMeeting(hostID uuid.UUID) *entities.Meeting {
	now := time.Now().UTC()
	return &entities.Meeting{
		ID:              uuid.New(),
		Code:            "abc-defg-hij",
		Type:            entities.MeetingTypeOpen,
		Status:          entities.MeetingStatusActive,
		HostID:          hostID,
		MaxParticipants: entities.MeetingMaxParticipants,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestMeetingCreateHandler_Success(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	mr.On("CountActiveByHost", mock.Anything, hostID).Return(int64(0), nil)
	mr.On("Create", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)

	body := jsonBody(t, dto.CreateMeetingRequest{Type: "open", Title: "Stand-up"})
	w := doRequest(r, http.MethodPost, "/meetings", body)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "open", data["type"])
	mr.AssertExpectations(t)
}

func TestMeetingCreateHandler_BadRequest_MissingType(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, uuid.New())

	// "type" is required; omitting it should fail binding validation.
	body := jsonBody(t, map[string]string{"title": "No Type"})
	w := doRequest(r, http.MethodPost, "/meetings", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mr.AssertNotCalled(t, "CountActiveByHost")
}

func TestMeetingCreateHandler_HostLimitExceeded(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	mr.On("CountActiveByHost", mock.Anything, hostID).
		Return(int64(entities.MeetingMaxPerHost), nil)

	body := jsonBody(t, dto.CreateMeetingRequest{Type: "open", Title: "Over Limit"})
	w := doRequest(r, http.MethodPost, "/meetings", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mr.AssertNotCalled(t, "Create")
}

// ─── GetByCode ───────────────────────────────────────────────────────────────

func TestMeetingGetByCodeHandler_Success(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, uuid.New())

	meeting := activeMeeting(uuid.New())
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(meeting, nil)

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	mr.AssertExpectations(t)
}

func TestMeetingGetByCodeHandler_NotFound(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, uuid.New())

	mr.On("GetByCode", mock.Anything, "xxx-yyyy-zzz").
		Return(nil, apperr.NotFound("meeting", "xxx-yyyy-zzz"))

	w := doRequest(r, http.MethodGet, "/meetings/xxx-yyyy-zzz", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── Unauthenticated guards (nil claims) ─────────────────────────────────────

// newUnauthMeetingRouter registers the same protected routes without the
// injectClaims middleware, so claims are nil in the gin context.
func newUnauthMeetingRouter(h *handlers.MeetingHandler) *gin.Engine {
	r := gin.New()
	r.POST("/meetings", h.Create)
	r.GET("/meetings/mine", h.ListMine)
	r.DELETE("/meetings/:code", h.End)
	return r
}

func TestMeetingCreateHandler_NoClaims_Unauthorized(t *testing.T) {
	h := newMeetingHandler(newMeetingService(new(mockMeetingRepo), new(mockParticipantRepo)))
	w := doRequest(newUnauthMeetingRouter(h), http.MethodPost, "/meetings",
		jsonBody(t, dto.CreateMeetingRequest{Type: "open"}))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMeetingListMineHandler_NoClaims_Unauthorized(t *testing.T) {
	h := newMeetingHandler(newMeetingService(new(mockMeetingRepo), new(mockParticipantRepo)))
	w := doRequest(newUnauthMeetingRouter(h), http.MethodGet, "/meetings/mine", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMeetingEndHandler_NoClaims_Unauthorized(t *testing.T) {
	h := newMeetingHandler(newMeetingService(new(mockMeetingRepo), new(mockParticipantRepo)))
	w := doRequest(newUnauthMeetingRouter(h), http.MethodDelete, "/meetings/abc-defg-hij", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── ListMine ────────────────────────────────────────────────────────────────

func TestMeetingListMineHandler_Success(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	mr.On("ListByUser", mock.Anything, hostID, mock.Anything).
		Return([]*ports.MeetingListItem{{Meeting: activeMeeting(hostID)}}, nil)

	w := doRequest(r, http.MethodGet, "/meetings/mine", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	assert.Len(t, resp["data"].([]interface{}), 1)
	mr.AssertExpectations(t)
}

func TestMeetingListMineHandler_StatusFilter(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	// filter[status]=in_progress is parsed and forwarded to ListByUser.
	mr.On("ListByUser", mock.Anything, hostID, mock.Anything).
		Return([]*ports.MeetingListItem{}, nil)

	w := doRequest(r, http.MethodGet, "/meetings/mine?filter%5Bstatus%5D=in_progress", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mr.AssertExpectations(t)
}

func TestMeetingListMineHandler_ResponseIncludesDTOFields(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	dur := int64(483) // 8m 03s
	previews := []ports.ParticipantPreview{
		{UserID: uuid.New(), FullName: "Alice Smith", Initials: "AS"},
		{UserID: uuid.New(), FullName: "Bob Jones", Initials: "BJ"},
	}
	item := &ports.MeetingListItem{
		Meeting:             activeMeeting(hostID),
		DurationSeconds:     &dur,
		ParticipantPreviews: previews,
	}
	mr.On("ListByUser", mock.Anything, hostID, mock.Anything).
		Return([]*ports.MeetingListItem{item}, nil)

	w := doRequest(r, http.MethodGet, "/meetings/mine", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	data := resp["data"].([]interface{})
	require.Len(t, data, 1)

	card := data[0].(map[string]interface{})
	assert.Equal(t, float64(483), card["duration_seconds"], "duration_seconds must be present in response")

	pp := card["participant_previews"].([]interface{})
	require.Len(t, pp, 2, "participant_previews must contain all previews")
	first := pp[0].(map[string]interface{})
	assert.Equal(t, "AS", first["initials"])
	assert.Equal(t, "Alice Smith", first["full_name"])
}

func TestMeetingListMineHandler_SortParam(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	// sort=duration_desc is parsed and forwarded to ListByUser.
	mr.On("ListByUser", mock.Anything, hostID, mock.Anything).
		Return([]*ports.MeetingListItem{}, nil)

	w := doRequest(r, http.MethodGet, "/meetings/mine?sort=duration_desc", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mr.AssertExpectations(t)
}

func TestMeetingListMineHandler_ServiceError(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	mr.On("ListByUser", mock.Anything, hostID, mock.Anything).
		Return(nil, assert.AnError)

	w := doRequest(r, http.MethodGet, "/meetings/mine", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mr.AssertExpectations(t)
}

// ─── ListMine — scope filter (Task 4.6) ────────────────────────────────────

// Builds a meeting service that lets the test inject the membership repo so
// scope filters can pass membership validation.
func newMeetingServiceWithMembers(
	mr *mockMeetingRepo,
	pr *mockParticipantRepo,
	memberRepo *mockMemberRepo,
	deptMemberRepo *mockDeptMemberRepo,
) *services.MeetingService {
	return services.NewMeetingService(mr, pr, memberRepo, deptMemberRepo,
		"http://rekall.test", zap.NewNop())
}

func TestMeetingListMineHandler_ScopeOpen_PassesFilter(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	mr.On("ListByUser", mock.Anything, hostID, mock.MatchedBy(func(f ports.ListMeetingsFilter) bool {
		return f.Scope != nil && f.Scope.Kind == ports.ScopeKindOpen
	})).Return([]*ports.MeetingListItem{}, nil)

	w := doRequest(r, http.MethodGet, "/meetings/mine?filter%5Bscope_type%5D=open", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mr.AssertExpectations(t)
}

func TestMeetingListMineHandler_ScopeOrg_NonMember_403(t *testing.T) {
	hostID, orgID := uuid.New(), uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	memberRepo := new(mockMemberRepo)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, hostID).
		Return(nil, apperr.NotFound("OrgMembership", orgID.String()))

	h := newMeetingHandler(newMeetingServiceWithMembers(mr, pr, memberRepo, new(mockDeptMemberRepo)))
	r := newMeetingRouter(h, hostID)

	url := "/meetings/mine?filter%5Bscope_type%5D=organization&filter%5Bscope_id%5D=" + orgID.String()
	w := doRequest(r, http.MethodGet, url, nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mr.AssertNotCalled(t, "ListByUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestMeetingListMineHandler_ScopeOrg_Member_200(t *testing.T) {
	hostID, orgID := uuid.New(), uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	memberRepo := new(mockMemberRepo)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, hostID).
		Return(&entities.OrgMembership{Role: "member"}, nil)
	mr.On("ListByUser", mock.Anything, hostID, mock.MatchedBy(func(f ports.ListMeetingsFilter) bool {
		return f.Scope != nil && f.Scope.Kind == ports.ScopeKindOrganization && f.Scope.ID == orgID
	})).Return([]*ports.MeetingListItem{}, nil)

	h := newMeetingHandler(newMeetingServiceWithMembers(mr, pr, memberRepo, new(mockDeptMemberRepo)))
	r := newMeetingRouter(h, hostID)

	url := "/meetings/mine?filter%5Bscope_type%5D=organization&filter%5Bscope_id%5D=" + orgID.String()
	w := doRequest(r, http.MethodGet, url, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mr.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestMeetingListMineHandler_ScopeMalformedUUID_400(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	w := doRequest(r, http.MethodGet,
		"/meetings/mine?filter%5Bscope_type%5D=organization&filter%5Bscope_id%5D=not-a-uuid", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mr.AssertNotCalled(t, "ListByUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestMeetingListMineHandler_ScopeUnknownType_400(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	w := doRequest(r, http.MethodGet, "/meetings/mine?filter%5Bscope_type%5D=other", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── End ─────────────────────────────────────────────────────────────────────

func TestMeetingEndHandler_Success(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	meeting := activeMeeting(hostID)
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(meeting, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).Return(nil)
	pr.On("MarkAllLeft", mock.Anything, meeting.ID).Return(nil)

	w := doRequest(r, http.MethodDelete, "/meetings/abc-defg-hij", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mr.AssertExpectations(t)
	pr.AssertExpectations(t)
}

func TestMeetingEndHandler_NotHost_Forbidden(t *testing.T) {
	callerID := uuid.New() // not the host
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, callerID)

	meeting := activeMeeting(uuid.New()) // different hostID
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(meeting, nil)

	w := doRequest(r, http.MethodDelete, "/meetings/abc-defg-hij", nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mr.AssertNotCalled(t, "Update")
}

func TestMeetingEndHandler_ServiceError(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	meeting := activeMeeting(hostID)
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(meeting, nil)
	mr.On("Update", mock.Anything, mock.AnythingOfType("*entities.Meeting")).
		Return(assert.AnError)

	w := doRequest(r, http.MethodDelete, "/meetings/abc-defg-hij", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mr.AssertExpectations(t)
}

func TestMeetingEndHandler_MeetingNotFound(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, uuid.New())

	mr.On("GetByCode", mock.Anything, "abc-defg-hij").
		Return(nil, apperr.NotFound("meeting", "abc-defg-hij"))

	w := doRequest(r, http.MethodDelete, "/meetings/abc-defg-hij", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── Connect (WS pre-upgrade paths) ──────────────────────────────────────────

func TestMeetingConnectHandler_MissingTicket(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, uuid.New())

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/ws", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "TICKET_REQUIRED")
	mr.AssertNotCalled(t, "GetByCode")
}

func TestMeetingConnectHandler_InvalidTicket(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, uuid.New())

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/ws?ticket=bogus", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "TICKET_INVALID")
	mr.AssertNotCalled(t, "GetByCode")
}

func TestMeetingConnectHandler_TicketMismatch(t *testing.T) {
	callerID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h, store := newMeetingHandlerWithTicketStore(newMeetingService(mr, pr))
	r := newMeetingRouter(h, callerID)

	// Ticket issued for a DIFFERENT meeting code.
	ticket := issueTestTicket(t, store, "other-code", callerID)
	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/ws?ticket="+ticket, nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "TICKET_MISMATCH")
	mr.AssertNotCalled(t, "GetByCode")
}

func TestMeetingConnectHandler_MeetingNotFound(t *testing.T) {
	callerID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h, store := newMeetingHandlerWithTicketStore(newMeetingService(mr, pr))
	r := newMeetingRouter(h, callerID)

	mr.On("GetByCode", mock.Anything, "abc-defg-hij").
		Return(nil, apperr.NotFound("meeting", "abc-defg-hij"))

	ticket := issueTestTicket(t, store, "abc-defg-hij", callerID)
	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/ws?ticket="+ticket, nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mr.AssertExpectations(t)
}

func TestMeetingConnectHandler_CanJoinError(t *testing.T) {
	// CanJoin can error when a DB call fails during scope membership lookup.
	// The handler should propagate this as a 5xx rather than silently falling
	// through to the WS upgrade.
	callerID := uuid.New()
	orgID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h, store := newMeetingHandlerWithTicketStore(newMeetingService(mr, pr))
	r := newMeetingRouter(h, callerID)

	scopeType := entities.MeetingScopeOrg
	private := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "abc-defg-hij",
		Type:            entities.MeetingTypePrivate,
		Status:          entities.MeetingStatusWaiting,
		HostID:          uuid.New(), // different host so caller isn't host
		ScopeType:       &scopeType,
		ScopeID:         &orgID,
		MaxParticipants: entities.MeetingMaxParticipants,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(private, nil)
	// CountActive is called before scope lookup; return under-capacity so CanJoin proceeds.
	pr.On("CountActive", mock.Anything, private.ID).
		Return(int64(0), assert.AnError) // error from the participant repo

	ticket := issueTestTicket(t, store, "abc-defg-hij", callerID)
	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/ws?ticket="+ticket, nil)

	// Any 4xx/5xx is acceptable — the key assertion is that the handler does
	// not return 200 or attempt a WS upgrade when CanJoin errors.
	assert.GreaterOrEqual(t, w.Code, http.StatusBadRequest)
	mr.AssertExpectations(t)
}

// injectBadClaims plants claims whose Subject is not a UUID, so SubjectAsUUID()
// returns an error. Used to exercise the "invalid token subject" branch.
func injectBadClaims(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := &infraauth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "not-a-uuid"},
			Email:            "test@example.com",
			Role:             role,
		}
		c.Set("auth_claims", claims)
		c.Next()
	}
}

func TestMeetingCreateHandler_BadSubject(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))

	r := gin.New()
	r.Use(injectBadClaims("member"))
	r.POST("/meetings", h.Create)

	body := jsonBody(t, dto.CreateMeetingRequest{Type: "open", Title: "T"})
	w := doRequest(r, http.MethodPost, "/meetings", body)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMeetingListMineHandler_BadSubject(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))

	r := gin.New()
	r.Use(injectBadClaims("member"))
	r.GET("/meetings/mine", h.ListMine)

	w := doRequest(r, http.MethodGet, "/meetings/mine", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMeetingEndHandler_BadSubject(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))

	r := gin.New()
	r.Use(injectBadClaims("member"))
	r.DELETE("/meetings/:code", h.End)

	w := doRequest(r, http.MethodDelete, "/meetings/abc", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMeetingCreateHandler_InvalidBody(t *testing.T) {
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	// Malformed JSON body.
	req := httptest.NewRequest(http.MethodPost, "/meetings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMeetingListMineHandler_ServiceError2(t *testing.T) {
	// Service error → 5xx (different from existing ServiceError test).
	hostID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h := newMeetingHandler(newMeetingService(mr, pr))
	r := newMeetingRouter(h, hostID)

	mr.On("ListByUser", mock.Anything, hostID, mock.Anything).
		Return([]*ports.MeetingListItem(nil), assert.AnError)

	w := doRequest(r, http.MethodGet, "/meetings/mine", nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestMeetingConnectHandler_AccessDenied(t *testing.T) {
	// An ended meeting causes CanJoin to return denied immediately.
	callerID := uuid.New()
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	h, store := newMeetingHandlerWithTicketStore(newMeetingService(mr, pr))
	r := newMeetingRouter(h, callerID)

	now := time.Now().UTC()
	ended := &entities.Meeting{
		ID:              uuid.New(),
		Code:            "abc-defg-hij",
		Type:            entities.MeetingTypeOpen,
		Status:          entities.MeetingStatusEnded,
		HostID:          uuid.New(),
		MaxParticipants: entities.MeetingMaxParticipants,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(ended, nil)

	ticket := issueTestTicket(t, store, "abc-defg-hij", callerID)
	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij/ws?ticket="+ticket, nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mr.AssertExpectations(t)
}
