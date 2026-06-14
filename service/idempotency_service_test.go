package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

func TestBookingServicePaymentIdempotency(t *testing.T) {
	t.Run("new key creates processing record", func(t *testing.T) {
		repo := newFakeIdempotencyRepository()
		svc := &BookingService{IdempotencyRepo: repo}

		record, replay, err := svc.BeginPaymentIdempotency(context.Background(), BeginIdempotencyInput{
			Key:         "pay-key-001",
			UserID:      3,
			Endpoint:    "POST /payments",
			RequestHash: "hash-001",
			Now:         time.Date(2026, 6, 14, 20, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("預期建立 idempotency key 成功，實際錯誤: %v", err)
		}
		if replay {
			t.Fatal("新 key 不應該 replay")
		}
		if record.Status != domain.IdempotencyStatusProcessing {
			t.Fatalf("預期狀態 processing，實際為 %d", record.Status)
		}
	})

	t.Run("completed key replays stored response", func(t *testing.T) {
		repo := newFakeIdempotencyRepository()
		svc := &BookingService{IdempotencyRepo: repo}
		now := time.Date(2026, 6, 14, 20, 0, 0, 0, time.UTC)

		_, _, err := svc.BeginPaymentIdempotency(context.Background(), BeginIdempotencyInput{
			Key:         "pay-key-002",
			UserID:      3,
			Endpoint:    "POST /payments",
			RequestHash: "hash-002",
			Now:         now,
		})
		if err != nil {
			t.Fatalf("預期建立 idempotency key 成功，實際錯誤: %v", err)
		}
		if err := svc.CompletePaymentIdempotency(context.Background(), "pay-key-002", "POST /payments", 200, []byte(`{"id":1}`), now); err != nil {
			t.Fatalf("預期完成 idempotency key 成功，實際錯誤: %v", err)
		}

		record, replay, err := svc.BeginPaymentIdempotency(context.Background(), BeginIdempotencyInput{
			Key:         "pay-key-002",
			UserID:      3,
			Endpoint:    "POST /payments",
			RequestHash: "hash-002",
			Now:         now,
		})
		if err != nil {
			t.Fatalf("預期 replay 成功，實際錯誤: %v", err)
		}
		if !replay {
			t.Fatal("預期 replay stored response")
		}
		if record.ResponseStatus == nil || *record.ResponseStatus != 200 {
			t.Fatal("預期 response status 被保存")
		}
	})

	t.Run("same key with different request hash is conflict", func(t *testing.T) {
		repo := newFakeIdempotencyRepository()
		svc := &BookingService{IdempotencyRepo: repo}

		_, _, err := svc.BeginPaymentIdempotency(context.Background(), BeginIdempotencyInput{
			Key:         "pay-key-003",
			UserID:      3,
			Endpoint:    "POST /payments",
			RequestHash: "hash-003",
		})
		if err != nil {
			t.Fatalf("預期建立 idempotency key 成功，實際錯誤: %v", err)
		}

		_, _, err = svc.BeginPaymentIdempotency(context.Background(), BeginIdempotencyInput{
			Key:         "pay-key-003",
			UserID:      3,
			Endpoint:    "POST /payments",
			RequestHash: "different-hash",
		})
		if !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("預期 idempotency conflict，實際為 %v", err)
		}
	})
}

type fakeIdempotencyRepository struct {
	nextID  int64
	records map[string]*domain.IdempotencyKey
}

func newFakeIdempotencyRepository() *fakeIdempotencyRepository {
	return &fakeIdempotencyRepository{
		nextID:  1,
		records: map[string]*domain.IdempotencyKey{},
	}
}

func (f *fakeIdempotencyRepository) FindByKeyAndEndpoint(ctx context.Context, key, endpoint string) (*domain.IdempotencyKey, error) {
	record, ok := f.records[idempotencyMapKey(key, endpoint)]
	if !ok {
		return nil, repository.ErrIdempotencyKeyNotFound
	}

	cloned := *record
	return &cloned, nil
}

func (f *fakeIdempotencyRepository) Create(ctx context.Context, record *domain.IdempotencyKey) error {
	record.ID = f.nextID
	f.nextID++

	cloned := *record
	f.records[idempotencyMapKey(record.Key, record.Endpoint)] = &cloned
	return nil
}

func (f *fakeIdempotencyRepository) Complete(ctx context.Context, key, endpoint string, status int, responseBody []byte, now time.Time) error {
	record, ok := f.records[idempotencyMapKey(key, endpoint)]
	if !ok {
		return repository.ErrIdempotencyKeyNotFound
	}

	record.Status = domain.IdempotencyStatusCompleted
	record.ResponseStatus = &status
	record.ResponseBody = responseBody
	record.LockedUntil = nil
	record.UpdatedAt = now
	return nil
}

func idempotencyMapKey(key, endpoint string) string {
	return endpoint + ":" + key
}
