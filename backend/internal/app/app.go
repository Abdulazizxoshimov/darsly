package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	api "github.com/zoom/darsly/api"
	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/email"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/rabbitmq"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/pkg/casbin"
	"github.com/zoom/darsly/internal/pkg/config"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/pkg/postgres"
	"github.com/zoom/darsly/internal/pkg/token"
	"github.com/zoom/darsly/internal/pkg/tracing"
	"github.com/zoom/darsly/internal/storage"
	"github.com/zoom/darsly/internal/usecase"
	"github.com/zoom/darsly/internal/worker"
)

func Run(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	gin.DefaultWriter = os.Stdout
	gin.DefaultErrorWriter = os.Stderr
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	log := logger.New(
		cfg.App.LogLevel, "darsly", "v1",
		logger.WithLoki(cfg.Loki.URL, cfg.Loki.User, cfg.Loki.Password, map[string]string{
			"app":         "darsly-backend",
			"environment": cfg.App.Env,
		}),
	)
	defer func() { _ = logger.Cleanup(log) }()

	ctx := context.Background()
	if cfg.Loki.URL != "" {
		log.Info(ctx, "loki configured", logger.String("url", cfg.Loki.URL))
	} else {
		log.Warn(ctx, "loki not configured — set LOKI_URL to enable log shipping")
	}

	// ── Sentry ──────────────────────────────────────────────────────────────
	if cfg.Sentry.DSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.Sentry.DSN,
			Environment:      cfg.App.Env,
			TracesSampleRate: cfg.Sentry.TracesSampleRate,
			AttachStacktrace: true,
		}); err != nil {
			log.Warn(ctx, "sentry init failed", logger.SafeString("err", err.Error()))
		} else {
			defer sentry.Flush(3 * time.Second)
		}
	}

	// ── Tracing (OpenTelemetry) — OTEL endpoint bo'lmasa no-op ────────────────
	shutdownTracing, err := tracing.Init(ctx, "darsly-backend", cfg.App.Env)
	if err != nil {
		log.Warn(ctx, "tracing init failed", logger.SafeString("err", err.Error()))
		shutdownTracing = func(context.Context) error { return nil }
	}
	defer func() {
		sctx, scancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer scancel()
		_ = shutdownTracing(sctx)
	}()

	// ── Infrastructure ──────────────────────────────────────────────────────
	pg, err := postgres.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pg.DB.Close()

	if err := postgres.RunMigrations(cfg.Postgres, "migrations"); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	log.Info(ctx, "migrations applied")

	cache, err := redis.New(cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	minioClient, err := minio.New(cfg.Minio)
	if err != nil {
		log.Warn(ctx, "minio unavailable — file storage disabled", logger.SafeString("err", err.Error()))
		minioClient = minio.NewNop()
	} else {
		// EnsureBucket deadline bilan — MinIO javob bermasa startup osilib qolmasin.
		bctx, bcancel := context.WithTimeout(ctx, 10*time.Second)
		if err := minioClient.EnsureBucket(bctx); err != nil {
			log.Warn(ctx, "minio bucket check failed — file storage disabled", logger.SafeString("err", err.Error()))
			minioClient = minio.NewNop()
		}
		bcancel()
	}

	var directSender email.Sender
	if cfg.Email.Enabled {
		directSender, err = email.New(cfg.Email, log)
		if err != nil {
			return fmt.Errorf("email: %w", err)
		}
	} else {
		log.Warn(ctx, "email disabled (EMAIL_ENABLED=false) — all email sends will be no-ops")
		directSender = email.NewNopSender()
	}

	mq, err := rabbitmq.New(cfg.RabbitMQ.URL, log)
	if err != nil {
		log.Warn(ctx, "rabbitmq unavailable", logger.SafeString("err", err.Error()))
		mq = nil
	} else {
		defer mq.Close()
	}

	// Email: RabbitMQ bor bo'lsa — async (navbat orqali), aks holda to'g'ridan-to'g'ri.
	// Async email SMTP kutishni request oqimidan ajratadi (latency'ni buzmaydi).
	var emailSender email.Sender = directSender
	if mq != nil {
		emailSender = email.NewQueuedSender(mq)
	}

	hub := websocket.NewHub(log)
	// Ko'p-instans (horizontal scale) uchun Redis fan-out — real-time xabarlar
	// barcha instanslarга tarqaladi.
	hub.EnableFanout(ctx, cache)

	lkClient := livekit.New(cfg.LiveKit)
	if lkClient.Enabled() {
		log.Info(ctx, "livekit configured", logger.String("host", cfg.LiveKit.Host))
	} else {
		log.Warn(ctx, "livekit not configured — video rooms disabled (set LIVEKIT_API_KEY/SECRET/HOST)")
	}

	// Egress (dars yozib olish) MinIO'ga yozadi. Endpoint Egress konteyneri uchun
	// docker-reachable bo'lishi kerak (MINIO_S3_ENDPOINT bilan override qilinadi).
	s3Scheme := "http://"
	if cfg.Minio.UseSSL {
		s3Scheme = "https://"
	}
	s3Endpoint := os.Getenv("MINIO_S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = s3Scheme + cfg.Minio.Endpoint
	}
	recordingS3 := livekit.S3Config{
		AccessKey: cfg.Minio.AccessKey,
		Secret:    cfg.Minio.SecretKey,
		Region:    "us-east-1",
		Endpoint:  s3Endpoint,
		Bucket:    cfg.Minio.Bucket,
	}

	tokenMaker := token.NewJWTMaker(
		[]byte(cfg.JWT.Secret),
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
		cfg.JWT.RefreshGrace,
		cache.Client(),
		// Prefiks ataylab "session:" (oxirida ikki nuqta bilan) — JWTMaker o'zi ham
		// ":sess:"/":refresh:" qo'shgani uchun kalitlar IKKI ikki nuqtali chiqadi:
		// `session::sess:<sid>`, `session::refresh:<jti>`, `session::refresh:used:<jti>`.
		// Kosmetik nuqson, LEKIN ATAYLAB O'ZGARTIRILMAYDI: prefiksni "session"ga
		// tuzatish barcha mavjud kalitlarni "yo'q" qilib qo'yadi → deploy paytida
		// BARCHA foydalanuvchi tizimdan chiqib ketadi (dars o'rtasida ham). Kosmetik
		// foyda bu uzilishga arzimaydi. Operatsion skriptlar shu shaklga qarab yozilsin:
		//   redis-cli --scan --pattern 'session::sess:*'
		"session:",
		log,
	)

	enforcer, err := casbin.NewEnforcer("internal/pkg/casbin/model.conf", "internal/pkg/casbin/policy.csv")
	if err != nil {
		log.Warn(ctx, "casbin enforcer not loaded, RBAC disabled")
		enforcer = nil
	}

	// ── Metrics ─────────────────────────────────────────────────────────────
	metrics.RegisterPool(pg.DB)
	metrics.RegisterRedis(cache.Client())
	metrics.RegisterWSActive(func() float64 { return float64(hub.Count()) })

	// ── Storage / Usecase / Handler ─────────────────────────────────────────
	store := storage.New(pg)

	// Seed-admin — user-management admin roliga cheklangani uchun (H-2) hech bo'lmasa
	// bitta admin kerak. SEED_ADMIN_EMAIL/PASSWORD berilsa va admin hali yo'q bo'lsa yaratadi.
	if err := seedAdmin(ctx, cfg, store.User, hasher.New(cfg.App.BcryptCost), log); err != nil {
		log.Warn(ctx, "seed admin failed", logger.SafeString("err", err.Error()))
	}

	uc := usecase.New(usecase.Deps{
		Store:           store,
		TokenMaker:      tokenMaker,
		Hasher:          hasher.New(cfg.App.BcryptCost),
		Minio:           minioClient,
		Cache:           cache,
		Log:             log,
		Hub:             hub,
		EmailSender:     emailSender,
		LiveKit:         lkClient,
		RecordingS3:     recordingS3,
		RefreshTTL:      cfg.JWT.RefreshTTL,
		FrontendBaseURL: cfg.App.FrontendBaseURL,
	})

	h := BuildHandler(uc, hub, lkClient, cfg)

	// ── Background workers ────────────────────────────────────────────────────
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	reminderInterval := time.Minute
	if v, err := time.ParseDuration(os.Getenv("REMINDER_INTERVAL")); err == nil && v > 0 {
		reminderInterval = v
	}
	reminderLead := 15 * time.Minute
	if v, err := time.ParseDuration(os.Getenv("REMINDER_LEAD")); err == nil && v > 0 {
		reminderLead = v
	}
	// WaitGroup — shutdown'da worker'lar in-flight ishni tugatishini kutamiz.
	var workersWG sync.WaitGroup
	reminder := worker.NewReminderWorker(store.Lesson, uc.Notification, log)
	workersWG.Add(1)
	go func() { defer workersWG.Done(); reminder.Run(workerCtx, reminderInterval, reminderLead) }()

	// Async email worker (RabbitMQ navbatidan SMTP orqali yuboradi).
	if mq != nil {
		workersWG.Add(1)
		go func() { defer workersWG.Done(); worker.NewEmailWorker(mq, directSender, log).Run(workerCtx) }()
	}

	// ── Server ──────────────────────────────────────────────────────────────
	readyFn := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := pg.DB.Ping(ctx); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
		if err := cache.Ping(ctx); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		return nil
	}

	router := api.NewRouter(h, tokenMaker, enforcer, cache, log, readyFn)
	srv := api.NewServer(":"+cfg.App.Port, router)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(context.Background(), "server error", logger.SafeString("err", err.Error()))
		}
	}()

	log.Info(ctx, fmt.Sprintf("server started on :%s", cfg.App.Port))

	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Avval HTTP serverni drain qilamiz (yangi WS upgrade'lar to'xtaydi),
	// keyin ochiq WS ulanishlarni yopamiz — teskari tartibdagi race oldini oladi.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	hub.Stop()

	// Worker'larга to'xtash signali va in-flight ishни (email ack, reminder tick) drain.
	workerCancel()
	workersDone := make(chan struct{})
	go func() { workersWG.Wait(); close(workersDone) }()
	select {
	case <-workersDone:
	case <-shutdownCtx.Done():
		log.Warn(ctx, "workers did not drain within shutdown window")
	}

	log.Info(ctx, "server stopped")
	return nil
}

