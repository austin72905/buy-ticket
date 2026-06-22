package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		return responseBody.Bytes(), fmt.Errorf("mock payment service returned status %d", response.StatusCode)
	}

	return responseBody.Bytes(), nil
}
