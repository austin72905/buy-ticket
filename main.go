package main

import (
	"log"

	"buy-ticket/controller"
	"buy-ticket/repository"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
)

func main() {
	events, sections := repository.SeedSampleData()

	eventRepo := repository.NewMemoryEventRepository(events)
	sectionRepo := repository.NewMemorySectionRepository(sections)
	reservationRepo := repository.NewMemoryReservationRepository()
	orderRepo := repository.NewMemoryOrderRepository()
	paymentRepo := repository.NewMemoryPaymentRepository()

	bookingService := service.NewBookingService(
		eventRepo,
		sectionRepo,
		reservationRepo,
		orderRepo,
		paymentRepo,
	)

	bookingController := controller.NewBookingController(bookingService)

	router := gin.Default()
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})
	bookingController.RegisterRoutes(router)

	log.Println("server started at :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
