// @title           Rekall API
// @version         1.0.0
// @description     Rekall is a call intelligence platform. Record, transcribe, and analyse voice and video calls.
// @description     This API powers the Rekall web application and is available for direct integration.
// @description
// @description     ## Authentication
// @description     Protected endpoints require a JWT access token obtained from `POST /api/v1/auth/login`.
// @description     Pass the token in the `Authorization` header as `Bearer <token>`.
// @description     The refresh token is transported exclusively via an `HttpOnly` cookie set by the server.
// @description
// @description     ## Response Envelope
// @description     All responses are wrapped in a standard envelope:
// @description     - Success: `{ "success": true, "data": <T>, "meta": <pagination|null> }`
// @description     - Error:   `{ "success": false, "error": { "code": "ERROR_CODE", "message": "Human readable message", "details": <null|object> } }`
// @description
// @description     ## Error Codes
// @description     Every error response includes a machine-readable `code` field. Common codes:
// @description     `INVALID_CREDENTIALS`, `EMAIL_NOT_VERIFIED`, `EMAIL_ALREADY_REGISTERED`,
// @description     `INVALID_OR_EXPIRED_TOKEN`, `INVALID_REFRESH_TOKEN`, `MISSING_TOKEN`, `INVALID_TOKEN`,
// @description     `CALL_NOT_FOUND`, `USER_NOT_FOUND`, `SLUG_ALREADY_TAKEN`, `ALREADY_A_MEMBER`,
// @description     `CANNOT_REMOVE_OWNER`, `INVITATION_EMAIL_MISMATCH`, `DEPARTMENT_NOT_FOUND`, `FORBIDDEN`

// @contact.name    Rekall Engineering
// @contact.email   engineering@rekall.io

// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT

// @host        localhost:8080
// @BasePath    /
// @schemes     http https

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Enter your JWT access token as: **Bearer &lt;token&gt;**

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/infrastructure/database"
	infraemail "github.com/rekall/backend/internal/infrastructure/email"
	"github.com/rekall/backend/internal/infrastructure/repositories"
	"github.com/rekall/backend/internal/infrastructure/storage"
	httpserver "github.com/rekall/backend/internal/interfaces/http"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
	"github.com/rekall/backend/pkg/config"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

