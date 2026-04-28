package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/infrastructure/database"
	"github.com/rekall/backend/internal/interfaces/http/dto"
)

// HealthHandler handles liveness and readiness probes.
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a HealthHandler with a GORM DB for readiness checks.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness handles GET /health.
//
// @Summary      Liveness probe
// @Description  Returns 200 as long as the server process is running. Used by load balancers and orchestrators to verify the process is alive.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string  "Server is alive"
// @Header       200  {string}  X-Request-ID       "Unique request identifier"
// @Router       /health [get]
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, dto.OK(gin.H{"status": "ok"}))
}

// Readiness handles GET /ready.
//
// @Summary      Readiness probe
// @Description  Returns 200 when the database connection is healthy. Returns 503 if the database is unreachable. Used by orchestrators to gate traffic until the service is fully initialised.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string  "Server is ready and database is reachable"
// @Failure      503  {object}  dto.ErrorResponse  "Database unavailable (DB_UNAVAILABLE)"
// @Header       200  {string}  X-Request-ID       "Unique request identifier"
// @Router       /ready [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	if err := database.Ping(c.Request.Context(), h.db); err != nil {
		c.JSON(http.StatusServiceUnavailable, dto.Err(
			"DB_UNAVAILABLE",
			"database is not reachable",
			nil,
		))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"status": "ready", "database": "ok"}))
}
