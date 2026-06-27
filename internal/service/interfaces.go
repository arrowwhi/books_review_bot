package service

//go:generate go run go.uber.org/mock/mockgen@latest -source=interfaces.go -destination=mock_interfaces.go -package=service

import (
	"context"
	"time"

	"github.com/arrowwhi/books_review_bot/internal/domain"
)

const PageSize = 5

type AddBookInput struct {
	Title        string
	Author       string
	GenreID      *int32
	OLKey        string
	CoverURL     string
	Status       domain.BookStatus
	Rating       *int16
	Emotion      *domain.Emotion
	AspectPlot   *int16
	AspectChars  *int16
	AspectAtmo   *int16
	AspectIdeas  *int16
	AspectStyle  *int16
	AspectTempo  *int16
	LikedText    string
	DislikedText string
	InsightText  string
	Recommend    *bool
	FinishedAt   *time.Time
}

type UpdateBookInput struct {
	Title        *string
	Author       *string
	GenreID      *int32
	Rating       *int16
	Emotion      *domain.Emotion
	AspectPlot   *int16
	AspectChars  *int16
	AspectAtmo   *int16
	AspectIdeas  *int16
	AspectStyle  *int16
	AspectTempo  *int16
	LikedText    *string
	DislikedText *string
	InsightText  *string
	Recommend    *bool
}

type BookList struct {
	Books      []*domain.Book
	Total      int
	Page       int
	TotalPages int
}

type BookService interface {
	Add(ctx context.Context, userID int64, input AddBookInput) (*domain.Book, error)
	GetByID(ctx context.Context, id int64) (*domain.Book, error)
	Update(ctx context.Context, id int64, input UpdateBookInput) (*domain.Book, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, userID int64, status domain.BookStatus, page int) (*BookList, error)
	Search(ctx context.Context, userID int64, query string, page int) (*BookList, error)
	MoveToRead(ctx context.Context, id int64) (*domain.Book, error)
}

type GenreService interface {
	List(ctx context.Context) ([]*domain.Genre, error)
	Create(ctx context.Context, name string) (*domain.Genre, error)
}

type StatsService interface {
	Get(ctx context.Context, userID int64) (*domain.Stats, error)
}

type RecommendService interface {
	Recommend(ctx context.Context, userID int64) (string, error)
}

type ReminderService interface {
	Set(ctx context.Context, userID int64, intervalDays int) error
	Disable(ctx context.Context, userID int64) error
	Get(ctx context.Context, userID int64) (*domain.Reminder, error)
	Start(ctx context.Context, send func(userID int64, text string))
}
