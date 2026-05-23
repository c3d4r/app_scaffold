package handler

import (
	"net/http"

	"github.com/c3d4r/app_scaffold/internal/auth"
	"github.com/c3d4r/app_scaffold/internal/store"
)

type ProcessStarter func(chatID, msgID string) error

type Handler struct {
	store         store.ChatStore
	processMsg    ProcessStarter
	sessionStore  auth.SessionStore
	cognito       *auth.CognitoConfig
	cognitoClient *CognitoClient
	callbackURL   string
}

func New(store store.ChatStore, processMsg ProcessStarter) *Handler {
	return &Handler{
		store:      store,
		processMsg: processMsg,
	}
}

func (h *Handler) WithAuth(sessionStore auth.SessionStore, cognito *auth.CognitoConfig, cognitoClient *CognitoClient, callbackURL string) *Handler {
	h.sessionStore = sessionStore
	h.cognito = cognito
	h.cognitoClient = cognitoClient
	h.callbackURL = callbackURL
	return h
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /about", h.handlePublic)
	mux.HandleFunc("/auth/login", h.handleLoginPage)
	mux.HandleFunc("GET /auth/signup", h.handleSignupPage)
	mux.HandleFunc("POST /auth/signup", h.handleSignupPage)
	mux.HandleFunc("GET /auth/confirm", h.handleConfirmPage)
	mux.HandleFunc("POST /auth/confirm", h.handleConfirmPage)
	mux.HandleFunc("GET /auth/callback", h.handleCallback)
	mux.HandleFunc("POST /auth/logout", h.handleLogout)

	if h.cognito == nil {
		mux.HandleFunc("GET /auth/dev-sign-in", h.handleSignInDev)
	}

	protected := http.NewServeMux()
	protected.HandleFunc("GET /{$}", h.handleHome)
	protected.HandleFunc("POST /chats", h.handleCreateChat)
	protected.HandleFunc("GET /favicon.ico", h.handleFavicon)
	protected.HandleFunc("GET /{chatId}", h.handleChat)
	protected.HandleFunc("POST /{chatId}", h.handleSend)
	protected.HandleFunc("GET /{chatId}/msgs/{msgId}", h.handlePoll)

	mux.Handle("/", h.authMiddleware(protected))
	return mux
}
