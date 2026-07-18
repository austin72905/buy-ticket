package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"buy-ticket/domain"
)

const OutboxEventPaymentSucceeded = "PAYMENT_SUCCEEDED"

type PaymentSucceededPayload struct {
	PaymentID     int64  `json:"payment_id"`
	PaymentNo     string `json:"payment_no"`
	OrderID       int64  `json:"order_id"`
	OrderNo       string `json:"order_no"`
	ReservationID int64  `json:"reservation_id"`
	UserID        int64  `json:"user_id"`
	EventID       int64  `json:"event_id"`
	SectionID     int64  `json:"section_id"`
	Quantity      int    `json:"quantity"`
	Amount        int64  `json:"amount"`
	Method        string `json:"method"`
	PaidAt        string `json:"paid_at"`
}

type PublishOutboxEventsInput struct {
	Now        time.Time
	Limit      int
	RetryAfter time.Duration
}

func newPaymentSucceededOutboxEvent(order *domain.Order, reservation *domain.Reservation, payment *domain.Payment, paidAt time.Time) (*domain.OutboxEvent, error) {
	payload, err := json.Marshal(PaymentSucceededPayload{
		PaymentID:     payment.ID,
		PaymentNo:     payment.PaymentNo,
		OrderID:       order.ID,
		OrderNo:       order.OrderNo,
		ReservationID: reservation.ID,
		UserID:        order.UserID,
		EventID:       order.EventID,
		SectionID:     order.SectionID,
		Quantity:      order.Quantity,
		Amount:        payment.Amount,
		Method:        payment.Method,
		PaidAt:        paidAt.Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}

	return &domain.OutboxEvent{
		EventID:       newOutboxEventID(),
		EventType:     OutboxEventPaymentSucceeded,
		AggregateType: "PAYMENT",
		AggregateID:   payment.ID,
		Payload:       payload,
		Status:        domain.OutboxEventStatusPending,
		MaxAttempts:   5,
		CreatedAt:     paidAt,
		UpdatedAt:     paidAt,
	}, nil
}

func (s *BookingService) PublishOutboxEvents(ctx context.Context, input PublishOutboxEventsInput) (int, error) {
	if s.OutboxRepo == nil {
		return 0, nil
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	retryAfter := input.RetryAfter
	if retryAfter <= 0 {
		retryAfter = 30 * time.Second
	}

	events, err := s.OutboxRepo.ListPending(ctx, now, limit)
	if err != nil {
		return 0, err
	}

	published := 0
	/*
		避免這種寫法，會拿到錯誤位置
		for _, event := range events {
		    repo.UpdatePublishState(&event)
		}

	*/
	for index := range events {
		event := events[index]
		if err := publishOutboxEvent(ctx, event); err != nil {
			nextAttemptAt := now.Add(retryAfter)
			event.MarkPublishFailed(err.Error(), &nextAttemptAt, now)
			if saveErr := s.OutboxRepo.UpdatePublishState(ctx, &event); saveErr != nil {
				return published, saveErr
			}
			continue
		}

		event.MarkPublished(now)
		if err := s.OutboxRepo.UpdatePublishState(ctx, &event); err != nil {
			return published, err
		}
		published++
	}

	return published, nil
}

// 之後可以改成真的用email 通知
func publishOutboxEvent(ctx context.Context, event domain.OutboxEvent) error {
	_ = ctx
	switch event.EventType {
	case OutboxEventPaymentSucceeded:
		log.Printf("outbox notification payment succeeded event_id=%s payload=%s", event.EventID, string(event.Payload))
		return nil
	default:
		return fmt.Errorf("unsupported outbox event type: %s", event.EventType)
	}
}

func newOutboxEventID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	return "evt_" + hex.EncodeToString(bytes[:])
}
