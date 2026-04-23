package http

import (
	"github.com/gin-gonic/gin"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	"github.com/rekall/backend/pkg/constants"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	"go.uber.org/zap"

	_ "github.com/rekall/backend/docs" // swagger docs registration
)

// RouterDeps bundles all handler dependencies for route registration.
type RouterDeps struct {
	Logger         *zap.Logger
	JWTSecret      string
	JWTIssuer      string
	HealthH        *handlers.HealthHandler
	CallH          *handlers.CallHandler
	UserH          *handlers.UserHandler
	AuthH          *handlers.AuthHandler
	OrgH           *handlers.OrganizationHandler
	DeptH          *handlers.DepartmentHandler
	MeetingH       *handlers.MeetingHandler
	CORSOrigins    []string
	SwaggerEnabled bool
}

// NewRouter builds and returns a configured Gin engine.
func NewRouter(deps RouterDeps) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.Recovery(deps.Logger))
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(deps.Logger))
	router.Use(middleware.CORS(deps.CORSOrigins))

	// Probe endpoints (no /api/v1 prefix — used by load balancers)
	router.GET("/health", deps.HealthH.Liveness)
	router.GET("/ready", deps.HealthH.Readiness)

	// API v1
	v1 := router.Group("/api/v1")

	// ── Auth (public) ─────────────────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", deps.AuthH.Register)
		auth.POST("/login", deps.AuthH.Login)
		auth.POST("/logout", deps.AuthH.Logout)
		auth.POST("/refresh", deps.AuthH.Refresh)
		auth.GET("/verify", deps.AuthH.VerifyEmail)
		auth.POST("/verify/resend", deps.AuthH.ResendVerification)
		auth.POST("/password/forgot", deps.AuthH.ForgotPassword)
		auth.POST("/password/reset", deps.AuthH.ResetPassword)
	}

	// ── Protected routes — require valid JWT ──────────────────────────────────
	protected := v1.Group("")
	protected.Use(middleware.Authenticate(deps.JWTSecret, deps.JWTIssuer, deps.Logger))
	{
		// Current user
		protected.GET("/auth/me", deps.AuthH.Me)

		// Calls
		calls := protected.Group("/calls")
		{
			calls.GET("", deps.CallH.List)
			calls.POST("", deps.CallH.Create)
			calls.GET("/:id", deps.CallH.Get)
			calls.PATCH("/:id", deps.CallH.Update)
			calls.DELETE("/:id", deps.CallH.Delete)
		}

		// Users — platform-admin only
		users := protected.Group("/users")
		users.Use(middleware.RequireRole(constants.UserRoleAdmin))
		{
			users.GET("", deps.UserH.List)
			users.POST("", deps.UserH.Create)
			users.GET("/:id", deps.UserH.Get)
			users.DELETE("/:id", deps.UserH.Delete)
		}

		// Organizations
		orgs := protected.Group("/organizations")
		{
			orgs.GET("", deps.OrgH.List)
			orgs.POST("", deps.OrgH.Create)
			orgs.GET("/:id", deps.OrgH.Get)
			orgs.PATCH("/:id", deps.OrgH.Update)
			orgs.DELETE("/:id", deps.OrgH.Delete)

			// Members
			orgs.GET("/:id/members", deps.OrgH.ListMembers)
			orgs.PATCH("/:id/members/:userID", deps.OrgH.UpdateMember)
			orgs.DELETE("/:id/members/:userID", deps.OrgH.RemoveMember)

			// Invitations
			orgs.POST("/:id/invitations", deps.OrgH.InviteUser)

			// Departments (nested under org for creation and listing)
			orgs.GET("/:id/departments", deps.DeptH.ListByOrg)
			orgs.POST("/:id/departments", deps.DeptH.Create)
		}

		// Meetings
		if deps.MeetingH != nil {
			meetings := protected.Group("/meetings")
			{
				meetings.POST("", deps.MeetingH.Create)
				meetings.GET("/mine", deps.MeetingH.ListMine)
				meetings.GET("/:code", deps.MeetingH.GetByCode)
				meetings.DELETE("/:code", deps.MeetingH.End)
				meetings.GET("/:code/messages", deps.MeetingH.ListMessages)
			}
			// WebSocket — no JWT middleware (token passed as query param)
			v1.GET("/meetings/:code/ws", deps.MeetingH.Connect)
		}

		// Invitation acceptance (cross-org endpoint)
		protected.POST("/invitations/accept", deps.OrgH.AcceptInvitation)

		// Departments (standalone routes for get/update/delete and membership management)
		depts := protected.Group("/departments")
		{
			depts.GET("/:deptID", deps.DeptH.Get)
			depts.PATCH("/:deptID", deps.DeptH.Update)
			depts.DELETE("/:deptID", deps.DeptH.Delete)

			depts.GET("/:deptID/members", deps.DeptH.ListMembers)
			depts.POST("/:deptID/members", deps.DeptH.AddMember)
			depts.PATCH("/:deptID/members/:userID", deps.DeptH.UpdateMember)
			depts.DELETE("/:deptID/members/:userID", deps.DeptH.RemoveMember)
		}
	}

	// ── Swagger UI (development / opt-in only) ────────────────────────────────
	if deps.SwaggerEnabled {
		router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	return router
}
