package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	handlerhelpers "github.com/rekall/backend/internal/interfaces/http/helpers"
	httputils "github.com/rekall/backend/internal/interfaces/http/utils"
	apperr "github.com/rekall/backend/pkg/errors"
)

// OrganizationHandler handles HTTP requests for organization, membership, and invitation endpoints.
type OrganizationHandler struct {
	service *services.OrganizationService
	logger  *zap.Logger
}

// NewOrganizationHandler creates an OrganizationHandler with its required dependencies.
func NewOrganizationHandler(service *services.OrganizationService, log *zap.Logger) *OrganizationHandler {
	return &OrganizationHandler{service: service, logger: log}
}

// ── Organization CRUD ─────────────────────────────────────────────────────────

// List handles GET /api/v1/organizations.
//
// @Summary      List my organizations
// @Description  Returns all organizations the authenticated user is a member of (any role). There is no pagination — the list is bounded by the user's membership count.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.OrgListResponse  "List of organizations"
// @Failure      401  {object}  dto.ErrorResponse    "Missing or invalid token"
// @Failure      500  {object}  dto.ErrorResponse    "Internal server error"
// @Router       /api/v1/organizations [get]
func (h *OrganizationHandler) List(c *gin.Context) {
	userID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	orgs, err := h.service.ListOrganizations(c.Request.Context(), userID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	resp := make([]dto.OrgResponse, len(orgs))
	for i, o := range orgs {
		resp[i] = httputils.ToOrgResponse(o)
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// Create handles POST /api/v1/organizations.
//
// @Summary      Create an organization
// @Description  Creates a new organization workspace. The authenticated user automatically becomes the `owner` member. The slug is derived from the name if not provided.
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateOrgRequest     true  "Organization details"
// @Success      201   {object}  dto.OrgResponseEnvelope  "Organization created"
// @Failure      401   {object}  dto.ErrorResponse        "Missing or invalid token"
// @Failure      409   {object}  dto.ErrorResponse        "Slug already taken (SLUG_ALREADY_TAKEN)"
// @Failure      422   {object}  dto.ErrorResponse        "Validation error"
// @Failure      500   {object}  dto.ErrorResponse        "Internal server error"
// @Router       /api/v1/organizations [post]
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req dto.CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	userID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	org, err := h.service.CreateOrganization(c.Request.Context(), userID, req.Name, req.OwnerEmail)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusCreated, dto.OK(httputils.ToOrgResponse(org)))
}

// Get handles GET /api/v1/organizations/:id.
//
// @Summary      Get an organization
// @Description  Returns a single organization by ID. Non-members receive **404** (not 403) to avoid leaking existence.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                   true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Success      200  {object}  dto.OrgResponseEnvelope  "Organization details"
// @Failure      400  {object}  dto.ErrorResponse        "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse        "Missing or invalid token"
// @Failure      404  {object}  dto.ErrorResponse        "Organization not found or access denied"
// @Failure      500  {object}  dto.ErrorResponse        "Internal server error"
// @Router       /api/v1/organizations/{id} [get]
func (h *OrganizationHandler) Get(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	userID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	org, err := h.service.GetOrganization(c.Request.Context(), orgID, userID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(httputils.ToOrgResponse(org)))
}

// Update handles PATCH /api/v1/organizations/:id.
//
// @Summary      Update an organization
// @Description  Updates the organization name. Restricted to members with `owner` or `admin` role.
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                   true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Param        body  body      dto.UpdateOrgRequest     true  "Fields to update"
// @Success      200   {object}  dto.OrgResponseEnvelope  "Updated organization"
// @Failure      400   {object}  dto.ErrorResponse        "Invalid UUID format"
// @Failure      401   {object}  dto.ErrorResponse        "Missing or invalid token"
// @Failure      403   {object}  dto.ErrorResponse        "Insufficient role — owner or admin required (FORBIDDEN)"
// @Failure      404   {object}  dto.ErrorResponse        "Organization not found or access denied"
// @Failure      422   {object}  dto.ErrorResponse        "Validation error"
// @Failure      500   {object}  dto.ErrorResponse        "Internal server error"
// @Router       /api/v1/organizations/{id} [patch]
func (h *OrganizationHandler) Update(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	userID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	org, err := h.service.UpdateOrganization(c.Request.Context(), orgID, userID, req.Name)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(httputils.ToOrgResponse(org)))
}