func main() {
	// ── Configuration ────────────────────────────────────────────────────────
	// Load first — the logger depends on config.Logger, so errors go to stderr.
	cfg, err := config.Load()
	if err != nil {
		_, _ = os.Stderr.WriteString("FATAL [SYS_CONFIG_INVALID] " + err.Error() + "\n")
		os.Exit(1)
	}

	// ── Logger ───────────────────────────────────────────────────────────────
	log, err := applogger.New(cfg.Logger.Level, cfg.Logger.Format, cfg.Server.IsDevelopment())
	if err != nil {
		_, _ = os.Stderr.WriteString("FATAL [SYS_LOGGER_FAILED] " + err.Error() + "\n")
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	// First structured line — intentionally verbose for ops dashboards.
	catalog.SysInitialising.Info(log,
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
		zap.String("log_level", cfg.Logger.Level),
		zap.String("log_format", cfg.Logger.Format),
		zap.Duration("read_timeout", cfg.Server.ReadTimeout),
		zap.Duration("write_timeout", cfg.Server.WriteTimeout),
		zap.Duration("idle_timeout", cfg.Server.IdleTimeout),
	)

	catalog.SysConfigLoaded.Info(log,
		zap.String("db_host", cfg.Database.Host),
		zap.String("db_port", cfg.Database.Port),
		zap.String("db_name", cfg.Database.DBName),
		zap.String("db_ssl_mode", cfg.Database.SSLMode),
		zap.Int("db_max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("db_max_idle_conns", cfg.Database.MaxIdleConns),
		zap.Duration("db_conn_max_lifetime", cfg.Database.ConnMaxLifetime),
		zap.Strings("cors_origins", cfg.CORS.AllowedOrigins),
	)

	catalog.SysLoggerReady.Info(log,
		zap.String("level", cfg.Logger.Level),
		zap.String("format", cfg.Logger.Format),
	)

	// ── Database (GORM) ──────────────────────────────────────────────────────
	catalog.DBPoolOpening.Info(log,
		zap.String("host", cfg.Database.Host),
		zap.String("database", cfg.Database.DBName),
		zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
	)

	db, err := database.New(cfg.Database, cfg.Server.IsDevelopment())
	if err != nil {
		catalog.DBConnFailed.Fatal(log,
			zap.Error(err),
			zap.String("host", cfg.Database.Host),
			zap.String("database", cfg.Database.DBName),
		)
	}
	sqlDB, _ := db.DB()
	defer func() {
		_ = sqlDB.Close()
		catalog.DBPoolClosed.Info(log)
	}()

	catalog.DBConnected.Info(log,
		zap.String("host", cfg.Database.Host),
		zap.String("database", cfg.Database.DBName),
		zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
		zap.Duration("conn_max_lifetime", cfg.Database.ConnMaxLifetime),
	)

	// ── Repositories ─────────────────────────────────────────────────────────
	callRepo            := repositories.NewCallRepository(db)
	userRepo            := repositories.NewUserRepository(db)
	tokenRepo           := repositories.NewTokenRepository(db)
	orgRepo             := repositories.NewOrganizationRepository(db)
	memberRepo          := repositories.NewOrgMembershipRepository(db)
	inviteRepo          := repositories.NewInvitationRepository(db)
	deptRepo            := repositories.NewDepartmentRepository(db)
	deptMemberRepo      := repositories.NewDepartmentMembershipRepository(db)
	meetingRepo         := repositories.NewMeetingRepository(db)
	meetingParticipRepo := repositories.NewMeetingParticipantRepository(db)
	meetingMessageRepo  := repositories.NewMeetingMessageRepository(db)

	// ── Infrastructure ────────────────────────────────────────────────────────
	mailer := infraemail.NewSMTPSender(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
		cfg.SMTP.UseTLS,
		log,
	)

	// ── Services ─────────────────────────────────────────────────────────────
	callSvc := services.NewCallService(callRepo, log)
	userSvc := services.NewUserService(userRepo, log)
	authSvc := services.NewAuthService(
		userRepo,
		tokenRepo,
		mailer,
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTIssuer,
		cfg.Auth.AppBaseURL,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
		cfg.Auth.PasswordResetTTL,
		cfg.Auth.EmailVerifyTTL,
		log,
	)
	orgSvc := services.NewOrganizationService(
		orgRepo,
		memberRepo,
		inviteRepo,
		userRepo,
		mailer,
		cfg.Auth.AppBaseURL,
		cfg.Auth.InvitationTTL,
		log,
	)
	deptSvc := services.NewDepartmentService(deptRepo, deptMemberRepo, memberRepo, log)
	meetingSvc := services.NewMeetingService(
		meetingRepo,
		meetingParticipRepo,
		memberRepo,
		deptMemberRepo,
		cfg.Auth.AppBaseURL,
		log,
	)
	chatMessageSvc := services.NewChatMessageService(
		meetingRepo,
		meetingParticipRepo,
		meetingMessageRepo,
		log,
	)
	cleanupJob := services.NewMeetingCleanupJob(
		meetingRepo,
		meetingParticipRepo,
		services.MeetingCleanupConfig{
			Interval:       cfg.Meeting.CleanupInterval,
			WaitingTimeout: cfg.Meeting.WaitingTimeout,
			MaxDuration:    cfg.Meeting.MaxDuration,
		},
		log,
	)

	// ── WebSocket Hub Manager ─────────────────────────────────────────────────
	hubManager := wsHub.NewHubManager(meetingMessageRepo, log)
	defer hubManager.Shutdown()

	// ── WebSocket ticket store (secure WS auth) ───────────────────────────────
	wsTicketStore := storage.NewMemoryWSTicketStore(log)
	defer wsTicketStore.Close()

	// ── Handlers ─────────────────────────────────────────────────────────────
	healthH  := handlers.NewHealthHandler(db)
	callH    := handlers.NewCallHandler(callSvc, log)
	userH    := handlers.NewUserHandler(userSvc, log)
	authH    := handlers.NewAuthHandler(authSvc, cfg.Auth.RefreshTokenTTL, log)
	orgH     := handlers.NewOrganizationHandler(orgSvc, log)
	deptH    := handlers.NewDepartmentHandler(deptSvc, log)
	meetingH := handlers.NewMeetingHandler(meetingSvc, chatMessageSvc, userSvc, hubManager, wsTicketStore, cfg.Auth.AppBaseURL, log)

	// ── Router ───────────────────────────────────────────────────────────────
	ginMode := gin.DebugMode
	if !cfg.Server.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
		ginMode = gin.ReleaseMode
	}

	router := httpserver.NewRouter(httpserver.RouterDeps{
		Logger:         log,
		JWTSecret:      cfg.Auth.JWTSecret,
		JWTIssuer:      cfg.Auth.JWTIssuer,
		HealthH:        healthH,
		CallH:          callH,
		UserH:          userH,
		AuthH:          authH,
		OrgH:           orgH,
		DeptH:          deptH,
		MeetingH:       meetingH,
		CORSOrigins:    cfg.CORS.AllowedOrigins,
		SwaggerEnabled: cfg.Server.SwaggerEnabled,
	})

	catalog.SysRouterConfigured.Info(log,
		zap.String("gin_mode", ginMode),
		zap.Strings("cors_origins", cfg.CORS.AllowedOrigins),
		zap.String("api_prefix", "/api/v1"),
	)

	// ── Background jobs ───────────────────────────────────────────────────────
	jobCtx, cancelJobs := context.WithCancel(context.Background())
	defer cancelJobs()
	go cleanupJob.Run(jobCtx)

	// ── Server ───────────────────────────────────────────────────────────────
	srv := httpserver.NewServer(cfg.Server, router, log)

	serverErr := make(chan error, 1)
	go func() {
		catalog.HTTPServerStarting.Info(log, zap.String("addr", ":"+cfg.Server.Port))
		serverErr <- srv.Start()
	}()

	catalog.SysReady.Info(log,
		zap.String("addr", ":"+cfg.Server.Port),
		zap.String("env", cfg.Server.Env),
	)

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		catalog.HTTPServerError.Error(log,
			zap.Error(err),
			zap.String("addr", ":"+cfg.Server.Port),
		)
	case sig := <-quit:
		catalog.SysShutdownSignal.Info(log,
			zap.String("signal", sig.String()),
		)
	}

	const drainTimeout = 30 * time.Second
	catalog.SysShutdownDraining.Info(log,
		zap.Duration("drain_timeout", drainTimeout),
	)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		catalog.SysShutdownTimeout.Error(log,
			zap.Error(err),
			zap.Duration("drain_timeout", drainTimeout),
		)
	} else {
		catalog.SysShutdownOK.Info(log)
	}
}
