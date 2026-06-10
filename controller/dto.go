package controller

import (
	"time"

	"buy-ticket/domain"
	"buy-ticket/service"
)

type ReserveTicketRequest struct {
	UserID        int64     `json:"user_id" example:"1"`
	EventID       int64     `json:"event_id" example:"1"`
	SectionID     int64     `json:"section_id" example:"1"`
	Quantity      int       `json:"quantity" example:"2"`
	HoldUntil     time.Time `json:"hold_until" example:"2026-06-10T19:00:00+08:00"`
	PurchaseToken string    `json:"purchase_token" example:"pt_01JXABCDEFG1234567890"`
}

type JoinQueueRequest struct {
	EventID    int64  `json:"event_id"`
	UserID     int64  `json:"user_id"`
	ClientID   string `json:"client_id"`
	RequestID  string `json:"request_id"`
	Channel    string `json:"channel"`
	AccessCode string `json:"access_code,omitempty"`
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

type EventResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Venue       string `json:"venue"`
	Status      int8   `json:"status"`
	StartAt     string `json:"start_at"`
	EndAt       string `json:"end_at"`
	SaleStartAt string `json:"sale_start_at"`
	SaleEndAt   string `json:"sale_end_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type SectionResponse struct {
	ID               int64  `json:"id"`
	EventID          int64  `json:"event_id"`
	Name             string `json:"name"`
	Price            int64  `json:"price"`
	TotalQuantity    int    `json:"total_quantity"`
	ReservedQuantity int    `json:"reserved_quantity"`
	SoldQuantity     int    `json:"sold_quantity"`
	PurchaseLimit    int    `json:"purchase_limit"`
	Status           int8   `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type SectionAvailabilityResponse struct {
	SectionID         int64  `json:"section_id"`
	Name              string `json:"name"`
	Price             int64  `json:"price"`
	AvailableQuantity int    `json:"available_quantity"`
	ReservedQuantity  int    `json:"reserved_quantity"`
	SoldQuantity      int    `json:"sold_quantity"`
	Status            int8   `json:"status"`
}

type SaleStatusResponse struct {
	EventID      int64  `json:"event_id"`
	EventStatus  int8   `json:"event_status"`
	IsOnSale     bool   `json:"is_on_sale"`
	QueueEnabled bool   `json:"queue_enabled"`
	CanJoinQueue bool   `json:"can_join_queue"`
	CanReserve   bool   `json:"can_reserve"`
	SaleStartAt  string `json:"sale_start_at"`
	SaleEndAt    string `json:"sale_end_at"`
	ServerTime   string `json:"server_time"`
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

func newEventResponse(event *domain.Event) EventResponse {
	return EventResponse{
		ID:          event.ID,
		Name:        event.Name,
		Venue:       event.Venue,
		Status:      int8(event.Status),
		StartAt:     event.StartAt.Format(time.RFC3339),
		EndAt:       event.EndAt.Format(time.RFC3339),
		SaleStartAt: event.SaleStartAt.Format(time.RFC3339),
		SaleEndAt:   event.SaleEndAt.Format(time.RFC3339),
		CreatedAt:   event.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   event.UpdatedAt.Format(time.RFC3339),
	}
}

func newSectionResponse(section domain.Section) SectionResponse {
	return SectionResponse{
		ID:               section.ID,
		EventID:          section.EventID,
		Name:             section.Name,
		Price:            section.Price,
		TotalQuantity:    section.TotalQuantity,
		ReservedQuantity: section.ReservedQuantity,
		SoldQuantity:     section.SoldQuantity,
		PurchaseLimit:    section.PurchaseLimit,
		Status:           int8(section.Status),
		CreatedAt:        section.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        section.UpdatedAt.Format(time.RFC3339),
	}
}

func newSectionAvailabilityResponse(availability service.SectionAvailability) SectionAvailabilityResponse {
	return SectionAvailabilityResponse{
		SectionID:         availability.Section.ID,
		Name:              availability.Section.Name,
		Price:             availability.Section.Price,
		AvailableQuantity: availability.Available,
		ReservedQuantity:  availability.Section.ReservedQuantity,
		SoldQuantity:      availability.Section.SoldQuantity,
		Status:            int8(availability.Section.Status),
	}
}

func newSaleStatusResponse(status *service.SaleStatus) SaleStatusResponse {
	return SaleStatusResponse{
		EventID:      status.EventID,
		EventStatus:  int8(status.EventStatus),
		IsOnSale:     status.IsOnSale,
		QueueEnabled: status.QueueEnabled,
		CanJoinQueue: status.CanJoinQueue,
		CanReserve:   status.CanReserve,
		SaleStartAt:  status.SaleStartAt.Format(time.RFC3339),
		SaleEndAt:    status.SaleEndAt.Format(time.RFC3339),
		ServerTime:   status.ServerTime.Format(time.RFC3339),
	}
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

type ErrorResponse struct {
	Error string `json:"error"`
}

type JoinQueueResponse struct {
	QueueToken             string  `json:"queue_token"`
	Status                 int8    `json:"status"`
	EventID                int64   `json:"event_id"`
	UserID                 int64   `json:"user_id"`
	QueuePosition          int64   `json:"queue_position"`
	AheadCount             int64   `json:"ahead_count"`
	EstimatedWaitSeconds   int64   `json:"estimated_wait_seconds"`
	PurchaseToken          *string `json:"purchase_token,omitempty"`
	PurchaseTokenExpiresAt *string `json:"purchase_token_expires_at,omitempty"`
	JoinedAt               string  `json:"joined_at"`
	ExpiredAt              string  `json:"expired_at"`
}

func newJoinQueueResponse(snapshot *service.QueueStatusSnapshot) JoinQueueResponse {
	response := JoinQueueResponse{
		QueueToken:           snapshot.QueueToken,
		Status:               int8(snapshot.Status),
		EventID:              snapshot.EventID,
		UserID:               snapshot.UserID,
		QueuePosition:        snapshot.QueuePosition,
		AheadCount:           snapshot.AheadCount,
		EstimatedWaitSeconds: snapshot.EstimatedWaitSeconds,
		JoinedAt:             snapshot.JoinedAt.Format(time.RFC3339),
		ExpiredAt:            snapshot.ExpiredAt.Format(time.RFC3339),
	}

	if snapshot.PurchaseToken != nil {
		response.PurchaseToken = snapshot.PurchaseToken
	}

	if snapshot.PurchaseTokenExpiresAt != nil {
		value := snapshot.PurchaseTokenExpiresAt.Format(time.RFC3339)
		response.PurchaseTokenExpiresAt = &value
	}

	return response
}

type QueueStatusResponse struct {
	QueueToken             string  `json:"queue_token"`
	Status                 int8    `json:"status"`
	EventID                int64   `json:"event_id"`
	UserID                 int64   `json:"user_id"`
	QueuePosition          int64   `json:"queue_position"`
	AheadCount             int64   `json:"ahead_count"`
	EstimatedWaitSeconds   int64   `json:"estimated_wait_seconds"`
	PurchaseToken          *string `json:"purchase_token,omitempty"`
	PurchaseTokenExpiresAt *string `json:"purchase_token_expires_at,omitempty"`
	JoinedAt               string  `json:"joined_at"`
	ExpiredAt              string  `json:"expired_at"`
	UpdatedAt              string  `json:"updated_at"`
}

func newQueueStatusResponse(snapshot *service.QueueStatusSnapshot) QueueStatusResponse {
	response := QueueStatusResponse{
		QueueToken:           snapshot.QueueToken,
		Status:               int8(snapshot.Status),
		EventID:              snapshot.EventID,
		UserID:               snapshot.UserID,
		QueuePosition:        snapshot.QueuePosition,
		AheadCount:           snapshot.AheadCount,
		EstimatedWaitSeconds: snapshot.EstimatedWaitSeconds,
		JoinedAt:             snapshot.JoinedAt.Format(time.RFC3339),
		ExpiredAt:            snapshot.ExpiredAt.Format(time.RFC3339),
		UpdatedAt:            snapshot.UpdatedAt.Format(time.RFC3339),
	}

	if snapshot.PurchaseToken != nil {
		response.PurchaseToken = snapshot.PurchaseToken
	}

	if snapshot.PurchaseTokenExpiresAt != nil {
		value := snapshot.PurchaseTokenExpiresAt.Format(time.RFC3339)
		response.PurchaseTokenExpiresAt = &value
	}

	return response
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
