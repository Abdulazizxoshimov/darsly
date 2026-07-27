# Darsly — Optimallashtirish Rejasi
## Xavfsizlik · Professionallik · Latency (yomon internet)

**Holat:** 2026-07-23 · MVP tugagan, senior code-review tuzatishlari kiritilgan.
**Maqsad:** Ishlaydigan MVP → **production-grade** eng optimal holat.

**Ustuvorlik belgilari:** `P0` = productionni bloklaydi (majburiy) · `P1` = sifat/tezlik (kerak) · `P2` = masshtab/ilg'or (keyin).

**Har vazifa formati:** *nima → nega → "bajarildi" mezoni (acceptance)*.

---

## FAZA 0 — Production blokerlari (P0) · ~1–1.5 hafta

> Bularsiz productionga chiqib bo'lmaydi. Ketma-ketlik shu.

### 0.1 Testlar — ✅ BAJARILDI

**Bajarildi (2026-07-23):** `internal/testutil` in-memory faqe'lar (barcha repo + token/cache/minio/room, kompilyatsiya-vaqti interfeys assertion bilan); **37 unit test** — auth (register/login/logout/orphan-cleanup/timing/reset/refresh), lesson (slug/ownership/passcode), waitingroom (atomik admit/reject, TOCTOU), notification, recording (webhook mapping/download), joinlink (passcode/waiting-room/locked), token (rotate reuse-detection, revoke — real Redis), HTTP-qatlam (validatsiya wiring + privilege-escalation). Usecase coverage: joinlink 92% · notification 75% · token 71% · lesson/waitingroom 70% · auth 53% · recording 34%. `make test` = `go test -race -coverprofile`. CI: `.github/workflows/ci.yml` (pg+redis servis, build/vet/lint/test).

**Integration + e2e (2026-07-23):** `internal/testutil/pg.go` (`SetupTestDB` — darsly_test yaratadi, migratsiya, truncate; PG yo'q bo'lsa skip). Repo integration testlari (`.../postgres/integration_test.go`): unique email + DeleteHard, **ClaimReminder atomik**, **TransitionFromPending 20-goroutine konkurentlik ostida faqat 1 yutadi (TOCTOU SQL-darajada isbotlangan)**, recording lifecycle, notification unread. E2e (`internal/apitests/e2e_test.go`): butun wired stack — register+validatsiya→me→RBAC(student 403)→mentor promote→lesson→joinlink preview+waiting-room. **Jami 43 test.**

**Original reja (quyida):**
- **Unit testlar** (usecase qatlami): auth (register/login/logout/refresh, timing), lesson (slug collision, ownership), waitingroom (holat mashinasi, atomik transition), recording (webhook mapping), token (rotate/revoke), validator.
  - *Nega:* regressiyani ushlaydi; tuzatilgan buglar qaytmasligini kafolatlaydi.
  - *Mezon:* usecase qatlamida ≥70% coverage; `go test ./... -race` yashil.
- **Integration testlar** (`internal/apitests` yoki testcontainers/dockertest): real Postgres+Redis bilan asosiy oqimlar (auth→lesson→joinlink→admit; webhook→ready→download).
  - *Mezon:* CI'da konteynerlar bilan o'tadi; migratsiyalar up/down sinaladi.
- **k6 API load** allaqachon bor (`tests/load/`) — CI'ga smoke sifatida ulash.
  - *Mezon:* p95 < 800ms, xato < 1% chegaralari CI'da.

