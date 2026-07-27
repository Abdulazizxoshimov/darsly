# Darsly — MVP Chiqarish oldidan bajarilishi kerak bo'lgan ishlar

> **Manba:** 6 mustaqil senior audit (Backend, Frontend, Xavfsizlik, Integratsiya, Mahsulot, DevOps) — 2026-07-24.
> **Umumiy verdikt:** Kod release-candidate darajasида pishiq, lekin (1) prod deployment stack yig'ilmagan, (2) jonli video hech qachon real brauzerда sinalmagan, (3) bir nechta arzon xavfsizlik/UX teshigi bor. **Hozir PUBLIC uchun NO-GO; A-fazasi yopilsa yopiq beta uchun GO.**

## Ballar

| Qatlam | Ball |
|--------|------|
| Front↔Back integratsiya | 97% |
| Backend | 90% |
| Xavfsizlik | 82% |
| Frontend | 80% |
| Mahsulot | 80% |
| Infratuzilma | 58% |

## Ishlash tartibi (fazalar)

- **A-FAZA** — yopiq beta uchun MAJBURIY (release blocker). Bularsiz chiqarmaslik.
- **B-FAZA** — public reliz uchun MAJBURIY (A + bular).
- **C-FAZA** — polish / nice-to-have (chiqargandan keyin ham bo'ladi).

Har bir band: `[ ]` checkbox, `fayl:qator`, muammo, ta'sir, tuzatish. Manba agent qavs ichida.

---

# A-FAZA — Yopiq beta blocker'lari (MAJBURIY)

## A0. Jonli video — real brauzer sinovi *(mahsulot yadrosi)* [Product]

- [ ] **A0.1 — 2-brauzer + real kamera bilan 2-tomonlama video sinovi.**
  - Muammo: Kod to'liq (`LiveRoom/Stage/Controls/useRoom`), SFU 500 obunachigacha load-test qilingan, LEKIN 2 ta real brauzer + kamera bilan hech kim ishlatib ko'rmagan. `ws_url`, TURN, brauzer ruxsati, autoplay siyosati faqat inson sinovida chiqadi.
  - Ta'sir: Butun mahsulotning sababi. Ishlamasa MVP yo'q.
  - Sinov stsenariylari (docs/final-report.md dagi 5 stsenariy):
    1. Ikki tomon bir-birini ko'radi/eshitadi (mentor + guest)
    2. Waiting-room: guest so'rov → mentor admit → guest xonaga kiradi
    3. Xona-boshqaruvi: mute-all / kick / allow-speak tugma → real natija
    4. REC start/stop → yozuv MinIO'да paydo bo'ladi va yuklab olinadi
    5. Wi-Fi uzib-ulash (reconnect) + silent-refresh sessiya

## A1. Prod deployment stack'ni yig'ish *(5 blocker)* [DevOps]

- [x] **A1.1 (F2) — Prod app dev `.env` ni yuklaydi.**
  - Fayl: `backend/deployments/docker-compose.yml:42-43` (`env_file: ../.env`)
  - Muammo: `backend/.env` dev qiymatlari — `APP_PORT=8087`, `DB_HOST=localhost`, `DB_PORT=5442`. Lekin compose `expose: 8080`, healthcheck `:8080`, nginx `upstream app:8080`, DB host = `postgres:5432`.
  - Ta'sir: app 8087'да tinglaydi → nginx/healthcheck 8080'ga uradi (stack ishlamaydi); DB `localhost:5442` ga ulanmoqchi (konteynerда yo'q).
  - Tuzatish: Alohida prod `.env` (`APP_PORT=8080`, `DB_HOST=postgres`, `DB_PORT=5432`, `REDIS_HOST=redis`, `MINIO_ENDPOINT=minio:9000`, ...). Compose'да `../.env` → `.env.prod` ga o'zgartirish.

- [x] **A1.2 (F5) — `frontend_dist` volume hech qachon to'ldirilmaydi.**
  - Fayl: `backend/deployments/docker-compose.yml:11,150`
  - Muammo: nginx `frontend_dist:/usr/share/nginx/html:ro` dan SPA beradi, lekin stack'да frontend'ni `npm run build` qilib bu volume'ga yozadigan service yo'q.
  - Ta'sir: nginx bo'sh papka beradi → SPA umuman ochilmaydi.
  - Tuzatish: frontend build stage / init-konteyner qo'shish yoki CI artefaktни volume'ga ko'chirish.

- [x] **A1.3 (F1) — `egress.yaml` hardcode dev kaliti, env substitution yo'q.**
  - Fayl: `services/livekit/egress.yaml:6-9` (`api_key: devkey`, `api_secret: secret_at_least_32...`)
  - Muammo: `docker-compose.prod.yml:67-69` shu `egress.yaml` ni `${...}` siz ishlatadi.
  - Ta'sir: prod recording (Egress) dev kaliti bilan ishlaydi → LiveKit bilan imzo mos kelmaydi → yozib olish ishlamaydi.
  - Tuzatish: `egress.yaml` da `${LIVEKIT_API_KEY}` / `${LIVEKIT_API_SECRET}` ishlatib compose'да env uzatish (`livekit.prod.yaml` allaqachon shunday).

- [x] **A1.4 (F7) — backend stack va livekit stack alohida tarmoq.**
  - Fayl: `backend/deployments/docker-compose.yml` va `services/livekit/docker-compose.prod.yml`; `services/livekit/Caddyfile:9-21`
  - Muammo: Caddy `app:8080`, `minio:9000`, `livekit:7880` ga murojaat qiladi, lekin `app`/`minio` boshqa stack'да. Ikkita reverse-proxy (nginx VA caddy) bir vaqtда 80/443 talab qiladi.
  - Ta'sir: Caddy backend/MinIO'ga yeta olmaydi; yaxlit prod topologiya yo'q.
  - Tuzatish: Umumiy `external network` yoki bitta compose fayl. Bitta reverse-proxy'ga qaror (nginx YOKI caddy).

