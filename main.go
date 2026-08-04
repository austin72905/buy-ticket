package main

import (
	"context"
	"log/slog"
	"net/http/pprof"
	"os"
	"time"

	"buy-ticket/controller"
	db "buy-ticket/db/sqlc"
	docs "buy-ticket/docs"
	"buy-ticket/observability"
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

// @title buy-ticket API
// @version 1.0
// @description 搶票系統 API
// @BasePath /
func main() {
	configureLogging()
	infraapp.Start(&BuyTicketApp{})
}

func configureLogging() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}

func fatalLog(message string, attrs ...any) {
	slog.Error(message, attrs...)
	os.Exit(1)
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
	role := parseAppRole()
	if role == appRoleScheduler {
		app.Runtime.Lifecycle.Startup.Serve = nil
	}

	cfg := loadConfig()
	slog.Info("config loaded", configSummary(cfg)...)

	repos := buildRepositories(app.Runtime, cfg)
	bookingService := buildBookingService(app.Runtime, cfg, repos)
	sessionStore := buildSessionStore(app.Runtime, cfg)
	sessionTTL := sessionTTL(cfg)

	if role == appRoleAll || role == appRoleScheduler {
		if count, err := bookingService.AdvanceEventStatuses(context.Background(), time.Now()); err != nil {
			fatalLog("advance event statuses failed", "error", err)
		} else if count > 0 {
			slog.Info("event status advanced on startup", "count", count)
		}
		if err := bookingService.RebuildStock(context.Background()); err != nil {
			fatalLog("rebuild stock failed", "error", err)
		}
		registerBackgroundJobs(app.Runtime, cfg, bookingService)
		slog.Info("scheduler configured")
	}

	if role == appRoleAll || role == appRoleAPI {
		registerHTTPServer(app.Runtime, cfg, repos.user, repos.adminUser, repos.organizer, repos.adminAuditLog, repos.adminOrder, repos.adminEvent, bookingService, sessionStore, sessionTTL)
	}

	slog.Info("app role configured", "role", role)
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
		fatalLog("unsupported APP_ROLE", "role", value)
		return ""
	}
}

func registerHTTPServer(
	runtime *infraapp.Runtime,
	cfg Config,
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
	bookingController.QueueJoinMaxInFlight = queueJoinMaxInFlight(cfg)
	bookingController.QueueJoinRetryAfter = queueJoinRetryAfter(cfg)

	router := runtime.Web.Router()
	router.Use(controller.RecoveryMiddleware())
	router.Use(controller.RequestIDMiddleware())
	router.Use(controller.RequestLoggingMiddleware())
	router.Use(controller.AttachCurrentUser(authService, sessionStore))
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})
	if cfg.App.PprofEnabled {
		registerPprofRoutes(router)
		slog.Info("pprof routes enabled")
	}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	authController.RegisterRoutes(router)
	adminAuthController.RegisterRoutes(router)
	adminController.RegisterRoutes(router)
	bookingController.RegisterRoutes(router)

	addr := cfg.App.ServerAddr
	runtime.Web.Listen(addr)
	slog.Info("server configured", "addr", addr)
}

