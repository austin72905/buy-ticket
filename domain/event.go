package domain

import "time"

type EventStatus int8

const (
	EventStatusDraft EventStatus = iota + 1
	EventStatusPublished
	EventStatusOnSale
	EventStatusEnded
)

type Event struct {
	ID          int64
	Name        string
	StartAt     time.Time
	EndAt       time.Time
	SaleStartAt time.Time
	SaleEndAt   time.Time
	Venue       string
	Status      EventStatus
	Sections    []Section
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (e Event) IsSaleStarted(now time.Time) bool {
	return !now.Before(e.SaleStartAt)
}

func (e Event) IsSaleEnded(now time.Time) bool {
	return now.After(e.SaleEndAt)
}

func (e Event) IsOnSale(now time.Time) bool {
	if e.Status != EventStatusOnSale {
		return false
	}

	return e.IsSaleStarted(now) && !e.IsSaleEnded(now)
}
