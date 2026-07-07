package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	AdminAuthService *service.AdminAuthService
	AdminService     *service.AdminService
	SessionStore     service.SessionStore
}

func NewAdminController(adminAuthService *service.AdminAuthService, adminService *service.AdminService, sessionStore service.SessionStore) *AdminController {
	return &AdminController{
		AdminAuthService: adminAuthService,
		AdminService:     adminService,
		SessionStore:     sessionStore,
	}
}

func (c *AdminController) RegisterRoutes(router gin.IRouter) {
	admin := router.Group("/admin")
	admin.Use(AttachCurrentAdmin(c.AdminAuthService, c.SessionStore), RequireAdmin())
	admin.GET("/users", RequireAdminRole(domain.AdminRoleSuperAdmin), c.ListAdminUsers)
	admin.POST("/users", RequireAdminRole(domain.AdminRoleSuperAdmin), c.CreateAdminUser)
	admin.GET("/organizers", RequireAdminRole(domain.AdminRoleSuperAdmin), c.ListOrganizers)
	admin.POST("/organizers", RequireAdminRole(domain.AdminRoleSuperAdmin), c.CreateOrganizer)
	admin.GET("/orders", c.ListOrders)
	admin.POST("/orders/:orderId/reveal-sensitive", RequireAdminRole(domain.AdminRoleSuperAdmin), c.RevealOrderSensitive)
	admin.GET("/events", c.ListEvents)
	admin.POST("/events", c.CreateEvent)
	admin.GET("/events/:eventId", c.GetEvent)
	admin.GET("/events/:eventId/sections", c.ListEventSections)
	admin.POST("/events/:eventId/sections", c.CreateEventSection)
	admin.GET("/audit-logs", c.ListAuditLogs)
}

// AdminListUsers godoc
// @Summary List admin users
// @Description List backoffice admin users. Only SUPER_ADMIN can use this endpoint.
// @Tags admin-users
// @Produce json
// @Success 200 {array} AdminUserResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/users [get]
func (c *AdminController) ListAdminUsers(ctx *gin.Context) {
	adminUser, _ := CurrentAdmin(ctx)
	adminUsers, err := c.AdminService.ListAdminUsers(ctx.Request.Context(), adminUser)
	if err != nil {
		writeAdminError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newAdminUserResponses(adminUsers))
}

// AdminCreateUser godoc
// @Summary Create admin user
// @Description Create a backoffice admin user. Only SUPER_ADMIN can use this endpoint.
// @Tags admin-users
// @Accept json
// @Produce json
// @Param request body CreateAdminUserRequest true "Admin user payload"
// @Success 201 {object} AdminUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/users [post]
func (c *AdminController) CreateAdminUser(ctx *gin.Context) {
	var request CreateAdminUserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	created, err := c.AdminService.CreateAdminUser(ctx.Request.Context(), service.CreateAdminUserInput{
		AdminUser:   adminUser,
		OrganizerID: request.OrganizerID,
		Name:        request.Name,
		Email:       request.Email,
		Password:    request.Password,
		Role:        domain.AdminRole(request.Role),
	})
	if err != nil {
		writeAdminError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newAdminUserResponse(created))
}

// AdminListOrganizers godoc
// @Summary List organizers
// @Description List organizers. Only SUPER_ADMIN can use this endpoint.
// @Tags admin-organizers
// @Produce json
// @Success 200 {array} OrganizerResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/organizers [get]
func (c *AdminController) ListOrganizers(ctx *gin.Context) {
	adminUser, _ := CurrentAdmin(ctx)
	organizers, err := c.AdminService.ListOrganizers(ctx.Request.Context(), adminUser)
	if err != nil {
		writeAdminError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newOrganizerResponses(organizers))
}

