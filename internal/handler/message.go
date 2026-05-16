package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/c3d4r/app_scaffold/internal/models"
	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handleSend(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("chatId")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "content required", http.StatusBadRequest)
		return
	}

	chat, err := h.store.GetChat(r.Context(), chatID)
	if err != nil {
		http.Error(w, "failed to load chat", http.StatusInternalServerError)
		return
	}
	if chat == nil {
		chat = models.NewChat(chatID)
	}

	userMsg := models.Message{
		ID:      uuid.New().String(),
		Role:    "user",
		Content: content,
		Status:  "complete",
	}
	assistantMsg := models.Message{
		ID:     uuid.New().String(),
		Role:   "assistant",
		Status: "processing",
	}

	chat.AddMessage(userMsg)
	chat.AddMessage(assistantMsg)

	if err := h.store.SaveChat(r.Context(), chat); err != nil {
		http.Error(w, "failed to save chat", http.StatusInternalServerError)
		return
	}

	if err := h.processMsg(chatID, assistantMsg.ID); err != nil {
		http.Error(w, "failed to start processing", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.Message(userMsg, chatID).Render(r.Context(), w)
	template.Loader(chatID, assistantMsg.ID).Render(r.Context(), w)
}

func (h *Handler) handlePoll(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("chatId")
	msgID := r.PathValue("msgId")

	chat, err := h.store.GetChat(r.Context(), chatID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if chat == nil {
		http.Error(w, "chat not found", http.StatusNotFound)
		return
	}

	var target *models.Message
	for i := range chat.Messages {
		if chat.Messages[i].ID == msgID {
			target = &chat.Messages[i]
			break
		}
	}
	if target == nil {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if target.Status == "processing" {
		template.Loader(chatID, msgID).Render(r.Context(), w)
		return
	}

	fragment, err := h.store.GetFragment(r.Context(), chatID, msgID)
	if err != nil {
		http.Error(w, "fragment not found", http.StatusNotFound)
		return
	}

	w.Write(fragment)
}
