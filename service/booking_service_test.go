package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"buy-ticket/domain"
)

func TestBookingServiceReserveTicket(t *testing.T) {
	t.Run("驗證 purchase token 後成功建立 reservation", func(t *testing.T) {
		now := time.Now()
		purchaseToken := "pt_reserve_ok"
		eventRepo := &fakeEventRepository{
			event: &domain.Event{
				ID:          1,
				Name:        "Jay Concert",
				Status:      domain.EventStatusOnSale,
				SaleStartAt: now.Add(-time.Hour),
				SaleEndAt:   now.Add(time.Hour),
			},
		}
		sectionRepo := &fakeSectionRepository{
			section: &domain.Section{
				ID:            2,
				EventID:       1,
				Name:          "A 區",
				Price:         1800,
				TotalQuantity: 10,
				Status:        domain.SectionStatusActive,
			},
		}
		reservationRepo := &fakeReservationRepository{}
		svc := NewBookingService(eventRepo, sectionRepo, reservationRepo, &fakeOrderRepository{}, &fakePaymentRepository{})
		svc.SaveQueueStatus(QueueStatusSnapshot{
			QueueToken:             "qt_reserve_ok",
			QueueSequence:          1,
			Status:                 QueueStatusReady,
			EventID:                1,
			UserID:                 3,
			QueuePosition:          1,
			AheadCount:             0,
			EstimatedWaitSeconds:   0,
			PurchaseToken:          &purchaseToken,
			PurchaseTokenExpiresAt: ptrTime(now.Add(5 * time.Minute)),
			JoinedAt:               now.Add(-time.Minute),
			ExpiredAt:              now.Add(29 * time.Minute),
			UpdatedAt:              now,
		})

		reservation, err := svc.ReserveTicket(context.Background(), ReserveTicketInput{
			UserID:        3,
			EventID:       1,
			SectionID:     2,
			Quantity:      2,
			HoldUntil:     now.Add(5 * time.Minute),
			PurchaseToken: purchaseToken,
		})

		if err != nil {
			t.Fatalf("預期建立 reservation 成功，但得到錯誤: %v", err)
		}
		if reservation.Status != domain.ReservationStatusHolding {
			t.Fatal("預期 reservation 狀態為 holding")
		}
		if reservation.TotalAmount != 3600 {
			t.Fatalf("預期總金額為 3600，實際為 %d", reservation.TotalAmount)
		}
		if sectionRepo.section.ReservedQuantity != 2 {
			t.Fatalf("預期區域保留數量為 2，實際為 %d", sectionRepo.section.ReservedQuantity)
		}
	})

	t.Run("活動未開賣時應回傳 event is not on sale", func(t *testing.T) {
		now := time.Now()
		purchaseToken := "pt_event_not_on_sale"
		eventRepo := &fakeEventRepository{
			event: &domain.Event{
				ID:          1,
				Status:      domain.EventStatusOnSale,
				SaleStartAt: now.Add(time.Hour),
				SaleEndAt:   now.Add(2 * time.Hour),
			},
		}
		sectionRepo := &fakeSectionRepository{
			section: &domain.Section{
				ID:            2,
				EventID:       1,
				Price:         1800,
				TotalQuantity: 10,
				Status:        domain.SectionStatusActive,
			},
		}
		svc := NewBookingService(eventRepo, sectionRepo, &fakeReservationRepository{}, &fakeOrderRepository{}, &fakePaymentRepository{})
		svc.SaveQueueStatus(QueueStatusSnapshot{
			QueueToken:             "qt_event_not_on_sale",
			QueueSequence:          1,
			Status:                 QueueStatusReady,
			EventID:                1,
			UserID:                 3,
			QueuePosition:          1,
			AheadCount:             0,
			EstimatedWaitSeconds:   0,
			PurchaseToken:          &purchaseToken,
			PurchaseTokenExpiresAt: ptrTime(now.Add(5 * time.Minute)),
			JoinedAt:               now.Add(-time.Minute),
			ExpiredAt:              now.Add(29 * time.Minute),
			UpdatedAt:              now,
		})

		_, err := svc.ReserveTicket(context.Background(), ReserveTicketInput{
			UserID:        3,
			EventID:       1,
			SectionID:     2,
			Quantity:      2,
			HoldUntil:     now.Add(5 * time.Minute),
			PurchaseToken: purchaseToken,
		})

		if !errors.Is(err, ErrEventNotOnSale) {
			t.Fatalf("預期錯誤為 ErrEventNotOnSale，實際為 %v", err)
		}
	})

	t.Run("purchase token 重複使用時應失敗", func(t *testing.T) {
		now := time.Now()
		purchaseToken := "pt_used_once"
		eventRepo := &fakeEventRepository{
			event: &domain.Event{
				ID:          1,
				Name:        "Jay Concert",
				Status:      domain.EventStatusOnSale,
				SaleStartAt: now.Add(-time.Hour),
				SaleEndAt:   now.Add(time.Hour),
			},
		}
		sectionRepo := &fakeSectionRepository{
			section: &domain.Section{
				ID:            2,
				EventID:       1,
				Name:          "A 區",
				Price:         1800,
				TotalQuantity: 10,
				Status:        domain.SectionStatusActive,
			},
		}
		svc := NewBookingService(eventRepo, sectionRepo, &fakeReservationRepository{}, &fakeOrderRepository{}, &fakePaymentRepository{})
		svc.SaveQueueStatus(QueueStatusSnapshot{
			QueueToken:             "qt_used_once",
			QueueSequence:          1,
			Status:                 QueueStatusReady,
			EventID:                1,
			UserID:                 3,
			QueuePosition:          1,
			AheadCount:             0,
			EstimatedWaitSeconds:   0,
			PurchaseToken:          &purchaseToken,
			PurchaseTokenExpiresAt: ptrTime(now.Add(5 * time.Minute)),
			JoinedAt:               now.Add(-time.Minute),
			ExpiredAt:              now.Add(29 * time.Minute),
			UpdatedAt:              now,
		})

		_, err := svc.ReserveTicket(context.Background(), ReserveTicketInput{
			UserID:        3,
			EventID:       1,
			SectionID:     2,
			Quantity:      1,
			HoldUntil:     now.Add(5 * time.Minute),
			PurchaseToken: purchaseToken,
		})
		if err != nil {
			t.Fatalf("第一次使用 purchase token 不應失敗: %v", err)
		}

		_, err = svc.ReserveTicket(context.Background(), ReserveTicketInput{
			UserID:        3,
			EventID:       1,
			SectionID:     2,
			Quantity:      1,
			HoldUntil:     now.Add(5 * time.Minute),
			PurchaseToken: purchaseToken,
		})
		if !errors.Is(err, ErrPurchaseTokenNotFound) {
			t.Fatalf("預期錯誤為 ErrPurchaseTokenNotFound，實際為 %v", err)
		}
	})
}

