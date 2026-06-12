package controller

import (
	"net/http"
	"strconv"
	"time"

	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "buy_ticket_session"

type AuthController struct {
	AuthService *service.AuthService
}

func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{
		AuthService: authService,
	}
}

func (c *AuthController) RegisterRoutes(router gin.IRouter) {
	router.POST("/auth/register", c.Register)
	router.POST("/auth/login", c.Login)

	authenticated := router.Group("/")
	authenticated.Use(AttachCurrentUser(c.AuthService), RequireAuth())
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

	writeSessionCookie(ctx, user.ID)
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

	writeSessionCookie(ctx, user.ID)
	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func (c *AuthController) Logout(ctx *gin.Context) {
	clearSessionCookie(ctx)
	ctx.Status(http.StatusNoContent)
}

func (c *AuthController) Me(ctx *gin.Context) {
	user, _ := CurrentUser(ctx)
	ctx.JSON(http.StatusOK, newUserResponse(user))
}

func writeSessionCookie(ctx *gin.Context, userID int64) {
	maxAge := int((7 * 24 * time.Hour).Seconds())
	ctx.SetCookie(sessionCookieName, strconv.FormatInt(userID, 10), maxAge, "/", "", false, true)
}

func clearSessionCookie(ctx *gin.Context) {
	ctx.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
}
