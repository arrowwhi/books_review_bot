package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

type ReminderSvc struct {
	repo   repository.ReminderRepository
	logger *zap.Logger
}

func NewReminderService(repo repository.ReminderRepository, logger *zap.Logger) ReminderService {
	return &ReminderSvc{repo: repo, logger: logger}
}

func (s *ReminderSvc) Set(ctx context.Context, userID int64, intervalDays int) error {
	r := &domain.Reminder{
		UserID:       userID,
		IntervalDays: intervalDays,
		Enabled:      true,
	}
	if err := s.repo.Upsert(ctx, r); err != nil {
		return fmt.Errorf("service.Set: %w", err)
	}
	return nil
}

func (s *ReminderSvc) Disable(ctx context.Context, userID int64) error {
	r, err := s.repo.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("service.Disable: %w", err)
	}
	r.Enabled = false
	if err := s.repo.Upsert(ctx, r); err != nil {
		return fmt.Errorf("service.Disable: %w", err)
	}
	return nil
}

func (s *ReminderSvc) Get(ctx context.Context, userID int64) (*domain.Reminder, error) {
	r, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.Get: %w", err)
	}
	return r, nil
}

func (s *ReminderSvc) Start(ctx context.Context, send func(userID int64, text string)) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.checkAndSend(ctx, send)
			}
		}
	}()
}

func (s *ReminderSvc) checkAndSend(ctx context.Context, send func(int64, string)) {
	reminders, err := s.repo.ListDue(ctx)
	if err != nil {
		s.logger.Error("reminder list due", zap.Error(err))
		return
	}
	now := time.Now()
	for _, r := range reminders {
		send(r.UserID, "📚 Привет! Ты давно не добавлял новые книги. Есть что-то интересное, что прочитал?")
		r.LastSentAt = &now
		if err := s.repo.Upsert(ctx, r); err != nil {
			s.logger.Error("reminder upsert", zap.Int64("user_id", r.UserID), zap.Error(err))
		}
	}
}
