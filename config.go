package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App                AppConfig
	Queue              QueueConfig
	Order              OrderConfig
	Session            SessionConfig
	Postgres           PostgresConfig
	Redis              RedisConfig
	Payment            PaymentConfig
	Outbox             OutboxConfig
	ConfigSourceLoaded string
}

type AppConfig struct {
	Env          string
	ServerAddr   string
	PprofEnabled bool
}

type QueueConfig struct {
	Store                 string
	ReleaseLimit          int
	JoinMaxInFlight       int
	JoinRetryAfterSeconds int
}

type OrderConfig struct {
	ExpireBatchSize   int
	PaymentTTLMinutes int
}

type SessionConfig struct {
	TTLHours int
}

type PostgresConfig struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	DB       int
	Password string
}

type PaymentConfig struct {
	Mock      PaymentMockConfig
	Breaker   servicePaymentBreakerConfig
	Reconcile PaymentReconcileConfig
}

type PaymentMockConfig struct {
	MerchantID     string
	HashKey        string
	HashIV         string
	BaseURL        string
	BackupBaseURL  string
	CallbackURL    string
	TimeoutSeconds int
}

type servicePaymentBreakerConfig struct {
	Enabled             bool
	ConsecutiveFailures uint32
	OpenTimeoutSeconds  int
	HalfOpenMaxRequests uint32
}

type PaymentReconcileConfig struct {
	Enabled           bool
	BatchSize         int
	DelaySeconds      int
	RetryAfterSeconds int
	MaxAttempts       int
}

type OutboxConfig struct {
	Publish OutboxPublishConfig
}

type OutboxPublishConfig struct {
	Enabled           bool
	BatchSize         int
	RetryAfterSeconds int
}

func loadConfig() Config {
	sourceLoaded := loadDotEnvIfLocal()
	appEnv := envString("APP_ENV", "local")
	queueStoreDefault := "memory"
	orderPaymentTTLDefault := 10
	redisAddrDefault := ""
	paymentMockMerchantIDDefault := ""
	paymentMockHashKeyDefault := ""
	paymentMockHashIVDefault := ""
	if appEnv == "dev" {
		queueStoreDefault = "redis"
		orderPaymentTTLDefault = 3
		redisAddrDefault = "localhost:6379"
		paymentMockMerchantIDDefault = "3002607"
		paymentMockHashKeyDefault = "pwFHCqoQZGmho4w6"
		paymentMockHashIVDefault = "EkRm7iFT261dpevs"
	}

	cfg := Config{
		App: AppConfig{
			Env:          appEnv,
			ServerAddr:   envString("SERVER_ADDR", ":8080"),
			PprofEnabled: envBool("PPROF_ENABLED", false),
		},
		Queue: QueueConfig{
			Store:                 envString("QUEUE_STORE", queueStoreDefault),
			ReleaseLimit:          envInt("QUEUE_RELEASE_LIMIT", 50),
			JoinMaxInFlight:       envInt("QUEUE_JOIN_MAX_IN_FLIGHT", 500),
			JoinRetryAfterSeconds: envInt("QUEUE_JOIN_RETRY_AFTER_SECONDS", 1),
		},
		Order: OrderConfig{
			ExpireBatchSize:   envInt("ORDER_EXPIRE_BATCH_SIZE", 100),
			PaymentTTLMinutes: envInt("ORDER_PAYMENT_TTL_MINUTES", orderPaymentTTLDefault),
		},
		Session: SessionConfig{
			TTLHours: envInt("SESSION_TTL_HOURS", 168),
		},
		Postgres: PostgresConfig{
			DSN:             envString("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/buy_ticket?sslmode=disable"),
			MaxIdleConns:    envInt("POSTGRES_POOL_MAX_IDLE_CONNS", 5),
			MaxOpenConns:    envInt("POSTGRES_POOL_MAX_OPEN_CONNS", 20),
			ConnMaxIdleTime: envDuration("POSTGRES_CONN_MAX_IDLE_TIME", 30*time.Minute),
			ConnMaxLifetime: envDuration("POSTGRES_CONN_MAX_LIFETIME", 2*time.Hour),
		},
		Redis: RedisConfig{
			Addr:     envString("REDIS_ADDR", redisAddrDefault),
			DB:       envInt("REDIS_DB", 0),
			Password: envString("REDIS_PASSWORD", ""),
		},
		Payment: PaymentConfig{
			Mock: PaymentMockConfig{
				MerchantID:     envString("PAYMENT_MOCK_MERCHANT_ID", paymentMockMerchantIDDefault),
				HashKey:        envString("PAYMENT_MOCK_HASH_KEY", paymentMockHashKeyDefault),
				HashIV:         envString("PAYMENT_MOCK_HASH_IV", paymentMockHashIVDefault),
				BaseURL:        envString("PAYMENT_MOCK_BASE_URL", "http://localhost:8081"),
				BackupBaseURL:  envString("PAYMENT_MOCK_BACKUP_BASE_URL", ""),
				CallbackURL:    envString("PAYMENT_MOCK_CALLBACK_URL", "http://localhost:8080/payments/provider/ecpay/callback"),
				TimeoutSeconds: envInt("PAYMENT_MOCK_TIMEOUT_SECONDS", 3),
			},
			Breaker: servicePaymentBreakerConfig{
				Enabled:             envBool("PAYMENT_BREAKER_ENABLED", true),
				ConsecutiveFailures: envUint32("PAYMENT_BREAKER_CONSECUTIVE_FAILURES", 5),
				OpenTimeoutSeconds:  envInt("PAYMENT_BREAKER_OPEN_TIMEOUT_SECONDS", 30),
				HalfOpenMaxRequests: envUint32("PAYMENT_BREAKER_HALF_OPEN_MAX_REQUESTS", 1),
			},
			Reconcile: PaymentReconcileConfig{
				Enabled:           envBool("PAYMENT_RECONCILE_ENABLED", true),
				BatchSize:         envInt("PAYMENT_RECONCILE_BATCH_SIZE", 100),
				DelaySeconds:      envInt("PAYMENT_RECONCILE_DELAY_SECONDS", 120),
				RetryAfterSeconds: envInt("PAYMENT_RECONCILE_RETRY_AFTER_SECONDS", 30),
				MaxAttempts:       envInt("PAYMENT_RECONCILE_MAX_ATTEMPTS", 5),
			},
		},
		Outbox: OutboxConfig{
			Publish: OutboxPublishConfig{
				Enabled:           envBool("OUTBOX_PUBLISH_ENABLED", true),
				BatchSize:         envInt("OUTBOX_PUBLISH_BATCH_SIZE", 100),
				RetryAfterSeconds: envInt("OUTBOX_PUBLISH_RETRY_AFTER_SECONDS", 30),
			},
		},
		ConfigSourceLoaded: sourceLoaded,
	}

	validateConfig(cfg)
	return cfg
}

