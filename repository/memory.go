package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"buy-ticket/domain"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEventNotFound       = errors.New("event not found")
	ErrSectionNotFound     = errors.New("section not found")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrOrderNotFound       = errors.New("order not found")
	ErrPaymentNotFound     = errors.New("payment not found")
)

type MemoryUserRepository struct {
	mu         sync.RWMutex
	users      map[int64]*domain.User
	emailIndex map[string]int64
	nextID     int64
}

type MemoryAdminUserRepository struct {
	mu         sync.RWMutex
	adminUsers map[int64]*domain.AdminUser
	emailIndex map[string]int64
	nextID     int64
}

func NewMemoryAdminUserRepository(adminUsers []*domain.AdminUser) *MemoryAdminUserRepository {
	repo := &MemoryAdminUserRepository{
		adminUsers: make(map[int64]*domain.AdminUser, len(adminUsers)),
		emailIndex: make(map[string]int64, len(adminUsers)),
		nextID:     1,
	}

	var maxID int64
	for _, adminUser := range adminUsers {
		cloned := *adminUser
		repo.adminUsers[adminUser.ID] = &cloned
		repo.emailIndex[adminUser.Email] = adminUser.ID
		if adminUser.ID > maxID {
			maxID = adminUser.ID
		}
	}

	repo.nextID = maxID + 1
	return repo
}

func (r *MemoryAdminUserRepository) FindByID(ctx context.Context, adminUserID int64) (*domain.AdminUser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adminUser, ok := r.adminUsers[adminUserID]
	if !ok {
		return nil, ErrAdminUserNotFound
	}

	cloned := *adminUser
	return &cloned, nil
}

func (r *MemoryAdminUserRepository) FindByEmail(ctx context.Context, email string) (*domain.AdminUser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adminUserID, ok := r.emailIndex[email]
	if !ok {
		return nil, ErrAdminUserNotFound
	}

	adminUser, exists := r.adminUsers[adminUserID]
	if !exists {
		return nil, ErrAdminUserNotFound
	}

	cloned := *adminUser
	return &cloned, nil
}

func (r *MemoryAdminUserRepository) List(ctx context.Context) ([]domain.AdminUser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adminUsers := make([]domain.AdminUser, 0, len(r.adminUsers))
	for _, adminUser := range r.adminUsers {
		cloned := *adminUser
		adminUsers = append(adminUsers, cloned)
	}

	sort.SliceStable(adminUsers, func(i, j int) bool {
		return adminUsers[i].ID < adminUsers[j].ID
	})

	return adminUsers, nil
}

func (r *MemoryAdminUserRepository) Save(ctx context.Context, adminUser *domain.AdminUser) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *adminUser
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		adminUser.ID = cloned.ID
	}

	r.adminUsers[cloned.ID] = &cloned
	r.emailIndex[cloned.Email] = cloned.ID
	return nil
}

type MemoryOrganizerRepository struct {
	mu         sync.RWMutex
	organizers map[int64]*domain.Organizer
	nextID     int64
}

func NewMemoryOrganizerRepository(organizers []*domain.Organizer) *MemoryOrganizerRepository {
	repo := &MemoryOrganizerRepository{
		organizers: make(map[int64]*domain.Organizer, len(organizers)),
		nextID:     1,
	}

	var maxID int64
	for _, organizer := range organizers {
		cloned := *organizer
		repo.organizers[organizer.ID] = &cloned
		if organizer.ID > maxID {
			maxID = organizer.ID
		}
	}

	repo.nextID = maxID + 1
	return repo
}

func (r *MemoryOrganizerRepository) FindByID(ctx context.Context, organizerID int64) (*domain.Organizer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	organizer, ok := r.organizers[organizerID]
	if !ok {
		return nil, ErrOrganizerNotFound
	}

	cloned := *organizer
	return &cloned, nil
}

func (r *MemoryOrganizerRepository) List(ctx context.Context) ([]domain.Organizer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	organizers := make([]domain.Organizer, 0, len(r.organizers))
	for _, organizer := range r.organizers {
		cloned := *organizer
		organizers = append(organizers, cloned)
	}

	sort.SliceStable(organizers, func(i, j int) bool {
		return organizers[i].ID < organizers[j].ID
	})

	return organizers, nil
}

