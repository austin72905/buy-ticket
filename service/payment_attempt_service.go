package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

var ErrPaymentAttemptRepositoryNotConfigured = errors.New("payment attempt repository is not configured")
var ErrPaymentAttemptIdempotencyConflict = errors.New("payment attempt idempotency key reused with different request")
var ErrMockPaymentClientNotConfigured = errors.New("mock payment client is not configured")

type CreatePaymentAttemptInput struct {
	OrderID        int64
	IdempotencyKey string
	Provider       string
	Method         string
	ExpiresAt      *time.Time
	RequestPayload []byte
}

func (s *BookingService) CreatePaymentAttempt(ctx context.Context, input CreatePaymentAttemptInput) (*domain.PaymentAttempt, error) {
	if s.PaymentAttemptRepo == nil {
		return nil, ErrPaymentAttemptRepositoryNotConfigured
	}

	provider := normalizePaymentAttemptProvider(input.Provider)
	if input.IdempotencyKey != "" {
		existingAttempt, err := s.PaymentAttemptRepo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
		if err == nil {
			if existingAttempt.OrderID != input.OrderID || existingAttempt.Method != input.Method || !paymentAttemptProviderMatches(provider, existingAttempt.Provider) {
				return nil, ErrPaymentAttemptIdempotencyConflict
			}
			return existingAttempt, nil
		}
		if !errors.Is(err, repository.ErrPaymentAttemptNotFound) {
			return nil, err
		}
	}

	order, err := s.OrderRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return nil, ErrOrderCannotBePaid
	}

	now := time.Now()
	attempt := &domain.PaymentAttempt{
		OrderID:         order.ID,
		IdempotencyKey:  optionalAttemptString(input.IdempotencyKey),
		Provider:        provider,
		MerchantTradeNo: generateMerchantTradeNo(order.ID, now),
		Method:          input.Method,
		Amount:          order.TotalAmount,
		Status:          domain.PaymentAttemptStatusProcessing,
		RequestPayload:  input.RequestPayload,
		ExpiresAt:       input.ExpiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.PaymentAttemptRepo.Save(ctx, attempt); err != nil {
		return nil, err
	}

	return attempt, nil
}

func (s *BookingService) StartMockPaymentAttempt(ctx context.Context, input CreatePaymentAttemptInput) (*domain.PaymentAttempt, error) {
	if s.PaymentAttemptRepo == nil {
		return nil, ErrPaymentAttemptRepositoryNotConfigured
	}

	provider := normalizePaymentAttemptProvider(input.Provider)
	if input.IdempotencyKey != "" {
		existingAttempt, err := s.PaymentAttemptRepo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
		if err == nil {
			if existingAttempt.OrderID != input.OrderID || existingAttempt.Method != input.Method || !paymentAttemptProviderMatches(provider, existingAttempt.Provider) {
				return nil, ErrPaymentAttemptIdempotencyConflict
			}
			return existingAttempt, nil
		}
		if !errors.Is(err, repository.ErrPaymentAttemptNotFound) {
			return nil, err
		}
	}

	paymentClient := s.MockPaymentClient
	var providerSelectErr error
	if s.MockPaymentRouter != nil {
		preferredProvider := ""
		if provider != "mock_ecpay" {
			preferredProvider = provider
		}
		selectedProvider, selectErr := s.MockPaymentRouter.Select(ctx, preferredProvider)
		if selectErr == nil {
			input.Provider = selectedProvider.Name
			paymentClient = selectedProvider.Client
		}
		if selectErr != nil && !errors.Is(selectErr, ErrPaymentProviderCircuitOpen) {
			return nil, selectErr
		}
		providerSelectErr = selectErr
	}

	attempt, err := s.CreatePaymentAttempt(ctx, input)
	if err != nil {
		return nil, err
	}

	if attempt.Status != domain.PaymentAttemptStatusProcessing || len(attempt.ResponsePayload) > 0 {
		return attempt, nil
	}

	if errors.Is(providerSelectErr, ErrPaymentProviderCircuitOpen) {
		if !attempt.MarkTimeout(ErrPaymentProviderCircuitOpen.Error(), nil, time.Now()) {
			return attempt, nil
		}
		return attempt, s.PaymentAttemptRepo.Save(ctx, attempt)
	}

	if paymentClient == nil || s.MockPaymentCallbackURL == "" {
		if !attempt.MarkTimeout(ErrMockPaymentClientNotConfigured.Error(), nil, time.Now()) {
			return attempt, nil
		}
		return attempt, s.PaymentAttemptRepo.Save(ctx, attempt)
	}

	responsePayload, err := paymentClient.Process(ctx, MockPaymentProcessInput{
		MerchantTradeNo: attempt.MerchantTradeNo,
		Amount:          attempt.Amount,
		PayType:         "ECPAY",
		CallbackURL:     s.MockPaymentCallbackURL,
	})
	if err != nil {
		if !attempt.MarkTimeout(err.Error(), responsePayload, time.Now()) {
			return attempt, nil
		}
		return attempt, s.PaymentAttemptRepo.Save(ctx, attempt)
	}

	attempt.ResponsePayload = responsePayload
	attempt.UpdatedAt = time.Now()
	return attempt, s.PaymentAttemptRepo.Save(ctx, attempt)
}

func (s *BookingService) GetPaymentAttemptByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (*domain.PaymentAttempt, error) {
	if s.PaymentAttemptRepo == nil {
		return nil, ErrPaymentAttemptRepositoryNotConfigured
	}
	return s.PaymentAttemptRepo.FindByMerchantTradeNo(ctx, merchantTradeNo)
}

func normalizePaymentAttemptProvider(provider string) string {
	if provider == "" {
		return "mock_ecpay"
	}
	return provider
}

func paymentAttemptProviderMatches(requestedProvider, existingProvider string) bool {
	if requestedProvider == "" || requestedProvider == "mock_ecpay" {
		return existingProvider == "mock_ecpay" ||
			existingProvider == "mock_ecpay_primary" ||
			existingProvider == "mock_ecpay_backup"
	}

	return existingProvider == requestedProvider
}

func generateMerchantTradeNo(orderID int64, now time.Time) string {
	return fmt.Sprintf("ORD-%d-ATT-%s", orderID, now.UTC().Format("20060102150405-000000000"))
}

func optionalAttemptString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
