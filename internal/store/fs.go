package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
