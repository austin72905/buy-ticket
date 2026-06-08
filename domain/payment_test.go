package domain

import (
	"testing"
	"time"
)

func TestPaymentStatusChange(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("pending 應可標記為付款成功", func(t *testing.T) {
		payment := Payment{Status: PaymentStatusPending}

		ok := payment.MarkPaid(now)

		if !ok {
			t.Fatal("預期付款成功標記成功")
		}
		if payment.Status != PaymentStatusPaid {
			t.Fatal("預期狀態變為 paid")
		}
		if payment.PaidAt == nil {
			t.Fatal("預期 PaidAt 被寫入")
		}
	})

	t.Run("pending 應可標記為付款失敗", func(t *testing.T) {
		payment := Payment{Status: PaymentStatusPending}

		ok := payment.MarkFailed(now)

		if !ok {
			t.Fatal("預期付款失敗標記成功")
		}
		if payment.Status != PaymentStatusFailed {
			t.Fatal("預期狀態變為 failed")
		}
		if payment.FailedAt == nil {
			t.Fatal("預期 FailedAt 被寫入")
		}
	})

	t.Run("非 pending 狀態不可再標記成功", func(t *testing.T) {
		payment := Payment{Status: PaymentStatusFailed}

		ok := payment.MarkPaid(now)

		if ok {
			t.Fatal("預期非 pending 狀態標記成功失敗")
		}
	})
}
