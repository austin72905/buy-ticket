package controller

import (
	"net/http"
	"time"

	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "buy_ticket_session"

type AuthController struct {
	AuthService  *service.AuthService
	SessionStore service.SessionStore
	SessionTTL   time.Duration
}

func NewAuthController(authService *service.AuthService, sessionStore service.SessionStore, sessionTTL time.Duration) *AuthController {
	return &AuthController{
		AuthService:  authService,
		SessionStore: sessionStore,
		SessionTTL:   sessionTTL,
	}
}

func (c *AuthController) RegisterRoutes(router gin.IRouter) {
	router.POST("/auth/register", c.Register)
	router.POST("/auth/login", c.Login)

	authenticated := router.Group("/")
	authenticated.Use(AttachCurrentUser(c.AuthService, c.SessionStore), RequireAuth())
	authenticated.POST("/auth/logout", c.Logout)
	authenticated.GET("/me", c.Me)
}

func (c *AuthController) Register(ctx *gin.Context) {
	var request RegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	user, err := c.AuthService.Register(ctx.Request.Context(), service.RegisterInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	token, err := c.SessionStore.Create(ctx.Request.Context(), service.SessionKindUser, user.ID, c.SessionTTL)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	writeSessionCookie(ctx, token, c.SessionTTL)
	ctx.JSON(http.StatusCreated, newUserResponse(user))
}

func (c *AuthController) Login(ctx *gin.Context) {
	var request LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	user, err := c.AuthService.Login(ctx.Request.Context(), service.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeError(ctx, http.StatusUnauthorized, err)
		return
	}

	token, err := c.SessionStore.Create(ctx.Request.Context(), service.SessionKindUser, user.ID, c.SessionTTL)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	writeSessionCookie(ctx, token, c.SessionTTL)
	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (c *AuthController) Logout(ctx *gin.Context) {
	if token, err := ctx.Cookie(sessionCookieName); err == nil && token != "" {
		_ = c.SessionStore.Delete(ctx.Request.Context(), service.SessionKindUser, token)
	}
	clearSessionCookie(ctx)
	ctx.Status(http.StatusNoContent)
}

func (c *AuthController) Me(ctx *gin.Context) {
	user, _ := CurrentUser(ctx)
	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func writeSessionCookie(ctx *gin.Context, token string, ttl time.Duration) {
	maxAge := int(ttl.Seconds())
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookieName, token, maxAge, "/", "", false, true)
}

func clearSessionCookie(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
}
