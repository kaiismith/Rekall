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

// CreateUserRequest is the body expected for POST /api/v1/users.
type CreateUserRequest struct {
	Email    string `json:"email"     binding:"required,email"        example:"bob@example.com"`
	FullName string `json:"full_name" binding:"required,min=1,max=255" example:"Bob Smith"`
	Role     string `json:"role"      binding:"omitempty,oneof=admin member" example:"member" enums:"admin,member"`
}

// UserHandler handles HTTP requests for the /users resource.
type UserHandler struct {
	service *services.UserService
	logger  *zap.Logger
}

// NewUserHandler creates a UserHandler with its required dependencies.
func NewUserHandler(service *services.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{service: service, logger: logger}
}

// List handles GET /api/v1/users.
//
// @Summary      List users  [admin]
// @Description  Returns a paginated list of all platform users. **Requires `admin` role.** Regular members receive 403.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int  false  "Page number (1-based)"    minimum(1)   default(1)
// @Param        per_page  query     int  false  "Number of items per page" minimum(1)   maximum(100) default(20)
// @Success      200  {object}  dto.UserListResponse  "Paginated user list"
// @Failure      401  {object}  dto.ErrorResponse     "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse     "Insufficient role — admin required (FORBIDDEN)"
// @Failure      500  {object}  dto.ErrorResponse     "Internal server error"
// @Router       /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	page := httputils.QueryInt(c, "page", 1)
	perPage := httputils.QueryInt(c, "per_page", 20)

	users, total, err := h.service.ListUsers(c.Request.Context(), page, perPage)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated(users, page, perPage, total))
}

// Get handles GET /api/v1/users/:id.
//
// @Summary      Get a user  [admin]
// @Description  Returns a single user profile by UUID. **Requires `admin` role.**
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                    true  "User UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000001)
// @Success      200  {object}  dto.UserResponseEnvelope  "User profile"
// @Failure      400  {object}  dto.ErrorResponse         "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse         "Insufficient role — admin required (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse         "User not found (USER_NOT_FOUND)"
// @Failure      500  {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(user))
}

// Create handles POST /api/v1/users.
//
// @Summary      Create a user  [admin]
// @Description  Creates a new platform user directly (admin operation — bypasses the self-registration flow). **Requires `admin` role.**
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CreateUserRequest         true  "User payload"
// @Success      201   {object}  dto.UserResponseEnvelope  "User created"
// @Failure      401   {object}  dto.ErrorResponse         "Missing or invalid token"
// @Failure      403   {object}  dto.ErrorResponse         "Insufficient role — admin required (FORBIDDEN)"
// @Failure      409   {object}  dto.ErrorResponse         "Email already registered (EMAIL_ALREADY_REGISTERED)"
// @Failure      422   {object}  dto.ErrorResponse         "Validation error"
// @Failure      500   {object}  dto.ErrorResponse         "Internal server error"
// @Router       /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerhelpers.RespondError(c, h.logger, apperr.Unprocessable("invalid request body", err.Error()))
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req.Email, req.FullName, req.Role)
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusCreated, dto.OK(user))
}

// Delete handles DELETE /api/v1/users/:id.
//
// @Summary      Delete a user  [admin]
// @Description  Soft-deletes a platform user. **Requires `admin` role.**
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "User UUID"  format(uuid)  example(00000000-0000-0000-0000-000000000001)
// @Success      204  "User deleted"
// @Failure      400  {object}  dto.ErrorResponse  "Invalid UUID format"
// @Failure      401  {object}  dto.ErrorResponse  "Missing or invalid token"
// @Failure      403  {object}  dto.ErrorResponse  "Insufficient role — admin required (FORBIDDEN)"
// @Failure      404  {object}  dto.ErrorResponse  "User not found (USER_NOT_FOUND)"
// @Failure      500  {object}  dto.ErrorResponse  "Internal server error"
// @Router       /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := httputils.ParseUUID(c, "id")
	if err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), id); err != nil {
		handlerhelpers.RespondError(c, h.logger, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
