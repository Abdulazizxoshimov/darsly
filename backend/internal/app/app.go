package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	"github.com/zoom/darsly/internal/infrastructure/telegram"
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

	// ── Telegram arxivi (PRODUCT.md, 2026-08-01) ────────────────────────────
	//
	// Sozlanmagan bo'lsa JIMGINA O'CHIQ: `telegram.New` nop klient qaytaradi,
	// hech bir Telegram ishchisi ishga tushmaydi va yozib olish avvalgidek
	// ishlayveradi. Sentry (`SENTRY_DSN`) bilan bir xil naqsh.
	tgClient := telegram.New(telegram.Config{
		BotToken:       cfg.Telegram.BotToken,
		APIURL:         cfg.Telegram.APIURL,
		ArchiveChatID:  cfg.Telegram.ArchiveChatID,
		MaxUploadBytes: cfg.Telegram.UploadMaxBytes,
		FileRoot:       cfg.Telegram.FileRoot,
	})
	if tgClient.Enabled() {
		// `getMe` — token haqiqiyligini tekshiradi va bot username'ini keshlaydi
		// (deep-link `t.me/<bot>?start=<kod>` uchun SHART).
		ictx, icancel := context.WithTimeout(ctx, 10*time.Second)
		if err := tgClient.Init(ictx); err != nil {
			// Ishga tushishni TO'XTATMAYMIZ: noto'g'ri token butun platformani
			// yiqitmasligi kerak — darslar Telegramsiz ham o'tadi. Lekin
			// integratsiya o'chiriladi, aks holda har yozuv uchun bekorga
			// urinilardi.
			log.Error(ctx, "telegram: bot tokeni tekshirilmadi — integratsiya o'chirildi",
				logger.SafeString("err", err.Error()))
			tgClient = telegram.NewNop()
		} else {
			log.Info(ctx, "telegram configured",
				logger.String("bot", tgClient.BotUsername()),
				logger.String("local_api", strconv.FormatBool(tgClient.Local())),
				logger.String("archive_chat", strconv.FormatInt(cfg.Telegram.ArchiveChatID, 10)))
		}
		icancel()
	} else {
		log.Info(ctx, "telegram not configured — dars arxivi o'chiq (TELEGRAM_BOT_TOKEN)")
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

	enforcer, err := casbin.NewEnforcer()
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
		// Yozuvlar ro'yxatida `expires_at` shu muddatdan hisoblanadi (PRODUCT.md №5).
		RecordingRetention: cfg.Recording.Retention,
		Telegram:           tgClient,
		RecordingCacheTTL:  cfg.Recording.CacheTTL,
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

	// Dars avto-yakuni (PRODUCT.md №2): 4 soatlik texnik limit va bo'shagan
	// xona grace'i. Qoidalar `room.SweepAutoEnd` da; bu yerda faqat tick.
	autoEnd := worker.NewAutoEndWorker(uc.Room, log)
	workersWG.Add(1)
	go func() {
		defer workersWG.Done()
		autoEnd.Run(workerCtx, cfg.Lesson.SweepInterval, cfg.Lesson.MaxDuration, cfg.Lesson.EmptyGrace)
	}()

	// Yozuvlar retention'i (PRODUCT.md №5): 30 kundan keyin MinIO'dan
	// o'chirish + o'chishdan 3 kun oldin mentorga ogohlantirish.
	retentionCfg := worker.RetentionConfig{
		Retention:  cfg.Recording.Retention,
		WarnBefore: cfg.Recording.WarnBefore,
		Interval:   cfg.Recording.SweepInterval,
		BatchLimit: worker.DefaultRetentionConfig().BatchLimit,
		// ⭐ Telegram yoqilgan bo'lsa retention MA'NOSI o'zgaradi: fayl
		// o'chirilmaydi, `archived` bo'ladi va faqat Telegramda tasdiqlangani
		// o'chadi. O'chiq bo'lsa eski xulq (`expired`) saqlanadi.
		TelegramArchive: tgClient.Enabled(),
	}
	workersWG.Add(1)
	go func() {
		defer workersWG.Done()
		worker.NewRetentionWorker(store.Recording, store.Lesson, minioClient, uc.Notification, retentionCfg, log).Run(workerCtx)
	}()

	// Async email worker (RabbitMQ navbatidan SMTP orqali yuboradi).
	if mq != nil {
		workersWG.Add(1)
		go func() { defer workersWG.Done(); worker.NewEmailWorker(mq, directSender, log).Run(workerCtx) }()
	}

	// Yozuvni qayta kodlash (CRF) — hajmni Zoom darajasiga tushiradi.
	// `RECORDING_TRANSCODE=0` bilan o'chiriladi; ffmpeg yo'q bo'lsa ishchi
	// o'zi ishga tushmaydi (bir marta ogohlantirib chiqadi).
	tcCfg := worker.DefaultTranscodeConfig()
	tcCfg.Enabled = os.Getenv("RECORDING_TRANSCODE") != "0"
	if v, err := strconv.Atoi(os.Getenv("RECORDING_TRANSCODE_CRF")); err == nil && v >= 0 && v <= 51 {
		tcCfg.CRF = v
	}
	if v := os.Getenv("RECORDING_TRANSCODE_PRESET"); v != "" {
		tcCfg.Preset = v
	}
	workersWG.Add(1)
	go func() {
		defer workersWG.Done()
		worker.NewTranscodeWorker(store.Recording, minioClient, log, tcCfg).Run(workerCtx)
	}()

	// ── Telegram arxivi ─────────────────────────────────────────────────────
	//
	// Uch ishchi, va ular BIR-BIRIGA BOG'LIQ tartibda yasaladi:
	//   bot   → mentordan «qaysi guruhga?» so'raydi (TelegramPrompter);
	//   upload→ yozuvni arxiv guruhiga yuboradi va tugagach bot'dan so'rashni
	//           iltimos qiladi;
	//   restore→ `restoring` yozuvlarni Telegramdan qaytaradi.
	//
	// Klient o'chiq bo'lsa uchalasi ham darhol chiqib ketadi (`Run` ichida
	// tekshiruv) — shu sababli bu yerda shartli wiring yo'q va kod tarmoqlanmaydi.
	botWorker := worker.NewTelegramBotWorker(
		tgClient, uc.Telegram, store.Recording, store.Lesson, cache,
		worker.NewChatTranscript(store.Chat), log,
	)
	workersWG.Add(1)
	go func() { defer workersWG.Done(); botWorker.Run(workerCtx) }()

	workersWG.Add(1)
	go func() {
		defer workersWG.Done()
		worker.NewTelegramUploadWorker(
			store.Recording, store.Lesson, minioClient, tgClient,
			uc.Notification, botWorker, worker.DefaultTelegramUploadConfig(), log,
		).Run(workerCtx)
	}()

	if tgClient.Enabled() {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			worker.NewRestoreWorker(store.Recording, uc.Recording, log).Run(workerCtx)
		}()
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
		// Mahsulot O'zbekiston uchun — `entity.DefaultTimezone` ga qarang.
		Timezone: entity.DefaultTimezone,
		Language: entity.DefaultLanguage,
		IsActive: true,
	}
	if err := repo.Create(ctx, u); err != nil {
		return fmt.Errorf("seed admin create: %w", err)
	}
	log.Info(ctx, "seed admin created", logger.SafeEmail("email", email))
	return nil
}
