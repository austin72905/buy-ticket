package controller

import (
	"buy-ticket/domain"
	"buy-ticket/service"
)

func eventStatusText(status domain.EventStatus) string {
	switch status {
	case domain.EventStatusDraft:
		return "DRAFT"
	case domain.EventStatusPublished:
		return "PUBLISHED"
	case domain.EventStatusOnSale:
		return "ON_SALE"
	case domain.EventStatusEnded:
		return "ENDED"
	default:
		return "UNKNOWN"
	}
}

func sectionStatusText(status domain.SectionStatus) string {
	switch status {
	case domain.SectionStatusActive:
		return "ACTIVE"
	case domain.SectionStatusInactive:
		return "INACTIVE"
	case domain.SectionStatusSoldOut:
		return "SOLD_OUT"
	default:
		return "UNKNOWN"
	}
}

func reservationStatusText(status domain.ReservationStatus) string {
	switch status {
	case domain.ReservationStatusHolding:
		return "HOLDING"
	case domain.ReservationStatusConfirmed:
		return "CONFIRMED"
	case domain.ReservationStatusExpired:
		return "EXPIRED"
	case domain.ReservationStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

func orderStatusText(status domain.OrderStatus) string {
	switch status {
	case domain.OrderStatusPendingPayment:
		return "PENDING_PAYMENT"
	case domain.OrderStatusPaid:
		return "PAID"
	case domain.OrderStatusExpired:
		return "EXPIRED"
	case domain.OrderStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

func paymentStatusText(status domain.PaymentStatus) string {
	switch status {
	case domain.PaymentStatusPending:
		return "PENDING"
	case domain.PaymentStatusPaid:
		return "PAID"
	case domain.PaymentStatusFailed:
		return "FAILED"
	case domain.PaymentStatusRefunded:
		return "REFUNDED"
	default:
		return "UNKNOWN"
	}
}

func paymentAttemptStatusText(status domain.PaymentAttemptStatus) string {
	switch status {
	case domain.PaymentAttemptStatusProcessing:
		return "PROCESSING"
	case domain.PaymentAttemptStatusSucceeded:
		return "SUCCEEDED"
	case domain.PaymentAttemptStatusFailed:
		return "FAILED"
	case domain.PaymentAttemptStatusTimeout:
		return "TIMEOUT"
	case domain.PaymentAttemptStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

func queueStatusText(status service.QueueStatus) string {
	switch status {
	case service.QueueStatusWaiting:
		return "WAITING"
	case service.QueueStatusReady:
		return "READY"
	case service.QueueStatusExpired:
		return "EXPIRED"
	default:
		return "UNKNOWN"
	}
}

func adminUserStatusText(status domain.AdminUserStatus) string {
	switch status {
	case domain.AdminUserStatusActive:
		return "ACTIVE"
	case domain.AdminUserStatusDisabled:
		return "DISABLED"
	default:
		return "UNKNOWN"
	}
}

func organizerStatusText(status domain.OrganizerStatus) string {
	switch status {
	case domain.OrganizerStatusActive:
		return "ACTIVE"
	case domain.OrganizerStatusDisabled:
		return "DISABLED"
	default:
		return "UNKNOWN"
	}
}
