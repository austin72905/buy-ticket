package controller

import (
	"strings"
	"time"

	"buy-ticket/observability"

	"github.com/gin-gonic/gin"
)

const headerRequestID = observability.HeaderRequestID

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
