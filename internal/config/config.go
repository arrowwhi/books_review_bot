package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken       string
	BotMode        string
	WebhookURL     string
	WebhookPort    string
	WebhookPath    string
	DatabaseURL    string
	AnthropicKey   string
	GoogleBooksKey string
	LogLevel       string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		BotToken:       os.Getenv("BOT_TOKEN"),
		BotMode:        getEnvOrDefault("BOT_MODE", "webhook"),
		WebhookURL:     os.Getenv("WEBHOOK_URL"),
		WebhookPort:    getEnvOrDefault("WEBHOOK_PORT", "8443"),
		WebhookPath:    getEnvOrDefault("WEBHOOK_PATH", "/tg"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		AnthropicKey:   os.Getenv("ANTHROPIC_API_KEY"),
		GoogleBooksKey: os.Getenv("GOOGLE_BOOKS_API_KEY"),
		LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
	}

	if cfg.BotToken == "" {
		return nil, errors.New("BOT_TOKEN is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
