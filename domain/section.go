package domain

import "time"

type SectionStatus int8

const (
	SectionStatusActive SectionStatus = iota + 1
	SectionStatusInactive
	SectionStatusSoldOut
)

type Section struct {
	ID               int64
	EventID          int64
	Name             string
	Price            int64
	TotalQuantity    int
	ReservedQuantity int
	SoldQuantity     int
	PurchaseLimit    int
	Status           SectionStatus
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s Section) AvailableQuantity() int {
	return s.TotalQuantity - s.ReservedQuantity - s.SoldQuantity
}

func (s Section) CanReserve(quantity int) bool {
	if s.Status != SectionStatusActive {
		return false
	}

	if quantity <= 0 {
		return false
	}

	if s.PurchaseLimit > 0 && quantity > s.PurchaseLimit {
		return false
	}

	return s.AvailableQuantity() >= quantity
}

func (s *Section) Reserve(quantity int) bool {
	if !s.CanReserve(quantity) {
		return false
	}

	s.ReservedQuantity += quantity
	return true
}

func (s *Section) Release(quantity int) bool {
	if quantity <= 0 || quantity > s.ReservedQuantity {
		return false
	}

	s.ReservedQuantity -= quantity
	if s.Status == SectionStatusSoldOut && s.AvailableQuantity() > 0 {
		s.Status = SectionStatusActive
	}

	return true
}

func (s *Section) ConfirmSale(quantity int) bool {
	if quantity <= 0 || quantity > s.ReservedQuantity {
		return false
	}

	s.ReservedQuantity -= quantity
	s.SoldQuantity += quantity
	if s.AvailableQuantity() == 0 {
		s.Status = SectionStatusSoldOut
	}

	return true
}