func TestBookingServiceCreateOrder(t *testing.T) {
	t.Run("從有效 reservation 建立待付款訂單", func(t *testing.T) {
		now := time.Now()
		reservationRepo := &fakeReservationRepository{
			reservations: map[int64]*domain.Reservation{
				10: {
					ID:          10,
					UserID:      3,
					EventID:     1,
					SectionID:   2,
					Quantity:    2,
					UnitPrice:   1800,
					TotalAmount: 3600,
					Status:      domain.ReservationStatusHolding,
					ExpiresAt:   now.Add(5 * time.Minute),
				},
			},
			nextID: 10,
		}
		orderRepo := &fakeOrderRepository{}
		svc := NewBookingService(&fakeEventRepository{}, &fakeSectionRepository{}, reservationRepo, orderRepo, &fakePaymentRepository{})

		order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
			ReservationID: 10,
			OrderNo:       "ORD-001",
			ExpiresAt:     now.Add(10 * time.Minute),
		})

		if err != nil {
			t.Fatalf("預期建立訂單成功，但得到錯誤: %v", err)
		}
		if order.Status != domain.OrderStatusPendingPayment {
			t.Fatal("預期訂單狀態為 pending payment")
		}
		if order.TotalAmount != 3600 {
			t.Fatalf("預期訂單金額為 3600，實際為 %d", order.TotalAmount)
		}
		if order.Quantity != 2 || order.UnitPrice != 1800 {
			t.Fatalf("預期 quantity=2 unit_price=1800，實際為 quantity=%d unit_price=%d", order.Quantity, order.UnitPrice)
		}
	})
}

