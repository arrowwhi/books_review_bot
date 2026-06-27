package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

type GenreSvc struct {
	repo   repository.GenreRepository
	logger *zap.Logger
}

func NewGenreService(repo repository.GenreRepository, logger *zap.Logger) GenreService {
	return &GenreSvc{repo: repo, logger: logger}
}

func (s *GenreSvc) List(ctx context.Context) ([]*domain.Genre, error) {
	genres, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("service.List: %w", err)
	}
	return genres, nil
}

func (s *GenreSvc) Create(ctx context.Context, name string) (*domain.Genre, error) {
	genre, err := s.repo.Create(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}
	return genre, nil
}
