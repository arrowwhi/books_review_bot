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

type LibraryHandler struct {
	books  service.BookService
	genres service.GenreService
	logger *zap.Logger
}

func NewLibraryHandler(books service.BookService, genres service.GenreService, logger *zap.Logger) *LibraryHandler {
	return &LibraryHandler{books: books, genres: genres, logger: logger}
}

func (h *LibraryHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	result, err := h.books.List(ctx, userID, domain.StatusRead, 1)
	if err != nil {
		return fmt.Errorf("handler.Library.HandleCommand: %w", err)
	}
	if result.Total == 0 {
		return sendMessage(bot, chatID, "У тебя пока нет прочитанных книг\\. Добавь первую через /add")
	}
	return h.showList(bot, chatID, result)
}

func (h *LibraryHandler) showList(bot *tgbotapi.BotAPI, chatID int64, result *service.BookList) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 *Библиотека* \\(страница %d/%d, всего %d книг\\)\n\n",
		result.Page, result.TotalPages, result.Total))

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
			tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("l:v:%d", b.ID)),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if result.Page > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("← Назад", fmt.Sprintf("l:p:%d", result.Page-1)))
	}
	if result.Page < result.TotalPages {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперёд →", fmt.Sprintf("l:p:%d", result.Page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return sendMessageWithKeyboard(bot, chatID, sb.String(), kb)
}

func (h *LibraryHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	q := update.CallbackQuery
	answerCallback(bot, q.ID)
	chatID := q.Message.Chat.ID
	userID := q.From.ID
	data := q.Data

	switch {
	case strings.HasPrefix(data, "l:p:"):
		pageStr := strings.TrimPrefix(data, "l:p:")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
		result, err := h.books.List(ctx, userID, domain.StatusRead, page)
		if err != nil {
			return fmt.Errorf("handler.Library.HandleCallback: %w", err)
		}
		return h.showList(bot, chatID, result)

	case strings.HasPrefix(data, "l:v:"):
		idStr := strings.TrimPrefix(data, "l:v:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		return h.showBookCard(ctx, bot, chatID, id)

	case strings.HasPrefix(data, "b:ef:"):
		parts := strings.SplitN(strings.TrimPrefix(data, "b:ef:"), ":", 2)
		if len(parts) < 2 {
			return nil
		}
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil
		}
		field := parts[1]
		sess.Draft.BookID = id
		sess.Draft.EditField = field
		sess.State = session.StateEditField
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", fmt.Sprintf("b:back:%d", id)),
			),
		)
		return sendMessageWithKeyboard(bot, chatID,
			fmt.Sprintf("Введи новое значение для поля %s:", fieldLabel(field)), kb)

	case strings.HasPrefix(data, "b:e:"):
		idStr := strings.TrimPrefix(data, "b:e:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Название", fmt.Sprintf("b:ef:%d:title", id)),
				tgbotapi.NewInlineKeyboardButtonData("👤 Автор", fmt.Sprintf("b:ef:%d:author", id)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Зацепило", fmt.Sprintf("b:ef:%d:liked", id)),
				tgbotapi.NewInlineKeyboardButtonData("😞 Не понравилось", fmt.Sprintf("b:ef:%d:disliked", id)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💡 Инсайт", fmt.Sprintf("b:ef:%d:insight", id)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", fmt.Sprintf("b:back:%d", id)),
			),
		)
		return sendMessageWithKeyboard(bot, chatID, "Что изменить?", kb)

	case strings.HasPrefix(data, "b:dc:"):
		idStr := strings.TrimPrefix(data, "b:dc:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		if err := h.books.Delete(ctx, id); err != nil {
			return fmt.Errorf("handler.Library.HandleCallback: %w", err)
		}
		sess.State = session.StateIdle
		sess.Draft = session.Draft{}
		return sendMessage(bot, chatID, "✅ Книга удалена\\.")

	case strings.HasPrefix(data, "b:d:"):
		idStr := strings.TrimPrefix(data, "b:d:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		book, err := h.books.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return sendMessage(bot, chatID, "Книга не найдена\\.")
			}
			return fmt.Errorf("handler.Library.HandleCallback: %w", err)
		}
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить", fmt.Sprintf("b:dc:%d", id)),
				tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", fmt.Sprintf("b:back:%d", id)),
			),
		)
		return sendMessageWithKeyboard(bot, chatID,
			fmt.Sprintf("🗑 Удалить книгу *%s*? Это действие нельзя отменить\\.", escapeMarkdown(book.Title)), kb)

	case strings.HasPrefix(data, "b:back:"):
		idStr := strings.TrimPrefix(data, "b:back:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil
		}
		sess.State = session.StateIdle
		sess.Draft = session.Draft{}
		return h.showBookCard(ctx, bot, chatID, id)
	}

	return nil
}

func (h *LibraryHandler) HandleEditMessage(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	bookID := sess.Draft.BookID
	field := sess.Draft.EditField

	var input service.UpdateBookInput
	switch field {
	case "title":
		input.Title = &text
	case "author":
		input.Author = &text
	case "liked":
		input.LikedText = &text
	case "disliked":
		input.DislikedText = &text
	case "insight":
		input.InsightText = &text
	default:
		sess.State = session.StateIdle
		sess.Draft = session.Draft{}
		return sendMessage(bot, chatID, "Неизвестное поле\\.")
	}

	book, err := h.books.Update(ctx, bookID, input)
	if err != nil {
		return fmt.Errorf("handler.Library.HandleEditMessage: %w", err)
	}

	sess.State = session.StateIdle
	sess.Draft = session.Draft{}

	card := formatBookCard(book)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("b:e:%d", book.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("b:d:%d", book.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅ К списку", "l:p:1"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "✅ Обновлено\\!\n\n"+card, kb)
}

func (h *LibraryHandler) showBookCard(ctx context.Context, bot *tgbotapi.BotAPI, chatID, id int64) error {
	book, err := h.books.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return sendMessage(bot, chatID, "Книга не найдена\\.")
		}
		return fmt.Errorf("handler.Library.showBookCard: %w", err)
	}
	card := formatBookCard(book)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("b:e:%d", book.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("b:d:%d", book.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅ К списку", "l:p:1"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, card, kb)
}

func fieldLabel(field string) string {
	switch field {
	case "title":
		return "название"
	case "author":
		return "автора"
	case "liked":
		return "что зацепило"
	case "disliked":
		return "что не понравилось"
	case "insight":
		return "инсайт"
	default:
		return field
	}
}
