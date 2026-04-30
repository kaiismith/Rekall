package http

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/interfaces/http/handlers"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	"github.com/rekall/backend/pkg/constants"

	_ "github.com/rekall/backend/docs" // swagger docs registration
)

// RouterDeps bundles all handler dependencies for route registration.
type RouterDeps struct {
	Logger         *zap.Logger
	JWTSecret      string
	JWTIssuer      string
	HealthH        *handlers.HealthHandler
	KatH           *handlers.KatHandler
	CallH          *handlers.CallHandler
	UserH          *handlers.UserHandler
	AuthH          *handlers.AuthHandler
	OrgH           *handlers.OrganizationHandler
	DeptH          *handlers.DepartmentHandler
	MeetingH       *handlers.MeetingHandler
	ASRH           *handlers.ASRHandler
	TranscriptH    *handlers.TranscriptHandler
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
	if deps.KatH != nil {
		router.GET("/healthz/kat", deps.KatH.Health)
	}

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
		protected.PATCH("/auth/me", deps.AuthH.UpdateMe)
		protected.POST("/auth/password/change", deps.AuthH.ChangePassword)

		// Calls
		calls := protected.Group("/calls")
		{
			calls.GET("", deps.CallH.List)
			calls.POST("", deps.CallH.Create)
			calls.GET("/:id", deps.CallH.Get)
			calls.PATCH("/:id", deps.CallH.Update)
			calls.DELETE("/:id", deps.CallH.Delete)

			// ASR live-captions session endpoints. The handler returns
			// 503 ASR_NOT_CONFIGURED when the feature flag is off, so
			// these routes are always wired and the frontend probes them.
			if deps.ASRH != nil {
				calls.POST("/:id/asr-session", deps.ASRH.Request)
				calls.POST("/:id/asr-session/end", deps.ASRH.End)
				calls.POST("/:id/asr-session/:session_id/segments", deps.ASRH.PostCallSegment)
			}
			if deps.TranscriptH != nil {
				calls.GET("/:id/transcript", deps.TranscriptH.GetCallTranscript)
			}
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
			// Org creation is reserved for platform admins. Per-org owners are
			// derived from the request body (caller becomes owner unless they
			// pass an `owner_email` to create on someone else's behalf).
			orgs.POST("", middleware.RequireRole(constants.UserRoleAdmin), deps.OrgH.Create)
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
				// Ticket endpoint — authenticated via bearer; returns a short-lived
				// ticket that the WS upgrade consumes in place of the JWT.
				meetings.POST("/:code/ws-ticket", deps.MeetingH.IssueWSTicket)

				// ASR live-captions session endpoints — gated by both
				// ASR_FEATURE_ENABLED on the backend AND the per-meeting
				// `transcription_enabled` flag set by the host at creation.
				if deps.ASRH != nil {
					meetings.POST("/:code/asr-session", deps.ASRH.RequestForMeeting)
					meetings.POST("/:code/asr-session/end", deps.ASRH.EndForMeeting)
				}
				if deps.TranscriptH != nil {
					meetings.GET("/:code/transcript", deps.TranscriptH.GetMeetingTranscript)
				}
			}
			// WebSocket — no JWT middleware; authenticates via the ticket.
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
