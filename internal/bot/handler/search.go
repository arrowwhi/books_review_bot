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

type SearchHandler struct {
	books  service.BookService
	logger *zap.Logger
}

func NewSearchHandler(books service.BookService, logger *zap.Logger) *SearchHandler {
	return &SearchHandler{books: books, logger: logger}
}

func (h *SearchHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	args := update.Message.CommandArguments()

	if args == "" {
		return sendMessage(bot, chatID, "Введи запрос: /search название или автор")
	}
	sess.SearchQuery = args
	return h.doSearch(ctx, bot, chatID, userID, args, 1, sess)
}

func (h *SearchHandler) doSearch(ctx context.Context, bot *tgbotapi.BotAPI, chatID, userID int64, query string, page int, sess *session.Session) error {
	result, err := h.books.Search(ctx, userID, query, page)
	if err != nil {
		return fmt.Errorf("handler.Search.doSearch: %w", err)
	}

	if result.Total == 0 {
		return sendMessage(bot, chatID,
			fmt.Sprintf("По запросу *%s* ничего не найдено\\.", escapeMarkdown(query)))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 Результаты по *%s*: \\(%d результатов\\)\n\n",
		escapeMarkdown(query), result.Total))

	for i, b := range result.Books {
		num := (result.Page-1)*service.PageSize + i + 1
		line := fmt.Sprintf("%d\\. *%s* — %s", num, escapeMarkdown(b.Title), escapeMarkdown(b.Author))
		if b.Rating != nil {
			line += fmt.Sprintf(" ⭐%d", *b.Rating)
		}
		sb.WriteString(line + "\n")
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, b := range result.Books {
		runes := []rune(b.Title)
		btnText := "📖 "
		if len(runes) > 15 {
			btnText += string(runes[:15]) + "..."
		} else {
			btnText += string(runes)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("s:v:%d", b.ID)),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if result.Page > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("← Назад", fmt.Sprintf("s:p:%d", result.Page-1)))
	}
	if result.Page < result.TotalPages {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперёд →", fmt.Sprintf("s:p:%d", result.Page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return sendMessageWithKeyboard(bot, chatID, sb.String(), kb)
}

func (h *SearchHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	q := update.CallbackQuery
	answerCallback(bot, q.ID)
	chatID := q.Message.Chat.ID
	userID := q.From.ID
	data := q.Data

	switch {
	case strings.HasPrefix(data, "s:p:"):
		pageStr := strings.TrimPrefix(data, "s:p:")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
		return h.doSearch(ctx, bot, chatID, userID, sess.SearchQuery, page, sess)

	case strings.HasPrefix(data, "s:v:"):
		idStr := strings.TrimPrefix(data, "s:v:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		book, err := h.books.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return sendMessage(bot, chatID, "Книга не найдена\\.")
			}
			return fmt.Errorf("handler.Search.HandleCallback: %w", err)
		}
		card := formatBookCard(book)
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("b:e:%d", book.ID)),
				tgbotapi.NewInlineKeyboardButtonData("⬅ К поиску", "s:p:1"),
			),
		)
		return sendMessageWithKeyboard(bot, chatID, card, kb)
	}

	return nil
}
