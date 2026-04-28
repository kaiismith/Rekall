package handlerhelpers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rekall/backend/pkg/constants"
)

// SetRefreshCookie writes a Secure, HttpOnly refresh-token cookie.
func SetRefreshCookie(c *gin.Context, rawToken string, ttl time.Duration) {
	c.SetCookie(
		constants.CookieRefreshToken,
		rawToken,
		int(ttl.Seconds()),
		"/",
		"",   // domain — let browser infer
		true, // Secure
		true, // HttpOnly
	)
	c.SetSameSite(http.SameSiteStrictMode)
}

// ClearRefreshCookie expires the refresh-token cookie immediately.
func ClearRefreshCookie(c *gin.Context) {
	c.SetCookie(constants.CookieRefreshToken, "", -1, "/", "", true, true)
}
