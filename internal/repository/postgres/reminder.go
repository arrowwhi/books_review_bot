package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

type ReminderRepo struct {
	pool *pgxpool.Pool
}

func NewReminderRepo(pool *pgxpool.Pool) repository.ReminderRepository {
	return &ReminderRepo{pool: pool}
}

func (r *ReminderRepo) Get(ctx context.Context, userID int64) (*domain.Reminder, error) {
	var rem domain.Reminder
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, interval_days, last_sent_at, enabled FROM reminders WHERE user_id=$1`,
		userID,
	).Scan(&rem.UserID, &rem.IntervalDays, &rem.LastSentAt, &rem.Enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.Reminder{UserID: userID, IntervalDays: 14, Enabled: false}, nil
		}
		return nil, fmt.Errorf("postgres.Reminder.Get: %w", err)
	}
	return &rem, nil
}

func (r *ReminderRepo) Upsert(ctx context.Context, rem *domain.Reminder) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO reminders (user_id, interval_days, last_sent_at, enabled)
VALUES ($1,$2,$3,$4)
ON CONFLICT (user_id) DO UPDATE SET
    interval_days=EXCLUDED.interval_days,
    last_sent_at=EXCLUDED.last_sent_at,
    enabled=EXCLUDED.enabled`,
		rem.UserID, rem.IntervalDays, rem.LastSentAt, rem.Enabled,
	)
	if err != nil {
		return fmt.Errorf("postgres.Reminder.Upsert: %w", err)
	}
	return nil
}

func (r *ReminderRepo) ListDue(ctx context.Context) ([]*domain.Reminder, error) {
	rows, err := r.pool.Query(ctx, `
SELECT user_id, interval_days, last_sent_at, enabled
FROM reminders
WHERE enabled = true
  AND (last_sent_at IS NULL OR last_sent_at + interval_days * INTERVAL '1 day' <= NOW())`)
	if err != nil {
		return nil, fmt.Errorf("postgres.Reminder.ListDue: %w", err)
	}
	defer rows.Close()

	var reminders []*domain.Reminder
	for rows.Next() {
		var rem domain.Reminder
		if err := rows.Scan(&rem.UserID, &rem.IntervalDays, &rem.LastSentAt, &rem.Enabled); err != nil {
			return nil, fmt.Errorf("postgres.Reminder.ListDue scan: %w", err)
		}
		reminders = append(reminders, &rem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.Reminder.ListDue rows: %w", err)
	}

	return reminders, nil
}