func TestBookingServiceJoinQueue(t *testing.T) {
	t.Run("活動開賣時應建立 queue token 並回 ready", func(t *testing.T) {
		now := time.Now()
		svc := NewBookingService(
			&fakeEventRepository{
				event: &domain.Event{
					ID:          1,
					Status:      domain.EventStatusOnSale,
					SaleStartAt: now.Add(-time.Hour),
					SaleEndAt:   now.Add(time.Hour),
				},
			},
			&fakeSectionRepository{},
			&fakeReservationRepository{},
			&fakeOrderRepository{},
			&fakePaymentRepository{},
		)

		snapshot, err := svc.JoinQueue(context.Background(), JoinQueueInput{
			EventID:   1,
			UserID:    2,
			ClientID:  "web-device-001",
			RequestID: "req-001",
			Channel:   "web",
		}, now)
		if err != nil {
			t.Fatalf("預期加入排隊成功，但得到錯誤: %v", err)
		}
		if snapshot.Status != QueueStatusReady {
			t.Fatalf("預期 status=ready(2)，實際為 %d", snapshot.Status)
		}
		if snapshot.QueueToken == "" {
			t.Fatal("預期產生 queue token")
		}
		if snapshot.PurchaseToken == nil || *snapshot.PurchaseToken == "" {
			t.Fatal("預期產生 purchase token")
		}
	})

	t.Run("同一使用者重複加入有效 queue 時應回錯誤", func(t *testing.T) {
		now := time.Now()
		svc := NewBookingService(
			&fakeEventRepository{
				event: &domain.Event{
					ID:          1,
					Status:      domain.EventStatusOnSale,
					SaleStartAt: now.Add(-time.Hour),
					SaleEndAt:   now.Add(time.Hour),
				},
			},
			&fakeSectionRepository{},
			&fakeReservationRepository{},
			&fakeOrderRepository{},
			&fakePaymentRepository{},
		)

		_, err := svc.JoinQueue(context.Background(), JoinQueueInput{
			EventID:   1,
			UserID:    2,
			ClientID:  "web-device-001",
			RequestID: "req-001",
			Channel:   "web",
		}, now)
		if err != nil {
			t.Fatalf("第一次加入不應失敗: %v", err)
		}

		_, err = svc.JoinQueue(context.Background(), JoinQueueInput{
			EventID:   1,
			UserID:    2,
			ClientID:  "web-device-001",
			RequestID: "req-002",
			Channel:   "web",
		}, now.Add(time.Second))
		if !errors.Is(err, ErrUserAlreadyJoinedQueue) {
			t.Fatalf("預期錯誤為 ErrUserAlreadyJoinedQueue，實際為 %v", err)
		}
	})
}

