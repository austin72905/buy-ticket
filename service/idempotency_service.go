package service

import (
	"context"
	"errors"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

var (
	ErrIdempotencyConflict   = errors.New("idempotency key reused with different request")
	ErrIdempotencyInProgress = errors.New("idempotency request is still processing")
)

type BeginIdempotencyInput struct {
	Key         string
	UserID      int64
	Endpoint    string
	RequestHash string
	Now         time.Time
	TTL         time.Duration
	LockTTL     time.Duration
}

func (s *BookingService) BeginPaymentIdempotency(ctx context.Context, input BeginIdempotencyInput) (*domain.IdempotencyKey, bool, error) {
	if s.IdempotencyRepo == nil || input.Key == "" {
		return nil, false, nil
	}

	existing, err := s.IdempotencyRepo.FindByKeyAndEndpoint(ctx, input.Key, input.Endpoint)
	if err == nil {
		if existing.UserID == nil || *existing.UserID != input.UserID || existing.RequestHash != input.RequestHash {
			return nil, false, ErrIdempotencyConflict
		}
		if existing.Status == domain.IdempotencyStatusCompleted && existing.ResponseStatus != nil && len(existing.ResponseBody) > 0 {
			return existing, true, nil
		}
		return nil, false, ErrIdempotencyInProgress
	}
	if !errors.Is(err, repository.ErrIdempotencyKeyNotFound) {
		return nil, false, err
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	lockTTL := input.LockTTL
	if lockTTL <= 0 {
		lockTTL = 5 * time.Minute
	}

	userID := input.UserID
	lockedUntil := now.Add(lockTTL)
	record := &domain.IdempotencyKey{
		Key:         input.Key,
		UserID:      &userID,
		Endpoint:    input.Endpoint,
		RequestHash: input.RequestHash,
		Status:      domain.IdempotencyStatusProcessing,
		LockedUntil: &lockedUntil,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.IdempotencyRepo.Create(ctx, record); err != nil {
		return nil, false, err
	}

	return record, false, nil
}

func (s *BookingService) CompletePaymentIdempotency(ctx context.Context, key, endpoint string, responseStatus int, responseBody []byte, now time.Time) error {
	if s.IdempotencyRepo == nil || key == "" {
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	return s.IdempotencyRepo.Complete(ctx, key, endpoint, responseStatus, responseBody, now)
}
