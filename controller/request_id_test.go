package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"buy-ticket/observability"

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

func TestRequestIDMiddlewareUsesIncomingRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/request-id", func(ctx *gin.Context) {
		if got := observability.RequestIDFromContext(ctx.Request.Context()); got != "req-incoming-001" {
			t.Fatalf("expected request id in context, got %q", got)
		}
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	req.Header.Set(headerRequestID, "req-incoming-001")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Header().Get(headerRequestID); got != "req-incoming-001" {
		t.Fatalf("expected response request id header, got %q", got)
	}
}

func TestRequestIDMiddlewareGeneratesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/request-id", func(ctx *gin.Context) {
		if got := observability.RequestIDFromContext(ctx.Request.Context()); got == "" {
			t.Fatal("expected generated request id in context")
		}
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Header().Get(headerRequestID); got == "" {
		t.Fatal("expected generated response request id header")
	}
}

func TestRequestLoggingMiddlewareIncludesWriteErrorFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buffer bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, nil)))
	defer slog.SetDefault(previousLogger)

	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(RequestLoggingMiddleware())
	router.GET("/error", func(ctx *gin.Context) {
		writeError(ctx, http.StatusBadRequest, errors.New("bad input"))
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	req.Header.Set(headerRequestID, "req-log-error-001")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var logRecord map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &logRecord); err != nil {
		t.Fatalf("unmarshal log record failed: %v; log=%q", err, buffer.String())
	}
	if got := logRecord["request_id"]; got != "req-log-error-001" {
		t.Fatalf("expected request_id req-log-error-001, got %v", got)
	}
	if got := logRecord["error_code"]; got != "BAD_REQUEST" {
		t.Fatalf("expected error_code BAD_REQUEST, got %v", got)
	}
	if got := logRecord["error_message"]; got != "bad input" {
		t.Fatalf("expected error_message bad input, got %v", got)
	}
}
