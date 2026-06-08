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
	router.GET("/events/:eventId", c.GetEvent)
	router.GET("/events/:eventId/sections", c.GetSections)
	router.GET("/events/:eventId/availability", c.GetAvailability)
	router.GET("/orders/:orderId", c.GetOrder)
	router.POST("/reservations", c.ReserveTicket)
	router.POST("/orders", c.CreateOrder)
	router.POST("/payments", c.PayOrder)
	router.POST("/reservations/expire", c.ExpireReservation)
	router.POST("/reservations/cancel", c.CancelReservation)
}

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

func parseInt64Param(ctx *gin.Context, key string) (int64, bool) {
	value, err := strconv.ParseInt(ctx.Param(key), 10, 64)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err)
		return 0, false
	}

	return value, true
}
