package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type SessionStore interface {
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	PutSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID string) error
}

func NewSession(userID, username, email string) *Session {
	return &Session{
		ID:        newSessionID(),
		UserID:    userID,
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func newSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
