package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"buy-ticket/domain"
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
		controller.RegisterRoutes(router)

		body := `{"event_id":1,"user_id":2,"client_id":"web-device-001","request_id":"req-001","channel":"web"}`
		req := httptest.NewRequest(http.MethodPost, "/queue/join", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("預期狀態碼 201，實際為 %d", resp.Code)
		}

		var response JoinQueueResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
			t.Fatalf("預期回傳合法 JSON，但解析失敗: %v", err)
		}
		if response.Status != int8(service.QueueStatusReady) {
			t.Fatalf("預期 status=%d，實際為 %d", service.QueueStatusReady, response.Status)
		}
		if response.QueueToken == "" {
			t.Fatal("預期回傳 queue token")
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