func loadDotEnvIfLocal() string {
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	if appEnv != "" && appEnv != "local" && appEnv != "dev" {
		return "env"
	}
	if err := loadDotEnv(".env"); err != nil {
		return "env"
	}
	return ".env"
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func validateConfig(cfg Config) {
	if cfg.App.ServerAddr == "" {
		fatalLog("SERVER_ADDR is required")
	}
	if cfg.Postgres.DSN == "" {
		fatalLog("POSTGRES_DSN is required")
	}
	switch cfg.Queue.Store {
	case "memory", "redis":
	default:
		fatalLog("unsupported QUEUE_STORE", "value", cfg.Queue.Store)
	}
	if cfg.Queue.Store == "redis" && cfg.Redis.Addr == "" {
		fatalLog("REDIS_ADDR is required when QUEUE_STORE=redis")
	}
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envUint32(key string, fallback uint32) uint32 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil || parsed == 0 {
		return fallback
	}
	return uint32(parsed)
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func durationSeconds(seconds int, fallbackSeconds int) time.Duration {
	if seconds <= 0 {
		seconds = fallbackSeconds
	}
	return time.Duration(seconds) * time.Second
}

func durationMinutes(minutes int, fallbackMinutes int) time.Duration {
	if minutes <= 0 {
		minutes = fallbackMinutes
	}
	return time.Duration(minutes) * time.Minute
}

func durationHours(hours int, fallbackHours int) time.Duration {
	if hours <= 0 {
		hours = fallbackHours
	}
	return time.Duration(hours) * time.Hour
}

func positiveInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func nonNegativeInt(value int, fallback int) int {
	if value < 0 {
		return fallback
	}
	return value
}

func positiveUint32(value uint32, fallback uint32) uint32 {
	if value == 0 {
		return fallback
	}
	return value
}

func configSummary(cfg Config) []any {
	return []any{
		"env", cfg.App.Env,
		"source", cfg.ConfigSourceLoaded,
		"queue_store", cfg.Queue.Store,
		"server_addr", cfg.App.ServerAddr,
		"redis_configured", cfg.Redis.Addr != "",
	}
}
