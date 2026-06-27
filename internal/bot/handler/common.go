package handler

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/arrowwhi/books_review_bot/internal/domain"
)

func escapeMarkdown(s string) string {
	r := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return r.Replace(s)
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err := bot.Send(msg)
	return err
}

func sendMessageWithKeyboard(bot *tgbotapi.BotAPI, chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.ReplyMarkup = kb
	_, err := bot.Send(msg)
	return err
}

func answerCallback(bot *tgbotapi.BotAPI, queryID string) {
	cb := tgbotapi.NewCallback(queryID, "")
	_, _ = bot.Request(cb)
}

func ratingStars(r int16) string {
	var sb strings.Builder
	for i := int16(1); i <= 5; i++ {
		if i <= r {
			sb.WriteRune('★')
		} else {
			sb.WriteRune('☆')
		}
	}
	return sb.String()
}

func emotionLabel(e domain.Emotion) string {
	switch e {
	case domain.EmotionLove:
		return "😍 Восторг"
	case domain.EmotionLike:
		return "🙂 Понравилось"
	case domain.EmotionNeutral:
		return "😐 Нейтрально"
	case domain.EmotionDislike:
		return "😕 Разочарование"
	case domain.EmotionMixed:
		return "🤔 Неоднозначно"
	default:
		return string(e)
	}
}

func formatBookCard(b *domain.Book) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📚 *%s*\n", escapeMarkdown(b.Title)))

	author := b.Author
	if author == "" {
		author = "неизвестен"
	}
	sb.WriteString(fmt.Sprintf("👤 Автор: %s\n", escapeMarkdown(author)))

	genre := "не указан"
	if b.Genre != nil {
		genre = b.Genre.Name
	}
	sb.WriteString(fmt.Sprintf("🏷 Жанр: %s\n", escapeMarkdown(genre)))

	if b.Rating != nil {
		sb.WriteString(fmt.Sprintf("⭐️ Рейтинг: %s \\(%d/5\\)\n", ratingStars(*b.Rating), *b.Rating))
	}

	if b.Emotion != nil {
		sb.WriteString(emotionLabel(*b.Emotion) + "\n")
	}

	hasAspect := b.AspectPlot != nil || b.AspectChars != nil || b.AspectAtmo != nil ||
		b.AspectIdeas != nil || b.AspectStyle != nil || b.AspectTempo != nil

	if hasAspect {
		sb.WriteString("\n📊 *Аспекты:*\n")
		if b.AspectPlot != nil {
			sb.WriteString(fmt.Sprintf("• Сюжет: %d/10\n", *b.AspectPlot))
		}
		if b.AspectChars != nil {
			sb.WriteString(fmt.Sprintf("• Персонажи: %d/10\n", *b.AspectChars))
		}
		if b.AspectAtmo != nil {
			sb.WriteString(fmt.Sprintf("• Атмосфера: %d/10\n", *b.AspectAtmo))
		}
		if b.AspectIdeas != nil {
			sb.WriteString(fmt.Sprintf("• Идеи: %d/10\n", *b.AspectIdeas))
		}
		if b.AspectStyle != nil {
			sb.WriteString(fmt.Sprintf("• Язык: %d/10\n", *b.AspectStyle))
		}
		if b.AspectTempo != nil {
			sb.WriteString(fmt.Sprintf("• Темп: %d/10\n", *b.AspectTempo))
		}
	}

	if b.LikedText != "" {
		sb.WriteString(fmt.Sprintf("\n💬 Зацепило: %s\n", escapeMarkdown(b.LikedText)))
	}
	if b.DislikedText != "" {
		sb.WriteString(fmt.Sprintf("😞 Не понравилось: %s\n", escapeMarkdown(b.DislikedText)))
	}
	if b.InsightText != "" {
		sb.WriteString(fmt.Sprintf("💡 Инсайт: %s\n", escapeMarkdown(b.InsightText)))
	}

	if b.Recommend != nil {
		rec := "Нет"
		if *b.Recommend {
			rec = "Да"
		}
		sb.WriteString(fmt.Sprintf("\n👍 Порекомендую: %s\n", rec))
	}

	date := b.CreatedAt
	if b.FinishedAt != nil {
		date = *b.FinishedAt
	}
	sb.WriteString(fmt.Sprintf("\n📅 Прочитано: %s", escapeMarkdown(date.Format("2 January 2006"))))

	return sb.String()
}
