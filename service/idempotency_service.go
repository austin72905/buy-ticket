package service

import (
	"context"
	"errors"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

var (
	ErrIdempotencyConflict     = errors.New("idempotency key reused with different request")
	ErrIdempotencyInProgress   = errors.New("idempotency request is still processing")
	ErrIdempotencyUserRequired = errors.New("idempotency user is required")
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
	if input.Key == "" {
		return nil, false, nil
	}
	if input.UserID <= 0 {
		return nil, false, ErrIdempotencyUserRequired
	}

	existing, err := s.IdempotencyRepo.FindByKeyAndEndpoint(ctx, input.UserID, input.Key, input.Endpoint)
	if err == nil {
		return evaluatePaymentIdempotency(existing, input.RequestHash)
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

	lockedUntil := now.Add(lockTTL)
	record := &domain.IdempotencyKey{
		Key:         input.Key,
		UserID:      input.UserID,
		Endpoint:    input.Endpoint,
		RequestHash: input.RequestHash,
		Status:      domain.IdempotencyStatusProcessing,
		LockedUntil: &lockedUntil,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.IdempotencyRepo.Create(ctx, record); err != nil {
		if !errors.Is(err, repository.ErrUniqueConstraintViolation) {
			return nil, false, err
		}

		existing, findErr := s.IdempotencyRepo.FindByKeyAndEndpoint(ctx, input.UserID, input.Key, input.Endpoint)
		if findErr != nil {
			return nil, false, err
		}
		return evaluatePaymentIdempotency(existing, input.RequestHash)
	}

	return record, false, nil
}

func evaluatePaymentIdempotency(existing *domain.IdempotencyKey, requestHash string) (*domain.IdempotencyKey, bool, error) {
	if existing.RequestHash != requestHash {
		return nil, false, ErrIdempotencyConflict
	}
	if existing.Status == domain.IdempotencyStatusCompleted && existing.ResponseStatus != nil && len(existing.ResponseBody) > 0 {
		return existing, true, nil
	}
	return nil, false, ErrIdempotencyInProgress
}

func (s *BookingService) CompletePaymentIdempotency(ctx context.Context, userID int64, key, endpoint string, responseStatus int, responseBody []byte, now time.Time) error {
	if key == "" {
		return nil
	}
	if userID <= 0 {
		return ErrIdempotencyUserRequired
	}
	if now.IsZero() {
		now = time.Now()
	}
	return s.IdempotencyRepo.Complete(ctx, userID, key, endpoint, responseStatus, responseBody, now)
}