func TestBookingServicePayOrder(t *testing.T) {
	t.Run("付款成功後應更新 payment、order、reservation、section", func(t *testing.T) {
		now := time.Now()
		orderRepo := &fakeOrderRepository{
			orders: map[int64]*domain.Order{
				20: {
					ID:            20,
					OrderNo:       "ORD-001",
					ReservationID: 10,
					UserID:        3,
					EventID:       1,
					SectionID:     2,
					Quantity:      2,
					UnitPrice:     1800,
					TotalAmount:   3600,
					Status:        domain.OrderStatusPendingPayment,
					ExpiresAt:     now.Add(10 * time.Minute),
				},
			},
			nextID: 20,
		}
		reservationRepo := &fakeReservationRepository{
			reservations: map[int64]*domain.Reservation{
				10: {
					ID:          10,
					EventID:     1,
					SectionID:   2,
					UserID:      3,
					Quantity:    2,
					UnitPrice:   1800,
					TotalAmount: 3600,
					Status:      domain.ReservationStatusHolding,
					ExpiresAt:   now.Add(5 * time.Minute),
				},
			},
			nextID: 10,
		}
		sectionRepo := &fakeSectionRepository{
			section: &domain.Section{
				ID:               2,
				EventID:          1,
				ReservedQuantity: 2,
				SoldQuantity:     3,
				TotalQuantity:    10,
				Status:           domain.SectionStatusActive,
			},
		}
		paymentRepo := &fakePaymentRepository{}
		svc := NewBookingService(&fakeEventRepository{}, sectionRepo, reservationRepo, orderRepo, paymentRepo)

		payment, err := svc.PayOrder(context.Background(), PayOrderInput{
			OrderID:   20,
			PaymentNo: "PAY-001",
			Method:    "credit_card",
			Amount:    3600,
			PaidAt:    now,
		})

		if err != nil {
			t.Fatalf("預期付款成功，但得到錯誤: %v", err)
		}
		if payment.Status != domain.PaymentStatusPaid {
			t.Fatal("預期 payment 狀態為 paid")
		}
		if orderRepo.orders[20].Status != domain.OrderStatusPaid {
			t.Fatal("預期 order 狀態為 paid")
		}
		if reservationRepo.reservations[10].Status != domain.ReservationStatusConfirmed {
			t.Fatal("預期 reservation 狀態為 confirmed")
		}
		if sectionRepo.section.ReservedQuantity != 0 || sectionRepo.section.SoldQuantity != 5 {
			t.Fatalf("預期 reserved=0 sold=5，實際為 reserved=%d sold=%d", sectionRepo.section.ReservedQuantity, sectionRepo.section.SoldQuantity)
		}
	})

	t.Run("付款金額不一致時應回傳錯誤", func(t *testing.T) {
		now := time.Now()
		orderRepo := &fakeOrderRepository{
			orders: map[int64]*domain.Order{
				20: {
					ID:            20,
					ReservationID: 10,
					TotalAmount:   3600,
					Status:        domain.OrderStatusPendingPayment,
					ExpiresAt:     now.Add(10 * time.Minute),
				},
			},
		}
		svc := NewBookingService(&fakeEventRepository{}, &fakeSectionRepository{}, &fakeReservationRepository{}, orderRepo, &fakePaymentRepository{})

		_, err := svc.PayOrder(context.Background(), PayOrderInput{
			OrderID:   20,
			PaymentNo: "PAY-001",
			Method:    "credit_card",
			Amount:    3000,
			PaidAt:    now,
		})

		if !errors.Is(err, ErrPaymentAmountMismatch) {
			t.Fatalf("預期錯誤為 ErrPaymentAmountMismatch，實際為 %v", err)
		}
	})
}

func TestBookingServiceExpireOrder(t *testing.T) {
	t.Run("訂單過期時應同時釋放 reservation 與 section", func(t *testing.T) {
		now := time.Now()
		orderRepo := &fakeOrderRepository{
			orders: map[int64]*domain.Order{
				30: {
					ID:            30,
					OrderNo:       "ORD-EXPIRE-001",
					ReservationID: 11,
					UserID:        3,
					EventID:       1,
					SectionID:     2,
					Quantity:      2,
					UnitPrice:     1800,
					TotalAmount:   3600,
					Status:        domain.OrderStatusPendingPayment,
					ExpiresAt:     now.Add(-time.Minute),
				},
			},
			nextID: 30,
		}
		reservationRepo := &fakeReservationRepository{
			reservations: map[int64]*domain.Reservation{
				11: {
					ID:          11,
					EventID:     1,
					SectionID:   2,
					UserID:      3,
					Quantity:    2,
					UnitPrice:   1800,
					TotalAmount: 3600,
					Status:      domain.ReservationStatusHolding,
					ExpiresAt:   now.Add(5 * time.Minute),
				},
			},
			nextID: 11,
		}
		sectionRepo := &fakeSectionRepository{
			section: &domain.Section{
				ID:               2,
				EventID:          1,
				ReservedQuantity: 2,
				TotalQuantity:    10,
				Status:           domain.SectionStatusActive,
			},
		}
		svc := NewBookingService(&fakeEventRepository{}, sectionRepo, reservationRepo, orderRepo, &fakePaymentRepository{})

		order, err := svc.ExpireOrder(context.Background(), ExpireOrderInput{
			OrderID:   30,
			ExpiredAt: now,
		})
		if err != nil {
			t.Fatalf("預期訂單過期成功，但得到錯誤: %v", err)
		}
		if order.Status != domain.OrderStatusExpired {
			t.Fatal("預期 order 狀態為 expired")
		}
		if reservationRepo.reservations[11].Status != domain.ReservationStatusExpired {
			t.Fatal("預期 reservation 狀態為 expired")
		}
		if sectionRepo.section.ReservedQuantity != 0 {
			t.Fatalf("預期區域保留數量為 0，實際為 %d", sectionRepo.section.ReservedQuantity)
		}
	})

	t.Run("已付款訂單不可過期", func(t *testing.T) {
		now := time.Now()
		orderRepo := &fakeOrderRepository{
			orders: map[int64]*domain.Order{
				31: {
					ID:            31,
					ReservationID: 12,
					Status:        domain.OrderStatusPaid,
					ExpiresAt:     now.Add(-time.Minute),
				},
			},
		}
		svc := NewBookingService(&fakeEventRepository{}, &fakeSectionRepository{}, &fakeReservationRepository{}, orderRepo, &fakePaymentRepository{})

		_, err := svc.ExpireOrder(context.Background(), ExpireOrderInput{
			OrderID:   31,
			ExpiredAt: now,
		})
		if !errors.Is(err, ErrOrderCannotExpire) {
			t.Fatalf("預期錯誤為 ErrOrderCannotExpire，實際為 %v", err)
		}
	})
}

