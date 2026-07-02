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

type appRole string

const (
	appRoleAll       appRole = "all"
	appRoleAPI       appRole = "api"
	appRoleScheduler appRole = "scheduler"
)

func (app *BuyTicketApp) Initialize() {
	role := parseAppRole()
	if role == appRoleScheduler {
		app.Runtime.Lifecycle.Startup.Serve = nil
	}

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
	applyEnvOverrides(app.Runtime)

	userRepo, adminUserRepo, adminAuditLogRepo, eventRepo, adminEventRepo, sectionRepo, reservationRepo, orderRepo, adminOrderRepo, paymentRepo, dbPool := buildRepositories(app.Runtime)

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
	sessionStore := buildSessionStore(app.Runtime)
	sessionTTL := sessionTTL(app.Runtime)

	if role == appRoleAll || role == appRoleScheduler {
		if err := bookingService.RebuildStock(context.Background()); err != nil {
			log.Fatalf("rebuild stock failed: %v", err)
		}
		registerBackgroundJobs(app.Runtime, bookingService)
		log.Printf("scheduler configured")
	}

	if role == appRoleAll || role == appRoleAPI {
		registerHTTPServer(app.Runtime, userRepo, adminUserRepo, adminAuditLogRepo, adminOrderRepo, adminEventRepo, bookingService, sessionStore, sessionTTL)
	}

	log.Printf("app role configured: %s", role)
}

func parseAppRole() appRole {
	value := os.Getenv("APP_ROLE")
	if value == "" {
		return appRoleAll
	}

	role := appRole(value)
	switch role {
	case appRoleAll, appRoleAPI, appRoleScheduler:
		return role
	default:
		log.Fatalf("unsupported APP_ROLE %q", value)
		return ""
	}
}

func registerHTTPServer(
	runtime *infraapp.Runtime,
	userRepo repository.UserRepository,
	adminUserRepo repository.AdminUserRepository,
	adminAuditLogRepo repository.AdminAuditLogRepository,
	adminOrderRepo repository.AdminOrderRepository,
	adminEventRepo repository.AdminEventRepository,
	bookingService *service.BookingService,
	sessionStore service.SessionStore,
	sessionTTL time.Duration,
) {
	authService := service.NewAuthService(userRepo)
	adminAuthService := service.NewAdminAuthService(adminUserRepo)
	adminService := service.NewAdminService(adminOrderRepo, adminEventRepo, adminAuditLogRepo)
	authController := controller.NewAuthController(authService, sessionStore, sessionTTL)
	adminAuthController := controller.NewAdminAuthController(adminAuthService, sessionStore, sessionTTL)
	adminController := controller.NewAdminController(adminAuthService, adminService, sessionStore)
	bookingController := controller.NewBookingController(bookingService)
	bookingController.QueueJoinMaxInFlight = queueJoinMaxInFlight(runtime)
	bookingController.QueueJoinRetryAfter = queueJoinRetryAfter(runtime)

	router := runtime.Web.Router()
	router.Use(controller.AttachCurrentUser(authService, sessionStore))
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	authController.RegisterRoutes(router)
	adminAuthController.RegisterRoutes(router)
	adminController.RegisterRoutes(router)
	bookingController.RegisterRoutes(router)

	addr := runtime.Property.RequiredProperty("server.addr")
	runtime.Web.Listen(addr)
	log.Printf("server configured at %s", addr)
}

