package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	infraauth "github.com/rekall/backend/internal/infrastructure/auth"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// contextAuthUserKey is the Gin context key used to store the parsed JWT claims.
const contextAuthUserKey = "auth_claims"

// Authenticate is a Gin middleware that validates the Bearer JWT in the Authorization header.
// On success it stores the *infraauth.Claims under the "auth_claims" key.
// On failure it aborts with 401 Unauthorized.
func Authenticate(jwtSecret, jwtIssuer string, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			catalog.TokenMissing.Warn(log)
			c.AbortWithStatusJSON(401, apperr.Unauthorized("authentication required"))
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			catalog.TokenInvalid.Warn(log)
			c.AbortWithStatusJSON(401, apperr.Unauthorized("malformed authorization header"))
			return
		}

		claims, err := infraauth.ParseAccessToken(parts[1], jwtSecret, jwtIssuer)
		if err != nil {
			catalog.TokenInvalid.Warn(log, zap.Error(err))
			c.AbortWithStatusJSON(401, apperr.Unauthorized("invalid or expired token"))
			return
		}

		c.Set(contextAuthUserKey, claims)
		c.Next()
	}
}

// RequireRole returns a middleware that aborts with 403 if the authenticated user's role
// is not in the allowed list. Must be placed after Authenticate in the middleware chain.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[strings.ToLower(r)] = struct{}{}
	}

	return func(c *gin.Context) {
		claims := ClaimsFromContext(c)
		if claims == nil {
			c.AbortWithStatusJSON(401, apperr.Unauthorized("authentication required"))
			return
		}
		if _, ok := allowed[strings.ToLower(claims.Role)]; !ok {
			c.AbortWithStatusJSON(403, apperr.Forbidden("insufficient permissions"))
			return
		}
		c.Next()
	}
}

// ClaimsFromContext returns the JWT claims stored by Authenticate, or nil if absent.
func ClaimsFromContext(c *gin.Context) *infraauth.Claims {
	v, exists := c.Get(contextAuthUserKey)
	if !exists {
		return nil
	}
	claims, _ := v.(*infraauth.Claims)
	return claims
}
