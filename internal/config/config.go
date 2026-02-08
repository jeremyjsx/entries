package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	S3Bucket    string
	AWSRegion   string
	S3Endpoint  string
	RabbitMQURL string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Default().Warn("loading .env failed", "error", err)
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		S3Bucket:    getEnv("S3_BUCKET", ""),
		AWSRegion:   getEnv("AWS_REGION", "us-east-1"),
		S3Endpoint:  getEnv("S3_ENDPOINT", ""),
		RabbitMQURL: getEnv("RABBITMQ_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