func registerPprofRoutes(router gin.IRouter) {
	router.GET("/debug/pprof/", gin.WrapF(pprof.Index))
	router.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	router.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	router.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	router.POST("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	router.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
	router.GET("/debug/pprof/:profile", func(ctx *gin.Context) {
		pprof.Handler(ctx.Param("profile")).ServeHTTP(ctx.Writer, ctx.Request)
	})
}

func buildBookingService(runtime *infraapp.Runtime, cfg Config, repos *appRepositories) *service.BookingService {
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
	bookingService.OrderPaymentTTL = orderPaymentTTL(cfg)
	bookingService.MockPaymentCallbackURL = cfg.Payment.Mock.CallbackURL
	bookingService.QueueStore = buildQueueStore(runtime, cfg)
	bookingService.StockStore = buildStockStore(runtime, cfg)
	bookingService.MockPaymentSignature = service.MockPaymentSignatureConfig{
		MerchantID: cfg.Payment.Mock.MerchantID,
		HashKey:    cfg.Payment.Mock.HashKey,
		HashIV:     cfg.Payment.Mock.HashIV,
	}

	paymentBreakerConfig := paymentCircuitBreakerConfig(cfg)
	if mockPaymentRouter := buildMockPaymentProviderRouter(cfg, paymentBreakerConfig); mockPaymentRouter != nil {
		bookingService.MockPaymentRouter = mockPaymentRouter
		bookingService.MockPaymentClient = mockPaymentRouter.PrimaryClient()
	}

	return bookingService
}

func buildQueueStore(runtime *infraapp.Runtime, cfg Config) service.QueueStore {
	releaseLimit := queueReleaseLimit(cfg)
	if cfg.Queue.Store == "redis" {
		redisComponent := buildRedisComponent(runtime, "queue", cfg)
		return service.NewRedisQueueStore(redisComponent.Client(), releaseLimit)
	}

	return service.NewMemoryQueueStore(releaseLimit)
}

func buildStockStore(runtime *infraapp.Runtime, cfg Config) service.StockStore {
	if cfg.Redis.Addr == "" {
		return nil
	}

	redisComponent := buildRedisComponent(runtime, "stock", cfg)
	return service.NewRedisStockStore(redisComponent.Client())
}

func buildSessionStore(runtime *infraapp.Runtime, cfg Config) service.SessionStore {
	if cfg.Redis.Addr == "" {
		return service.NewMemorySessionStore()
	}

	redisComponent := buildRedisComponent(runtime, "session", cfg)
	return service.NewRedisSessionStore(redisComponent.Client())
}

func buildRedisComponent(runtime *infraapp.Runtime, name string, cfg Config) *infraredis.Component {
	redisComponent := infraredis.Register(runtime, name)
	redisComponent.SetAddr(cfg.Redis.Addr)
	redisComponent.SetDB(cfg.Redis.DB)
	redisComponent.SetPassword(cfg.Redis.Password)
	return redisComponent
}

func buildMockPaymentProviderRouter(cfg Config, breakerConfig service.PaymentCircuitBreakerConfig) *service.MockPaymentProviderRouter {
	timeout := mockPaymentTimeout(cfg)
	providers := make([]service.MockPaymentProvider, 0, 2)

	if baseURL := cfg.Payment.Mock.BaseURL; baseURL != "" {
		providers = append(providers, service.MockPaymentProvider{
			Name: "mock_ecpay_primary",
			Client: service.NewCircuitBreakerMockPaymentClient(
				service.NewHTTPMockPaymentClient(baseURL, timeout),
				breakerConfig,
			),
		})
	}

	if baseURL := cfg.Payment.Mock.BackupBaseURL; baseURL != "" {
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

func registerBackgroundJobs(runtime *infraapp.Runtime, cfg Config, bookingService *service.BookingService) {
	scheduler := infrascheduler.Register(runtime, "")
	scheduler.SetPanicOnAnyAddError(true)
	orderExpireLimit := orderExpireBatchSize(cfg)
	paymentReconcileJobEnabled := paymentReconcileEnabled(cfg)
	paymentReconcileJobDelay := paymentReconcileDelay(cfg)
	paymentReconcileJobRetryAfter := paymentReconcileRetryAfter(cfg)
	paymentReconcileJobLimit := paymentReconcileBatchSize(cfg)
	paymentReconcileJobMaxAttempts := paymentReconcileMaxAttempts(cfg)
	outboxPublishJobEnabled := outboxPublishEnabled(cfg)
	outboxPublishJobLimit := outboxPublishBatchSize(cfg)
	outboxPublishJobRetryAfter := outboxPublishRetryAfter(cfg)

	_, err := scheduler.AddFuncJobWithName("*/1 * * * * *", "queue-promote-ready", func(ctx context.Context) {
		now := time.Now()
		if err := bookingService.QueueStore.PromoteReady(ctx, now); err != nil {
			observability.Error(ctx, "scheduler job failed", err, "job_name", "queue-promote-ready")
		}
	})
	if err != nil {
		fatalLog("register scheduler job failed", "job_name", "queue-promote-ready", "error", err)
	}

	_, err = scheduler.AddFuncJobWithName("*/5 * * * * *", "order-expire-sweep", func(ctx context.Context) {
		count, err := bookingService.SweepExpiredOrders(ctx, service.SweepExpiredOrdersInput{
			Now:   time.Now(),
			Limit: orderExpireLimit,
		})
		if err != nil {
			observability.Error(ctx, "scheduler job failed", err, "job_name", "order-expire-sweep")
			return
		}
		if count > 0 {
			observability.Info(ctx, "scheduler job completed", "job_name", "order-expire-sweep", "expired_count", count)
		}
	})
	if err != nil {
		fatalLog("register scheduler job failed", "job_name", "order-expire-sweep", "error", err)
	}

	_, err = scheduler.AddFuncJobWithName("*/5 * * * * *", "event-status-advance", func(ctx context.Context) {
		count, err := bookingService.AdvanceEventStatuses(ctx, time.Now())
		if err != nil {
			observability.Error(ctx, "scheduler job failed", err, "job_name", "event-status-advance")
			return
		}
		if count > 0 {
			observability.Info(ctx, "scheduler job completed", "job_name", "event-status-advance", "advanced_count", count)
		}
	})
	if err != nil {
		fatalLog("register scheduler job failed", "job_name", "event-status-advance", "error", err)
	}

	_, err = scheduler.AddFuncJobWithName("*/1 * * * * *", "purchase-token-cleanup", func(ctx context.Context) {
		count, err := bookingService.CleanupExpiredPurchaseTokens(ctx, time.Now())
		if err != nil {
			observability.Error(ctx, "scheduler job failed", err, "job_name", "purchase-token-cleanup")
			return
		}
		if count > 0 {
			observability.Info(ctx, "scheduler job completed", "job_name", "purchase-token-cleanup", "expired_count", count)
		}
	})
	if err != nil {
		fatalLog("register scheduler job failed", "job_name", "purchase-token-cleanup", "error", err)
	}

	_, err = scheduler.AddFuncJobWithName("*/10 * * * * *", "queue-timeout-cleanup", func(ctx context.Context) {
		count, err := bookingService.CleanupExpiredQueues(ctx, time.Now())
		if err != nil {
			observability.Error(ctx, "scheduler job failed", err, "job_name", "queue-timeout-cleanup")
			return
		}
		if count > 0 {
			observability.Info(ctx, "scheduler job completed", "job_name", "queue-timeout-cleanup", "expired_count", count)
		}
	})
	if err != nil {
		fatalLog("register scheduler job failed", "job_name", "queue-timeout-cleanup", "error", err)
	}

	_, err = scheduler.AddFuncJobWithName("0 * * * * *", "stock-reconcile", func(ctx context.Context) {
		result, err := bookingService.ReconcileStock(ctx)
		if err != nil {
			observability.Error(ctx, "scheduler job failed", err, "job_name", "stock-reconcile")
			return
		}
		if result.Fixed > 0 {
			observability.Info(ctx, "scheduler job completed", "job_name", "stock-reconcile", "fixed_count", result.Fixed, "checked_count", result.Checked)
		}
	})
	if err != nil {
		fatalLog("register scheduler job failed", "job_name", "stock-reconcile", "error", err)
	}

	if paymentReconcileJobEnabled {
		_, err = scheduler.AddFuncJobWithName("*/30 * * * * *", "payment-attempt-reconcile", func(ctx context.Context) {
			count, err := bookingService.ReconcilePaymentAttempts(ctx, service.ReconcilePaymentAttemptsInput{
				Now:         time.Now(),
				Delay:       paymentReconcileJobDelay,
				RetryAfter:  paymentReconcileJobRetryAfter,
				Limit:       paymentReconcileJobLimit,
				MaxAttempts: paymentReconcileJobMaxAttempts,
			})
			if err != nil {
				observability.Error(ctx, "scheduler job failed", err, "job_name", "payment-attempt-reconcile")
				return
			}
			if count > 0 {
				observability.Info(ctx, "scheduler job completed", "job_name", "payment-attempt-reconcile", "completed_count", count)
			}
		})
		if err != nil {
			fatalLog("register scheduler job failed", "job_name", "payment-attempt-reconcile", "error", err)
		}
	}

	if outboxPublishJobEnabled {
		_, err = scheduler.AddFuncJobWithName("*/10 * * * * *", "outbox-publish", func(ctx context.Context) {
			count, err := bookingService.PublishOutboxEvents(ctx, service.PublishOutboxEventsInput{
				Now:        time.Now(),
				Limit:      outboxPublishJobLimit,
				RetryAfter: outboxPublishJobRetryAfter,
			})
			if err != nil {
				observability.Error(ctx, "scheduler job failed", err, "job_name", "outbox-publish")
				return
			}
			if count > 0 {
				observability.Info(ctx, "scheduler job completed", "job_name", "outbox-publish", "published_count", count)
			}
		})
		if err != nil {
			fatalLog("register scheduler job failed", "job_name", "outbox-publish", "error", err)
		}
	}
}

func queueReleaseLimit(cfg Config) int {
	return positiveInt(cfg.Queue.ReleaseLimit, 1)
}

func paymentReconcileEnabled(cfg Config) bool {
	return cfg.Payment.Reconcile.Enabled
}

func paymentReconcileBatchSize(cfg Config) int {
	return positiveInt(cfg.Payment.Reconcile.BatchSize, 100)
}

func paymentReconcileDelay(cfg Config) time.Duration {
	return durationSeconds(cfg.Payment.Reconcile.DelaySeconds, 120)
}

func paymentReconcileRetryAfter(cfg Config) time.Duration {
	return durationSeconds(cfg.Payment.Reconcile.RetryAfterSeconds, 30)
}

func paymentReconcileMaxAttempts(cfg Config) int {
	return positiveInt(cfg.Payment.Reconcile.MaxAttempts, 5)
}

func outboxPublishEnabled(cfg Config) bool {
	return cfg.Outbox.Publish.Enabled
}

func outboxPublishBatchSize(cfg Config) int {
	return positiveInt(cfg.Outbox.Publish.BatchSize, 100)
}

func outboxPublishRetryAfter(cfg Config) time.Duration {
	return durationSeconds(cfg.Outbox.Publish.RetryAfterSeconds, 30)
}

func queueJoinMaxInFlight(cfg Config) int {
	return nonNegativeInt(cfg.Queue.JoinMaxInFlight, 0)
}

func queueJoinRetryAfter(cfg Config) time.Duration {
	return durationSeconds(cfg.Queue.JoinRetryAfterSeconds, 1)
}

func orderExpireBatchSize(cfg Config) int {
	return positiveInt(cfg.Order.ExpireBatchSize, 100)
}

func orderPaymentTTL(cfg Config) time.Duration {
	return durationMinutes(cfg.Order.PaymentTTLMinutes, 10)
}

func sessionTTL(cfg Config) time.Duration {
	return durationHours(cfg.Session.TTLHours, 168)
}

func mockPaymentTimeout(cfg Config) time.Duration {
	return durationSeconds(cfg.Payment.Mock.TimeoutSeconds, 3)
}

func paymentCircuitBreakerConfig(cfg Config) service.PaymentCircuitBreakerConfig {
	return service.PaymentCircuitBreakerConfig{
		Enabled:             cfg.Payment.Breaker.Enabled,
		ConsecutiveFailures: positiveUint32(cfg.Payment.Breaker.ConsecutiveFailures, 5),
		OpenTimeout:         durationSeconds(cfg.Payment.Breaker.OpenTimeoutSeconds, 30),
		HalfOpenMaxRequests: positiveUint32(cfg.Payment.Breaker.HalfOpenMaxRequests, 1),
	}
}
func buildRepositories(runtime *infraapp.Runtime, cfg Config) *appRepositories {
	pg := infrapostgres.Register(runtime, "main")
	pg.SetDSN(cfg.Postgres.DSN)
	pg.SetPoolSize(cfg.Postgres.MaxIdleConns, cfg.Postgres.MaxOpenConns)
	pg.SetConnMaxIdleTime(cfg.Postgres.ConnMaxIdleTime)
	pg.SetConnMaxLifetime(cfg.Postgres.ConnMaxLifetime)
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
