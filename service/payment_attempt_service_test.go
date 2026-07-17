package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

var errFakePaymentAttemptNotFound = errors.New("payment attempt not found")

func TestBookingServiceCreatePaymentAttempt(t *testing.T) {
	t.Run("creates processing attempt from pending order", func(t *testing.T) {
		orderRepo := &fakeOrderRepository{
			orders: map[int64]*domain.Order{
				10: {
					ID:          10,
					Status:      domain.OrderStatusPendingPayment,
					TotalAmount: 2800,
				},
			},
		}
		attemptRepo := &fakePaymentAttemptRepository{}
		svc := &BookingService{
			OrderRepo:          orderRepo,
			PaymentAttemptRepo: attemptRepo,
		}

		attempt, err := svc.CreatePaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-key-001",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
			RequestPayload: []byte(`{"order_id":10}`),
		})
		if err != nil {
			t.Fatalf("create payment attempt should not fail: %v", err)
		}
		if attempt.Status != domain.PaymentAttemptStatusProcessing {
			t.Fatalf("expected processing status, got %d", attempt.Status)
		}
		if attempt.Amount != 2800 {
			t.Fatalf("expected order amount 2800, got %d", attempt.Amount)
		}
		if attempt.IdempotencyKey == nil || *attempt.IdempotencyKey != "pay-key-001" {
			t.Fatal("expected idempotency key to be stored")
		}
		if attempt.MerchantTradeNo == "" {
			t.Fatal("expected generated merchant trade no")
		}
	})

	t.Run("rejects non pending order", func(t *testing.T) {
		svc := &BookingService{
			OrderRepo: &fakeOrderRepository{
				orders: map[int64]*domain.Order{
					10: {
						ID:          10,
						Status:      domain.OrderStatusPaid,
						TotalAmount: 2800,
					},
				},
			},
			PaymentAttemptRepo: &fakePaymentAttemptRepository{},
		}

		_, err := svc.CreatePaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID: 10,
			Method:  "credit_card",
		})
		if err != ErrOrderCannotBePaid {
			t.Fatalf("expected ErrOrderCannotBePaid, got %v", err)
		}
	})

	t.Run("same idempotency key returns existing attempt", func(t *testing.T) {
		orderRepo := &fakeOrderRepository{
			orders: map[int64]*domain.Order{
				10: {
					ID:          10,
					Status:      domain.OrderStatusPendingPayment,
					TotalAmount: 2800,
				},
			},
		}
		attemptRepo := &fakePaymentAttemptRepository{}
		svc := &BookingService{
			OrderRepo:          orderRepo,
			PaymentAttemptRepo: attemptRepo,
		}

		firstAttempt, err := svc.CreatePaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-key-001",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
		})
		if err != nil {
			t.Fatalf("first create should not fail: %v", err)
		}

		secondAttempt, err := svc.CreatePaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-key-001",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
		})
		if err != nil {
			t.Fatalf("retry should not fail: %v", err)
		}
		if secondAttempt.ID != firstAttempt.ID {
			t.Fatalf("expected same attempt id, got %d and %d", firstAttempt.ID, secondAttempt.ID)
		}
	})
}

