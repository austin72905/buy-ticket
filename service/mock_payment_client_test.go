package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerMockPaymentClient(t *testing.T) {
	t.Run("opens after configured consecutive failures and fails fast", func(t *testing.T) {
		client := &countingMockPaymentClient{err: errors.New("connection refused")}
		breakerClient := NewCircuitBreakerMockPaymentClient(client, PaymentCircuitBreakerConfig{
			Enabled:             true,
			ConsecutiveFailures: 2,
			OpenTimeout:         time.Minute,
			HalfOpenMaxRequests: 1,
		})

		for i := 0; i < 2; i++ {
			if _, err := breakerClient.Process(context.Background(), MockPaymentProcessInput{}); err == nil {
				t.Fatal("expected provider error")
			}
		}

		_, err := breakerClient.Process(context.Background(), MockPaymentProcessInput{})
		if !errors.Is(err, ErrPaymentProviderCircuitOpen) {
			t.Fatalf("expected circuit open error, got %v", err)
		}
		if client.calls != 2 {
			t.Fatalf("expected fail fast without provider call, got %d calls", client.calls)
		}
	})

	t.Run("excludes 4xx provider errors from breaker counts", func(t *testing.T) {
		client := &countingMockPaymentClient{
			err: &MockPaymentHTTPStatusError{StatusCode: 400},
		}
		breakerClient := NewCircuitBreakerMockPaymentClient(client, PaymentCircuitBreakerConfig{
			Enabled:             true,
			ConsecutiveFailures: 2,
			OpenTimeout:         time.Minute,
			HalfOpenMaxRequests: 1,
		})

		for i := 0; i < 3; i++ {
			var statusErr *MockPaymentHTTPStatusError
			_, err := breakerClient.Process(context.Background(), MockPaymentProcessInput{})
			if !errors.As(err, &statusErr) || statusErr.StatusCode != 400 {
				t.Fatalf("expected 400 status error, got %v", err)
			}
		}

		if client.calls != 3 {
			t.Fatalf("expected 4xx calls to keep reaching provider, got %d calls", client.calls)
		}
	})

	t.Run("counts 5xx provider errors as breaker failures", func(t *testing.T) {
		client := &countingMockPaymentClient{
			err: &MockPaymentHTTPStatusError{StatusCode: 500},
		}
		breakerClient := NewCircuitBreakerMockPaymentClient(client, PaymentCircuitBreakerConfig{
			Enabled:             true,
			ConsecutiveFailures: 2,
			OpenTimeout:         time.Minute,
			HalfOpenMaxRequests: 1,
		})

		for i := 0; i < 2; i++ {
			if _, err := breakerClient.Process(context.Background(), MockPaymentProcessInput{}); err == nil {
				t.Fatal("expected 5xx status error")
			}
		}

		_, err := breakerClient.Process(context.Background(), MockPaymentProcessInput{})
		if !errors.Is(err, ErrPaymentProviderCircuitOpen) {
			t.Fatalf("expected circuit open error, got %v", err)
		}
	})
}

type countingMockPaymentClient struct {
	calls    int
	response []byte
	err      error
}

func (c *countingMockPaymentClient) Process(ctx context.Context, input MockPaymentProcessInput) ([]byte, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	return c.response, nil
}
