package controller

import (
	"net/http"
	"strconv"

	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

type BookingController struct {
	BookingService *service.BookingService
}

func NewBookingController(bookingService *service.BookingService) *BookingController {
	return &BookingController{
		BookingService: bookingService,
	}
}

func (c *BookingController) RegisterRoutes(router gin.IRouter) {
	router.GET("/events", c.ListEvents)
	router.GET("/events/:eventId", c.GetEvent)
	router.GET("/events/:eventId/sections", c.GetSections)
	router.GET("/events/:eventId/availability", c.GetAvailability)
	router.GET("/users/:userId/reservations", c.ListUserReservations)
	router.GET("/users/:userId/orders", c.ListUserOrders)
	router.GET("/reservations/:reservationId", c.GetReservation)
	router.GET("/orders/:orderId", c.GetOrder)
	router.GET("/orders/order-no/:orderNo", c.GetOrderByOrderNo)
	router.GET("/payments/:paymentNo", c.GetPaymentByPaymentNo)
	router.POST("/reservations", c.ReserveTicket)
	router.POST("/orders", c.CreateOrder)
	router.POST("/payments", c.PayOrder)
	router.POST("/reservations/expire", c.ExpireReservation)
	router.POST("/reservations/cancel", c.CancelReservation)
}

// ListEvents godoc
// @Summary 查詢活動列表
// @Description 取得目前系統中的活動列表
// @Tags events
// @Produce json
// @Success 200 {array} EventResponse
// @Failure 500 {object} ErrorResponse
// @Router /events [get]
func (c *BookingController) ListEvents(ctx *gin.Context) {
	events, err := c.BookingService.ListEvents(ctx.Request.Context())
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	response := make([]EventResponse, 0, len(events))
	for _, event := range events {
		response = append(response, newEventResponse(&event))
	}

	ctx.JSON(http.StatusOK, response)
}

// GetEvent godoc
// @Summary 取得活動資訊
// @Description 依活動 ID 取得活動基本資訊
// @Tags events
// @Produce json
// @Param eventId path int true "活動 ID"
// @Success 200 {object} EventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /events/{eventId} [get]
func (c *BookingController) GetEvent(ctx *gin.Context) {
	eventID, ok := parseInt64Param(ctx, "eventId")
	if !ok {
		return
	}

	event, err := c.BookingService.GetEvent(ctx.Request.Context(), eventID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, newEventResponse(event))
}

// GetSections godoc
// @Summary 取得活動票區
// @Description 依活動 ID 取得票區列表
// @Tags events
// @Produce json
// @Param eventId path int true "活動 ID"
// @Success 200 {array} SectionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /events/{eventId}/sections [get]
func (c *BookingController) GetSections(ctx *gin.Context) {
	eventID, ok := parseInt64Param(ctx, "eventId")
	if !ok {
		return
	}

	sections, err := c.BookingService.GetSections(ctx.Request.Context(), eventID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	response := make([]SectionResponse, 0, len(sections))
	for _, section := range sections {
		response = append(response, newSectionResponse(section))
	}

	ctx.JSON(http.StatusOK, response)
}

// GetAvailability godoc
// @Summary 取得票區可售量
// @Description 依活動 ID 取得各票區剩餘可售量
// @Tags events
// @Produce json
// @Param eventId path int true "活動 ID"
// @Success 200 {array} SectionAvailabilityResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /events/{eventId}/availability [get]
func (c *BookingController) GetAvailability(ctx *gin.Context) {
	eventID, ok := parseInt64Param(ctx, "eventId")
	if !ok {
		return
	}

	availabilities, err := c.BookingService.GetAvailability(ctx.Request.Context(), eventID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	response := make([]SectionAvailabilityResponse, 0, len(availabilities))
	for _, availability := range availabilities {
		response = append(response, newSectionAvailabilityResponse(availability))
	}

	ctx.JSON(http.StatusOK, response)
}

// ListUserReservations godoc
// @Summary 查詢使用者 reservation 列表
// @Description 依照 user ID 取得該使用者的鎖票列表
// @Tags reservations
// @Produce json
// @Param userId path int true "使用者 ID"
// @Success 200 {array} ReservationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{userId}/reservations [get]
func (c *BookingController) ListUserReservations(ctx *gin.Context) {
	userID, ok := parseInt64Param(ctx, "userId")
	if !ok {
		return
	}

	reservations, err := c.BookingService.ListReservationsByUserID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	response := make([]ReservationResponse, 0, len(reservations))
	for _, reservation := range reservations {
		response = append(response, newReservationResponse(&reservation))
	}

	ctx.JSON(http.StatusOK, response)
}

// ListUserOrders godoc
// @Summary 查詢使用者訂單列表
// @Description 依照 user ID 取得該使用者的訂單列表
// @Tags orders
// @Produce json
// @Param userId path int true "使用者 ID"
// @Success 200 {array} OrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{userId}/orders [get]
func (c *BookingController) ListUserOrders(ctx *gin.Context) {
	userID, ok := parseInt64Param(ctx, "userId")
	if !ok {
		return
	}

	orders, err := c.BookingService.ListOrdersByUserID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	response := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, newOrderResponse(&order))
	}

	ctx.JSON(http.StatusOK, response)
}

// GetReservation godoc
// @Summary 查詢 reservation
// @Description 依照 reservation ID 取得鎖票資料
// @Tags reservations
// @Produce json
// @Param reservationId path int true "reservation ID"
// @Success 200 {object} ReservationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /reservations/{reservationId} [get]
func (c *BookingController) GetReservation(ctx *gin.Context) {
	reservationID, ok := parseInt64Param(ctx, "reservationId")
	if !ok {
		return
	}

	reservation, err := c.BookingService.GetReservation(ctx.Request.Context(), reservationID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, newReservationResponse(reservation))
}

// GetOrder godoc
// @Summary 取得訂單資訊
// @Description 依訂單 ID 取得訂單內容
// @Tags orders
// @Produce json
// @Param orderId path int true "訂單 ID"
// @Success 200 {object} OrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /orders/{orderId} [get]
func (c *BookingController) GetOrder(ctx *gin.Context) {
	orderID, ok := parseInt64Param(ctx, "orderId")
	if !ok {
		return
	}

	order, err := c.BookingService.GetOrder(ctx.Request.Context(), orderID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, newOrderResponse(order))
}

// GetOrderByOrderNo godoc
// @Summary 依訂單編號查詢訂單
// @Description 依照 order_no 取得訂單資料
// @Tags orders
// @Produce json
// @Param orderNo path string true "訂單編號"
// @Success 200 {object} OrderResponse
// @Failure 404 {object} ErrorResponse
// @Router /orders/order-no/{orderNo} [get]
func (c *BookingController) GetOrderByOrderNo(ctx *gin.Context) {
	order, err := c.BookingService.GetOrderByOrderNo(ctx.Request.Context(), ctx.Param("orderNo"))
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, newOrderResponse(order))
}

