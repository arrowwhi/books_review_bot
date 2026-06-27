package handler

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/service"
)

type RemindHandler struct {
	remind service.ReminderService
	logger *zap.Logger
}

func NewRemindHandler(remind service.ReminderService, logger *zap.Logger) *RemindHandler {
	return &RemindHandler{remind: remind, logger: logger}
}

func (h *RemindHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	statusLine := "не настроено"
	reminder, err := h.remind.Get(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("handler.Remind.HandleCommand: %w", err)
	}
	if reminder != nil {
		if reminder.Enabled {
			statusLine = fmt.Sprintf("Включено, раз в %d дней", reminder.IntervalDays)
		} else {
			statusLine = "Выключено"
		}
	}

	text := fmt.Sprintf("⏰ *Напоминания*\n\nСтатус: %s\n\nВыбери интервал:", escapeMarkdown(statusLine))
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Раз в 2 недели", "rm:2w"),
			tgbotapi.NewInlineKeyboardButtonData("Раз в месяц", "rm:1m"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔕 Выключить", "rm:off"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, text, kb)
}

func (h *RemindHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	q := update.CallbackQuery
	answerCallback(bot, q.ID)
	chatID := q.Message.Chat.ID
	userID := q.From.ID
	data := q.Data

	switch data {
	case "rm:2w":
		if err := h.remind.Set(ctx, userID, 14); err != nil {
			return fmt.Errorf("handler.Remind.HandleCallback: %w", err)
		}
		return sendMessage(bot, chatID, "✅ Напомню через 2 недели если не добавишь книги\\.")

	case "rm:1m":
		if err := h.remind.Set(ctx, userID, 30); err != nil {
			return fmt.Errorf("handler.Remind.HandleCallback: %w", err)
		}
		return sendMessage(bot, chatID, "✅ Напомню через месяц если не добавишь книги\\.")

	case "rm:off":
		if err := h.remind.Disable(ctx, userID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return sendMessage(bot, chatID, "🔕 Напоминания выключены\\.")
			}
			return fmt.Errorf("handler.Remind.HandleCallback: %w", err)
		}
		return sendMessage(bot, chatID, "🔕 Напоминания выключены\\.")
	}

	return nil
}
