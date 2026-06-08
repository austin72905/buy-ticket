package domain

import "time"

type ReservationStatus int8

const (
	ReservationStatusHolding ReservationStatus = iota + 1
	ReservationStatusConfirmed
	ReservationStatusExpired
	ReservationStatusCancelled
)

type Reservation struct {
	ID          int64
	EventID     int64
	SectionID   int64
	UserID      int64
	Quantity    int
	UnitPrice   int64
	TotalAmount int64
	Status      ReservationStatus
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r Reservation) IsExpired(now time.Time) bool {
	return !r.ExpiresAt.IsZero() && now.After(r.ExpiresAt)
}

func (r Reservation) IsActive(now time.Time) bool {
	return r.Status == ReservationStatusHolding && !r.IsExpired(now)
}

func (r *Reservation) Confirm(now time.Time) bool {
	if r.Status != ReservationStatusHolding || r.IsExpired(now) {
		return false
	}

	r.Status = ReservationStatusConfirmed
	r.UpdatedAt = now
	return true
}

func (r *Reservation) Expire(now time.Time) bool {
	if r.Status != ReservationStatusHolding {
		return false
	}

	r.Status = ReservationStatusExpired
	r.UpdatedAt = now
	return true
}

func (r *Reservation) Cancel(now time.Time) bool {
	if r.Status != ReservationStatusHolding {
		return false
	}

	r.Status = ReservationStatusCancelled
	r.UpdatedAt = now
	return true
}
