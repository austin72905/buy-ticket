package service

import (
	"context"
	"errors"
	"time"

	"buy-ticket/db/sqlc"
	"buy-ticket/domain"
	"buy-ticket/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEventNotOnSale         = errors.New("event is not on sale")
	ErrSectionNotReservable   = errors.New("section cannot reserve requested quantity")
	ErrReservationNotActive   = errors.New("reservation is not active")
	ErrReservationAlreadyUsed = errors.New("reservation already confirmed or closed")
	ErrReservationCannotClose = errors.New("reservation cannot be expired or cancelled")
	ErrOrderCannotBePaid      = errors.New("order cannot be paid")
	ErrPaymentAmountMismatch  = errors.New("payment amount mismatch")
	ErrQueueTokenNotFound     = errors.New("queue token not found")
	ErrUserAlreadyJoinedQueue = errors.New("user already joined queue")
	ErrPurchaseTokenRequired  = errors.New("purchase token is required")
	ErrPurchaseTokenNotFound  = errors.New("purchase token not found")
	ErrPurchaseTokenExpired   = errors.New("purchase token expired")
	ErrPurchaseTokenUsed      = errors.New("purchase token already used")
	ErrPurchaseTokenMismatch  = errors.New("purchase token does not match user or event")
)

type BookingService struct {
	DB              *pgxpool.Pool
	EventRepo       repository.EventRepository
	SectionRepo     repository.SectionRepository
	ReservationRepo repository.ReservationRepository
	OrderRepo       repository.OrderRepository
	PaymentRepo     repository.PaymentRepository
	QueueStore      QueueStore
}

type ReserveTicketInput struct {
	UserID        int64
	EventID       int64
	SectionID     int64
	Quantity      int
	HoldUntil     time.Time
	PurchaseToken string
}

type CreateOrderInput struct {
	ReservationID int64
	OrderNo       string
	ExpiresAt     time.Time
}

type PayOrderInput struct {
	OrderID   int64
	PaymentNo string
	Method    string
	Amount    int64
	PaidAt    time.Time
}

type JoinQueueInput struct {
	EventID    int64
	UserID     int64
	ClientID   string
	RequestID  string
	Channel    string
	AccessCode string
}

type ExpireReservationInput struct {
	ReservationID int64
	ExpiredAt     time.Time
}

type CancelReservationInput struct {
	ReservationID int64
	CancelledAt   time.Time
}

type SectionAvailability struct {
	Section   domain.Section
	Available int
}

type SaleStatus struct {
	EventID      int64
	EventStatus  domain.EventStatus
	IsOnSale     bool
	QueueEnabled bool
	CanJoinQueue bool
	CanReserve   bool
	SaleStartAt  time.Time
	SaleEndAt    time.Time
	ServerTime   time.Time
}

type QueueStatus int8

const (
	QueueStatusWaiting QueueStatus = 1
	QueueStatusReady   QueueStatus = 2
	QueueStatusExpired QueueStatus = 3
)

type QueueStatusSnapshot struct {
	QueueToken             string
	QueueSequence          int64
	Status                 QueueStatus
	EventID                int64
	UserID                 int64
	QueuePosition          int64
	AheadCount             int64
	EstimatedWaitSeconds   int64
	PurchaseToken          *string
	PurchaseTokenExpiresAt *time.Time
	JoinedAt               time.Time
	ExpiredAt              time.Time
	UpdatedAt              time.Time
	PurchaseTokenUsedAt    *time.Time
}

func NewBookingService(
	eventRepo repository.EventRepository,
	sectionRepo repository.SectionRepository,
	reservationRepo repository.ReservationRepository,
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
) *BookingService {
	return &BookingService{
		EventRepo:       eventRepo,
		SectionRepo:     sectionRepo,
		ReservationRepo: reservationRepo,
		OrderRepo:       orderRepo,
		PaymentRepo:     paymentRepo,
		QueueStore:      NewMemoryQueueStore(1),
	}
}

