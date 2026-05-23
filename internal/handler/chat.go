package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/c3d4r/app_scaffold/internal/models"
	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user != nil {
		chats, err := h.store.ListChats(r.Context(), user.UserID)
		if err == nil && len(chats) > 0 {
			http.Redirect(w, r, "/"+chats[0].ID, http.StatusFound)
			return
		}
	}
	h.handleCreateChat(w, r)
}

func (h *Handler) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	chatID := uuid.New().String()
	chat := models.NewChat(chatID)

	if err := h.store.SaveChat(r.Context(), chat); err != nil {
		log.Printf("ERROR create chat: %v", err)
		http.Error(w, "failed to create chat", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	chats, _ := h.store.ListChats(r.Context(), user.UserID)
	chats = append([]models.ChatSummary{{
		ID:        chatID,
		Title:     "New conversation",
		UpdatedAt: now,
	}}, chats...)
	if err := h.store.PutChatIndex(r.Context(), user.UserID, chats); err != nil {
		log.Printf("ERROR put chat index: %v", err)
	}

	http.Redirect(w, r, "/"+chatID, http.StatusFound)
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

	var chatList []models.ChatSummary
	if user != nil {
		chatList, _ = h.store.ListChats(r.Context(), user.UserID)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := template.ChatPage(*chat, chatList, user).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render", http.StatusInternalServerError)
	}
}

func (h *Handler) updateChatIndex(ctx context.Context, userID string, chat *models.Chat) {
	chats, err := h.store.ListChats(ctx, userID)
	if err != nil {
		return
	}

	title := "New conversation"
	for _, m := range chat.Messages {
		if m.Role == "user" && m.Content != "" {
			title = truncate(m.Content, 40)
			break
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	found := false
	for i := range chats {
		if chats[i].ID == chat.ID {
			chats[i].Title = title
			chats[i].UpdatedAt = now
			found = true
			break
		}
	}
	if !found {
		chats = append([]models.ChatSummary{{
			ID:        chat.ID,
			Title:     title,
			UpdatedAt: now,
		}}, chats...)
	}

	h.store.PutChatIndex(ctx, userID, chats)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
