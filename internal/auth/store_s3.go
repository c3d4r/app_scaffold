package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type S3SessionStore struct {
	client *s3.Client
	bucket string
}

func NewS3SessionStore(client *s3.Client, bucket string) *S3SessionStore {
	return &S3SessionStore{client: client, bucket: bucket}
}

func (s *S3SessionStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	key := "sessions/" + sessionID + ".json"
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isKeyNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get session from s3: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read session body: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &session, nil
}

func (s *S3SessionStore) PutSession(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	key := "sessions/" + session.ID + ".json"
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put session to s3: %w", err)
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

func (s *S3SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	key := "sessions/" + sessionID + ".json"
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
