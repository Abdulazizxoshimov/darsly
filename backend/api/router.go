package api

import (
	"context"
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

// isProduction — muhit production'mi (`middleware.SecurityHeaders` va
// `websocket` bilan bir xil belgi ishlatiladi: APP_ENV).
func isProduction() bool { return os.Getenv("APP_ENV") == "production" }

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
		middleware.RequestHost(), // livekit "auto" client-URL uchun (usecase kontekstiga)
		middleware.Logger(log), // har so'rovni strukturaviy (request_id bilan) yozadi
		middleware.Recover(log),
		middleware.CORS(allowedOrigins...),
		middleware.SecurityHeaders(),
		metrics.Middleware(),
		// Javoblarni siqish (yomon internetda har bayt muhim). WS/metrics istisno.
		gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/api/v1/ws", "/metrics"})),
	)

	r.MaxMultipartMemory = 10 << 20 // 10 MB multipart

	// So'rov tanasi hajmi.
	//
	// Default 2 MB (JSON). ISTISNO — chatda fayl ulashish (№15): u multipart
	// va chegarasi usecase'da 20 MB (`chat.MaxChatFileBytes`). Bu yerdagi
	// qopqoq undan biroz katta bo'lishi SHART: multipart chegaralari va
	// qo'shimcha maydonlar ham tanaga kiradi, ya'ni aynan 20 MB qo'yilsa
	// to'liq hajmli fayl "hech qachon o'tmaydigan" bo'lib qolardi.
	//
	// Cheklov baribir ikki qatlamli: bu yerda tarmoq qatlami (o'qishni
	// to'xtatadi), usecase'da esa mahsulot qoidasi (aniq xato matni bilan).
	r.Use(func(c *gin.Context) {
		limit := int64(2 << 20)
		if strings.HasSuffix(c.Request.URL.Path, v1.ChatUploadPathSuffix) {
			limit = v1.ChatUploadBodyLimit
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	})

	// Swagger — FAQAT production'dan tashqarida.
	//
	// Production'da u butun API sxemasini (barcha endpointlar, so'rov/javob
	// shakllari, validatsiya qoidalari) autentifikatsiyasiz oshkor qiladi.
	// Bu hujum yuzasini bepul xaritalashtirib beradi va hech qanday foyda
	// keltirmaydi — sxema kerak bo'lsa dev muhitida ochiq.
	if !isProduction() {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	r.GET("/health", v1.HealthCheck())
	r.GET("/ready", v1.ReadyCheck(readyFn, log))

	// Metrikalar — production'da token bilan.
	//
	// `/metrics` ichki ma'lumot oqadi: endpoint nomlari, so'rovlar hajmi,
	// xatolar taqsimoti, Go runtime holati. Bu raqobat/razvedka ma'lumoti va
	// mavjud endpointlar ro'yxatini ham beradi. Prometheus esa `Authorization`
	// header'ini `bearer_token` bilan yubora oladi, ya'ni bu hech narsani
	// buzmaydi.
	//
	// METRICS_TOKEN bo'sh bo'lsa production'da endpoint UMUMAN yoqilmaydi:
	// "himoyasiz ochiq qolgani"dan "yo'q" holati xavfsizroq va nosozlik
	// darhol ko'rinadi (jimgina oqib turmaydi).
	switch {
	case !isProduction():
		r.GET("/metrics", gin.WrapH(metrics.Handler()))
	case os.Getenv("METRICS_TOKEN") != "":
		r.GET("/metrics", middleware.BearerToken(os.Getenv("METRICS_TOKEN")), gin.WrapH(metrics.Handler()))
	default:
		log.Warn(context.Background(),
			"router: METRICS_TOKEN o'rnatilmagan — /metrics production'da o'chirildi")
	}

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

	// LiveKit webhook (ochiq, imzo bilan himoyalangan) — Egress + xona hodisalari.
	//
	// ⚠️ CHEGARA KATTA VA BU ATAYLAB. Avval 30/s (burst 60) edi va bu jonli
	// muammo yasardi: LiveKit har ishtirokchi uchun bir necha hodisa yuboradi
	// (`participant_joined`, `track_published`, `participant_left`), ya'ni 150
	// kishilik dars boshlanishida bir necha yuz so'rov BIR NECHA SONIYADA keladi.
	// 2026-07-31 dagi yuklama sinovida aynan shu ko'rindi: 10 ta webhook 429
	// bilan RAD ETILDI. Oqibatlari jimgina va og'ir:
	//   · `track_published` yo'qolsa — dars UMUMAN yozib olinmaydi;
	//   · `participant_joined` yo'qolsa — ban qo'llanmaydi (chiqarilgan qaytadi);
	//   · `participant_left` yo'qolsa — bo'sh xona avto-yakuni ishlamaydi.
	// Manba BITTA va u imzo bilan tekshiriladi (`ParseWebhook`), shuning uchun
	// chegara faqat DoS to'sig'i sifatida qoladi — mahsulot yo'lini bo'g'masin.
	api.POST("/webhooks/livekit", middleware.RateLimit(200, 400), v1.LiveKitWebhook(h))

	// ── Xona holati (ochiq — LiveKit room-token bilan autentifikatsiya) ──────────
	// Guest'da JWT yo'q, lekin imzolangan room-token'i bor; handler token xonasining
	// dars bilan mosligini tekshiradi (`roomTokenIdentity`). So'rovnoma ovozida ham
	// AYNAN shu naqsh ishlatilgan.
	//
	// Rate-limit reaksiya uchun ataylab kengroq (emoji tez-tez bosiladi), lekin
	// ASOSIY cheklov usecase ichida — har ishtirokchi uchun 2 soniyada 1 marta.
	// Bu yerdagisi esa IP darajasidagi DoS himoyasi.
	rooms := api.Group("/rooms")
	{
		rooms.POST("/:lessonID/hand", middleware.RateLimit(20, 40), v1.SetHand(h))
		rooms.POST("/:lessonID/reaction", middleware.RateLimit(30, 60), v1.SendReaction(h))
		rooms.GET("/:lessonID/state", middleware.RateLimit(20, 40), v1.GetRoomState(h))
		// Chat — ISHTIROKCHI yo'li. Avval faqat host xabari saqlanardi; o'quvchi
		// yozgani hech qayerda qolmasdi (tarixda ham, yozuvda ham) va kech kirgan
		// hech nima ko'rmasdi. Asosiy cheklov usecase ichida (5 soniyada 5 xabar).
		rooms.POST("/:lessonID/chat", middleware.RateLimit(30, 60), v1.SendRoomChat(h))
		rooms.GET("/:lessonID/chat", middleware.RateLimit(20, 40), v1.RoomChatHistory(h))
		// Fayl ulashish (№15) — o'quvchi ham yubora oladi.
		//
		// IP bo'yicha cheklov chatdan qattiqroq (har yuklama 20 MB gacha),
		// lekin BUTUNLAY qattiq emas: bitta sinf ko'pincha bitta NAT ortida
		// bo'ladi va 2-3 o'quvchi bir vaqtda yuborsa hammasi bloklanardi.
		// Haqiqiy himoya — usecase'dagi IDENTITY bo'yicha cheklov (`uploadMax`,
		// daqiqasiga 5 ta), u NAT'dan ta'sirlanmaydi.
		rooms.POST("/:lessonID/chat/upload", middleware.RateLimit(5, 15), v1.UploadRoomChatFile(h))
	}

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
		// «Hammasini kiritish» — bitta amal, bitta so'rov (mobil/web tugmasi).
		// GET `/:id/waitingroom` bilan TO'QNASHMAYDI: gin har HTTP metodi uchun
		// alohida daraxt tutadi, bu POST.
		lessons.POST("/:id/waitingroom/admit-all", v1.AdmitAllWaitingRoom(h))
		// Host boshqaruvi
		lessons.GET("/:id/participants", v1.ListParticipants(h))
		lessons.POST("/:id/mute-all", v1.MuteAll(h))
		lessons.POST("/:id/participants/:identity/mute", v1.MuteParticipant(h))
		lessons.POST("/:id/participants/:identity/remove", v1.RemoveParticipant(h))
		lessons.POST("/:id/participants/:identity/allow-speak", v1.AllowSpeak(h))
		lessons.POST("/:id/participants/:identity/revoke-speak", v1.RevokeSpeak(h))
		// Qo'l ko'tarish — HOST tomonidagi boshqaruv (ko'tarish o'quvchining ishi,
		// u ochiq `/rooms/:lessonID/hand` orqali ketadi).
		//
		// `identity` yo'lda emas, TANADA: (a) gin daraxtida `/hands/:identity/lower`
		// va `/hands/lower-all` bir pozitsiyada param va statik segment sifatida
		// to'qnashadi; (b) guest identity'si ixtiyoriy satr bo'lishi mumkin va uni
		// URL'ga joylash kodlash muammolarini keltiradi.
		lessons.POST("/:id/hands/lower", v1.LowerHand(h))
		lessons.POST("/:id/hands/lower-all", v1.LowerAllHands(h))
		// Chat (host — persist + LiveKit broadcast; guest chat frontend data-channel orqali)
		lessons.GET("/:id/chat", v1.ChatHistory(h))
		lessons.POST("/:id/chat", v1.SendChat(h))
		// Fayl ulashish (№15). `/chat/upload` statik segment `/chat/:messageID`
		// bilan TO'QNASHMAYDI: gin har HTTP metodi uchun alohida daraxt tutadi,
		// bu esa POST, o'chirish esa DELETE.
		lessons.POST("/:id/chat/upload", v1.UploadChatFile(h))
		// Chat transkripti (№21) — TXT/HTML fayl. GET daraxtida `/chat` bilan
		// to'qnashmaydi: `/chat` va `/chat/transcript` turli chuqurlikda.
		lessons.GET("/:id/chat/transcript", v1.ChatTranscript(h))
		// Moderatsiya (№6) — xabarni o'chirish (faqat dars egasi).
		lessons.DELETE("/:id/chat/:messageID", v1.DeleteChatMessage(h))
		// Dars arxivi (№20) — video + chat + materiallar bitta javobda.
		lessons.GET("/:id/archive", v1.GetLessonArchive(h))
		// So'rovnomalar (host)
		lessons.GET("/:id/polls", v1.ListPolls(h))
		lessons.POST("/:id/polls", v1.CreatePoll(h))
		// Natijani e'lon qilish (№7)
		lessons.POST("/:id/polls/:pollID/publish", v1.PublishPollResults(h))
		// Yozib olish
		lessons.POST("/:id/recording/start", v1.StartRecording(h))
		lessons.GET("/:id/recordings", v1.ListRecordings(h))
		// Client-side (telefon) lokal yozuv — «Zoom local recording».
		lessons.POST("/:id/recording/local-start", v1.LocalStartRecording(h))
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
		// Bitta yozuv — tiklash holatini poll qilish uchun (archived →
		// restoring → ready). Ro'yxatni har 5 soniyada so'rash isrof bo'lardi.
		recordings.GET("/:id", v1.GetRecording(h))
		// Telegram arxividan qaytarib olish (202 + poll).
		recordings.POST("/:id/restore", v1.RestoreRecording(h))
		// Client-side lokal yozuv: telefon → MinIO to'g'ridan (presigned PUT), keyin finalize.
		recordings.POST("/:id/upload-url", v1.LocalUploadURL(h))
		recordings.POST("/:id/complete", v1.LocalCompleteRecording(h))
	}

	// Telegram bog'lanishi (mentor). `/me/...` ostida, chunki bu FOYDALANUVCHI
	// atributi — dars yoki yozuv emas.
	me := protected.Group("/me/telegram")
	{
		me.GET("", v1.TelegramStatus(h))
		me.POST("/link", v1.TelegramStartLink(h))
		me.DELETE("", v1.TelegramUnlink(h))
	}

	// Mentorning doimiy qora ro'yxati (kick scope=mentor yozuvlari)
	blocklist := protected.Group("/blocklist")
	{
		blocklist.GET("", v1.ListBlocklist(h))
		blocklist.DELETE("/:id", v1.Unblock(h))
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
		// M5 — o'z akkauntini o'chirish (Play Store majburiyati).
		users.DELETE("/me", v1.DeleteCurrentUser(h))
		users.GET("/:id", v1.GetUser(h))
		users.PUT("/:id", v1.UpdateUser(h))
		users.DELETE("/:id", v1.DeleteUser(h))
		users.POST("/:id/deactivate", v1.DeactivateUser(h))
		users.POST("/:id/activate", v1.ActivateUser(h))
		users.PUT("/:id/password", v1.AdminResetPassword(h))
	}

	return r
}
