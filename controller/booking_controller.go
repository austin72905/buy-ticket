package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

type BookingController struct {
	BookingService       *service.BookingService
	QueueJoinMaxInFlight int
	QueueJoinRetryAfter  time.Duration
}

const (
	idempotencyKeyHeader     = "Idempotency-Key"
	paymentIdempotencyRoute  = "POST /payments"
	paymentIdempotencyMaxLen = 255
)

func NewBookingController(bookingService *service.BookingService) *BookingController {
	return &BookingController{
		BookingService: bookingService,
	}
}

func (c *BookingController) RegisterRoutes(router gin.IRouter) {
	router.GET("/sale/status", c.GetSaleStatus)
	router.GET("/queue/status/:queueToken", c.GetQueueStatus)
	router.GET("/events", c.ListEvents)
	router.GET("/events/:eventId", c.GetEvent)
	router.GET("/events/:eventId/sections", c.GetSections)
	router.GET("/events/:eventId/availability", c.GetAvailability)
	router.GET("/reservations/:reservationId", c.GetReservation)
	router.GET("/orders/:orderId", c.GetOrder)
	router.GET("/orders/order-no/:orderNo", c.GetOrderByOrderNo)
	router.GET("/payments/:paymentNo", c.GetPaymentByPaymentNo)
	router.POST("/payments/provider/ecpay/callback", c.HandleECPayCallback)

	// 以下掛在 authenticated 上的 route 都會先經過 RequireAuth()
	authenticated := router.Group("/")
	authenticated.Use(RequireAuth())
	authenticated.GET("/me/reservations", c.ListUserReservations)
	authenticated.GET("/me/orders", c.ListUserOrders)
	authenticated.GET("/me/payments", c.ListUserPayments)
	queueJoinHandlers := []gin.HandlerFunc{c.JoinQueue}
	if c.QueueJoinMaxInFlight > 0 {
		queueJoinHandlers = append([]gin.HandlerFunc{
			QueueJoinBackpressure(c.QueueJoinMaxInFlight, c.QueueJoinRetryAfter),
		}, queueJoinHandlers...)
	}
	authenticated.POST("/queue/join", queueJoinHandlers...) // 可以接多個handler  router.POST("/queue/join", RequireAuth(), QueueJoinBackpressure(...), c.JoinQueue)
	authenticated.POST("/reservations", c.ReserveTicket)
	authenticated.POST("/orders", c.CreateOrder)
	authenticated.POST("/payments", c.PayOrder)
	authenticated.POST("/payments/start", c.StartPayment)
	authenticated.POST("/reservations/expire", c.ExpireReservation)
	authenticated.POST("/reservations/cancel", c.CancelReservation)
}

// JoinQueue godoc
// @Summary Join queue
// @Description Join the queue for an event and return queue status.
// @Tags queue
// @Accept json
// @Produce json
// @Param request body JoinQueueRequest true "Join queue request"
// @Success 201 {object} JoinQueueResponse
// @Failure 400 {object} ErrorResponse
// @Router /queue/join [post]
func (c *BookingController) JoinQueue(ctx *gin.Context) {
	var request JoinQueueRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	status, err := c.BookingService.JoinQueue(ctx.Request.Context(), service.JoinQueueInput{
		EventID:    request.EventID,
		UserID:     user.ID,
		ClientID:   request.ClientID,
		RequestID:  request.RequestID,
		Channel:    request.Channel,
		AccessCode: request.AccessCode,
	}, time.Now())
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusCreated, newJoinQueueResponse(status))
}

// GetQueueStatus godoc
// @Summary 查詢排隊狀態
// @Description 依 queue token 查詢目前排隊進度與是否已取得 purchase token
// @Tags queue
// @Produce json
// @Param queueToken path string true "Queue Token"
// @Success 200 {object} QueueStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /queue/status/{queueToken} [get]
func (c *BookingController) GetQueueStatus(ctx *gin.Context) {
	queueToken := ctx.Param("queueToken")
	if queueToken == "" {
		writeError(ctx, http.StatusBadRequest, errors.New("invalid queue token"))
		return
	}

	status, err := c.BookingService.GetQueueStatus(ctx.Request.Context(), queueToken)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, newQueueStatusResponse(status))
}

