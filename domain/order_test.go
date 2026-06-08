package domain

import (
	"testing"
	"time"
)

func TestOrderPay(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("待付款且未過期時應可付款", func(t *testing.T) {
		order := Order{
			Status:    OrderStatusPendingPayment,
			ExpiresAt: now.Add(time.Minute),
		}

		ok := order.MarkPaid(now)

		if !ok {
			t.Fatal("預期訂單付款成功")
		}
		if order.Status != OrderStatusPaid {
			t.Fatal("預期狀態變為 paid")
		}
	})

	t.Run("已過期訂單不可付款", func(t *testing.T) {
		order := Order{
			Status:    OrderStatusPendingPayment,
			ExpiresAt: now.Add(-time.Minute),
		}

		ok := order.MarkPaid(now)

		if ok {
			t.Fatal("預期過期訂單付款失敗")
		}
	})
}

func TestOrderCancel(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("未付款訂單應可取消", func(t *testing.T) {
		order := Order{Status: OrderStatusPendingPayment}

		ok := order.Cancel(now)

		if !ok {
			t.Fatal("預期訂單取消成功")
		}
		if order.Status != OrderStatusCancelled {
			t.Fatal("預期狀態變為 cancelled")
		}
	})

	t.Run("已付款訂單不可取消", func(t *testing.T) {
		order := Order{Status: OrderStatusPaid}

		ok := order.Cancel(now)

		if ok {
			t.Fatal("預期已付款訂單取消失敗")
		}
	})
}
