package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
	"github.com/arrowwhi/books_review_bot/internal/client/openlibrary"
	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/service"
)

type AddHandler struct {
	books  service.BookService
	genres service.GenreService
	ol     openlibrary.Client
	logger *zap.Logger
}

func NewAddHandler(books service.BookService, genres service.GenreService, ol openlibrary.Client, logger *zap.Logger) *AddHandler {
	return &AddHandler{books: books, genres: genres, ol: ol, logger: logger}
}

func (h *AddHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message, sess *session.Session) error {
	sess.Draft = session.Draft{}
	sess.State = session.StateAddTitle
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "a:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, msg.Chat.ID, "Введи название книги:", kb)
}

func (h *AddHandler) HandleMessage(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message, sess *session.Session) error {
	chatID := msg.Chat.ID
	text := msg.Text

	switch sess.State {
	case session.StateAddTitle:
		sess.Draft.Title = text
		results, err := h.ol.Search(ctx, text)
		if err != nil || len(results) == 0 {
			sess.State = session.StateAddAuthor
			return h.askAuthor(bot, chatID)
		}
		sess.Draft.OLResults = results
		sess.State = session.StateAddSearchResult
		return h.showOLResults(bot, chatID, results)

	case session.StateAddAuthor:
		sess.Draft.Author = text
		sess.State = session.StateAddGenre
		return h.askGenre(ctx, bot, chatID, sess)

	case session.StateAddCustomGenre:
		genre, err := h.genres.Create(ctx, text)
		if err != nil {
			return sendMessage(bot, chatID, escapeMarkdown(fmt.Sprintf("Ошибка создания жанра: %v", err)))
		}
		id32 := genre.ID
		sess.Draft.GenreID = &id32
		sess.State = session.StateAddRating
		return h.askRating(bot, chatID)

	case session.StateAddLiked:
		sess.Draft.LikedText = text
		sess.State = session.StateAddDisliked
		return h.askDisliked(bot, chatID)

	case session.StateAddDisliked:
		sess.Draft.DislikedText = text
		sess.State = session.StateAddInsight
		return h.askInsight(bot, chatID)

	case session.StateAddInsight:
		sess.Draft.InsightText = text
		sess.State = session.StateAddRecommend
		return h.askRecommend(bot, chatID)
	}

	return nil
}

