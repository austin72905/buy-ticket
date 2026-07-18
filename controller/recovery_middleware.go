package controller

import (
	"errors"
	"net/http"
	"runtime/debug"

	"buy-ticket/observability"

	"github.com/gin-gonic/gin"
)

// global panic handler
func RecoveryMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				observability.Error(
					ctx.Request.Context(),
					"http request panic recovered",
					nil,
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
