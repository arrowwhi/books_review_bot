package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
	"github.com/arrowwhi/books_review_bot/internal/service"
)

type StatsHandler struct {
	stats  service.StatsService
	logger *zap.Logger
}

func NewStatsHandler(stats service.StatsService, logger *zap.Logger) *StatsHandler {
	return &StatsHandler{stats: stats, logger: logger}
}

func (h *StatsHandler) HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	stats, err := h.stats.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("handler.Stats.HandleCommand: %w", err)
	}

	if stats.TotalBooks == 0 {
		return sendMessage(bot, chatID, "Статистика пока пуста\\. Добавь первую книгу через /add")
	}

	year := time.Now().Year()
	var sb strings.Builder

	sb.WriteString("📊 *Ваша статистика*\n\n")
	sb.WriteString(fmt.Sprintf("📚 Прочитано: %d \\(%d в %d году\\)\n",
		stats.TotalBooks, stats.BooksThisYear, year))
	sb.WriteString(fmt.Sprintf("⭐️ Средний рейтинг: %s\n",
		escapeMarkdown(fmt.Sprintf("%.1f", stats.AvgRating))))

	if len(stats.GenreStats) > 0 {
		sb.WriteString("\n📁 *По жанрам:*\n")
		for _, gs := range stats.GenreStats {
			sb.WriteString(fmt.Sprintf("• %s — %d кн\\., ср\\. %s ⭐\n",
				escapeMarkdown(gs.Genre.Name),
				gs.Count,
				escapeMarkdown(fmt.Sprintf("%.1f", gs.AvgRating)),
			))
		}
	}

	if len(stats.TopBooks) > 0 {
		sb.WriteString("\n🏆 *Топ книги:*\n")
		for i, b := range stats.TopBooks {
			line := fmt.Sprintf("%d\\. *%s*", i+1, escapeMarkdown(b.Title))
			if b.Rating != nil {
				line += fmt.Sprintf(" — %s", ratingStars(*b.Rating))
			}
			sb.WriteString(line + "\n")
		}
	}

	if stats.FavoriteAspect != "" {
		sb.WriteString(fmt.Sprintf("\n💪 Любимый аспект: %s \\(ср\\. %s\\)\n",
			escapeMarkdown(stats.FavoriteAspect),
			escapeMarkdown(fmt.Sprintf("%.1f", stats.FavAspectAvg)),
		))
	}

	sb.WriteString(fmt.Sprintf("\n🎯 В вишлисте: %d книг", stats.WishlistCount))

	return sendMessage(bot, chatID, sb.String())
}
