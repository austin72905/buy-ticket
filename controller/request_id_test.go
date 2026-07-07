package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteErrorIncludesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Header(headerRequestID, "req-write-error-001")
		ctx.Next()
	})
	router.GET("/error", func(ctx *gin.Context) {
		writeError(ctx, http.StatusBadRequest, errors.New("bad input"))
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var body ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error response failed: %v", err)
	}
	if body.RequestID != "req-write-error-001" {
		t.Fatalf("expected request id req-write-error-001, got %q", body.RequestID)
	}
}
