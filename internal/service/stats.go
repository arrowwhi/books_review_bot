package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

type StatsSvc struct {
	repo   repository.BookRepository
	logger *zap.Logger
}

func NewStatsService(repo repository.BookRepository, logger *zap.Logger) StatsService {
	return &StatsSvc{repo: repo, logger: logger}
}

func (s *StatsSvc) Get(ctx context.Context, userID int64) (*domain.Stats, error) {
	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.Get: %w", err)
	}
	return stats, nil
}
