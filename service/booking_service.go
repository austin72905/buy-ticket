package service

import (
	"context"
	"errors"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

var (
	ErrEventNotOnSale         = errors.New("event is not on sale")
	ErrSectionNotReservable   = errors.New("section cannot reserve requested quantity")
	ErrReservationNotActive   = errors.New("reservation is not active")
	ErrReservationAlreadyUsed = errors.New("reservation already confirmed or closed")
	ErrReservationCannotClose = errors.New("reservation cannot be expired or cancelled")
	ErrOrderCannotBePaid      = errors.New("order cannot be paid")
	ErrPaymentAmountMismatch  = errors.New("payment amount mismatch")
)

type BookingService struct {
	EventRepo       repository.EventRepository
	SectionRepo     repository.SectionRepository
	ReservationRepo repository.ReservationRepository
	OrderRepo       repository.OrderRepository
	PaymentRepo     repository.PaymentRepository
}

type ReserveTicketInput struct {
	UserID    int64
	EventID   int64
	SectionID int64
	Quantity  int
	HoldUntil time.Time
}

type CreateOrderInput struct {
	ReservationID int64
	OrderNo       string
	ExpiresAt     time.Time
}

type PayOrderInput struct {
	OrderID    int64
	PaymentNo  string
	Method     string
	Amount     int64
	PaidAt     time.Time
}

type ExpireReservationInput struct {
	ReservationID int64
	ExpiredAt     time.Time
}

type CancelReservationInput struct {
	ReservationID int64
	CancelledAt   time.Time
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
	}
}

func (s *BookingService) ReserveTicket(ctx context.Context, input ReserveTicketInput) (*domain.Reservation, error) {
	now := time.Now()

	event, err := s.EventRepo.FindByID(ctx, input.EventID)
	if err != nil {
		return nil, err
	}

	if !event.IsOnSale(now) {
		return nil, ErrEventNotOnSale
	}

	section, err := s.SectionRepo.FindByEventAndID(ctx, input.EventID, input.SectionID)
	if err != nil {
		return nil, err
	}

	if !section.Reserve(input.Quantity) {
		return nil, ErrSectionNotReservable
	}

	section.UpdatedAt = now
	if err := s.SectionRepo.Save(ctx, section); err != nil {
		return nil, err
	}

	reservation := &domain.Reservation{
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

	if err := s.ReservationRepo.Save(ctx, reservation); err != nil {
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
		ReservationID: reservation.ID,
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
	order, err := s.OrderRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}

	if !order.CanPay(input.PaidAt) {
		return nil, ErrOrderCannotBePaid
	}

	if input.Amount != order.TotalAmount {
		return nil, ErrPaymentAmountMismatch
	}

	reservation, err := s.ReservationRepo.FindByID(ctx, order.ReservationID)
	if err != nil {
		return nil, err
	}

	if !reservation.Confirm(input.PaidAt) {
		return nil, ErrReservationAlreadyUsed
	}

	section, err := s.SectionRepo.FindByEventAndID(ctx, reservation.EventID, reservation.SectionID)
	if err != nil {
		return nil, err
	}

	if !section.ConfirmSale(reservation.Quantity) {
		return nil, ErrSectionNotReservable
	}

	if !order.MarkPaid(input.PaidAt) {
		return nil, ErrOrderCannotBePaid
	}

	payment := &domain.Payment{
		OrderID:   order.ID,
		PaymentNo: input.PaymentNo,
		Method:    input.Method,
		Amount:    input.Amount,
		Status:    domain.PaymentStatusPending,
		CreatedAt: input.PaidAt,
		UpdatedAt: input.PaidAt,
	}

	if !payment.MarkPaid(input.PaidAt) {
		return nil, ErrOrderCannotBePaid
	}

	section.UpdatedAt = input.PaidAt
	if err := s.SectionRepo.Save(ctx, section); err != nil {
		return nil, err
	}

	if err := s.ReservationRepo.Save(ctx, reservation); err != nil {
		return nil, err
	}

	if err := s.OrderRepo.Save(ctx, order); err != nil {
		return nil, err
	}

	if err := s.PaymentRepo.Save(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *BookingService) ExpireReservation(ctx context.Context, input ExpireReservationInput) (*domain.Reservation, error) {
	reservation, err := s.ReservationRepo.FindByID(ctx, input.ReservationID)
	if err != nil {
		return nil, err
	}

	if !reservation.Expire(input.ExpiredAt) {
		return nil, ErrReservationCannotClose
	}

	section, err := s.SectionRepo.FindByEventAndID(ctx, reservation.EventID, reservation.SectionID)
	if err != nil {
		return nil, err
	}

	if !section.Release(reservation.Quantity) {
		return nil, ErrReservationCannotClose
	}

	section.UpdatedAt = input.ExpiredAt
	if err := s.SectionRepo.Save(ctx, section); err != nil {
		return nil, err
	}

	if err := s.ReservationRepo.Save(ctx, reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}

func (s *BookingService) CancelReservation(ctx context.Context, input CancelReservationInput) (*domain.Reservation, error) {
	reservation, err := s.ReservationRepo.FindByID(ctx, input.ReservationID)
	if err != nil {
		return nil, err
	}

	if !reservation.Cancel(input.CancelledAt) {
		return nil, ErrReservationCannotClose
	}

	section, err := s.SectionRepo.FindByEventAndID(ctx, reservation.EventID, reservation.SectionID)
	if err != nil {
		return nil, err
	}

	if !section.Release(reservation.Quantity) {
		return nil, ErrReservationCannotClose
	}

	section.UpdatedAt = input.CancelledAt
	if err := s.SectionRepo.Save(ctx, section); err != nil {
		return nil, err
	}

	if err := s.ReservationRepo.Save(ctx, reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}
