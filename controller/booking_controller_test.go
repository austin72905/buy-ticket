package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

func TestBookingControllerGetQueueStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("查得到 queue token 時應回傳 200", func(t *testing.T) {
		now := time.Now()
		bookingService := service.NewBookingService(nil, nil, nil, nil, nil)
		bookingService.SaveQueueStatus(service.QueueStatusSnapshot{
			QueueToken:           "qt_001",
			QueueSequence:        1,
			Status:               service.QueueStatusWaiting,
			EventID:              1,
			UserID:               2,
			QueuePosition:        10,
			AheadCount:           9,
			EstimatedWaitSeconds: 60,
			JoinedAt:             now.Add(-time.Minute),
			ExpiredAt:            now.Add(29 * time.Minute),
			UpdatedAt:            now,
		})

		controller := NewBookingController(bookingService)
		router := gin.New()
		controller.RegisterRoutes(router)

		req := httptest.NewRequest(http.MethodGet, "/queue/status/qt_001", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("預期狀態碼 200，實際為 %d", resp.Code)
		}

		var body QueueStatusResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("預期回傳合法 JSON，但解析失敗: %v", err)
		}
		if body.Status != int8(service.QueueStatusWaiting) {
			t.Fatalf("預期 status=%d，實際為 %d", service.QueueStatusWaiting, body.Status)
		}
		if body.QueueToken != "qt_001" {
			t.Fatalf("預期 queue_token=qt_001，實際為 %s", body.QueueToken)
		}
	})

	t.Run("查不到 queue token 時應回傳 404", func(t *testing.T) {
		controller := NewBookingController(service.NewBookingService(nil, nil, nil, nil, nil))
		router := gin.New()
		controller.RegisterRoutes(router)

		req := httptest.NewRequest(http.MethodGet, "/queue/status/qt_not_found", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("預期狀態碼 404，實際為 %d", resp.Code)
		}
	})
}

func TestBookingControllerJoinQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("加入排隊成功時應回傳 201", func(t *testing.T) {
		now := time.Now()
		bookingService := service.NewBookingService(
			&fakeEventRepositoryForController{
				eventID:     1,
				saleStartAt: now.Add(-time.Hour),
				saleEndAt:   now.Add(time.Hour),
			},
			nil,
			nil,
			nil,
			nil,
		)

		controller := NewBookingController(bookingService)
		router := gin.New()
		userRepo := repository.NewMemoryUserRepository([]*domain.User{
			{
				ID:    1,
				Name:  "Test User",
				Email: "test@example.com",
			},
		})
		router.Use(AttachCurrentUser(service.NewAuthService(userRepo)))
		controller.RegisterRoutes(router)

		body := `{"event_id":1,"client_id":"web-device-001","request_id":"req-001","channel":"web"}`
		req := httptest.NewRequest(http.MethodPost, "/queue/join", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  sessionCookieName,
			Value: "1",
		})
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("預期狀態碼 201，實際為 %d", resp.Code)
		}

		var response JoinQueueResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
			t.Fatalf("預期回傳合法 JSON，但解析失敗: %v", err)
		}
		if response.Status != int8(service.QueueStatusWaiting) {
			t.Fatalf("預期 status=%d，實際為 %d", service.QueueStatusWaiting, response.Status)
		}
		if response.QueueToken == "" {
			t.Fatal("預期回傳 queue token")
		}
	})
}

func TestBookingControllerHandleECPayCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("成功回呼會回 1|OK", func(t *testing.T) {
		bookingService := service.NewBookingService(
			&fakeEventRepositoryForController{},
			&fakeSectionRepositoryForController{
				section: &domain.Section{
					ID:               2,
					EventID:          1,
					ReservedQuantity: 2,
					TotalQuantity:    10,
					Status:           domain.SectionStatusActive,
				},
			},
			&fakeReservationRepositoryForController{
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
						ExpiresAt:   time.Now().Add(5 * time.Minute),
					},
				},
			},
			&fakeOrderRepositoryForController{
				orders: map[int64]*domain.Order{
					20: {
						ID:            20,
						OrderNo:       "ORD-CB-HTTP-001",
						ReservationID: 10,
						UserID:        3,
						EventID:       1,
						SectionID:     2,
						Quantity:      2,
						UnitPrice:     1800,
						TotalAmount:   3600,
						Status:        domain.OrderStatusPendingPayment,
						ExpiresAt:     time.Now().Add(10 * time.Minute),
					},
				},
			},
			&fakePaymentRepositoryForController{},
		)
		bookingService.PaymentAttemptRepo = &fakePaymentAttemptRepositoryForController{
			attempts: map[int64]*domain.PaymentAttempt{
				30: {
					ID:              30,
					OrderID:         20,
					Provider:        "mock_ecpay",
					MerchantTradeNo: "MT-CB-HTTP-001",
					Method:          "credit_card",
					Amount:          3600,
					Status:          domain.PaymentAttemptStatusProcessing,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				},
			},
		}

		controller := NewBookingController(bookingService)
		router := gin.New()
		controller.RegisterRoutes(router)

		form := url.Values{}
		form.Set("MerchantTradeNo", "MT-CB-HTTP-001")
		form.Set("RtnCode", "1")
		form.Set("TradeNo", "TRADE-HTTP-001")
		form.Set("TradeAmt", "3600")
		form.Set("PaymentDate", "2026/06/11 18:30:00")
		form.Set("PaymentType", "Credit")

		req := httptest.NewRequest(http.MethodPost, "/payments/provider/ecpay/callback", bytes.NewBufferString(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("預期狀態碼 200，實際為 %d", resp.Code)
		}
		if resp.Body.String() != "1|OK" {
			t.Fatalf("預期回應 1|OK，實際為 %s", resp.Body.String())
		}
	})
}

