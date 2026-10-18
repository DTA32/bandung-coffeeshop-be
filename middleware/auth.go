package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

const userContextKey = "bdgcafe.user"

// LoadUser attaches the session's user to the context when there is one.
// Requests without a session cookie never touch Redis or the DB; any lookup
// failure degrades to anonymous.
func LoadUser(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(constants.SessionCookie)
		if err != nil || token == "" {
			c.Next()
			return
		}
		u, err := svc.UserFromSession(c.Request.Context(), token)
		if err != nil {
			if !errors.Is(err, service.ErrNoSession) {
				log.Printf("auth: session lookup failed: %v", err)
			}
			c.Next()
			return
		}
		c.Set(userContextKey, u)
		c.Next()
	}
}

func RequireUser(c *gin.Context) {
	if _, ok := CurrentUser(c); !ok {
		helper.Error(c, http.StatusUnauthorized, "unauthorized")
		c.Abort()
		return
	}
	c.Next()
}

func CurrentUser(c *gin.Context) (*model.User, bool) {
	v, ok := c.Get(userContextKey)
	if !ok {
		return nil, false
	}
	u, ok := v.(*model.User)
	return u, ok && u != nil
}
