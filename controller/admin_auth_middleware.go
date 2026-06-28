package controller

import (
	"net/http"
	"strconv"

	"buy-ticket/domain"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

const currentAdminContextKey = "current_admin"

func AttachCurrentAdmin(adminAuthService *service.AdminAuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminUserIDValue, err := ctx.Cookie(adminSessionCookieName)
		if err != nil || adminUserIDValue == "" {
			ctx.Next()
			return
		}

		adminUserID, parseErr := strconv.ParseInt(adminUserIDValue, 10, 64)
		if parseErr != nil {
			ctx.Next()
			return
		}

		adminUser, serviceErr := adminAuthService.GetMe(ctx.Request.Context(), adminUserID)
		if serviceErr == nil {
			ctx.Set(currentAdminContextKey, adminUser)
		}

		ctx.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if _, ok := CurrentAdmin(ctx); !ok {
			writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

func RequireAdminRole(roles ...domain.AdminRole) gin.HandlerFunc {
	allowed := make(map[domain.AdminRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(ctx *gin.Context) {
		adminUser, ok := CurrentAdmin(ctx)
		if !ok {
			writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
			ctx.Abort()
			return
		}

		if _, exists := allowed[adminUser.Role]; !exists {
			writeError(ctx, http.StatusForbidden, service.ErrForbidden)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

func CurrentAdmin(ctx *gin.Context) (*domain.AdminUser, bool) {
	value, exists := ctx.Get(currentAdminContextKey)
	if !exists {
		return nil, false
	}

	adminUser, ok := value.(*domain.AdminUser)
	return adminUser, ok
}
