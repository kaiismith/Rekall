package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

// Logger returns a Gin middleware that emits a structured access-log entry after
// every request, using the event catalog for consistent event_code tagging.
//
// Fields logged on every request:
//
//	request_id     — correlation ID from RequestID middleware
//	method         — HTTP verb (GET, POST, …)
//	route          — Gin route pattern (e.g. /api/v1/calls/:id) — useful for grouping in aggregators
//	path           — actual resolved URL path
//	query          — raw query string
//	status         — HTTP response status code
//	latency_ms     — handler duration in milliseconds (float64 for sub-ms precision)
//	bytes_written  — response body size in bytes
//	ip             — client IP address
//	user_agent     — User-Agent header
//	proto          — HTTP protocol version (HTTP/1.1, HTTP/2.0 …)
//	content_length — incoming request body size (-1 when unknown)
func Logger(log *zap.Logger) gin.HandlerFunc {
	// Tag all access-log lines with the middleware component.
	log = applogger.WithComponent(log, "http.access_log")

	return func(c *gin.Context) {
		start := time.Now()

		// Capture request metadata before c.Next() mutates the context.
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("request_id", c.GetString("request_id")),
			zap.String("method", c.Request.Method),
			zap.String("route", c.FullPath()),   // e.g. /api/v1/calls/:id
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Float64("latency_ms", latencyMs),
			zap.Int("bytes_written", c.Writer.Size()),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("proto", c.Request.Proto),
			zap.Int64("content_length", c.Request.ContentLength),
		}

		// Gin errors are attached explicitly by handlers; surface them all.
		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				catalog.HTTPServerError.Error(log, append(fields, zap.String("gin_error", e))...)
			}
			return
		}

		switch {
		case status >= 500:
			catalog.HTTPServerError.Error(log, fields...)
		case status >= 400:
			catalog.HTTPClientError.Warn(log, fields...)
		default:
			catalog.HTTPRequest.Info(log, fields...)
		}
	}
}
