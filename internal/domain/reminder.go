package domain

import "time"

type Reminder struct {
	UserID       int64
	IntervalDays int
	LastSentAt   *time.Time
	Enabled      bool
}