func TestBookingControllerStartPayment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("creates payment attempt for pending order", func(t *testing.T) {
		bookingService := service.NewBookingService(
			&fakeEventRepositoryForController{},
			&fakeSectionRepositoryForController{},
			&fakeReservationRepositoryForController{},
			&fakeOrderRepositoryForController{
				orders: map[int64]*domain.Order{
					20: {
						ID:          20,
						UserID:      3,
						TotalAmount: 3600,
						Status:      domain.OrderStatusPendingPayment,
					},
				},
			},
			&fakePaymentRepositoryForController{},
		)
		bookingService.PaymentAttemptRepo = &fakePaymentAttemptRepositoryForController{}

		controller := NewBookingController(bookingService)
		router := gin.New()
		userRepo := repository.NewMemoryUserRepository([]*domain.User{
			{
				ID:    3,
				Name:  "Test User",
				Email: "test@example.com",
			},
		})
		router.Use(AttachCurrentUser(service.NewAuthService(userRepo)))
		controller.RegisterRoutes(router)

		req := httptest.NewRequest(http.MethodPost, "/payments/start", bytes.NewBufferString(`{"order_id":20,"method":"credit_card","provider":"mock_ecpay"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "pay-key-controller-001")
		req.AddCookie(&http.Cookie{
			Name:  sessionCookieName,
			Value: "3",
		})
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusAccepted {
			t.Fatalf("expected status 202, got %d", resp.Code)
		}

		var response PaymentAttemptResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
			t.Fatalf("expected JSON response: %v", err)
		}
		if response.OrderID != 20 {
			t.Fatalf("expected order_id=20, got %d", response.OrderID)
		}
		if response.Status != int8(domain.PaymentAttemptStatusTimeout) {
			t.Fatalf("expected timeout status without mock client, got %d", response.Status)
		}
		if response.MerchantTradeNo == "" {
			t.Fatal("expected merchant_trade_no")
		}
	})
}

type fakeEventRepositoryForController struct {
	eventID     int64
	saleStartAt time.Time
	saleEndAt   time.Time
}

func (f *fakeEventRepositoryForController) FindByID(ctx context.Context, eventID int64) (*domain.Event, error) {
	if eventID != f.eventID {
		return nil, errors.New("event not found")
	}

	return &domain.Event{
		ID:          f.eventID,
		Status:      domain.EventStatusOnSale,
		SaleStartAt: f.saleStartAt,
		SaleEndAt:   f.saleEndAt,
	}, nil
}

func (f *fakeEventRepositoryForController) List(ctx context.Context) ([]domain.Event, error) {
	return []domain.Event{}, nil
}

type fakeSectionRepositoryForController struct {
	section *domain.Section
}

func (f *fakeSectionRepositoryForController) FindByEventAndID(ctx context.Context, eventID, sectionID int64) (*domain.Section, error) {
	if f.section == nil || f.section.EventID != eventID || f.section.ID != sectionID {
		return nil, errors.New("section not found")
	}
	return f.section, nil
}

func (f *fakeSectionRepositoryForController) ListByEventID(ctx context.Context, eventID int64) ([]domain.Section, error) {
	return []domain.Section{}, nil
}

func (f *fakeSectionRepositoryForController) ReserveInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	section, err := f.FindByEventAndID(ctx, eventID, sectionID)
	if err != nil {
		return nil, err
	}
	if !section.Reserve(quantity) {
		return nil, errors.New("section cannot reserve requested quantity")
	}
	section.UpdatedAt = now
	return section, nil
}

func (f *fakeSectionRepositoryForController) ReleaseInventory(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	section, err := f.FindByEventAndID(ctx, eventID, sectionID)
	if err != nil {
		return nil, err
	}
	if !section.Release(quantity) {
		return nil, errors.New("section cannot release requested quantity")
	}
	section.UpdatedAt = now
	return section, nil
}

func (f *fakeSectionRepositoryForController) ConfirmSale(ctx context.Context, eventID, sectionID int64, quantity int, now time.Time) (*domain.Section, error) {
	section, err := f.FindByEventAndID(ctx, eventID, sectionID)
	if err != nil {
		return nil, err
	}
	if !section.ConfirmSale(quantity) {
		return nil, errors.New("section cannot confirm requested quantity")
	}
	section.UpdatedAt = now
	return section, nil
}

func (f *fakeSectionRepositoryForController) Save(ctx context.Context, section *domain.Section) error {
	f.section = section
	return nil
}

type fakeReservationRepositoryForController struct {
	reservations map[int64]*domain.Reservation
}

func (f *fakeReservationRepositoryForController) FindByID(ctx context.Context, reservationID int64) (*domain.Reservation, error) {
	reservation, ok := f.reservations[reservationID]
	if !ok {
		return nil, errors.New("reservation not found")
	}
	return reservation, nil
}

func (f *fakeReservationRepositoryForController) ListByUserID(ctx context.Context, userID int64) ([]domain.Reservation, error) {
	return []domain.Reservation{}, nil
}

func (f *fakeReservationRepositoryForController) FindActiveByUserAndEvent(ctx context.Context, userID, eventID int64, now time.Time) (*domain.Reservation, error) {
	return nil, repository.ErrReservationNotFound
}

func (f *fakeReservationRepositoryForController) Save(ctx context.Context, reservation *domain.Reservation) error {
	f.reservations[reservation.ID] = reservation
	return nil
}

type fakeOrderRepositoryForController struct {
	orders map[int64]*domain.Order
}

func (f *fakeOrderRepositoryForController) FindByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	order, ok := f.orders[orderID]
	if !ok {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (f *fakeOrderRepositoryForController) FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	for _, order := range f.orders {
		if order.OrderNo == orderNo {
			return order, nil
		}
	}
	return nil, errors.New("order not found")
}

func (f *fakeOrderRepositoryForController) ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]domain.Order, error) {
	return []domain.Order{}, nil
}

func (f *fakeOrderRepositoryForController) ListByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	return []domain.Order{}, nil
}

func (f *fakeOrderRepositoryForController) Save(ctx context.Context, order *domain.Order) error {
	f.orders[order.ID] = order
	return nil
}

type fakePaymentRepositoryForController struct {
	payments map[int64]*domain.Payment
	nextID   int64
}

func (f *fakePaymentRepositoryForController) FindByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error) {
	for _, payment := range f.payments {
		if payment.PaymentNo == paymentNo {
			return payment, nil
		}
	}
	return nil, errors.New("payment not found")
}

func (f *fakePaymentRepositoryForController) ListByUserID(ctx context.Context, userID int64) ([]domain.Payment, error) {
	return []domain.Payment{}, nil
}

func (f *fakePaymentRepositoryForController) Save(ctx context.Context, payment *domain.Payment) error {
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

type fakePaymentAttemptRepositoryForController struct {
	attempts map[int64]*domain.PaymentAttempt
	nextID   int64
}

func (f *fakePaymentAttemptRepositoryForController) FindByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (*domain.PaymentAttempt, error) {
	for _, attempt := range f.attempts {
		if attempt.MerchantTradeNo == merchantTradeNo {
			return attempt, nil
		}
	}
	return nil, repository.ErrPaymentAttemptNotFound
}

func (f *fakePaymentAttemptRepositoryForController) FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.PaymentAttempt, error) {
	for _, attempt := range f.attempts {
		if attempt.IdempotencyKey != nil && *attempt.IdempotencyKey == idempotencyKey {
			return attempt, nil
		}
	}
	return nil, repository.ErrPaymentAttemptNotFound
}

func (f *fakePaymentAttemptRepositoryForController) ListByOrderID(ctx context.Context, orderID int64) ([]domain.PaymentAttempt, error) {
	attempts := make([]domain.PaymentAttempt, 0)
	for _, attempt := range f.attempts {
		if attempt.OrderID == orderID {
			attempts = append(attempts, *attempt)
		}
	}
	return attempts, nil
}

func (f *fakePaymentAttemptRepositoryForController) Save(ctx context.Context, attempt *domain.PaymentAttempt) error {
	if f.attempts == nil {
		f.attempts = map[int64]*domain.PaymentAttempt{}
	}
	if attempt.ID == 0 {
		f.nextID++
		attempt.ID = f.nextID
	}
	f.attempts[attempt.ID] = attempt
	return nil
}
