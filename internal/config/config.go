package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv              string
	GeneratedBucket     string
	DurableLambdaName   string
	BedrockModelID      string
	CognitoUserPoolID   string
	CognitoClientID     string
	CognitoClientSecret string
	CognitoDomain       string
	CognitoRegion       string
	CallbackURL         string
	MaxUploadSizeBytes  int64
}

func Load() *Config {
	return &Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		GeneratedBucket:     getEnv("GENERATED_BUCKET", "app-scaffold-generated"),
		DurableLambdaName:   getEnv("DURABLE_LAMBDA_NAME", "app-scaffold-durable"),
		BedrockModelID:      getEnv("BEDROCK_MODEL_ID", "us.anthropic.claude-sonnet-4-5-20250929-v1:0"),
		CognitoUserPoolID:   getEnv("COGNITO_USER_POOL_ID", ""),
		CognitoClientID:     getEnv("COGNITO_CLIENT_ID", ""),
		CognitoClientSecret: getEnv("COGNITO_CLIENT_SECRET", ""),
		CognitoDomain:       getEnv("COGNITO_DOMAIN", ""),
		CognitoRegion:       getEnv("COGNITO_REGION", ""),
		CallbackURL:         getEnv("CALLBACK_URL", ""),
		MaxUploadSizeBytes:  getEnvInt64("MAX_UPLOAD_SIZE_MB", 5) * 1024 * 1024,
	}
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}
