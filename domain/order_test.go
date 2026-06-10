package domain

import (
	"testing"
	"time"
)

func TestOrderPay(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("未過期的待付款訂單可以付款", func(t *testing.T) {
		order := Order{
			Status:    OrderStatusPendingPayment,
			ExpiresAt: now.Add(time.Minute),
		}

		ok := order.MarkPaid(now)

		if !ok {
			t.Fatal("預期訂單可付款")
		}
		if order.Status != OrderStatusPaid {
			t.Fatal("預期訂單狀態為 paid")
		}
	})

	t.Run("已過期的待付款訂單不可付款", func(t *testing.T) {
		order := Order{
			Status:    OrderStatusPendingPayment,
			ExpiresAt: now.Add(-time.Minute),
		}

		ok := order.MarkPaid(now)

		if ok {
			t.Fatal("預期已過期訂單不可付款")
		}
	})
}

func TestOrderExpire(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("待付款訂單可以過期", func(t *testing.T) {
		order := Order{Status: OrderStatusPendingPayment}

		ok := order.Expire(now)

		if !ok {
			t.Fatal("預期訂單可過期")
		}
		if order.Status != OrderStatusExpired {
			t.Fatal("預期訂單狀態為 expired")
		}
	})

	t.Run("已付款訂單不可過期", func(t *testing.T) {
		order := Order{Status: OrderStatusPaid}

		ok := order.Expire(now)

		if ok {
			t.Fatal("預期已付款訂單不可過期")
		}
	})
}

func TestOrderCancel(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("待付款訂單可以取消", func(t *testing.T) {
		order := Order{Status: OrderStatusPendingPayment}

		ok := order.Cancel(now)

		if !ok {
			t.Fatal("預期訂單可取消")
		}
		if order.Status != OrderStatusCancelled {
			t.Fatal("預期訂單狀態為 cancelled")
		}
	})

	t.Run("已付款訂單不可取消", func(t *testing.T) {
		order := Order{Status: OrderStatusPaid}

		ok := order.Cancel(now)

		if ok {
			t.Fatal("預期已付款訂單不可取消")
		}
	})
}