func TestBookingServiceCloseReservation(t *testing.T) {
	t.Run("reservation 過期後應釋放區域保留數量", func(t *testing.T) {
		now := time.Now()
		reservationRepo := &fakeReservationRepository{
			reservations: map[int64]*domain.Reservation{
				10: {
					ID:        10,
					EventID:   1,
					SectionID: 2,
					Quantity:  2,
					Status:    domain.ReservationStatusHolding,
				},
			},
		}
		sectionRepo := &fakeSectionRepository{
			section: &domain.Section{
				ID:               2,
				EventID:          1,
				ReservedQuantity: 2,
				TotalQuantity:    10,
				Status:           domain.SectionStatusActive,
			},
		}
		svc := NewBookingService(&fakeEventRepository{}, sectionRepo, reservationRepo, &fakeOrderRepository{}, &fakePaymentRepository{})

		reservation, err := svc.ExpireReservation(context.Background(), ExpireReservationInput{
			ReservationID: 10,
			ExpiredAt:     now,
		})

		if err != nil {
			t.Fatalf("預期過期成功，但得到錯誤: %v", err)
		}
		if reservation.Status != domain.ReservationStatusExpired {
			t.Fatal("預期 reservation 狀態為 expired")
		}
		if sectionRepo.section.ReservedQuantity != 0 {
			t.Fatalf("預期區域保留數量為 0，實際為 %d", sectionRepo.section.ReservedQuantity)
		}
	})

	t.Run("reservation 取消後應釋放區域保留數量", func(t *testing.T) {
		now := time.Now()
		reservationRepo := &fakeReservationRepository{
			reservations: map[int64]*domain.Reservation{
				11: {
					ID:        11,
					EventID:   1,
					SectionID: 2,
					Quantity:  1,
					Status:    domain.ReservationStatusHolding,
				},
			},
		}
		sectionRepo := &fakeSectionRepository{
			section: &domain.Section{
				ID:               2,
				EventID:          1,
				ReservedQuantity: 1,
				TotalQuantity:    10,
				Status:           domain.SectionStatusActive,
			},
		}
		svc := NewBookingService(&fakeEventRepository{}, sectionRepo, reservationRepo, &fakeOrderRepository{}, &fakePaymentRepository{})

		reservation, err := svc.CancelReservation(context.Background(), CancelReservationInput{
			ReservationID: 11,
			CancelledAt:   now,
		})

		if err != nil {
			t.Fatalf("預期取消成功，但得到錯誤: %v", err)
		}
		if reservation.Status != domain.ReservationStatusCancelled {
			t.Fatal("預期 reservation 狀態為 cancelled")
		}
		if sectionRepo.section.ReservedQuantity != 0 {
			t.Fatalf("預期區域保留數量為 0，實際為 %d", sectionRepo.section.ReservedQuantity)
		}
	})
}

