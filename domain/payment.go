package domain

import "time"

type PaymentStatus int8

const (
	PaymentStatusPending PaymentStatus = iota + 1
	PaymentStatusPaid
	PaymentStatusFailed
	PaymentStatusRefunded
)

type Payment struct {
	ID        int64
	OrderID   int64
	PaymentNo string
	Method    string
	Amount    int64
	Status    PaymentStatus
	PaidAt    *time.Time
	FailedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Payment) MarkPaid(now time.Time) bool {
	if p.Status != PaymentStatusPending {
		return false
	}

	p.Status = PaymentStatusPaid
	p.PaidAt = &now
	p.UpdatedAt = now
	return true
}

func (p *Payment) MarkFailed(now time.Time) bool {
	if p.Status != PaymentStatusPending {
		return false
	}

	p.Status = PaymentStatusFailed
	p.FailedAt = &now
	p.UpdatedAt = now
	return true
}
