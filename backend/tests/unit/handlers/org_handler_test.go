package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Router factory ───────────────────────────────────────────────────────────

func newOrgRouter(h *handlers.OrganizationHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/organizations", h.List)
	r.POST("/organizations", h.Create)
	r.GET("/organizations/:id", h.Get)
	r.PATCH("/organizations/:id", h.Update)
	r.DELETE("/organizations/:id", h.Delete)
	r.GET("/organizations/:id/members", h.ListMembers)
	r.PATCH("/organizations/:id/members/:userID", h.UpdateMember)
	r.DELETE("/organizations/:id/members/:userID", h.RemoveMember)
	r.POST("/organizations/:id/invitations", h.InviteUser)
	r.POST("/invitations/accept", h.AcceptInvitation)
	return r
}

func newOrgService(orgRepo *mockOrgRepo, memberRepo *mockMemberRepo, inviteRepo *mockInviteRepo, userRepo *mockUserRepo, mailer *mockMailer) *services.OrganizationService {
	return services.NewOrganizationService(
		orgRepo, memberRepo, inviteRepo, userRepo, mailer,
		"http://localhost:5173", 48*time.Hour, zap.NewNop(),
	)
}

func sampleOrg(ownerID uuid.UUID) *entities.Organization {
	return &entities.Organization{
		ID:        uuid.New(),
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func ownerMembership(orgID, userID uuid.UUID) *entities.OrgMembership {
	return &entities.OrgMembership{OrgID: orgID, UserID: userID, Role: "owner", JoinedAt: time.Now()}
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestOrgListHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	orgRepo.On("ListByUserID", mock.Anything, callerID).Return([]*entities.Organization{org}, nil)

	w := doRequest(r, http.MethodGet, "/organizations", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestOrgCreateHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	orgRepo.On("GetBySlug", mock.Anything, "acme-corp").Return(nil, apperr.NotFound("Org", "acme-corp"))
	orgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Organization")).Return(org, nil)
	memberRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.OrgMembership")).Return(nil)

	w := doRequest(r, http.MethodPost, "/organizations", jsonBody(t, map[string]string{"name": "Acme Corp"}))

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestOrgCreateHandler_MissingName(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/organizations", jsonBody(t, map[string]string{}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestOrgCreateHandler_EmptyBody(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	// POST with no JSON body at all → invalid request body.
	req := httptest.NewRequest(http.MethodPost, "/organizations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestOrgCreateHandler_BadBodyType(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader("{{bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestOrgUpdateHandler_BadBodyType(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	req := httptest.NewRequest(http.MethodPatch, "/organizations/"+uuid.New().String(), strings.NewReader("{{bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestOrgGetHandler_ServiceError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), userRepo, new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	orgID := uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("OrgMembership", ""))
	// Caller is a plain member at the platform level — no admin fallthrough.
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	w := doRequest(r, http.MethodGet, "/organizations/"+orgID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOrgCreateHandler_ServiceError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	// GetBySlug returns non-NotFound error → slug uniqueness lookup fails.
	orgRepo.On("GetBySlug", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	w := doRequest(r, http.MethodPost, "/organizations", jsonBody(t, map[string]string{"name": "Acme Corp"}))
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

// ─── Get ──────────────────────────────────────────────────────────────────────

func TestOrgGetHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(ownerMembership(org.ID, callerID), nil)

	w := doRequest(r, http.MethodGet, "/organizations/"+org.ID.String(), nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrgGetHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodGet, "/organizations/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgGetHandler_NotMember(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), userRepo, new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(uuid.New()) // different owner
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(nil, apperr.NotFound("Membership", ""))
	// Caller isn't a platform admin — fallthrough denied.
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	w := doRequest(r, http.MethodGet, "/organizations/"+org.ID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func TestOrgUpdateHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	updated := *org
	updated.Name = "Acme Corp 2"
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(ownerMembership(org.ID, callerID), nil)
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	orgRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.Organization")).Return(&updated, nil)

	w := doRequest(r, http.MethodPatch, "/organizations/"+org.ID.String(), jsonBody(t, map[string]string{"name": "Acme Corp 2"}))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrgUpdateHandler_InvalidBody(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPatch, "/organizations/"+uuid.New().String(), jsonBody(t, map[string]string{}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestOrgDeleteHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodDelete, "/organizations/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgDeleteHandler_NotOwner(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), userRepo, new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(uuid.New())
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(
		&entities.OrgMembership{OrgID: org.ID, UserID: callerID, Role: "member"}, nil)
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	w := doRequest(r, http.MethodDelete, "/organizations/"+org.ID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOrgListHandler_InternalError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	orgRepo.On("ListByUserID", mock.Anything, callerID).Return([]*entities.Organization{}, assert.AnError)

	w := doRequest(r, http.MethodGet, "/organizations", nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOrgUpdateHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPatch, "/organizations/not-a-uuid", jsonBody(t, map[string]string{"name": "X"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgInviteUserHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/organizations/not-a-uuid/invitations", jsonBody(t, map[string]string{"email": "x@y.z"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgListMembersHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodGet, "/organizations/not-a-uuid/members", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgListMembersHandler_ServiceError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	orgID := uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("OrgMembership", ""))

	w := doRequest(r, http.MethodGet, "/organizations/"+orgID.String()+"/members", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOrgRemoveMemberHandler_ServiceError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), userRepo, new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	orgID := uuid.New()
	targetID := uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("OrgMembership", ""))
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	w := doRequest(r, http.MethodDelete, "/organizations/"+orgID.String()+"/members/"+targetID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOrgInviteUserHandler_ServiceError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), userRepo, new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	orgID := uuid.New()
	// Caller isn't a member — should be forbidden.
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("OrgMembership", ""))
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	w := doRequest(r, http.MethodPost, "/organizations/"+orgID.String()+"/invitations",
		jsonBody(t, map[string]string{"email": "a@b.com", "role": "member"}))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOrgAcceptInvitationHandler_InvalidBody(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	req := httptest.NewRequest(http.MethodPost, "/invitations/accept", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestOrgAcceptInvitationHandler_ServiceError(t *testing.T) {
	inviteRepo := new(mockInviteRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), inviteRepo, new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	inviteRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, apperr.NotFound("Invitation", "x"))

	w := doRequest(r, http.MethodPost, "/invitations/accept", jsonBody(t, map[string]string{"token": "bad-token"}))
	// Service converts NotFound → BadRequest ("invalid or expired invitation")
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.GreaterOrEqual(t, w.Code, http.StatusBadRequest)
}

func TestOrgUpdateHandler_ServiceError(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	userRepo := new(mockUserRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), userRepo, new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	orgID := uuid.New()
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("OrgMembership", ""))
	userRepo.On("GetByID", mock.Anything, callerID).
		Return(&entities.User{ID: callerID, Role: "member"}, nil)

	w := doRequest(r, http.MethodPatch, "/organizations/"+orgID.String(), jsonBody(t, map[string]string{"name": "X"}))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOrgRemoveMemberHandler_InvalidOrgUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodDelete, "/organizations/not-a-uuid/members/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgDeleteHandler_OwnerSuccess(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(ownerMembership(org.ID, callerID), nil)
	orgRepo.On("SoftDelete", mock.Anything, org.ID).Return(nil)

	w := doRequest(r, http.MethodDelete, "/organizations/"+org.ID.String(), nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ─── ListMembers ──────────────────────────────────────────────────────────────

func TestOrgListMembersHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(ownerMembership(org.ID, callerID), nil)
	memberRepo.On("ListByOrg", mock.Anything, org.ID).Return([]*entities.OrgMembership{
		{UserID: callerID, OrgID: org.ID, Role: "owner", JoinedAt: time.Now()},
	}, nil)

	w := doRequest(r, http.MethodGet, "/organizations/"+org.ID.String()+"/members", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].([]interface{})
	assert.Len(t, data, 1)
}

// ─── RemoveMember ─────────────────────────────────────────────────────────────

func TestOrgRemoveMemberHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	targetID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(ownerMembership(org.ID, callerID), nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, targetID).Return(
		&entities.OrgMembership{OrgID: org.ID, UserID: targetID, Role: "member"}, nil)
	memberRepo.On("Delete", mock.Anything, org.ID, targetID).Return(nil)

	w := doRequest(r, http.MethodDelete, "/organizations/"+org.ID.String()+"/members/"+targetID.String(), nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOrgRemoveMemberHandler_InvalidTargetUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodDelete, "/organizations/"+uuid.New().String()+"/members/bad-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── InviteUser ───────────────────────────────────────────────────────────────

func TestOrgInviteUserHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	mailer := new(mockMailer)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, mailer), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(callerID)
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(ownerMembership(org.ID, callerID), nil)
	// InviteUser fetches the inviter by ID to embed their name in the email
	userRepo.On("GetByID", mock.Anything, callerID).Return(&entities.User{
		ID: callerID, Email: "owner@example.com", FullName: "Owner", Role: "member",
	}, nil)
	inviteRepo.On("GetPendingByOrgAndEmail", mock.Anything, org.ID, "new@example.com").Return(nil, apperr.NotFound("Invitation", ""))
	inviteRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entities.Invitation")).Return(nil)
	mailer.On("Send", mock.Anything, mock.AnythingOfType("ports.EmailMessage")).Return(nil)

	w := doRequest(r, http.MethodPost, "/organizations/"+org.ID.String()+"/invitations",
		jsonBody(t, map[string]string{"email": "new@example.com", "role": "member"}))

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestOrgInviteUserHandler_MissingEmail(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/organizations/"+uuid.New().String()+"/invitations",
		jsonBody(t, map[string]string{"role": "member"}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── AcceptInvitation ─────────────────────────────────────────────────────────

func TestOrgAcceptInvitationHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	inviteRepo := new(mockInviteRepo)
	userRepo := new(mockUserRepo)
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, inviteRepo, userRepo, new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	org := sampleOrg(uuid.New())
	inv := &entities.Invitation{
		ID:        uuid.New(),
		OrgID:     org.ID,
		Email:     "test@example.com", // must match user.Email returned by userRepo
		Role:      "member",
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	// AcceptInvitation fetches the caller's user record to verify email matches invitation
	userRepo.On("GetByID", mock.Anything, callerID).Return(&entities.User{
		ID: callerID, Email: "test@example.com", FullName: "Caller", Role: "member",
	}, nil)
	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	inviteRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(inv, nil)
	inviteRepo.On("MarkAccepted", mock.Anything, mock.Anything).Return(nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(nil, apperr.NotFound("Membership", ""))
	memberRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.OrgMembership")).Return(nil)

	w := doRequest(r, http.MethodPost, "/invitations/accept", jsonBody(t, map[string]string{"token": "raw-invite-token"}))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrgAcceptInvitationHandler_MissingToken(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/invitations/accept", jsonBody(t, map[string]string{}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── UpdateMember (organization) ──────────────────────────────────────────────

func TestOrgUpdateMemberHandler_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	targetID := uuid.New()
	org := sampleOrg(callerID)
	h := handlers.NewOrganizationHandler(newOrgService(orgRepo, memberRepo, new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	orgRepo.On("GetByID", mock.Anything, org.ID).Return(org, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, callerID).Return(ownerMembership(org.ID, callerID), nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, org.ID, targetID).Return(
		&entities.OrgMembership{OrgID: org.ID, UserID: targetID, Role: "member"}, nil)
	memberRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.OrgMembership")).Return(nil)

	w := doRequest(r, http.MethodPatch,
		"/organizations/"+org.ID.String()+"/members/"+targetID.String(),
		jsonBody(t, map[string]string{"role": "admin"}))

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOrgUpdateMemberHandler_InvalidOrgUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPatch, "/organizations/bad-uuid/members/"+uuid.New().String(),
		jsonBody(t, map[string]string{"role": "admin"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgUpdateMemberHandler_InvalidUserUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	w := doRequest(r, http.MethodPatch, "/organizations/"+uuid.New().String()+"/members/bad-uuid",
		jsonBody(t, map[string]string{"role": "admin"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrgUpdateMemberHandler_InvalidBody(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewOrganizationHandler(newOrgService(new(mockOrgRepo), new(mockMemberRepo), new(mockInviteRepo), new(mockUserRepo), new(mockMailer)), zap.NewNop())
	r := newOrgRouter(h, callerID)

	// Invalid JSON (empty body fails required role field)
	w := doRequest(r, http.MethodPatch,
		"/organizations/"+uuid.New().String()+"/members/"+uuid.New().String(),
		jsonBody(t, map[string]string{}))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
