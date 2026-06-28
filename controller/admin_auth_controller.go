package controller

import (
	"net/http"
	"strconv"
	"time"

	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

const adminSessionCookieName = "buy_ticket_admin_session"

type AdminAuthController struct {
	AdminAuthService *service.AdminAuthService
}

func NewAdminAuthController(adminAuthService *service.AdminAuthService) *AdminAuthController {
	return &AdminAuthController{
		AdminAuthService: adminAuthService,
	}
}

func (c *AdminAuthController) RegisterRoutes(router gin.IRouter) {
	router.POST("/admin/auth/login", c.Login)

	authenticated := router.Group("/admin")
	authenticated.Use(AttachCurrentAdmin(c.AdminAuthService), RequireAdmin())
	authenticated.POST("/auth/logout", c.Logout)
	authenticated.GET("/me", c.Me)
}

// AdminLogin godoc
// @Summary Admin login
// @Description Login with an admin account and create an admin session cookie.
// @Tags admin-auth
// @Accept json
// @Produce json
// @Param request body AdminLoginRequest true "Admin login request"
// @Success 200 {object} AdminUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/auth/login [post]
func (c *AdminAuthController) Login(ctx *gin.Context) {
	var request AdminLoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, err := c.AdminAuthService.Login(ctx.Request.Context(), service.AdminLoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeError(ctx, http.StatusUnauthorized, err)
		return
	}

	writeAdminSessionCookie(ctx, adminUser.ID)
	ctx.JSON(http.StatusOK, newAdminUserResponse(adminUser))
}

// AdminLogout godoc
// @Summary Admin logout
// @Description Clear the current admin session cookie.
// @Tags admin-auth
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Router /admin/auth/logout [post]
func (c *AdminAuthController) Logout(ctx *gin.Context) {
	clearAdminSessionCookie(ctx)
	ctx.Status(http.StatusNoContent)
}

// AdminMe godoc
// @Summary Get current admin
// @Description Get the currently authenticated admin user.
// @Tags admin-auth
// @Produce json
// @Success 200 {object} AdminUserResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/me [get]
func (c *AdminAuthController) Me(ctx *gin.Context) {
	adminUser, _ := CurrentAdmin(ctx)
	ctx.JSON(http.StatusOK, newAdminUserResponse(adminUser))
}

func writeAdminSessionCookie(ctx *gin.Context, adminUserID int64) {
	maxAge := int((7 * 24 * time.Hour).Seconds())
	ctx.SetCookie(adminSessionCookieName, strconv.FormatInt(adminUserID, 10), maxAge, "/", "", false, true)
}

func clearAdminSessionCookie(ctx *gin.Context) {
	ctx.SetCookie(adminSessionCookieName, "", -1, "/", "", false, true)
}