func TestBookingServiceGetQueueStatus(t *testing.T) {
	t.Run("查得到 queue token 時應回傳快照", func(t *testing.T) {
		now := time.Now()
		purchaseToken := "pt_001"
		purchaseTokenExpiresAt := now.Add(5 * time.Minute)

		svc := NewBookingService(
			&fakeEventRepository{},
			&fakeSectionRepository{},
			&fakeReservationRepository{},
			&fakeOrderRepository{},
			&fakePaymentRepository{},
		)
		svc.SaveQueueStatus(QueueStatusSnapshot{
			QueueToken:             "qt_001",
			QueueSequence:          1,
			Status:                 QueueStatusReady,
			EventID:                1,
			UserID:                 2,
			QueuePosition:          1,
			AheadCount:             0,
			EstimatedWaitSeconds:   0,
			PurchaseToken:          &purchaseToken,
			PurchaseTokenExpiresAt: &purchaseTokenExpiresAt,
			JoinedAt:               now.Add(-2 * time.Minute),
			ExpiredAt:              now.Add(28 * time.Minute),
			UpdatedAt:              now,
		})

		snapshot, err := svc.GetQueueStatus(context.Background(), "qt_001")
		if err != nil {
			t.Fatalf("預期查詢成功，但得到錯誤: %v", err)
		}
		if snapshot.Status != QueueStatusReady {
			t.Fatalf("預期 status=ready(2)，實際為 %d", snapshot.Status)
		}
		if snapshot.PurchaseToken == nil || *snapshot.PurchaseToken != "pt_001" {
			t.Fatal("預期回傳 purchase token")
		}
	})

	t.Run("查不到 queue token 時應回傳錯誤", func(t *testing.T) {
		svc := NewBookingService(
			&fakeEventRepository{},
			&fakeSectionRepository{},
			&fakeReservationRepository{},
			&fakeOrderRepository{},
			&fakePaymentRepository{},
		)

		_, err := svc.GetQueueStatus(context.Background(), "qt_not_found")
		if !errors.Is(err, ErrQueueTokenNotFound) {
			t.Fatalf("預期錯誤為 ErrQueueTokenNotFound，實際為 %v", err)
		}
	})
}

func TestMemoryQueueStorePromoteReady(t *testing.T) {
	t.Run("前一位完成後應放行下一位", func(t *testing.T) {
		now := time.Now()
		store := NewMemoryQueueStore(1)

		first, err := store.Join(context.Background(), JoinQueueInput{
			EventID: 1,
			UserID:  1,
		}, now)
		if err != nil {
			t.Fatalf("第一位加入失敗: %v", err)
		}
		second, err := store.Join(context.Background(), JoinQueueInput{
			EventID: 1,
			UserID:  2,
		}, now.Add(time.Millisecond))
		if err != nil {
			t.Fatalf("第二位加入失敗: %v", err)
		}
		if second.Status != QueueStatusWaiting {
			t.Fatalf("預期第二位初始為 waiting，實際為 %d", second.Status)
		}

		if first.PurchaseToken == nil {
			t.Fatal("預期第一位有 purchase token")
		}
		_, err = store.ConsumePurchaseToken(context.Background(), *first.PurchaseToken, first.EventID, first.UserID, now.Add(time.Second))
		if err != nil {
			t.Fatalf("第一位消耗 purchase token 失敗: %v", err)
		}

		if err := store.PromoteReady(context.Background(), now.Add(2*time.Second)); err != nil {
			t.Fatalf("promote ready 失敗: %v", err)
		}

		snapshot, err := store.Get(context.Background(), second.QueueToken, now.Add(2*time.Second))
		if err != nil {
			t.Fatalf("查第二位 queue status 失敗: %v", err)
		}
		if snapshot.Status != QueueStatusReady {
			t.Fatalf("預期第二位被放行為 ready，實際為 %d", snapshot.Status)
		}
		if snapshot.PurchaseToken == nil {
			t.Fatal("預期第二位取得 purchase token")
		}
	})

	t.Run("放行上限為 2 時前兩位應直接 ready", func(t *testing.T) {
		now := time.Now()
		store := NewMemoryQueueStore(2)

		first, err := store.Join(context.Background(), JoinQueueInput{EventID: 1, UserID: 1}, now)
		if err != nil {
			t.Fatalf("第一位加入失敗: %v", err)
		}
		second, err := store.Join(context.Background(), JoinQueueInput{EventID: 1, UserID: 2}, now.Add(time.Millisecond))
		if err != nil {
			t.Fatalf("第二位加入失敗: %v", err)
		}
		third, err := store.Join(context.Background(), JoinQueueInput{EventID: 1, UserID: 3}, now.Add(2*time.Millisecond))
		if err != nil {
			t.Fatalf("第三位加入失敗: %v", err)
		}

		if first.Status != QueueStatusReady {
			t.Fatalf("預期第一位為 ready，實際為 %d", first.Status)
		}
		if second.Status != QueueStatusReady {
			t.Fatalf("預期第二位為 ready，實際為 %d", second.Status)
		}
		if third.Status != QueueStatusWaiting {
			t.Fatalf("預期第三位為 waiting，實際為 %d", third.Status)
		}
	})
}