// GetSaleStatus godoc
// @Summary 查詢售票狀態
// @Description 依照 event_id 查詢活動是否開賣，以及是否可進入後續購票流程
// @Tags sale
// @Produce json
// @Param event_id query int true "活動 ID"
// @Success 200 {object} SaleStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /sale/status [get]
func (c *BookingController) GetSaleStatus(ctx *gin.Context) {
	eventID, err := strconv.ParseInt(ctx.Query("event_id"), 10, 64)
	if err != nil || eventID <= 0 {
		writeError(ctx, http.StatusBadRequest, errors.New("invalid event_id"))
		return
	}

	status, svcErr := c.BookingService.GetSaleStatus(ctx.Request.Context(), eventID, time.Now())
	if svcErr != nil {
		writeError(ctx, http.StatusNotFound, svcErr)
		return
	}

	ctx.JSON(http.StatusOK, newSaleStatusResponse(status))
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
// @Summary List my reservations
// @Description List reservations for the current authenticated user.
// @Tags reservations
// @Produce json
// @Success 200 {array} ReservationResponse
// @Failure 401 {object} ErrorResponse
// @Router /me/reservations [get]
func (c *BookingController) ListUserReservations(ctx *gin.Context) {
	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	reservations, err := c.BookingService.ListReservationsByUserID(ctx.Request.Context(), user.ID)
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
// @Summary List my orders
// @Description List orders for the current authenticated user.
// @Tags orders
// @Produce json
// @Success 200 {array} OrderResponse
// @Failure 401 {object} ErrorResponse
// @Router /me/orders [get]
func (c *BookingController) ListUserOrders(ctx *gin.Context) {
	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	orders, err := c.BookingService.ListOrdersByUserID(ctx.Request.Context(), user.ID)
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

// ListUserPayments godoc
// @Summary List my payments
// @Description List payments for the current authenticated user.
// @Tags payments
// @Produce json
// @Success 200 {array} PaymentResponse
// @Failure 401 {object} ErrorResponse
// @Router /me/payments [get]
func (c *BookingController) ListUserPayments(ctx *gin.Context) {
	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	payments, err := c.BookingService.ListPaymentsByUserID(ctx.Request.Context(), user.ID)
	if err != nil {
		writeError(ctx, http.StatusNotFound, err)
		return
	}

	response := make([]PaymentResponse, 0, len(payments))
	for _, payment := range payments {
		response = append(response, newPaymentResponse(&payment))
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

	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	reservation, err := c.BookingService.ReserveTicket(ctx.Request.Context(), service.ReserveTicketInput{
		UserID:        user.ID,
		EventID:       request.EventID,
		SectionID:     request.SectionID,
		Quantity:      request.Quantity,
		HoldUntil:     request.HoldUntil,
		PurchaseToken: request.PurchaseToken,
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

	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	reservation, err := c.BookingService.GetReservation(ctx.Request.Context(), request.ReservationID)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if reservation.UserID != user.ID {
		writeError(ctx, http.StatusForbidden, service.ErrUnauthorized)
		return
	}

	order, err := c.BookingService.CreateOrder(ctx.Request.Context(), service.CreateOrderInput{
		ReservationID: request.ReservationID,
		OrderNo:       request.OrderNo,
		PurchaseToken: request.PurchaseToken,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusCreated, newOrderResponse(order))
}

// PayOrder godoc
// @Summary Pay order
// @Description Pay a pending order. If Idempotency-Key is provided, retries with the same request replay the first successful payment response.
// @Tags payments
// @Accept json
// @Produce json
// @Param Idempotency-Key header string false "Idempotency key for safe payment retries"
// @Param request body PayOrderRequest true "Payment request"
// @Success 200 {object} PaymentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /payments [post]
func (c *BookingController) PayOrder(ctx *gin.Context) {
	var request PayOrderRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	order, err := c.BookingService.GetOrder(ctx.Request.Context(), request.OrderID)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if order.UserID != user.ID {
		writeError(ctx, http.StatusForbidden, service.ErrUnauthorized)
		return
	}

	idempotencyKey := ctx.GetHeader(idempotencyKeyHeader)
	if len(idempotencyKey) > paymentIdempotencyMaxLen {
		writeError(ctx, http.StatusBadRequest, errors.New("idempotency key is too long"))
		return
	}
	requestHash, err := hashPaymentRequest(request)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if idempotencyKey != "" {
		record, replay, err := c.BookingService.BeginPaymentIdempotency(ctx.Request.Context(), service.BeginIdempotencyInput{
			Key:         idempotencyKey,
			UserID:      user.ID,
			Endpoint:    paymentIdempotencyRoute,
			RequestHash: requestHash,
			Now:         time.Now(),
		})
		if err != nil {
			if errors.Is(err, service.ErrIdempotencyConflict) || errors.Is(err, service.ErrIdempotencyInProgress) {
				writeError(ctx, http.StatusConflict, err)
				return
			}
			writeError(ctx, http.StatusBadRequest, err)
			return
		}
		if replay {
			ctx.Data(*record.ResponseStatus, "application/json; charset=utf-8", record.ResponseBody)
			return
		}
	}

	payment, err := c.BookingService.PayOrder(ctx.Request.Context(), service.PayOrderInput{
		OrderID: request.OrderID,
		Method:  request.Method,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	response := newPaymentResponse(payment)
	if idempotencyKey != "" {
		responseBody, err := json.Marshal(response)
		if err != nil {
			writeError(ctx, http.StatusBadRequest, err)
			return
		}
		if err := c.BookingService.CompletePaymentIdempotency(ctx.Request.Context(), user.ID, idempotencyKey, paymentIdempotencyRoute, http.StatusOK, responseBody, time.Now()); err != nil {
			writeError(ctx, http.StatusBadRequest, err)
			return
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// StartPayment godoc
// @Summary Start provider payment
// @Description Create a payment attempt for a pending order. This does not mark the order as paid.
// @Tags payments
// @Accept json
// @Produce json
// @Param Idempotency-Key header string false "Idempotency key for safe payment start retries"
// @Param request body StartPaymentRequest true "Start payment request"
// @Success 202 {object} PaymentAttemptResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /payments/start [post]
func (c *BookingController) StartPayment(ctx *gin.Context) {
	var request StartPaymentRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	order, err := c.BookingService.GetOrder(ctx.Request.Context(), request.OrderID)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if order.UserID != user.ID {
		writeError(ctx, http.StatusForbidden, service.ErrUnauthorized)
		return
	}

	idempotencyKey := ctx.GetHeader(idempotencyKeyHeader)
	if len(idempotencyKey) > paymentIdempotencyMaxLen {
		writeError(ctx, http.StatusBadRequest, errors.New("idempotency key is too long"))
		return
	}
	requestPayload, err := json.Marshal(request)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	// 開始一次支付流程
	attempt, err := c.BookingService.StartMockPaymentAttempt(ctx.Request.Context(), service.CreatePaymentAttemptInput{
		OrderID:        request.OrderID,
		IdempotencyKey: idempotencyKey,
		Provider:       request.Provider,
		Method:         request.Method,
		RequestPayload: requestPayload,
	})
	if err != nil {
		if errors.Is(err, service.ErrPaymentAttemptIdempotencyConflict) {
			writeError(ctx, http.StatusConflict, err)
			return
		}
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusAccepted, newPaymentAttemptResponse(attempt))
}

// HandleECPayCallback godoc
// @Summary Handle ECPay callback
// @Description Handle ECPay callback and update payment status by MerchantTradeNo and RtnCode.
// @Tags payments
// @Accept x-www-form-urlencoded
// @Produce plain
// @Param MerchantID formData string false "Merchant ID"
// @Param MerchantTradeNo formData string true "Merchant Trade Number"
// @Param RtnCode formData string true "Return Code"
// @Param RtnMsg formData string false "Return Message"
// @Param TradeNo formData string false "Trade Number"
// @Param TradeAmt formData string false "Trade Amount"
// @Param PaymentDate formData string false "Payment Date"
// @Param PaymentType formData string false "Payment Type"
// @Success 200 {string} string "1|OK"
// @Failure 400 {object} ErrorResponse
// @Router /payments/provider/ecpay/callback [post]
func (c *BookingController) HandleECPayCallback(ctx *gin.Context) {
	var request ECPayCallbackRequest
	if err := ctx.ShouldBind(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if err := c.BookingService.HandleECPayCallback(ctx.Request.Context(), service.HandleECPayCallbackInput{
		MerchantID:           request.MerchantID,
		MerchantTradeNo:      request.MerchantTradeNo,
		RtnCode:              request.RtnCode,
		RtnMsg:               request.RtnMsg,
		TradeNo:              request.TradeNo,
		TradeAmt:             request.TradeAmt,
		PaymentDate:          request.PaymentDate,
		PaymentType:          request.PaymentType,
		PaymentTypeChargeFee: request.PaymentTypeChargeFee,
		TradeDate:            request.TradeDate,
		SimulatePaid:         request.SimulatePaid,
		CustomField1:         request.CustomField1,
		CustomField2:         request.CustomField2,
		CustomField3:         request.CustomField3,
		CustomField4:         request.CustomField4,
		CheckMacValue:        request.CheckMacValue,
		ReturnStatus:         request.ReturnStatus,
	}); err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("1|OK"))
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

	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	reservation, err := c.BookingService.GetReservation(ctx.Request.Context(), request.ReservationID)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if reservation.UserID != user.ID {
		writeError(ctx, http.StatusForbidden, service.ErrUnauthorized)
		return
	}

	reservation, err = c.BookingService.ExpireReservation(ctx.Request.Context(), service.ExpireReservationInput{
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

	user, ok := CurrentUser(ctx)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, service.ErrUnauthorized)
		return
	}

	reservation, err := c.BookingService.GetReservation(ctx.Request.Context(), request.ReservationID)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if reservation.UserID != user.ID {
		writeError(ctx, http.StatusForbidden, service.ErrUnauthorized)
		return
	}

	reservation, err = c.BookingService.CancelReservation(ctx.Request.Context(), service.CancelReservationInput{
		ReservationID: request.ReservationID,
		CancelledAt:   request.CancelledAt,
	})
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusOK, newReservationResponse(reservation))
}

func hashPaymentRequest(request PayOrderRequest) (string, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func writeError(ctx *gin.Context, statusCode int, err error) {
	appErr := newAppError(statusCode, err)
	appErr.RequestID = requestID(ctx)
	ctx.Set(contextErrorCodeKey, appErr.Code)
	ctx.Set(contextErrorMessageKey, appErr.Message)
	ctx.JSON(appErr.HTTPStatus, ErrorResponse{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: appErr.RequestID,
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
