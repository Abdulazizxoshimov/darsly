package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig      `json:"app"`
	Postgres PostgresConfig `json:"postgres"`
	Redis    RedisConfig    `json:"redis"`
	Minio    MinioConfig    `json:"minio"`
	RabbitMQ RabbitMQConfig `json:"rabbitmq"`
	JWT      JWTConfig      `json:"jwt"`
	Email    EmailConfig    `json:"email"`
	Sentry   SentryConfig   `json:"sentry"`
	Loki     LokiConfig     `json:"loki"`
	LiveKit  LiveKitConfig  `json:"livekit"`
	Mobile   MobileConfig   `json:"mobile"`
}

// MobileConfig — mobil klient versiya nazorati (`GET /api/v1/app-config`).
// Side-load qilingan APK'da avtomatik yangilanish yo'q, shuning uchun buzuq
// versiyani to'xtatishning yagona yo'li — klient ishga tushganda serverdan
// min_version so'rashi. Qiymatlar env'dan (deploy vaqtida o'zgartiriladi).
type MobileConfig struct {
	AndroidMinVersion    string `json:"android_min_version"`
	AndroidLatestVersion string `json:"android_latest_version"`
	AndroidAPKURL        string `json:"android_apk_url"`
	AndroidForceUpdate   bool   `json:"android_force_update"`
	AndroidReleaseNotes  string `json:"android_release_notes"`
}

// LiveKitConfig — SFU media server (video-dars) sozlamalari.
type LiveKitConfig struct {
	Host          string        `json:"host"`            // masalan: ws://localhost:7880
	APIKey        string        `json:"api_key"`         // LiveKit API key
	APISecret     string        `json:"api_secret"`      // LiveKit API secret
	WebhookAPIKey string        `json:"webhook_api_key"` // Egress webhook imzosini tekshirish uchun
	TokenTTL      time.Duration `json:"token_ttl"`       // kirish tokeni muddati (0 → 6h default)
	// EgressLayout — yozib olish kompozitsiya shabloni (LiveKit default template).
	// Yaroqli qiymatlar: speaker | single-speaker | grid (+ ixtiyoriy "-light" qo'shimchasi).
	// Default "speaker": dars yozuvida ustoz/ekran asosiy oynada bo'lsin (grid'da
	// slayd matni o'qilmas kichik plitkaga tushib qoladi). Qarang: livekit/egress.go.
	EgressLayout string `json:"egress_layout"`
}

type SentryConfig struct {
	DSN              string  `json:"dsn"`
	TracesSampleRate float64 `json:"traces_sample_rate"`
}

type LokiConfig struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type AppConfig struct {
	Port                  string `json:"port"`
	Env                   string `json:"env"`
	LogLevel              string `json:"log_level"`
	FrontendBaseURL       string `json:"frontend_base_url"`
	AllowOpenRegistration bool   `json:"allow_open_registration"`
	// BcryptCost — parol hash narxi (10-11 tavsiya). Yuqori qiymat xavfsizroq lekin
	// login/join hot-path'da CPU'ni ko'proq yeydi (thundering-herd xavfi).
	BcryptCost int `json:"bcrypt_cost"`
	// SeedAdminEmail/Password — startup'da admin roli mavjud bo'lmasa yaratiladi
	// (faqat ikkalasi ham to'ldirilsa; mavjud bo'lsa hech narsa qilmaydi). User-management
	// admin'ga cheklangani uchun (H-2) hech bo'lmasa bitta admin kerak.
	SeedAdminEmail    string `json:"-"`
	SeedAdminPassword string `json:"-"`
}

type PostgresConfig struct {
	Host              string        `json:"host"`
	Port              string        `json:"port"`
	Database          string        `json:"database"`
	Username          string        `json:"username"`
	Password          string        `json:"password"`
	MaxConns          int32         `json:"max_conns"`
	MinConns          int32         `json:"min_conns"`
	MaxConnIdleTime   time.Duration `json:"max_conn_idle_time"`
	MaxConnLifetime   time.Duration `json:"max_conn_lifetime"`
	HealthCheckPeriod time.Duration `json:"health_check_period"`
}

