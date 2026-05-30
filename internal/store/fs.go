package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/c3d4r/app_scaffold/internal/models"
)

type FSStore struct {
	root string
}

func NewFSStore(root string) *FSStore {
	return &FSStore{root: root}
}

func (s *FSStore) GetChat(_ context.Context, chatID string) (*models.Chat, error) {
	path := filepath.Join(s.root, "chats", chatID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read chat: %w", err)
	}
	var chat models.Chat
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, fmt.Errorf("parse chat: %w", err)
	}
	return &chat, nil
}

func (s *FSStore) SaveChat(_ context.Context, chat *models.Chat) error {
	dir := filepath.Join(s.root, "chats")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.MarshalIndent(chat, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, chat.ID+".json"), data, 0644)
}

func (s *FSStore) GetFragment(_ context.Context, chatID, msgID string) ([]byte, error) {
	path := filepath.Join(s.root, "messages", chatID, msgID+".html")
	return os.ReadFile(path)
}

func (s *FSStore) PutFragment(_ context.Context, chatID, msgID string, html []byte) error {
	dir := filepath.Join(s.root, "messages", chatID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, msgID+".html"), html, 0644)
}

func (s *FSStore) ListChats(_ context.Context, userID string) ([]models.ChatSummary, error) {
	path := filepath.Join(s.root, "users", userID, "chats.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read chat index: %w", err)
	}
	var chats []models.ChatSummary
	if err := json.Unmarshal(data, &chats); err != nil {
		return nil, fmt.Errorf("parse chat index: %w", err)
	}
	return chats, nil
}

func (s *FSStore) PutChatIndex(_ context.Context, userID string, chats []models.ChatSummary) error {
	dir := filepath.Join(s.root, "users", userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.Marshal(chats)
	if err != nil {
		return fmt.Errorf("marshal chat index: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "chats.json"), data, 0644)
}

func (s *FSStore) PutFile(_ context.Context, key string, data []byte, _ string) error {
	fullPath := filepath.Join(s.root, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(fullPath, data, 0644)
}

func (s *FSStore) GetFile(_ context.Context, key string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.root, key))
}

func (s *FSStore) GetPreSignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "/uploads/" + key, nil
}
