package service

import (
	"context"
	"testing"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

func TestBookingServiceAdvanceEventStatuses(t *testing.T) {
	now := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)
	eventRepo := repository.NewMemoryEventRepository([]*domain.Event{
		{
			ID:          1,
			Name:        "Draft Event",
			Status:      domain.EventStatusDraft,
			SaleStartAt: now.Add(-time.Hour),
			SaleEndAt:   now.Add(time.Hour),
		},
		{
			ID:          2,
			Name:        "Published Active Sale",
			Status:      domain.EventStatusPublished,
			SaleStartAt: now.Add(-time.Hour),
			SaleEndAt:   now.Add(time.Hour),
		},
		{
			ID:          3,
			Name:        "Published Missed Sale Window",
			Status:      domain.EventStatusPublished,
			SaleStartAt: now.Add(-2 * time.Hour),
			SaleEndAt:   now.Add(-time.Hour),
		},
		{
			ID:          4,
			Name:        "On Sale Ended",
			Status:      domain.EventStatusOnSale,
			SaleStartAt: now.Add(-2 * time.Hour),
			SaleEndAt:   now.Add(-time.Minute),
		},
	})
	svc := NewBookingService(eventRepo, nil, nil, nil, nil)

	count, err := svc.AdvanceEventStatuses(context.Background(), now)
	if err != nil {
		t.Fatalf("advance event statuses failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 advanced events, got %d", count)
	}

	draft, _ := eventRepo.FindByID(context.Background(), 1)
	if draft.Status != domain.EventStatusDraft {
		t.Fatalf("draft event should not auto start, got %d", draft.Status)
	}

	active, _ := eventRepo.FindByID(context.Background(), 2)
	if active.Status != domain.EventStatusOnSale {
		t.Fatalf("published event inside sale window should become on sale, got %d", active.Status)
	}

	missed, _ := eventRepo.FindByID(context.Background(), 3)
	if missed.Status != domain.EventStatusEnded {
		t.Fatalf("published event after sale window should become ended, got %d", missed.Status)
	}

	ended, _ := eventRepo.FindByID(context.Background(), 4)
	if ended.Status != domain.EventStatusEnded {
		t.Fatalf("on sale event after sale window should become ended, got %d", ended.Status)
	}
}
