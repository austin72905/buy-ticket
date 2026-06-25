package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"buy-ticket/controller"
	db "buy-ticket/db/sqlc"
	docs "buy-ticket/docs"
	"buy-ticket/repository"
	"buy-ticket/service"

	infraapp "github.com/austin72905/go-infra/app"
	infrapostgres "github.com/austin72905/go-infra/postgres"
	infraredis "github.com/austin72905/go-infra/redis"
	infrascheduler "github.com/austin72905/go-infra/scheduler"
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

func (app *BuyTicketApp) Start() {
	ctx := context.Background()
	if err := app.Runtime.Probe.Check(ctx); err != nil {
		panic(err)
	}
	app.Runtime.Lifecycle.Startup.RunAll(ctx)
	app.Runtime.Lifecycle.Started = true
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	<-stop
	app.Runtime.Lifecycle.Shutdown.RunAll(ctx)
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

	userRepo, eventRepo, sectionRepo, reservationRepo, orderRepo, paymentRepo, dbPool := buildRepositories(app.Runtime)

	authService := service.NewAuthService(userRepo)
	authController := controller.NewAuthController(authService)

	bookingService := service.NewBookingService(
		eventRepo,
		sectionRepo,
		reservationRepo,
		orderRepo,
		paymentRepo,
	)
	bookingService.DB = dbPool
	bookingService.OrderPaymentTTL = orderPaymentTTL(app.Runtime)
	if dbPool != nil {
		bookingService.PaymentAttemptRepo = repository.NewPostgresPaymentAttemptRepository(db.New(dbPool))
		bookingService.IdempotencyRepo = repository.NewPostgresIdempotencyRepository(db.New(dbPool))
	}
	bookingService.MockPaymentCallbackURL = app.Runtime.Property.Property("payment.mock.callback_url")
	if mockPaymentBaseURL := app.Runtime.Property.Property("payment.mock.base_url"); mockPaymentBaseURL != "" {
		bookingService.MockPaymentClient = service.NewHTTPMockPaymentClient(
			mockPaymentBaseURL,
			mockPaymentTimeout(app.Runtime),
		)
	}
	bookingService.QueueStore = buildQueueStore(app.Runtime)
	bookingService.StockStore = buildStockStore(app.Runtime)
	bookingService.MockPaymentSignature = service.MockPaymentSignatureConfig{
		MerchantID: app.Runtime.Property.Property("payment.mock.merchant_id"),
		HashKey:    app.Runtime.Property.Property("payment.mock.hash_key"),
		HashIV:     app.Runtime.Property.Property("payment.mock.hash_iv"),
	}
	if err := bookingService.RebuildStock(context.Background()); err != nil {
		log.Fatalf("rebuild stock failed: %v", err)
	}
	registerBackgroundJobs(app.Runtime, bookingService)

	bookingController := controller.NewBookingController(bookingService)

	router := app.Runtime.Web.Router()
	router.Use(controller.AttachCurrentUser(authService))
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	authController.RegisterRoutes(router)
	bookingController.RegisterRoutes(router)

	addr := app.Runtime.Property.RequiredProperty("server.addr")
	app.Runtime.Web.Listen(addr)
	log.Printf("server configured at %s", addr)
}

func buildQueueStore(runtime *infraapp.Runtime) service.QueueStore {
	releaseLimit := queueReleaseLimit(runtime)
	if runtime.Property.Property("queue.store") == "redis" {
		redisComponent := infraredis.Register(runtime, "queue")
		redisComponent.LoadFromPrefix("redis")
		return service.NewRedisQueueStore(redisComponent.Client(), releaseLimit)
	}

	return service.NewMemoryQueueStore(releaseLimit)
}

func buildStockStore(runtime *infraapp.Runtime) service.StockStore {
	if runtime.Property.Property("redis.addr") == "" {
		return nil
	}

	redisComponent := infraredis.Register(runtime, "stock")
	redisComponent.LoadFromPrefix("redis")
	return service.NewRedisStockStore(redisComponent.Client())
}

