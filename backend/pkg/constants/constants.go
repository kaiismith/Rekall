package constants

// Environment names
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTest        = "test"
)

// Pagination defaults
const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Context keys used to pass values through Gin context / request context.
const (
	CtxKeyRequestID = "request_id"
	CtxKeyLogger    = "logger"
	CtxKeyUserID    = "user_id"
)

// Call status values
const (
	CallStatusPending    = "pending"
	CallStatusProcessing = "processing"
	CallStatusDone       = "done"
	CallStatusFailed     = "failed"
)

// User role values
const (
	UserRoleAdmin  = "admin"
	UserRoleMember = "member"
)

// contextKey is an unexported type that prevents key collisions across packages.
type contextKey string

// Typed context keys for values stored in Gin / request context.
const (
	ContextKeyAuthUser = contextKey("auth_user")
)

// OrgMembership role values
const (
	OrgRoleOwner  = "owner"
	OrgRoleAdmin  = "admin"
	OrgRoleMember = "member"
)

// DepartmentMembership role values
const (
	DeptRoleHead   = "head"
	DeptRoleMember = "member"
)

// Cookie names
const (
	CookieRefreshToken = "refresh_token"
)

// API versioning
const (
	APIV1Prefix = "/api/v1"
)

// Header names
const (
	HeaderRequestID   = "X-Request-ID"
	HeaderContentType = "Content-Type"
)
