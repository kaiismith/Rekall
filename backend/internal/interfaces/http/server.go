package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rekall/backend/pkg/config"
	"go.uber.org/zap"
)

// Server wraps net/http.Server and handles graceful shutdown.
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

// NewServer constructs an HTTP server from the given config and router.
func NewServer(cfg config.ServerConfig, router *gin.Engine, logger *zap.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", cfg.Port),
			Handler:      router,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger: logger,
	}
}

// Start begins listening and serving HTTP requests. It blocks until the server
// stops (either due to an error or a graceful shutdown triggered by ctx).
func (s *Server) Start() error {
	s.logger.Info("http server starting", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

// Shutdown gracefully drains active connections within the context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}
