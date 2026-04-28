package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/interfaces/http/dto"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// Recovery returns a Gin middleware that catches panics, logs them via the
// event catalog with a full stack trace and request context, then returns a
// sanitised 500 JSON response so the server remains alive.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	log = applogger.WithComponent(log, "http.recovery")

	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				catalog.HTTPPanic.Error(log,
					zap.Any("panic_value", rec),
					zap.Stack("stack_trace"),
					zap.String("request_id", c.GetString("request_id")),
					zap.String("method", c.Request.Method),
					zap.String("route", c.FullPath()),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
					zap.String("user_agent", c.Request.UserAgent()),
				)
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					dto.Err("INTERNAL_ERROR", "an unexpected error occurred", nil),
				)
			}
		}()
		c.Next()
	}
}