func (s *BookingService) ReserveTicket(ctx context.Context, input ReserveTicketInput) (*domain.Reservation, error) {
	if input.PurchaseToken == "" {
		return nil, ErrPurchaseTokenRequired
	}

	now := time.Now()
	var reservation *domain.Reservation

	queueSnapshot, err := s.QueueStore.ConsumePurchaseToken(ctx, input.PurchaseToken, input.EventID, input.UserID, now)
	if err != nil {
		return nil, err
	}

	err = s.withTx(ctx, func(repos bookingRepos) error {
		event, err := repos.event.FindByID(ctx, input.EventID)
		if err != nil {
			return err
		}

		if !event.IsOnSale(now) {
			return ErrEventNotOnSale
		}

		section, err := repos.section.FindByEventAndID(ctx, input.EventID, input.SectionID)
		if err != nil {
			return err
		}

		if !section.Reserve(input.Quantity) {
			return ErrSectionNotReservable
		}

		section.UpdatedAt = now
		if err := repos.section.Save(ctx, section); err != nil {
			return err
		}

		reservation = &domain.Reservation{
			EventID:     input.EventID,
			SectionID:   input.SectionID,
			UserID:      input.UserID,
			Quantity:    input.Quantity,
			UnitPrice:   section.Price,
			TotalAmount: int64(input.Quantity) * section.Price,
			Status:      domain.ReservationStatusHolding,
			ExpiresAt:   input.HoldUntil,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		return repos.reservation.Save(ctx, reservation)
	})
	if err != nil {
		_ = s.QueueStore.RestorePurchaseToken(ctx, *queueSnapshot)
		return nil, err
	}

	return reservation, nil
}

func (s *BookingService) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	now := time.Now()

	reservation, err := s.ReservationRepo.FindByID(ctx, input.ReservationID)
	if err != nil {
		return nil, err
	}

	if !reservation.IsActive(now) {
		return nil, ErrReservationNotActive
	}

	order := &domain.Order{
		OrderNo:       input.OrderNo,
		UserID:        reservation.UserID,
		EventID:       reservation.EventID,
		SectionID:     reservation.SectionID,
		ReservationID: reservation.ID,
		Quantity:      reservation.Quantity,
		UnitPrice:     reservation.UnitPrice,
		TotalAmount:   reservation.TotalAmount,
		Status:        domain.OrderStatusPendingPayment,
		ExpiresAt:     input.ExpiresAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.OrderRepo.Save(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *BookingService) PayOrder(ctx context.Context, input PayOrderInput) (*domain.Payment, error) {
	var payment *domain.Payment

	err := s.withTx(ctx, func(repos bookingRepos) error {
		order, err := repos.order.FindByID(ctx, input.OrderID)
		if err != nil {
			return err
		}

		if !order.CanPay(input.PaidAt) {
			return ErrOrderCannotBePaid
		}

		if input.Amount != order.TotalAmount {
			return ErrPaymentAmountMismatch
		}

		reservation, err := repos.reservation.FindByID(ctx, order.ReservationID)
		if err != nil {
			return err
		}

		if !reservation.Confirm(input.PaidAt) {
			return ErrReservationAlreadyUsed
		}

		section, err := repos.section.FindByEventAndID(ctx, reservation.EventID, reservation.SectionID)
		if err != nil {
			return err
		}

		if !section.ConfirmSale(reservation.Quantity) {
			return ErrSectionNotReservable
		}

		if !order.MarkPaid(input.PaidAt) {
			return ErrOrderCannotBePaid
		}

		payment = &domain.Payment{
			OrderID:   order.ID,
			PaymentNo: input.PaymentNo,
			Method:    input.Method,
			Amount:    input.Amount,
			Status:    domain.PaymentStatusPending,
			CreatedAt: input.PaidAt,
			UpdatedAt: input.PaidAt,
		}

		if !payment.MarkPaid(input.PaidAt) {
			return ErrOrderCannotBePaid
		}

		section.UpdatedAt = input.PaidAt
		if err := repos.section.Save(ctx, section); err != nil {
			return err
		}

		if err := repos.reservation.Save(ctx, reservation); err != nil {
			return err
		}

		if err := repos.order.Save(ctx, order); err != nil {
			return err
		}

		return repos.payment.Save(ctx, payment)
	})
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *BookingService) ExpireReservation(ctx context.Context, input ExpireReservationInput) (*domain.Reservation, error) {
	var reservation *domain.Reservation

	err := s.withTx(ctx, func(repos bookingRepos) error {
		var err error
		reservation, err = repos.reservation.FindByID(ctx, input.ReservationID)
		if err != nil {
			return err
		}

		if !reservation.Expire(input.ExpiredAt) {
			return ErrReservationCannotClose
		}

		section, err := repos.section.FindByEventAndID(ctx, reservation.EventID, reservation.SectionID)
		if err != nil {
			return err
		}

		if !section.Release(reservation.Quantity) {
			return ErrReservationCannotClose
		}

		section.UpdatedAt = input.ExpiredAt
		if err := repos.section.Save(ctx, section); err != nil {
			return err
		}

		return repos.reservation.Save(ctx, reservation)
	})
	if err != nil {
		return nil, err
	}

	return reservation, nil
}

func (s *BookingService) CancelReservation(ctx context.Context, input CancelReservationInput) (*domain.Reservation, error) {
	var reservation *domain.Reservation

	err := s.withTx(ctx, func(repos bookingRepos) error {
		var err error
		reservation, err = repos.reservation.FindByID(ctx, input.ReservationID)
		if err != nil {
			return err
		}

		if !reservation.Cancel(input.CancelledAt) {
			return ErrReservationCannotClose
		}

		section, err := repos.section.FindByEventAndID(ctx, reservation.EventID, reservation.SectionID)
		if err != nil {
			return err
		}

		if !section.Release(reservation.Quantity) {
			return ErrReservationCannotClose
		}

		section.UpdatedAt = input.CancelledAt
		if err := repos.section.Save(ctx, section); err != nil {
			return err
		}

		return repos.reservation.Save(ctx, reservation)
	})
	if err != nil {
		return nil, err
	}

	return reservation, nil
}

func (s *BookingService) GetEvent(ctx context.Context, eventID int64) (*domain.Event, error) {
	return s.EventRepo.FindByID(ctx, eventID)
}

func (s *BookingService) ListEvents(ctx context.Context) ([]domain.Event, error) {
	return s.EventRepo.List(ctx)
}

func (s *BookingService) GetSections(ctx context.Context, eventID int64) ([]domain.Section, error) {
	return s.SectionRepo.ListByEventID(ctx, eventID)
}

func (s *BookingService) GetAvailability(ctx context.Context, eventID int64) ([]SectionAvailability, error) {
	sections, err := s.SectionRepo.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	availabilities := make([]SectionAvailability, 0, len(sections))
	for _, section := range sections {
		availabilities = append(availabilities, SectionAvailability{
			Section:   section,
			Available: section.AvailableQuantity(),
		})
	}

	return availabilities, nil
}

func (s *BookingService) GetOrder(ctx context.Context, orderID int64) (*domain.Order, error) {
	return s.OrderRepo.FindByID(ctx, orderID)
}

func (s *BookingService) GetReservation(ctx context.Context, reservationID int64) (*domain.Reservation, error) {
	return s.ReservationRepo.FindByID(ctx, reservationID)
}

func (s *BookingService) ListReservationsByUserID(ctx context.Context, userID int64) ([]domain.Reservation, error) {
	return s.ReservationRepo.ListByUserID(ctx, userID)
}

func (s *BookingService) GetOrderByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	return s.OrderRepo.FindByOrderNo(ctx, orderNo)
}

func (s *BookingService) ListOrdersByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	return s.OrderRepo.ListByUserID(ctx, userID)
}