func (r *MemoryOrganizerRepository) Save(ctx context.Context, organizer *domain.Organizer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *organizer
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		organizer.ID = cloned.ID
	}

	r.organizers[cloned.ID] = &cloned
	return nil
}

type MemoryAdminAuditLogRepository struct {
	mu     sync.RWMutex
	logs   map[int64]*domain.AdminAuditLog
	nextID int64
}

func NewMemoryAdminAuditLogRepository() *MemoryAdminAuditLogRepository {
	return &MemoryAdminAuditLogRepository{
		logs:   map[int64]*domain.AdminAuditLog{},
		nextID: 1,
	}
}

func (r *MemoryAdminAuditLogRepository) Create(ctx context.Context, log *domain.AdminAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *log
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		log.ID = cloned.ID
	}

	r.logs[cloned.ID] = &cloned
	return nil
}

func (r *MemoryAdminAuditLogRepository) List(ctx context.Context, filter domain.AdminAuditLogListFilter) ([]domain.AdminAuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	logs := make([]domain.AdminAuditLog, 0)
	for _, log := range r.logs {
		if filter.AdminUserID != 0 && log.AdminUserID != filter.AdminUserID {
			continue
		}
		if filter.Cursor.CreatedAt != nil {
			if log.CreatedAt.After(*filter.Cursor.CreatedAt) || log.CreatedAt.Equal(*filter.Cursor.CreatedAt) && log.ID >= filter.Cursor.ID {
				continue
			}
		}

		cloned := *log
		logs = append(logs, cloned)
	}

	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].CreatedAt.Equal(logs[j].CreatedAt) {
			return logs[i].ID > logs[j].ID
		}
		return logs[i].CreatedAt.After(logs[j].CreatedAt)
	})

	if len(logs) > limit {
		logs = logs[:limit]
	}

	return logs, nil
}

func NewMemoryUserRepository(users []*domain.User) *MemoryUserRepository {
	repo := &MemoryUserRepository{
		users:      make(map[int64]*domain.User, len(users)),
		emailIndex: make(map[string]int64, len(users)),
		nextID:     1,
	}

	var maxID int64
	for _, user := range users {
		cloned := *user
		repo.users[user.ID] = &cloned
		repo.emailIndex[user.Email] = user.ID
		if user.ID > maxID {
			maxID = user.ID
		}
	}

	repo.nextID = maxID + 1
	return repo
}

func (r *MemoryUserRepository) FindByID(ctx context.Context, userID int64) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}

	cloned := *user
	return &cloned, nil
}

func (r *MemoryUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.emailIndex[email]
	if !ok {
		return nil, ErrUserNotFound
	}

	user, exists := r.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	cloned := *user
	return &cloned, nil
}

func (r *MemoryUserRepository) Save(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *user
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		user.ID = cloned.ID
	}

	r.users[cloned.ID] = &cloned
	r.emailIndex[cloned.Email] = cloned.ID
	return nil
}

type MemoryEventRepository struct {
	mu     sync.RWMutex
	events map[int64]*domain.Event
	nextID int64
}

func NewMemoryEventRepository(events []*domain.Event) *MemoryEventRepository {
	repo := &MemoryEventRepository{
		events: make(map[int64]*domain.Event, len(events)),
		nextID: 1,
	}

	var maxID int64
	for _, event := range events {
		cloned := *event
		repo.events[event.ID] = &cloned
		if event.ID > maxID {
			maxID = event.ID
		}
	}

	repo.nextID = maxID + 1
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

func (r *MemoryEventRepository) ListAdminEvents(ctx context.Context, organizerID *int64) ([]domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := make([]domain.Event, 0, len(r.events))
	for _, event := range r.events {
		if organizerID != nil && event.OrganizerID != *organizerID {
			continue
		}

		cloned := *event
		events = append(events, cloned)
	}

	return events, nil
}

func (r *MemoryEventRepository) FindAdminEventByID(ctx context.Context, eventID int64, organizerID *int64) (*domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	event, ok := r.events[eventID]
	if !ok {
		return nil, ErrEventNotFound
	}
	if organizerID != nil && event.OrganizerID != *organizerID {
		return nil, ErrEventNotFound
	}

	cloned := *event
	return &cloned, nil
}

func (r *MemoryEventRepository) CreateAdminEvent(ctx context.Context, event *domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *event
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		event.ID = cloned.ID
	}
	if cloned.CreatedAt.IsZero() {
		cloned.CreatedAt = time.Now()
	}
	if cloned.UpdatedAt.IsZero() {
		cloned.UpdatedAt = cloned.CreatedAt
	}

	r.events[cloned.ID] = &cloned
	*event = cloned
	return nil
}

