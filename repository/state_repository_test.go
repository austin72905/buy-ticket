package repository

import (
	"context"
	"errors"
	"testing"

	"buy-ticket/domain"
)

func TestMemoryOrderRepositoryUpdateStatusUsesExpectedStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewMemoryOrderRepository()
	order := &domain.Order{Status: domain.OrderStatusPendingPayment}
	if err := repo.Save(ctx, order); err != nil {
		t.Fatalf("save order: %v", err)
	}

	updated := *order
	updated.Status = domain.OrderStatusPaid
	if err := repo.UpdateStatus(ctx, &updated, domain.OrderStatusExpired); !errors.Is(err, ErrResourceStateConflict) {
		t.Fatalf("expected state conflict, got %v", err)
	}

	stored, err := repo.FindByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("find order: %v", err)
	}
	if stored.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("status changed after conflict: %v", stored.Status)
	}

	if err := repo.UpdateStatus(ctx, &updated, domain.OrderStatusPendingPayment); err != nil {
		t.Fatalf("update order status: %v", err)
	}
}

func TestMemoryReservationRepositoryUpdateStatusUsesExpectedStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewMemoryReservationRepository()
	reservation := &domain.Reservation{Status: domain.ReservationStatusHolding}
	if err := repo.Save(ctx, reservation); err != nil {
		t.Fatalf("save reservation: %v", err)
	}

	updated := *reservation
	updated.Status = domain.ReservationStatusConfirmed
	if err := repo.UpdateStatus(ctx, &updated, domain.ReservationStatusExpired); !errors.Is(err, ErrResourceStateConflict) {
		t.Fatalf("expected state conflict, got %v", err)
	}

	stored, err := repo.FindByID(ctx, reservation.ID)
	if err != nil {
		t.Fatalf("find reservation: %v", err)
	}
	if stored.Status != domain.ReservationStatusHolding {
		t.Fatalf("status changed after conflict: %v", stored.Status)
	}

	if err := repo.UpdateStatus(ctx, &updated, domain.ReservationStatusHolding); err != nil {
		t.Fatalf("update reservation status: %v", err)
	}
}

var _ OrderStateRepository = (*MemoryOrderRepository)(nil)
var _ ReservationStateRepository = (*MemoryReservationRepository)(nil)
var _ OrderStateRepository = (*PostgresOrderRepository)(nil)
var _ ReservationStateRepository = (*PostgresReservationRepository)(nil)
