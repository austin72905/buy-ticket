package controller

import (
	"net/http"
	"strconv"

	"buy-ticket/domain"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

const currentUserContextKey = "current_user"

func AttachCurrentUser(authService *service.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userIDValue, err := ctx.Cookie(sessionCookieName)
		if err != nil || userIDValue == "" {
			ctx.Next()
			return
		}

		userID, parseErr := strconv.ParseInt(userIDValue, 10, 64)
		if parseErr != nil {
			ctx.Next()
			return
		}

		user, serviceErr := authService.GetMe(ctx.Request.Context(), userID)
		if serviceErr == nil {
			ctx.Set(currentUserContextKey, user)
		}

		ctx.Next()
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if _, ok := CurrentUser(ctx); !ok {
			writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

func CurrentUser(ctx *gin.Context) (*domain.User, bool) {
	value, exists := ctx.Get(currentUserContextKey)
	if !exists {
		return nil, false
	}

	user, ok := value.(*domain.User)
	return user, ok
}
