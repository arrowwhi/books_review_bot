package service

import (
	"context"
	"encoding/json"
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

func (s *RecommendSvc) Recommend(ctx context.Context, userID int64) ([]domain.RecommendedBook, error) {
	stats, err := s.statsRepo.GetStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.Recommend: %w", err)
	}
	prompt := buildPrompt(stats)
	s.logger.Info("recommend", zap.Int64("user_id", userID))
	result, err := s.claude.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("service.Recommend: %w", err)
	}
	books, err := parseRecommendations(result)
	if err != nil {
		return nil, fmt.Errorf("service.Recommend: parse: %w", err)
	}
	return books, nil
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

Порекомендуй ровно 3 книги, которые понравятся этому читателю.
Верни ответ строго в виде JSON-массива без markdown-обёрток и пояснений:
[{"title":"...","author":"...","year":2020,"reason":"одно предложение почему понравится"}]`,
		stats.TotalBooks, stats.AvgRating, genresStr, stats.FavoriteAspect, topBooksStr)
}

func parseRecommendations(raw string) ([]domain.RecommendedBook, error) {
	raw = strings.TrimSpace(raw)
	// strip markdown code block if Claude wrapped it anyway
	if idx := strings.Index(raw, "["); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "]"); idx >= 0 && idx < len(raw)-1 {
		raw = raw[:idx+1]
	}

	var items []struct {
		Title  string `json:"title"`
		Author string `json:"author"`
		Year   int    `json:"year"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w (raw: %q)", err, raw)
	}

	books := make([]domain.RecommendedBook, 0, len(items))
	for _, it := range items {
		books = append(books, domain.RecommendedBook{
			Title:  it.Title,
			Author: it.Author,
			Year:   it.Year,
			Reason: it.Reason,
		})
	}
	return books, nil
}
