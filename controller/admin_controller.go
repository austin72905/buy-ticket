package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"buy-ticket/domain"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	AdminAuthService *service.AdminAuthService
	AdminService     *service.AdminService
}

func NewAdminController(adminAuthService *service.AdminAuthService, adminService *service.AdminService) *AdminController {
	return &AdminController{
		AdminAuthService: adminAuthService,
		AdminService:     adminService,
	}
}

func (c *AdminController) RegisterRoutes(router gin.IRouter) {
	admin := router.Group("/admin")
	admin.Use(AttachCurrentAdmin(c.AdminAuthService), RequireAdmin())
	admin.GET("/orders", c.ListOrders)
}

// AdminListOrders godoc
// @Summary List admin orders
// @Description List orders for admin backoffice with keyset pagination. EVENT_ADMIN is scoped to its organizer.
// @Tags admin-orders
// @Produce json
// @Param limit query int false "Page size, default 20, max 100"
// @Param cursor_created_at query string false "Cursor created_at in RFC3339"
// @Param cursor_id query int false "Cursor order id"
// @Param event_id query int false "Filter by event id"
// @Param user_id query int false "Filter by user id"
// @Param status query int false "Filter by order status"
// @Success 200 {object} AdminOrderListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/orders [get]
func (c *AdminController) ListOrders(ctx *gin.Context) {
	limit, err := parseOptionalIntQuery(ctx, "limit")
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	eventID, err := parseOptionalInt64Query(ctx, "event_id")
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	userID, err := parseOptionalInt64Query(ctx, "user_id")
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	statusValue, err := parseOptionalIntQuery(ctx, "status")
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	cursorID, err := parseOptionalInt64Query(ctx, "cursor_id")
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	cursorCreatedAt, err := parseOptionalTimeQuery(ctx, "cursor_created_at")
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	queryLimit := limit
	if queryLimit <= 0 {
		queryLimit = 20
	}

	orders, err := c.AdminService.ListOrders(ctx.Request.Context(), service.ListAdminOrdersInput{
		AdminUser:       adminUser,
		EventID:         eventID,
		UserID:          userID,
		Status:          domain.OrderStatus(statusValue),
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		Limit:           queryLimit + 1,
	})
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(ctx, http.StatusForbidden, err)
			return
		}
		writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	hasNext := len(orders) > normalizeAdminResponseLimit(queryLimit)
	if hasNext {
		orders = orders[:normalizeAdminResponseLimit(queryLimit)]
	}

	ctx.JSON(http.StatusOK, newAdminOrderListResponse(orders, hasNext))
}

func parseOptionalIntQuery(ctx *gin.Context, name string) (int, error) {
	value := ctx.Query(name)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseOptionalInt64Query(ctx *gin.Context, name string) (int64, error) {
	value := ctx.Query(name)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func parseOptionalTimeQuery(ctx *gin.Context, name string) (*time.Time, error) {
	value := ctx.Query(name)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func normalizeAdminResponseLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}
