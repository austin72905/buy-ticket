package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"buy-ticket/domain"
)

var (
	ErrEventNotFound       = errors.New("event not found")
	ErrSectionNotFound     = errors.New("section not found")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrOrderNotFound       = errors.New("order not found")
	ErrPaymentNotFound     = errors.New("payment not found")
)

type MemoryEventRepository struct {
	mu     sync.RWMutex
	events map[int64]*domain.Event
}

func NewMemoryEventRepository(events []*domain.Event) *MemoryEventRepository {
	repo := &MemoryEventRepository{
		events: make(map[int64]*domain.Event, len(events)),
	}

	for _, event := range events {
		cloned := *event
		repo.events[event.ID] = &cloned
	}

	return repo
}

func (r *MemoryEventRepository) FindByID(ctx context.Context, eventID int64) (*domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	event, ok := r.events[eventID]
	if !ok {
		return nil, ErrEventNotFound
	}

	cloned := *event
	return &cloned, nil
}

func (r *MemoryEventRepository) List(ctx context.Context) ([]domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := make([]domain.Event, 0, len(r.events))
	for _, event := range r.events {
		cloned := *event
		events = append(events, cloned)
	}

	return events, nil
}

type MemorySectionRepository struct {
	mu       sync.RWMutex
	sections map[int64]*domain.Section
}

func NewMemorySectionRepository(sections []*domain.Section) *MemorySectionRepository {
	repo := &MemorySectionRepository{
		sections: make(map[int64]*domain.Section, len(sections)),
	}

	for _, section := range sections {
		cloned := *section
		repo.sections[section.ID] = &cloned
	}

	return repo
}

func (r *MemorySectionRepository) FindByEventAndID(ctx context.Context, eventID, sectionID int64) (*domain.Section, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	section, ok := r.sections[sectionID]
	if !ok || section.EventID != eventID {
		return nil, ErrSectionNotFound
	}

	cloned := *section
	return &cloned, nil
}

func (r *MemorySectionRepository) ListByEventID(ctx context.Context, eventID int64) ([]domain.Section, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sections := make([]domain.Section, 0)
	for _, section := range r.sections {
		if section.EventID != eventID {
			continue
		}

		cloned := *section
		sections = append(sections, cloned)
	}

	if len(sections) == 0 {
		return nil, ErrSectionNotFound
	}

	return sections, nil
}

func (r *MemorySectionRepository) Save(ctx context.Context, section *domain.Section) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *section
	r.sections[section.ID] = &cloned
	return nil
}

type MemoryReservationRepository struct {
	mu           sync.RWMutex
	reservations map[int64]*domain.Reservation
	nextID       int64
}

func NewMemoryReservationRepository() *MemoryReservationRepository {
	return &MemoryReservationRepository{
		reservations: map[int64]*domain.Reservation{},
		nextID:       1,
	}
}

func (r *MemoryReservationRepository) FindByID(ctx context.Context, reservationID int64) (*domain.Reservation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reservation, ok := r.reservations[reservationID]
	if !ok {
		return nil, ErrReservationNotFound
	}

	cloned := *reservation
	return &cloned, nil
}

func (r *MemoryReservationRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Reservation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reservations := make([]domain.Reservation, 0)
	for _, reservation := range r.reservations {
		if reservation.UserID != userID {
			continue
		}

		cloned := *reservation
		reservations = append(reservations, cloned)
	}

	return reservations, nil
}

func (r *MemoryReservationRepository) Save(ctx context.Context, reservation *domain.Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *reservation
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		reservation.ID = cloned.ID
	}

	r.reservations[cloned.ID] = &cloned
	return nil
}

type MemoryOrderRepository struct {
	mu     sync.RWMutex
	orders map[int64]*domain.Order
	nextID int64
}

func NewMemoryOrderRepository() *MemoryOrderRepository {
	return &MemoryOrderRepository{
		orders: map[int64]*domain.Order{},
		nextID: 1,
	}
}

func (r *MemoryOrderRepository) FindByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}

	cloned := *order
	return &cloned, nil
}

func (r *MemoryOrderRepository) FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, order := range r.orders {
		if order.OrderNo != orderNo {
			continue
		}

		cloned := *order
		return &cloned, nil
	}

	return nil, ErrOrderNotFound
}

func (r *MemoryOrderRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orders := make([]domain.Order, 0)
	for _, order := range r.orders {
		if order.UserID != userID {
			continue
		}

		cloned := *order
		orders = append(orders, cloned)
	}

	return orders, nil
}

func (r *MemoryOrderRepository) ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	orders := make([]domain.Order, 0)
	for _, order := range r.orders {
		if order.Status != domain.OrderStatusPendingPayment || !order.IsExpired(now) {
			continue
		}

		cloned := *order
		orders = append(orders, cloned)
		if len(orders) >= limit {
			break
		}
	}

	return orders, nil
}

func (r *MemoryOrderRepository) Save(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *order
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		order.ID = cloned.ID
	}

	r.orders[cloned.ID] = &cloned
	return nil
}

type MemoryPaymentRepository struct {
	mu       sync.RWMutex
	payments map[int64]*domain.Payment
	nextID   int64
}

func NewMemoryPaymentRepository() *MemoryPaymentRepository {
	return &MemoryPaymentRepository{
		payments: map[int64]*domain.Payment{},
		nextID:   1,
	}
}

func (r *MemoryPaymentRepository) FindByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, payment := range r.payments {
		if payment.PaymentNo != paymentNo {
			continue
		}

		cloned := *payment
		return &cloned, nil
	}

	return nil, ErrPaymentNotFound
}

func (r *MemoryPaymentRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Payment, error) {
	return []domain.Payment{}, nil
}

func (r *MemoryPaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *payment
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		payment.ID = cloned.ID
	}

	r.payments[cloned.ID] = &cloned
	return nil
}

func SeedSampleData() ([]*domain.Event, []*domain.Section) {
	now := time.Now()

	sections := []*domain.Section{
		{
			ID:            1,
			EventID:       1,
			Name:          "A 區",
			Price:         2800,
			TotalQuantity: 100,
			PurchaseLimit: 4,
			Status:        domain.SectionStatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            2,
			EventID:       1,
			Name:          "B 區",
			Price:         1800,
			TotalQuantity: 150,
			PurchaseLimit: 4,
			Status:        domain.SectionStatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	events := []*domain.Event{
		{
			ID:          1,
			Name:        "Sample Concert",
			StartAt:     now.Add(30 * 24 * time.Hour),
			EndAt:       now.Add(30*24*time.Hour + 2*time.Hour),
			SaleStartAt: now.Add(-time.Hour),
			SaleEndAt:   now.Add(24 * time.Hour),
			Venue:       "Taipei Arena",
			Status:      domain.EventStatusOnSale,
			Sections:    []domain.Section{*sections[0], *sections[1]},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	return events, sections
}
