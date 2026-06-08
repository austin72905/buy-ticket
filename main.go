package main

import (
	"context"
	"log"
	"os"

	"buy-ticket/controller"
	db "buy-ticket/db/sqlc"
	"buy-ticket/repository"
	"buy-ticket/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	eventRepo, sectionRepo, reservationRepo, orderRepo, paymentRepo, dbPool, cleanup := buildRepositories()
	defer cleanup()

	bookingService := service.NewBookingService(
		eventRepo,
		sectionRepo,
		reservationRepo,
		orderRepo,
		paymentRepo,
	)
	bookingService.DB = dbPool

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

// 需要設置APP_STORE, DATABASE_URL 兩個環境變數才會跑真的repo , 不然會用fake
func buildRepositories() (
	repository.EventRepository,
	repository.SectionRepository,
	repository.ReservationRepository,
	repository.OrderRepository,
	repository.PaymentRepository,
	*pgxpool.Pool,
	func(),
) {
	if os.Getenv("APP_STORE") == "postgres" {
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			log.Fatal("DATABASE_URL is required when APP_STORE=postgres")
		}

		pool, err := pgxpool.New(context.Background(), dsn)
		if err != nil {
			log.Fatalf("connect postgres failed: %v", err)
		}

		queries := db.New(pool)
		return repository.NewPostgresEventRepository(queries),
			repository.NewPostgresSectionRepository(queries),
			repository.NewPostgresReservationRepository(queries),
			repository.NewPostgresOrderRepository(queries),
			repository.NewPostgresPaymentRepository(queries),
			pool,
			func() { pool.Close() }
	}

	events, sections := repository.SeedSampleData()
	return repository.NewMemoryEventRepository(events),
		repository.NewMemorySectionRepository(sections),
		repository.NewMemoryReservationRepository(),
		repository.NewMemoryOrderRepository(),
		repository.NewMemoryPaymentRepository(),
		nil,
		func() {}
}
