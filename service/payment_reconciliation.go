package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"buy-ticket/domain"
)

type ReconcilePaymentAttemptsInput struct {
	Now         time.Time
	Delay       time.Duration
	RetryAfter  time.Duration
	Limit       int
	MaxAttempts int
}

// 定期找太久沒完成的 payment_attempts
/*
payment_attempt = PROCESSING / TIMEOUT
order = PENDING_PAYMENT
reservation = HOLDING
*/
func (s *BookingService) ReconcilePaymentAttempts(ctx context.Context, input ReconcilePaymentAttemptsInput) (int, error) {
	if s.PaymentAttemptRepo == nil {
		return 0, ErrPaymentAttemptRepositoryNotConfigured
	}
	if s.MockPaymentRouter == nil && s.MockPaymentClient == nil {
		return 0, ErrPaymentProviderNotConfigured
	}

	// 補預設值，手動測試才會用到
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	// 預設等 2 分鐘，避免 callback 還沒來就太早主動查
	delay := input.Delay
	if delay <= 0 {
		delay = 2 * time.Minute
	}
	// 預設 30 秒，避免一直狂打 payment service。
	retryAfter := input.RetryAfter
	if retryAfter <= 0 {
		retryAfter = 30 * time.Second
	}
	// 預設 100，避免一次撈太多打爆 DB / payment service。
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	// 預設 5 次，超過就不再查，避免死循環。
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	candidates, err := s.PaymentAttemptRepo.ListReconcileCandidates(ctx, now, now.Add(-delay), limit, maxAttempts)
	if err != nil {
		return 0, err
	}

	reconciled := 0
	for index := range candidates {
		attempt := candidates[index]
		completed, err := s.reconcilePaymentAttempt(ctx, &attempt, now, retryAfter)
		if err != nil {
			return reconciled, err
		}
		if completed {
			reconciled++
		}
	}

	return reconciled, nil
}

func (s *BookingService) reconcilePaymentAttempt(ctx context.Context, attempt *domain.PaymentAttempt, now time.Time, retryAfter time.Duration) (bool, error) {
	queryClient, err := s.paymentQueryClientForAttempt(ctx, attempt)
	if err != nil {
		s.markPaymentAttemptReconcileFailure(ctx, attempt, err, now, retryAfter)
		return false, nil
	}

	result, payload, err := queryClient.Query(ctx, MockPaymentQueryInput{MerchantTradeNo: attempt.MerchantTradeNo})
	if err != nil {
		s.markPaymentAttemptReconcileFailure(ctx, attempt, err, now, retryAfter)
		return false, nil
	}
	if result == nil {
		s.markPaymentAttemptReconcileFailure(ctx, attempt, errors.New("payment query returned empty result"), now, retryAfter)
		return false, nil
	}

	switch result.Status {
	case MockPaymentQueryStatusSuccess:
		paidAt := result.PaidAt
		if paidAt.IsZero() {
			paidAt = now
		}
		method := normalizeECPayPaymentMethod(result.Method)
		if method == "ecpay_ecpay" {
			method = "ecpay"
		}
		providerTradeNo := result.ProviderTradeNo
		if providerTradeNo == "" {
			providerTradeNo = attempt.MerchantTradeNo
		}
		amount := result.Amount
		if amount == 0 {
			amount = attempt.Amount
		}
		if len(payload) == 0 {
			payload, _ = json.Marshal(result)
		}
		if err := s.completeSuccessfulPaymentAttempt(ctx, attempt, providerTradeNo, amount, paidAt, method, payload); err != nil {
			s.markPaymentAttemptReconcileFailure(ctx, attempt, err, now, retryAfter)
			return false, nil
		}
		return true, nil
	case MockPaymentQueryStatusFailed:
		reason := result.FailureReason
		if reason == "" {
			reason = "provider payment failed"
		}
		if !attempt.MarkFailed(reason, payload, now) {
			return false, nil
		}
		return false, s.PaymentAttemptRepo.Save(ctx, attempt)
	case MockPaymentQueryStatusPending, MockPaymentQueryStatusUnknown:
		s.markPaymentAttemptReconcileFailure(ctx, attempt, fmt.Errorf("provider payment status is %s", result.Status), now, retryAfter)
		return false, nil
	default:
		s.markPaymentAttemptReconcileFailure(ctx, attempt, fmt.Errorf("provider payment status is %s", result.Status), now, retryAfter)
		return false, nil
	}
}

func (s *BookingService) paymentQueryClientForAttempt(ctx context.Context, attempt *domain.PaymentAttempt) (MockPaymentQueryClient, error) {
	if s.MockPaymentRouter != nil {
		_, queryClient, err := s.MockPaymentRouter.SelectQueryClient(ctx, attempt.Provider)
		return queryClient, err
	}
	queryClient, ok := s.MockPaymentClient.(MockPaymentQueryClient)
	if !ok {
		return nil, ErrPaymentProviderNotConfigured
	}
	return queryClient, nil
}

func (s *BookingService) markPaymentAttemptReconcileFailure(ctx context.Context, attempt *domain.PaymentAttempt, err error, now time.Time, retryAfter time.Duration) {
	next := now.Add(retryAfter)
	attempt.MarkReconcileFailure(err.Error(), &next, now)
	_ = s.PaymentAttemptRepo.Save(ctx, attempt)
}