func TestBookingServiceStartMockPaymentAttempt(t *testing.T) {
	t.Run("successful mock provider call keeps attempt processing and stores response", func(t *testing.T) {
		svc := &BookingService{
			OrderRepo: &fakeOrderRepository{
				orders: map[int64]*domain.Order{
					10: {
						ID:          10,
						Status:      domain.OrderStatusPendingPayment,
						TotalAmount: 2800,
					},
				},
			},
			PaymentAttemptRepo:     &fakePaymentAttemptRepository{},
			MockPaymentClient:      &fakeMockPaymentClient{response: []byte(`{"status":"processing"}`)},
			MockPaymentCallbackURL: "http://localhost:8080/payments/provider/ecpay/callback",
		}

		attempt, err := svc.StartMockPaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-start-key-001",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
		})
		if err != nil {
			t.Fatalf("start mock payment should not fail: %v", err)
		}
		if attempt.Status != domain.PaymentAttemptStatusProcessing {
			t.Fatalf("expected processing status, got %d", attempt.Status)
		}
		if string(attempt.ResponsePayload) != `{"status":"processing"}` {
			t.Fatalf("unexpected response payload: %s", string(attempt.ResponsePayload))
		}
	})

	t.Run("mock provider error marks attempt timeout", func(t *testing.T) {
		svc := &BookingService{
			OrderRepo: &fakeOrderRepository{
				orders: map[int64]*domain.Order{
					10: {
						ID:          10,
						Status:      domain.OrderStatusPendingPayment,
						TotalAmount: 2800,
					},
				},
			},
			PaymentAttemptRepo:     &fakePaymentAttemptRepository{},
			MockPaymentClient:      &fakeMockPaymentClient{err: errors.New("connection refused")},
			MockPaymentCallbackURL: "http://localhost:8080/payments/provider/ecpay/callback",
		}

		attempt, err := svc.StartMockPaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:  10,
			Provider: "mock_ecpay",
			Method:   "credit_card",
		})
		if err != nil {
			t.Fatalf("start mock payment should store timeout instead of failing: %v", err)
		}
		if attempt.Status != domain.PaymentAttemptStatusTimeout {
			t.Fatalf("expected timeout status, got %d", attempt.Status)
		}
	})

	t.Run("circuit breaker open marks attempt timeout", func(t *testing.T) {
		mockClient := NewCircuitBreakerMockPaymentClient(&fakeMockPaymentClient{err: errors.New("connection refused")}, PaymentCircuitBreakerConfig{
			Enabled:             true,
			ConsecutiveFailures: 1,
			OpenTimeout:         time.Minute,
			HalfOpenMaxRequests: 1,
		})
		svc := &BookingService{
			OrderRepo: &fakeOrderRepository{
				orders: map[int64]*domain.Order{
					10: {
						ID:          10,
						Status:      domain.OrderStatusPendingPayment,
						TotalAmount: 2800,
					},
				},
			},
			PaymentAttemptRepo:     &fakePaymentAttemptRepository{},
			MockPaymentClient:      mockClient,
			MockPaymentCallbackURL: "http://localhost:8080/payments/provider/ecpay/callback",
		}

		_, err := svc.StartMockPaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-open-key-001",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
		})
		if err != nil {
			t.Fatalf("first provider failure should be stored as timeout: %v", err)
		}

		attempt, err := svc.StartMockPaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-open-key-002",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
		})
		if err != nil {
			t.Fatalf("open circuit should be stored as timeout: %v", err)
		}
		if attempt.Status != domain.PaymentAttemptStatusTimeout {
			t.Fatalf("expected timeout status, got %d", attempt.Status)
		}
		if attempt.FailureReason == nil || *attempt.FailureReason != ErrPaymentProviderCircuitOpen.Error() {
			t.Fatalf("expected circuit open failure reason, got %v", attempt.FailureReason)
		}
	})

	t.Run("new attempt uses backup provider when primary circuit is open", func(t *testing.T) {
		primaryClient := NewCircuitBreakerMockPaymentClient(&fakeMockPaymentClient{err: errors.New("connection refused")}, PaymentCircuitBreakerConfig{
			Enabled:             true,
			ConsecutiveFailures: 1,
			OpenTimeout:         time.Minute,
			HalfOpenMaxRequests: 1,
		})
		backupClient := NewCircuitBreakerMockPaymentClient(&fakeMockPaymentClient{response: []byte(`{"status":"processing","provider":"backup"}`)}, PaymentCircuitBreakerConfig{
			Enabled:             true,
			ConsecutiveFailures: 1,
			OpenTimeout:         time.Minute,
			HalfOpenMaxRequests: 1,
		})
		svc := &BookingService{
			OrderRepo: &fakeOrderRepository{
				orders: map[int64]*domain.Order{
					10: {
						ID:          10,
						Status:      domain.OrderStatusPendingPayment,
						TotalAmount: 2800,
					},
				},
			},
			PaymentAttemptRepo: &fakePaymentAttemptRepository{},
			MockPaymentRouter: NewMockPaymentProviderRouter([]MockPaymentProvider{
				{Name: "mock_ecpay_primary", Client: primaryClient},
				{Name: "mock_ecpay_backup", Client: backupClient},
			}),
			MockPaymentCallbackURL: "http://localhost:8080/payments/provider/ecpay/callback",
		}

		firstAttempt, err := svc.StartMockPaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-router-key-001",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
		})
		if err != nil {
			t.Fatalf("first provider failure should be stored as timeout: %v", err)
		}
		if firstAttempt.Provider != "mock_ecpay_primary" {
			t.Fatalf("expected primary provider, got %s", firstAttempt.Provider)
		}
		if firstAttempt.Status != domain.PaymentAttemptStatusTimeout {
			t.Fatalf("expected timeout status, got %d", firstAttempt.Status)
		}

		secondAttempt, err := svc.StartMockPaymentAttempt(context.Background(), CreatePaymentAttemptInput{
			OrderID:        10,
			IdempotencyKey: "pay-router-key-002",
			Provider:       "mock_ecpay",
			Method:         "credit_card",
		})
		if err != nil {
			t.Fatalf("backup provider attempt should not fail: %v", err)
		}
		if secondAttempt.Provider != "mock_ecpay_backup" {
			t.Fatalf("expected backup provider, got %s", secondAttempt.Provider)
		}
		if secondAttempt.Status != domain.PaymentAttemptStatusProcessing {
			t.Fatalf("expected processing status, got %d", secondAttempt.Status)
		}
		if string(secondAttempt.ResponsePayload) != `{"status":"processing","provider":"backup"}` {
			t.Fatalf("unexpected response payload: %s", string(secondAttempt.ResponsePayload))
		}
	})
}

