package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

func TestBookingServicePaymentIdempotency(t *testing.T) {
	t.Run("new key creates processing record", func(t *testing.T) {
		repo := newFakeIdempotencyRepository()
		svc := newTestBookingService(testBookingDeps{
			IdempotencyRepo: repo,
		})

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
		svc := newTestBookingService(testBookingDeps{
			IdempotencyRepo: repo,
		})
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
		if err := svc.CompletePaymentIdempotency(context.Background(), 3, "pay-key-002", "POST /payments", 200, []byte(`{"id":1}`), now); err != nil {
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
		svc := newTestBookingService(testBookingDeps{
			IdempotencyRepo: repo,
		})

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

	t.Run("same key can be reused by a different user", func(t *testing.T) {
		repo := newFakeIdempotencyRepository()
		svc := newTestBookingService(testBookingDeps{IdempotencyRepo: repo})

		for _, userID := range []int64{3, 4} {
			_, replay, err := svc.BeginPaymentIdempotency(context.Background(), BeginIdempotencyInput{
				Key:         "shared-key",
				UserID:      userID,
				Endpoint:    "POST /payments",
				RequestHash: "hash-for-user",
			})
			if err != nil {
				t.Fatalf("user %d should be able to use the key: %v", userID, err)
			}
			if replay {
				t.Fatalf("user %d should create a separate record", userID)
			}
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

func (f *fakeIdempotencyRepository) FindByKeyAndEndpoint(ctx context.Context, userID int64, key, endpoint string) (*domain.IdempotencyKey, error) {
	record, ok := f.records[idempotencyMapKey(userID, key, endpoint)]
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
	f.records[idempotencyMapKey(record.UserID, record.Key, record.Endpoint)] = &cloned
	return nil
}

func (f *fakeIdempotencyRepository) Complete(ctx context.Context, userID int64, key, endpoint string, status int, responseBody []byte, now time.Time) error {
	record, ok := f.records[idempotencyMapKey(userID, key, endpoint)]
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

func idempotencyMapKey(userID int64, key, endpoint string) string {
	return fmt.Sprintf("%d:%s:%s", userID, endpoint, key)
}
