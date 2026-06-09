package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
