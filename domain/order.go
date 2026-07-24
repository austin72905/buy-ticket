package domain

import "time"

type OrderStatus int8

const (
	OrderStatusPendingPayment OrderStatus = iota + 1
	OrderStatusPaid
	OrderStatusExpired
	OrderStatusCancelled
)

type Order struct {
	ID            int64
	OrderNo       string
	ReservationNo string
	UserID        int64
	UserName      string
	EventID       int64
	EventName     string
	SectionID     int64
	SectionName   string
	ReservationID int64
	Quantity      int
	UnitPrice     int64
	TotalAmount   int64
	Status        OrderStatus
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (o Order) IsExpired(now time.Time) bool {
	return !o.ExpiresAt.IsZero() && now.After(o.ExpiresAt)
}

func (o Order) CanPay(now time.Time) bool {
	return o.Status == OrderStatusPendingPayment && !o.IsExpired(now)
}

func (o *Order) MarkPaid(now time.Time) bool {
	if !o.CanPay(now) {
		return false
	}

	o.Status = OrderStatusPaid
	o.UpdatedAt = now
	return true
}

func (o *Order) Expire(now time.Time) bool {
	if o.Status != OrderStatusPendingPayment {
		return false
	}

	o.Status = OrderStatusExpired
	o.UpdatedAt = now
	return true
}

func (o *Order) Cancel(now time.Time) bool {
	if o.Status == OrderStatusPaid {
		return false
	}

	o.Status = OrderStatusCancelled
	o.UpdatedAt = now
	return true
}
