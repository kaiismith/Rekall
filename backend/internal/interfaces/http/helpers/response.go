package handlerhelpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/interfaces/http/dto"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// RespondError writes an appropriate JSON error response for err.
// When err carries a positive RetryAfterSeconds (e.g. from a 503 with
// back-off guidance) the standard `Retry-After` header is included.
func RespondError(c *gin.Context, logger *zap.Logger, err error) {
	if appErr, ok := apperr.AsAppError(err); ok {
		if appErr.RetryAfterSeconds > 0 {
			c.Header("Retry-After", strconvI(appErr.RetryAfterSeconds))
		}
		c.JSON(appErr.Status, dto.Err(appErr.Code, appErr.Message, appErr.Details))
		return
	}
	catalog.HTTPServerError.Error(logger, zap.Error(err))
	c.JSON(http.StatusInternalServerError, dto.Err("INTERNAL_ERROR", "an unexpected error occurred", nil))
}

// strconvI inlines strconv.Itoa to avoid an import churn that touches every
// helper file in the package; the digit conversion is hot-path-friendly.
func strconvI(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
