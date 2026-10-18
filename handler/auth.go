package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/middleware"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

const oauthCookieMaxAge = 10 * 60

type AuthHandler struct {
	svc           *service.AuthService
	publicBaseURL string
	cookieSecure  bool
}

func NewAuthHandler(svc *service.AuthService, publicBaseURL string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{svc: svc, publicBaseURL: publicBaseURL, cookieSecure: cookieSecure}
}

func (h *AuthHandler) Login(c *gin.Context) {
	if c.Param("provider") != constants.AuthProviderGoogle {
		helper.Error(c, http.StatusNotFound, "provider not found")
		return
	}
	authURL, st, err := h.svc.BeginLogin(c.Request.Context(), c.Query("redirect"))
	if errors.Is(err, service.ErrAuthUnavailable) {
		helper.Error(c, http.StatusServiceUnavailable, "auth unavailable")
		return
	}
	if err != nil {
		log.Printf("auth: begin login failed: %v", err)
		helper.Error(c, http.StatusInternalServerError, "failed to start login")
		return
	}
	b, _ := json.Marshal(st)
	h.setCookie(c, constants.OAuthCookie, base64.RawURLEncoding.EncodeToString(b), oauthCookieMaxAge)
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	if c.Param("provider") != constants.AuthProviderGoogle {
		helper.Error(c, http.StatusNotFound, "provider not found")
		return
	}
	st := readLoginState(c)
	h.setCookie(c, constants.OAuthCookie, "", -1)
	c.Header("Cache-Control", "no-store")
	redirect := service.SafeRedirect(st.Redirect)

	if c.Query("error") != "" {
		h.redirect(c, redirect, constants.AuthErrorFailed)
		return
	}
	_, token, err := h.svc.CompleteLogin(c.Request.Context(), c.Query("code"), c.Query("state"), st)
	switch {
	case errors.Is(err, service.ErrAuthUnavailable):
		helper.Error(c, http.StatusServiceUnavailable, "auth unavailable")
	case errors.Is(err, service.ErrDeactivated):
		h.redirect(c, redirect, constants.AuthErrorDeactivated)
	case err != nil:
		log.Printf("auth: callback failed: %v", err)
		h.redirect(c, redirect, constants.AuthErrorFailed)
	default:
		h.setCookie(c, constants.SessionCookie, token, int(repository.SessionTTL.Seconds()))
		h.redirect(c, redirect, "")
	}
}

func (h *AuthHandler) Me(c *gin.Context) {
	u, _ := middleware.CurrentUser(c)
	c.Header("Cache-Control", "no-store")
	helper.Success(c, h.svc.Me(u))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(constants.SessionCookie)
	if err := h.svc.Logout(c.Request.Context(), token); err != nil {
		log.Printf("auth: logout failed: %v", err)
	}
	h.setCookie(c, constants.SessionCookie, "", -1)
	c.Status(http.StatusNoContent)
}

func readLoginState(c *gin.Context) service.LoginState {
	var st service.LoginState
	raw, err := c.Cookie(constants.OAuthCookie)
	if err != nil {
		return st
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return service.LoginState{}
	}
	if json.Unmarshal(b, &st) != nil {
		return service.LoginState{}
	}
	return st
}

func (h *AuthHandler) setCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// redirect sends the browser to PUBLIC_BASE_URL + path, adding auth_error
// when set.
func (h *AuthHandler) redirect(c *gin.Context, path, authError string) {
	target := path
	if authError != "" {
		base, frag, hasFrag := strings.Cut(path, "#")
		sep := "?"
		if strings.Contains(base, "?") {
			sep = "&"
		}
		target = base + sep + "auth_error=" + authError
		if hasFrag {
			target += "#" + frag
		}
	}
	c.Redirect(http.StatusFound, h.publicBaseURL+target)
}
