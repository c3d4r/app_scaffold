package handler

import (
	"net/http"

	"github.com/c3d4r/app_scaffold/internal/store"
)

type ProcessStarter func(chatID, msgID string) error

type Handler struct {
	store      store.ChatStore
	processMsg ProcessStarter
}

func New(store store.ChatStore, processMsg ProcessStarter) *Handler {
	return &Handler{
		store:      store,
		processMsg: processMsg,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.handleHome)
	mux.HandleFunc("GET /favicon.ico", h.handleFavicon)
	mux.HandleFunc("GET /{chatId}", h.handleChat)
	mux.HandleFunc("POST /{chatId}", h.handleSend)
	mux.HandleFunc("GET /{chatId}/msgs/{msgId}", h.handlePoll)
	return mux
}
