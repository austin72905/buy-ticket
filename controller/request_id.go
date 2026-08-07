package controller

import (
	"strings"
	"time"

	"buy-ticket/observability"

	"github.com/gin-gonic/gin"
)

/*
Client request

	↓

取得 X-Request-ID

	↓ 沒有就由系統產生

存進 context + response header

	↓

Handler / Service / Repository

	↓

Log 與錯誤 response 帶上同一個 Request ID
*/
const headerRequestID = observability.HeaderRequestID

const (
	contextErrorCodeKey    = "error_code"
	contextErrorMessageKey = "error_message"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestIDValue := strings.TrimSpace(ctx.GetHeader(headerRequestID))
		if requestIDValue == "" {
			requestIDValue = observability.NewRequestID()
		}
		if requestIDValue != "" {
			ctx.Header(headerRequestID, requestIDValue)
			ctx.Request = ctx.Request.WithContext(observability.WithRequestID(ctx.Request.Context(), requestIDValue))
		}
		ctx.Next()
	}
}

func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		latency := time.Since(startedAt)
		statusCode := ctx.Writer.Status()
		attrs := []any{
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"status", statusCode,
			"latency_ms", latency.Milliseconds(),
			"client_ip", ctx.ClientIP(),
			"user_agent", ctx.Request.UserAgent(),
		}
		if len(ctx.Errors) > 0 {
			attrs = append(attrs, "gin_errors", ctx.Errors.String())
		}
		if errorCode := stringContextValue(ctx, contextErrorCodeKey); errorCode != "" {
			attrs = append(attrs, "error_code", errorCode)
		}
		if errorMessage := stringContextValue(ctx, contextErrorMessageKey); errorMessage != "" {
			attrs = append(attrs, "error_message", errorMessage)
		}

		if statusCode >= 500 {
			observability.Error(ctx.Request.Context(), "http request completed", nil, attrs...)
			return
		}
		if statusCode >= 400 {
			observability.Warn(ctx.Request.Context(), "http request completed", attrs...)
			return
		}
		observability.Info(ctx.Request.Context(), "http request completed", attrs...)
	}
}

func stringContextValue(ctx *gin.Context, key string) string {
	value, ok := ctx.Get(key)
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func requestID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}

	if value := observability.RequestIDFromContext(ctx.Request.Context()); value != "" {
		return value
	}

	if value := strings.TrimSpace(ctx.Writer.Header().Get(headerRequestID)); value != "" {
		return value
	}

	return strings.TrimSpace(ctx.GetHeader(headerRequestID))
}
