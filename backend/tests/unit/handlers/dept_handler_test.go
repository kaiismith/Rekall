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

func newDeptRouter(h *handlers.DepartmentHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/organizations/:id/departments", h.ListByOrg)
	r.POST("/organizations/:id/departments", h.Create)
	r.GET("/departments/:deptID", h.Get)
	r.PATCH("/departments/:deptID", h.Update)
	r.DELETE("/departments/:deptID", h.Delete)
	r.GET("/departments/:deptID/members", h.ListMembers)
	r.POST("/departments/:deptID/members", h.AddMember)
	r.PATCH("/departments/:deptID/members/:userID", h.UpdateMember)
	r.DELETE("/departments/:deptID/members/:userID", h.RemoveMember)
	return r
}

func newDeptService(deptRepo *mockDeptRepo, deptMemberRepo *mockDeptMemberRepo, memberRepo *mockMemberRepo) *services.DepartmentService {
	return services.NewDepartmentService(deptRepo, deptMemberRepo, memberRepo, zap.NewNop())
}

func sampleDept(orgID, createdBy uuid.UUID) *entities.Department {
	return &entities.Department{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      "Engineering",
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ─── ListByOrg ────────────────────────────────────────────────────────────────

func TestDeptListByOrgHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "owner"}, nil)
	deptRepo.On("ListByOrg", mock.Anything, orgID).Return([]*entities.Department{dept}, nil)

	w := doRequest(r, http.MethodGet, "/organizations/"+orgID.String()+"/departments", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].([]interface{})
	assert.Len(t, data, 1)
}

func TestDeptListByOrgHandler_InvalidOrgUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodGet, "/organizations/not-a-uuid/departments", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestDeptCreateHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "admin"}, nil)
	deptRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Department")).Return(dept, nil)

	w := doRequest(r, http.MethodPost, "/organizations/"+orgID.String()+"/departments",
		jsonBody(t, map[string]string{"name": "Engineering", "description": "Build things"}))

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestDeptCreateHandler_MissingName(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/organizations/"+uuid.New().String()+"/departments",
		jsonBody(t, map[string]string{}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeptCreateHandler_NotOrgMember(t *testing.T) {
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("Membership", ""))

	w := doRequest(r, http.MethodPost, "/organizations/"+orgID.String()+"/departments",
		jsonBody(t, map[string]string{"name": "Eng"}))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ─── Get ──────────────────────────────────────────────────────────────────────

func TestDeptGetHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "member"}, nil)

	w := doRequest(r, http.MethodGet, "/departments/"+dept.ID.String(), nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeptGetHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodGet, "/departments/bad-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptGetHandler_NotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	deptID := uuid.New()
	deptRepo.On("GetByID", mock.Anything, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	w := doRequest(r, http.MethodGet, "/departments/"+deptID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestDeptDeleteHandler_AdminSuccess(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "admin"}, nil)
	deptRepo.On("SoftDelete", mock.Anything, dept.ID).Return(nil)

	w := doRequest(r, http.MethodDelete, "/departments/"+dept.ID.String(), nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeptDeleteHandler_Forbidden(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, deptMemberRepo, memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, uuid.New())
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, callerID).Return(
		nil, apperr.NotFound("DeptMembership", ""))

	w := doRequest(r, http.MethodDelete, "/departments/"+dept.ID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ─── ListMembers ──────────────────────────────────────────────────────────────

func TestDeptListMembersHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, deptMemberRepo, memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "member"}, nil)
	deptMemberRepo.On("ListByDept", mock.Anything, dept.ID).Return([]*entities.DepartmentMembership{
		{DepartmentID: dept.ID, UserID: callerID, Role: "member", JoinedAt: time.Now()},
	}, nil)

	w := doRequest(r, http.MethodGet, "/departments/"+dept.ID.String()+"/members", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].([]interface{})
	assert.Len(t, data, 1)
}

// ─── AddMember ────────────────────────────────────────────────────────────────

func TestDeptAddMemberHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	targetID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, deptMemberRepo, memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "admin"}, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, targetID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: targetID, Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, targetID).Return(nil, apperr.NotFound("DeptMembership", ""))
	deptMemberRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.DepartmentMembership")).Return(nil)

	w := doRequest(r, http.MethodPost, "/departments/"+dept.ID.String()+"/members",
		jsonBody(t, map[string]string{"user_id": targetID.String(), "role": "member"}))

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeptAddMemberHandler_InvalidUserID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/departments/"+uuid.New().String()+"/members",
		jsonBody(t, map[string]string{"user_id": "not-a-uuid", "role": "member"}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeptAddMemberHandler_MissingBody(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/departments/"+uuid.New().String()+"/members",
		jsonBody(t, map[string]string{}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ─── RemoveMember ─────────────────────────────────────────────────────────────

func TestDeptRemoveMemberHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	targetID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, deptMemberRepo, memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "admin"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, targetID).Return(
		&entities.DepartmentMembership{DepartmentID: dept.ID, UserID: targetID, Role: "member"}, nil)
	deptMemberRepo.On("Delete", mock.Anything, dept.ID, targetID).Return(nil)

	w := doRequest(r, http.MethodDelete, "/departments/"+dept.ID.String()+"/members/"+targetID.String(), nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeptRemoveMemberHandler_InvalidTargetUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodDelete, "/departments/"+uuid.New().String()+"/members/bad-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Update department ────────────────────────────────────────────────────────

func TestDeptUpdateHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "admin"}, nil)
	deptRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.Department")).Return(dept, nil)

	w := doRequest(r, http.MethodPatch, "/departments/"+dept.ID.String(), jsonBody(t, map[string]string{
		"name":        "Engineering Renamed",
		"description": "New desc",
	}))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeptUpdateHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPatch, "/departments/not-a-uuid", jsonBody(t, map[string]string{"name": "X"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptUpdateHandler_InvalidBody(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	// Missing all required fields
	req := httptest.NewRequest(http.MethodPatch, "/departments/"+uuid.New().String(), nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeptUpdateHandler_NotFound(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	deptRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, apperr.NotFound("Department", "x"))

	w := doRequest(r, http.MethodPatch, "/departments/"+uuid.New().String(), jsonBody(t, map[string]string{"name": "X"}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── UpdateMember (department) ────────────────────────────────────────────────

func TestDeptUpdateMemberHandler_Success(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	targetID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, deptMemberRepo, memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	dept := sampleDept(orgID, callerID)
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "admin"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, targetID).Return(
		&entities.DepartmentMembership{DepartmentID: dept.ID, UserID: targetID, Role: "member"}, nil)
	deptMemberRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.DepartmentMembership")).Return(nil)

	w := doRequest(r, http.MethodPatch,
		"/departments/"+dept.ID.String()+"/members/"+targetID.String(),
		jsonBody(t, map[string]string{"role": "head"}))

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeptUpdateMemberHandler_InvalidRole(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	// Service validates role — it returns 422 for invalid role before any repo calls.
	w := doRequest(r, http.MethodPatch,
		"/departments/"+uuid.New().String()+"/members/"+uuid.New().String(),
		jsonBody(t, map[string]string{"role": "not-a-role"}))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeptUpdateMemberHandler_InvalidDeptUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPatch, "/departments/bad-uuid/members/"+uuid.New().String(),
		jsonBody(t, map[string]string{"role": "head"}))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptUpdateMemberHandler_InvalidUserUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPatch, "/departments/"+uuid.New().String()+"/members/bad-uuid",
		jsonBody(t, map[string]string{"role": "head"}))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Invalid UUID edge cases for other dept handlers ─────────────────────────

func TestDeptDeleteHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodDelete, "/departments/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptListMembersHandler_InvalidUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodGet, "/departments/not-a-uuid/members", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptAddMemberHandler_InvalidDeptUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/departments/not-a-uuid/members",
		jsonBody(t, map[string]string{"user_id": uuid.New().String(), "role": "member"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptRemoveMemberHandler_InvalidDeptUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodDelete, "/departments/not-a-uuid/members/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptListMembersHandler_NotOrgMember(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	orgID := uuid.New()
	dept := sampleDept(orgID, uuid.New())
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("OrgMembership", ""))

	w := doRequest(r, http.MethodGet, "/departments/"+dept.ID.String()+"/members", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeptRemoveMemberHandler_NotManager(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, deptMemberRepo, memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	orgID := uuid.New()
	targetID := uuid.New()
	dept := sampleDept(orgID, uuid.New())
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		&entities.OrgMembership{OrgID: orgID, UserID: callerID, Role: "member"}, nil)
	deptMemberRepo.On("GetByDeptAndUser", mock.Anything, dept.ID, callerID).Return(
		nil, apperr.NotFound("DeptMembership", ""))

	w := doRequest(r, http.MethodDelete,
		"/departments/"+dept.ID.String()+"/members/"+targetID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeptListByOrgHandler_MissingMembership(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	orgID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(
		nil, apperr.NotFound("OrgMembership", ""))

	w := doRequest(r, http.MethodGet, "/organizations/"+orgID.String()+"/departments", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeptCreateHandler_InvalidBody(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	// Empty body → ShouldBindJSON rejects required "name".
	req := httptest.NewRequest(http.MethodPost, "/organizations/"+uuid.New().String()+"/departments", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeptGetHandler_NotOrgMember(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, new(mockDeptMemberRepo), memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	orgID := uuid.New()
	dept := sampleDept(orgID, uuid.New())
	deptRepo.On("GetByID", mock.Anything, dept.ID).Return(dept, nil)
	memberRepo.On("GetByOrgAndUser", mock.Anything, orgID, callerID).Return(nil, apperr.NotFound("OrgMembership", ""))

	w := doRequest(r, http.MethodGet, "/departments/"+dept.ID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeptAddMemberHandler_ServiceError(t *testing.T) {
	deptRepo := new(mockDeptRepo)
	deptMemberRepo := new(mockDeptMemberRepo)
	memberRepo := new(mockMemberRepo)
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(deptRepo, deptMemberRepo, memberRepo), zap.NewNop())
	r := newDeptRouter(h, callerID)

	deptID := uuid.New()
	deptRepo.On("GetByID", mock.Anything, deptID).Return(nil, apperr.NotFound("Department", deptID.String()))

	w := doRequest(r, http.MethodPost,
		"/departments/"+deptID.String()+"/members",
		jsonBody(t, map[string]string{"user_id": uuid.New().String(), "role": "member"}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeptUpdateHandler_BadBodyType(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	req := httptest.NewRequest(http.MethodPatch, "/departments/"+uuid.New().String(),
		strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeptCreateHandler_InvalidOrgUUID(t *testing.T) {
	callerID := uuid.New()
	h := handlers.NewDepartmentHandler(newDeptService(new(mockDeptRepo), new(mockDeptMemberRepo), new(mockMemberRepo)), zap.NewNop())
	r := newDeptRouter(h, callerID)

	w := doRequest(r, http.MethodPost, "/organizations/not-a-uuid/departments",
		jsonBody(t, map[string]string{"name": "Eng"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
