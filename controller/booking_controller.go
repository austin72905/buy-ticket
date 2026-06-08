package controller

import (
	"net/http"

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
	router.POST("/reservations", c.ReserveTicket)
	router.POST("/orders", c.CreateOrder)
	router.POST("/payments", c.PayOrder)
	router.POST("/reservations/expire", c.ExpireReservation)
	router.POST("/reservations/cancel", c.CancelReservation)
}

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
