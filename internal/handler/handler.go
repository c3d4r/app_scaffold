package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

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
	maxUploadSize int64
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

func (h *Handler) WithMaxUpload(size int64) *Handler {
	h.maxUploadSize = size
	return h
}

func (h *Handler) Routes() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	e.GET("/about", h.handlePublic)
	e.POST("/auth/login", h.handleLoginPage)
	e.GET("/auth/login", h.handleLoginPage)
	e.GET("/auth/signup", h.handleSignupPage)
	e.POST("/auth/signup", h.handleSignupPage)
	e.GET("/auth/confirm", h.handleConfirmPage)
	e.POST("/auth/confirm", h.handleConfirmPage)
	e.GET("/auth/callback", h.handleCallback)
	e.POST("/auth/logout", h.handleLogout)

	if h.cognito == nil {
		e.GET("/auth/dev-sign-in", h.handleSignInDev)
		e.Static("/uploads", "data/uploads")
	}

	g := e.Group("")
	g.Use(h.authMiddleware)
	g.GET("/", h.handleHome)
	g.POST("/chats", h.handleCreateChat)
	g.GET("/favicon.ico", h.handleFavicon)
	g.GET("/:chatId", h.handleChat)
	g.POST("/:chatId", h.handleSend)
	g.GET("/:chatId/msgs/:msgId", h.handlePoll)

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		msg := "Internal Server Error"
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			msg = he.Message.(string)
		}
		c.String(code, msg)
	}

	return e
}