// AdminCreateOrganizer godoc
// @Summary Create organizer
// @Description Create an organizer. Only SUPER_ADMIN can use this endpoint.
// @Tags admin-organizers
// @Accept json
// @Produce json
// @Param request body CreateOrganizerRequest true "Organizer payload"
// @Success 201 {object} OrganizerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/organizers [post]
func (c *AdminController) CreateOrganizer(ctx *gin.Context) {
	var request CreateOrganizerRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	organizer, err := c.AdminService.CreateOrganizer(ctx.Request.Context(), service.CreateOrganizerInput{
		AdminUser: adminUser,
		Name:      request.Name,
	})
	if err != nil {
		writeAdminError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newOrganizerResponse(organizer))
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

// AdminRevealOrderSensitive godoc
// @Summary Reveal order sensitive data
// @Description Reveal full user information for an order. Only SUPER_ADMIN can use this endpoint and a reason is required.
// @Tags admin-orders
// @Accept json
// @Produce json
// @Param orderId path int true "Order ID"
// @Param request body RevealSensitiveRequest true "Reveal reason"
// @Success 200 {object} AdminOrderSensitiveResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/orders/{orderId}/reveal-sensitive [post]
func (c *AdminController) RevealOrderSensitive(ctx *gin.Context) {
	orderID, err := strconv.ParseInt(ctx.Param("orderId"), 10, 64)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var request RevealSensitiveRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	ipAddress := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")
	order, err := c.AdminService.RevealOrderSensitive(ctx.Request.Context(), service.RevealOrderSensitiveInput{
		AdminUser: adminUser,
		OrderID:   orderID,
		Reason:    request.Reason,
		IPAddress: &ipAddress,
		UserAgent: &userAgent,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminRevealReason):
			writeError(ctx, http.StatusBadRequest, err)
		case errors.Is(err, service.ErrForbidden):
			writeError(ctx, http.StatusForbidden, err)
		case errors.Is(err, repository.ErrOrderNotFound):
			writeError(ctx, http.StatusNotFound, err)
		default:
			writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, newAdminOrderSensitiveResponse(order))
}

// AdminListEvents godoc
// @Summary List admin events
// @Description List events for admin backoffice. EVENT_ADMIN is scoped to its organizer.
// @Tags admin-events
// @Produce json
// @Success 200 {array} AdminEventResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/events [get]
func (c *AdminController) ListEvents(ctx *gin.Context) {
	adminUser, _ := CurrentAdmin(ctx)
	events, err := c.AdminService.ListEvents(ctx.Request.Context(), adminUser)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(ctx, http.StatusForbidden, err)
			return
		}
		writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, newAdminEventResponses(events))
}

// AdminCreateEvent godoc
// @Summary Create admin event
// @Description Create event for admin backoffice. SUPER_ADMIN must provide organizer_id; EVENT_ADMIN uses its own organizer.
// @Tags admin-events
// @Accept json
// @Produce json
// @Param request body CreateAdminEventRequest true "Event payload"
// @Success 201 {object} AdminEventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/events [post]
func (c *AdminController) CreateEvent(ctx *gin.Context) {
	var request CreateAdminEventRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	startAt, err := parseRequiredRFC3339(request.StartAt)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	endAt, err := parseRequiredRFC3339(request.EndAt)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	saleStartAt, err := parseRequiredRFC3339(request.SaleStartAt)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	saleEndAt, err := parseRequiredRFC3339(request.SaleEndAt)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	event, err := c.AdminService.CreateEvent(ctx.Request.Context(), service.CreateAdminEventInput{
		AdminUser:   adminUser,
		OrganizerID: request.OrganizerID,
		Name:        request.Name,
		Venue:       request.Venue,
		Status:      domain.EventStatus(request.Status),
		StartAt:     startAt,
		EndAt:       endAt,
		SaleStartAt: saleStartAt,
		SaleEndAt:   saleEndAt,
	})
	if err != nil {
		writeAdminError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newAdminEventResponse(*event))
}

// AdminGetEvent godoc
// @Summary Get admin event
// @Description Get an event for admin backoffice. EVENT_ADMIN is scoped to its organizer.
// @Tags admin-events
// @Produce json
// @Param eventId path int true "Event ID"
// @Success 200 {object} AdminEventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/events/{eventId} [get]
func (c *AdminController) GetEvent(ctx *gin.Context) {
	eventID, err := strconv.ParseInt(ctx.Param("eventId"), 10, 64)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	event, err := c.AdminService.GetEvent(ctx.Request.Context(), adminUser, eventID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(ctx, http.StatusForbidden, err)
		case errors.Is(err, repository.ErrEventNotFound):
			writeError(ctx, http.StatusNotFound, err)
		default:
			writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, newAdminEventResponse(*event))
}

