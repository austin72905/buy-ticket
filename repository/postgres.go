package repository

import (
	"context"
	"fmt"
	"time"

	"buy-ticket/db/sqlc"
	"buy-ticket/domain"

	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresEventRepository struct {
	queries *db.Queries
}

type PostgresUserRepository struct {
	queries *db.Queries
}

type PostgresSectionRepository struct {
	queries *db.Queries
}

type PostgresReservationRepository struct {
	queries *db.Queries
}

type PostgresOrderRepository struct {
	queries *db.Queries
}

type PostgresPaymentRepository struct {
	queries *db.Queries
}

func NewPostgresEventRepository(queries *db.Queries) *PostgresEventRepository {
	return &PostgresEventRepository{queries: queries}
}

func NewPostgresUserRepository(queries *db.Queries) *PostgresUserRepository {
	return &PostgresUserRepository{queries: queries}
}

func NewPostgresSectionRepository(queries *db.Queries) *PostgresSectionRepository {
	return &PostgresSectionRepository{queries: queries}
}

func NewPostgresReservationRepository(queries *db.Queries) *PostgresReservationRepository {
	return &PostgresReservationRepository{queries: queries}
}

func NewPostgresOrderRepository(queries *db.Queries) *PostgresOrderRepository {
	return &PostgresOrderRepository{queries: queries}
}

func NewPostgresPaymentRepository(queries *db.Queries) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{queries: queries}
}

func (r *PostgresEventRepository) FindByID(ctx context.Context, eventID int64) (*domain.Event, error) {
	record, err := r.queries.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	return toDomainEvent(record), nil
}

func (r *PostgresEventRepository) List(ctx context.Context) ([]domain.Event, error) {
	records, err := r.queries.ListEvents(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]domain.Event, 0, len(records))
	for _, record := range records {
		events = append(events, *toDomainEvent(record))
	}

	return events, nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, userID int64) (*domain.User, error) {
	record, err := r.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toDomainUserFromGetUserByID(record), nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	record, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return toDomainUserFromGetUserByEmail(record), nil
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	if user.ID == 0 {
		record, err := r.queries.CreateUser(ctx, db.CreateUserParams{
			Name:         user.Name,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
		})
		if err != nil {
			return err
		}

		*user = *toDomainUserFromCreateUser(record)
		return nil
	}

	return r.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           user.ID,
		PasswordHash: user.PasswordHash,
		UpdatedAt:    toPgTimestamp(user.UpdatedAt),
	})
}

func (r *PostgresSectionRepository) FindByEventAndID(ctx context.Context, eventID, sectionID int64) (*domain.Section, error) {
	record, err := r.queries.GetSectionByEventAndID(ctx, db.GetSectionByEventAndIDParams{
		EventID: eventID,
		ID:      sectionID,
	})
	if err != nil {
		return nil, err
	}

	return toDomainSection(record), nil
}

func (r *PostgresSectionRepository) ListByEventID(ctx context.Context, eventID int64) ([]domain.Section, error) {
	records, err := r.queries.ListSectionsByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	sections := make([]domain.Section, 0, len(records))
	for _, record := range records {
		sections = append(sections, *toDomainSection(record))
	}

	return sections, nil
}

func (r *PostgresSectionRepository) Save(ctx context.Context, section *domain.Section) error {
	if section.ID == 0 {
		event, err := r.queries.GetEventByID(ctx, section.EventID)
		if err != nil {
			return err
		}

		record, err := r.queries.CreateSection(ctx, db.CreateSectionParams{
			EventID:          section.EventID,
			EventName:        event.Name,
			SectionName:      section.Name,
			Price:            section.Price,
			TotalQuantity:    int32(section.TotalQuantity),
			ReservedQuantity: int32(section.ReservedQuantity),
			SoldQuantity:     int32(section.SoldQuantity),
			PurchaseLimit:    int32(section.PurchaseLimit),
			Status:           int16(section.Status),
		})
		if err != nil {
			return err
		}

		*section = *toDomainSection(record)
		return nil
	}

	return r.queries.UpdateSectionInventory(ctx, db.UpdateSectionInventoryParams{
		ID:               section.ID,
		ReservedQuantity: int32(section.ReservedQuantity),
		SoldQuantity:     int32(section.SoldQuantity),
		Status:           int16(section.Status),
		UpdatedAt:        toPgTimestamp(section.UpdatedAt),
	})
}

func (r *PostgresReservationRepository) FindByID(ctx context.Context, reservationID int64) (*domain.Reservation, error) {
	record, err := r.queries.GetReservationByID(ctx, reservationID)
	if err != nil {
		return nil, err
	}

	return toDomainReservation(record), nil
}

func (r *PostgresReservationRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Reservation, error) {
	records, err := r.queries.ListReservationsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	reservations := make([]domain.Reservation, 0, len(records))
	for _, record := range records {
		reservations = append(reservations, *toDomainReservation(record))
	}

	return reservations, nil
}

