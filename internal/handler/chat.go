package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/c3d4r/app_scaffold/internal/auth"
	"github.com/c3d4r/app_scaffold/internal/models"
	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handleHome(c echo.Context) error {
	user := getUser(c)
	if user != nil {
		chats, err := h.store.ListChats(c.Request().Context(), user.UserID)
		if err == nil && len(chats) > 0 {
			return c.Redirect(http.StatusFound, "/"+chats[0].ID)
		}
	}
	return h.handleCreateChat(c)
}

func (h *Handler) handleFavicon(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleCreateChat(c echo.Context) error {
	user := getUser(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "not authenticated")
	}

	chatID := uuid.New().String()
	chat := models.NewChat(chatID)

	if err := h.store.SaveChat(c.Request().Context(), chat); err != nil {
		log.Printf("ERROR create chat: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create chat")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	chats, _ := h.store.ListChats(c.Request().Context(), user.UserID)
	chats = append([]models.ChatSummary{{
		ID:        chatID,
		Title:     "New conversation",
		UpdatedAt: now,
	}}, chats...)
	if err := h.store.PutChatIndex(c.Request().Context(), user.UserID, chats); err != nil {
		log.Printf("ERROR put chat index: %v", err)
	}

	return c.Redirect(http.StatusFound, "/"+chatID)
}

func (h *Handler) handleChat(c echo.Context) error {
	chatID := c.Param("chatId")

	chat, err := h.store.GetChat(c.Request().Context(), chatID)
	if err != nil {
		log.Printf("ERROR GetChat(%q): %v", chatID, err)
		return c.String(http.StatusInternalServerError, "failed to load chat")
	}
	if chat == nil {
		chat = models.NewChat(chatID)
	}

	user := getUser(c)

	var chatList []models.ChatSummary
	if user != nil {
		chatList, _ = h.store.ListChats(c.Request().Context(), user.UserID)
	}

	h.injectAllAttachmentURLs(c.Request().Context(), chat)

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return template.ChatPage(*chat, chatList, user).Render(c.Request().Context(), c.Response().Writer)
}

func (h *Handler) injectAllAttachmentURLs(ctx context.Context, chat *models.Chat) {
	for i := range chat.Messages {
		for j := range chat.Messages[i].Attachments {
			url, err := h.store.GetPreSignedURL(ctx, chat.Messages[i].Attachments[j].Key, 1*time.Hour)
			if err != nil {
				continue
			}
			chat.Messages[i].Attachments[j].PreSignedURL = url
		}
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

func getUser(c echo.Context) *auth.Session {
	if s, ok := c.Get("session").(*auth.Session); ok {
		return s
	}
	return nil
}
