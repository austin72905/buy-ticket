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

// 這些檔案一起打包進去binary

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

type envConfig struct {
	fileName string
	fs       embed.FS
}

type appRepositories struct {
	user           repository.UserRepository
	adminUser      repository.AdminUserRepository
	organizer      repository.OrganizerRepository
	adminAuditLog  repository.AdminAuditLogRepository
	event          repository.EventRepository
	adminEvent     repository.AdminEventRepository
	section        repository.SectionRepository
	reservation    repository.ReservationRepository
	order          repository.OrderRepository
	adminOrder     repository.AdminOrderRepository
	payment        repository.PaymentRepository
	paymentAttempt repository.PaymentAttemptRepository
	outbox         repository.OutboxEventRepository
	idempotency    repository.IdempotencyRepository
	dbPool         *pgxpool.Pool
}

func (app *BuyTicketApp) Initialize() {
	// 目前將 程式分 api  、 scheduler 兩個image 部屬， 用 環境變數區分
	role := parseAppRole()
	if role == appRoleScheduler {
		app.Runtime.Lifecycle.Startup.Serve = nil
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	// 先載入 config/default/app.properties，把共用預設設定放進 runtime.Property.Store
	if err := app.Runtime.Property.LoadPropertiesByFS(
		defaultConfigFiles,
		"config/default/app.properties",
		defaultConfigFiles,
	); err != nil {
		log.Fatalf("load default app.properties failed: %v", err)
	}

	envConfigs := map[string]envConfig{
		"local": {
			fileName: "config/local/app.properties",
			fs:       localConfigFiles,
		},
		"dev": {
			fileName: "config/dev/app.properties",
			fs:       devConfigFiles,
		},
		"prod": {
			fileName: "config/prod/app.properties",
			fs:       prodConfigFiles,
		},
	}

	config, ok := envConfigs[env]
	if !ok {
		log.Fatalf("unsupported APP_ENV %q", env)
	}

	if err := app.Runtime.Property.LoadPropertiesByFS(
		config.fs,
		config.fileName,
		defaultConfigFiles,
	); err != nil {
		log.Fatalf("load %s app.properties failed: %v", env, err)
	}
	applyEnvOverrides(app.Runtime)

	// Build app dependencies after properties and env overrides are loaded.
	repos := buildRepositories(app.Runtime)
	// 同時被api scheduler 依賴所以放這
	bookingService := buildBookingService(app.Runtime, repos)
	sessionStore := buildSessionStore(app.Runtime)
	sessionTTL := sessionTTL(app.Runtime)

	if role == appRoleAll || role == appRoleScheduler {
		// Run once on startup so stale event statuses are corrected before the periodic scheduler.
		if count, err := bookingService.AdvanceEventStatuses(context.Background(), time.Now()); err != nil {
			log.Fatalf("advance event statuses failed: %v", err)
		} else if count > 0 {
			log.Printf("event status scheduler advanced %d events", count)
		}
		if err := bookingService.RebuildStock(context.Background()); err != nil {
			log.Fatalf("rebuild stock failed: %v", err)
		}
		registerBackgroundJobs(app.Runtime, bookingService)
		log.Printf("scheduler configured")
	}

	if role == appRoleAll || role == appRoleAPI {
		registerHTTPServer(app.Runtime, repos.user, repos.adminUser, repos.organizer, repos.adminAuditLog, repos.adminOrder, repos.adminEvent, bookingService, sessionStore, sessionTTL)
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
	organizerRepo repository.OrganizerRepository,
	adminAuditLogRepo repository.AdminAuditLogRepository,
	adminOrderRepo repository.AdminOrderRepository,
	adminEventRepo repository.AdminEventRepository,
	bookingService *service.BookingService,
	sessionStore service.SessionStore,
	sessionTTL time.Duration,
) {
	authService := service.NewAuthService(userRepo)
	adminAuthService := service.NewAdminAuthService(adminUserRepo)
	adminService := service.NewAdminService(adminUserRepo, organizerRepo, adminOrderRepo, adminEventRepo, adminAuditLogRepo)
	authController := controller.NewAuthController(authService, sessionStore, sessionTTL)
	adminAuthController := controller.NewAdminAuthController(adminAuthService, sessionStore, sessionTTL)
	adminController := controller.NewAdminController(adminAuthService, adminService, sessionStore)
	bookingController := controller.NewBookingController(bookingService)
	bookingController.QueueJoinMaxInFlight = queueJoinMaxInFlight(runtime)
	bookingController.QueueJoinRetryAfter = queueJoinRetryAfter(runtime)

	router := runtime.Web.Router()
	router.Use(controller.RecoveryMiddleware())
	router.Use(controller.RequestIDMiddleware())
	router.Use(controller.RequestLoggingMiddleware())
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

func buildBookingService(runtime *infraapp.Runtime, repos *appRepositories) *service.BookingService {
	bookingService := service.NewBookingService(
		repos.event,
		repos.section,
		repos.reservation,
		repos.order,
		repos.payment,
		repos.paymentAttempt,
		repos.outbox,
		repos.idempotency,
	)
	bookingService.DB = repos.dbPool
	bookingService.OrderPaymentTTL = orderPaymentTTL(runtime)
	bookingService.MockPaymentCallbackURL = runtime.Property.Property("payment.mock.callback_url")
	bookingService.QueueStore = buildQueueStore(runtime)
	bookingService.StockStore = buildStockStore(runtime)
	bookingService.MockPaymentSignature = service.MockPaymentSignatureConfig{
		MerchantID: runtime.Property.Property("payment.mock.merchant_id"),
		HashKey:    runtime.Property.Property("payment.mock.hash_key"),
		HashIV:     runtime.Property.Property("payment.mock.hash_iv"),
	}

	paymentBreakerConfig := paymentCircuitBreakerConfig(runtime)
	if mockPaymentRouter := buildMockPaymentProviderRouter(runtime, paymentBreakerConfig); mockPaymentRouter != nil {
		bookingService.MockPaymentRouter = mockPaymentRouter
		bookingService.MockPaymentClient = mockPaymentRouter.PrimaryClient()
	}

	return bookingService
}

// 環境變數覆寫邏輯: 先從 embed 進 binary 的 config/default/app.properties 和 config/{APP_ENV}/app.properties 載入。
// 再用部署環境給的環境變數覆蓋掉 properties 裡的值。
// 主要是讓 Helm/Kubernetes 的 ConfigMap / Secret 能真的生效
func applyEnvOverrides(runtime *infraapp.Runtime) {
	envOverrides := map[string]string{
		"SERVER_ADDR":                            "server.addr",
		"QUEUE_STORE":                            "queue.store",
		"QUEUE_RELEASE_LIMIT":                    "queue.release.limit",
		"QUEUE_JOIN_MAX_IN_FLIGHT":               "queue.join.max_in_flight",
		"QUEUE_JOIN_RETRY_AFTER_SECONDS":         "queue.join.retry_after_seconds",
		"ORDER_EXPIRE_BATCH_SIZE":                "order.expire.batch.size",
		"ORDER_PAYMENT_TTL_MINUTES":              "order.payment.ttl_minutes",
		"SESSION_TTL_HOURS":                      "session.ttl_hours",
		"POSTGRES_DSN":                           "postgres.dsn",
		"POSTGRES_POOL_MAX_IDLE_CONNS":           "postgres.pool.maxIdleConns",
		"POSTGRES_POOL_MAX_OPEN_CONNS":           "postgres.pool.maxOpenConns",
		"POSTGRES_CONN_MAX_IDLE_TIME":            "postgres.connMaxIdleTime",
		"POSTGRES_CONN_MAX_LIFETIME":             "postgres.connMaxLifetime",
		"REDIS_ADDR":                             "redis.addr",
		"REDIS_DB":                               "redis.db",
		"REDIS_PASSWORD":                         "redis.password",
		"PAYMENT_MOCK_MERCHANT_ID":               "payment.mock.merchant_id",
		"PAYMENT_MOCK_HASH_KEY":                  "payment.mock.hash_key",
		"PAYMENT_MOCK_HASH_IV":                   "payment.mock.hash_iv",
		"PAYMENT_MOCK_BASE_URL":                  "payment.mock.base_url",
		"PAYMENT_MOCK_BACKUP_BASE_URL":           "payment.mock.backup_base_url",
		"PAYMENT_MOCK_CALLBACK_URL":              "payment.mock.callback_url",
		"PAYMENT_MOCK_TIMEOUT_SECONDS":           "payment.mock.timeout_seconds",
		"PAYMENT_BREAKER_ENABLED":                "payment.breaker.enabled",
		"PAYMENT_BREAKER_CONSECUTIVE_FAILURES":   "payment.breaker.consecutive_failures",
		"PAYMENT_BREAKER_OPEN_TIMEOUT_SECONDS":   "payment.breaker.open_timeout_seconds",
		"PAYMENT_BREAKER_HALF_OPEN_MAX_REQUESTS": "payment.breaker.half_open_max_requests",
		"PAYMENT_RECONCILE_ENABLED":              "payment.reconcile.enabled",
		"PAYMENT_RECONCILE_BATCH_SIZE":           "payment.reconcile.batch_size",
		"PAYMENT_RECONCILE_DELAY_SECONDS":        "payment.reconcile.delay_seconds",
		"PAYMENT_RECONCILE_RETRY_AFTER_SECONDS":  "payment.reconcile.retry_after_seconds",
		"PAYMENT_RECONCILE_MAX_ATTEMPTS":         "payment.reconcile.max_attempts",
		"OUTBOX_PUBLISH_ENABLED":                 "outbox.publish.enabled",
		"OUTBOX_PUBLISH_BATCH_SIZE":              "outbox.publish.batch_size",
		"OUTBOX_PUBLISH_RETRY_AFTER_SECONDS":     "outbox.publish.retry_after_seconds",
	}

	for envName, propertyKey := range envOverrides {
		value, ok := os.LookupEnv(envName) // 即使值是空字串 也會覆蓋，跟 os.Getenv 不同
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
	// redis_queue_store
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

func buildMockPaymentProviderRouter(runtime *infraapp.Runtime, breakerConfig service.PaymentCircuitBreakerConfig) *service.MockPaymentProviderRouter {
	timeout := mockPaymentTimeout(runtime)
	providers := make([]service.MockPaymentProvider, 0, 2)

	if baseURL := runtime.Property.Property("payment.mock.base_url"); baseURL != "" {
		providers = append(providers, service.MockPaymentProvider{
			Name: "mock_ecpay_primary",
			Client: service.NewCircuitBreakerMockPaymentClient(
				service.NewHTTPMockPaymentClient(baseURL, timeout),
				breakerConfig,
			),
		})
	}

	if baseURL := runtime.Property.Property("payment.mock.backup_base_url"); baseURL != "" {
		providers = append(providers, service.MockPaymentProvider{
			Name: "mock_ecpay_backup",
			Client: service.NewCircuitBreakerMockPaymentClient(
				service.NewHTTPMockPaymentClient(baseURL, timeout),
				breakerConfig,
			),
		})
	}

	if len(providers) == 0 {
		return nil
	}

	return service.NewMockPaymentProviderRouter(providers)
}

func registerBackgroundJobs(runtime *infraapp.Runtime, bookingService *service.BookingService) {
	scheduler := infrascheduler.Register(runtime, "")
	scheduler.SetPanicOnAnyAddError(true)
	// 每秒  把排隊中的使用者從 waiting 推進成 ready，並發給他一個 purchaseToken
	_, err := scheduler.AddFuncJobWithName("*/1 * * * * *", "queue-promote-ready", func(ctx context.Context) {
		now := time.Now()
		if err := bookingService.QueueStore.PromoteReady(ctx, now); err != nil {
			log.Printf("queue scheduler promote failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("register queue scheduler failed: %v", err)
	}
	// 每 5 秒 找出已超過付款期限、但還是 pending payment 的訂單，將它們過期
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
	// 每 5 秒  活動到了開賣時間、結束時間，就更新 event status
	_, err = scheduler.AddFuncJobWithName("*/5 * * * * *", "event-status-advance", func(ctx context.Context) {
		count, err := bookingService.AdvanceEventStatuses(ctx, time.Now())
		if err != nil {
			log.Printf("event status scheduler advance failed: %v", err)
			return
		}
		if count > 0 {
			log.Printf("event status scheduler advanced %d events", count)
		}
	})
	if err != nil {
		log.Fatalf("register event status scheduler failed: %v", err)
	}
	// 每 1 秒  清掉已經過期的 purchaseToken
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
	// 每 10 秒  清掉整個 queue token 已過期的排隊紀錄
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
	// 每分鐘第 0 秒跑一次  校正 Redis stock 和資料庫 section inventory 的差異。
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

	if paymentReconcileEnabled(runtime) {
		_, err = scheduler.AddFuncJobWithName("*/30 * * * * *", "payment-attempt-reconcile", func(ctx context.Context) {
			count, err := bookingService.ReconcilePaymentAttempts(ctx, service.ReconcilePaymentAttemptsInput{
				Now:         time.Now(),
				Delay:       paymentReconcileDelay(runtime),
				RetryAfter:  paymentReconcileRetryAfter(runtime),
				Limit:       paymentReconcileBatchSize(runtime),
				MaxAttempts: paymentReconcileMaxAttempts(runtime),
			})
			if err != nil {
				log.Printf("payment attempt reconcile failed: %v", err)
				return
			}
			if count > 0 {
				log.Printf("payment attempt reconcile completed %d attempts", count)
			}
		})
		if err != nil {
			log.Fatalf("register payment attempt reconcile failed: %v", err)
		}
	}

	if outboxPublishEnabled(runtime) {
		_, err = scheduler.AddFuncJobWithName("*/10 * * * * *", "outbox-publish", func(ctx context.Context) {
			count, err := bookingService.PublishOutboxEvents(ctx, service.PublishOutboxEventsInput{
				Now:        time.Now(),
				Limit:      outboxPublishBatchSize(runtime),
				RetryAfter: outboxPublishRetryAfter(runtime),
			})
			if err != nil {
				log.Printf("outbox publish failed: %v", err)
				return
			}
			if count > 0 {
				log.Printf("outbox published %d events", count)
			}
		})
		if err != nil {
			log.Fatalf("register outbox publish failed: %v", err)
		}
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

func paymentReconcileEnabled(runtime *infraapp.Runtime) bool {
	value := runtime.Property.Property("payment.reconcile.enabled")
	if value == "" {
		return true
	}
	enabled, err := strconv.ParseBool(value)
	return err == nil && enabled
}

func paymentReconcileBatchSize(runtime *infraapp.Runtime) int {
	value := runtime.Property.Property("payment.reconcile.batch_size")
	if value == "" {
		return 100
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 100
	}
	return limit
}

func paymentReconcileDelay(runtime *infraapp.Runtime) time.Duration {
	value := runtime.Property.Property("payment.reconcile.delay_seconds")
	if value == "" {
		return 2 * time.Minute
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 2 * time.Minute
	}
	return time.Duration(seconds) * time.Second
}

func paymentReconcileRetryAfter(runtime *infraapp.Runtime) time.Duration {
	value := runtime.Property.Property("payment.reconcile.retry_after_seconds")
	if value == "" {
		return 30 * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func paymentReconcileMaxAttempts(runtime *infraapp.Runtime) int {
	value := runtime.Property.Property("payment.reconcile.max_attempts")
	if value == "" {
		return 5
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 5
	}
	return limit
}

func outboxPublishEnabled(runtime *infraapp.Runtime) bool {
	value := runtime.Property.Property("outbox.publish.enabled")
	if value == "" {
		return true
	}
	enabled, err := strconv.ParseBool(value)
	return err == nil && enabled
}

func outboxPublishBatchSize(runtime *infraapp.Runtime) int {
	value := runtime.Property.Property("outbox.publish.batch_size")
	if value == "" {
		return 100
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 100
	}
	return limit
}

func outboxPublishRetryAfter(runtime *infraapp.Runtime) time.Duration {
	value := runtime.Property.Property("outbox.publish.retry_after_seconds")
	if value == "" {
		return 30 * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(seconds) * time.Second
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

func paymentCircuitBreakerConfig(runtime *infraapp.Runtime) service.PaymentCircuitBreakerConfig {
	return service.PaymentCircuitBreakerConfig{
		Enabled:             boolProperty(runtime, "payment.breaker.enabled", true),
		ConsecutiveFailures: uint32Property(runtime, "payment.breaker.consecutive_failures", 5),
		OpenTimeout:         secondsProperty(runtime, "payment.breaker.open_timeout_seconds", 30),
		HalfOpenMaxRequests: uint32Property(runtime, "payment.breaker.half_open_max_requests", 1),
	}
}

func boolProperty(runtime *infraapp.Runtime, key string, fallback bool) bool {
	value := runtime.Property.Property(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func uint32Property(runtime *infraapp.Runtime, key string, fallback uint32) uint32 {
	value := runtime.Property.Property(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil || parsed == 0 {
		return fallback
	}

	return uint32(parsed)
}

func secondsProperty(runtime *infraapp.Runtime, key string, fallbackSeconds int) time.Duration {
	value := runtime.Property.Property(key)
	if value == "" {
		return time.Duration(fallbackSeconds) * time.Second
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(fallbackSeconds) * time.Second
	}

	return time.Duration(seconds) * time.Second
}

func buildRepositories(runtime *infraapp.Runtime) *appRepositories {
	pg := infrapostgres.Register(runtime, "main") // 註冊一個 PostgreSQL component，名字叫 "main"
	pg.LoadFromPrefix("postgres")                 // 從 property 裡讀 postgres.* 這組設定
	pool := pg.Pool()
	queries := db.New(pool)
	userRepo := repository.NewPostgresUserRepository(queries)
	adminUserRepo := repository.NewPostgresAdminUserRepository(queries)
	organizerRepo := repository.NewPostgresOrganizerRepository(queries)
	adminAuditLogRepo := repository.NewPostgresAdminAuditLogRepository(queries)
	eventRepo := repository.NewPostgresEventRepository(queries)
	sectionRepo := repository.NewPostgresSectionRepository(queries)
	reservationRepo := repository.NewPostgresReservationRepository(queries)
	orderRepo := repository.NewPostgresOrderRepository(queries)
	paymentRepo := repository.NewPostgresPaymentRepository(queries)
	paymentAttemptRepo := repository.NewPostgresPaymentAttemptRepository(queries)
	outboxRepo := repository.NewPostgresOutboxEventRepository(queries)
	idempotencyRepo := repository.NewPostgresIdempotencyRepository(queries)
	return &appRepositories{
		user:           userRepo,
		adminUser:      adminUserRepo,
		organizer:      organizerRepo,
		adminAuditLog:  adminAuditLogRepo,
		event:          eventRepo,
		adminEvent:     eventRepo,
		section:        sectionRepo,
		reservation:    reservationRepo,
		order:          orderRepo,
		adminOrder:     orderRepo,
		payment:        paymentRepo,
		paymentAttempt: paymentAttemptRepo,
		outbox:         outboxRepo,
		idempotency:    idempotencyRepo,
		dbPool:         pool,
	}
}