func (r *PostgresReservationRepository) Save(ctx context.Context, reservation *domain.Reservation) error {
	if reservation.ID == 0 {
		event, err := r.queries.GetEventByID(ctx, reservation.EventID)
		if err != nil {
			return err
		}

		section, err := r.queries.GetSectionByEventAndID(ctx, db.GetSectionByEventAndIDParams{
			EventID: reservation.EventID,
			ID:      reservation.SectionID,
		})
		if err != nil {
			return err
		}

		record, err := r.queries.CreateReservation(ctx, db.CreateReservationParams{
			ReservationNo: buildReservationNo(reservation.UserID),
			EventID:       reservation.EventID,
			EventName:     event.Name,
			SectionID:     reservation.SectionID,
			SectionName:   section.SectionName,
			UserID:        reservation.UserID,
			UserName:      "",
			Quantity:      int32(reservation.Quantity),
			UnitPrice:     reservation.UnitPrice,
			TotalAmount:   reservation.TotalAmount,
			Status:        int16(reservation.Status),
			ExpiresAt:     toPgTimestamp(reservation.ExpiresAt),
			CreatedAt:     toPgTimestamp(reservation.CreatedAt),
			UpdatedAt:     toPgTimestamp(reservation.UpdatedAt),
		})
		if err != nil {
			return err
		}

		*reservation = *toDomainReservation(record)
		return nil
	}

	return r.queries.UpdateReservationStatus(ctx, db.UpdateReservationStatusParams{
		ID:        reservation.ID,
		Status:    int16(reservation.Status),
		UpdatedAt: toPgTimestamp(reservation.UpdatedAt),
	})
}

func (r *PostgresOrderRepository) FindByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	record, err := r.queries.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return toDomainOrder(record), nil
}

func (r *PostgresOrderRepository) FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	record, err := r.queries.GetOrderByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}

	return toDomainOrder(record), nil
}

func (r *PostgresOrderRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	records, err := r.queries.ListOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	orders := make([]domain.Order, 0, len(records))
	for _, record := range records {
		orders = append(orders, *toDomainOrder(record))
	}

	return orders, nil
}

func (r *PostgresOrderRepository) ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]domain.Order, error) {
	if limit <= 0 {
		limit = 100
	}

	records, err := r.queries.ListExpiredPendingOrders(ctx, db.ListExpiredPendingOrdersParams{
		ExpiresAt: toPgTimestamp(now),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}

	orders := make([]domain.Order, 0, len(records))
	for _, record := range records {
		orders = append(orders, *toDomainOrder(record))
	}

	return orders, nil
}

func (r *PostgresOrderRepository) Save(ctx context.Context, order *domain.Order) error {
	if order.ID == 0 {
		reservation, err := r.queries.GetReservationByID(ctx, order.ReservationID)
		if err != nil {
			return err
		}

		record, err := r.queries.CreateOrder(ctx, db.CreateOrderParams{
			OrderNo:       order.OrderNo,
			ReservationID: order.ReservationID,
			ReservationNo: reservation.ReservationNo,
			EventID:       order.EventID,
			EventName:     reservation.EventName,
			SectionID:     order.SectionID,
			SectionName:   reservation.SectionName,
			UserID:        order.UserID,
			UserName:      reservation.UserName,
			Quantity:      int32(order.Quantity),
			UnitPrice:     order.UnitPrice,
			TotalAmount:   order.TotalAmount,
			Status:        int16(order.Status),
			ExpiresAt:     toPgTimestamp(order.ExpiresAt),
			PaidAt:        nullablePgTimestamp(orderPaidAt(order)),
			CreatedAt:     toPgTimestamp(order.CreatedAt),
			UpdatedAt:     toPgTimestamp(order.UpdatedAt),
		})
		if err != nil {
			return err
		}

		*order = *toDomainOrder(record)
		return nil
	}

	return r.queries.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:        order.ID,
		Status:    int16(order.Status),
		PaidAt:    nullablePgTimestamp(orderPaidAt(order)),
		UpdatedAt: toPgTimestamp(order.UpdatedAt),
	})
}

