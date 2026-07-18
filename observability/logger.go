package observability

import (
	"context"
	"log/slog"
)

func Info(ctx context.Context, msg string, attrs ...any) {
	slog.InfoContext(ctx, msg, withContextAttrs(ctx, attrs...)...)
}

func Warn(ctx context.Context, msg string, attrs ...any) {
	slog.WarnContext(ctx, msg, withContextAttrs(ctx, attrs...)...)
}

func Error(ctx context.Context, msg string, err error, attrs ...any) {
	values := []any{"error", err}
	values = append(values, attrs...)
	slog.ErrorContext(ctx, msg, withContextAttrs(ctx, values...)...)
}

func withContextAttrs(ctx context.Context, attrs ...any) []any {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		return attrs
	}
	values := make([]any, 0, len(attrs)+2)
	values = append(values, "request_id", requestID)
	values = append(values, attrs...)
	return values
}
