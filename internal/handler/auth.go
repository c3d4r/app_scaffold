package handler

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/c3d4r/app_scaffold/internal/auth"
)

func (h *Handler) handleLogin(c echo.Context) error {
	if h.cognito == nil {
		return c.String(http.StatusInternalServerError, "auth not configured")
	}

	redirectURI := h.callbackURL
	if redirectURI == "" {
		redirectURI = schemeHost(c.Request()) + "/auth/callback"
	}

	if dest := c.QueryParam("redirect"); dest != "" {
		c.SetCookie(&http.Cookie{
			Name:     "scaffold_redirect",
			Value:    dest,
			Path:     "/",
			HttpOnly: true,
			Secure:   c.Request().TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   300,
		})
	}

	return c.Redirect(http.StatusFound, h.cognito.LoginURL(redirectURI))
}

func (h *Handler) handleCallback(c echo.Context) error {
	if h.cognito == nil {
		return c.String(http.StatusInternalServerError, "auth not configured")
	}

	if errParam := c.QueryParam("error"); errParam != "" {
		desc := c.QueryParam("error_description")
		log.Printf("auth callback: cognito error: %s: %s", errParam, desc)
		return c.String(http.StatusUnauthorized, "authentication failed")
	}

	code := c.QueryParam("code")
	if code == "" {
		return c.String(http.StatusBadRequest, "missing authorization code")
	}

	redirectURI := h.callbackURL
	if redirectURI == "" {
		redirectURI = schemeHost(c.Request()) + "/auth/callback"
	}

	tokens, err := h.cognito.ExchangeCode(c.Request().Context(), code, redirectURI)
	if err != nil {
		log.Printf("auth callback: exchange code: %v", err)
		return c.String(http.StatusInternalServerError, "failed to exchange authorization code")
	}

	claims, err := h.cognito.VerifyIDToken(c.Request().Context(), tokens.IDToken)
	if err != nil {
		log.Printf("auth callback: verify id token: %v", err)
		return c.String(http.StatusInternalServerError, "failed to verify identity")
	}

	username := claims.Username
	if username == "" {
		username = claims.Email
	}

	session := auth.NewSession(claims.Sub, username, claims.Email)
	if err := h.sessionStore.PutSession(c.Request().Context(), session); err != nil {
		log.Printf("auth callback: save session: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create session")
	}

	c.SetCookie(&http.Cookie{
		Name:     "scaffold_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	dest := "/default"
	if cookie, err := c.Cookie("scaffold_redirect"); err == nil {
		dest = cookie.Value
		c.SetCookie(&http.Cookie{
			Name:     "scaffold_redirect",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
	}

	return c.Redirect(http.StatusFound, dest)
}

func (h *Handler) handleLogout(c echo.Context) error {
	if cookie, err := c.Cookie("scaffold_session"); err == nil && h.sessionStore != nil {
		h.sessionStore.DeleteSession(c.Request().Context(), cookie.Value)
	}

	c.SetCookie(&http.Cookie{
		Name:     "scaffold_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	return c.Redirect(http.StatusFound, "/about")
}

func schemeHost(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	return scheme + "://" + r.Host
}

func (h *Handler) handleSignInDev(c echo.Context) error {
	session := auth.NewSession("dev-user", "Developer", "dev@localhost")

	if err := h.sessionStore.PutSession(c.Request().Context(), session); err != nil {
		log.Printf("dev sign-in: save session: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create session")
	}

	c.SetCookie(&http.Cookie{
		Name:     "scaffold_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	dest := c.QueryParam("redirect")
	if dest == "" {
		dest = "/default"
	}
	return c.Redirect(http.StatusFound, dest)
}
