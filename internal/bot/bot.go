package bot

import (
	"context"
	"fmt"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/config"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	router *Router
	logger *zap.Logger
	cfg    *config.Config
}

func New(cfg *config.Config, router *Router, logger *zap.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("tgbotapi.New: %w", err)
	}
	api.Debug = cfg.LogLevel == "debug"
	return &Bot{api: api, router: router, logger: logger, cfg: cfg}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	var updates tgbotapi.UpdatesChannel

	if b.cfg.BotMode == "webhook" {
		wh, err := tgbotapi.NewWebhook(b.cfg.WebhookURL + b.cfg.WebhookPath)
		if err != nil {
			return fmt.Errorf("new webhook: %w", err)
		}
		if _, err = b.api.Request(wh); err != nil {
			return fmt.Errorf("set webhook: %w", err)
		}
		updates = b.api.ListenForWebhook(b.cfg.WebhookPath)
		go func() {
			b.logger.Info("webhook server listening", zap.String("port", b.cfg.WebhookPort))
			if err := http.ListenAndServe(":"+b.cfg.WebhookPort, nil); err != nil {
				b.logger.Error("webhook server", zap.Error(err))
			}
		}()
	} else {
		if _, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true}); err != nil {
			return fmt.Errorf("delete webhook: %w", err)
		}
		cfg := tgbotapi.NewUpdate(0)
		cfg.Timeout = 60
		updates = b.api.GetUpdatesChan(cfg)
	}

	b.logger.Info("bot started",
		zap.String("username", b.api.Self.UserName),
		zap.String("mode", b.cfg.BotMode),
	)

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("bot stopped")
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			go b.router.Handle(ctx, b.api, update)
		}
	}
}
