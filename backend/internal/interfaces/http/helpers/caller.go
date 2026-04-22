package handlerhelpers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/interfaces/http/middleware"
	apperr "github.com/rekall/backend/pkg/errors"
)

// CallerID extracts the authenticated user's UUID from the Gin context.
func CallerID(c *gin.Context) (uuid.UUID, error) {
	claims := middleware.ClaimsFromContext(c)
	if claims == nil {
		return uuid.Nil, apperr.Unauthorized("authentication required")
	}
	id, err := claims.SubjectAsUUID()
	if err != nil {
		return uuid.Nil, apperr.Unauthorized("invalid token subject")
	}
	return id, nil
}