func (h *AddHandler) HandleCallback(ctx context.Context, bot *tgbotapi.BotAPI, q *tgbotapi.CallbackQuery, sess *session.Session) error {
	answerCallback(bot, q.ID)
	chatID := q.Message.Chat.ID
	userID := q.From.ID
	data := q.Data

	switch {
	case data == "a:cancel":
		sess.State = session.StateIdle
		sess.Draft = session.Draft{}
		return sendMessage(bot, chatID, "❌ Добавление отменено\\.")

	case strings.HasPrefix(data, "a:s:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) < 3 {
			return nil
		}
		if parts[2] == "skip" {
			sess.State = session.StateAddAuthor
			return h.askAuthor(bot, chatID)
		}
		idx, err := strconv.Atoi(parts[2])
		if err != nil || idx >= len(sess.Draft.OLResults) {
			sess.State = session.StateAddAuthor
			return h.askAuthor(bot, chatID)
		}
		result := sess.Draft.OLResults[idx]
		if len(result.AuthorNames) > 0 {
			sess.Draft.Author = result.AuthorNames[0]
		}
		sess.Draft.OLKey = result.Key
		sess.Draft.CoverURL = result.CoverURL
		sess.State = session.StateAddRating
		return h.askRating(bot, chatID)

	case data == "a:au:skip":
		sess.State = session.StateAddGenre
		return h.askGenre(ctx, bot, chatID, sess)

	case strings.HasPrefix(data, "a:g:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) < 3 {
			return nil
		}
		switch parts[2] {
		case "oth":
			return h.showCustomGenres(ctx, bot, chatID)
		case "new":
			sess.State = session.StateAddCustomGenre
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅ Назад", "a:g:oth"),
				),
			)
			return sendMessageWithKeyboard(bot, chatID, "Введи название нового жанра:", kb)
		default:
			id64, err := strconv.ParseInt(parts[2], 10, 32)
			if err != nil {
				return nil
			}
			id32 := int32(id64)
			sess.Draft.GenreID = &id32
			sess.State = session.StateAddRating
			return h.askRating(bot, chatID)
		}

	case strings.HasPrefix(data, "a:r:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			if v, err := strconv.Atoi(parts[2]); err == nil {
				v16 := int16(v)
				sess.Draft.Rating = &v16
			}
		}
		sess.State = session.StateAddEmotion
		return h.askEmotion(bot, chatID)

	case strings.HasPrefix(data, "a:e:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			e := domain.Emotion(parts[2])
			sess.Draft.Emotion = &e
		}
		sess.State = session.StateAddAspectPlot
		return h.askAspect(bot, chatID, "Сюжет / интрига", "a:ap")

	case strings.HasPrefix(data, "a:ap:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			if v, err := strconv.Atoi(parts[2]); err == nil {
				v16 := int16(v)
				sess.Draft.AspectPlot = &v16
			}
		}
		sess.State = session.StateAddAspectChars
		return h.askAspect(bot, chatID, "Персонажи", "a:ac")

	case strings.HasPrefix(data, "a:ac:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			if v, err := strconv.Atoi(parts[2]); err == nil {
				v16 := int16(v)
				sess.Draft.AspectChars = &v16
			}
		}
		sess.State = session.StateAddAspectAtmo
		return h.askAspect(bot, chatID, "Атмосфера / мир", "a:aa")

	case strings.HasPrefix(data, "a:aa:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			if v, err := strconv.Atoi(parts[2]); err == nil {
				v16 := int16(v)
				sess.Draft.AspectAtmo = &v16
			}
		}
		sess.State = session.StateAddAspectIdeas
		return h.askAspect(bot, chatID, "Идеи и смыслы", "a:ai")

	case strings.HasPrefix(data, "a:ai:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			if v, err := strconv.Atoi(parts[2]); err == nil {
				v16 := int16(v)
				sess.Draft.AspectIdeas = &v16
			}
		}
		sess.State = session.StateAddAspectStyle
		return h.askAspect(bot, chatID, "Язык и стиль", "a:as")

	case strings.HasPrefix(data, "a:as:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			if v, err := strconv.Atoi(parts[2]); err == nil {
				v16 := int16(v)
				sess.Draft.AspectStyle = &v16
			}
		}
		sess.State = session.StateAddAspectTempo
		return h.askAspect(bot, chatID, "Темп", "a:at")

	case strings.HasPrefix(data, "a:at:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) >= 3 && parts[2] != "skip" {
			if v, err := strconv.Atoi(parts[2]); err == nil {
				v16 := int16(v)
				sess.Draft.AspectTempo = &v16
			}
		}
		sess.State = session.StateAddLiked
		return h.askLiked(bot, chatID)

	case data == "a:asp:skip":
		sess.State = session.StateAddLiked
		return h.askLiked(bot, chatID)

	case data == "a:lk:skip":
		sess.State = session.StateAddDisliked
		return h.askDisliked(bot, chatID)

	case data == "a:dl:skip":
		sess.State = session.StateAddInsight
		return h.askInsight(bot, chatID)

	case data == "a:in:skip":
		sess.State = session.StateAddRecommend
		return h.askRecommend(bot, chatID)

	case data == "a:rec:yes" || data == "a:rec:no":
		yes := data == "a:rec:yes"
		sess.Draft.Recommend = &yes
		return h.saveAndShow(ctx, bot, chatID, userID, sess)
	}

	return nil
}

func (h *AddHandler) askAuthor(bot *tgbotapi.BotAPI, chatID int64) error {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "a:au:skip"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "a:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "Введи автора\\:", kb)
}

func (h *AddHandler) showOLResults(bot *tgbotapi.BotAPI, chatID int64, results []openlibrary.Book) error {
	var sb strings.Builder
	sb.WriteString("🔍 *Нашёл по названию:*\n\n")
	for i, book := range results {
		sb.WriteString(fmt.Sprintf("%d\\. %s\n", i+1, escapeMarkdown(book.Title)))
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, book := range results {
		author := ""
		if len(book.AuthorNames) > 0 {
			author = book.AuthorNames[0]
		}
		btnText := fmt.Sprintf("%d. %s — %s", i+1, book.Title, author)
		runes := []rune(btnText)
		if len(runes) > 40 {
			btnText = string(runes[:37]) + "..."
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("a:s:%d", i)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Не то", "a:s:skip"),
		tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return sendMessageWithKeyboard(bot, chatID, sb.String(), kb)
}

func (h *AddHandler) askGenre(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, sess *session.Session) error {
	genres, err := h.genres.List(ctx)
	if err != nil {
		return fmt.Errorf("handler.askGenre: %w", err)
	}

	var defaultGenres []*domain.Genre
	for _, g := range genres {
		if g.IsDefault {
			defaultGenres = append(defaultGenres, g)
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(defaultGenres); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(defaultGenres[i].Name, fmt.Sprintf("a:g:%d", defaultGenres[i].ID)),
		}
		if i+1 < len(defaultGenres) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(defaultGenres[i+1].Name, fmt.Sprintf("a:g:%d", defaultGenres[i+1].ID)))
		}
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📂 Другое", "a:g:oth"),
		tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
	))

	_ = sess
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return sendMessageWithKeyboard(bot, chatID, "Выбери жанр:", kb)
}

func (h *AddHandler) showCustomGenres(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64) error {
	genres, err := h.genres.List(ctx)
	if err != nil {
		return fmt.Errorf("handler.showCustomGenres: %w", err)
	}

	var customGenres []*domain.Genre
	for _, g := range genres {
		if !g.IsDefault {
			customGenres = append(customGenres, g)
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	if len(customGenres) == 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Новый жанр", "a:g:new"),
		))
	} else {
		for _, g := range customGenres {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(g.Name, fmt.Sprintf("a:g:%d", g.ID)),
			))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Новый жанр", "a:g:new"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
		))
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return sendMessageWithKeyboard(bot, chatID, "Выбери жанр:", kb)
}

