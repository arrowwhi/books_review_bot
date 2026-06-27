package repository

//go:generate go run go.uber.org/mock/mockgen@latest -source=genre.go -destination=mock_genre.go -package=repository

import (
	"context"

	"github.com/arrowwhi/books_review_bot/internal/domain"
)

type GenreRepository interface {
	List(ctx context.Context) ([]*domain.Genre, error)
	GetByID(ctx context.Context, id int32) (*domain.Genre, error)
	Create(ctx context.Context, name string) (*domain.Genre, error)
}
