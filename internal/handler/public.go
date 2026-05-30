package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handlePublic(c echo.Context) error {
	user := getUser(c)
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return template.PublicPage(user).Render(c.Request().Context(), c.Response().Writer)
}
