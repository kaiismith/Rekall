package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	httputils "github.com/rekall/backend/internal/interfaces/http/utils"
	apperr "github.com/rekall/backend/pkg/errors"
	"go.uber.org/zap"
)

// DepartmentHandler handles HTTP requests for department and department-membership endpoints.
type DepartmentHandler struct {
	service *services.DepartmentService
	logger  *zap.Logger
}

// NewDepartmentHandler creates a DepartmentHandler with its required dependencies.
func NewDepartmentHandler(service *services.DepartmentService, log *zap.Logger) *DepartmentHandler {
	return &DepartmentHandler{service: service, logger: log}
}

// ── Departments ───────────────────────────────────────────────────────────────

// ListByOrg handles GET /api/v1/organizations/:id/departments.
//
// @Summary      List departments in an organization
// @Description  Returns all departments belonging to the specified organization. Restricted to organization members.
// @Tags         Departments
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string               true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Success      200  {object}  dto.DeptListResponse  "List of departments"
// @Failure      400  {object}  dto.ErrorResponse     "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse     "Missing or invalid token"
// @Failure      404  {object}  dto.ErrorResponse     "Organization not found or access denied"
// @Failure      500  {object}  dto.ErrorResponse     "Internal server error"
// @Router       /api/v1/organizations/{id}/departments [get]
func (h *DepartmentHandler) ListByOrg(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	depts, err := h.service.ListDepartments(c.Request.Context(), orgID, callerID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	resp := make([]dto.DeptResponse, len(depts))
	for i, d := range depts {
		resp[i] = httputils.ToDeptResponse(d)
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// Create handles POST /api/v1/organizations/:id/departments.
//
// @Summary      Create a department
// @Description  Creates a new department within the specified organization. Restricted to `owner` or `admin` members.
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Param        body  body      dto.CreateDeptRequest     true  "Department details"
// @Success      201   {object}  dto.DeptResponseEnvelope  "Department created"
// @Failure      400   {object}  dto.ErrorResponse         "Invalid UUID format"
// @Failure      401   {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      403   {object}  dto.ErrorResponse         "Insufficient role (FORBIDDEN)"
// @Failure      404   {object}  dto.ErrorResponse         "Organization not found or access denied"
// @Failure      422   {object}  dto.ErrorResponse         "Validation error"
// @Failure      500   {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/organizations/{id}/departments [post]
func (h *DepartmentHandler) Create(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.CreateDeptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	dept, err := h.service.CreateDepartment(c.Request.Context(), orgID, callerID, req.Name, req.Description)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusCreated, dto.OK(httputils.ToDeptResponse(dept)))
}

// Get handles GET /api/v1/departments/:deptID.
//
// @Summary      Get a department
// @Description  Returns a single department by its UUID. Restricted to members of the parent organization.
// @Tags         Departments
// @Produce      json
// @Security     BearerAuth
// @Param        deptID  path      string                    true  "Department UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000020)
// @Success      200     {object}  dto.DeptResponseEnvelope  "Department details"
// @Failure      400     {object}  dto.ErrorResponse         "Invalid UUID format"
// @Failure      401     {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      404     {object}  dto.ErrorResponse         "Department not found (DEPARTMENT_NOT_FOUND)"
// @Failure      500     {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/departments/{deptID} [get]
func (h *DepartmentHandler) Get(c *gin.Context) {
	deptID, err := httputils.ParseUUID(c, "deptID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	dept, err := h.service.GetDepartment(c.Request.Context(), deptID, callerID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(httputils.ToDeptResponse(dept)))
}

// Update handles PATCH /api/v1/departments/:deptID.
//
// @Summary      Update a department
// @Description  Updates a department's name and/or description. Restricted to `owner` or `admin` members of the parent organization.
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        deptID  path      string                    true  "Department UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000020)
// @Param        body    body      dto.UpdateDeptRequest     true  "Fields to update"
// @Success      200     {object}  dto.DeptResponseEnvelope  "Updated department"
// @Failure      400     {object}  dto.ErrorResponse         "Invalid UUID format"
// @Failure      401     {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      403     {object}  dto.ErrorResponse         "Insufficient role (FORBIDDEN)"
// @Failure      404     {object}  dto.ErrorResponse         "Department not found (DEPARTMENT_NOT_FOUND)"
// @Failure      422     {object}  dto.ErrorResponse         "Validation error"
// @Failure      500     {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/departments/{deptID} [patch]
func (h *DepartmentHandler) Update(c *gin.Context) {
	deptID, err := httputils.ParseUUID(c, "deptID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.UpdateDeptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	dept, err := h.service.UpdateDepartment(c.Request.Context(), deptID, callerID, req.Name, req.Description)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(httputils.ToDeptResponse(dept)))
}

// Delete handles DELETE /api/v1/departments/:deptID.
//
// @Summary      Delete a department
// @Description  Soft-deletes a department and removes all its memberships. Restricted to `owner` or `admin` members of the parent organization.
// @Tags         Departments
// @Produce      json
// @Security     BearerAuth
// @Param        deptID  path  string  true  "Department UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000020)
// @Success      204  "Department deleted"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "Department not found (DEPARTMENT_NOT_FOUND)"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/departments/{deptID} [delete]
func (h *DepartmentHandler) Delete(c *gin.Context) {
	deptID, err := httputils.ParseUUID(c, "deptID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.DeleteDepartment(c.Request.Context(), deptID, callerID); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ── Department Membership ─────────────────────────────────────────────────────

// ListMembers handles GET /api/v1/departments/:deptID/members.
//
// @Summary      List department members
// @Description  Returns all members of the specified department with their roles.
// @Tags         Departments
// @Produce      json
// @Security     BearerAuth
// @Param        deptID  path      string                      true  "Department UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000020)
// @Success      200     {object}  dto.DeptMemberListResponse  "List of department members"
// @Failure      400     {object}  dto.ErrorResponse           "Invalid UUID format"
// @Failure      401     {object}  dto.ErrorResponse           "Missing or invalid token"
// @Failure      404     {object}  dto.ErrorResponse           "Department not found (DEPARTMENT_NOT_FOUND)"
// @Failure      500     {object}  dto.ErrorResponse           "Internal server error"
// @Router       /api/v1/departments/{deptID}/members [get]
func (h *DepartmentHandler) ListMembers(c *gin.Context) {
	deptID, err := httputils.ParseUUID(c, "deptID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	members, err := h.service.ListDeptMembers(c.Request.Context(), deptID, callerID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	resp := make([]dto.DeptMemberResponse, len(members))
	for i, m := range members {
		resp[i] = httputils.ToDeptMemberResponse(m)
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// AddMember handles POST /api/v1/departments/:deptID/members.
//
// @Summary      Add a member to a department
// @Description  Adds an existing organization member to the department with the specified role. The target user must already be a member of the parent organization.
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        deptID  path      string                    true  "Department UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000020)
// @Param        body    body      dto.AddDeptMemberRequest  true  "Member to add"
// @Success      204  "Member added to department"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "Department or user not found"
// @Failure      409  {object}  dto.ErrorResponse  "User is already a department member (ALREADY_A_MEMBER)"
// @Failure      422  {object}  dto.ErrorResponse  "Validation error"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/departments/{deptID}/members [post]
func (h *DepartmentHandler) AddMember(c *gin.Context) {
	deptID, err := httputils.ParseUUID(c, "deptID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.AddDeptMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid user_id", nil))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.AddDeptMember(c.Request.Context(), deptID, callerID, targetID, req.Role); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UpdateMember handles PATCH /api/v1/departments/:deptID/members/:userID.
//
// @Summary      Update a department member's role
// @Description  Changes a department member's role (`head` or `member`). Restricted to `owner` or `admin` members of the parent organization.
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        deptID  path      string                          true  "Department UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000020)
// @Param        userID  path      string                          true  "Target user UUID" format(uuid)  example(00000000-0000-0000-0000-000000000001)
// @Param        body    body      dto.UpdateDeptMemberRoleRequest true  "New role"
// @Success      204  "Member role updated"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "Department or member not found"
// @Failure      422  {object}  dto.ErrorResponse  "Validation error"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/departments/{deptID}/members/{userID} [patch]
func (h *DepartmentHandler) UpdateMember(c *gin.Context) {
	deptID, err := httputils.ParseUUID(c, "deptID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	targetID, err := httputils.ParseUUID(c, "userID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.UpdateDeptMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.UpdateDeptMemberRole(c.Request.Context(), deptID, callerID, targetID, req.Role); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// RemoveMember handles DELETE /api/v1/departments/:deptID/members/:userID.
//
// @Summary      Remove a member from a department
// @Description  Removes a user from the department. The user remains a member of the parent organization. Restricted to `owner` or `admin`.
// @Tags         Departments
// @Produce      json
// @Security     BearerAuth
// @Param        deptID  path  string  true  "Department UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000020)
// @Param        userID  path  string  true  "Target user UUID" format(uuid)  example(00000000-0000-0000-0000-000000000001)
// @Success      204  "Member removed from department"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "Department or member not found"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/departments/{deptID}/members/{userID} [delete]
func (h *DepartmentHandler) RemoveMember(c *gin.Context) {
	deptID, err := httputils.ParseUUID(c, "deptID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	targetID, err := httputils.ParseUUID(c, "userID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.RemoveDeptMember(c.Request.Context(), deptID, callerID, targetID); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