func (r *MemoryEventRepository) ListAdminEventSections(ctx context.Context, eventID int64, organizerID *int64) ([]domain.Section, error) {
	event, err := r.FindAdminEventByID(ctx, eventID, organizerID)
	if err != nil {
		return nil, err
	}

	sections := make([]domain.Section, 0, len(event.Sections))
	for _, section := range event.Sections {
		cloned := section
		sections = append(sections, cloned)
	}

	return sections, nil
}

func (r *MemoryEventRepository) CreateAdminEventSection(ctx context.Context, eventID int64, organizerID *int64, section *domain.Section) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	event, ok := r.events[eventID]
	if !ok {
		return ErrEventNotFound
	}
	if organizerID != nil && event.OrganizerID != *organizerID {
		return ErrEventNotFound
	}

	var maxID int64
	for _, existingEvent := range r.events {
		for _, existingSection := range existingEvent.Sections {
			if existingSection.ID > maxID {
				maxID = existingSection.ID
			}
		}
	}

	cloned := *section
	cloned.EventID = eventID
	if cloned.ID == 0 {
		cloned.ID = maxID + 1
		section.ID = cloned.ID
	}
	if cloned.CreatedAt.IsZero() {
		cloned.CreatedAt = time.Now()
	}
	if cloned.UpdatedAt.IsZero() {
		cloned.UpdatedAt = cloned.CreatedAt
	}

	event.Sections = append(event.Sections, cloned)
	*section = cloned
	return nil
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

func (r *MemorySectionRepository) ReserveInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	section, ok := r.sections[sectionID]
	if !ok || section.EventID != eventID {
		return nil, ErrSectionNotFound
	}
	if !section.Reserve(quantity) {
		return nil, ErrSectionNotFound
	}

	section.UpdatedAt = now
	cloned := *section
	return &cloned, nil
}

func (r *MemorySectionRepository) ReleaseInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	section, ok := r.sections[sectionID]
	if !ok || section.EventID != eventID {
		return nil, ErrSectionNotFound
	}
	if !section.Release(quantity) {
		return nil, ErrSectionNotFound
	}

	section.UpdatedAt = now
	cloned := *section
	return &cloned, nil
}