type fakePaymentAttemptRepository struct {
	nextID   int64
	attempts map[int64]*domain.PaymentAttempt
}

type fakeMockPaymentClient struct {
	response []byte
	err      error
}

func (f *fakeMockPaymentClient) Process(ctx context.Context, input MockPaymentProcessInput) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f *fakePaymentAttemptRepository) FindByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (*domain.PaymentAttempt, error) {
	for _, attempt := range f.attempts {
		if attempt.MerchantTradeNo == merchantTradeNo {
			cloned := *attempt
			return &cloned, nil
		}
	}
	return nil, errFakePaymentAttemptNotFound
}

func (f *fakePaymentAttemptRepository) FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.PaymentAttempt, error) {
	for _, attempt := range f.attempts {
		if attempt.IdempotencyKey != nil && *attempt.IdempotencyKey == idempotencyKey {
			cloned := *attempt
			return &cloned, nil
		}
	}
	return nil, repository.ErrPaymentAttemptNotFound
}

func (f *fakePaymentAttemptRepository) ListByOrderID(ctx context.Context, orderID int64) ([]domain.PaymentAttempt, error) {
	attempts := make([]domain.PaymentAttempt, 0)
	for _, attempt := range f.attempts {
		if attempt.OrderID == orderID {
			attempts = append(attempts, *attempt)
		}
	}
	return attempts, nil
}

func (f *fakePaymentAttemptRepository) ListReconcileCandidates(ctx context.Context, now time.Time, cutoff time.Time, limit int, maxAttempts int) ([]domain.PaymentAttempt, error) {
	attempts := make([]domain.PaymentAttempt, 0)
	for _, attempt := range f.attempts {
		if attempt.Status != domain.PaymentAttemptStatusProcessing && attempt.Status != domain.PaymentAttemptStatusTimeout {
			continue
		}
		if attempt.CreatedAt.After(cutoff) {
			continue
		}
		attempts = append(attempts, *attempt)
	}
	return attempts, nil
}

func (f *fakePaymentAttemptRepository) Save(ctx context.Context, attempt *domain.PaymentAttempt) error {
	if f.attempts == nil {
		f.attempts = map[int64]*domain.PaymentAttempt{}
	}
	if f.nextID == 0 {
		f.nextID = 1
	}
	if attempt.ID == 0 {
		attempt.ID = f.nextID
		f.nextID++
	}
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now()
	}
	if attempt.UpdatedAt.IsZero() {
		attempt.UpdatedAt = attempt.CreatedAt
	}
	cloned := *attempt
	f.attempts[attempt.ID] = &cloned
	return nil
}