// Delete handles DELETE /api/v1/organizations/:id.
//
// @Summary      Delete an organization
// @Description  Soft-deletes the organization and all its memberships. Restricted to the `owner`.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Success      204  "Organization deleted"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role — owner required (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "Organization not found or access denied"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/organizations/{id} [delete]
func (h *OrganizationHandler) Delete(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	userID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.DeleteOrganization(c.Request.Context(), orgID, userID); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ── Membership ────────────────────────────────────────────────────────────────

// ListMembers handles GET /api/v1/organizations/:id/members.
//
// @Summary      List organization members
// @Description  Returns all active members of the organization with their roles. Restricted to current members.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                  true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Success      200  {object}  dto.MemberListResponse  "List of members"
// @Failure      400  {object}  dto.ErrorResponse       "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse       "Missing or invalid token"
// @Failure      404  {object}  dto.ErrorResponse       "Organization not found or access denied"
// @Failure      500  {object}  dto.ErrorResponse       "Internal server error"
// @Router       /api/v1/organizations/{id}/members [get]
func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	userID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	members, err := h.service.ListMembers(c.Request.Context(), orgID, userID)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	resp := make([]dto.MemberResponse, len(members))
	for i, m := range members {
		resp[i] = httputils.ToMemberResponse(m)
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// UpdateMember handles PATCH /api/v1/organizations/:id/members/:userID.
//
// @Summary      Update a member's role
// @Description  Changes a member's role within the organization. Restricted to `owner` or `admin`. An `admin` cannot promote another user to `owner` or modify another `admin`.
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string                       true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Param        userID  path      string                       true  "Target user UUID"   format(uuid)  example(00000000-0000-0000-0000-000000000002)
// @Param        body    body      dto.UpdateMemberRoleRequest  true  "New role"
// @Success      204  "Member role updated"
// @Failure      400  {object}  dto.ErrorResponse  "Cannot remove or demote the owner (CANNOT_REMOVE_OWNER)"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "Organization or member not found"
// @Failure      422  {object}  dto.ErrorResponse  "Validation error"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/organizations/{id}/members/{userID} [patch]
func (h *OrganizationHandler) UpdateMember(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	targetID, err := httputils.ParseUUID(c, "userID")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.UpdateMemberRole(c.Request.Context(), orgID, callerID, targetID, req.Role); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// RemoveMember handles DELETE /api/v1/organizations/:id/members/:userID.
//
// @Summary      Remove a member
// @Description  Removes a user from the organization. The `owner` cannot be removed via this endpoint — ownership must be transferred first. Restricted to `owner` or `admin`.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  string  true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Param        userID  path  string  true  "Target user UUID"   format(uuid)  example(00000000-0000-0000-0000-000000000002)
// @Success      204  "Member removed"
// @Failure      400  {object}  dto.ErrorResponse  "Cannot remove the owner (CANNOT_REMOVE_OWNER)"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "Organization or member not found"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/organizations/{id}/members/{userID} [delete]
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
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

	if err := h.service.RemoveMember(c.Request.Context(), orgID, callerID, targetID); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ── Invitations ───────────────────────────────────────────────────────────────

// InviteUser handles POST /api/v1/organizations/:id/invitations.
//
// @Summary      Invite a user to an organization
// @Description  Sends an email invitation to the given address. If a pending invitation for that email already exists it is refreshed (expiry reset, email re-sent) rather than creating a duplicate. Restricted to `owner` or `admin`.
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                 true  "Organization UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000010)
// @Param        body  body      dto.InviteUserRequest  true  "Invitation details"
// @Success      202   {object}  dto.MessageResponse    "Invitation sent"
// @Failure      400   {object}  dto.ErrorResponse      "Invalid UUID format"
// @Failure      401   {object}  dto.ErrorResponse      "Missing or invalid token"
// @Failure      403   {object}  dto.ErrorResponse      "Insufficient role (FORBIDDEN)"
// @Failure      404   {object}  dto.ErrorResponse      "Organization not found or access denied"
// @Failure      409   {object}  dto.ErrorResponse      "User is already a member (ALREADY_A_MEMBER)"
// @Failure      422   {object}  dto.ErrorResponse      "Validation error"
// @Failure      500   {object}  dto.ErrorResponse      "Internal server error"
// @Router       /api/v1/organizations/{id}/invitations [post]
func (h *OrganizationHandler) InviteUser(c *gin.Context) {
	orgID, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	var req dto.InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.InviteUser(c.Request.Context(), orgID, callerID, req.Email, req.Role); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.OK(gin.H{"message": "invitation sent"}))
}

// AcceptInvitation handles POST /api/v1/invitations/accept.
//
// @Summary      Accept an organization invitation
// @Description  Accepts a pending invitation and adds the authenticated user to the organization with the role specified in the invitation. The invitation token is single-use and valid for 7 days.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.AcceptInvitationRequest  true  "Invitation token"
// @Success      200   {object}  dto.OrgResponseEnvelope      "Joined organization successfully"
// @Failure      400   {object}  dto.ErrorResponse            "Token invalid, expired, or already accepted (INVALID_OR_EXPIRED_INVITATION)"
// @Failure      401   {object}  dto.ErrorResponse            "Missing or invalid token"
// @Failure      403   {object}  dto.ErrorResponse            "Authenticated user email does not match invitation email (INVITATION_EMAIL_MISMATCH)"
// @Failure      422   {object}  dto.ErrorResponse            "Validation error"
// @Failure      500   {object}  dto.ErrorResponse            "Internal server error"
// @Router       /api/v1/invitations/accept [post]
func (h *OrganizationHandler) AcceptInvitation(c *gin.Context) {
	var req dto.AcceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	callerID, err := handlerhelpers.CallerID(c)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	org, err := h.service.AcceptInvitation(c.Request.Context(), callerID, req.Token)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(httputils.ToOrgResponse(org)))
}
