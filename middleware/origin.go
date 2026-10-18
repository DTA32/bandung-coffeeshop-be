package middleware

import (
	"net/http"
	"slices"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/gin-gonic/gin"
)

// RequireAllowedOrigin rejects state-changing requests whose Origin header is
// present but not allowlisted (CSRF guard alongside SameSite=Lax cookies).
func RequireAllowedOrigin(allowed []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			if origin := c.GetHeader("Origin"); origin != "" && !slices.Contains(allowed, origin) {
				helper.Error(c, http.StatusForbidden, "origin not allowed")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