func (s *BookingService) GetPaymentByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error) {
	return s.PaymentRepo.FindByPaymentNo(ctx, paymentNo)
}

func (s *BookingService) ListPaymentsByUserID(ctx context.Context, userID int64) ([]domain.Payment, error) {
	return s.PaymentRepo.ListByUserID(ctx, userID)
}

func (s *BookingService) GetSaleStatus(ctx context.Context, eventID int64, now time.Time) (*SaleStatus, error) {
	event, err := s.EventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	isOnSale := event.IsOnSale(now)
	return &SaleStatus{
		EventID:      event.ID,
		EventStatus:  event.Status,
		IsOnSale:     isOnSale,
		QueueEnabled: false,
		CanJoinQueue: isOnSale,
		CanReserve:   isOnSale,
		SaleStartAt:  event.SaleStartAt,
		SaleEndAt:    event.SaleEndAt,
		ServerTime:   now,
	}, nil
}

func (s *BookingService) JoinQueue(ctx context.Context, input JoinQueueInput, now time.Time) (*QueueStatusSnapshot, error) {
	event, err := s.EventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return nil, err
	}

	if !event.IsOnSale(now) {
		return nil, ErrEventNotOnSale
	}

	return s.QueueStore.Join(ctx, input, now)
}

func (s *BookingService) GetQueueStatus(ctx context.Context, queueToken string) (*QueueStatusSnapshot, error) {
	return s.QueueStore.Get(ctx, queueToken, time.Now())
}

func (s *BookingService) SaveQueueStatus(snapshot QueueStatusSnapshot) {
	_ = s.QueueStore.SaveSnapshot(context.Background(), snapshot)
}

type bookingRepos struct {
	event       repository.EventRepository
	section     repository.SectionRepository
	reservation repository.ReservationRepository
	order       repository.OrderRepository
	payment     repository.PaymentRepository
}

func (s *BookingService) withTx(ctx context.Context, fn func(repos bookingRepos) error) error {
	if s.DB == nil {
		return fn(bookingRepos{
			event:       s.EventRepo,
			section:     s.SectionRepo,
			reservation: s.ReservationRepo,
			order:       s.OrderRepo,
			payment:     s.PaymentRepo,
		})
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)
	repos := bookingRepos{
		event:       repository.NewPostgresEventRepository(queries),
		section:     repository.NewPostgresSectionRepository(queries),
		reservation: repository.NewPostgresReservationRepository(queries),
		order:       repository.NewPostgresOrderRepository(queries),
		payment:     repository.NewPostgresPaymentRepository(queries),
	}

	if err := fn(repos); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
