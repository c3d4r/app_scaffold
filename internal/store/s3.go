package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/c3d4r/app_scaffold/internal/models"
)

type S3Store struct {
	client *s3.Client
	bucket string
}

func NewS3Store(client *s3.Client, bucket string) *S3Store {
	return &S3Store{client: client, bucket: bucket}
}

func (s *S3Store) GetChat(ctx context.Context, chatID string) (*models.Chat, error) {
	key := "chats/" + chatID + ".json"
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isKeyNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get chat from s3: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read chat body: %w", err)
	}

	var chat models.Chat
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, fmt.Errorf("parse chat: %w", err)
	}
	return &chat, nil
}

func (s *S3Store) SaveChat(ctx context.Context, chat *models.Chat) error {
	data, err := json.MarshalIndent(chat, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	key := "chats/" + chat.ID + ".json"
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put chat to s3: %w", err)
	}
	return nil
}

func (s *S3Store) GetFragment(ctx context.Context, chatID, msgID string) ([]byte, error) {
	key := "messages/" + chatID + "/" + msgID + ".html"
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get fragment from s3: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (s *S3Store) PutFragment(ctx context.Context, chatID, msgID string, html []byte) error {
	key := "messages/" + chatID + "/" + msgID + ".html"
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(html),
		ContentType: aws.String("text/html"),
	})
	if err != nil {
		return fmt.Errorf("put fragment to s3: %w", err)
	}
	return nil
}

func (s *S3Store) ListChats(ctx context.Context, userID string) ([]models.ChatSummary, error) {
	key := "users/" + userID + "/chats.json"
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isKeyNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get chat index from s3: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read chat index: %w", err)
	}

	var chats []models.ChatSummary
	if err := json.Unmarshal(data, &chats); err != nil {
		return nil, fmt.Errorf("parse chat index: %w", err)
	}
	return chats, nil
}

func (s *S3Store) PutChatIndex(ctx context.Context, userID string, chats []models.ChatSummary) error {
	data, err := json.Marshal(chats)
	if err != nil {
		return fmt.Errorf("marshal chat index: %w", err)
	}
	key := "users/" + userID + "/chats.json"
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put chat index to s3: %w", err)
	}
	return nil
}

func isKeyNotFound(err error) bool {
	var nsk *s3types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var nf *s3types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	var re *smithyhttp.ResponseError
	if errors.As(err, &re) {
		return re.HTTPStatusCode() == http.StatusNotFound
	}
	return false
}
