package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecoveryMiddlewareReturnsErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RecoveryMiddleware())
	router.GET("/panic", func(ctx *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set(headerRequestID, "req-test-001")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, resp.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error response failed: %v", err)
	}
	if body.Code != "INTERNAL_SERVER_ERROR" {
		t.Fatalf("expected code INTERNAL_SERVER_ERROR, got %s", body.Code)
	}
	if body.Message != "internal server error" {
		t.Fatalf("expected sanitized message, got %q", body.Message)
	}
	if body.RequestID != "req-test-001" {
		t.Fatalf("expected request id req-test-001, got %q", body.RequestID)
	}
}