func (r *MemorySectionRepository) ConfirmSale(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	section, ok := r.sections[sectionID]
	if !ok || section.EventID != eventID {
		return nil, ErrSectionNotFound
	}
	if !section.ConfirmSale(quantity) {
		return nil, ErrSectionNotFound
	}

	section.UpdatedAt = now
	cloned := *section
	return &cloned, nil
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

func (r *MemoryReservationRepository) FindActiveByUserAndEvent(ctx context.Context, userID, eventID int64, now time.Time) (*domain.Reservation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, reservation := range r.reservations {
		if reservation.UserID != userID || reservation.EventID != eventID || !reservation.IsActive(now) {
			continue
		}

		cloned := *reservation
		return &cloned, nil
	}

	return nil, ErrReservationNotFound
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

func (r *MemoryOrderRepository) ListAdminOrders(ctx context.Context, filter domain.AdminOrderListFilter) ([]domain.AdminOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	orders := make([]domain.AdminOrder, 0)
	for _, order := range r.orders {
		if filter.EventID != 0 && order.EventID != filter.EventID {
			continue
		}
		if filter.UserID != 0 && order.UserID != filter.UserID {
			continue
		}
		if filter.Status != 0 && order.Status != filter.Status {
			continue
		}
		if filter.Cursor.CreatedAt != nil {
			if order.CreatedAt.After(*filter.Cursor.CreatedAt) || order.CreatedAt.Equal(*filter.Cursor.CreatedAt) && order.ID >= filter.Cursor.ID {
				continue
			}
		}

		orders = append(orders, domain.AdminOrder{
			ID:            order.ID,
			OrderNo:       order.OrderNo,
			ReservationID: order.ReservationID,
			EventID:       order.EventID,
			SectionID:     order.SectionID,
			UserID:        order.UserID,
			UserName:      "",
			UserEmail:     "",
			Quantity:      order.Quantity,
			UnitPrice:     order.UnitPrice,
			TotalAmount:   order.TotalAmount,
			Status:        order.Status,
			ExpiresAt:     order.ExpiresAt,
			CreatedAt:     order.CreatedAt,
			UpdatedAt:     order.UpdatedAt,
		})
	}

	sort.SliceStable(orders, func(i, j int) bool {
		if orders[i].CreatedAt.Equal(orders[j].CreatedAt) {
			return orders[i].ID > orders[j].ID
		}
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})

	if len(orders) > limit {
		orders = orders[:limit]
	}

	return orders, nil
}

func (r *MemoryOrderRepository) FindAdminOrderSensitiveByID(ctx context.Context, orderID int64) (*domain.AdminOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}

	return &domain.AdminOrder{
		ID:            order.ID,
		OrderNo:       order.OrderNo,
		ReservationID: order.ReservationID,
		EventID:       order.EventID,
		SectionID:     order.SectionID,
		UserID:        order.UserID,
		UserName:      "",
		UserEmail:     "",
		Quantity:      order.Quantity,
		UnitPrice:     order.UnitPrice,
		TotalAmount:   order.TotalAmount,
		Status:        order.Status,
		ExpiresAt:     order.ExpiresAt,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
	}, nil
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

type MemoryPaymentAttemptRepository struct {
	mu       sync.RWMutex
	attempts map[int64]*domain.PaymentAttempt
	nextID   int64
}

func NewMemoryPaymentAttemptRepository(attempts []*domain.PaymentAttempt) *MemoryPaymentAttemptRepository {
	repo := &MemoryPaymentAttemptRepository{
		attempts: map[int64]*domain.PaymentAttempt{},
		nextID:   1,
	}

	var maxID int64
	for _, attempt := range attempts {
		cloned := *attempt
		repo.attempts[cloned.ID] = &cloned
		if cloned.ID > maxID {
			maxID = cloned.ID
		}
	}
	repo.nextID = maxID + 1

	return repo
}

func (r *MemoryPaymentAttemptRepository) FindByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (*domain.PaymentAttempt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, attempt := range r.attempts {
		if attempt.MerchantTradeNo != merchantTradeNo {
			continue
		}

		cloned := *attempt
		return &cloned, nil
	}

	return nil, ErrPaymentAttemptNotFound
}

func (r *MemoryPaymentAttemptRepository) FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.PaymentAttempt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, attempt := range r.attempts {
		if attempt.IdempotencyKey == nil || *attempt.IdempotencyKey != idempotencyKey {
			continue
		}

		cloned := *attempt
		return &cloned, nil
	}

	return nil, ErrPaymentAttemptNotFound
}

func (r *MemoryPaymentAttemptRepository) ListByOrderID(ctx context.Context, orderID int64) ([]domain.PaymentAttempt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	attempts := make([]domain.PaymentAttempt, 0)
	for _, attempt := range r.attempts {
		if attempt.OrderID == orderID {
			attempts = append(attempts, *attempt)
		}
	}

	return attempts, nil
}

func (r *MemoryPaymentAttemptRepository) Save(ctx context.Context, attempt *domain.PaymentAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *attempt
	if cloned.ID == 0 {
		cloned.ID = r.nextID
		r.nextID++
		attempt.ID = cloned.ID
	}

	r.attempts[cloned.ID] = &cloned
	return nil
}

func SeedSampleData() ([]*domain.User, []*domain.AdminUser, []*domain.Organizer, []*domain.Event, []*domain.Section) {
	now := time.Now()

	users := []*domain.User{
		{
			ID:           1,
			Name:         "Austin Lin",
			Email:        "austin@example.com",
			PasswordHash: "PENDING_RESET",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	adminUsers := []*domain.AdminUser{
		{
			ID:           1,
			Name:         "Super Admin",
			Email:        "admin@example.com",
			PasswordHash: "$2a$10$C98Lze8JLKbSUnfmKjJ9VeKsmGNw4q71gUA53CSVEJHrpYJcGQG3G",
			Role:         domain.AdminRoleSuperAdmin,
			Status:       domain.AdminUserStatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	organizers := []*domain.Organizer{
		{
			ID:        1,
			Name:      "Default Organizer",
			Status:    domain.OrganizerStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

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
			OrganizerID: 1,
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

	return users, adminUsers, organizers, events, sections
}
