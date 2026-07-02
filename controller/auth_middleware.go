package controller

import (
	"net/http"

	"buy-ticket/domain"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

const currentUserContextKey = "current_user"

func AttachCurrentUser(authService *service.AuthService, sessionStore service.SessionStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, err := ctx.Cookie(sessionCookieName)
		if err != nil || token == "" {
			ctx.Next()
			return
		}

		userID, sessionErr := sessionStore.Get(ctx.Request.Context(), service.SessionKindUser, token)
		if sessionErr != nil {
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
