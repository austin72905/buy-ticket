package repository

import (
	"context"

	"buy-ticket/domain"
)

type EventRepository interface {
	FindByID(ctx context.Context, eventID int64) (*domain.Event, error)
	List(ctx context.Context) ([]domain.Event, error)
}

type SectionRepository interface {
	FindByEventAndID(ctx context.Context, eventID, sectionID int64) (*domain.Section, error)
	ListByEventID(ctx context.Context, eventID int64) ([]domain.Section, error)
	Save(ctx context.Context, section *domain.Section) error
}

type ReservationRepository interface {
	FindByID(ctx context.Context, reservationID int64) (*domain.Reservation, error)
	ListByUserID(ctx context.Context, userID int64) ([]domain.Reservation, error)
	Save(ctx context.Context, reservation *domain.Reservation) error
}

type OrderRepository interface {
	FindByID(ctx context.Context, orderID int64) (*domain.Order, error)
	FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error)
	ListByUserID(ctx context.Context, userID int64) ([]domain.Order, error)
	Save(ctx context.Context, order *domain.Order) error
}

type PaymentRepository interface {
	FindByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error)
	ListByUserID(ctx context.Context, userID int64) ([]domain.Payment, error)
	Save(ctx context.Context, payment *domain.Payment) error
}
