package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/zoom/darsly/api/handlers"
	v1 "github.com/zoom/darsly/api/handlers/v1"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/pkg/token"
)

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

// trustedProxies TRUSTED_PROXIES (vergul bilan) dan CIDR/IP ro'yxatini o'qiydi.
// Bo'sh bo'lsa nil — gin hech qanday proksiga ishonmaydi (ClientIP = RemoteAddr).
func trustedProxies() []string {
	raw := os.Getenv("TRUSTED_PROXIES")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func NewRouter(h *handlers.Handler, tokenMaker token.Maker, enforcer *casbin.Enforcer, cache redis.Cache, log logger.Logger, readyFn func() error) *gin.Engine {
	installValidator() // ShouldBindJSON `validate` teglarini yuritishi uchun
	r := gin.New()

	// Ishonchli proksilar — aks holda X-Forwarded-For'ni istalgan mijoz soxtalashtirishi
	// mumkin (rate-limit va IP-log buziladi). TRUSTED_PROXIES bo'sh → hech kimga ishonmaydi
	// (ClientIP = RemoteAddr). Reverse-proxy ortida bo'lsangiz proksi IP/CIDR'ini kiriting.
	_ = r.SetTrustedProxies(trustedProxies())

	allowedOrigins := []string{}
	if frontendURL := os.Getenv("FRONTEND_BASE_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}

	r.Use(
		otelgin.Middleware("darsly-backend"), // OpenTelemetry span (OTEL endpoint bo'lmasa no-op)
		middleware.Sentry(),
		middleware.RequestID(),
		middleware.Logger(log), // har so'rovni strukturaviy (request_id bilan) yozadi
		middleware.Recover(log),
		middleware.CORS(allowedOrigins...),
		middleware.SecurityHeaders(),
		metrics.Middleware(),
		// Javoblarni siqish (yomon internetda har bayt muhim). WS/metrics istisno.
		gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/api/v1/ws", "/metrics"})),
	)

	r.MaxMultipartMemory = 10 << 20 // 10 MB multipart

	// JSON body hajmini 2 MB bilan cheklash
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
		c.Next()
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", v1.HealthCheck())
	r.GET("/ready", v1.ReadyCheck(readyFn, log))
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	auth := middleware.Auth(tokenMaker)
	rbac := middleware.EnforceCasbin(enforcer, log)

	rps := envFloat("RATE_LIMIT_RPS", 30)
	burst := envInt("RATE_LIMIT_BURST", 60)

	api := r.Group("/api/v1")

	// Ilova konfiguratsiyasi (ochiq, auth'siz) — mobil klient ishga tushishida
	// min_version/force_update ni shu yerdan oladi. Auth'dan OLDIN chaqirilganligi
	// uchun token talab qilinmaydi; javob statik (env'dan) — IP bo'yicha cheklangan.
	api.GET("/app-config", middleware.RateLimit(20, 40), v1.GetAppConfig(h))

	// Auth (public) — Redis-backed rate-limit (ko'p-instansda umumiy hisoblagich).
	//
	// BE-11: login/refresh uchun endi FAQAT IP bo'yicha cheklov yo'q. Mobil operator
	// CGNAT'i ortida minglab abonent bitta IP'da bo'ladi — yagona 60/min/IP ertalabki
	// ommaviy kirishda hammani 429 bilan to'sardi. Har endpoint o'z ma'noli o'lchamiga
	// ko'chirildi (batafsil asos: middleware/ratelimit_auth.go).
	authGroup := api.Group("/auth")
	{
		// Login: (IP+email) qattiq → email (har qanday IP) → IP shifti.
		loginLimits := middleware.RateLimitLogin(cache,
			int64(envInt("RATE_LIMIT_LOGIN_IP_EMAIL", 10)),
			int64(envInt("RATE_LIMIT_LOGIN_EMAIL", 30)),
			int64(envInt("RATE_LIMIT_LOGIN_IP", 200)),
			time.Minute,
		)
		authGroup.POST("/login", append(loginLimits, v1.Login(h))...)

		// Refresh: sessiya (sid) bo'yicha + IP shifti. Refresh brute-force qilinmaydi
		// (imzolangan sir), shuning uchun IP emas, sessiya to'g'ri o'lcham.
		// tokenMaker uzatiladi: sid rate-limit kaliti bo'lishidan OLDIN imzo tekshiriladi
		// (aks holda hujumchi qurbonning sid'i bilan uning bucket'ini to'ldirardi).
		refreshLimits := middleware.RateLimitRefresh(cache, tokenMaker,
			int64(envInt("RATE_LIMIT_REFRESH_SID", 30)),
			int64(envInt("RATE_LIMIT_REFRESH_IP", 600)),
			time.Minute,
		)
		authGroup.POST("/refresh", append(refreshLimits, v1.Refresh(h))...)

		// Qolganlari (spam/abuse vektorlari) — avvalgi IP bo'yicha cheklovda qoladi.
		ipLimit := middleware.RateLimitRedis(cache, "auth", int64(envInt("RATE_LIMIT_AUTH_IP", 60)), time.Minute)
		authGroup.POST("/register", ipLimit, v1.Register(h))
		authGroup.POST("/forgot-password", ipLimit, v1.ForgotPassword(h))
		authGroup.POST("/reset-password", ipLimit, v1.ResetPassword(h))
	}

	// Join havolasi (ochiq) — slug orqali dars ma'lumoti va kirish.
	// Redis-backed (instanslar aro izchil). NAT ortidagi butun sinf bir vaqtda
	// kiradi (500+ o'quvchi bitta IP) — shuning uchun daqiqasiga 600 so'rov/IP
	// saxiy, lekin bitta scraper'ni baribir cheklaydi. Slug crypto-random.
	joinGroup := api.Group("/joinlink", middleware.RateLimitRedis(cache, "join", 600, time.Minute))
	{
		joinGroup.GET("/:slug", v1.PreviewJoinLink(h))
		joinGroup.POST("/:slug", v1.JoinLink(h))
	}

	// Kutish xonasi holati (ochiq) — guest polling / WS fallback. Polling ko'p bo'lishi
	// mumkin, lekin IP bo'yicha cheklangan (DoS himoyasi).
	api.GET("/waitingroom/:id/status", middleware.RateLimit(20, 40), v1.WaitingRoomStatus(h))

	// LiveKit webhook (ochiq, imzo bilan himoyalangan) — Egress status. Bitta manba (LiveKit).
	api.POST("/webhooks/livekit", middleware.RateLimit(30, 60), v1.LiveKitWebhook(h))

	// WebSocket'lar — ulanish urinishlari IP bo'yicha cheklangan (DoS himoyasi).
	api.GET("/ws", middleware.RateLimit(10, 20), auth, v1.WSConnect(h))                // mentor (authed, ?token=)
	api.GET("/ws/waitingroom", middleware.RateLimit(10, 20), v1.WSGuestWaitingRoom(h)) // guest (ochiq, ?request_id=)

	// Himoyalangan route'lar
	protected := api.Group("/", auth, rbac, middleware.RateLimitByUser(rps, burst))

	protected.POST("/auth/logout", v1.Logout(h))
	protected.GET("/auth/me", v1.Me(h))

	// Lessons (mentor)
	lessons := protected.Group("/lessons")
	{
		lessons.POST("", v1.CreateLesson(h))
		lessons.GET("", v1.ListLessons(h))
		lessons.GET("/:id", v1.GetLesson(h))
		lessons.PATCH("/:id", v1.UpdateLesson(h))
		lessons.DELETE("/:id", v1.DeleteLesson(h))
		// Video xona (LiveKit)
		lessons.POST("/:id/token", v1.GetRoomToken(h))         // host token
		lessons.POST("/:id/end", v1.EndLesson(h))              // darsni yakunlash
		lessons.GET("/:id/waitingroom", v1.ListWaitingRoom(h)) // kutayotgan so'rovlar
		// Host boshqaruvi
		lessons.GET("/:id/participants", v1.ListParticipants(h))
		lessons.POST("/:id/mute-all", v1.MuteAll(h))
		lessons.POST("/:id/participants/:identity/mute", v1.MuteParticipant(h))
		lessons.POST("/:id/participants/:identity/remove", v1.RemoveParticipant(h))
		lessons.POST("/:id/participants/:identity/allow-speak", v1.AllowSpeak(h))
		lessons.POST("/:id/participants/:identity/revoke-speak", v1.RevokeSpeak(h))
		// Chat (host — persist + LiveKit broadcast; guest chat frontend data-channel orqali)
		lessons.GET("/:id/chat", v1.ChatHistory(h))
		lessons.POST("/:id/chat", v1.SendChat(h))
		// So'rovnomalar (host)
		lessons.GET("/:id/polls", v1.ListPolls(h))
		lessons.POST("/:id/polls", v1.CreatePoll(h))
		// Yozib olish
		lessons.POST("/:id/recording/start", v1.StartRecording(h))
		lessons.GET("/:id/recordings", v1.ListRecordings(h))
	}

	// So'rovnoma yopish (host)
	protected.POST("/polls/:id/close", v1.ClosePoll(h))

	// So'rovnoma ovoz/natija (ochiq — LiveKit room-token bilan)
	api.POST("/polls/:id/vote", middleware.RateLimit(20, 40), v1.VotePoll(h))
	api.GET("/polls/:id/results", middleware.RateLimit(20, 40), v1.PollResults(h))

	// Yozuvlar (mentor)
	recordings := protected.Group("/recordings")
	{
		recordings.POST("/:id/stop", v1.StopRecording(h))
		recordings.GET("/:id/download", v1.DownloadRecording(h))
	}

	// Kutish xonasi qarorlari (mentor)
	waiting := protected.Group("/waitingroom")
	{
		waiting.POST("/:id/admit", v1.AdmitWaitingRoom(h))
		waiting.POST("/:id/reject", v1.RejectWaitingRoom(h))
	}

	// Bildirishnomalar (mentor + student)
	notifications := protected.Group("/notifications")
	{
		notifications.GET("", v1.ListNotifications(h))
		notifications.GET("/unread-count", v1.UnreadCount(h))
		notifications.POST("/:id/read", v1.MarkNotificationRead(h))
		notifications.POST("/read-all", v1.MarkAllNotificationsRead(h))
	}

	// Users
	users := protected.Group("/users")
	{
		users.GET("", v1.ListUsers(h))
		users.POST("", v1.CreateUser(h))
		users.GET("/me", v1.GetCurrentUser(h))
		users.PUT("/me", v1.UpdateCurrentUser(h))
		users.PUT("/me/password", v1.ChangeCurrentPassword(h))
		users.GET("/:id", v1.GetUser(h))
		users.PUT("/:id", v1.UpdateUser(h))
		users.DELETE("/:id", v1.DeleteUser(h))
		users.POST("/:id/deactivate", v1.DeactivateUser(h))
		users.POST("/:id/activate", v1.ActivateUser(h))
		users.PUT("/:id/password", v1.AdminResetPassword(h))
	}

	return r
}
