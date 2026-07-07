package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func QueueJoinBackpressure(maxInFlight int, retryAfter time.Duration) gin.HandlerFunc {
	if maxInFlight <= 0 {
		return func(ctx *gin.Context) {
			ctx.Next()
		}
	}

	guard := make(chan struct{}, maxInFlight)
	retryAfterSeconds := int(retryAfter.Seconds())
	if retryAfterSeconds <= 0 {
		retryAfterSeconds = 1
	}

	return func(ctx *gin.Context) {
		select {
		case guard <- struct{}{}:
			defer func() { <-guard }()
			ctx.Next()
		default:
			ctx.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
			appErr := newAppError(http.StatusTooManyRequests, errors.New("queue join is busy, retry later"))
			appErr.RequestID = requestID(ctx)
			ctx.JSON(appErr.HTTPStatus, ErrorResponse{
				Code:      appErr.Code,
				Message:   appErr.Message,
				RequestID: appErr.RequestID,
			})
			ctx.Abort()
		}
	}
}
