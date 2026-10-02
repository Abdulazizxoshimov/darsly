# Jonly (Darsly) Backend — To'liq Audit Hisoboti

**Sana:** 2026-09-30
**Usul:** 7 ta AI sub-agent (Fable) backendni parallel, har biri handler→usecase→repo→DB/Redis bo'ylab boshidan oxirigacha tekshirdi.
**Qamrov:** ~25.6k satr, 88 endpoint, 14 domen, 32 migratsiya, 75 test fayl.
**Yo'nalishlar:** auth/user/RBAC · lesson/joinlink/room/recording · realtime(WS/chat/poll/notif) · DB layer · Redis/infra/config · security(butun backend) · system-design/arxitektura.

---

## ✅ Bajarilgan — System-design tuzatishlari (2026-09-30)

Quyidagi scale/barqarorlik ishlari amalga oshirildi (build/vet/test toza, 25 paket ✅):

**Faza A — bir hostni himoya (barqarorlik):**
- A1 Global egress cap `EGRESS_MAX_CONCURRENT` (default 3) — `recordingRepo.CountActive`, SFU'ni CPU och qolishdan himoya (R1).
- A2 Manual `StartRecording` double-start guard → 409 (topilma #4).
- A3 `MarkReady`/`MarkFailed` status-guard + bool qaytish → webhook replay/kech hodisa holatni buzmaydi; `EGRESS_LIMIT_REACHED` yaroqli fayl bo'lsa muvaffaqiyat (topilma #5).
- A4 Recording-start reconciliation `SweepAutoEnd`da — yo'qolgan `track_published` webhook o'rnini bosadi (R3).
- A5 `docker-compose.yml` resurs limitlari (egress 2cpu/2.5G, backend 2cpu/1.5G, PG tuning: shared_buffers/lock_timeout/idle_in_transaction).

**Faza B — gorizontal-scale tayyorlik:**
- B1 `APP_ROLE=all|api|worker` — jarayonni ajratish; compose'da `backend-worker` (profile `worker`) — ffmpeg yukini API'dan ajratadi (R5).
- B2 Dars-darajali ban PG'ga write-through (`lesson_bans`, migr. 000033) — Redis o'chsa ham kick amalda qoladi; asosiy gate PG-fallback (R2). Durability testi ✅.
- B3 Backend Redis'i `REDIS_DB=1` (deploy) — LiveKit db0'dan ajratildi (M11).
- B4 `Notifier` port — usecase'dan konkret `*ws.Hub` uzildi (R7).
- B5 Redis `Incr` atomik Lua — doimiy lockout yo'q (topilma #6).

**Faza C — skelet (bitta host, YOQILMAGAN):**
- C1 Redis Sentinel failover-klient qo'llab-quvvatlashi (`REDIS_SENTINEL_ADDRS` berilганda faollashadi) + compose skeleti.
- C2 Multi-node LiveKit + egress-alohida-host izohlari (compose).
- C3 PgBouncer compose skeleti (izohli).

## ✅ Bajarilgan — Qolgan BARCHA topilmalar (2026-09-30, ikkinchi bosqich)

Auditda topilgan qolgan topilmalar 5 to'lqinda tuzatildi. Yakuniy holat: **`go build ./...` toza, `go vet` toza, 29 test paketi ✅ 0 FAIL, gofmt toza** (Postgres+Redis bilan integration/apitest'lar ham o'tdi).

**Lesson/Room/Recording/Joinlink (16):** lesson lost-update → partial `UpdateFields`+`LessonPatch`; `ClaimStart` atomik; `PATCH` dan `status` olib tashlandi; `EndLesson`→`ClaimEnd` (idempotent, `ended_at` saqlanadi); `LocalStart` sentinel yo'q; `LocalUploadURL/Complete` status+`local:` precondition+clamp; `live` dars delete 409; local yozuvda egress skip/400 + stale reaper (`FailStale`+sweep); host token'dan `RoomRecord` olib tashlandi; `randIndex` crypto/rand; scheduled dars → `waiting_for_host` (token yo'q); `shared.RoomName`; presigned size cheklovi; failed egress obyekti o'chadi; `ended_at` parse xato→400; `recordings.Create` unique→409.

**DB (8):** partial email UNIQUE (000034); o'lik/duplikat indekslar olib tashlandi + `(mentor_id,created_at)` kompozit + retention COALESCE indeksi (000035); yetishmayotgan CHECK'lar (000036); pool `lock_timeout`/`idle_in_transaction_session_timeout`/`application_name`; cheksiz jadvallar uchun **janitor worker** (refresh/reset/waitingroom/notifications).

**Auth/User (14):** user soft-delete → mentor darslari `cancelled` (`CancelByMentor`); logout har doim sessiyani bekor qiladi; account-lockout (email+IP); reset-TTL 1h + eskilar invalidatsiyasi; jwt Rotate sessiyani tiriltirmaydi (Lua `EXISTS`); bcrypt dummy cost startup'da; deaktiv-akkaunt enumeratsiya yopildi; `last_login_at` yoziladi; `SoftDelete` 0-qator→404; oxirgi-admin himoyasi; `avatar_url` http(s)-only validator; forgot-password per-email throttle; `?token=` faqat WS; logout `sub` tekshiruvi.

**Realtime (15):** o'lik `subscribe`/`BroadcastToRoom` olib tashlandi (+per-user WS cap 5); `admitOne` token-avval; ghost-pending (status+LIMIT); pending cap 200 (429); chat caption/`to` validatsiyasi; `notification.MarkRead` ValidateID; poll vote-after-close shartli upsert; `roomstate.State`/`poll.Results` ban tekshiruvi; Admit lesson-live; UTF-8 rune kesish; reminder retry + **`UnclaimReminder`** (xatoda qayta uriniladi); LiveKit-off→503; `VoteReq.Token` 401; WS snapshot `onRegistered` callback.

**Infra/Observability (22):** audit→Loki (Warn); email DLQ/attempt + poison-loop to'xtatildi; `ReadHeaderTimeout` (upload o'lmaydi); prod fail-fast (LiveKit/MinIO) + `/ready`ga MinIO; Sentry query redaksiyasi; log'da `FullPath`; panic Sentry'ga stack bilan; Loki async (bloklamaydi); prod JSON loglar; config parse xato log; RabbitMQ publisher confirm + fallback; `Hub.Stop` `WriteControl`; Casbin fail-fast; SMTP timeout; `mentorname` cache invalidatsiya; `X-Request-ID` validatsiya; Sentry `RequestID`dan keyin + ready-503 chiqarildi; CORS `Vary`; `/ready` ~2s cache; o'lik MQ kodi olib tashlandi; config secret'lar `json:"-"`; Redis client Close.

> Barcha topilmalar yopildi. Ochiq qolgan katta strukturaviy ish faqat: to'liq `WithTx` refaktori (M6 — hozir kompensatsiya bilan qoplangan, audit "acceptable" dedi) va keyset-pagination (INFO). Frontend/mobil e'tibor: joinlink `next_step:"waiting_for_host"` va logout bo'sh-tana xulqi (W2/W1 hisobotlarida).

---

## Umumiy verdikt

**Yetuklik: "kuchli pilot / erta-production".** Go arxitekturasi kutilganidan yaxshi — rewrite kerak emas.
**CRITICAL yo'q, haqiqiy security HIGH yo'q.** Eng katta risklar kodda emas — infratuzilmada va bir nechta yozuv-modeli poyga (race) holatlarida. Kod u ishlab turgan infratuzilmadan oldinda.

### Nima yaxshi qilingan
- Dependency rule buzilmagan (entity pok, handlerda biznes-logika yo'q), entity→infra sızıntısı yo'q.
- Atomik holat o'tishlari izchil: `ClaimEnd`, `ClaimReminder`, `TransitionFromPending`, `FOR UPDATE SKIP LOCKED` navbatlar → workerlar multi-instance-safe.
- JWT: HS256 alg-pinning, ≥32 belgi secret, atomik Lua refresh rotation + reuse detection. Eski ma'lum HIGH (reset/deactivate Redis sessiyani o'chirmasligi) **tuzatilgan va tasdiqlangan**.
- IDOR himoyasi izchil: har lesson/recording/poll/chat/blocklist/notification yo'lida ownership tekshiruvi. RBAC 63 protected route'ning hammasini qamraydi.
- SQL 100% parametrlangan (Squirrel), ORDER BY allow-list, LIKE escape. Injection topilmadi.
- WS hub Redis fan-out bilan gorizontal-scale poydevoriga ega; graceful shutdown to'g'ri tartibda.

---

## 🔴 TOP muammolar (bir nechta agent mustaqil tasdiqlagan — ishonch yuqori)

### 1. `lessons` jadvalida lost-update poyga (to'liq-qator overwrite) — HIGH
`internal/infrastructure/repository/postgres/lesson.go:170-195` `Update` 14 ustunni in-memory entity'dan yozadi.
Chaqiruvchilar: `internal/usecase/room/room.go:172` (HostToken→live), `room.go:265` (EndLesson→ended), `room.go:537` (MuteAll), `internal/usecase/lesson/lesson.go:88` (PATCH).
**Ta'sir:** mentor webda tahrirlayotganda mobil HostToken `live` qiladi → PUT `status='scheduled', started_at=NULL` qaytarib yozadi. Dars LiveKit'da jonli, DB'da emas → `ClaimEnd` ishlamaydi, auto-end/recording/teardown ishlamaydi.
**Tuzatish:** `Update` ni partial qiling (faqat `UpdateLessonReq` maydonlari); holat o'tishlarini `ClaimStart` (mavjud `ClaimEnd` kabi) orqali; ixtiyoriy `WHERE updated_at=$expected` optimistic lock.

### 2. `PATCH /lessons/:id` ixtiyoriy `status` qabul qiladi — HIGH
`internal/usecase/lesson/lesson.go:124-126` — egasi `ended` ni teardown'siz (LiveKit xona ochiq, egress ishlaydi), `live` ni HostToken'siz, yoki `ended` darsni `scheduled` ga "tiriltirib" eski join-link'ni qayta ochishi mumkin.
**Tuzatish:** `Status` ni `UpdateLessonReq` (entity/lesson.go) dan olib tashlang (yoki faqat `scheduled→cancelled`); lifecycle'ni `HostToken`/`EndLesson` orqali.

### 3. Foydalanuvchi soft-delete tarqalmaydi + email UNIQUE partial emas — HIGH
`internal/usecase/user/user.go:288` faqat `users.deleted_at`; `GetBySlug`/join oqimi mentor `deleted_at`/`is_active` ni tekshirmaydi → o'chirilgan mentorning join-linklari ishlaydi.
`migrations/000002_users_and_auth.up.sql:22` `email CITEXT UNIQUE` partial emas → soft-delete'dan keyin o'sha email hech qachon qayta ro'yxatdan o'tolmaydi (Play-Store compliance oqimini buzadi; `ExistsByEmail` "bo'sh" deydi, INSERT esa 23505 beradi).
**Tuzatish:** `user.Delete` da bitta tx'da `UPDATE lessons SET deleted_at=NOW(), status='cancelled' WHERE mentor_id=$1 AND deleted_at IS NULL`; inline UNIQUE ni `CREATE UNIQUE INDEX users_email_active ON users(email) WHERE deleted_at IS NULL` bilan (2-bosqichli migratsiya) almashtiring, `idx_users_email` ni tashlang.

### 4. Manual `StartRecording` da active-check/lock yo'q — HIGH
`internal/usecase/recording/recording.go:243-264` — avto-yo'l `EnsureRecording` (`:151-177`) `activeFor`+Redis lock tekshiradi, manual endpoint to'g'ridan-to'g'ri `startEgress`. Ikki marta bosish → 2× parallel egress (4-core boxda 2× CPU, 2 fayl).
**Tuzatish:** `StartRecording` da ham `activeFor`+`startLockKey` guardini qayta ishlating, active bo'lsa 409.

### 5. Egress webhook holat-guardsiz → replay/kech yetkazish holatni buzadi — MEDIUM (3 agent)
`MarkReady`/`MarkFailed` (`recording.go:248,540`; `postgres/recording.go:242`) `WHERE egress_id=$1` faqat.
- `stop→webhook` poygasi `ready`→`processing` ga qaytaradi (`StopRecording` shartsiz `UpdateStatus(processing)`).
- Kech `egress_ended` `archived`→`ready` qiladi (obyekt o'chirilgan → download 404); `content_offset_sec` ikki marta jamlanadi.
- `EGRESS_LIMIT_REACHED` yaroqli fayl bo'lsa ham `failed` (`api/handlers/v1/webhook.go:98`) → yetim MP4 abadiy MinIO'da.
**Tuzatish:** `WHERE egress_id=$1 AND status IN ('recording','processing')` + `RowsAffected==0 → log & nil`; `LIMIT_REACHED` ni `len(FileResults)>0 && FileResults[0].Size>0` bo'lsa muvaffaqiyat deb hisoblang; `EnqueueTranscode ... AND transcode_status='skipped'`.

### 6. Redis `INCR`+`EXPIRE` atomik emas → doimiy lockout — MEDIUM/HIGH (3 agent)
`internal/infrastructure/redis/cache.go:57-66` — `INCR` keyin faqat `n==1` da `EXPIRE`, xato ham e'tiborsiz. Orada crash/timeout → `rl:login:*`, `loginfail:*`, `joinfail:*`, `rl:chat:*` kalitlar TTL'siz → foydalanuvchi/IP doimiy 429/qulf (`redis-cli DEL` gacha).
**Tuzatish:** bitta Lua script (`INCR`; `if TTL==-1 then PEXPIRE`) yoki `SET NX EX`+`INCR` pipeline.

### 7. Logout `refresh_token`siz jimgina no-op — MEDIUM (2 agent)
`api/handlers/v1/auth.go:104` `_ = c.ShouldBindJSON` xatoni tashlaydi; `internal/usecase/auth/auth.go:299-313` faqat `RevokeRefresh(req.RefreshToken)`; `jwt.go:434` bo'sh tokenda nil qaytaradi → klient 204 oladi, access+sessiya TTL'gacha tirik. `middleware.CtxSessionID` o'rnatilgan-u iste'molchisi yo'q.
**Tuzatish:** `Logout` da doim `tokens.RevokeSession(ctx, c.GetString(CtxSessionID))`, refresh berilsa JTI ni ham.

### 8. `?token=` har protected route'da + Sentry'ga JWT sızıntısı — MEDIUM (3 agent)
`api/middleware/auth.go:32` query-token butun `protected` guruhga (faqat WS emas); `api/middleware/sentry.go:17` `SetRequest` raw query string'ni (jonli JWT / guest capability) 5xx/panic'da Sentry'ga yuboradi. App log redaksiya qiladi, Sentry/Caddy proxy log yo'q.
**Tuzatish:** query-token faqat `c.FullPath()=="/api/v1/ws"`; Sentry `BeforeSend` da `token`/`request_id` ni tozalang; log'da `c.FullPath()` (route shabloni) + hashlangan parametrlar.

### 9. Kick/ban re-join orqali aylanib o'tiladi — HIGH (room) / MEDIUM
Ban identity (`shared.BanKey(lessonID, identity)`) bo'yicha, lekin har `POST /joinlink/:slug` (`joinlink.go:168`) yangi `guest_<uuid>` beradi → chiqarilgan mehmon sahifani qayta yuklaydi. Bir xil teshik poll ballot-stuffing'ga imkon (`postgres/poll.go:117` identity bo'yicha dedup).
**Tuzatish:** client-bound kalit (IP hash / imzolangan device cookie) bilan ham ban, `joinlink.Join` da tekshiring; yoki kick'da darsni auto-lock.

---

## Xavfsizlik (qolgan topilmalar)

- **M — Account-lockout DoS:** `auth.go:34,213,392` — 5 xato parol istalgan email'ni 15 daq qulflaydi (IP'dan mustaqil; darsdan oldin mentorni bloklash mumkin). Fix: qulfni (email+IP) ga bog'lang, global email-hisoblagichni faqat soft-signal.
- **M — Reset-token TTL 24h:** `internal/usecase/usecase.go:99` — token kuchli (UUIDv4, SHA-256, single-use) lekin GET URL'da referrer/history/mail-scanner'ga bir kun ochiq. Fix: 15–60 daq + yangi so'rovda eski tokenlarni invalidatsiya.
- **L — bcrypt timing oracle:** `auth.go:383` dummy-hash cost 12 vs `config.go:299` default 11 → noma'lum email ~2× sekin (enumeratsiya). Fix: startup'da configlangan cost bilan dummy hash.
- **L — Deaktiv akkaunt enumeratsiyasi:** `auth.go:209-212` parol tekshiruvidan oldin 403 "deactivated". Fix: avval parol, keyin generic 401.
- **L — `avatar_url` `javascript:` qabul qiladi:** `entity/user.go` `validate:"omitempty,url"` (go-playground har schemeni qabul). Fix: custom http/https validator; frontend href'da ishlatmasin.
- **L — Participant `CanPublishData:true`:** `livekit/token.go:45` — backend rate-limit/emoji allow-list'ni chetlab o'tib data-channel'ga to'g'ridan-to'g'ri `{kind:"reaction"|"chat"}` yozish mumkin. Fix: klientlar non-host'dan server-owned kind'larni e'tiborsiz qoldirsin, yoki participant'ga `CanPublishData:false`.
- **L — Forgot-password per-email throttle yo'q:** `router.go:192` faqat 60/min/IP → email-bombing. Fix: `ForgotPassword` ichida normallashtirilgan email bo'yicha Redis Incr.

---

## Ma'lumotlar bazasi

- **M — 4 jadval cheksiz o'sadi:** `DeleteExpiredTokens`/`DeleteExpiredPasswordResets` (`postgres/auth.go:108,173`) chaqiruvchisiz; `waiting_room_requests`/`notifications` da cleanup umuman yo'q. Fix: kechalik janitor worker (batched `LIMIT 5000`).
- **M — Retention indeksi query bilan mos emas:** `migrations/000016` `idx_recordings_retention(ended_at) WHERE status='ready'` vs query `COALESCE(ended_at, created_at)` (`recording.go:106`). Fix: `((COALESCE(ended_at,created_at))) WHERE status='ready'`.
- **M — `EndLesson` atomik `ClaimEnd` ni chetlab o'tadi:** `room.go:255-268` to'liq `Update` + status-guard yo'q → worker vs "Yakunlash" tugmasi ikkalasi teardown. Fix: `ClaimEnd`, `!claimed` da 409/idempotent.
- **M — `WithTx` faqat User+Auth:** `storage.go:82-92`; qolgan 8 repo `*pgxpool.Pool` ushlaydi → claim+insert atomik emas (reminder/retention-warning yo'qolishi mumkin). Fix: repolarni `pg.Querier` ga o'tkazing.
- **M — Pool sozlamalari:** 4-core VPS'da `DB_MAX_CONNS=50` (oshirilgan); `idle_in_transaction_session_timeout`/`lock_timeout`/`application_name` o'rnatilmagan. Fix: 20 conn + runtime params.
- **L — Duplikat/o'lik indekslar:** `idx_users_email` (`users_email_key` bilan takror), `idx_users_is_active`, `idx_wrr_created_at`, `idx_chat_messages_dm` (predikat hech qachon bajarilmaydi). Fix: tashlang.
- **L — Yetishmayotgan kompozit indeks:** `ListByMentor` (`lesson.go:117`) `(mentor_id, created_at DESC) WHERE deleted_at IS NULL` kerak.
- **L — Yetishmayotgan CHECK:** `recordings.transcode_status`, `poll_votes.option_index >= 0`, `lessons.duration_min > 0`, `telegram_chats.type`.
- **L — Vote-after-close race:** `poll.go:98-118` Go'da `is_active` o'qiydi keyin shartsiz INSERT. Fix: `INSERT ... SELECT ... WHERE EXISTS(...is_active)` + RowsAffected.

### Yetishmayotgan / mos kelmagan indekslar
| Jadval | Tavsiya | Nima uchun |
|---|---|---|
| `lessons` | `(mentor_id, created_at DESC) WHERE deleted_at IS NULL` | `ListByMentor` default page |
| `recordings` | `((COALESCE(ended_at, created_at))) WHERE status='ready'` | 4 retention query |
| `users` | `UNIQUE(email) WHERE deleted_at IS NULL` (table-UNIQUE o'rniga) | soft-delete email reuse |
| Tashlash | `idx_users_email`, `idx_users_is_active`, `idx_wrr_created_at`, `idx_chat_messages_dm` | o'lik/duplikat |

---

## Infra / Observability

- **H — Audit loglar Loki'ga yetib bormaydi:** `logger.go:69` faqat Warn+, `audit.go:22` `log.Info`. "Kim nima qildi" izi faqat container stdout'da. Fix: audit'ni Warn'da yoki Info-Loki core'i `event=audit` filtri bilan.
- **H — Email navbati poison-message loop / DLQ yo'q:** `worker/email.go:68` SMTP xatosida abadiy `Nack(requeue)`, `rabbitmq.go:85` da `x-dead-letter-exchange` yo'q, `Attempt` oshmaydi. Hozir `EMAIL_ENABLED=false` bilan yashiringan. Fix: attempt++, backoff bilan republish, N dan keyin DLQ.
- **H — `ReadTimeout:15s` sekin internetda upload'ni o'ldiradi:** `api/server.go:18` — chat fayl upload server-proxied 21MB (`router.go:105`). "Yomon internet" targetiga zid. Fix: `ReadHeaderTimeout` + per-route deadline, yoki chat fayllarni presigned PUT'ga.
- **H — Prod yetishmayotgan bog'liqliklar bilan jimgina ishlaydi:** `config.Validate()` prod'da LiveKit kalitlarini talab qilmaydi (`app.go:158` faqat warn), MinIO xatosi `NewNop()` ga tushadi, `/ready` faqat PG+Redis. → "sog'lom" deploy dars o'tkazolmaydi. Fix: prod'da LiveKit/MinIO'da fail-fast; MinIO'ni readiness'ga.
- **M — Panic'lar Sentry'ga stack'siz:** middleware tartibi `Sentry→…→Logger→Recover` (`router.go:79`); `Recover` `c.Error(...)` chaqirmaydi → Sentry bo'sh `CaptureMessage`. Fix: `Recover` da `c.Error(fmt.Errorf("panic: %v", r))`.
- **M — Loki transport logging goroutine'ni bloklaydi:** `loki/core.go:69` `add()` sync `flush()` (5s HTTP). Xato bo'ronida handlerlar 5s stall. Fix: buffered channel + alohida sender goroutine.
- **M — Prod loglar ANSI rangli console-format:** `logger.go:165` `NewConsoleEncoder`+`CapitalColorLevelEncoder`. JSON pipeline buziladi. Fix: prod'da JSON encoder.
- **M — `mentorname:` cache hech qachon invalidatsiya bo'lmaydi:** `joinlink.go:236` 1h cache, hech joyda `Del`. Fix: `user.UpdateCurrentUser`/admin update'da `Del`.
- **M — Redis keyspace namespace'siz + PII + DB 0 LiveKit bilan umumiy:** `loginfail:<email>` plaintext email. Fix: `darsly:` prefix yoki `REDIS_DB=1`, email'ni hash.
- **L:** Redis client Close qilinmaydi; `X-Request-ID` klientdan cheksiz uzunlik (log injection); ~10 joyda `os.Getenv` `config.Load` ni chetlab o'tadi; `TRUSTED_PROXIES` example butun RFC1918'ga ishonadi.

---

## Realtime / WebSocket

- **H — `subscribe` cheksiz + `BroadcastToRoom` o'lik kod:** `websocket.go:376-385` istalgan klientdan (autentifikatsiyasiz guest ham) istalgan room'ga `subscribe` qabul qiladi, cap yo'q; `BroadcastToRoom` faqat testlar chaqiradi → xotira DoS. Fix: `subscribe/unsubscribe` handling'ni olib tashlang.
- **H — `admitOne` token'dan oldin `admitted` qiladi:** `waitingroom.go:170-181` — `room.ParticipantToken` fail bo'lsa (LiveKit off, guest banned) so'rov abadiy tokensiz `admitted`, recovery yo'q. Fix: avval token, keyin `TransitionFromPending`.
- **M — Ghost pending push:** `ListPendingByMentor` faqat `l.deleted_at IS NULL` (status filtri/LIMIT yo'q) → har mentor WS reconnect'ida o'tgan darslardan ghost `waiting_room.request` push, 256-slot bufer to'lib real event'lar tushib qoladi. Fix: `l.status='live'` + LIMIT; dars-end'da pending'larni `rejected`.
- **M — Validatsiya/guard yetishmovchiliklari:** `notification.MarkRead` da `ValidateID` yo'q (500); chat upload caption uzunlik cheksiz (20MB); `entity/chat.go` `To` da `max=128` yo'q (500); `roomstate.State`/`poll.Results` da `GuardRoomAction` yo'q (kick'langan ishtirokchi o'qiy oladi).
- **M — Pending so'rovlar cap yo'q:** har join-link POST (600/min/IP) mentor UI'ga event push + DB row. Fix: per-lesson cap.
- **L:** `reminder.go:50` claim-before-send (Notify fail → reminder abadiy yo'q); chat `file.go:111` multibyte rune kesish (invalid UTF-8 → 500); global WS cap non-atomik + per-user cap yo'q.

### Tasdiqlangan OK (realtime)
Waitingroom race (admit token 15 daq cache, kech guest xizmat, double-admit→409); `ClaimReminder` distributed-safe; Redis double-encode yo'q; chat DM SQL'da izolyatsiya; fayl upload extension whitelist+content-sniff+`filepath.Base`; poll bir vote/identity.

---

## System Design — strukturaviy risklar

| # | Risk | Ta'sir | Tavsiya |
|---|---|---|---|
| R1 | **Yagona host** — SFU+egress(Chrome)+ffmpeg+PG+Redis+MinIO+Caddy 4core/7GB'da; egress'da global cap yo'q | 2–3 bir vaqtli yozuvli dars SFU'ni och qoldiradi → barcha xonalarda jitter | Global egress semaphore + compose resurs limitlari; egress'ni alohida hostga |
| R2 | **Redis compound SPOF** — sessiya, ban, live-state, WS fan-out, (recording'da) LiveKit ham | Redis o'chsa: video + yangi login (5daq'da) o'chadi, kick'langanlar qayta kiradi (ban PG'da emas) | Ban'ni PG'ga persist; redis_exporter+alert; Sentinel (2-host bo'lganda) |
| R3 | **Webhook control-plane at-least-once** — `track_published` yo'qolsa dars jimgina yozilmaydi (reconciliation yo'q) | Recording start uchun fallback yo'q | `SweepAutoEnd` ga recording-reconciliation; webhook'ni `event.Id` bo'yicha dedup |
| R4 | **Durability** — PG restore hech sinalmagan, MinIO backup umuman yo'q, RPO 24h | Ma'lumot yo'qotish | Restore testini ishga tushirish; dumps off-host; MinIO replication yoki majburiy Telegram archive |
| R5 | 8 in-process worker har API instance ichida (ffmpeg, 1.9GB upload) | Multi-instance'da takror scan, LiveKit ListParticipants har daqiqa | `cmd/worker` split (`WORKERS_ENABLED` flag) |
| R6 | Tranzaksiya faqat bitta oqimda; qolgani kompensatsiya | Stuck `recording`/`processing`/`restoring` qatorlar | Reconciliation/invariant checker worker |
| R7 | Arxitektura eroziyasi | maintainability | `*ws.Hub` o'rniga `Notifier` port; `livekit.S3Config`/`tg.File` ni usecase interface'dan; guest token verify'ni handlerdan middleware'ga |
| R8 | Observability yarim-yoqilgan | Prod incident debug qiyin | node/postgres/redis exporter + 5 alert (disk/PG/Redis/pool/WS-drop) |
| R9 | TURN over TLS/443 yoqilmagan | UDP-blocklangan tarmoqlarda faqat 7881/tcp fallback | `jonly.uz` bo'lgach `turn.jonly.uz` |

### Scalability shift
Cheklov backend emas — **media**. Joriy hostda **~60–100 ishtirokchi/xona, 2–3 bir vaqtli yozuvli dars**. Birinchi qulaydigan joy: 100 talabali mentor + ikkinchi yozuvli dars bir hostda. Ikkinchi: kun o'rtasida Redis restart. Go kodi bu ro'yxatda yo'q.

---

## Tavsiya etilgan yo'l xaritasi

### P0 — pilotdan oldin (kunlar)
1. `#1` lesson partial Update + `ClaimStart`, `#2` PATCH status olib tashlash (bir PR, migratsiyasiz).
2. `#4` recording double-start guard + `#5` webhook status-guard/replay-dedupe.
3. `#6` Redis INCR+EXPIRE atomik (Lua) + `#7` logout to'g'rilash.
4. `#8` `?token=` faqat WS + Sentry redaksiya.
5. Backup-restore testini serverda ishga tushirish, dumps off-host (R4); global egress cap + compose resurs limitlari (R1).
6. Exporterlar + 5 alert (R8); audit→Loki (H1).

### P1 — public launch'dan oldin (haftalar)
7. `#3` user soft-delete propagatsiya + partial email UNIQUE.
8. `#9` ban'ni client-bound + PG'ga persist (R2); recording-start reconciliation (R3).
9. `cmd/worker` process split (R5); email DLQ+attempt (H2); ReadTimeout upload fix (H4).
10. DB janitor worker, retention indeks, `(mentor_id, created_at)` indeks; `WithTx` ni qolgan repolarga.
11. Chaos + webhook-replay testlari; SFU media load (`sfu_media_load -n 100`) natijasini yozib qo'yish.

### P2 — scale (2+ host bo'lganda)
12. Redis Sentinel; PG WAL archiving + replica; PgBouncer; multi-node LiveKit; MinIO off-host replication.
13. O'lik JiraFlow kodini o'chirish (`BroadcastToRoom`, ishlatilmagan MQ navbatlari, obsolete `deployments/docker-compose.yml`); doc drift (Caddyfile eski IP, DEPLOYMENT.md domenlari).
14. `room` usecase dekompozitsiyasi (~2.5k LOC'dan oshganda).

---

## Xulosa

Go arxitekturasi sog'lom holatda va qayta yozish talab qilmaydi. Keyingi haftalar quyidagilarga sarflanishi kerak:
1. **Capacity isolation** — egress/worker'ni API host'idan ajratish.
2. **Durability** — backup/restore testi, off-host nusxalar.
3. Ikki "jim" nosozlikni yopish — **yozilmagan dars** (R3) va **Redis'dan yo'qolgan xavfsizlik holati** (R2).

Va P0 dagi yozuv-modeli poyga (#1, #2) hamda recording edge-buglar (#4, #5) — bular kam kod bilan katta correctness yutug'i beradi.
