package handlerhelpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

// RespondError writes an appropriate JSON error response for err.
func RespondError(c *gin.Context, logger *zap.Logger, err error) {
	if appErr, ok := apperr.AsAppError(err); ok {
		c.JSON(appErr.Status, dto.Err(appErr.Code, appErr.Message, appErr.Details))
		return
	}
	catalog.HTTPServerError.Error(logger, zap.Error(err))
	c.JSON(http.StatusInternalServerError, dto.Err("INTERNAL_ERROR", "an unexpected error occurred", nil))
}
