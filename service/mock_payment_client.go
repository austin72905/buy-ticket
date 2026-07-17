package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sony/gobreaker/v2"
)

type MockPaymentClient interface {
	Process(ctx context.Context, input MockPaymentProcessInput) ([]byte, error)
}

type MockPaymentQueryClient interface {
	Query(ctx context.Context, input MockPaymentQueryInput) (*MockPaymentQueryResult, []byte, error)
}

type MockPaymentProcessInput struct {
	MerchantTradeNo string
	Amount          int64
	PayType         string
	CallbackURL     string
}

type MockPaymentQueryInput struct {
	MerchantTradeNo string
}

type MockPaymentQueryStatus string

const (
	MockPaymentQueryStatusUnknown MockPaymentQueryStatus = "UNKNOWN"
	MockPaymentQueryStatusPending MockPaymentQueryStatus = "PENDING"
	MockPaymentQueryStatusSuccess MockPaymentQueryStatus = "SUCCESS"
	MockPaymentQueryStatusFailed  MockPaymentQueryStatus = "FAILED"
)

type MockPaymentQueryResult struct {
	MerchantTradeNo string
	ProviderTradeNo string
	Amount          int64
	PaidAt          time.Time
	Method          string
	Status          MockPaymentQueryStatus
	FailureReason   string
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

func (c *HTTPMockPaymentClient) Query(ctx context.Context, input MockPaymentQueryInput) (*MockPaymentQueryResult, []byte, error) {
	if c == nil || c.BaseURL == "" {
		return nil, nil, ErrMockPaymentClientNotConfigured
	}
	if input.MerchantTradeNo == "" {
		return nil, nil, ErrInvalidPaymentCallback
	}

	query := url.Values{}
	query.Set("recordNo", input.MerchantTradeNo)
	query.Set("merchantTradeNo", input.MerchantTradeNo)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/payment/query?"+query.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	responseBody := bytes.Buffer{}
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		return nil, nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, responseBody.Bytes(), &MockPaymentHTTPStatusError{
			StatusCode: response.StatusCode,
			Body:       responseBody.Bytes(),
		}
	}

	result, err := parseMockPaymentQueryResult(responseBody.Bytes(), input.MerchantTradeNo)
	if err != nil {
		return nil, responseBody.Bytes(), err
	}
	return result, responseBody.Bytes(), nil
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

func (c *CircuitBreakerMockPaymentClient) Query(ctx context.Context, input MockPaymentQueryInput) (*MockPaymentQueryResult, []byte, error) {
	queryClient, ok := c.client.(MockPaymentQueryClient)
	if !ok {
		return nil, nil, ErrPaymentProviderNotConfigured
	}
	if c.breaker.State() == gobreaker.StateOpen {
		return nil, nil, ErrPaymentProviderCircuitOpen
	}
	return queryClient.Query(ctx, input)
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

func (r *MockPaymentProviderRouter) SelectQueryClient(ctx context.Context, providerName string) (MockPaymentProvider, MockPaymentQueryClient, error) {
	provider, err := r.Select(ctx, providerName)
	if err != nil {
		return MockPaymentProvider{}, nil, err
	}
	queryClient, ok := provider.Client.(MockPaymentQueryClient)
	if !ok {
		return MockPaymentProvider{}, nil, ErrPaymentProviderNotConfigured
	}
	return provider, queryClient, nil
}

func mockPaymentProviderAvailable(provider MockPaymentProvider) bool {
	if availability, ok := provider.Client.(interface{ Available() bool }); ok {
		return availability.Available()
	}
	return true
}

func parseMockPaymentQueryResult(body []byte, fallbackMerchantTradeNo string) (*MockPaymentQueryResult, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if data, ok := raw["data"].(map[string]interface{}); ok {
		raw = data
	}

	status := normalizeMockPaymentQueryStatus(firstString(raw, "status", "paymentStatus", "tradeStatus", "returnStatus", "rtnCode", "RtnCode"))
	result := &MockPaymentQueryResult{
		MerchantTradeNo: firstNonEmpty(firstString(raw, "merchantTradeNo", "MerchantTradeNo", "recordNo"), fallbackMerchantTradeNo),
		ProviderTradeNo: firstString(raw, "tradeNo", "TradeNo", "providerTradeNo", "paymentNo"),
		Method:          firstNonEmpty(firstString(raw, "paymentType", "PaymentType", "method", "payType"), "ecpay"),
		Status:          status,
		FailureReason:   firstString(raw, "rtnMsg", "RtnMsg", "message", "failureReason"),
	}
	if amount, ok := firstInt64(raw, "amount", "tradeAmt", "TradeAmt"); ok {
		result.Amount = amount
	}
	if paidAt, ok := firstTime(raw, "paidAt", "paymentDate", "PaymentDate", "tradeDate", "TradeDate"); ok {
		result.PaidAt = paidAt
	}
	return result, nil
}

func normalizeMockPaymentQueryStatus(value string) MockPaymentQueryStatus {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "1", "SUCCESS", "SUCCEEDED", "PAID", "OK":
		return MockPaymentQueryStatusSuccess
	case "0", "PENDING", "PROCESSING":
		return MockPaymentQueryStatusPending
	case "2", "FAILED", "FAIL", "CANCELLED", "CANCELED":
		return MockPaymentQueryStatusFailed
	default:
		return MockPaymentQueryStatusUnknown
	}
}

func firstString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case json.Number:
			return typed.String()
		case float64:
			return strconv.FormatInt(int64(typed), 10)
		case int64:
			return strconv.FormatInt(typed, 10)
		case int:
			return strconv.Itoa(typed)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstInt64(values map[string]interface{}, keys ...string) (int64, bool) {
	text := firstString(values, keys...)
	if text == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(text, 10, 64)
	return value, err == nil
}

func firstTime(values map[string]interface{}, keys ...string) (time.Time, bool) {
	text := firstString(values, keys...)
	if text == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006/01/02 15:04:05", "2006-01-02 15:04:05"} {
		parsed, err := time.ParseInLocation(layout, text, time.Local)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
