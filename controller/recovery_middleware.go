package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(
					ctx.Request.Context(),
					"http request panic recovered",
					"method", ctx.Request.Method,
					"path", ctx.Request.URL.Path,
					"panic", recovered,
					"stackTrace", string(debug.Stack()),
				)

				ctx.Abort()
				if ctx.Writer.Written() {
					return
				}

				appErr := newAppError(http.StatusInternalServerError, errors.New("internal server error"))
				appErr.RequestID = requestID(ctx)
				ctx.JSON(appErr.HTTPStatus, ErrorResponse{
					Code:      appErr.Code,
					Message:   appErr.Message,
					RequestID: appErr.RequestID,
				})
			}
		}()

		ctx.Next()
	}
}
