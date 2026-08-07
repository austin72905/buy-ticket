package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

func TestReconcilePaymentAttemptsLimitsProviderConcurrency(t *testing.T) {
	const workerCount = 5

	now := time.Now()
	attempts := make([]*domain.PaymentAttempt, 0, 10)
	for index := 0; index < 10; index++ {
		attempts = append(attempts, &domain.PaymentAttempt{
			ID:              int64(index + 1),
			OrderID:         int64(index + 1),
			Provider:        "mock_ecpay",
			MerchantTradeNo: "trade-" + time.Unix(int64(index), 0).Format("150405"),
			Method:          "credit_card",
			Amount:          100,
			Status:          domain.PaymentAttemptStatusProcessing,
			CreatedAt:       now.Add(-3 * time.Minute),
			UpdatedAt:       now.Add(-3 * time.Minute),
		})
	}

	client := &blockingPaymentQueryClient{
		entered: make(chan struct{}, len(attempts)),
		release: make(chan struct{}),
	}
	svc := &BookingService{
		PaymentAttemptRepo: repository.NewMemoryPaymentAttemptRepository(attempts),
		MockPaymentClient:  client,
	}

	done := make(chan error, 1)
	go func() {
		_, err := svc.ReconcilePaymentAttempts(context.Background(), ReconcilePaymentAttemptsInput{
			Now:         now,
			Delay:       2 * time.Minute,
			RetryAfter:  30 * time.Second,
			Limit:       len(attempts),
			Workers:     workerCount,
			MaxAttempts: 5,
		})
		done <- err
	}()

	for range workerCount {
		select {
		case <-client.entered:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for reconciliation workers")
		}
	}

	select {
	case <-client.entered:
		t.Fatal("provider concurrency exceeded configured worker count")
	case <-time.After(50 * time.Millisecond):
	}

	close(client.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reconciliation failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reconciliation to finish")
	}

	if maximum := client.maximum.Load(); maximum != workerCount {
		t.Fatalf("expected maximum concurrency %d, got %d", workerCount, maximum)
	}
}

type blockingPaymentQueryClient struct {
	current atomic.Int32
	maximum atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (c *blockingPaymentQueryClient) Process(context.Context, MockPaymentProcessInput) ([]byte, error) {
	return nil, nil
}

func (c *blockingPaymentQueryClient) Query(ctx context.Context, _ MockPaymentQueryInput) (*MockPaymentQueryResult, []byte, error) {
	current := c.current.Add(1)
	defer c.current.Add(-1)
	for {
		maximum := c.maximum.Load()
		if current <= maximum || c.maximum.CompareAndSwap(maximum, current) {
			break
		}
	}

	c.entered <- struct{}{}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-c.release:
		return &MockPaymentQueryResult{Status: MockPaymentQueryStatusPending}, nil, nil
	}
}
