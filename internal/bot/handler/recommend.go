package handler

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
	"github.com/arrowwhi/books_review_bot/internal/service"
)

type RecommendHandler struct {
	recommend service.RecommendService
	logger    *zap.Logger
}

func NewRecommendHandler(recommend service.RecommendService, logger *zap.Logger) *RecommendHandler {
	return &RecommendHandler{recommend: recommend, logger: logger}
}

func (h *RecommendHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	if err := sendMessage(bot, chatID, "🤔 Подбираю рекомендации на основе твоих вкусов\\.\\.\\."); err != nil {
		h.logger.Warn("recommend: failed to send thinking message", zap.Error(err))
	}

	result, err := h.recommend.Recommend(ctx, userID)
	if err != nil {
		h.logger.Error("recommend", zap.Int64("user_id", userID), zap.Error(err))
		return sendMessage(bot, chatID, "Не удалось получить рекомендации\\. Попробуй позже\\.")
	}

	return sendMessage(bot, chatID, escapeMarkdown(result))
}