func (h *AddHandler) askRating(bot *tgbotapi.BotAPI, chatID int64) error {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("★1", "a:r:1"),
			tgbotapi.NewInlineKeyboardButtonData("★2", "a:r:2"),
			tgbotapi.NewInlineKeyboardButtonData("★3", "a:r:3"),
			tgbotapi.NewInlineKeyboardButtonData("★4", "a:r:4"),
			tgbotapi.NewInlineKeyboardButtonData("★5", "a:r:5"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "a:r:skip"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "Поставь рейтинг:", kb)
}

func (h *AddHandler) askEmotion(bot *tgbotapi.BotAPI, chatID int64) error {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("😍 Восторг", "a:e:love"),
			tgbotapi.NewInlineKeyboardButtonData("🙂 Понравилось", "a:e:like"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("😐 Нейтрально", "a:e:neutral"),
			tgbotapi.NewInlineKeyboardButtonData("😕 Разочарование", "a:e:dislike"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤔 Неоднозначно", "a:e:mixed"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "a:e:skip"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "Какое ощущение от книги?", kb)
}

func (h *AddHandler) askAspect(bot *tgbotapi.BotAPI, chatID int64, name, prefix string) error {
	text := escapeMarkdown(name) + " \\(1–10\\):"

	row1 := make([]tgbotapi.InlineKeyboardButton, 5)
	for i := 0; i < 5; i++ {
		n := i + 1
		row1[i] = tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d", n), fmt.Sprintf("%s:%d", prefix, n))
	}
	row2 := make([]tgbotapi.InlineKeyboardButton, 5)
	for i := 0; i < 5; i++ {
		n := i + 6
		row2[i] = tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d", n), fmt.Sprintf("%s:%d", prefix, n))
	}

	row3 := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", prefix+":skip"),
	}
	if prefix == "a:ap" {
		row3 = append(row3, tgbotapi.NewInlineKeyboardButtonData("⏭⏭ Пропустить всё", "a:asp:skip"))
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(row1, row2, row3)
	return sendMessageWithKeyboard(bot, chatID, text, kb)
}

func (h *AddHandler) askLiked(bot *tgbotapi.BotAPI, chatID int64) error {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "a:lk:skip"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "💬 Что зацепило в книге? Напиши свободно или пропусти:", kb)
}

func (h *AddHandler) askDisliked(bot *tgbotapi.BotAPI, chatID int64) error {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "a:dl:skip"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "😞 Что не понравилось? Или пропусти:", kb)
}

func (h *AddHandler) askInsight(bot *tgbotapi.BotAPI, chatID int64) error {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "a:in:skip"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отмена", "a:cancel"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "💡 Мысль или инсайт от книги? Или пропусти:", kb)
}

func (h *AddHandler) askRecommend(bot *tgbotapi.BotAPI, chatID int64) error {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👍 Да", "a:rec:yes"),
			tgbotapi.NewInlineKeyboardButtonData("👎 Нет", "a:rec:no"),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "👍 Порекомендовал бы эту книгу?", kb)
}

func (h *AddHandler) saveAndShow(ctx context.Context, bot *tgbotapi.BotAPI, chatID, userID int64, sess *session.Session) error {
	now := time.Now()
	input := service.AddBookInput{
		Title:        sess.Draft.Title,
		Author:       sess.Draft.Author,
		OLKey:        sess.Draft.OLKey,
		CoverURL:     sess.Draft.CoverURL,
		GenreID:      sess.Draft.GenreID,
		Status:       domain.StatusRead,
		Rating:       sess.Draft.Rating,
		Emotion:      sess.Draft.Emotion,
		AspectPlot:   sess.Draft.AspectPlot,
		AspectChars:  sess.Draft.AspectChars,
		AspectAtmo:   sess.Draft.AspectAtmo,
		AspectIdeas:  sess.Draft.AspectIdeas,
		AspectStyle:  sess.Draft.AspectStyle,
		AspectTempo:  sess.Draft.AspectTempo,
		LikedText:    sess.Draft.LikedText,
		DislikedText: sess.Draft.DislikedText,
		InsightText:  sess.Draft.InsightText,
		Recommend:    sess.Draft.Recommend,
		FinishedAt:   &now,
	}
	book, err := h.books.Add(ctx, userID, input)
	if err != nil {
		return fmt.Errorf("handler.saveAndShow: %w", err)
	}

	sess.State = session.StateIdle
	sess.Draft = session.Draft{}

	card := formatBookCard(book)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("b:e:%d", book.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("b:d:%d", book.ID)),
		),
	)
	return sendMessageWithKeyboard(bot, chatID, "✅ Книга добавлена\\!\n\n"+card, kb)
}
