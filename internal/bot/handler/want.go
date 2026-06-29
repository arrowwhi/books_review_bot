package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/service"
)

type WantHandler struct {
	books  service.BookService
	logger *zap.Logger
}

func NewWantHandler(books service.BookService, logger *zap.Logger) *WantHandler {
	return &WantHandler{books: books, logger: logger}
}

func (h *WantHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message, sess *session.Session) error {
	sess.Draft = session.Draft{}
	sess.State = session.StateWantTitle
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "w:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, msg.Chat.ID, "Введи название книги для вишлиста:", kb)
}

func (h *WantHandler) HandleMessage(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message, sess *session.Session) error {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	text := msg.Text

	switch sess.State {
	case session.StateWantTitle:
		sess.Draft.Title = text
		sess.State = session.StateWantAuthor
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "w:skip:author"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "w:cancel"),
			),
		)
		return sendMessageWithKeyboard(bot, chatID, "Введи автора \\(необязательно\\):", kb)

	case session.StateWantAuthor:
		sess.Draft.Author = text
		return h.saveToWishlist(ctx, bot, chatID, userID, sess)
	}

	return nil
}

func (h *WantHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, q *tgbotapi.CallbackQuery, sess *session.Session) error {
	answerCallback(bot, q.ID)
	chatID := q.Message.Chat.ID
	userID := q.From.ID
	data := q.Data

	switch {
	case data == "w:cancel":
		sess.State = session.StateIdle
		sess.Draft = session.Draft{}
		return sendMessage(bot, chatID, "❌ Добавление отменено\\.")

	case data == "w:skip:author":
		return h.saveToWishlist(ctx, bot, chatID, userID, sess)

	case strings.HasPrefix(data, "w:r:"):
		idStr := strings.TrimPrefix(data, "w:r:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		book, err := h.books.MoveToRead(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return sendMessage(bot, chatID, "Книга не найдена")
			}
			return fmt.Errorf("handler.HandleCallback: %w", err)
		}
		card := formatBookCard(book)
		return sendMessage(bot, chatID, "✅ Перенесено в прочитанное\n\n"+card)
	}

	return nil
}

func (h *WantHandler) saveToWishlist(ctx context.Context, bot *tgbotapi.BotAPI, chatID, userID int64, sess *session.Session) error {
	input := service.AddBookInput{
		Title:  sess.Draft.Title,
		Author: sess.Draft.Author,
		Status: domain.StatusWishlist,
	}
	book, err := h.books.Add(ctx, userID, input)
	if err != nil {
		return fmt.Errorf("handler.saveToWishlist: %w", err)
	}

	sess.State = session.StateIdle
	sess.Draft = session.Draft{}

	text := fmt.Sprintf("📌 Добавлено в вишлист\\!\n\n📚 *%s*", escapeMarkdown(book.Title))
	if book.Author != "" {
		text += fmt.Sprintf("\n👤 %s", escapeMarkdown(book.Author))
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Уже прочитал", fmt.Sprintf("w:r:%d", book.ID)),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, text, kb)
}