// DSN returns a pgxpool-compatible connection string.
func (p PostgresConfig) DSN() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	sslMode := "disable"
	if os.Getenv("APP_ENV") == "production" {
		sslMode = "require"
	}
	return "host=" + p.Host +
		" port=" + p.Port +
		" dbname=" + p.Database +
		" user=" + p.Username +
		" password=" + p.Password +
		" sslmode=" + sslMode
}

// Validate checks that required config values are present and safe.
// Call this at startup before wiring any infrastructure.
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET must be set")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}
	if c.Postgres.Password == "" && os.Getenv("DATABASE_URL") == "" {
		return fmt.Errorf("DB_PASSWORD must be set (or use DATABASE_URL)")
	}

	// Production'da qo'shimcha qat'iy talablar.
	if c.App.Env == "production" {
		low := strings.ToLower(c.JWT.Secret)
		if strings.Contains(low, "dev") || strings.Contains(low, "change") || strings.Contains(low, "secret_at_least") {
			return fmt.Errorf("production JWT_SECRET must not be a dev/default value")
		}
		if c.App.FrontendBaseURL == "" {
			return fmt.Errorf("FRONTEND_BASE_URL must be set in production (required for WebSocket Origin check)")
		}
		if c.LiveKit.APISecret != "" {
			ls := strings.ToLower(c.LiveKit.APISecret)
			if strings.Contains(ls, "secret_at_least") || strings.Contains(ls, "change") {
				return fmt.Errorf("production LIVEKIT_API_SECRET must not be a dev/default value")
			}
		}
		if c.Redis.Password == "" {
			return fmt.Errorf("REDIS_PASSWORD must be set in production")
		}
		if c.Minio.SecretKey == "" || c.Minio.SecretKey == "minioadmin" {
			return fmt.Errorf("production MINIO_SECRET_KEY must not be empty/default")
		}
	}
	return nil
}

type RedisConfig struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Password     string `json:"password"`
	DB           int    `json:"db"`
	PoolSize     int    `json:"pool_size"`      // 0 → go-redis default (10×GOMAXPROCS)
	MinIdleConns int    `json:"min_idle_conns"` // issiq ulanishlar (latency spike'ni kamaytiradi)
}

// Addr returns "host:port" for redis.NewClient.
func (r RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
}

type MinioConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"use_ssl"`
	// PublicEndpoint — presigned URL'lar shu manzilga imzolanadi (tashqi klient
	// yeta oladigan ochiq domen). Bo'sh bo'lsa Endpoint ishlatiladi.
	PublicEndpoint string `json:"public_endpoint"`
	PublicUseSSL   bool   `json:"public_use_ssl"`
}

type RabbitMQConfig struct {
	URL string `json:"url"`
}

type JWTConfig struct {
	Secret     string        `json:"secret"`
	AccessTTL  time.Duration `json:"access_ttl"`
	RefreshTTL time.Duration `json:"refresh_ttl"`
	// RefreshGrace — refresh rotatsiyasining idempotentlik oynasi. Mobil tarmoqda
	// javob yo'qolsa yoki ikki parallel 401-retry ketsa, klient eski refresh bilan
	// qayta uradi; shu oyna ichida server ayni o'sha juftlikni qaytaradi (sessiyani
	// o'ldirmaydi). Oyna tashqarisida reuse haliyam o'g'irlik deb qaraladi.
	// 0 → grace o'chirilgan (eski xatti-harakat).
	RefreshGrace time.Duration `json:"refresh_grace"`
}

type EmailConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	From     string `json:"from"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Load reads environment variables into Config.
// In local development, set values via .env file loaded externally or shell exports.
func Load() *Config {
	return &Config{
		App: AppConfig{
			Port:                  getEnv("APP_PORT", "8080"),
			Env:                   getEnv("APP_ENV", "development"),
			LogLevel:              getEnv("LOG_LEVEL", "info"),
			FrontendBaseURL:       getEnv("FRONTEND_BASE_URL", "http://localhost:3000"),
			AllowOpenRegistration: getEnvBool("ALLOW_OPEN_REGISTRATION", false),
			BcryptCost:            getEnvInt("BCRYPT_COST", 11),
			SeedAdminEmail:        getEnv("SEED_ADMIN_EMAIL", ""),
			SeedAdminPassword:     getEnv("SEED_ADMIN_PASSWORD", ""),
		},
		Postgres: PostgresConfig{
			Host:              getEnv("DB_HOST", "localhost"),
			Port:              getEnv("DB_PORT", "5432"),
			Database:          getEnv("DB_NAME", "darsly"),
			Username:          getEnv("DB_USER", "postgres"),
			Password:          getEnv("DB_PASSWORD", ""),
			MaxConns:          int32(getEnvInt("DB_MAX_CONNS", 50)),
			MinConns:          int32(getEnvInt("DB_MIN_CONNS", 5)),
			MaxConnIdleTime:   getEnvDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
			MaxConnLifetime:   getEnvDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
			HealthCheckPeriod: getEnvDuration("DB_HEALTH_CHECK_PERIOD", 1*time.Minute),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 0),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 0),
		},
		Minio: MinioConfig{
			Endpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:         getEnv("MINIO_BUCKET", "darsly"),
			UseSSL:         getEnvBool("MINIO_USE_SSL", false),
			PublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", ""),
			PublicUseSSL:   getEnvBool("MINIO_PUBLIC_USE_SSL", true),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		JWT: JWTConfig{
			Secret:       getEnv("JWT_SECRET", ""),
			AccessTTL:    getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:   getEnvDuration("JWT_REFRESH_TTL", 720*time.Hour),
			RefreshGrace: getEnvDuration("JWT_REFRESH_GRACE", 60*time.Second),
		},
		Email: EmailConfig{
			Enabled:  getEnvBool("EMAIL_ENABLED", true),
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnvInt("SMTP_PORT", 587),
			From:     getEnv("SMTP_FROM", ""),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
		},
		Sentry: SentryConfig{
			DSN:              getEnv("SENTRY_DSN", ""),
			TracesSampleRate: getEnvFloat("SENTRY_TRACES_SAMPLE_RATE", 0.2),
		},
		Loki: LokiConfig{
			URL:      getEnv("LOKI_URL", ""),
			User:     getEnv("LOKI_USER", ""),
			Password: getEnv("LOKI_PASSWORD", ""),
		},
		LiveKit: LiveKitConfig{
			Host:          getEnv("LIVEKIT_HOST", "ws://localhost:7880"),
			APIKey:        getEnv("LIVEKIT_API_KEY", ""),
			APISecret:     getEnv("LIVEKIT_API_SECRET", ""),
			WebhookAPIKey: getEnv("LIVEKIT_WEBHOOK_API_KEY", ""),
			TokenTTL:      getEnvDuration("LIVEKIT_TOKEN_TTL", 6*time.Hour),
			EgressLayout:  getEnv("LIVEKIT_EGRESS_LAYOUT", "speaker"),
		},
		Mobile: MobileConfig{
			AndroidMinVersion:    getEnv("APP_ANDROID_MIN_VERSION", "1.0.0"),
			AndroidLatestVersion: getEnv("APP_ANDROID_LATEST_VERSION", "1.0.0"),
			AndroidAPKURL:        getEnv("APP_ANDROID_APK_URL", ""),
			AndroidForceUpdate:   getEnvBool("APP_ANDROID_FORCE_UPDATE", false),
			AndroidReleaseNotes:  getEnv("APP_ANDROID_RELEASE_NOTES", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getEnvFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}
