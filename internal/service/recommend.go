package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/client/claude"
	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

type RecommendSvc struct {
	statsRepo repository.BookRepository
	claude    claude.Client
	logger    *zap.Logger
}

func NewRecommendService(statsRepo repository.BookRepository, claude claude.Client, logger *zap.Logger) RecommendService {
	return &RecommendSvc{statsRepo: statsRepo, claude: claude, logger: logger}
}

func (s *RecommendSvc) Recommend(ctx context.Context, userID int64) (string, error) {
	stats, err := s.statsRepo.GetStats(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("service.Recommend: %w", err)
	}
	prompt := buildPrompt(stats)
	s.logger.Info("recommend", zap.Int64("user_id", userID))
	result, err := s.claude.Complete(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("service.Recommend: %w", err)
	}
	return result, nil
}

func buildPrompt(stats *domain.Stats) string {
	genreParts := make([]string, 0, len(stats.GenreStats))
	for _, gs := range stats.GenreStats {
		genreParts = append(genreParts, fmt.Sprintf("%s (рейтинг %.1f)", gs.Genre.Name, gs.AvgRating))
	}
	genresStr := strings.Join(genreParts, ", ")

	topParts := make([]string, 0, len(stats.TopBooks))
	for _, b := range stats.TopBooks {
		topParts = append(topParts, fmt.Sprintf("%q — %s", b.Title, b.Author))
	}
	topBooksStr := strings.Join(topParts, ", ")

	return fmt.Sprintf(`Ты помогаешь читателю найти новые книги.

Профиль читателя:
- Прочитано книг: %d
- Средний рейтинг: %.1f
- Любимые жанры: %s
- Любимый аспект: %s
- Топ книги: %s

Порекомендуй 3 книги которые понравятся этому читателю.
Формат: "Название" — Автор (Год) — одно предложение почему понравится.`,
		stats.TotalBooks, stats.AvgRating, genresStr, stats.FavoriteAspect, topBooksStr)
}
