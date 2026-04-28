package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rekall/backend/pkg/constants"
)

// RequestID injects a unique request ID into each request context and response header.
// If the client supplies an X-Request-ID header it is reused; otherwise a new UUID is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(constants.HeaderRequestID)
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(constants.CtxKeyRequestID, id)
		c.Header(constants.HeaderRequestID, id)
		c.Next()
	}
}