// GetPaymentByPaymentNo godoc
// @Summary 依付款編號查詢付款
// @Description 依照 payment_no 取得付款資料
// @Tags payments
// @Produce json
// @Param paymentNo path string true "付款編號"
// @Success 200 {object} PaymentResponse
// @Failure 404 {object} ErrorResponse
// @Router /payments/{paymentNo} [get]
func (c *BookingController) GetPaymentByPaymentNo(ctx *gin.Context) {
	payment, err := c.BookingService.GetPaymentByPaymentNo(ctx.Request.Context(), ctx.Param("paymentNo"))
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, newPaymentResponse(payment))
}

// ReserveTicket godoc
// @Summary 保留票券
// @Description 建立 reservation 並保留票區數量
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body ReserveTicketRequest true "保留票券請求"
// @Success 201 {object} ReservationResponse
// @Failure 400 {object} ErrorResponse
// @Router /reservations [post]
func (c *BookingController) ReserveTicket(ctx *gin.Context) {
	var request ReserveTicketRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	reservation, err := c.BookingService.ReserveTicket(ctx.Request.Context(), service.ReserveTicketInput{
		UserID:    request.UserID,
		EventID:   request.EventID,
		SectionID: request.SectionID,
		Quantity:  request.Quantity,
		HoldUntil: request.HoldUntil,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusCreated, newReservationResponse(reservation))
}

// CreateOrder godoc
// @Summary 建立訂單
// @Description 由 reservation 建立待付款訂單
// @Tags orders
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "建立訂單請求"
// @Success 201 {object} OrderResponse
// @Failure 400 {object} ErrorResponse
// @Router /orders [post]
func (c *BookingController) CreateOrder(ctx *gin.Context) {
	var request CreateOrderRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	order, err := c.BookingService.CreateOrder(ctx.Request.Context(), service.CreateOrderInput{
		ReservationID: request.ReservationID,
		OrderNo:       request.OrderNo,
		ExpiresAt:     request.ExpiresAt,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusCreated, newOrderResponse(order))
}

// PayOrder godoc
// @Summary 訂單付款
// @Description 付款成功後確認 reservation 並轉成售出
// @Tags payments
// @Accept json
// @Produce json
// @Param request body PayOrderRequest true "付款請求"
// @Success 200 {object} PaymentResponse
// @Failure 400 {object} ErrorResponse
// @Router /payments [post]
func (c *BookingController) PayOrder(ctx *gin.Context) {
	var request PayOrderRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	payment, err := c.BookingService.PayOrder(ctx.Request.Context(), service.PayOrderInput{
		OrderID:   request.OrderID,
		PaymentNo: request.PaymentNo,
		Method:    request.Method,
		Amount:    request.Amount,
		PaidAt:    request.PaidAt,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusOK, newPaymentResponse(payment))
}

// ExpireReservation godoc
// @Summary 過期 reservation
// @Description 將 reservation 標記為 expired 並釋放保留量
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body ExpireReservationRequest true "過期請求"
// @Success 200 {object} ReservationResponse
// @Failure 400 {object} ErrorResponse
// @Router /reservations/expire [post]
func (c *BookingController) ExpireReservation(ctx *gin.Context) {
	var request ExpireReservationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	reservation, err := c.BookingService.ExpireReservation(ctx.Request.Context(), service.ExpireReservationInput{
		ReservationID: request.ReservationID,
		ExpiredAt:     request.ExpiredAt,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusOK, newReservationResponse(reservation))
}

// CancelReservation godoc
// @Summary 取消 reservation
// @Description 主動取消 reservation 並釋放保留量
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body CancelReservationRequest true "取消請求"
// @Success 200 {object} ReservationResponse
// @Failure 400 {object} ErrorResponse
// @Router /reservations/cancel [post]
func (c *BookingController) CancelReservation(ctx *gin.Context) {
	var request CancelReservationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	reservation, err := c.BookingService.CancelReservation(ctx.Request.Context(), service.CancelReservationInput{
		ReservationID: request.ReservationID,
		CancelledAt:   request.CancelledAt,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusOK, newReservationResponse(reservation))
}

func writeError(ctx *gin.Context, statusCode int, err error) {
	ctx.JSON(statusCode, gin.H{
		"error": err.Error(),
	})
}

func parseInt64Param(ctx *gin.Context, key string) (int64, bool) {
	value, err := strconv.ParseInt(ctx.Param(key), 10, 64)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return 0, false
	}

	return value, true
}
