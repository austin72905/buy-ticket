package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"buy-ticket/db/sqlc"
	"buy-ticket/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresEventRepository struct {
	queries *db.Queries
}

type PostgresUserRepository struct {
	queries *db.Queries
}

type PostgresAdminUserRepository struct {
	queries *db.Queries
}

type PostgresOrganizerRepository struct {
	queries *db.Queries
}

type PostgresAdminAuditLogRepository struct {
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

type PostgresPaymentAttemptRepository struct {
	queries *db.Queries
}

type PostgresIdempotencyRepository struct {
	queries *db.Queries
}

func NewPostgresEventRepository(queries *db.Queries) *PostgresEventRepository {
	return &PostgresEventRepository{queries: queries}
}

func NewPostgresUserRepository(queries *db.Queries) *PostgresUserRepository {
	return &PostgresUserRepository{queries: queries}
}

func NewPostgresAdminUserRepository(queries *db.Queries) *PostgresAdminUserRepository {
	return &PostgresAdminUserRepository{queries: queries}
}

func NewPostgresOrganizerRepository(queries *db.Queries) *PostgresOrganizerRepository {
	return &PostgresOrganizerRepository{queries: queries}
}

func NewPostgresAdminAuditLogRepository(queries *db.Queries) *PostgresAdminAuditLogRepository {
	return &PostgresAdminAuditLogRepository{queries: queries}
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

func NewPostgresPaymentAttemptRepository(queries *db.Queries) *PostgresPaymentAttemptRepository {
	return &PostgresPaymentAttemptRepository{queries: queries}
}

func NewPostgresIdempotencyRepository(queries *db.Queries) *PostgresIdempotencyRepository {
	return &PostgresIdempotencyRepository{queries: queries}
}

func (r *PostgresEventRepository) FindByID(ctx context.Context, eventID int64) (*domain.Event, error) {
	record, err := r.queries.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	return toDomainEventFromGetEventByID(record), nil
}

func (r *PostgresEventRepository) List(ctx context.Context) ([]domain.Event, error) {
	records, err := r.queries.ListEvents(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]domain.Event, 0, len(records))
	for _, record := range records {
		events = append(events, *toDomainEventFromListEvents(record))
	}

	return events, nil
}

func (r *PostgresEventRepository) ListAdminEvents(ctx context.Context, organizerID *int64) ([]domain.Event, error) {
	organizerIDValue := int64(0)
	if organizerID != nil {
		organizerIDValue = *organizerID
	}

	records, err := r.queries.ListAdminEvents(ctx, organizerIDValue)
	if err != nil {
		return nil, err
	}

	events := make([]domain.Event, 0, len(records))
	for _, record := range records {
		events = append(events, *toDomainEventFromListAdminEvents(record))
	}

	return events, nil
}

func (r *PostgresEventRepository) FindAdminEventByID(ctx context.Context, eventID int64, organizerID *int64) (*domain.Event, error) {
	organizerIDValue := int64(0)
	if organizerID != nil {
		organizerIDValue = *organizerID
	}

	record, err := r.queries.GetAdminEventByID(ctx, db.GetAdminEventByIDParams{
		EventID:     eventID,
		OrganizerID: organizerIDValue,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	return toDomainEventFromGetAdminEventByID(record), nil
}

func (r *PostgresEventRepository) CreateAdminEvent(ctx context.Context, event *domain.Event) error {
	record, err := r.queries.CreateEvent(ctx, db.CreateEventParams{
		OrganizerID: event.OrganizerID,
		Name:        event.Name,
		Venue:       event.Venue,
		Status:      int16(event.Status),
		StartAt:     toPgTimestamp(event.StartAt),
		EndAt:       toPgTimestamp(event.EndAt),
		SaleStartAt: toPgTimestamp(event.SaleStartAt),
		SaleEndAt:   toPgTimestamp(event.SaleEndAt),
	})
	if err != nil {
		return err
	}

	*event = *toDomainEventFromCreateEvent(record)
	return nil
}

func (r *PostgresEventRepository) UpdateAdminEvent(ctx context.Context, event *domain.Event) error {
	record, err := r.queries.UpdateEvent(ctx, db.UpdateEventParams{
		ID:          event.ID,
		OrganizerID: event.OrganizerID,
		Name:        event.Name,
		Venue:       event.Venue,
		Status:      int16(event.Status),
		StartAt:     toPgTimestamp(event.StartAt),
		EndAt:       toPgTimestamp(event.EndAt),
		SaleStartAt: toPgTimestamp(event.SaleStartAt),
		SaleEndAt:   toPgTimestamp(event.SaleEndAt),
		UpdatedAt:   toPgTimestamp(event.UpdatedAt),
		Version:     event.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceVersionConflict
		}
		return err
	}

	*event = *toDomainEventFromUpdateEvent(record)
	return nil
}

func (r *PostgresEventRepository) AdvanceEventStatuses(ctx context.Context, now time.Time) (int64, error) {
	return r.queries.AdvanceEventStatuses(ctx, toPgTimestamp(now))
}

func (r *PostgresEventRepository) ListAdminEventSections(ctx context.Context, eventID int64, organizerID *int64) ([]domain.Section, error) {
	organizerIDValue := int64(0)
	if organizerID != nil {
		organizerIDValue = *organizerID
	}

	records, err := r.queries.ListAdminEventSections(ctx, db.ListAdminEventSectionsParams{
		EventID:     eventID,
		OrganizerID: organizerIDValue,
	})
	if err != nil {
		return nil, err
	}

	sections := make([]domain.Section, 0, len(records))
	for _, record := range records {
		sections = append(sections, *toDomainSectionFromListAdminEventSections(record))
	}

	return sections, nil
}

func (r *PostgresEventRepository) CreateAdminEventSection(ctx context.Context, eventID int64, organizerID *int64, section *domain.Section) error {
	event, err := r.FindAdminEventByID(ctx, eventID, organizerID)
	if err != nil {
		return err
	}

	record, err := r.queries.CreateSection(ctx, db.CreateSectionParams{
		EventID:          eventID,
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

	*section = *toDomainSectionFromCreateSection(record)
	return nil
}

func (r *PostgresEventRepository) UpdateAdminEventSection(ctx context.Context, eventID int64, organizerID *int64, section *domain.Section) error {
	if _, err := r.FindAdminEventByID(ctx, eventID, organizerID); err != nil {
		return err
	}

	record, err := r.queries.UpdateSection(ctx, db.UpdateSectionParams{
		EventID:       eventID,
		ID:            section.ID,
		SectionName:   section.Name,
		Price:         section.Price,
		TotalQuantity: int32(section.TotalQuantity),
		PurchaseLimit: int32(section.PurchaseLimit),
		Status:        int16(section.Status),
		UpdatedAt:     toPgTimestamp(section.UpdatedAt),
		Version:       section.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceVersionConflict
		}
		return err
	}

	*section = *toDomainSectionFromUpdateSection(record)
	return nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, userID int64) (*domain.User, error) {
	record, err := r.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return toDomainUserFromGetUserByID(record), nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	record, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
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

	return toDomainSectionFromGetSectionByEventAndID(record), nil
}

func (r *PostgresSectionRepository) ListByEventID(ctx context.Context, eventID int64) ([]domain.Section, error) {
	records, err := r.queries.ListSectionsByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	sections := make([]domain.Section, 0, len(records))
	for _, record := range records {
		sections = append(sections, *toDomainSectionFromListSectionsByEventID(record))
	}

	return sections, nil
}

func (r *PostgresSectionRepository) ReserveInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	record, err := r.queries.ReserveSectionInventory(ctx, db.ReserveSectionInventoryParams{
		EventID:   eventID,
		ID:        sectionID,
		Quantity:  int32(quantity),
		UpdatedAt: toPgTimestamp(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSectionNotFound
		}
		return nil, err
	}

	return toDomainSectionFromReserveSectionInventory(record), nil
}

func (r *PostgresSectionRepository) ReleaseInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	record, err := r.queries.ReleaseSectionInventory(ctx, db.ReleaseSectionInventoryParams{
		EventID:   eventID,
		ID:        sectionID,
		Quantity:  int32(quantity),
		UpdatedAt: toPgTimestamp(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSectionNotFound
		}
		return nil, err
	}

	return toDomainSectionFromReleaseSectionInventory(record), nil
}

func (r *PostgresSectionRepository) ConfirmSale(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	record, err := r.queries.ConfirmSectionSale(ctx, db.ConfirmSectionSaleParams{
		EventID:   eventID,
		ID:        sectionID,
		Quantity:  int32(quantity),
		UpdatedAt: toPgTimestamp(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSectionNotFound
		}
		return nil, err
	}

	return toDomainSectionFromConfirmSectionSale(record), nil
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

		*section = *toDomainSectionFromCreateSection(record)
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

func (r *PostgresReservationRepository) FindActiveByUserAndEvent(ctx context.Context, userID, eventID int64, now time.Time) (*domain.Reservation, error) {
	record, err := r.queries.GetActiveReservationByUserAndEvent(ctx, db.GetActiveReservationByUserAndEventParams{
		UserID:    userID,
		EventID:   eventID,
		ExpiresAt: toPgTimestamp(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	return toDomainReservation(record), nil
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

func (r *PostgresOrderRepository) ListAdminOrders(ctx context.Context, filter domain.AdminOrderListFilter) ([]domain.AdminOrder, error) {
	organizerID := int64(0)
	if filter.OrganizerID != nil {
		organizerID = *filter.OrganizerID
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	records, err := r.queries.ListAdminOrders(ctx, db.ListAdminOrdersParams{
		OrganizerID:     organizerID,
		EventID:         filter.EventID,
		UserID:          filter.UserID,
		Status:          int16(filter.Status),
		CursorCreatedAt: nullablePgTimestamp(filter.Cursor.CreatedAt),
		CursorID:        filter.Cursor.ID,
		PageLimit:       int32(limit),
	})
	if err != nil {
		return nil, err
	}

	orders := make([]domain.AdminOrder, 0, len(records))
	for _, record := range records {
		orders = append(orders, *toDomainAdminOrder(record))
	}

	return orders, nil
}

func (r *PostgresOrderRepository) FindAdminOrderSensitiveByID(ctx context.Context, orderID int64) (*domain.AdminOrder, error) {
	record, err := r.queries.GetAdminOrderSensitiveByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	return toDomainAdminOrderFromSensitive(record), nil
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

func (r *PostgresPaymentAttemptRepository) FindByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (*domain.PaymentAttempt, error) {
	record, err := r.queries.GetPaymentAttemptByMerchantTradeNo(ctx, merchantTradeNo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentAttemptNotFound
		}
		return nil, err
	}

	return toDomainPaymentAttempt(record), nil
}

func (r *PostgresPaymentAttemptRepository) FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.PaymentAttempt, error) {
	record, err := r.queries.GetPaymentAttemptByIdempotencyKey(ctx, pgtype.Text{
		String: idempotencyKey,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentAttemptNotFound
		}
		return nil, err
	}

	return toDomainPaymentAttempt(record), nil
}

func (r *PostgresPaymentAttemptRepository) ListByOrderID(ctx context.Context, orderID int64) ([]domain.PaymentAttempt, error) {
	records, err := r.queries.ListPaymentAttemptsByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	attempts := make([]domain.PaymentAttempt, 0, len(records))
	for _, record := range records {
		attempts = append(attempts, *toDomainPaymentAttempt(record))
	}

	return attempts, nil
}

func (r *PostgresPaymentAttemptRepository) Save(ctx context.Context, attempt *domain.PaymentAttempt) error {
	if attempt.ID == 0 {
		record, err := r.queries.CreatePaymentAttempt(ctx, db.CreatePaymentAttemptParams{
			OrderID:         attempt.OrderID,
			PaymentID:       nullablePgInt8(attempt.PaymentID),
			IdempotencyKey:  nullablePgText(attempt.IdempotencyKey),
			Provider:        attempt.Provider,
			MerchantTradeNo: attempt.MerchantTradeNo,
			ProviderTradeNo: nullablePgText(attempt.ProviderTradeNo),
			Method:          attempt.Method,
			Amount:          attempt.Amount,
			Status:          int16(attempt.Status),
			RequestPayload:  attempt.RequestPayload,
			ResponsePayload: attempt.ResponsePayload,
			CallbackPayload: attempt.CallbackPayload,
			FailureReason:   nullablePgText(attempt.FailureReason),
			ExpiresAt:       nullablePgTimestamp(attempt.ExpiresAt),
			SucceededAt:     nullablePgTimestamp(attempt.SucceededAt),
			FailedAt:        nullablePgTimestamp(attempt.FailedAt),
			CreatedAt:       toPgTimestamp(attempt.CreatedAt),
		})
		if err != nil {
			return err
		}

		*attempt = *toDomainPaymentAttempt(record)
		return nil
	}

	return r.queries.UpdatePaymentAttemptStatus(ctx, db.UpdatePaymentAttemptStatusParams{
		ID:              attempt.ID,
		PaymentID:       nullablePgInt8(attempt.PaymentID),
		ProviderTradeNo: nullablePgText(attempt.ProviderTradeNo),
		Status:          int16(attempt.Status),
		ResponsePayload: attempt.ResponsePayload,
		CallbackPayload: attempt.CallbackPayload,
		FailureReason:   nullablePgText(attempt.FailureReason),
		SucceededAt:     nullablePgTimestamp(attempt.SucceededAt),
		FailedAt:        nullablePgTimestamp(attempt.FailedAt),
		UpdatedAt:       toPgTimestamp(attempt.UpdatedAt),
	})
}

func (r *PostgresIdempotencyRepository) FindByKeyAndEndpoint(ctx context.Context, key, endpoint string) (*domain.IdempotencyKey, error) {
	record, err := r.queries.GetIdempotencyKey(ctx, db.GetIdempotencyKeyParams{
		Key:      key,
		Endpoint: endpoint,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIdempotencyKeyNotFound
		}
		return nil, err
	}

	return toDomainIdempotencyKey(record), nil
}

func (r *PostgresIdempotencyRepository) Create(ctx context.Context, record *domain.IdempotencyKey) error {
	created, err := r.queries.CreateIdempotencyKey(ctx, db.CreateIdempotencyKeyParams{
		Key:         record.Key,
		UserID:      nullablePgInt8(record.UserID),
		Endpoint:    record.Endpoint,
		RequestHash: record.RequestHash,
		Status:      int16(record.Status),
		LockedUntil: nullablePgTimestamp(record.LockedUntil),
		ExpiresAt:   toPgTimestamp(record.ExpiresAt),
		CreatedAt:   toPgTimestamp(record.CreatedAt),
	})
	if err != nil {
		return err
	}

	*record = *toDomainIdempotencyKey(created)
	return nil
}

func (r *PostgresAdminUserRepository) FindByID(ctx context.Context, adminUserID int64) (*domain.AdminUser, error) {
	record, err := r.queries.GetAdminUserByID(ctx, adminUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminUserNotFound
		}
		return nil, err
	}

	return toDomainAdminUserFromGetAdminUserByID(record), nil
}

func (r *PostgresAdminUserRepository) FindByEmail(ctx context.Context, email string) (*domain.AdminUser, error) {
	record, err := r.queries.GetAdminUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminUserNotFound
		}
		return nil, err
	}

	return toDomainAdminUserFromGetAdminUserByEmail(record), nil
}

func (r *PostgresAdminUserRepository) List(ctx context.Context) ([]domain.AdminUser, error) {
	records, err := r.queries.ListAdminUsers(ctx)
	if err != nil {
		return nil, err
	}

	adminUsers := make([]domain.AdminUser, 0, len(records))
	for _, record := range records {
		adminUsers = append(adminUsers, *toDomainAdminUserFromListAdminUsers(record))
	}

	return adminUsers, nil
}

func (r *PostgresAdminUserRepository) Save(ctx context.Context, adminUser *domain.AdminUser) error {
	if adminUser.ID != 0 {
		record, err := r.queries.UpdateAdminUser(ctx, db.UpdateAdminUserParams{
			ID:           adminUser.ID,
			OrganizerID:  nullablePgInt8(adminUser.OrganizerID),
			Name:         adminUser.Name,
			Email:        adminUser.Email,
			PasswordHash: adminUser.PasswordHash,
			Role:         string(adminUser.Role),
			Status:       int16(adminUser.Status),
			UpdatedAt:    toPgTimestamp(adminUser.UpdatedAt),
			Version:      adminUser.Version,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrResourceVersionConflict
			}
			return err
		}

		*adminUser = *toDomainAdminUserFromUpdateAdminUser(record)
		return nil
	}

	record, err := r.queries.CreateAdminUser(ctx, db.CreateAdminUserParams{
		OrganizerID:  nullablePgInt8(adminUser.OrganizerID),
		Name:         adminUser.Name,
		Email:        adminUser.Email,
		PasswordHash: adminUser.PasswordHash,
		Role:         string(adminUser.Role),
		Status:       int16(adminUser.Status),
	})
	if err != nil {
		return err
	}

	*adminUser = *toDomainAdminUserFromCreateAdminUser(record)
	return nil
}

func (r *PostgresOrganizerRepository) FindByID(ctx context.Context, organizerID int64) (*domain.Organizer, error) {
	record, err := r.queries.GetOrganizerByID(ctx, organizerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrganizerNotFound
		}
		return nil, err
	}

	return toDomainOrganizerFromGetOrganizerByID(record), nil
}

func (r *PostgresOrganizerRepository) List(ctx context.Context) ([]domain.Organizer, error) {
	records, err := r.queries.ListOrganizers(ctx)
	if err != nil {
		return nil, err
	}

	organizers := make([]domain.Organizer, 0, len(records))
	for _, record := range records {
		organizers = append(organizers, *toDomainOrganizerFromListOrganizers(record))
	}

	return organizers, nil
}

func (r *PostgresOrganizerRepository) Save(ctx context.Context, organizer *domain.Organizer) error {
	if organizer.ID != 0 {
		record, err := r.queries.UpdateOrganizer(ctx, db.UpdateOrganizerParams{
			ID:        organizer.ID,
			Name:      organizer.Name,
			Status:    int16(organizer.Status),
			UpdatedAt: toPgTimestamp(organizer.UpdatedAt),
			Version:   organizer.Version,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrResourceVersionConflict
			}
			return err
		}

		*organizer = *toDomainOrganizerFromUpdateOrganizer(record)
		return nil
	}

	record, err := r.queries.CreateOrganizer(ctx, db.CreateOrganizerParams{
		Name:   organizer.Name,
		Status: int16(organizer.Status),
	})
	if err != nil {
		return err
	}

	*organizer = *toDomainOrganizerFromCreateOrganizer(record)
	return nil
}

func (r *PostgresAdminAuditLogRepository) Create(ctx context.Context, log *domain.AdminAuditLog) error {
	record, err := r.queries.CreateAdminAuditLog(ctx, db.CreateAdminAuditLogParams{
		AdminUserID: log.AdminUserID,
		Action:      log.Action,
		TargetType:  log.TargetType,
		TargetID:    log.TargetID,
		Reason:      nullablePgText(log.Reason),
		IpAddress:   nullablePgText(log.IPAddress),
		UserAgent:   nullablePgText(log.UserAgent),
	})
	if err != nil {
		return err
	}

	*log = *toDomainAdminAuditLogFromCreateAdminAuditLog(record)
	return nil
}

func (r *PostgresAdminAuditLogRepository) List(ctx context.Context, filter domain.AdminAuditLogListFilter) ([]domain.AdminAuditLog, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	records, err := r.queries.ListAdminAuditLogs(ctx, db.ListAdminAuditLogsParams{
		AdminUserID:     filter.AdminUserID,
		CursorCreatedAt: nullablePgTimestamp(filter.Cursor.CreatedAt),
		CursorID:        filter.Cursor.ID,
		PageLimit:       int32(limit),
	})
	if err != nil {
		return nil, err
	}

	logs := make([]domain.AdminAuditLog, 0, len(records))
	for _, record := range records {
		logs = append(logs, *toDomainAdminAuditLogFromListAdminAuditLogs(record))
	}

	return logs, nil
}

func (r *PostgresIdempotencyRepository) Complete(ctx context.Context, key, endpoint string, status int, responseBody []byte, now time.Time) error {
	return r.queries.CompleteIdempotencyKey(ctx, db.CompleteIdempotencyKeyParams{
		Key:            key,
		Endpoint:       endpoint,
		Status:         int16(domain.IdempotencyStatusCompleted),
		ResponseStatus: nullablePgInt4(status),
		ResponseBody:   responseBody,
		UpdatedAt:      toPgTimestamp(now),
	})
}

func toDomainEvent(record db.Event) *domain.Event {
	return &domain.Event{
		ID:          record.ID,
		OrganizerID: record.OrganizerID,
		Name:        record.Name,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		Version:     record.Version,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
}

func toDomainEventFromGetEventByID(record db.GetEventByIDRow) *domain.Event {
	return &domain.Event{
		ID:          record.ID,
		OrganizerID: record.OrganizerID,
		Name:        record.Name,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		Version:     record.Version,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
}

func toDomainEventFromListEvents(record db.ListEventsRow) *domain.Event {
	return &domain.Event{
		ID:          record.ID,
		OrganizerID: record.OrganizerID,
		Name:        record.Name,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		Version:     record.Version,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
}

func toDomainEventFromListAdminEvents(record db.ListAdminEventsRow) *domain.Event {
	event := &domain.Event{
		ID:          record.ID,
		OrganizerID: record.OrganizerID,
		Name:        record.Name,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		Version:     record.Version,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
	event.Organizer = toDomainOrganizerSummary(record.OrganizerID, record.OrganizerName, record.OrganizerStatus)
	return event
}

func toDomainEventFromGetAdminEventByID(record db.GetAdminEventByIDRow) *domain.Event {
	event := &domain.Event{
		ID:          record.ID,
		OrganizerID: record.OrganizerID,
		Name:        record.Name,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		Version:     record.Version,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
	event.Organizer = toDomainOrganizerSummary(record.OrganizerID, record.OrganizerName, record.OrganizerStatus)
	return event
}

func toDomainEventFromCreateEvent(record db.CreateEventRow) *domain.Event {
	return &domain.Event{
		ID:          record.ID,
		OrganizerID: record.OrganizerID,
		Name:        record.Name,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		Version:     record.Version,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
}

func toDomainOrganizer(record db.Organizer) *domain.Organizer {
	return &domain.Organizer{
		ID:        record.ID,
		Name:      record.Name,
		Status:    domain.OrganizerStatus(record.Status),
		Version:   record.Version,
		CreatedAt: record.CreatedAt.Time,
		UpdatedAt: record.UpdatedAt.Time,
	}
}

func toDomainOrganizerFromGetOrganizerByID(record db.GetOrganizerByIDRow) *domain.Organizer {
	return &domain.Organizer{
		ID:        record.ID,
		Name:      record.Name,
		Status:    domain.OrganizerStatus(record.Status),
		Version:   record.Version,
		CreatedAt: record.CreatedAt.Time,
		UpdatedAt: record.UpdatedAt.Time,
	}
}

func toDomainOrganizerFromListOrganizers(record db.ListOrganizersRow) *domain.Organizer {
	return &domain.Organizer{
		ID:        record.ID,
		Name:      record.Name,
		Status:    domain.OrganizerStatus(record.Status),
		Version:   record.Version,
		CreatedAt: record.CreatedAt.Time,
		UpdatedAt: record.UpdatedAt.Time,
	}
}

func toDomainOrganizerFromCreateOrganizer(record db.CreateOrganizerRow) *domain.Organizer {
	return &domain.Organizer{
		ID:        record.ID,
		Name:      record.Name,
		Status:    domain.OrganizerStatus(record.Status),
		Version:   record.Version,
		CreatedAt: record.CreatedAt.Time,
		UpdatedAt: record.UpdatedAt.Time,
	}
}

func toDomainOrganizerFromUpdateOrganizer(record db.UpdateOrganizerRow) *domain.Organizer {
	return &domain.Organizer{
		ID:        record.ID,
		Name:      record.Name,
		Status:    domain.OrganizerStatus(record.Status),
		Version:   record.Version,
		CreatedAt: record.CreatedAt.Time,
		UpdatedAt: record.UpdatedAt.Time,
	}
}

func toDomainOrganizerSummary(id int64, name pgtype.Text, status pgtype.Int2) *domain.Organizer {
	if !name.Valid {
		return nil
	}

	organizer := &domain.Organizer{
		ID:   id,
		Name: name.String,
	}
	if status.Valid {
		organizer.Status = domain.OrganizerStatus(status.Int16)
	}
	return organizer
}

func toDomainAdminUser(record db.AdminUser) *domain.AdminUser {
	adminUser := &domain.AdminUser{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         domain.AdminRole(record.Role),
		Status:       domain.AdminUserStatus(record.Status),
		Version:      record.Version,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
	if record.OrganizerID.Valid {
		organizerID := record.OrganizerID.Int64
		adminUser.OrganizerID = &organizerID
	}
	return adminUser
}

func toDomainAdminUserFromGetAdminUserByID(record db.GetAdminUserByIDRow) *domain.AdminUser {
	adminUser := &domain.AdminUser{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         domain.AdminRole(record.Role),
		Status:       domain.AdminUserStatus(record.Status),
		Version:      record.Version,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
	if record.OrganizerID.Valid {
		organizerID := record.OrganizerID.Int64
		adminUser.OrganizerID = &organizerID
	}
	return adminUser
}

func toDomainAdminUserFromGetAdminUserByEmail(record db.GetAdminUserByEmailRow) *domain.AdminUser {
	adminUser := &domain.AdminUser{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         domain.AdminRole(record.Role),
		Status:       domain.AdminUserStatus(record.Status),
		Version:      record.Version,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
	if record.OrganizerID.Valid {
		organizerID := record.OrganizerID.Int64
		adminUser.OrganizerID = &organizerID
	}
	return adminUser
}

func toDomainAdminUserFromListAdminUsers(record db.ListAdminUsersRow) *domain.AdminUser {
	adminUser := &domain.AdminUser{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         domain.AdminRole(record.Role),
		Status:       domain.AdminUserStatus(record.Status),
		Version:      record.Version,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
	if record.OrganizerID.Valid {
		organizerID := record.OrganizerID.Int64
		adminUser.OrganizerID = &organizerID
	}
	return adminUser
}

func toDomainAdminUserFromCreateAdminUser(record db.CreateAdminUserRow) *domain.AdminUser {
	adminUser := &domain.AdminUser{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         domain.AdminRole(record.Role),
		Status:       domain.AdminUserStatus(record.Status),
		Version:      record.Version,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
	if record.OrganizerID.Valid {
		organizerID := record.OrganizerID.Int64
		adminUser.OrganizerID = &organizerID
	}
	return adminUser
}

func toDomainAdminUserFromUpdateAdminUser(record db.UpdateAdminUserRow) *domain.AdminUser {
	adminUser := &domain.AdminUser{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         domain.AdminRole(record.Role),
		Status:       domain.AdminUserStatus(record.Status),
		Version:      record.Version,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}
	if record.OrganizerID.Valid {
		organizerID := record.OrganizerID.Int64
		adminUser.OrganizerID = &organizerID
	}
	return adminUser
}

func toDomainAdminAuditLogFromCreateAdminAuditLog(record db.AdminAuditLog) *domain.AdminAuditLog {
	return toDomainAdminAuditLog(record)
}

func toDomainAdminAuditLogFromListAdminAuditLogs(record db.ListAdminAuditLogsRow) *domain.AdminAuditLog {
	log := &domain.AdminAuditLog{
		ID:          record.ID,
		AdminUserID: record.AdminUserID,
		Action:      record.Action,
		TargetType:  record.TargetType,
		TargetID:    record.TargetID,
		CreatedAt:   record.CreatedAt.Time,
	}
	if record.AdminUserName.Valid {
		adminUser := &domain.AdminUser{
			ID:     record.AdminUserID,
			Name:   record.AdminUserName.String,
			Email:  record.AdminUserEmail.String,
			Role:   domain.AdminRole(record.AdminUserRole.String),
			Status: domain.AdminUserStatus(record.AdminUserStatus.Int16),
		}
		log.AdminUser = adminUser
	}
	if targetName, ok := stringFromSQLValue(record.TargetName); ok {
		log.TargetName = &targetName
	}
	if record.Reason.Valid {
		reason := record.Reason.String
		log.Reason = &reason
	}
	if record.IpAddress.Valid {
		ipAddress := record.IpAddress.String
		log.IPAddress = &ipAddress
	}
	if record.UserAgent.Valid {
		userAgent := record.UserAgent.String
		log.UserAgent = &userAgent
	}
	return log
}

func toDomainAdminAuditLog(record db.AdminAuditLog) *domain.AdminAuditLog {
	log := &domain.AdminAuditLog{
		ID:          record.ID,
		AdminUserID: record.AdminUserID,
		Action:      record.Action,
		TargetType:  record.TargetType,
		TargetID:    record.TargetID,
		CreatedAt:   record.CreatedAt.Time,
	}
	if record.Reason.Valid {
		reason := record.Reason.String
		log.Reason = &reason
	}
	if record.IpAddress.Valid {
		ipAddress := record.IpAddress.String
		log.IPAddress = &ipAddress
	}
	if record.UserAgent.Valid {
		userAgent := record.UserAgent.String
		log.UserAgent = &userAgent
	}
	return log
}

func stringFromSQLValue(value interface{}) (string, bool) {
	switch typed := value.(type) {
	case nil:
		return "", false
	case string:
		return typed, typed != ""
	case []byte:
		text := string(typed)
		return text, text != ""
	default:
		text := fmt.Sprint(typed)
		return text, text != ""
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

func toDomainEventFromUpdateEvent(record db.UpdateEventRow) *domain.Event {
	return &domain.Event{
		ID:          record.ID,
		OrganizerID: record.OrganizerID,
		Name:        record.Name,
		Venue:       record.Venue,
		Status:      domain.EventStatus(record.Status),
		Version:     record.Version,
		StartAt:     record.StartAt.Time,
		EndAt:       record.EndAt.Time,
		SaleStartAt: record.SaleStartAt.Time,
		SaleEndAt:   record.SaleEndAt.Time,
		CreatedAt:   record.CreatedAt.Time,
		UpdatedAt:   record.UpdatedAt.Time,
	}
}

func newDomainSection(
	id int64,
	eventID int64,
	name string,
	price int64,
	totalQuantity int32,
	reservedQuantity int32,
	soldQuantity int32,
	purchaseLimit int32,
	status int16,
	version int64,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) *domain.Section {
	return &domain.Section{
		ID:               id,
		EventID:          eventID,
		Name:             name,
		Price:            price,
		TotalQuantity:    int(totalQuantity),
		ReservedQuantity: int(reservedQuantity),
		SoldQuantity:     int(soldQuantity),
		PurchaseLimit:    int(purchaseLimit),
		Status:           domain.SectionStatus(status),
		Version:          version,
		CreatedAt:        createdAt.Time,
		UpdatedAt:        updatedAt.Time,
	}
}

func toDomainSection(record db.EventSection) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromGetSectionByEventAndID(record db.GetSectionByEventAndIDRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromListSectionsByEventID(record db.ListSectionsByEventIDRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromListAdminEventSections(record db.ListAdminEventSectionsRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromCreateSection(record db.CreateSectionRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromUpdateSection(record db.UpdateSectionRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromReserveSectionInventory(record db.ReserveSectionInventoryRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromReleaseSectionInventory(record db.ReleaseSectionInventoryRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
}

func toDomainSectionFromConfirmSectionSale(record db.ConfirmSectionSaleRow) *domain.Section {
	return newDomainSection(record.ID, record.EventID, record.SectionName, record.Price, record.TotalQuantity, record.ReservedQuantity, record.SoldQuantity, record.PurchaseLimit, record.Status, record.Version, record.CreatedAt, record.UpdatedAt)
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

func toDomainAdminOrder(record db.ListAdminOrdersRow) *domain.AdminOrder {
	order := &domain.AdminOrder{
		ID:            record.ID,
		OrderNo:       record.OrderNo,
		ReservationID: record.ReservationID,
		EventID:       record.EventID,
		EventName:     record.EventName,
		SectionID:     record.SectionID,
		SectionName:   record.SectionName,
		UserID:        record.UserID,
		UserName:      record.UserName,
		UserEmail:     record.UserEmail,
		Quantity:      int(record.Quantity),
		UnitPrice:     record.UnitPrice,
		TotalAmount:   record.TotalAmount,
		Status:        domain.OrderStatus(record.Status),
		ExpiresAt:     record.ExpiresAt.Time,
		CreatedAt:     record.CreatedAt.Time,
		UpdatedAt:     record.UpdatedAt.Time,
	}

	if record.PaidAt.Valid {
		paidAt := record.PaidAt.Time
		order.PaidAt = &paidAt
	}

	return order
}

func toDomainAdminOrderFromSensitive(record db.GetAdminOrderSensitiveByIDRow) *domain.AdminOrder {
	order := &domain.AdminOrder{
		ID:            record.ID,
		OrderNo:       record.OrderNo,
		ReservationID: record.ReservationID,
		EventID:       record.EventID,
		EventName:     record.EventName,
		SectionID:     record.SectionID,
		SectionName:   record.SectionName,
		UserID:        record.UserID,
		UserName:      record.UserName,
		UserEmail:     record.UserEmail,
		Quantity:      int(record.Quantity),
		UnitPrice:     record.UnitPrice,
		TotalAmount:   record.TotalAmount,
		Status:        domain.OrderStatus(record.Status),
		ExpiresAt:     record.ExpiresAt.Time,
		CreatedAt:     record.CreatedAt.Time,
		UpdatedAt:     record.UpdatedAt.Time,
	}

	if record.PaidAt.Valid {
		paidAt := record.PaidAt.Time
		order.PaidAt = &paidAt
	}

	return order
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

func toDomainPaymentAttempt(record db.PaymentAttempt) *domain.PaymentAttempt {
	attempt := &domain.PaymentAttempt{
		ID:              record.ID,
		OrderID:         record.OrderID,
		Provider:        record.Provider,
		MerchantTradeNo: record.MerchantTradeNo,
		Method:          record.Method,
		Amount:          record.Amount,
		Status:          domain.PaymentAttemptStatus(record.Status),
		RequestPayload:  record.RequestPayload,
		ResponsePayload: record.ResponsePayload,
		CallbackPayload: record.CallbackPayload,
		CreatedAt:       record.CreatedAt.Time,
		UpdatedAt:       record.UpdatedAt.Time,
	}

	if record.PaymentID.Valid {
		paymentID := record.PaymentID.Int64
		attempt.PaymentID = &paymentID
	}
	if record.IdempotencyKey.Valid {
		idempotencyKey := record.IdempotencyKey.String
		attempt.IdempotencyKey = &idempotencyKey
	}
	if record.ProviderTradeNo.Valid {
		providerTradeNo := record.ProviderTradeNo.String
		attempt.ProviderTradeNo = &providerTradeNo
	}
	if record.FailureReason.Valid {
		failureReason := record.FailureReason.String
		attempt.FailureReason = &failureReason
	}
	if record.ExpiresAt.Valid {
		expiresAt := record.ExpiresAt.Time
		attempt.ExpiresAt = &expiresAt
	}
	if record.SucceededAt.Valid {
		succeededAt := record.SucceededAt.Time
		attempt.SucceededAt = &succeededAt
	}
	if record.FailedAt.Valid {
		failedAt := record.FailedAt.Time
		attempt.FailedAt = &failedAt
	}

	return attempt
}

func toDomainIdempotencyKey(record db.IdempotencyKey) *domain.IdempotencyKey {
	idempotencyKey := &domain.IdempotencyKey{
		ID:           record.ID,
		Key:          record.Key,
		Endpoint:     record.Endpoint,
		RequestHash:  record.RequestHash,
		Status:       domain.IdempotencyStatus(record.Status),
		ResponseBody: record.ResponseBody,
		ExpiresAt:    record.ExpiresAt.Time,
		CreatedAt:    record.CreatedAt.Time,
		UpdatedAt:    record.UpdatedAt.Time,
	}

	if record.UserID.Valid {
		userID := record.UserID.Int64
		idempotencyKey.UserID = &userID
	}
	if record.ResponseStatus.Valid {
		responseStatus := int(record.ResponseStatus.Int32)
		idempotencyKey.ResponseStatus = &responseStatus
	}
	if record.LockedUntil.Valid {
		lockedUntil := record.LockedUntil.Time
		idempotencyKey.LockedUntil = &lockedUntil
	}

	return idempotencyKey
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

func nullablePgInt8(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{
		Int64: *value,
		Valid: true,
	}
}

func nullablePgInt4(value int) pgtype.Int4 {
	return pgtype.Int4{
		Int32: int32(value),
		Valid: true,
	}
}

func nullablePgText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: *value,
		Valid:  true,
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
