package config

import "os"

type Config struct {
	AppEnv        string
	GeneratedBucket string
	DurableLambdaName string
	BedrockModelID string
}

func Load() *Config {
	return &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		GeneratedBucket:   getEnv("GENERATED_BUCKET", "app-scaffold-generated"),
		DurableLambdaName: getEnv("DURABLE_LAMBDA_NAME", "app-scaffold-durable"),
		BedrockModelID:    getEnv("BEDROCK_MODEL_ID", "us.anthropic.claude-sonnet-4-6"),
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
