package domain

import "time"

type OutboxEventStatus int8

const (
	OutboxEventStatusPending OutboxEventStatus = iota + 1
	OutboxEventStatusPublished
	OutboxEventStatusFailed
)

type OutboxEvent struct {
	ID            int64
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   int64
	Payload       []byte
	Status        OutboxEventStatus
	Attempts      int
	MaxAttempts   int
	NextAttemptAt *time.Time
	LastError     *string
	PublishedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (e *OutboxEvent) MarkPublished(now time.Time) {
	e.Status = OutboxEventStatusPublished
	e.PublishedAt = &now
	e.NextAttemptAt = nil
	e.LastError = nil
	e.UpdatedAt = now
}

func (e *OutboxEvent) MarkPublishFailed(reason string, nextAttemptAt *time.Time, now time.Time) {
	e.Attempts++
	e.LastError = optionalString(reason)
	e.NextAttemptAt = nextAttemptAt
	if e.MaxAttempts > 0 && e.Attempts >= e.MaxAttempts {
		e.Status = OutboxEventStatusFailed
		e.NextAttemptAt = nil
	}
	e.UpdatedAt = now
}
