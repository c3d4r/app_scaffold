package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/c3d4r/app_scaffold/internal/models"
	"github.com/c3d4r/app_scaffold/internal/template"
)

var allowedMIMETypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"text/plain":      true,
	"text/csv":        true,
	"text/html":       true,
}

func (h *Handler) handleSend(c echo.Context) error {
	chatID := c.Param("chatId")
	maxSize := h.maxUploadSize
	if maxSize == 0 {
		maxSize = 5 * 1024 * 1024
	}

	ct := c.Request().Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := c.Request().ParseMultipartForm(maxSize); err != nil {
			log.Printf("handleSend multipart parse error: %v", err)
			return c.String(http.StatusBadRequest, "file too large or invalid form data")
		}
	}

	content := strings.TrimSpace(c.FormValue("content"))

	userMsgID := uuid.New().String()

	var attachments []models.Attachment

	if c.Request().MultipartForm != nil {
		files := c.Request().MultipartForm.File["files"]
		for _, fh := range files {
			mimeType := fh.Header.Get("Content-Type")
			if !allowedMIMETypes[mimeType] {
				ext := strings.ToLower(filepath.Ext(fh.Filename))
				if ext == ".md" || ext == ".markdown" {
					mimeType = "text/markdown"
				} else if mimeType == "" {
					mimeType = mime.TypeByExtension(ext)
				}
				if !allowedMIMETypes[mimeType] {
					return c.String(http.StatusBadRequest, fmt.Sprintf("unsupported file type: %s", fh.Filename))
				}
			}

			if fh.Size > maxSize {
				return c.String(http.StatusBadRequest, fmt.Sprintf("file too large: %s (%d bytes)", fh.Filename, fh.Size))
			}

			file, err := fh.Open()
			if err != nil {
				return c.String(http.StatusBadRequest, fmt.Sprintf("failed to read file: %s", fh.Filename))
			}
			data, err := io.ReadAll(file)
			file.Close()
			if err != nil {
				return c.String(http.StatusBadRequest, fmt.Sprintf("failed to read file: %s", fh.Filename))
			}

			attKey := fmt.Sprintf("uploads/%s/%s/%s", chatID, userMsgID, fh.Filename)
			if err := h.store.PutFile(c.Request().Context(), attKey, data, mimeType); err != nil {
				log.Printf("ERROR upload file %s: %v", fh.Filename, err)
				return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to upload: %s", fh.Filename))
			}

			attachments = append(attachments, models.Attachment{
				Name: fh.Filename,
				Type: mimeType,
				Size: fh.Size,
				Key:  attKey,
			})
		}
	}

	if content == "" && len(attachments) == 0 {
		return c.String(http.StatusBadRequest, "message or file required")
	}

	if content == "" {
		content = "Please analyze the attached file(s)."
	}

	chat, err := h.store.GetChat(c.Request().Context(), chatID)
	if err != nil {
		log.Printf("ERROR handleSend GetChat(%q): %v", chatID, err)
		return c.String(http.StatusInternalServerError, "failed to load chat")
	}
	if chat == nil {
		chat = models.NewChat(chatID)
	}

	userMsg := models.Message{
		ID:          userMsgID,
		Role:        "user",
		Content:     content,
		Status:      "complete",
		Attachments: attachments,
	}
	assistantMsg := models.Message{
		ID:     uuid.New().String(),
		Role:   "assistant",
		Status: "processing",
	}

	chat.AddMessage(userMsg)
	chat.AddMessage(assistantMsg)

	if err := h.store.SaveChat(c.Request().Context(), chat); err != nil {
		log.Printf("ERROR handleSend SaveChat(%q): %v", chatID, err)
		return c.String(http.StatusInternalServerError, "failed to save chat")
	}

	if user := getUser(c); user != nil {
		h.updateChatIndex(c.Request().Context(), user.UserID, chat)
	}

	if err := h.processMsg(chatID, assistantMsg.ID); err != nil {
		return c.String(http.StatusInternalServerError, "failed to start processing")
	}

	h.injectAttachmentURLs(c.Request().Context(), &userMsg)

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := template.Message(userMsg, chatID).Render(c.Request().Context(), c.Response().Writer); err != nil {
		return err
	}
	return template.Loader(chatID, assistantMsg.ID).Render(c.Request().Context(), c.Response().Writer)
}

func (h *Handler) handlePoll(c echo.Context) error {
	chatID := c.Param("chatId")
	msgID := c.Param("msgId")

	chat, err := h.store.GetChat(c.Request().Context(), chatID)
	if err != nil {
		return c.String(http.StatusNotFound, "not found")
	}
	if chat == nil {
		return c.String(http.StatusNotFound, "chat not found")
	}

	var target *models.Message
	for i := range chat.Messages {
		if chat.Messages[i].ID == msgID {
			target = &chat.Messages[i]
			break
		}
	}
	if target == nil {
		return c.String(http.StatusNotFound, "message not found")
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")

	if target.Status == "processing" {
		return template.Loader(chatID, msgID).Render(c.Request().Context(), c.Response().Writer)
	}

	h.injectAttachmentURLs(c.Request().Context(), target)
	return template.Message(*target, chatID).Render(c.Request().Context(), c.Response().Writer)
}

func (h *Handler) injectAttachmentURLs(ctx context.Context, msg *models.Message) {
	for i := range msg.Attachments {
		url, err := h.store.GetPreSignedURL(ctx, msg.Attachments[i].Key, 1*time.Hour)
		if err != nil {
			log.Printf("WARN presigned url for %s: %v", msg.Attachments[i].Key, err)
			continue
		}
		msg.Attachments[i].PreSignedURL = url
	}
}
