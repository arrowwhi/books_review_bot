package repository

//go:generate go run go.uber.org/mock/mockgen@latest -source=reminder.go -destination=mock_reminder.go -package=repository

import (
	"context"

	"github.com/arrowwhi/books_review_bot/internal/domain"
)

type ReminderRepository interface {
	Get(ctx context.Context, userID int64) (*domain.Reminder, error)
	Upsert(ctx context.Context, r *domain.Reminder) error
	ListDue(ctx context.Context) ([]*domain.Reminder, error)
}
