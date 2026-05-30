package handler

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/c3d4r/app_scaffold/internal/models"
	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handleSend(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("chatId")

	log.Printf("handleSend Content-Type=%q ContentLength=%d",
		r.Header.Get("Content-Type"), r.ContentLength)

	var content string
	if r.Body != nil {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("handleSend read body error: %v", err)
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		values, err := url.ParseQuery(string(raw))
		if err != nil {
			log.Printf("handleSend parse body error: %v", err)
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		content = values.Get("content")
	}
	log.Printf("handleSend content=%q", content)
	if content == "" {
		http.Error(w, "content required", http.StatusBadRequest)
		return
	}

	chat, err := h.store.GetChat(r.Context(), chatID)
	if err != nil {
		log.Printf("ERROR handleSend GetChat(%q): %v", chatID, err)
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
		log.Printf("ERROR handleSend SaveChat(%q): %v", chatID, err)
		http.Error(w, "failed to save chat", http.StatusInternalServerError)
		return
	}

	if user := h.userFromSession(r); user != nil {
		h.updateChatIndex(r.Context(), user.UserID, chat)
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

	// Render from the stored message content via the same template used on a
	// full page load, so live polling and reload produce identical markup
	// (and identical markdown rendering). The persisted S3 fragment is no
	// longer read; it is now vestigial.
	template.Message(*target, chatID).Render(r.Context(), w)
}
