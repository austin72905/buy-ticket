package domain

import (
	"testing"
	"time"
)

func TestEventIsOnSale(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("未開賣時應回傳 false", func(t *testing.T) {
		event := Event{
			Status:      EventStatusOnSale,
			SaleStartAt: now.Add(time.Hour),
			SaleEndAt:   now.Add(2 * time.Hour),
		}

		if event.IsOnSale(now) {
			t.Fatal("預期未開賣時不應可購買")
		}
	})

	t.Run("開賣期間內應回傳 true", func(t *testing.T) {
		event := Event{
			Status:      EventStatusOnSale,
			SaleStartAt: now.Add(-time.Hour),
			SaleEndAt:   now.Add(time.Hour),
		}

		if !event.IsOnSale(now) {
			t.Fatal("預期開賣期間內應可購買")
		}
	})

	t.Run("已超過售票時間應回傳 false", func(t *testing.T) {
		event := Event{
			Status:      EventStatusOnSale,
			SaleStartAt: now.Add(-2 * time.Hour),
			SaleEndAt:   now.Add(-time.Minute),
		}

		if event.IsOnSale(now) {
			t.Fatal("預期售票結束後不應可購買")
		}
	})

	t.Run("狀態不是開賣中時應回傳 false", func(t *testing.T) {
		event := Event{
			Status:      EventStatusPublished,
			SaleStartAt: now.Add(-time.Hour),
			SaleEndAt:   now.Add(time.Hour),
		}

		if event.IsOnSale(now) {
			t.Fatal("預期非開賣狀態時不應可購買")
		}
	})
}
