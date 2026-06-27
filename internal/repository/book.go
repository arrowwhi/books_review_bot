package repository

//go:generate go run go.uber.org/mock/mockgen@latest -source=book.go -destination=mock_book.go -package=repository

import (
	"context"

	"github.com/arrowwhi/books_review_bot/internal/domain"
)

type BookRepository interface {
	Create(ctx context.Context, book *domain.Book) (*domain.Book, error)
	GetByID(ctx context.Context, id int64) (*domain.Book, error)
	Update(ctx context.Context, book *domain.Book) (*domain.Book, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, userID int64, status domain.BookStatus, offset, limit int) ([]*domain.Book, int, error)
	Search(ctx context.Context, userID int64, query string, offset, limit int) ([]*domain.Book, int, error)
	GetStats(ctx context.Context, userID int64) (*domain.Stats, error)
}
