package handler

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
)

const helpText = `👋 *Книжный дневник*

Команды:
/add — добавить прочитанную книгу
/want — добавить в «хочу прочитать»
/library — все прочитанные книги
/wishlist — список желаемого
/search — поиск по своим книгам
/stats — статистика чтения
/recommend — рекомендации книг
/remind — настроить напоминание
/help — эта справка`

type HelpHandler struct{}

func NewHelpHandler() *HelpHandler {
	return &HelpHandler{}
}

func (h *HelpHandler) HandleStart(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	return sendMessage(bot, update.Message.Chat.ID, helpText)
}

func (h *HelpHandler) HandleHelp(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	return sendMessage(bot, update.Message.Chat.ID, helpText)
}
