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

type WishlistHandler struct {
	books  service.BookService
	logger *zap.Logger
}

func NewWishlistHandler(books service.BookService, logger *zap.Logger) *WishlistHandler {
	return &WishlistHandler{books: books, logger: logger}
}

func (h *WishlistHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	result, err := h.books.List(ctx, userID, domain.StatusWishlist, 1)
	if err != nil {
		return fmt.Errorf("handler.Wishlist.HandleCommand: %w", err)
	}
	if result.Total == 0 {
		return sendMessage(bot, chatID, "Вишлист пуст\\. Добавь книгу через /want")
	}
	return h.showList(bot, chatID, result)
}

func (h *WishlistHandler) showList(bot *tgbotapi.BotAPI, chatID int64, result *service.BookList) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📌 *Вишлист* \\(страница %d/%d, всего %d книг\\)\n\n",
		result.Page, result.TotalPages, result.Total))

	for i, b := range result.Books {
		num := (result.Page-1)*service.PageSize + i + 1
		sb.WriteString(fmt.Sprintf("%d\\. *%s*", num, escapeMarkdown(b.Title)))
		if b.Author != "" {
			sb.WriteString(fmt.Sprintf(" — %s", escapeMarkdown(b.Author)))
		}
		sb.WriteString("\n")
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, b := range result.Books {
		runes := []rune(b.Title)
		btnText := "📌 "
		if len(runes) > 15 {
			btnText += string(runes[:15]) + "..."
		} else {
			btnText += string(runes)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("w:v:%d", b.ID)),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if result.Page > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("← Назад", fmt.Sprintf("w:p:%d", result.Page-1)))
	}
	if result.Page < result.TotalPages {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперёд →", fmt.Sprintf("w:p:%d", result.Page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return sendMessageWithKeyboard(bot, chatID, sb.String(), kb)
}

func (h *WishlistHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	q := update.CallbackQuery
	answerCallback(bot, q.ID)
	chatID := q.Message.Chat.ID
	userID := q.From.ID
	data := q.Data

	switch {
	case data == "w:cancel":
		sess.State = session.StateIdle
		sess.Draft = session.Draft{}
		return sendMessage(bot, chatID, "❌ Отменено\\.")

	case data == "w:skip:author":
		return h.saveToWishlist(ctx, bot, chatID, userID, sess)

	case strings.HasPrefix(data, "w:r:"):
		idStr := strings.TrimPrefix(data, "w:r:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		if _, err := h.books.MoveToRead(ctx, id); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return sendMessage(bot, chatID, "Книга не найдена\\.")
			}
			return fmt.Errorf("handler.Wishlist.HandleCallback: %w", err)
		}
		return sendMessage(bot, chatID, "✅ Книга перенесена в прочитанное\\! Теперь можешь добавить рецензию через /add")

	case strings.HasPrefix(data, "w:v:"):
		idStr := strings.TrimPrefix(data, "w:v:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		book, err := h.books.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return sendMessage(bot, chatID, "Книга не найдена\\.")
			}
			return fmt.Errorf("handler.Wishlist.HandleCallback: %w", err)
		}
		card := fmt.Sprintf("📌 *%s*", escapeMarkdown(book.Title))
		if book.Author != "" {
			card += fmt.Sprintf("\n👤 %s", escapeMarkdown(book.Author))
		}
		card += fmt.Sprintf("\n📅 Добавлено: %s", escapeMarkdown(book.CreatedAt.Format("2 January 2006")))
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Прочитал!", fmt.Sprintf("w:r:%d", book.ID)),
				tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("b:d:%d", book.ID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅ К списку", "w:p:1"),
			),
		)
		return sendMessageWithKeyboard(bot, chatID, card, kb)

	case strings.HasPrefix(data, "w:p:"):
		pageStr := strings.TrimPrefix(data, "w:p:")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
		result, err := h.books.List(ctx, userID, domain.StatusWishlist, page)
		if err != nil {
			return fmt.Errorf("handler.Wishlist.HandleCallback: %w", err)
		}
		return h.showList(bot, chatID, result)
	}

	return nil
}

func (h *WishlistHandler) saveToWishlist(ctx context.Context, bot *tgbotapi.BotAPI, chatID, userID int64, sess *session.Session) error {
	input := service.AddBookInput{
		Title:  sess.Draft.Title,
		Author: sess.Draft.Author,
		Status: domain.StatusWishlist,
	}
	book, err := h.books.Add(ctx, userID, input)
	if err != nil {
		return fmt.Errorf("handler.Wishlist.saveToWishlist: %w", err)
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
