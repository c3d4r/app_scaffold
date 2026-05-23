package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type FSSessionStore struct {
	root string
}

func NewFSSessionStore(root string) *FSSessionStore {
	return &FSSessionStore{root: root}
}

func (s *FSSessionStore) GetSession(_ context.Context, sessionID string) (*Session, error) {
	path := filepath.Join(s.root, "sessions", sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session: %w", err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &session, nil
}

func (s *FSSessionStore) PutSession(_ context.Context, session *Session) error {
	dir := filepath.Join(s.root, "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, session.ID+".json"), data, 0644)
}

func (s *FSSessionStore) DeleteSession(_ context.Context, sessionID string) error {
	path := filepath.Join(s.root, "sessions", sessionID+".json")
	return os.Remove(path)
}