### 0.2 Xavfsizlik yakuni — ✅ BAJARILDI (2026-07-23)
- ✅ **ResetPassword sessiyalarni bekor qiladi** — `authRepo.RevokeAllUserTokens` (refresh vektori o'ladi; access ≤15m eskiradi). Test bor.
- ✅ **Passcode brute-force lockout** — `redis.Cache.Incr` bilan slug bo'yicha hisoblagich; 5 noto'g'ri urinishdan keyin 5 daqiqa qulf. Test bor (`TestJoin_PasscodeBruteForceLockout`).
- ✅ **WS/webhook DoS** — `/ws`(10/20), `/ws/waitingroom`(10/20), `/webhooks/livekit`(30/60), `/waitingroom/:id/status`(20/40) rate-limit; Hub global ulanish cap (`WS_MAX_CONNECTIONS`, def 10000); prod'da Origin majburiy (`APP_ENV=production` + `FRONTEND_BASE_URL` bo'sh → rad).
- ✅ **TrustedProxies** — `r.SetTrustedProxies(trustedProxies())`, `TRUSTED_PROXIES` env (bo'sh → hech kimga ishonmaydi).
- ✅ **Config prod validatsiyasi** — `config.Validate()` production'da dev/default JWT & LiveKit secret'ni va bo'sh FRONTEND_BASE_URL'ni rad etadi. Testlar bor. (Secret-manager integratsiyasi — deployment bosqichi.)
- ✅ **Audit log** — `internal/pkg/audit` strukturaviy `event=audit` log; waitingroom admit/reject, recording start/stop'ga ulangan (actor + target). *(Qoladi: user deactivate/delete uchun actor-threading; DB audit jadvali.)*

### 0.3 Transport TLS/TURN (latency P0 — yomon internet uchun MAJBURIY)
- **`wss://` signaling:** reverse-proxy (Caddy/nginx) + TLS, LIVEKIT_HOST=`wss://`.
  - *Nega:* HTTPS frontend/mobil-proksi shifrsiz WS'ni uzadi.
  - *Mezon:* signaling 443/wss ustidan; mixed-content yo'q.
- **TURN over TLS (443):** `livekit.yaml` `turn.tls_port: 5349` + domen + sert (yoki 443 relay); `use_external_ip: true`/`node_ip`.
  - *Nega:* simmetrik NAT / korporativ firewall ortida media faqat 443-TURN orqali o'tadi.
  - *Mezon:* NAT ortidagi klient ulanadi (real muhitda test).
- **Presigned URL ochiq-yuzli endpoint:** yozuv yuklab olish uchun public S3/MinIO domeni (+TLS), ichki `MINIO_ENDPOINT` emas.
  - *Mezon:* tashqi tarmoqdan yozuv yuklab olinadi.

---

## FAZA 1 — Professionallik (P1) · ~1 hafta

- **CI/CD** (`.github/workflows`): build → `golangci-lint` (qat'iy) → `go test -race` → `govulncheck` → docker build → (opsiyonal) deploy.
  - *Mezon:* har PR'da yashil pipeline; lint xatolari bloklaydi.
- **Ma'lumot yaxlitligi — `WithTx`:** mavjud `internal/pkg/postgres/transaction.go`ni ko'p-yozuvli oqimlarga ulash (register user+token; kelajakdagi enrollment). Repolarni `Querier` (pool|tx) qabul qiladigan qilib refactor.
  - *Mezon:* register bitta tranzaksiyada; kompensatsiya o'rniga rollback.
- **Config qattiqlashtirish:** casbin `model.conf`/`policy.csv` → `embed`; default user maydonlari (`#6366F1`,`student`,`UTC`,`uz`) → konstanta; magic numberlar (bufer 256, ping/deadline, EmptyTimeout) → nomli konstanta; `config.Validate()` kengaytirish (LiveKit prod'da majburiy).
  - *Mezon:* CWD'ga bog'liqlik yo'q; sozlamalar bir joyda.
- **DRY:** `ownedLesson` bitta umumiy helperga (lesson/room/recording/waitingroom takrorini olib tashlash).
- **Observability:**
  - Domen metrikalari (Prometheus): faol xonalar, ishtirokchilar, yozuvlar soni, admit/reject, WS ulanishlar.
  - Distributed tracing (OpenTelemetry) — asosiy oqimlar (allaqachon otel dep bor).
  - Grafana dashboard + alertlar (xato darajasi, p95, DB pool, Redis).
  - *Mezon:* `/metrics`da domen metrikalari; dashboard bor.
- **API hujjatlari:** `make swagger` regen; barcha endpoint annotatsiyalari to'liq; xato-envelope izchilligi + response'da `request_id`.
- **Kod tozaligi:** `rbac.go` `validRoles` (mentor/student) tuzatish yoki o'lik `RBAC()`ni olib tashlash; ishlatilmagan Hub metodlari (`BroadcastToRoom/Online`) — ishlatish yoki olib tashlash.
- **Dockerfile:** multi-stage, non-root user, `HEALTHCHECK`, minimal image (distroless).
- **Hujjatlar:** `README`, `ARCHITECTURE.md`, deploy runbook.

---

## FAZA 2 — Latency optimizatsiyasi (P1) · ~1 hafta

### Backend
- **DB:** per-query `statement_timeout`; indekslarni qayta ko'rib chiqish (covering); `EXPLAIN ANALYZE` issiq so'rovlarga; prepared statements; pool tuning yuk ostida.
- **HTTP:** gzip/br compression, HTTP/2, keep-alive; javob hajmini kamaytirish (faqat kerakli maydonlar).
- **Redis:** pipelining, pool tuning; kutish-token TTL/urf sozlash.
- **Round-trip kamaytirish:** joinlink allaqachon bitta chaqiruvda token beradi ✅; boshqa oqimlarda ham birlashtirish.

### Media (LiveKit)
- **Simulcast/dynacast/adaptive-stream** frontend klientda yoqilganini tasdiqlash (yomon tarmoqda past sifatga tushish).
- **Kodek strategiyasi:** moslik uchun H264/VP8; bandwidth uchun VP9/AV1 (opsiyonal), qurilmaga qarab.
- **ICE fallback tartibi:** UDP → TCP → TURN/TLS; klient ICE-restart va bandwidth-estimation.
- **Egress alohida node:** yozib olishni SFU'dan ajratish (CPU izolyatsiya), Chrome resurs limitlari.

### Yetkazib berish
- **CDN** frontend + yozuvlar oldida (Cloudflare/CloudFront); signaling uchun edge/anycast (imkon bo'lsa).
- *Mezon:* yozuv yuklab olish CDN-edge'dan; TTFB pasayadi.

---

## FAZA 3 — Masshtab va ilg'or chidamlilik (P2) · keyinroq

- **LiveKit multi-node + Redis** (horizontal), region-aware routing; foydalanuvchini eng yaqin node'ga.
- **Egress autoscale** (navbat + worker pool).
- **DB read-replica** og'ir o'qishlarga; PgBouncer.
- **Reminder/worker leader-election** yoki job-queue (bir nechта instans uchun) — hozir atomik claim bilan xavfsiz, lekin queue tozaroq.
- **SLO'lar + doimiy load-test:** p95 join < 1.5s, connect-success > 99%, 100+ ishtirokchili xona; `k6` + `livekit-cli load-test` CI-cron.
- **Chaos/resilience:** node/DB/Redis uzilishida graceful degradation testlari.
- **Ko'p regionli DR** (disaster recovery), backup/restore mashqi.

---

## Ketma-ketlik va bog'liqliklar

```
FAZA 0 (P0) ──► FAZA 1 (P1) ──► FAZA 2 (P1) ──► FAZA 3 (P2)
   │
   ├─ 0.1 Testlar        (mustaqil, DARHOL boshlanadi — keyingi hamma ish uchun setka)
   ├─ 0.2 Xavfsizlik     (testlar bilan parallel)
   └─ 0.3 TLS/TURN       (deployment/infra — parallel, real muhit talab qiladi)
```

**Tavsiya qilingan boshlanish:** `0.1 Testlar` — chunki testsiz keyingi har bir o'zgarish (tranzaksiya refactor, config, latency tuning) regressiya xavfi tug'diradi. Test setkasi o'rnatilgach, `0.2 Xavfsizlik yakuni` va `0.3 TLS/TURN` parallel boradi.

## Baholar (hozir → maqsad)
| O'lcham | Hozir | FAZA 0 dan keyin | Maqsad (FAZA 2) |
|---------|-------|------------------|-----------------|
| Xavfsizlik | ~7 | ~8.5 | 9+ |
| Professionallik | ~6 | ~8 | 9+ |
| Latency/yomon internet | ~6 | ~8 (TLS/TURN) | 9+ |
