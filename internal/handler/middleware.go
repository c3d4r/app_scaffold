package handler

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

func (h *Handler) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if h.sessionStore == nil {
			return next(c)
		}

		cookie, err := c.Cookie("scaffold_session")
		if err != nil {
			return redirectMissingSession(c, h.cognito == nil)
		}

		session, err := h.sessionStore.GetSession(c.Request().Context(), cookie.Value)
		if err != nil || session == nil || session.IsExpired() {
			c.SetCookie(&http.Cookie{
				Name:     "scaffold_session",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				MaxAge:   -1,
			})
			return redirectMissingSession(c, h.cognito == nil)
		}

		c.Set("session", session)
		return next(c)
	}
}

func redirectMissingSession(c echo.Context, isDev bool) error {
	u := "/auth/login?redirect=" + url.QueryEscape(c.Request().URL.String())
	if isDev {
		u = "/auth/dev-sign-in?redirect=" + url.QueryEscape(c.Request().URL.String())
	}
	return c.Redirect(http.StatusFound, u)
}
