package controller

import (
	"time"

	"buy-ticket/domain"
)

type ReserveTicketRequest struct {
	UserID    int64     `json:"user_id"`
	EventID   int64     `json:"event_id"`
	SectionID int64     `json:"section_id"`
	Quantity  int       `json:"quantity"`
	HoldUntil time.Time `json:"hold_until"`
}

type CreateOrderRequest struct {
	ReservationID int64     `json:"reservation_id"`
	OrderNo       string    `json:"order_no"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type PayOrderRequest struct {
	OrderID   int64     `json:"order_id"`
	PaymentNo string    `json:"payment_no"`
	Method    string    `json:"method"`
	Amount    int64     `json:"amount"`
	PaidAt    time.Time `json:"paid_at"`
}

type ExpireReservationRequest struct {
	ReservationID int64     `json:"reservation_id"`
	ExpiredAt     time.Time `json:"expired_at"`
}

type CancelReservationRequest struct {
	ReservationID int64     `json:"reservation_id"`
	CancelledAt   time.Time `json:"cancelled_at"`
}

type ReservationResponse struct {
	ID          int64  `json:"id"`
	EventID     int64  `json:"event_id"`
	SectionID   int64  `json:"section_id"`
	UserID      int64  `json:"user_id"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	TotalAmount int64  `json:"total_amount"`
	Status      int8   `json:"status"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type OrderResponse struct {
	ID            int64  `json:"id"`
	OrderNo       string `json:"order_no"`
	UserID        int64  `json:"user_id"`
	EventID       int64  `json:"event_id"`
	SectionID     int64  `json:"section_id"`
	ReservationID int64  `json:"reservation_id"`
	Quantity      int    `json:"quantity"`
	UnitPrice     int64  `json:"unit_price"`
	TotalAmount   int64  `json:"total_amount"`
	Status        int8   `json:"status"`
	ExpiresAt     string `json:"expires_at"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type PaymentResponse struct {
	ID        int64   `json:"id"`
	OrderID   int64   `json:"order_id"`
	PaymentNo string  `json:"payment_no"`
	Method    string  `json:"method"`
	Amount    int64   `json:"amount"`
	Status    int8    `json:"status"`
	PaidAt    *string `json:"paid_at,omitempty"`
	FailedAt  *string `json:"failed_at,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func newReservationResponse(reservation *domain.Reservation) ReservationResponse {
	return ReservationResponse{
		ID:          reservation.ID,
		EventID:     reservation.EventID,
		SectionID:   reservation.SectionID,
		UserID:      reservation.UserID,
		Quantity:    reservation.Quantity,
		UnitPrice:   reservation.UnitPrice,
		TotalAmount: reservation.TotalAmount,
		Status:      int8(reservation.Status),
		ExpiresAt:   reservation.ExpiresAt.Format(time.RFC3339),
		CreatedAt:   reservation.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   reservation.UpdatedAt.Format(time.RFC3339),
	}
}

func newOrderResponse(order *domain.Order) OrderResponse {
	return OrderResponse{
		ID:            order.ID,
		OrderNo:       order.OrderNo,
		UserID:        order.UserID,
		EventID:       order.EventID,
		SectionID:     order.SectionID,
		ReservationID: order.ReservationID,
		Quantity:      order.Quantity,
		UnitPrice:     order.UnitPrice,
		TotalAmount:   order.TotalAmount,
		Status:        int8(order.Status),
		ExpiresAt:     order.ExpiresAt.Format(time.RFC3339),
		CreatedAt:     order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     order.UpdatedAt.Format(time.RFC3339),
	}
}

func newPaymentResponse(payment *domain.Payment) PaymentResponse {
	response := PaymentResponse{
		ID:        payment.ID,
		OrderID:   payment.OrderID,
		PaymentNo: payment.PaymentNo,
		Method:    payment.Method,
		Amount:    payment.Amount,
		Status:    int8(payment.Status),
		CreatedAt: payment.CreatedAt.Format(time.RFC3339),
		UpdatedAt: payment.UpdatedAt.Format(time.RFC3339),
	}

	if payment.PaidAt != nil {
		paidAt := payment.PaidAt.Format(time.RFC3339)
		response.PaidAt = &paidAt
	}

	if payment.FailedAt != nil {
		failedAt := payment.FailedAt.Format(time.RFC3339)
		response.FailedAt = &failedAt
	}

	return response
}
