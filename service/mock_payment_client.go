package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sony/gobreaker/v2"
)

type MockPaymentClient interface {
	Process(ctx context.Context, input MockPaymentProcessInput) ([]byte, error)
}

type MockPaymentProcessInput struct {
	MerchantTradeNo string
	Amount          int64
	PayType         string
	CallbackURL     string
}

var ErrPaymentProviderCircuitOpen = errors.New("payment provider circuit breaker is open")
var ErrPaymentProviderNotConfigured = errors.New("payment provider is not configured")

type MockPaymentHTTPStatusError struct {
	StatusCode int
	Body       []byte
}

func (e *MockPaymentHTTPStatusError) Error() string {
	return fmt.Sprintf("mock payment service returned status %d", e.StatusCode)
}

type HTTPMockPaymentClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewHTTPMockPaymentClient(baseURL string, timeout time.Duration) *HTTPMockPaymentClient {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	return &HTTPMockPaymentClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HTTPMockPaymentClient) Process(ctx context.Context, input MockPaymentProcessInput) ([]byte, error) {
	if c == nil || c.BaseURL == "" {
		return nil, ErrMockPaymentClientNotConfigured
	}

	payload := map[string]string{
		"recordNo":    input.MerchantTradeNo,
		"amount":      strconv.FormatInt(input.Amount, 10),
		"payType":     input.PayType,
		"callbackUrl": input.CallbackURL,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/payment/process", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody := bytes.Buffer{}
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return responseBody.Bytes(), &MockPaymentHTTPStatusError{
			StatusCode: response.StatusCode,
			Body:       responseBody.Bytes(),
		}
	}

	return responseBody.Bytes(), nil
}

type PaymentCircuitBreakerConfig struct {
	Enabled             bool
	ConsecutiveFailures uint32
	OpenTimeout         time.Duration
	HalfOpenMaxRequests uint32
}

type CircuitBreakerMockPaymentClient struct {
	client  MockPaymentClient
	breaker *gobreaker.CircuitBreaker[[]byte]
}

func NewCircuitBreakerMockPaymentClient(client MockPaymentClient, config PaymentCircuitBreakerConfig) MockPaymentClient {
	if client == nil || !config.Enabled {
		return client
	}
	if config.ConsecutiveFailures == 0 {
		config.ConsecutiveFailures = 5
	}
	if config.OpenTimeout <= 0 {
		config.OpenTimeout = 30 * time.Second
	}
	if config.HalfOpenMaxRequests == 0 {
		config.HalfOpenMaxRequests = 1
	}

	breaker := gobreaker.NewCircuitBreaker[[]byte](gobreaker.Settings{
		Name:        "mock_payment",
		MaxRequests: config.HalfOpenMaxRequests,
		Timeout:     config.OpenTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= config.ConsecutiveFailures
		},
		IsExcluded: func(err error) bool {
			var statusErr *MockPaymentHTTPStatusError
			return errors.As(err, &statusErr) && statusErr.StatusCode >= 400 && statusErr.StatusCode < 500
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("payment circuit breaker %s changed from %s to %s", name, from, to)
		},
	})

	return &CircuitBreakerMockPaymentClient{
		client:  client,
		breaker: breaker,
	}
}

func (c *CircuitBreakerMockPaymentClient) Process(ctx context.Context, input MockPaymentProcessInput) ([]byte, error) {
	response, err := c.breaker.Execute(func() ([]byte, error) {
		return c.client.Process(ctx, input)
	})
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return nil, ErrPaymentProviderCircuitOpen
	}
	return response, err
}

func (c *CircuitBreakerMockPaymentClient) Available() bool {
	return c.breaker.State() != gobreaker.StateOpen
}

type MockPaymentProvider struct {
	Name   string
	Client MockPaymentClient
}

type MockPaymentProviderRouter struct {
	providers []MockPaymentProvider
}

func NewMockPaymentProviderRouter(providers []MockPaymentProvider) *MockPaymentProviderRouter {
	enabledProviders := make([]MockPaymentProvider, 0, len(providers))
	for _, provider := range providers {
		if provider.Name == "" || provider.Client == nil {
			continue
		}
		enabledProviders = append(enabledProviders, provider)
	}

	return &MockPaymentProviderRouter{providers: enabledProviders}
}

func (r *MockPaymentProviderRouter) Select(ctx context.Context, preferredProvider string) (MockPaymentProvider, error) {
	_ = ctx
	if r == nil || len(r.providers) == 0 {
		return MockPaymentProvider{}, ErrPaymentProviderNotConfigured
	}

	if preferredProvider != "" {
		for _, provider := range r.providers {
			if provider.Name == preferredProvider {
				if !mockPaymentProviderAvailable(provider) {
					return MockPaymentProvider{}, ErrPaymentProviderCircuitOpen
				}
				return provider, nil
			}
		}
	}

	for _, provider := range r.providers {
		if !mockPaymentProviderAvailable(provider) {
			continue
		}
		return provider, nil
	}

	return MockPaymentProvider{}, ErrPaymentProviderCircuitOpen
}

func (r *MockPaymentProviderRouter) PrimaryClient() MockPaymentClient {
	if r == nil || len(r.providers) == 0 {
		return nil
	}
	return r.providers[0].Client
}

func mockPaymentProviderAvailable(provider MockPaymentProvider) bool {
	if availability, ok := provider.Client.(interface{ Available() bool }); ok {
		return availability.Available()
	}
	return true
}
