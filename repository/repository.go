package repository

import (
	"context"

	"buy-ticket/domain"
)

type EventRepository interface {
	FindByID(ctx context.Context, eventID int64) (*domain.Event, error)
}

type SectionRepository interface {
	FindByEventAndID(ctx context.Context, eventID, sectionID int64) (*domain.Section, error)
	Save(ctx context.Context, section *domain.Section) error
}

type ReservationRepository interface {
	FindByID(ctx context.Context, reservationID int64) (*domain.Reservation, error)
	Save(ctx context.Context, reservation *domain.Reservation) error
}

type OrderRepository interface {
	FindByID(ctx context.Context, orderID int64) (*domain.Order, error)
	Save(ctx context.Context, order *domain.Order) error
}

type PaymentRepository interface {
	Save(ctx context.Context, payment *domain.Payment) error
}
