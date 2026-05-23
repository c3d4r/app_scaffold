package handler

import (
	"log"
	"net/http"

	"github.com/c3d4r/app_scaffold/internal/models"
	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/default", http.StatusFound)
}

func (h *Handler) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("chatId")

	chat, err := h.store.GetChat(r.Context(), chatID)
	if err != nil {
		log.Printf("ERROR GetChat(%q): %v", chatID, err)
		http.Error(w, "failed to load chat", http.StatusInternalServerError)
		return
	}
	if chat == nil {
		chat = models.NewChat(chatID)
	}

	user := h.userFromSession(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := template.ChatPage(*chat, user).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render", http.StatusInternalServerError)
	}
}
