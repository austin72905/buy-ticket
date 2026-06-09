package main

import (
	"embed"
	"log"
	"os"

	"buy-ticket/controller"
	db "buy-ticket/db/sqlc"
	docs "buy-ticket/docs"
	"buy-ticket/repository"
	"buy-ticket/service"

	infraapp "github.com/austin72905/go-infra/app"
	infrapostgres "github.com/austin72905/go-infra/postgres"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//go:embed config/default/app.properties
var defaultConfigFiles embed.FS

//go:embed config/local/app.properties
var localConfigFiles embed.FS

//go:embed config/dev/app.properties
var devConfigFiles embed.FS

//go:embed config/prod/app.properties
var prodConfigFiles embed.FS

// @title buy-ticket API
// @version 1.0
// @description 搶票系統 API
// @BasePath /
func main() {
	infraapp.Start(&BuyTicketApp{})
}

type BuyTicketApp struct {
	infraapp.App
}

func (app *BuyTicketApp) Initialize() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	if err := app.Runtime.Property.LoadPropertiesByFS(
		defaultConfigFiles,
		"config/default/app.properties",
		defaultConfigFiles,
	); err != nil {
		log.Fatalf("load default app.properties failed: %v", err)
	}

	envConfigFile := map[string]string{
		"local": "config/local/app.properties",
		"dev":   "config/dev/app.properties",
		"prod":  "config/prod/app.properties",
	}
	envFile, ok := envConfigFile[env]
	if !ok {
		log.Fatalf("unsupported APP_ENV %q", env)
	}

	envConfigFS := map[string]embed.FS{
		"local": localConfigFiles,
		"dev":   devConfigFiles,
		"prod":  prodConfigFiles,
	}

	if err := app.Runtime.Property.LoadPropertiesByFS(
		envConfigFS[env],
		envFile,
		defaultConfigFiles,
	); err != nil {
		log.Fatalf("load %s app.properties failed: %v", env, err)
	}

	eventRepo, sectionRepo, reservationRepo, orderRepo, paymentRepo, dbPool := buildRepositories(app.Runtime)

	bookingService := service.NewBookingService(
		eventRepo,
		sectionRepo,
		reservationRepo,
		orderRepo,
		paymentRepo,
	)
	bookingService.DB = dbPool

	bookingController := controller.NewBookingController(bookingService)

	router := app.Runtime.Web.Router()
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	bookingController.RegisterRoutes(router)

	addr := app.Runtime.Property.RequiredProperty("server.addr")
	app.Runtime.Web.Listen(addr)
	log.Printf("server configured at %s", addr)
}

func buildRepositories(runtime *infraapp.Runtime) (
	repository.EventRepository,
	repository.SectionRepository,
	repository.ReservationRepository,
	repository.OrderRepository,
	repository.PaymentRepository,
	*pgxpool.Pool,
) {
	if runtime.Property.RequiredProperty("app.store") == "postgres" {
		pg := infrapostgres.Register(runtime, "main")
		pg.LoadFromPrefix("postgres")
		pool := pg.Pool()
		queries := db.New(pool)
		return repository.NewPostgresEventRepository(queries),
			repository.NewPostgresSectionRepository(queries),
			repository.NewPostgresReservationRepository(queries),
			repository.NewPostgresOrderRepository(queries),
			repository.NewPostgresPaymentRepository(queries),
			pool
	}

	events, sections := repository.SeedSampleData()
	return repository.NewMemoryEventRepository(events),
		repository.NewMemorySectionRepository(sections),
		repository.NewMemoryReservationRepository(),
		repository.NewMemoryOrderRepository(),
		repository.NewMemoryPaymentRepository(),
		nil
}
