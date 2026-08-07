package domain

import "time"

type IdempotencyStatus int8

const (
	IdempotencyStatusProcessing IdempotencyStatus = iota + 1
	IdempotencyStatusCompleted
)

type IdempotencyKey struct {
	ID             int64
	Key            string
	UserID         int64
	Endpoint       string
	RequestHash    string
	Status         IdempotencyStatus
	ResponseStatus *int
	ResponseBody   []byte
	LockedUntil    *time.Time
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