- [x] **A1.5 (F20) — `TRUSTED_PROXIES` bo'sh.**
  - Fayl: `backend/.env.example:82`
  - Muammo: Reverse-proxy ortida `ClientIP` = nginx/caddy IP bo'ladi.
  - Ta'sir: (a) rate-limit barcha foydalanuvchini bitta bucket'ga yig'adi (bloklaydi yoki mantiqsiz), (b) real IP loglanmaydi, audit yo'qoladi.
  - Tuzatish: Prod'да majburiy to'ldirish (nginx `X-Forwarded-For` yuboradi — `nginx.conf:46`).

## A2. Xavfsizlik — arzon lekin jiddiy HIGH'lar [Security]

- [x] **A2.1 (H-2) — Mentor = global admin, tenant izolyatsiyasi yo'q.**
  - Fayl: `backend/internal/pkg/casbin/policy.csv:19-26`
  - Muammo: Har qanday `mentor` roli BUTUN user bazasi ustidan CRUD oladi: `DELETE /users/:id`, `/deactivate`, `/activate`, `PUT /users/:id/password` (joriy-parolsiz reset), `PUT /users/:id` (rol o'zgartirish).
  - Ta'sir: Bitta mentor buzilsa → istalgan hisobni o'chirish/deaktivatsiya/parol reset qilib akkaunt egallash → butun platforma.
  - Tuzatish: Alohida `admin` roli kiritish; user-management siyosatlarini `mentor`'dan `admin`'ga ko'chirish. Mentorga faqat o'z profili + o'z darslari. Parol reset uchun target = o'z-o'zi yoki admin ekanini enforce qilish.

- [x] **A2.2 (H-3) — Access JWT ilova loglariga ochiq yoziladi.**
  - Fayl: `backend/api/middleware/logging.go:31-33` + `backend/api/router.go:138` (`/ws?token=`)
  - Muammo: Logger har so'rovда `URL.RawQuery` ni log qiladi; WS token'ni `?token=<jwt>` orqali oladi → jonli access JWT Loki/stdout'ga plaintext. Guest `?request_id=<uuid>` ham (admit-token capability).
  - Ta'sir: Log oqimiga kirgan har kim jonli token'ni o'g'irlab sessiya egallaydi.
  - Tuzatish: Loglashда `token` va sezgir query param'larni redaksiya (`token=REDACTED`). Iloji bo'lsa WS'да `Sec-WebSocket-Protocol` subprotocol yoki bir martalik ticket.

## A3. Mahsulot — o'lik oqim va brend [Product/DevOps]

- [x] **A3.1 — "Parolni tiklash" o'lik oqim.**
  - Fayl: `backend/.env.example` (`EMAIL_ENABLED=false`), `backend/internal/infrastructure/email/email.go` (`nopSender`)
  - Muammo: Email o'chiq → sender xatni jimgina tashlab yuboradi. Foydalanuvchi "email yuborildi" ko'radi, lekin hech narsa kelmaydi.
  - Ta'sir: Parolini unutgan foydalanuvchi tizimga qayta kira olmaydi.
  - Tuzatish: YO SMTP yoqib real email (+ brend, pastда) YO MVP'да "Parolni unutdim" havolasini vaqtincha yashirish.

- [x] **A3.2 (F24) — Email shablonlarida "JiraFlow" brendi.**
  - Fayl: `backend/internal/infrastructure/email/email.go:92,128` (footer "JiraFlow", display-name "Jiraflow")
  - Muammo: SMTP yoqilsa foydalanuvchi begona brend ko'radi.
  - Ta'sir: Ishonchsizlik, unprofessional.
  - Tuzatish: Barcha "JiraFlow" → "Darsly", `noreply@darsly.uz`.

## A4. Frontend — UX blocker'lari [Frontend]

- [x] **A4.1 — Mobil navigatsiya butunlay yo'q.**
  - Fayl: `frontend/src/styles.css:904` (`@media (max-width:900px){.sidebar{display:none}}`) + `frontend/src/components/AppShell.jsx`
  - Muammo: 900px'дан pastда sidebar yashiriladi, o'rniga mobil menyu (hamburger/bottom-nav) yo'q. Topbar'да faqat bell + logout.
  - Ta'sir: Telefonда `/app` ичида Jadval/Yozuvlar/Profil'ga umuman o'ta bo'lmaydi (o'quvchilarning katta qismi telefonда).
  - Tuzatish: Mobil hamburger yoki pastki tab-bar navigatsiya (`AppShell.jsx` + CSS).

- [x] **A4.2 — Guest xonада sahifa yangilansa dars yo'qoladi.**
  - Fayl: `frontend/src/lib/roomSession.js:3`, `frontend/src/views/LiveRoom.jsx:29,73`
  - Muammo: `roomSession` faqat in-memory. Guest F5 bosса → `token=null` → "Token topilmadi"; "Orqaga" tugmasi `navigate('/')` (bosh sahifa) — join-link ham qolmaydi.
  - Ta'sir: Tasodifiy refresh = darsdan chiqib ketish, qayta ulanish yo'li yo'q.
  - Tuzatish: `roomSession` ni `sessionStorage`'ga ko'chirish, yoki `error` holatда "Orqaga"ni `/r/:slug` (join sahifa)ga yo'naltirish, `slug`ni saqlab.

---

# B-FAZA — Public reliz blocker'lari (A + bular)

## B1. Xavfsizlik — HIGH (yumshatiladigan) va MEDIUM [Security]

- [ ] **B1.1 (H-1) — Sessiya bekor qilish Redis uzilganда fail-open.**
  - Fayl: `backend/internal/pkg/token/jwt.go:113-120` (`ValidateAccess`)
  - Muammo: Redis `Exists` xato bersa kod "JWT-only" degrade rejimga o'tib, sessiya tekshiruvsiz token qabul qiladi.
  - Ta'sir: Redis outage (15 daq TTL oynasi)да logout/parol-reset/deaktivatsiya qilingan token qayta ishlaydi. Hujumchi Redis'ni DoS bilan cho'ktirib revocation'ni chetlab o'tishi mumkin.
  - Tuzatish: Yozuvli/sezgir endpointlar (parol, admin, to'lov) uchun config bilan fail-closed (deny-on-error) rejim; access TTL'ni qisqartirish; metrika + alert.
  - *(Backend agentda ham #1 sifatida — bir muammo.)*

- [ ] **B1.2 (H-1 juftligi) — `RevokeAllUserSessions` butun keyspace SCAN qiladi.**
  - Fayl: `backend/internal/pkg/token/jwt.go:227-266`; qo'zg'atuvchi `jwt.go:148` (`Rotate` reuse-detection)
  - Muammo: Har parol-reset/deaktivatsiya/refresh-reuse `prefix:sess:*` va `prefix:refresh:*` to'liq SCAN ishga tushiradi.
  - Ta'sir: Hujumchi reuse'ni ataylab qo'zg'atib DoS-amplifikatsiya (katta sessiya bazasida Redis latency spike).
  - Tuzatish: `user:<id>:sessions` set-index yuritib O(1) revokatsiya.

- [ ] **B1.3 (M-1) — Frontend token'larni localStorage'да saqlaydi.**
  - Fayl: `frontend/src/api/api.jsx:14-25`; refresh TTL 720h (30 kun)
  - Muammo: localStorage XSS bilan to'liq o'qiladi; bitta XSS → 30 kunlik refresh eksfiltratsiya (rotation yordam bermaydi).
  - Ta'sir: XSS → uzoq muddatli akkaunt egallash.
  - Tuzatish: refresh token'ni `HttpOnly; Secure; SameSite` cookie'да; access'ni memory'да. Yoki refresh TTL kamaytirish + qurilma-binding.

- [ ] **B1.4 (M-2) — CSP `script-src 'unsafe-inline'`.**
  - Fayl: `backend/api/middleware/security.go:20`
  - Muammo: Inline skriptlarga ruxsat XSS'ni osonlashtiradi (M-1 bilan token o'g'irlash).
  - Ta'sir: XSS surface kengroq.
  - Tuzatish: nonce/hash asosли CSP; inline skriptlarni olib tashlash. (`'wasm-unsafe-eval'` LiveKit uchun qoladi.)

- [ ] **B1.5 (M-3) — Login'да account-lockout yo'q (faqat IP rate-limit).**
  - Fayl: `backend/api/router.go:111` (60/min/IP), `backend/internal/usecase/auth/auth.go:117`
  - Muammo: IP bo'yicha limit + bcrypt(11) bor, lekin hisob bo'yicha lockout yo'q → IP-rotatsiyали (botnet) distributed brute-force.
  - Ta'sir: Bitta account'ni nishonga olish mumkin.
  - Tuzatish: Email bo'yicha muvaffaqiyatsizlik hisoblagichi + progressiv lockout/CAPTCHA.

- [ ] **B1.6 (M-4) — LiveKit token TTL 6h, revocation yo'q.**
  - Fayl: `backend/internal/infrastructure/livekit/token.go`, `backend/internal/pkg/config/config.go:239`
  - Muammo: Token 6h amal qiladi; leak bo'lsa xona o'chirilgunча qayta ishlatiladi (JWT stateless).
  - Ta'sir: Leak oynasi katta.
  - Tuzatish: TTL 1-2h; reconnect uchun backend token-refresh endpoint; dars tugaganда `DeleteRoom` ishonchli.

## B2. DevOps — prod hardening [DevOps]

- [ ] **B2.1 (F13) — TURNS 443/TLS aslida ulanmagan.**
  - Fayl: `services/livekit/docker-compose.prod.yml:12-14`, `services/livekit/Caddyfile:23-26`
  - Muammo: Caddy L4/xom-TLS uzata olmaydi; TURNS faqat 5349'да. "Faqat-443 korporativ firewall" foydalanuvchilari media ololmaydi.
  - Ta'sir: Yomon-internet/korporativ tarmoq goal'i qoplanmagan.
  - Tuzatish: HAProxy/nginx-stream L4 LB bilan TURNS 443 ustidan.

- [ ] **B2.2 (F6) — Restart policy va resurs limitlari nomukammal.**
  - Fayl: `backend/deployments/docker-compose.yml:62-147`
  - Muammo: `restart: unless-stopped` faqat nginx/certbot/app'да (postgres/redis/minio/rabbitmq/loki/grafana'да yo'q); resurs limit faqat app'да.
  - Ta'sir: DB crash bo'lsa ko'tarilmaydi; DB/MinIO/RabbitMQ cheksiz → OOM riski.
  - Tuzatish: Barcha service'ga `restart: unless-stopped` + resurs limit.

- [ ] **B2.3 (F16) — CD (deploy) bosqichi yo'q.**
  - Fayl: `.github/workflows/ci.yml`
  - Muammo: CI faqat quality-gate (build/vet/lint/test-race). Image build/push, integration test (`-tags=integration`), migratsiya smoke, image skanlash yo'q.
  - Ta'sir: Deploy qo'lda, release avtomatlashuvi nol.
  - Tuzatish: CD pipeline (image build/push + integration test + smoke).

- [ ] **B2.4 (F18) — Prometheus scraper prod stack'да yo'q.**
  - Fayl: `backend/deployments/docker-compose.yml`
  - Muammo: `/metrics` ochiladi, Grafana+Loki bor, lekin Prometheus konteyner/scrape yo'q.
  - Ta'sir: Metrikalarni hech kim yig'maydi (faqat log dashboard).
  - Tuzatish: Prometheus konteyner + scrape config qo'shish.

- [ ] **B2.5 (F21) — Backup strategiyasi umuman yo'q.**
  - Muammo: Hech qayerда `pg_dump`/volume-snapshot/cron yo'q; `pg_data` yagona nusxa.
  - Ta'sir: Ma'lumot yo'qotish riski yuqori.
  - Tuzatish: Hech bo'lmasa kunlik `pg_dump` cron + offsite.

- [ ] **B2.6 (F22) — MinIO ochiq endpoint prod'да ulanmagan.**
  - Fayl: `backend/.env.example:48` (`MINIO_PUBLIC_ENDPOINT=` bo'sh)
  - Muammo: Presigned URL ichki manzilga imzolanadi (tashqi klient yeta olmaydi).
  - Ta'sir: Recording yuklab olish ishlamaydi.
  - Tuzatish: `MINIO_PUBLIC_ENDPOINT=files.darsly.uz` ni ulash (`Caddyfile:19-21`).

- [ ] **B2.7 (F19) — `/metrics` va `/swagger` prod'да ochiq.**
  - Fayl: `backend/api/router.go:98` (`/metrics`), `backend/deployments/nginx/nginx.conf:62-65` (`/swagger` tashqariga ochiq)
  - Muammo: `/swagger` omma uchun ochilgan.
  - Ta'sir: API dokumentatsiyasi ommaga; `/metrics` ichki tarmoqда auth'siz.
  - Tuzatish: Prod'да `/swagger` yopish; `/metrics` ni faqat ichki/scraper uchun.

- [ ] **B2.8 (F11) — Migratsiya rollback/backward-compat strategiyasi yo'q.**
  - Fayl: `backend/internal/pkg/postgres/migrate.go`, `backend/migrations/`
  - Muammo: Faqat oldinga migratsiya; "dirty" holatда qo'lда `force` kerak. `.down.sql` lar CI'да sinovдан o'tkazilmagan. Zero-downtime backward-compat hujjatlanmagan.
  - Ta'sir: Yarim-bajarilgan migratsiya app boot fail.
  - Tuzatish: CI'да `migrate down/up` smoke; migratsiyalar backward-compatible bo'lishi hujjatlansин.

- [ ] **B2.9 (F23) — Redis prod persistence yo'q.**
  - Fayl: `backend/deployments/docker-compose.yml` (redis)
  - Muammo: `appendonly`/RDB config yo'q.
  - Ta'sir: Restart'да sessiya/refresh/admit-token yo'qoladi (qayta login).
  - Tuzatish: `appendonly yes` yoki RDB snapshot.

- [ ] **B2.10 (F8 note) — Go versiya mosligi.**
  - Fayl: `backend/deployments/Dockerfile` (`golang:1.26-alpine`) vs `backend/go.mod` (`1.25`)
  - Muammo: Build image go 1.26, modul 1.25.
  - Ta'sir: Nomuvofiqlik ehtimoli.
  - Tuzatish: Versiyalarni moslashtirish.

## B3. Backend — production hardening [Backend]

- [ ] **B3.1 — casbin/migrations nisbiy yo'llar.**
  - Fayl: `backend/internal/app/app.go:93` (`RunMigrations(..., "migrations")`), `app.go:186` (`casbin.NewEnforcer("internal/pkg/casbin/model.conf", ...)`)
  - Muammo: CWD'ga bog'liq. Hozir Dockerfile WORKDIR `/app` bilan ishlaydi, lekin WORKDIR/ishga tushirish joyi o'zgarsa: casbin enforcer=nil → himoyalangan route 503, yoki migratsiya abort.
  - Ta'sir: Deployment mo'rtligi.
  - Tuzatish: `go:embed` bilan model/policy va migratsiyalarni binarга joylash.

- [ ] **B3.2 (Backend#4 / Security L-2) — `PollResults` to'liq ochiq.**
  - Fayl: `backend/api/handlers/v1/poll.go:113,118`, `backend/api/router.go:182` (`/polls/:id/results`)
  - Muammo: Vote room-token talab qiladi, natijalar tokensiz ochiq.
  - Ta'sir: PollID (UUID) bilgan har kim natijani o'qiydi (info-leak, efemer).
  - Tuzatish: Results'ni ham room-token yoki host-only.

- [x] **B3.3 (L-1) — Rol qiymati validatsiyasiz (mentor set qiladi).**
  - Fayl: `backend/internal/usecase/user/user.go:91-93`
  - Muammo: Mentor `PUT /users/:id` yoki `CreateUser`да ixtiyoriy `role` string qo'yishi mumkin (allow-list `{mentor,student,guest}` yo'q).
  - Ta'sir: Noto'g'ri rol → casbin policy topilmай akkaunt bloklanadi (DoS).
  - Tuzatish: Enum/allow-list validatsiya. *(A2.1 admin-roli ishi bilan birga qilish mantiqli.)*

- [ ] **B3.4 — Test qamrovi bo'shliqlari.**
  - Fayl: `backend/internal/infrastructure/repository/postgres/*` (faqat `integration_test.go`, jonli Postgres bo'lsagина), `internal/usecase/user`, `internal/usecase/shared`, `internal/worker`
  - Muammo: apitests fake repo ishlatadi → real Squirrel query CI'да (DB'siz) sinalmaydi. usecase/user, shared, worker (reminder/email) unit testsiz. H-1 degrade yo'li va `RevokeAllUserSessions` testsiz.
  - Ta'sir: SQL regressiyasi va edge-case'lar sezilmай qoladi.
  - Tuzatish: CI'да Postgres service majburiy qilib real query'larni qamrash; usecase/worker unit testlari.

## B4. Integratsiya — kontrakt nomuvofiqligi [Integration]

- [ ] **B4.1 — 429 (rate-limit) xato tanasi formati mos emas.**
  - Fayl: frontend `frontend/src/api/api.jsx:40,91-101`; backend `backend/api/middleware/rate_limit.go:85-88`, `backend/api/middleware/ratelimit_redis.go:62-64`
  - Muammo: (a) backend kod `RATE_LIMITED`, frontend `TOO_MANY_REQUESTS` kutadi (o'lik map); (b) backend xabarni `error` maydonида beradi, frontend `message` o'qiydi.
  - Ta'sir: 429'да "Juda ko'p urinish — kuting" o'rniga umumiy "Xatolik" (kosmetik).
  - Tuzatish: Backend 429 → `{code:"TOO_MANY_REQUESTS", message:...}`, YOKI frontend map'ga `RATE_LIMITED` + `error` fallback.

---

# C-FAZA — Polish / nice-to-have (chiqargandan keyin)

## C1. Frontend — muhim lekin blocker emas [Frontend]

- [ ] **C1.1 — Guest'да poll "yopilishi" ishlanmagan.**
  - Fayl: `frontend/src/views/LiveRoom.jsx:145` (`DataReceived` faqat `action==='open'`); yuboruvchi `frontend/src/panels/PollsPanel.jsx:78` (`doClose`), refetch `PollsPanel.jsx:168`
  - Muammo: Host poll yopgach guest'да `'close'` e'tiborsiz → poll faol qoladi, `usePollResults` `refetchInterval:3000` cheksiz so'rov.
  - Ta'sir: Behuda so'rovlar + noto'g'ri UI.
  - Tuzatish: `'close'`да `setGuestPoll(null)` yoki poll'ni yopiq belgilash.

- [ ] **C1.2 — Refresh muvaffaqiyatli, retry 401 bo'lsa logout bo'lmaydi.**
  - Fayl: `frontend/src/api/api.jsx:117-129`
  - Muammo: Retry ham 401 qaytarsa `!_retried` false → `onUnauthorized()` chaqirilmaydi, shunchaki `ApiError`.
  - Ta'sir: Yaroqsiz sessiya tozalanmay qolishi mumkin.
  - Tuzatish: Retry'дан keyingi 401'да ham `onUnauthorized()`.

- [ ] **C1.3 — Yozuvlar sahifasида bo'sh holat ko'rinmaydi.**
  - Fayl: `frontend/src/views/Recordings.jsx:11,42`
  - Muammo: Ro'yxat `is_recording_enabled` bo'yicha filtrlangan, `LessonRecordings` haqiqiy yozuv yo'q bo'lsa `return null`. Yozuv yoqilgan lekin hali yozilmagan → sarlavha ostida bo'sh maydon, "yozuv yo'q" xabarsiz.
  - Ta'sir: Chalg'ituvchi bo'sh ekran.
  - Tuzatish: Haqiqiy yozuvlar sonini yig'ib empty-state ko'rsatish.

- [ ] **C1.4 — `navigator.clipboard.writeText` try/catch'siz.**
  - Fayl: `frontend/src/views/Dashboard.jsx:62`
  - Muammo: HTTPS bo'lmagan/ruxsat rad etilgан kontekstда reject, lekin `toast.success('Havola nusxalandi')` baribir chiqadi (yolg'on ijobiy).
  - Ta'sir: Foydalanuvchi nusxalanmagan havolani nusxalandi deb o'ylaydi.
  - Tuzatish: `await` + try/catch, xatoда `toast.error`.

- [ ] **C1.5 — WaitingRoom `admit` qo'sh navigate.**
  - Fayl: `frontend/src/views/WaitingRoom.jsx:20,29,38`
  - Muammo: `admit()` WS handler (28) va polling effect (37) dan qo'sh chaqirilishi mumkin.
  - Ta'sir: Race (`replace:true` bilan katta zarar yo'q).
  - Tuzatish: `useRef` bilan bir marta guard.

## C2. Frontend — kichik / UX [Frontend]

- [ ] **C2.1 — `frontend/src/views/NotFound.jsx`** — bosh sahifaga qaytish havolasi yo'q (boshi berk 404). → havola qo'shish.
- [ ] **C2.2 — `frontend/src/views/Landing.jsx:52-57`** — "Boshlash" va "Ustoz sifatida kirish" ikkalasi `/auth`ga (bir xil), rol ajratilmagan. → ajratish yoki bittasini olib tashlash.
- [ ] **C2.3 — `frontend/src/views/Profile.jsx:61-62`** — timezone/language erkin matn input. → `select` qilish.
- [ ] **C2.4 — `frontend/src/livekit/useRoom.js:80`, `frontend/src/views/LiveRoom.jsx:110`** — `setReactions`/`setTimeout` unmount'да tozalanmaydi (React 18'да no-op, leak emas). → cleanup qo'shish.
- [ ] **C2.5 — `frontend/src/lib/ws.js:36-41`** — token yaroqsiz bo'lsa cheksiz reconnect loop (backoff bilan), to'xtash sharti yo'q. → max-retry yoki auth-fail'да to'xtatish.
- [ ] **C2.6 — Responsive:** `.room`, `.grid-cards`, `.panel` uchun 900px'дан pastда maxsus layout yo'q → jonli xona telefonда siqiladi. → mobil layout.

## C3. Backend — kichik [Backend]

- [ ] **C3.1 — `MuteAll` ketma-ket N ta LiveKit chaqiruvi.**
  - Fayl: `backend/internal/usecase/room/room.go:250-266`
  - Muammo: Har ishtirokchi uchun navbatма-navbat 10s-context `MuteParticipant`.
  - Ta'sir: Katta sinfда "mute all" sekin.
  - Tuzatish: `UpdateRoomMetadata`/broadcast yoki concurrent bounded pool.

- [ ] **C3.2 — slug TOCTOU.**
  - Fayl: `backend/internal/usecase/lesson/lesson.go:159-172`
  - Muammo: `SlugExists` → `Create` orasида poyga; DB UNIQUE ushlaydi, lekin do'stona xato o'rniga 500.
  - Ta'sir: Juda kam ehtimol, do'stona emas.
  - Tuzatish: `Create`да unique-violation'ni tutib retry.

- [ ] **C3.3 — `mentorName` cache 1 soat stale.**
  - Fayl: `backend/internal/usecase/joinlink/joinlink.go:170-181`
  - Muammo: Mentor ismini o'zgartirsa join-preview'да 1 soatgacha eski ism.
  - Ta'sir: Kosmetik.
  - Tuzatish: Cache invalidatsiya yoki TTL qisqartirish.

## C4. Xavfsizlik — LOW / info [Security]

- [ ] **C4.1 (L-3) — `LIVEKIT_WEBHOOK_API_KEY` o'qiladi lekin ishlatilmaydi.**
  - Fayl: `backend/internal/pkg/config/config.go:238`; webhook `backend/internal/infrastructure/livekit/egress.go:59` `apiKey/apiSecret` bilan tekshiradi.
  - Tuzatish: Ops'ni chalg'itмаслик uchun o'chirish yoki hujjatlash.

- [ ] **C4.2 (L-5) — Webhook replay himoyasi (nonce/idempotency) yo'q.**
  - Fayl: `backend/internal/infrastructure/livekit/egress.go:58`
  - Muammo: `ReceiveWebhookEvent` body-sha256+JWT timing-safe tekshiradi, lekin nonce store yo'q. `HandleEgress` deyarli idempotent → ta'sir past.
  - Tuzatish: Idempotency-store (ixtiyoriy).

## C5. Mahsulot — MVP scope'дан tashqari [Product]

- [ ] **C5.1 — Guest chat/poll faqat LiveKit data-channel (client).** Backend'да guest push yo'q → data-channel ishlamasa guest jim qoladi. → backend WS orqali guest push (data-channel bog'liqligini kamaytirish).
- [ ] **C5.2 — Mobil-optimallashtirilgan layout** (video-grid + boshqaruv panellari) tekshirilmagan.
- [ ] **C5.3 — Fail-open Redis → fail-closed qarori** (B1.1 bilan bog'liq).
- [ ] **C5.4 — Dizaynда bor, MVP'дан tashqари:** whiteboard (doska), breakout xonalar, subtitr/captions, guest-to-guest DM. Backend domeni yo'q — ataylab keyingi bosqich.

---

# Ijobiy — regressiya bo'lmasin uchun qayd (o'zgartirmang)

Bu qismlar to'g'ri yozilган, buzib qo'ymang:

- **SQL:** barcha query Squirrel parametrли; dinamik `ORDER BY` faqat allow-list (`allowedUserSortCols`, `allowedLessonSortCols`).
- **IDOR/ownership:** `shared.OwnedLesson` barcha lesson-bog'liq endpointда; `GetUser` rol+ownership; notifications `user_id` scoped.
- **Privilege escalation:** `UpdateCurrentUser` `req.Role=nil`; refresh rotation + reuse-detection (sessiya oilasi kill).
- **Atomik state:** waitingroom `TransitionFromPending` (2-admit → 409); reminder `ClaimReminder`; poll `ON CONFLICT`.
- **Fail-closed:** RBAC enforcer nil → 503; rate-limit Redis xato → in-memory fallback.
- **Timing/enumeration:** dummy bcrypt; ForgotPassword har doim 200.
- **Config `Validate()`** prod'да dev/default secretlarni rad etadi (fail-fast).
- **Webhook** imzo majburiy (body-sha256 + JWT, timing-safe).
- **Concurrency:** `-race` toza; graceful shutdown (HTTP drain → WS → worker WaitGroup).
- **Observability wiring:** Sentry/Loki/OTEL/Prometheus + `/health`+`/ready` — namunaviy.
- **Integratsiya 97%:** 40+ endpoint yo'l/metod/shakl/auth/WS 1:1 mos.

---

## Jonli sinov davomida topilgan va TUZATILGAN bug (2026-07-24)

- [x] **LIVE-1 — `userRepo.Update` `role` ustunini saqlamasди.**
  - Fayl: `backend/internal/infrastructure/repository/postgres/user.go:154` (Update SQL `Set(...)` da `role` yo'q edi)
  - Muammo: Admin `PUT /users/:id {role:mentor}` yuborsa javob `role:mentor` qaytardi, lekin DB'ga yozilmasди → admin hech kimni mentor qila olмасди (A2.1 admin modeli amalда ishlамасди).
  - Aniqlandi: dev'да admin→mentor promote qilinди, DB'да rol `student` qoldi.
  - Tuzatildi: Update SQL'ga `.Set("role", user.Role)` qo'shildi. Self-update (`PUT /users/me`) xavfsiz — `req.Role=nil` bo'lganда `GetByID`дан kelgan mavjud rol saqlanadi.
  - Tekshirildi: admin→mentor promote → DB `mentor` ✅, mentor login JWT `role:mentor` ✅, mentor `GET /users` → 403 ✅, admin → 200 ✅, `go test` (repo+apitests) yashил ✅.

- [x] **LIVE-2 — Jonli xona qora oyna (host) — `null.length` crash.**
  - Fayl: `frontend/src/views/LiveRoom.jsx:104,337`, `frontend/src/panels/ParticipantsPanel.jsx:13`, chat: `LiveRoom.jsx:127`
  - Muammo: Backend bo'sh kutish-ro'yxatini `{"data":null}` qaytaradi (Go nil slice→JSON null). Frontend `const { data: waiting = [] }` — default `[]` faqat `undefined`da ishlaydi, `null`da EMAS → `waiting = null` → `waiting.length` render paytida throw → RoomStage crash → React unmount → LiveKit `disconnect()` → qora oyna. Faqat HOST (useWaiting mentor uchun). Chat tarixi ham (`for..of null`) xuddi shunday xavf.
  - Aniqlandi: serverда jonli xonaga kirilди → qora oyna; headless brauzer (Playwright+soxta media) `TypeError: Cannot read properties of null (reading 'length')` PAGEERROR ko'rsatdi; ICE/media aslida ULANGAN edi (server aybsiz).
  - Tuzatildi (frontend, null-safe): `waiting = waitingData || []` (2 fayl) + chat `for (const m of msgs || [])`. Frontend qayta build + serverга sync (www bind-mount → darhol jonli).
  - Tekshirildi: repro qayta — 0 xato, ulanish `connected`da qoladi, `videos:1`, UI render ✅.
  - **Backend kontrakt (BAJARILDI):** `api/http_status/response.go` — `jsonData()` reflect bilan nil slice → `[]` (Success/Created/List markazида). Endi HAR list endpoint bo'sh bo'lса `{"data":[]}` (null emas) — kelajakdаги shunga o'xshash crashларни butunlay yopadi. Serverга redeploy qilinди (backend image qayta build), tekshirildi: waitingroom+chat `{"data":[]}` ✅, `go test` yashил ✅.

- [x] **LIVE-3 — Ekran ulashish (screen share) — UX + mobil.**
  - Fayl: `frontend/src/livekit/Controls.jsx` `toggleScreen`
  - Muammo: Xato generic toast'да yutilardi; mobil brauzerда `getDisplayMedia` yo'q (ishlamaydi), foydalanuvchi sababни bilмасди.
  - Tuzatildi: mobil aniqlash ("faqat kompyuterда ishlaydi"), bekor qilinса jim o'tish, `canPublish` guard. Headless test: desktopда ekran track publish bo'ldi ✅. Serverга deploy.
  - Eslatma: ekran ulashish **faqat desktop** (Chrome/Edge/Firefox) — mobil platформа cheklovi.

- [x] **A0.1 — Ikki-tomonlama video (headless E2E) TASDIQLANDI.**
  - Contabo serverда: mentor kamera publish → guest qo'shildi → "2 ishtirokchi", guest mentor videosini real ko'rди (640×360, jonli oqim), 0 crash. Media public IP+UDP orqali ikki WebRTC brauzer o'rtasида. (Real kamera+inson sinovi hali tavsiya, lekin pipeline to'liq ishlaydi.)
  - Guest qora oyna = eski keşlangan bundle edi ([[LIVE-2]] fixдан oldин); yangi bundle bilan guest to'g'ri ishlaydi.

- [x] **LIVE-4 — Oq doska (whiteboard) qo'shildi.**
  - Sabab: matematika ustozi planshet+S Pen bilan misol yozadi; mobil brauzerlar screen-share'ni qo'llamaydi (Android/iPad platforma cheklovi), shuning uchun screen-share o'rniga real-time canvas doska.
  - Fayllar: yangi `frontend/src/livekit/Whiteboard.jsx`; `LiveRoom.jsx` (wb state/applyWb/hostDraw/snapshot/render), `Controls.jsx` ("Doska" tugma), `messaging.js`, `styles.css`.
  - Protokol: mavjud LiveKit data-channel (`kind:'wb'` — stroke/clear/on/off/snapshot). **Backend'ga tegilmadi.** Host chizadi, guest ko'radi (webinar); S Pen bosim sezgir; kech kirgan guest snapshot oladi; koordinatalar en bo'yicha normalizatsiya (barcha qurilmada bir xil).
  - Tekshirildi: ikki-brauzer E2E — host 2 chiziq chizdi, guest canvas'da AYNAN bir xil piksel (7592), 0 pageerror. Serverga deploy qilindi.
  - Ishlaydi: Tab S9+S Pen, telefon, kompyuter (ekran-capture emas).

- [x] **LIVE-5 — Mobil boshqaruv paneli kesilib qolishi tuzatildi.**
  - Fayl: `frontend/src/styles.css` (@media ≤900px va ≤480px)
  - Muammo: `.controls` (11 tugma) telefonda bir qatorda sig'may, mikrofon/kamera chapga, "Yakunlash" o'ngga chiqib ketardi (kesilardi).
  - Tuzatildi: mobilda `flex-wrap`, tugmalar kichrayadi (44/40px), `ctrl-sep` yashirin, safe-area padding, panellar to'liq ekran. 320px'da tekshirildi: overflow yo'q, 11/11 tugma + Yakunlash 3 qatorda to'liq ko'rinadi (mobil-viewport screenshot).

- [x] **LIVE-6 — PDF sahifasini doskaga fon qilish (screen-share o'rniga).**
  - Sabab: matematika ustozi PDF kitobdan savol o'qib doskada yechadi; screen-share Tab S9'da imkonsiz (Android platforma cheklovi, native app kerak — Zoom shuning uchun native).
  - Fayllar: `Whiteboard.jsx` (pdfjs-dist, fon render, PDF toolbar + sahifa nav), `LiveRoom.jsx` (bg holati + `bg_start/chunk/end/clear` protokol + snapshot), `styles.css`. **Backend'ga tegilmadi.**
  - Arxitektura: host PDF'ni brauzerda pdfjs bilan render qiladi → joriy sahifani JPEG rasm → LiveKit data-channel orqali ~12KB chunk'larda tarqatadi (guest'lar autentifikatsiyasiz oladi). Faqat joriy sahifa uzatiladi. Kech kirgan guest snapshot bilan sahifa+yozuvni oladi.
  - Tekshirildi: ikki-brauzer E2E — host test PDF yukladi + chizdi, guest canvas AYNAN bir xil piksel (78210), 0 pageerror. Deploy qilindi. pdfjs worker offline bundle (`pdf.worker.min-*.mjs`, same-origin).
  - Ishlaydi: Tab S9 (host PDF+S Pen), telefon/kompyuter (guest ko'radi).

- [x] **LIVE-7 — Cheksiz suriladigan/kattalashtiriladigan doska (infinite pan/zoom + follow-presenter).**
  - Fayl: `Whiteboard.jsx` (to'liq qayta yozildi — world/view model), `LiveRoom.jsx` (`view` act + snapshot'ga view), `styles.css` (toolbar guruhlar, yorliqli PDF tugma).
  - Model: stroke'lar world piksel koordinatada (cheksiz), `view={cx,cy,scale}` (markaz asosli, qurilma o'lchamiga bog'liq emas). Pan: 2 barmoq (sensor) / g'ildirak+Hand asbob (desktop). Zoom: chimchilash / Ctrl+g'ildirak / +−. Fit tugmasi.
  - **Follow-presenter:** host pan/zoom qilsa `{kind:'wb',act:'view'}` tarqaladi → guest avtomatik ergashadi (guest'da pan/zoom yo'q). Kech kirgan guest joriy view+strokes+PDF oladi.
  - PDF fon world (0,0)'da, yuklanganda auto-fit + tarqatiladi.
  - Tekshirildi (ikki-brauzer E2E, local + deployed): chizish/pan/zoom/follow — host va guest AYNAN bir xil piksel/markaz; PDF guest'da identical (64340). 0 pageerror. Deploy qilindi.
  - **Eslatma:** PDF/saqlash tugmalari avvaldan bor edi — foydalanuvchi eski keshdagi bundle'ni ko'rgan (cache-control endi tuzatilgan, hard-refresh yetadi). PDF tugma endi yorliqli ("PDF").

- [x] **LIVE-8 — Doska: fit-follow + kech-kirish yozuv + save host-only.**
  - Muammo (foydalanuvchi): ustoz keng planshetда PDF YONIGA yozadi; o'quvchi tor telefonда — markaz+zoom follow bilan yon-yozuv ekrandan tashqarida qolib ko'rinmasdi. Kech kirgan o'quvchi oldingi yozuvni ko'rmasdi (snapshot bg strokes'ni tozalardi). Save-tugma hammada edi.
  - Tuzatildi: (1) `view` protokoli `{cx,cy,ww,wh}` (host ko'rinayotgan world-to'rtburchak); guest o'z ekraniga CONTAIN qilib fit qiladi → ustoz ko'rgan BUTUN maydonni ko'radi (yon-yozuv ham). (2) `loadBg`/`bg_end` endi strokes'ni tozalamaydi; sahifa almashishda host aniq `clear` yuboradi → snapshot fon+strokes birga → kech guest yozuvni ko'radi. (3) `.wb-save` faqat `canDraw` (host).
  - Tekshirildi (keng-host 1280 + tor-guest 390, local va deployed): guest PDF+yon-yozuvni ko'radi (screenshot bilan ham), kech-kirish yozuv ko'rinadi (red 119), save guest'da yo'q. Deploy qilindi.
  - Tradeoff: tor telefonда ustozning butun keng ko'rinishi sig'diriladi → kontent kichikroq ko'rinadi (lekin hech narsa kesilmaydi).

- [x] **LIVE-9 — Doska: EN-FIT + vertikal scroll (telefonda "too small" tuzatildi).**
  - Muammo: erkin pan/zoom + fit-rectangle follow → ustoz keng ekranда PDF yonига yozsa, telefonда butun keng ko'rinish sig'diriib JUDA KICHIK bo'lardi.
  - Yechim: model o'zgardi — `WORLD_W=1000` qat'iy en, `scale=cssW/WORLD_W` (en DOIM to'liq), view=`{scrollY}` (faqat vertikal). PDF tepада to'liq enда, yozuv TAGIDA, pastга scroll (2-barmoq/wheel/Qo'l), chizishда avto-scroll. Follow: faqat `scrollY` sinxron → hech narsa kichraymaydi. Erkin zoom/gorizontal pan olib tashlandi, "Tepaga" tugmasi qo'shildi.
  - Tekshirildi (planshet-host 800 + telefon-guest 390, local+deployed): telefonда PDF en to'ldiradi (dark 63936 vs eski 5800), PDF tagidagi yozuv guest'да ko'rinadi (red 957). Deploy qilindi.

- [x] **LIVE-10 — Dars ichida muloqot (o'quvchi audio + qo'l + kim gapiryapti).**
  - Fayllar: yangi `RoomRail.jsx` (o'ng chekka: gapirayotganlar + qo'l ko'targanlar), `useRoom.js` (`ParticipantPermissionsChanged`), `LiveRoom.jsx`, `Controls.jsx` (badge=waiting+qo'l), `styles.css` (rail + `.controls` z-index). Backend `allow-speak`/`revoke-speak` allaqachon bor edi.
  - #5 O'quvchi VIDEO'SIZ gapiradi: host `ParticipantsPanel`da "So'zlashga ruxsat" bosadi → o'quvchi mikrofoni faollashadi (`canPublish` real-time), toast; o'quvchi audio yoqib gapiradi. #4 O'ng chekkada "Gapiryapti" indikator (activeSpeakers). #3 qo'l ko'tarish doska rejimida ham rail'da + Ishtirokchilar badge'da. #2 doska rejimida barcha komandalar bosiladi (z-index).
  - Tekshirildi (local+deployed): guest mic boshida o'chiq → host ruxsat → mic faollashadi; qo'l ko'tarish rail'da; DOSKA rejimida ham rail+komandalar ko'rinadi. 0 pageerror. Deploy qilindi.
  - **Kutilmoqda (foydalanuvchi so'radi):** doskani J Notes uslubida boyitish — undo/redo, marker, shakllar, grafik-qog'oz fon (tasdiq kutilmoqda).

*So'nggi yangilanish: 2026-07-25 (dars ichi muloqot: audio+qo'l+speaker)*
