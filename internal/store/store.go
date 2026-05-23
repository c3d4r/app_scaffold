package store

import (
	"context"

	"github.com/c3d4r/app_scaffold/internal/models"
)

type ChatStore interface {
	GetChat(ctx context.Context, chatID string) (*models.Chat, error)
	SaveChat(ctx context.Context, chat *models.Chat) error
	GetFragment(ctx context.Context, chatID, msgID string) ([]byte, error)
	PutFragment(ctx context.Context, chatID, msgID string, html []byte) error
	ListChats(ctx context.Context, userID string) ([]models.ChatSummary, error)
	PutChatIndex(ctx context.Context, userID string, chats []models.ChatSummary) error
}