type fakeEventRepository struct {
	event *domain.Event
}

func (f *fakeEventRepository) FindByID(ctx context.Context, eventID int64) (*domain.Event, error) {
	if f.event == nil || f.event.ID != eventID {
		return nil, errors.New("event not found")
	}

	return f.event, nil
}

func (f *fakeEventRepository) List(ctx context.Context) ([]domain.Event, error) {
	if f.event == nil {
		return []domain.Event{}, nil
	}

	return []domain.Event{*f.event}, nil
}

type fakeSectionRepository struct {
	section *domain.Section
}

func (f *fakeSectionRepository) FindByEventAndID(ctx context.Context, eventID, sectionID int64) (*domain.Section, error) {
	if f.section == nil || f.section.EventID != eventID || f.section.ID != sectionID {
		return nil, errors.New("section not found")
	}

	return f.section, nil
}

func (f *fakeSectionRepository) ListByEventID(ctx context.Context, eventID int64) ([]domain.Section, error) {
	if f.section == nil || f.section.EventID != eventID {
		return nil, errors.New("section not found")
	}

	return []domain.Section{*f.section}, nil
}

func (f *fakeSectionRepository) Save(ctx context.Context, section *domain.Section) error {
	f.section = section
	return nil
}

type fakeReservationRepository struct {
	reservations map[int64]*domain.Reservation
	nextID       int64
}

func (f *fakeReservationRepository) FindByID(ctx context.Context, reservationID int64) (*domain.Reservation, error) {
	if f.reservations == nil {
		return nil, errors.New("reservation not found")
	}

	reservation, ok := f.reservations[reservationID]
	if !ok {
		return nil, errors.New("reservation not found")
	}

	return reservation, nil
}

func (f *fakeReservationRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Reservation, error) {
	reservations := make([]domain.Reservation, 0)
	for _, reservation := range f.reservations {
		if reservation.UserID != userID {
			continue
		}
		reservations = append(reservations, *reservation)
	}
	return reservations, nil
}

func (f *fakeReservationRepository) Save(ctx context.Context, reservation *domain.Reservation) error {
	if f.reservations == nil {
		f.reservations = map[int64]*domain.Reservation{}
	}

	if reservation.ID == 0 {
		f.nextID++
		reservation.ID = f.nextID
	}

	f.reservations[reservation.ID] = reservation
	return nil
}

type fakeOrderRepository struct {
	orders map[int64]*domain.Order
	nextID int64
}

func (f *fakeOrderRepository) FindByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	if f.orders == nil {
		return nil, errors.New("order not found")
	}

	order, ok := f.orders[orderID]
	if !ok {
		return nil, errors.New("order not found")
	}

	return order, nil
}

func (f *fakeOrderRepository) FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	for _, order := range f.orders {
		if order.OrderNo == orderNo {
			return order, nil
		}
	}

	return nil, errors.New("order not found")
}

func (f *fakeOrderRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	orders := make([]domain.Order, 0)
	for _, order := range f.orders {
		if order.UserID != userID {
			continue
		}
		orders = append(orders, *order)
	}
	return orders, nil
}

func (f *fakeOrderRepository) Save(ctx context.Context, order *domain.Order) error {
	if f.orders == nil {
		f.orders = map[int64]*domain.Order{}
	}

	if order.ID == 0 {
		f.nextID++
		order.ID = f.nextID
	}

	f.orders[order.ID] = order
	return nil
}

type fakePaymentRepository struct {
	payments map[int64]*domain.Payment
	nextID   int64
}

func (f *fakePaymentRepository) FindByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error) {
	for _, payment := range f.payments {
		if payment.PaymentNo == paymentNo {
			return payment, nil
		}
	}

	return nil, errors.New("payment not found")
}

func (f *fakePaymentRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.Payment, error) {
	payments := make([]domain.Payment, 0, len(f.payments))
	for _, payment := range f.payments {
		payments = append(payments, *payment)
	}
	return payments, nil
}

func (f *fakePaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	if f.payments == nil {
		f.payments = map[int64]*domain.Payment{}
	}

	if payment.ID == 0 {
		f.nextID++
		payment.ID = f.nextID
	}

	f.payments[payment.ID] = payment
	return nil
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