// AdminListEventSections godoc
// @Summary List admin event sections
// @Description List event sections with inventory quantities. EVENT_ADMIN is scoped to its organizer.
// @Tags admin-events
// @Produce json
// @Param eventId path int true "Event ID"
// @Success 200 {array} AdminSectionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/events/{eventId}/sections [get]
func (c *AdminController) ListEventSections(ctx *gin.Context) {
	eventID, err := strconv.ParseInt(ctx.Param("eventId"), 10, 64)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	sections, err := c.AdminService.ListEventSections(ctx.Request.Context(), adminUser, eventID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(ctx, http.StatusForbidden, err)
		case errors.Is(err, repository.ErrEventNotFound):
			writeError(ctx, http.StatusNotFound, err)
		default:
			writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, newAdminSectionResponses(sections))
}

// AdminCreateEventSection godoc
// @Summary Create admin event section
// @Description Create section for an event. EVENT_ADMIN is scoped to its organizer.
// @Tags admin-events
// @Accept json
// @Produce json
// @Param eventId path int true "Event ID"
// @Param request body CreateAdminSectionRequest true "Section payload"
// @Success 201 {object} AdminSectionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/events/{eventId}/sections [post]
func (c *AdminController) CreateEventSection(ctx *gin.Context) {
	eventID, err := strconv.ParseInt(ctx.Param("eventId"), 10, 64)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var request CreateAdminSectionRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	adminUser, _ := CurrentAdmin(ctx)
	section, err := c.AdminService.CreateEventSection(ctx.Request.Context(), service.CreateAdminSectionInput{
		AdminUser:     adminUser,
		EventID:       eventID,
		Name:          request.Name,
		Price:         request.Price,
		TotalQuantity: request.TotalQuantity,
		PurchaseLimit: request.PurchaseLimit,
		Status:        domain.SectionStatus(request.Status),
	})
	if err != nil {
		writeAdminError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newAdminSectionResponse(*section))
}

// AdminListAuditLogs godoc
// @Summary List admin audit logs
// @Description List admin audit logs with keyset pagination. SUPER_ADMIN can see all logs; EVENT_ADMIN can only see its own logs.
// @Tags admin-audit-logs
// @Produce json
// @Param limit query int false "Page size, default 20, max 100"
// @Param cursor_created_at query string false "Cursor created_at in RFC3339"
// @Param cursor_id query int false "Cursor audit log id"
// @Success 200 {object} AdminAuditLogListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/audit-logs [get]
func (c *AdminController) ListAuditLogs(ctx *gin.Context) {
	limit, err := parseOptionalIntQuery(ctx, "limit")
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

	logs, err := c.AdminService.ListAuditLogs(ctx.Request.Context(), service.ListAdminAuditLogsInput{
		AdminUser:       adminUser,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		Limit:           queryLimit + 1,
	})
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	hasNext := len(logs) > normalizeAdminResponseLimit(queryLimit)
	if hasNext {
		logs = logs[:normalizeAdminResponseLimit(queryLimit)]
	}

	ctx.JSON(http.StatusOK, newAdminAuditLogListResponse(logs, hasNext))
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

func parseRequiredRFC3339(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("time is required")
	}
	return time.Parse(time.RFC3339, value)
}

func writeAdminError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidAdminInput):
		writeError(ctx, http.StatusBadRequest, err)
	case errors.Is(err, service.ErrUnauthorized):
		writeError(ctx, http.StatusUnauthorized, err)
	case errors.Is(err, service.ErrForbidden):
		writeError(ctx, http.StatusForbidden, err)
	case errors.Is(err, repository.ErrEventNotFound),
		errors.Is(err, repository.ErrOrganizerNotFound),
		errors.Is(err, repository.ErrAdminUserNotFound):
		writeError(ctx, http.StatusNotFound, err)
	default:
		writeError(ctx, http.StatusInternalServerError, err)
	}
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