func applyEnvOverrides(runtime *infraapp.Runtime) {
	envOverrides := map[string]string{
		"SERVER_ADDR":                    "server.addr",
		"APP_STORE":                      "app.store",
		"QUEUE_STORE":                    "queue.store",
		"QUEUE_RELEASE_LIMIT":            "queue.release.limit",
		"QUEUE_JOIN_MAX_IN_FLIGHT":       "queue.join.max_in_flight",
		"QUEUE_JOIN_RETRY_AFTER_SECONDS": "queue.join.retry_after_seconds",
		"ORDER_EXPIRE_BATCH_SIZE":        "order.expire.batch.size",
		"ORDER_PAYMENT_TTL_MINUTES":      "order.payment.ttl_minutes",
		"SESSION_TTL_HOURS":              "session.ttl_hours",
		"POSTGRES_DSN":                   "postgres.dsn",
		"POSTGRES_POOL_MAX_IDLE_CONNS":   "postgres.pool.maxIdleConns",
		"POSTGRES_POOL_MAX_OPEN_CONNS":   "postgres.pool.maxOpenConns",
		"POSTGRES_CONN_MAX_IDLE_TIME":    "postgres.connMaxIdleTime",
		"POSTGRES_CONN_MAX_LIFETIME":     "postgres.connMaxLifetime",
		"REDIS_ADDR":                     "redis.addr",
		"REDIS_DB":                       "redis.db",
		"REDIS_PASSWORD":                 "redis.password",
		"PAYMENT_MOCK_MERCHANT_ID":       "payment.mock.merchant_id",
		"PAYMENT_MOCK_HASH_KEY":          "payment.mock.hash_key",
		"PAYMENT_MOCK_HASH_IV":           "payment.mock.hash_iv",
		"PAYMENT_MOCK_BASE_URL":          "payment.mock.base_url",
		"PAYMENT_MOCK_CALLBACK_URL":      "payment.mock.callback_url",
		"PAYMENT_MOCK_TIMEOUT_SECONDS":   "payment.mock.timeout_seconds",
	}

	for envName, propertyKey := range envOverrides {
		value, ok := os.LookupEnv(envName)
		if ok {
			runtime.Property.Store.Set(propertyKey, value)
		}
	}
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

func buildSessionStore(runtime *infraapp.Runtime) service.SessionStore {
	if runtime.Property.Property("redis.addr") == "" {
		return service.NewMemorySessionStore()
	}

	redisComponent := infraredis.Register(runtime, "session")
	redisComponent.LoadFromPrefix("redis")
	return service.NewRedisSessionStore(redisComponent.Client())
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

func queueJoinMaxInFlight(runtime *infraapp.Runtime) int {
	value := runtime.Property.Property("queue.join.max_in_flight")
	if value == "" {
		return 0
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0
	}

	return limit
}

func queueJoinRetryAfter(runtime *infraapp.Runtime) time.Duration {
	value := runtime.Property.Property("queue.join.retry_after_seconds")
	if value == "" {
		return time.Second
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Second
	}

	return time.Duration(seconds) * time.Second
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

func sessionTTL(runtime *infraapp.Runtime) time.Duration {
	value := runtime.Property.Property("session.ttl_hours")
	if value == "" {
		return 7 * 24 * time.Hour
	}

	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		return 7 * 24 * time.Hour
	}

	return time.Duration(hours) * time.Hour
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
	repository.AdminUserRepository,
	repository.AdminAuditLogRepository,
	repository.EventRepository,
	repository.AdminEventRepository,
	repository.SectionRepository,
	repository.ReservationRepository,
	repository.OrderRepository,
	repository.AdminOrderRepository,
	repository.PaymentRepository,
	*pgxpool.Pool,
) {
	if runtime.Property.RequiredProperty("app.store") == "postgres" {
		pg := infrapostgres.Register(runtime, "main")
		pg.LoadFromPrefix("postgres")
		pool := pg.Pool()
		queries := db.New(pool)
		eventRepo := repository.NewPostgresEventRepository(queries)
		orderRepo := repository.NewPostgresOrderRepository(queries)
		return repository.NewPostgresUserRepository(queries),
			repository.NewPostgresAdminUserRepository(queries),
			repository.NewPostgresAdminAuditLogRepository(queries),
			eventRepo,
			eventRepo,
			repository.NewPostgresSectionRepository(queries),
			repository.NewPostgresReservationRepository(queries),
			orderRepo,
			orderRepo,
			repository.NewPostgresPaymentRepository(queries),
			pool
	}

	users, adminUsers, events, sections := repository.SeedSampleData()
	eventRepo := repository.NewMemoryEventRepository(events)
	orderRepo := repository.NewMemoryOrderRepository()
	return repository.NewMemoryUserRepository(users),
		repository.NewMemoryAdminUserRepository(adminUsers),
		repository.NewMemoryAdminAuditLogRepository(),
		eventRepo,
		eventRepo,
		repository.NewMemorySectionRepository(sections),
		repository.NewMemoryReservationRepository(),
		orderRepo,
		orderRepo,
		repository.NewMemoryPaymentRepository(),
		nil
}
