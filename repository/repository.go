package repository

import (
	"context"
	"errors"
	"time"

	"buy-ticket/domain"
)

var ErrIdempotencyKeyNotFound = errors.New("idempotency key not found")
var ErrPaymentAttemptNotFound = errors.New("payment attempt not found")
var ErrAdminUserNotFound = errors.New("admin user not found")
var ErrOrganizerNotFound = errors.New("organizer not found")

type AdminUserRepository interface {
	FindByID(ctx context.Context, adminUserID int64) (*domain.AdminUser, error)
	FindByEmail(ctx context.Context, email string) (*domain.AdminUser, error)
	List(ctx context.Context) ([]domain.AdminUser, error)
	Save(ctx context.Context, adminUser *domain.AdminUser) error
}

type OrganizerRepository interface {
	FindByID(ctx context.Context, organizerID int64) (*domain.Organizer, error)
	List(ctx context.Context) ([]domain.Organizer, error)
	Save(ctx context.Context, organizer *domain.Organizer) error
}

type AdminAuditLogRepository interface {
	Create(ctx context.Context, log *domain.AdminAuditLog) error
	List(ctx context.Context, filter domain.AdminAuditLogListFilter) ([]domain.AdminAuditLog, error)
}

type AdminOrderRepository interface {
	ListAdminOrders(ctx context.Context, filter domain.AdminOrderListFilter) ([]domain.AdminOrder, error)
	FindAdminOrderSensitiveByID(ctx context.Context, orderID int64) (*domain.AdminOrder, error)
}

type AdminEventRepository interface {
	ListAdminEvents(ctx context.Context, organizerID *int64) ([]domain.Event, error)
	FindAdminEventByID(ctx context.Context, eventID int64, organizerID *int64) (*domain.Event, error)
	CreateAdminEvent(ctx context.Context, event *domain.Event) error
	ListAdminEventSections(ctx context.Context, eventID int64, organizerID *int64) ([]domain.Section, error)
	CreateAdminEventSection(ctx context.Context, eventID int64, organizerID *int64, section *domain.Section) error
}

type EventRepository interface {
	FindByID(ctx context.Context, eventID int64) (*domain.Event, error)
	List(ctx context.Context) ([]domain.Event, error)
}

type UserRepository interface {
	FindByID(ctx context.Context, userID int64) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Save(ctx context.Context, user *domain.User) error
}

type SectionRepository interface {
	FindByEventAndID(ctx context.Context, eventID, sectionID int64) (*domain.Section, error)
	ListByEventID(ctx context.Context, eventID int64) ([]domain.Section, error)
	ReserveInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error)
	ReleaseInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error)
	ConfirmSale(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error)
	Save(ctx context.Context, section *domain.Section) error
}

type ReservationRepository interface {
	FindByID(ctx context.Context, reservationID int64) (*domain.Reservation, error)
	FindActiveByUserAndEvent(ctx context.Context, userID, eventID int64, now time.Time) (*domain.Reservation, error)
	ListByUserID(ctx context.Context, userID int64) ([]domain.Reservation, error)
	Save(ctx context.Context, reservation *domain.Reservation) error
}

type OrderRepository interface {
	FindByID(ctx context.Context, orderID int64) (*domain.Order, error)
	FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error)
	ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]domain.Order, error)
	ListByUserID(ctx context.Context, userID int64) ([]domain.Order, error)
	Save(ctx context.Context, order *domain.Order) error
}

type PaymentRepository interface {
	FindByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error)
	ListByUserID(ctx context.Context, userID int64) ([]domain.Payment, error)
	Save(ctx context.Context, payment *domain.Payment) error
}

type PaymentAttemptRepository interface {
	FindByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (*domain.PaymentAttempt, error)
	FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.PaymentAttempt, error)
	ListByOrderID(ctx context.Context, orderID int64) ([]domain.PaymentAttempt, error)
	Save(ctx context.Context, attempt *domain.PaymentAttempt) error
}

type IdempotencyRepository interface {
	FindByKeyAndEndpoint(ctx context.Context, key, endpoint string) (*domain.IdempotencyKey, error)
	Create(ctx context.Context, record *domain.IdempotencyKey) error
	Complete(ctx context.Context, key, endpoint string, status int, responseBody []byte, now time.Time) error
}
