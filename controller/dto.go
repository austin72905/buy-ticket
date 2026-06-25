package controller

import (
	"time"

	"buy-ticket/domain"
	"buy-ticket/service"
)

type ReserveTicketRequest struct {
	EventID       int64     `json:"event_id" example:"1"`
	SectionID     int64     `json:"section_id" example:"1"`
	Quantity      int       `json:"quantity" example:"2"`
	HoldUntil     time.Time `json:"hold_until" example:"2026-06-10T19:00:00+08:00"`
	PurchaseToken string    `json:"purchase_token" example:"pt_01JXABCDEFG1234567890"`
}

type JoinQueueRequest struct {
	EventID    int64  `json:"event_id"`
	ClientID   string `json:"client_id"`
	RequestID  string `json:"request_id"`
	Channel    string `json:"channel"`
	AccessCode string `json:"access_code,omitempty"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateOrderRequest struct {
	ReservationID int64  `json:"reservation_id"`
	OrderNo       string `json:"order_no"`
	PurchaseToken string `json:"purchase_token"`
}

type PayOrderRequest struct {
	OrderID int64  `json:"order_id"`
	Method  string `json:"method"`
}

type StartPaymentRequest struct {
	OrderID  int64  `json:"order_id"`
	Method   string `json:"method"`
	Provider string `json:"provider,omitempty"`
}

type ECPayCallbackRequest struct {
	MerchantID           string `form:"MerchantID" json:"MerchantID"`
	MerchantTradeNo      string `form:"MerchantTradeNo" json:"MerchantTradeNo"`
	RtnCode              string `form:"RtnCode" json:"RtnCode"`
	RtnMsg               string `form:"RtnMsg" json:"RtnMsg"`
	TradeNo              string `form:"TradeNo" json:"TradeNo"`
	TradeAmt             string `form:"TradeAmt" json:"TradeAmt"`
	PaymentDate          string `form:"PaymentDate" json:"PaymentDate"`
	PaymentType          string `form:"PaymentType" json:"PaymentType"`
	PaymentTypeChargeFee string `form:"PaymentTypeChargeFee" json:"PaymentTypeChargeFee"`
	TradeDate            string `form:"TradeDate" json:"TradeDate"`
	SimulatePaid         string `form:"SimulatePaid" json:"SimulatePaid"`
	CustomField1         string `form:"CustomField1" json:"CustomField1"`
	CustomField2         string `form:"CustomField2" json:"CustomField2"`
	CustomField3         string `form:"CustomField3" json:"CustomField3"`
	CustomField4         string `form:"CustomField4" json:"CustomField4"`
	CheckMacValue        string `form:"CheckMacValue" json:"CheckMacValue"`
	ReturnStatus         string `form:"ReturnStatus" json:"ReturnStatus"`
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

type UserResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
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

func newUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
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

type PaymentAttemptResponse struct {
	ID              int64   `json:"id"`
	OrderID         int64   `json:"order_id"`
	IdempotencyKey  *string `json:"idempotency_key,omitempty"`
	Provider        string  `json:"provider"`
	MerchantTradeNo string  `json:"merchant_trade_no"`
	ProviderTradeNo *string `json:"provider_trade_no,omitempty"`
	Method          string  `json:"method"`
	Amount          int64   `json:"amount"`
	Status          int8    `json:"status"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	SucceededAt     *string `json:"succeeded_at,omitempty"`
	FailedAt        *string `json:"failed_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
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

func newPaymentAttemptResponse(attempt *domain.PaymentAttempt) PaymentAttemptResponse {
	response := PaymentAttemptResponse{
		ID:              attempt.ID,
		OrderID:         attempt.OrderID,
		IdempotencyKey:  attempt.IdempotencyKey,
		Provider:        attempt.Provider,
		MerchantTradeNo: attempt.MerchantTradeNo,
		ProviderTradeNo: attempt.ProviderTradeNo,
		Method:          attempt.Method,
		Amount:          attempt.Amount,
		Status:          int8(attempt.Status),
		CreatedAt:       attempt.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       attempt.UpdatedAt.Format(time.RFC3339),
	}

	if attempt.ExpiresAt != nil {
		expiresAt := attempt.ExpiresAt.Format(time.RFC3339)
		response.ExpiresAt = &expiresAt
	}
	if attempt.SucceededAt != nil {
		succeededAt := attempt.SucceededAt.Format(time.RFC3339)
		response.SucceededAt = &succeededAt
	}
	if attempt.FailedAt != nil {
		failedAt := attempt.FailedAt.Format(time.RFC3339)
		response.FailedAt = &failedAt
	}

	return response
}
