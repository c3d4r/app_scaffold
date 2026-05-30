package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"

	appconfig "github.com/c3d4r/app_scaffold/internal/config"
	"github.com/c3d4r/app_scaffold/internal/auth"
	"github.com/c3d4r/app_scaffold/internal/handler"
	"github.com/c3d4r/app_scaffold/internal/store"
)

func createHandler() (http.Handler, error) {
	cfg := appconfig.Load()
	ctx := context.Background()

	var chatStore store.ChatStore
	var starter handler.ProcessStarter
	var sessionStore auth.SessionStore
	var cognito *auth.CognitoConfig
	var cognitoClient *handler.CognitoClient

	if cfg.IsProduction() {
		awsCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws config: %w", err)
		}
		chatStore = store.NewS3Store(s3.NewFromConfig(awsCfg), cfg.GeneratedBucket)
		starter = handler.LambdaProcessStarter(lambda.NewFromConfig(awsCfg), cfg.DurableLambdaName)
		sessionStore = auth.NewS3SessionStore(s3.NewFromConfig(awsCfg), cfg.GeneratedBucket)

		if cfg.CognitoUserPoolID != "" {
			cip := cognitoidentityprovider.NewFromConfig(awsCfg)
			cognitoClient = handler.NewCognitoClient(cip, cfg.CognitoClientID, cfg.CognitoClientSecret)
			cognito = &auth.CognitoConfig{
				UserPoolID:   cfg.CognitoUserPoolID,
				ClientID:     cfg.CognitoClientID,
				ClientSecret: cfg.CognitoClientSecret,
				Domain:       cfg.CognitoDomain,
				Region:       cfg.CognitoRegion,
			}
		}
	} else {
		chatStore = store.NewFSStore("data")
		sessionStore = auth.NewFSSessionStore("data")

		awsCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Printf("no AWS config, using echo mode: %v", err)
			starter = handler.EchoProcessStarter(chatStore)
		} else {
			starter = handler.InlineProcessStarter(
				chatStore,
				bedrockruntime.NewFromConfig(awsCfg),
				cfg.BedrockModelID,
			)
			if cfg.CognitoUserPoolID != "" {
				cip := cognitoidentityprovider.NewFromConfig(awsCfg)
				cognitoClient = handler.NewCognitoClient(cip, cfg.CognitoClientID, cfg.CognitoClientSecret)
				cognito = &auth.CognitoConfig{
					UserPoolID:   cfg.CognitoUserPoolID,
					ClientID:     cfg.CognitoClientID,
					ClientSecret: cfg.CognitoClientSecret,
					Domain:       cfg.CognitoDomain,
					Region:       cfg.CognitoRegion,
				}
			}
		}
	}

	h := handler.New(chatStore, starter).WithAuth(sessionStore, cognito, cognitoClient, cfg.CallbackURL).WithMaxUpload(cfg.MaxUploadSizeBytes).Routes()

	if e, ok := h.(*echo.Echo); ok && !cfg.IsProduction() {
		e.Static("/uploads", "data/uploads")
	}

	return h, nil
}

func startDevServer(h http.Handler) {
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	fmt.Println("dev server listening on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}