// seedAdmin — SEED_ADMIN_EMAIL/SEED_ADMIN_PASSWORD berilgan bo'lsa va shu email'li
// foydalanuvchi hali mavjud bo'lmasa, `admin` rolli foydalanuvchi yaratadi.
// Idempotent: mavjud bo'lsa hech narsa qilmaydi (parolni qayta yozmaydi).
func seedAdmin(ctx context.Context, cfg *config.Config, repo repository.UserRepository, h hasher.Hasher, log logger.Logger) error {
	email := cfg.App.SeedAdminEmail
	password := cfg.App.SeedAdminPassword
	if email == "" || password == "" {
		return nil // seed o'chirilgan
	}
	exists, err := repo.ExistsByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("seed admin exists-check: %w", err)
	}
	if exists {
		log.Info(ctx, "seed admin already exists — skipping", logger.SafeEmail("email", email))
		return nil
	}
	hashed, err := h.Hash(password)
	if err != nil {
		return fmt.Errorf("seed admin hash: %w", err)
	}
	u := &entity.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hashed,
		FullName:     "Administrator",
		Color:        "#6366F1",
		Role:         "admin",
		Timezone:     "UTC",
		Language:     "uz",
		IsActive:     true,
	}
	if err := repo.Create(ctx, u); err != nil {
		return fmt.Errorf("seed admin create: %w", err)
	}
	log.Info(ctx, "seed admin created", logger.SafeEmail("email", email))
	return nil
}
