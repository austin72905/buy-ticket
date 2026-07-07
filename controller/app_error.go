package controller

import (
	"errors"
	"net/http"

	"buy-ticket/repository"
	"buy-ticket/service"
)

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	RequestID  string
}

func newAppError(fallbackStatus int, err error) AppError {
	if err == nil {
		return AppError{
			Code:       errorCodeForStatus(fallbackStatus),
			Message:    http.StatusText(fallbackStatus),
			HTTPStatus: fallbackStatus,
		}
	}

	return AppError{
		Code:       errorCodeForError(fallbackStatus, err),
		Message:    err.Error(),
		HTTPStatus: fallbackStatus,
	}
}

func errorCodeForError(fallbackStatus int, err error) string {
	switch {
	case errors.Is(err, service.ErrPurchaseTokenRequired):
		return "PURCHASE_TOKEN_REQUIRED"
	case errors.Is(err, service.ErrPurchaseTokenNotFound):
		return "PURCHASE_TOKEN_NOT_FOUND"
	case errors.Is(err, service.ErrPurchaseTokenExpired):
		return "PURCHASE_TOKEN_EXPIRED"
	case errors.Is(err, service.ErrPurchaseTokenUsed):
		return "PURCHASE_TOKEN_USED"
	case errors.Is(err, service.ErrPurchaseTokenMismatch):
		return "PURCHASE_TOKEN_MISMATCH"
	case errors.Is(err, service.ErrQueueTokenNotFound):
		return "QUEUE_TOKEN_NOT_FOUND"
	case errors.Is(err, service.ErrUserAlreadyJoinedQueue):
		return "USER_ALREADY_JOINED_QUEUE"
	case errors.Is(err, service.ErrEventNotOnSale):
		return "EVENT_NOT_ON_SALE"
	case errors.Is(err, service.ErrSectionNotReservable):
		return "SECTION_NOT_RESERVABLE"
	case errors.Is(err, service.ErrActiveReservationExists):
		return "ACTIVE_RESERVATION_EXISTS"
	case errors.Is(err, service.ErrReservationNotActive):
		return "RESERVATION_NOT_ACTIVE"
	case errors.Is(err, service.ErrReservationAlreadyUsed):
		return "RESERVATION_ALREADY_USED"
	case errors.Is(err, service.ErrReservationCannotClose):
		return "RESERVATION_CANNOT_CLOSE"
	case errors.Is(err, service.ErrOrderCannotBePaid):
		return "ORDER_CANNOT_BE_PAID"
	case errors.Is(err, service.ErrOrderCannotExpire):
		return "ORDER_CANNOT_EXPIRE"
	case errors.Is(err, service.ErrPaymentAmountMismatch):
		return "PAYMENT_AMOUNT_MISMATCH"
	case errors.Is(err, service.ErrUserEmailExists):
		return "USER_EMAIL_EXISTS"
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrInvalidAdminCredentials):
		return "INVALID_CREDENTIALS"
	case errors.Is(err, service.ErrInvalidRegisterInput):
		return "INVALID_REGISTER_INPUT"
	case errors.Is(err, service.ErrInvalidLoginInput):
		return "INVALID_LOGIN_INPUT"
	case errors.Is(err, service.ErrUnauthorized):
		return errorCodeForStatus(fallbackStatus)
	case errors.Is(err, service.ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, service.ErrAdminDisabled):
		return "ADMIN_DISABLED"
	case errors.Is(err, service.ErrInvalidAdminRevealReason):
		return "INVALID_ADMIN_REVEAL_REASON"
	case errors.Is(err, service.ErrInvalidAdminInput):
		return "INVALID_ADMIN_INPUT"
	case errors.Is(err, service.ErrIdempotencyConflict):
		return "IDEMPOTENCY_CONFLICT"
	case errors.Is(err, service.ErrIdempotencyInProgress):
		return "IDEMPOTENCY_IN_PROGRESS"
	case errors.Is(err, service.ErrPaymentAttemptIdempotencyConflict):
		return "PAYMENT_ATTEMPT_IDEMPOTENCY_CONFLICT"
	case errors.Is(err, service.ErrPaymentProviderCircuitOpen):
		return "PAYMENT_PROVIDER_CIRCUIT_OPEN"
	case errors.Is(err, service.ErrPaymentProviderNotConfigured), errors.Is(err, service.ErrMockPaymentClientNotConfigured):
		return "PAYMENT_PROVIDER_NOT_CONFIGURED"
	case errors.Is(err, service.ErrPaymentAttemptRepositoryNotConfigured):
		return "PAYMENT_ATTEMPT_REPOSITORY_NOT_CONFIGURED"
	case errors.Is(err, service.ErrInvalidPaymentCallback):
		return "INVALID_PAYMENT_CALLBACK"
	case errors.Is(err, service.ErrInvalidPaymentSignature):
		return "INVALID_PAYMENT_SIGNATURE"
	case errors.Is(err, service.ErrInsufficientStock):
		return "SOLD_OUT"
	case errors.Is(err, service.ErrSessionNotFound):
		return "SESSION_NOT_FOUND"
	case errors.Is(err, repository.ErrUserNotFound):
		return "USER_NOT_FOUND"
	case errors.Is(err, repository.ErrEventNotFound):
		return "EVENT_NOT_FOUND"
	case errors.Is(err, repository.ErrSectionNotFound):
		return "SECTION_NOT_FOUND"
	case errors.Is(err, repository.ErrReservationNotFound):
		return "RESERVATION_NOT_FOUND"
	case errors.Is(err, repository.ErrOrderNotFound):
		return "ORDER_NOT_FOUND"
	case errors.Is(err, repository.ErrPaymentNotFound):
		return "PAYMENT_NOT_FOUND"
	case errors.Is(err, repository.ErrPaymentAttemptNotFound):
		return "PAYMENT_ATTEMPT_NOT_FOUND"
	case errors.Is(err, repository.ErrIdempotencyKeyNotFound):
		return "IDEMPOTENCY_KEY_NOT_FOUND"
	case errors.Is(err, repository.ErrAdminUserNotFound):
		return "ADMIN_USER_NOT_FOUND"
	case errors.Is(err, repository.ErrOrganizerNotFound):
		return "ORGANIZER_NOT_FOUND"
	default:
		return errorCodeForStatus(fallbackStatus)
	}
}

func errorCodeForStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case http.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	default:
		return "ERROR"
	}
}
