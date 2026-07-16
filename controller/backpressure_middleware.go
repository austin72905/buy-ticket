package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 限流
// 同一個 API process 內，最多同時處理 maxInFlight (500) 個 /queue/join。
// 第 501 個同時間進來的請求，直接回 429，叫 client 1 秒後再試。
// 避免開賣瞬間大量使用者同時按「加入排隊」把系統打爆
func QueueJoinBackpressure(maxInFlight int, retryAfter time.Duration) gin.HandlerFunc {
	if maxInFlight <= 0 {
		return func(ctx *gin.Context) {
			ctx.Next() // 去下一個handler
		}
	}

	guard := make(chan struct{}, maxInFlight)
	retryAfterSeconds := int(retryAfter.Seconds())
	if retryAfterSeconds <= 0 {
		retryAfterSeconds = 1
	}

	return func(ctx *gin.Context) {
		select {
		// 還能夠處理就進去處理
		case guard <- struct{}{}:
			// ctx.Next() 執行後面所有的 handler 全部處理完後釋放名額
			defer func() {
				<-guard
			}()
			ctx.Next()
		default:
			ctx.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
			writeError(ctx, http.StatusTooManyRequests, errors.New("queue join is busy, retry later"))
			ctx.Abort()
		}
	}
}
