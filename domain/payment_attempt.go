package domain

import "time"

type PaymentAttemptStatus int8

const (
	PaymentAttemptStatusProcessing PaymentAttemptStatus = iota + 1
	PaymentAttemptStatusSucceeded
	PaymentAttemptStatusFailed
	PaymentAttemptStatusTimeout
	PaymentAttemptStatusCancelled
)

type PaymentAttempt struct {
	ID                 int64
	OrderID            int64
	PaymentID          *int64
	IdempotencyKey     *string
	Provider           string
	MerchantTradeNo    string
	ProviderTradeNo    *string
	Method             string
	Amount             int64
	Status             PaymentAttemptStatus
	RequestPayload     []byte
	ResponsePayload    []byte
	CallbackPayload    []byte
	FailureReason      *string
	ExpiresAt          *time.Time
	SucceededAt        *time.Time
	FailedAt           *time.Time
	ReconcileAttempts  int
	NextReconcileAt    *time.Time
	LastReconcileError *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (a *PaymentAttempt) MarkSucceeded(paymentID int64, providerTradeNo string, callbackPayload []byte, now time.Time) bool {
	if a.Status != PaymentAttemptStatusProcessing && a.Status != PaymentAttemptStatusTimeout {
		return false
	}

	a.PaymentID = &paymentID
	a.ProviderTradeNo = optionalString(providerTradeNo)
	a.CallbackPayload = callbackPayload
	a.Status = PaymentAttemptStatusSucceeded
	a.SucceededAt = &now
	a.NextReconcileAt = nil
	a.LastReconcileError = nil
	a.UpdatedAt = now
	return true
}

func (a *PaymentAttempt) MarkFailed(reason string, responsePayload []byte, now time.Time) bool {
	if a.Status != PaymentAttemptStatusProcessing {
		return false
	}

	a.FailureReason = optionalString(reason)
	a.ResponsePayload = responsePayload
	a.Status = PaymentAttemptStatusFailed
	a.FailedAt = &now
	a.NextReconcileAt = nil
	a.LastReconcileError = nil
	a.UpdatedAt = now
	return true
}

func (a *PaymentAttempt) MarkTimeout(reason string, responsePayload []byte, now time.Time) bool {
	if a.Status != PaymentAttemptStatusProcessing {
		return false
	}

	a.FailureReason = optionalString(reason)
	a.ResponsePayload = responsePayload
	a.Status = PaymentAttemptStatusTimeout
	a.FailedAt = &now
	a.UpdatedAt = now
	return true
}

func (a *PaymentAttempt) MarkReconcileFailure(reason string, nextReconcileAt *time.Time, now time.Time) {
	a.ReconcileAttempts++
	a.LastReconcileError = optionalString(reason)
	a.NextReconcileAt = nextReconcileAt
	a.UpdatedAt = now
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
