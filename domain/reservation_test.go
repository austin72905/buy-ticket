package domain

import (
	"testing"
	"time"
)

func TestReservationConfirm(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("holding 且未過期時應確認成功", func(t *testing.T) {
		reservation := Reservation{
			Status:    ReservationStatusHolding,
			ExpiresAt: now.Add(time.Minute),
		}

		ok := reservation.Confirm(now)

		if !ok {
			t.Fatal("預期 reservation 確認成功")
		}
		if reservation.Status != ReservationStatusConfirmed {
			t.Fatal("預期狀態變為 confirmed")
		}
	})

	t.Run("已過期時應確認失敗", func(t *testing.T) {
		reservation := Reservation{
			Status:    ReservationStatusHolding,
			ExpiresAt: now.Add(-time.Minute),
		}

		ok := reservation.Confirm(now)

		if ok {
			t.Fatal("預期過期 reservation 確認失敗")
		}
	})
}

func TestReservationClose(t *testing.T) {
	now := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)

	t.Run("holding 狀態應可過期", func(t *testing.T) {
		reservation := Reservation{Status: ReservationStatusHolding}

		ok := reservation.Expire(now)

		if !ok {
			t.Fatal("預期 reservation 過期成功")
		}
		if reservation.Status != ReservationStatusExpired {
			t.Fatal("預期狀態變為 expired")
		}
	})

	t.Run("holding 狀態應可取消", func(t *testing.T) {
		reservation := Reservation{Status: ReservationStatusHolding}

		ok := reservation.Cancel(now)

		if !ok {
			t.Fatal("預期 reservation 取消成功")
		}
		if reservation.Status != ReservationStatusCancelled {
			t.Fatal("預期狀態變為 cancelled")
		}
	})

	t.Run("非 holding 狀態不可取消", func(t *testing.T) {
		reservation := Reservation{Status: ReservationStatusConfirmed}

		ok := reservation.Cancel(now)

		if ok {
			t.Fatal("預期非 holding 狀態取消失敗")
		}
	})
}
