package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const headerRequestID = "X-Request-Id"

func requestID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}

	if value := strings.TrimSpace(ctx.Writer.Header().Get(headerRequestID)); value != "" {
		return value
	}

	return strings.TrimSpace(ctx.GetHeader(headerRequestID))
}
