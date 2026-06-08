package domain

import "testing"

func TestSectionReserve(t *testing.T) {
	t.Run("庫存足夠時應保留成功", func(t *testing.T) {
		section := Section{
			Status:        SectionStatusActive,
			TotalQuantity: 10,
			PurchaseLimit: 4,
		}

		ok := section.Reserve(3)

		if !ok {
			t.Fatal("預期保留成功")
		}
		if section.ReservedQuantity != 3 {
			t.Fatalf("預期 ReservedQuantity 為 3，實際為 %d", section.ReservedQuantity)
		}
	})

	t.Run("超過剩餘數量時應保留失敗", func(t *testing.T) {
		section := Section{
			Status:           SectionStatusActive,
			TotalQuantity:    5,
			ReservedQuantity: 2,
			SoldQuantity:     2,
		}

		ok := section.Reserve(2)

		if ok {
			t.Fatal("預期保留失敗")
		}
	})

	t.Run("超過限購數量時應保留失敗", func(t *testing.T) {
		section := Section{
			Status:        SectionStatusActive,
			TotalQuantity: 10,
			PurchaseLimit: 2,
		}

		ok := section.Reserve(3)

		if ok {
			t.Fatal("預期超過限購時保留失敗")
		}
	})

	t.Run("票區不是 active 時應保留失敗", func(t *testing.T) {
		section := Section{
			Status:        SectionStatusInactive,
			TotalQuantity: 10,
		}

		ok := section.Reserve(1)

		if ok {
			t.Fatal("預期非 active 票區保留失敗")
		}
	})
}

func TestSectionRelease(t *testing.T) {
	t.Run("有保留票時應釋放成功", func(t *testing.T) {
		section := Section{
			Status:           SectionStatusSoldOut,
			TotalQuantity:    5,
			ReservedQuantity: 2,
			SoldQuantity:     3,
		}

		ok := section.Release(1)

		if !ok {
			t.Fatal("預期釋放成功")
		}
		if section.ReservedQuantity != 1 {
			t.Fatalf("預期 ReservedQuantity 為 1，實際為 %d", section.ReservedQuantity)
		}
		if section.Status != SectionStatusActive {
			t.Fatal("預期 sold_out 釋放後恢復為 active")
		}
	})

	t.Run("釋放超過保留數量時應失敗", func(t *testing.T) {
		section := Section{
			Status:           SectionStatusActive,
			ReservedQuantity: 1,
		}

		ok := section.Release(2)

		if ok {
			t.Fatal("預期釋放失敗")
		}
	})
}

func TestSectionConfirmSale(t *testing.T) {
	t.Run("保留票轉成售出票應成功", func(t *testing.T) {
		section := Section{
			Status:           SectionStatusActive,
			TotalQuantity:    5,
			ReservedQuantity: 2,
			SoldQuantity:     2,
		}

		ok := section.ConfirmSale(2)

		if !ok {
			t.Fatal("預期確認售出成功")
		}
		if section.ReservedQuantity != 0 {
			t.Fatalf("預期 ReservedQuantity 為 0，實際為 %d", section.ReservedQuantity)
		}
		if section.SoldQuantity != 4 {
			t.Fatalf("預期 SoldQuantity 為 4，實際為 %d", section.SoldQuantity)
		}
	})

	t.Run("賣完最後一張時應變成 sold_out", func(t *testing.T) {
		section := Section{
			Status:           SectionStatusActive,
			TotalQuantity:    5,
			ReservedQuantity: 1,
			SoldQuantity:     4,
		}

		ok := section.ConfirmSale(1)

		if !ok {
			t.Fatal("預期確認售出成功")
		}
		if section.Status != SectionStatusSoldOut {
			t.Fatal("預期最後一張售出後應為 sold_out")
		}
	})
}
