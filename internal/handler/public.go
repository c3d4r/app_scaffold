package handler

import (
	"net/http"

	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handlePublic(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.PublicPage(user).Render(r.Context(), w)
}
