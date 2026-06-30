package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/service"
)

type RecommendHandler struct {
	recommend service.RecommendService
	books     service.BookService
	logger    *zap.Logger
}

func NewRecommendHandler(recommend service.RecommendService, books service.BookService, logger *zap.Logger) *RecommendHandler {
	return &RecommendHandler{recommend: recommend, books: books, logger: logger}
}

func (h *RecommendHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	if err := sendMessage(bot, chatID, "🤔 Подбираю рекомендации на основе твоих вкусов\\.\\.\\."); err != nil {
		h.logger.Warn("recommend: failed to send thinking message", zap.Error(err))
	}

	books, err := h.recommend.Recommend(ctx, userID)
	if err != nil {
		h.logger.Error("recommend", zap.Int64("user_id", userID), zap.Error(err))
		return sendMessage(bot, chatID, "Не удалось получить рекомендации\\. Попробуй позже\\.")
	}

	sess.RecommendedBooks = books

	text := formatRecommendations(books)
	buttons := make([]tgbotapi.InlineKeyboardButton, len(books))
	for i := range books {
		n := strconv.Itoa(i + 1)
		buttons[i] = tgbotapi.NewInlineKeyboardButtonData(n, fmt.Sprintf("rec:add:%d", i))
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
	_, err = bot.Send(msg)
	return err
}

func (h *RecommendHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	cb := update.CallbackQuery
	bot.Request(tgbotapi.NewCallback(cb.ID, "")) //nolint:errcheck

	parts := strings.SplitN(cb.Data, ":", 3)
	if len(parts) != 3 || parts[0] != "rec" || parts[1] != "add" {
		return nil
	}
	idx, err := strconv.Atoi(parts[2])
	if err != nil || idx < 0 || idx >= len(sess.RecommendedBooks) {
		return sendMessage(bot, cb.Message.Chat.ID, "Не удалось определить книгу\\. Попробуй ещё раз /recommend\\.")
	}

	rec := sess.RecommendedBooks[idx]
	userID := cb.From.ID

	_, err = h.books.Add(ctx, userID, service.AddBookInput{
		Title:  rec.Title,
		Author: rec.Author,
		Status: domain.StatusWishlist,
	})
	if err != nil {
		h.logger.Error("recommend: add to wishlist", zap.Int64("user_id", userID), zap.Error(err))
		return sendMessage(bot, cb.Message.Chat.ID, "Не удалось добавить в вишлист\\. Попробуй позже\\.")
	}

	text := fmt.Sprintf("✅ «%s» добавлена в вишлист\\.", escapeMarkdown(rec.Title))
	return sendMessage(bot, cb.Message.Chat.ID, text)
}

func formatRecommendations(books []domain.RecommendedBook) string {
	var sb strings.Builder
	sb.WriteString("📚 *Рекомендации для тебя:*\n")

	for i, b := range books {
		sb.WriteString("\n")
		// author as subheader
		sb.WriteString(fmt.Sprintf("*%s*\n", escapeMarkdown(b.Author)))
		// numbered book entry
		year := ""
		if b.Year > 0 {
			year = fmt.Sprintf(" \\(%d\\)", b.Year)
		}
		sb.WriteString(fmt.Sprintf("%d\\. «%s»%s\n", i+1, escapeMarkdown(b.Title), year))
		sb.WriteString(escapeMarkdown(b.Reason) + "\n")
	}

	sb.WriteString("\nНажми на номер, чтобы добавить книгу в вишлист\\.")
	return sb.String()
}