func (r *PostgresPaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	if payment.ID == 0 {
		order, err := r.queries.GetOrderByID(ctx, payment.OrderID)
		if err != nil {
			return err
		}

		record, err := r.queries.CreatePayment(ctx, db.CreatePaymentParams{
			PaymentNo:     payment.PaymentNo,
			OrderID:       payment.OrderID,
			OrderNo:       order.OrderNo,
			ReservationID: order.ReservationID,
			EventID:       order.EventID,
			EventName:     order.EventName,
			UserID:        order.UserID,
			UserName:      order.UserName,
			Method:        payment.Method,
			Amount:        payment.Amount,
			Status:        int16(payment.Status),
			PaidAt:        nullablePgTimestamp(payment.PaidAt),
			FailedAt:      nullablePgTimestamp(payment.FailedAt),
			CreatedAt:     toPgTimestamp(payment.CreatedAt),
			UpdatedAt:     toPgTimestamp(payment.UpdatedAt),
		})
		if err != nil {
			return err
		}

		*payment = *toDomainPayment(record)
		return nil
	}

	return r.queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:        payment.ID,
		Status:    int16(payment.Status),
		PaidAt:    nullablePgTimestamp(payment.PaidAt),
		FailedAt:  nullablePgTimestamp(payment.FailedAt),
		UpdatedAt: toPgTimestamp(payment.UpdatedAt),
	})
}

func (r *PostgresPaymentRepository) FindByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error) {
	record, err := r.queries.GetPaymentByPaymentNo(ctx, paymentNo)
	if err != nil {
		return nil, err
	}

	return toDomainPayment(record), nil
}

func (r *PostgresPaymentRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Payment, error) {
	records, err := r.queries.ListPaymentsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	payments := make([]domain.Payment, 0, len(records))
	for _, record := range records {
		payments = append(payments, *toDomainPayment(record))
	}

	return payments, nil
}

func toDomainEvent(record db.Event) *domain.Event {
	return &domain.Event{
		ID:          record.ID,
		Name:        record.Name,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
}

func toDomainUser(record db.User) *domain.User {
	return &domain.User{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
}

func toDomainUserFromGetUserByID(record db.GetUserByIDRow) *domain.User {
	return &domain.User{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
}

func toDomainUserFromGetUserByEmail(record db.GetUserByEmailRow) *domain.User {
	return &domain.User{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
}

func toDomainUserFromCreateUser(record db.CreateUserRow) *domain.User {
	return &domain.User{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
}

func toDomainSection(record db.EventSection) *domain.Section {
	return &domain.Section{
		ID:               record.ID,
		EventID:          record.EventID,
		Name:             record.SectionName,
		Price:            record.Price,
		TotalQuantity:    int(record.TotalQuantity),
		ReservedQuantity: int(record.ReservedQuantity),
		SoldQuantity:     int(record.SoldQuantity),
		PurchaseLimit:    int(record.PurchaseLimit),
		Status:           domain.SectionStatus(record.Status),
		CreatedAt:        record.CreatedAt.Time,
		UpdatedAt:        record.UpdatedAt.Time,
	}
}

func toDomainReservation(record db.Reservation) *domain.Reservation {
	return &domain.Reservation{
		ID:          record.ID,
		EventID:     record.EventID,
		SectionID:   record.SectionID,
		UserID:      record.UserID,
		Quantity:    int(record.Quantity),
		UnitPrice:   record.UnitPrice,
		TotalAmount: record.TotalAmount,
		Status:      domain.ReservationStatus(record.Status),
		ExpiresAt:   record.ExpiresAt.Time,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
}

func toDomainOrder(record db.Order) *domain.Order {
	return &domain.Order{
		ID:            record.ID,
		OrderNo:       record.OrderNo,
		UserID:        record.UserID,
		EventID:       record.EventID,
		SectionID:     record.SectionID,
		ReservationID: record.ReservationID,
		Quantity:      int(record.Quantity),
		UnitPrice:     record.UnitPrice,
		TotalAmount:   record.TotalAmount,
		Status:        domain.OrderStatus(record.Status),
		ExpiresAt:     record.ExpiresAt.Time,
		CreatedAt:     record.CreatedAt.Time,
		UpdatedAt:     record.UpdatedAt.Time,
	}
}

func toDomainPayment(record db.Payment) *domain.Payment {
	payment := &domain.Payment{
		ID:        record.ID,
		OrderID:   record.OrderID,
		PaymentNo: record.PaymentNo,
		Method:    record.Method,
		Amount:    record.Amount,
		Status:    domain.PaymentStatus(record.Status),
		CreatedAt: record.CreatedAt.Time,
		UpdatedAt: record.UpdatedAt.Time,
	}

	if record.PaidAt.Valid {
		paidAt := record.PaidAt.Time
		payment.PaidAt = &paidAt
	}

	if record.FailedAt.Valid {
		failedAt := record.FailedAt.Time
		payment.FailedAt = &failedAt
	}

	return payment
}

func toPgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: !value.IsZero(),
	}
}

func nullablePgTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{
		Time:  *value,
		Valid: true,
	}
}

func orderPaidAt(order *domain.Order) *time.Time {
	if order.Status != domain.OrderStatusPaid {
		return nil
	}

	paidAt := order.UpdatedAt
	return &paidAt
}

func buildReservationNo(userID int64) string {
	return fmt.Sprintf("RSV-%d-%d", userID, time.Now().UnixNano())
}