func registerBackgroundJobs(runtime *infraapp.Runtime, bookingService *service.BookingService) {
	scheduler := infrascheduler.Register(runtime, "")
	scheduler.SetPanicOnAnyAddError(true)

	_, err := scheduler.AddFuncJobWithName("*/1 * * * * *", "queue-promote-ready", func(ctx context.Context) {
		now := time.Now()
		if err := bookingService.QueueStore.PromoteReady(ctx, now); err != nil {
			log.Printf("queue scheduler promote failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("register queue scheduler failed: %v", err)
	}

	_, err = scheduler.AddFuncJobWithName("*/5 * * * * *", "order-expire-sweep", func(ctx context.Context) {
		count, err := bookingService.SweepExpiredOrders(ctx, service.SweepExpiredOrdersInput{
			Now:   time.Now(),
			Limit: orderExpireBatchSize(runtime),
		})
		if err != nil {
			log.Printf("order expire scheduler sweep failed: %v", err)
			return
		}
		if count > 0 {
			log.Printf("order expire scheduler expired %d orders", count)
		}
	})
	if err != nil {
		log.Fatalf("register order expire scheduler failed: %v", err)
	}

	_, err = scheduler.AddFuncJobWithName("*/1 * * * * *", "purchase-token-cleanup", func(ctx context.Context) {
		count, err := bookingService.CleanupExpiredPurchaseTokens(ctx, time.Now())
		if err != nil {
			log.Printf("purchase token cleanup failed: %v", err)
			return
		}
		if count > 0 {
			log.Printf("purchase token cleanup expired %d tokens", count)
		}
	})
	if err != nil {
		log.Fatalf("register purchase token cleanup failed: %v", err)
	}

	_, err = scheduler.AddFuncJobWithName("*/10 * * * * *", "queue-timeout-cleanup", func(ctx context.Context) {
		count, err := bookingService.CleanupExpiredQueues(ctx, time.Now())
		if err != nil {
			log.Printf("queue timeout cleanup failed: %v", err)
			return
		}
		if count > 0 {
			log.Printf("queue timeout cleanup expired %d queues", count)
		}
	})
	if err != nil {
		log.Fatalf("register queue timeout cleanup failed: %v", err)
	}

	_, err = scheduler.AddFuncJobWithName("0 * * * * *", "stock-reconcile", func(ctx context.Context) {
		result, err := bookingService.ReconcileStock(ctx)
		if err != nil {
			log.Printf("stock reconcile failed: %v", err)
			return
		}
		if result.Fixed > 0 {
			log.Printf("stock reconcile fixed %d/%d sections", result.Fixed, result.Checked)
		}
	})
	if err != nil {
		log.Fatalf("register stock reconcile failed: %v", err)
	}
}

func queueReleaseLimit(runtime *infraapp.Runtime) int {
	value := runtime.Property.Property("queue.release.limit")
	if value == "" {
		return 1
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 1
	}

	return limit
}

func orderExpireBatchSize(runtime *infraapp.Runtime) int {
	value := runtime.Property.Property("order.expire.batch.size")
	if value == "" {
		return 100
	}

	size, err := strconv.Atoi(value)
	if err != nil || size <= 0 {
		return 100
	}

	return size
}

func orderPaymentTTL(runtime *infraapp.Runtime) time.Duration {
	value := runtime.Property.Property("order.payment.ttl_minutes")
	if value == "" {
		return 10 * time.Minute
	}

	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		return 10 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}

func mockPaymentTimeout(runtime *infraapp.Runtime) time.Duration {
	value := runtime.Property.Property("payment.mock.timeout_seconds")
	if value == "" {
		return 3 * time.Second
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 3 * time.Second
	}

	return time.Duration(seconds) * time.Second
}

func buildRepositories(runtime *infraapp.Runtime) (
	repository.UserRepository,
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
		return repository.NewPostgresUserRepository(queries),
			repository.NewPostgresEventRepository(queries),
			repository.NewPostgresSectionRepository(queries),
			repository.NewPostgresReservationRepository(queries),
			repository.NewPostgresOrderRepository(queries),
			repository.NewPostgresPaymentRepository(queries),
			pool
	}

	users, events, sections := repository.SeedSampleData()
	return repository.NewMemoryUserRepository(users),
		repository.NewMemoryEventRepository(events),
		repository.NewMemorySectionRepository(sections),
		repository.NewMemoryReservationRepository(),
		repository.NewMemoryOrderRepository(),
		repository.NewMemoryPaymentRepository(),
		nil
}
