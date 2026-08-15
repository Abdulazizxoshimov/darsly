# Darsly — ochiq ishlar

> Yagona backlog. Oldin bu ma'lumot **7 xil reja hujjatida** tarqoq edi
> (`mvp-todo`, `OPTIMAL-PLAN`, `PERFECTION_PLAN`, `OPTIMIZATION_PLAN`,
> `PRODUCTION-READINESS`, `MOBILE-STATUS`, `MOBILE-DEVICE-TEST`) va ularning
> katta qismi allaqachon bajarilgan bandlardan iborat edi — ya'ni ro'yxatlarga
> ishonib bo'lmasdi. Ular o'chirildi (git tarixida qoladi), qolgan **haqiqiy**
> ishlar shu yerda.
>
> Oxirgi tekshiruv: **2026-07-28** (har band kod bilan solishtirildi).
> Qoida: band shu yerga faqat TEKSHIRILGANDAN keyin tushadi. "Rejada bor" —
> yetarli asos emas.

---

## 📋 Qilinishi kerak bo'lgan ishlar — ustuvor ro'yxat (2026-08-15)

> Bir joyda, ustuvorlik bo'yicha. Har band quyidagi bo'limlarda batafsil.
> Belgilar: 🔴 pilot bloklovchisi · 🟠 ommaviy launchdan oldin · 🟡 keyin (30 kun).
> «Kim»: 👤 = odam/tashqi amal kerak · 🖥 = server/deploy kerak · 💻 = sof kod.

**🔴 Pilotni ochish uchun (asosan deploy'ga bog'liq):**
- [ ] **I1 — Serverga Docker + to'liq deploy** 🖥 — hammasini ochadi (Grafana, real sinov)
- [ ] **I2 — 2 brauzer + real kamera bilan video sinov** 👤
- [ ] **M1 — Nativ WebRTC SIGABRT crash** 🖥 — deploy+telemetriyadan keyin chastota bo'yicha
- [ ] **M2 — 3 pilot ustoz bilan 90 daq real dars** 👤
- [ ] **B1 — Backup real tiklash sinovi** 🖥 (skript tayyor, serverda bir marta sinash)
- [ ] **B2 — Sentry DSN** 👤 (hozircha ATAYLAB o'chiq — kelishilgan; keyin qo'shiladi)

**🟠 Ommaviy launchdan oldin:**
- [ ] **B3 — Refresh token → HttpOnly cookie** 💻 (deploy'dan keyin, orqaga-mos)
- [ ] **I4 — TURN TLS/443** 🖥 (deploy'dan keyin sinaladi)
- [ ] **I3 — Kuzatuv (Grafana) deploy** 🖥 (I1 bilan avtomatik keladi)
- [ ] **F2 — 2-brauzerli real-media E2E** 🖥 (real SFU/CI kerak)
- [ ] **M3 — Real domen** (`darsly.uz`/`jonly.uz`) 👤
- [ ] **M5 qoldig'i** — maxfiylik siyosatini to'ldirish+ochiq URL 👤 · Data Safety formasi (Play Console) 👤
- [ ] **M6 — Yakuniy ilova ikonkasi** 👤 (dizayner)

**🟡 Keyin (30 kun ichida):**
- [ ] **M4 — RoomViewModel refaktori** (1333 qator) 💻
- [ ] **F3 — 287 inline style → tokenlar** 💻
- [ ] **B7 — sessiya teskari indeks** 💻 (premature — ehtiyoj bo'lganda)
- [ ] **I5 — CD (avtomatik deploy)** 🖥
- [ ] **B5 qoldig'i** — Redis `up==0` alerti aynan Redis'ni qamrashini tekshirish (redis_exporter) 🖥
- [ ] Redis prod persistence (AOF) tasdiqlash 🖥 · Masshtab (1000+) — `DEPLOYMENT.md` §6

**✅ 2026-08-15 sessiyada bajarilgani:** F1, F4, web skeleton + `prefers-reduced-motion`,
F5 · B4 (login lockout), B5/B6 hujjat · M5 (akkaunt o'chirish backend+mobil), M9, M10,
APK sayqal (haptika/skeleton/illyustratsiya/edge-to-edge) · I6 runbook · PRIVACY.md ·
backup skriptlari (B1 kod). Batafsil — quyidagi bo'limlarda.

---

## 📌 Joriy holat (2026-07-31)

Loyiha **Jonly** brendiga o'tdi va mahsulot yo'nalishi asoschi intervyusi bilan
belgilandi — manba: `docs/PRODUCT.md`, reja: `docs/LAUNCH-PLAN.md`.

**1-hafta (tugadi):** Zoom modeli (o'quvchi mic+kamera), kirganda mute /
Mute All / self-unmute qulfi, galereya sahifalash, dars-oldi kutish sahifasi,
ban qamrovi (dars/doimiy) + qora ro'yxat, yakunlangan dars havolasining o'lishi.

**2-hafta (TUGADI — backend + web + mobil):** bitta faol sessiya
(`SESSION_REVOKED`), 4 soat limit + bo'sh xona avto-yakuni, yozuv retention
(30 kun, `expires_at`), chat xabarini o'chirish, poll ikki rejimi + e'lon,
emoji reaksiyalar (allowlist), chatda fayl ulashish, sifat profillari
(ovoz > ekran > kamera).

Sinov: `frontend/final-sweep.mjs` — mentor + 2 o'quvchi bilan to'liq oqim
(oxirgi yurish: **17/17 PASS**).

**3-hafta (qisman bajarildi):** ✅ yuklama testi — **300/300 ishtirokchi**
(`docs/load-test-report.md`; yo'lda kritik webhook rate-limit bug'i topib tuzatildi).
Qolgani: Telegram-bot eslatmalari, staging yangilash + jonly.uz, pilot onboarding.

---

## 🔴 Bloklovchi — server qaytishini kutadi

~~Contabo VPS `194.163.139.242` o'chgan~~ → **YANGI SERVER OLINDI (2026-07-31):
`169.58.104.245`** (Ubuntu 24.04, 4 yadro, 7 GB RAM, 94 GB bo'sh). SSH ulanish
tekshirildi ✅. Ma'lumotlar: `deploy/server/SERVER-CREDENTIALS.txt`.

⚠️ Serverda `telegram/Tg-bot` ning ikki boti systemd'da ishlaydi — ularga tegilmaydi.
Serverda **Docker hali yo'q** va Jonly deploy qilinmagan.

- [x] ~~Serverni tiklash~~ — yangi server olindi
- [ ] **Docker + compose o'rnatish**, so'ng `deploy/DEPLOY-CHECKLIST.md`
- [ ] **`deploy/server/Caddyfile` domenlarini yangilash** — hozir eski IP
      (`194.163.139.242.sslip.io`) 3 ta subdomen va CSP'da qattiq yozilgan
- [ ] **Kuzatuv stackini deploy qilish** — konfiguratsiya tayyor va mahalliy
      tekshirilgan (`deploy/server/observability/`). Qadamlar:
      `deploy/DEPLOY-CHECKLIST.md`
- [ ] **Sentry DSN** — backend (`.env`) va mobil
      (`-PdarslySentryDsn=...`). Kod tayyor; DSN'siz telemetriya **jim**
- [~] **DB backup** — skriptlar yozildi: `deploy/server/backup.sh` (kunlik cron
      `remote-setup.sh` orqali → MinIO `darsly-backups`) + `backup-restore-test.sh`
      + `deploy/server/BACKUP.md`. **Qoldi:** serverda deploy'dan keyin tiklashni
      **bir marta real sinash** (sinalmagan backup — backup emas)
- [ ] **A0.1 — 2 brauzer + real kamera bilan uchdan-uchga video sinovi**
      (odam bilan; avtomatlashtirib bo'lmaydi)

## 🟠 Xavfsizlik / ishonchlilik

- [x] **Sessiya bekor qilish Redis fail-open siyosati (B5)** — ONGLI QAROR
      hujjatlashtirildi (`deploy/server/RUNBOOK.md` §7). Xulq allaqachon
      CHEKLANGAN fail-open (`jwt.go:162` — 5 daqiqa grace). Qoldi: Redis
      `up==0` alerti aynan Redis'ni qamrashini tekshirish (redis_exporter).
- [x] **Login account-lockout (B4)** — bajarildi: hisob bo'yicha (IP'dan
      mustaqil) 5 xato → 15 daqiqa qulf (`auth.go`, test bilan). Faqat mavjud
      hisoblar uchun (Redis spam'i oldini olish).
- [ ] `RevokeAllUserSessions` — teskari indeks yo'q, SCAN qoladi (B7). **ATAYLAB
      qoldirildi** — audit ham "hozircha muammo emas" deydi; ~10⁶ kalitdan keyin
      ko'riladi. Pilotda foyda yo'q, yozuv-yo'lига murakkablik qo'shadi.
- [ ] **Refresh token `localStorage`da → HttpOnly cookie (B3)** — P1, pilot
      bloklovchisi EMAS. Web+mobil auth'ga tegadi (mobil Keystore ishlatadi),
      jonli stack'siz to'liq E2E sinab bo'lmaydi. Deploy'dan keyin alohida,
      orqaga-mos (cookie YOKI body qabul qilish) qilinishi tavsiya etiladi.
- [x] Migratsiya rollback / backward-compat siyosati (B6) — `migrations/MIGRATIONS.md`
- [ ] Redis prod persistence (AOF/RDB) — compose'da `--appendonly yes` bor;
      prod'da tasdiqlansin

## 🟡 Infratuzilma

- [ ] **CD yo'q (I5)** — deploy hali qo'lda (`DEPLOY-CHECKLIST.md`). Deploy
      maqsadi (I1) bo'lmaguncha o'rnatib bo'lmaydi.
- [ ] Compose gigiyenasi: resurs limitlari, log rotatsiyasi, image tag'larini
      pin qilish (healthcheck'lar ✅ bajarildi)
- [ ] **TURN TLS/443 (I4)** — ATAYLAB kechiktirilgan: sslip.io muhitida 443
      Caddy'da band. Yoqish tartibi `deploy/server/livekit.yaml.example` da.
      Deploy'dan keyin sinaladi.
- [x] **Runbook (I6)** — `deploy/server/RUNBOOK.md` (deploy/rollback/incident/
      backup/kuzatuv + B5 Redis siyosati)
- [ ] Masshtab (1000+ ishtirokchi) — talablar `DEPLOYMENT.md` §6 da
      (PgBouncer, multi-node LiveKit, Redis HA)

## 🔵 Mobil

- [ ] **Real domen** — hozir default staging (`sslip.io`). URL sozlanadigan
      (`-PdarslyApiUrl`), lekin domen hali yo'q
- [ ] **Nativ WebRTC yiqilishi** — `libjingle_peerconnection_so.so` · SIGABRT,
      tarmoq almashuvida (qurilma sinovida 3 takrordan 1 tasida). Telemetriya
      ulandi (Sentry NDK); **chastota ma'lum bo'lgach** `livekit-android`
      yangilash yoki `disconnect()` xulqini o'zgartirish hal qilinadi
- [ ] Ikonka — hozirgisi ishlaydigan brend variant; dizayn jamoasidan yakuniysi (M6)
- [~] **Play Store majburiylari (M5):** ✅ **akkaunt o'chirish** bajarildi
      (`DELETE /users/me` + Kabinet → Akkaunt, tasdiq bilan); ✅ **maxfiylik
      siyosati** asos matni (`docs/PRIVACY.md` — to'ldirish+ochiq URL kerak);
      ⏳ **Data Safety** formasi — Play Console'da qo'lda to'ldiriladi (tashqi).
- [ ] **`RoomViewModel.kt` (1333 qator) refaktori (M4)** — "god ViewModel"ni
      mas'uliyati aniq bo'laklarga. **ATAYLAB keyinга** — katta/xavfli refaktor,
      pilot qiymati past, jonli xona bilan alohida sinov talab qiladi.

### ✅ APK sayqal (2026-08-15, LAUNCH-PLAN Faza 2 — kod, sinov qoldi)
Auditning "zamonaviylashtirish" bandlari qo'llandi (`compileDebugKotlin` toza):
- **Haptika** — `ui/common/Interactions.kt` (`pressClickable`, `rememberHapticClick`).
  Ulangan: Darslar/Jadval FAB, xona ControlBar (mic/kamera/share/chiqish — yagona joy), login.
- **Skeleton (shimmer)** — `ui/common/Skeleton.kt`. Spinner o'rniga Darslar/Bildirishnoma/
  Arxiv/Jadval yuklanishida.
- **Bo'sh/xato illyustratsiyalari** — 6 ta line-art vector drawable (`res/drawable/il_*`)
  + `ui/common/StateViews.kt` (EmptyState/ErrorState, kirish fade+slide animatsiyasi).
- **`enableEdgeToEdge()`** — `MainActivity` (Android 15+ majburiy).
- **Login (M10)** — mint fon gradienti + "Kirish" tugmasida nur (glow qoidasiga mos).

## 🟢 Sifat

- [ ] **2-brauzerli E2E (F2)** (mentor + guest, real LiveKit) — media legi
      qamralmagan (CI'da real SFU + media qurilmasi yo'q). Infratuzilma talab qiladi.
- [x] `react-hooks/set-state-in-effect` (F4) — **0 ogohlantirish**. 5 ta joy
      (AppShell/useRoom/app/LiveRoom/WaitingRoom) tashqi-tizim sinxroni ekani
      izohlanib, per-line `eslint-disable` bilan ataylab bostirildi.

### ✅ Web sayqal / F-bandlar (2026-08-15)
- **F1** — Landing soxta statistika/testimonial olib tashlandi (halol framing).
- **F4** — 0 eslint ogohlantirish (yuqorida).
- **`prefers-reduced-motion`** — global CSS (harakatga sezgir foydalanuvchilar).
- **Skelet-yuklanish (web)** — `components/Skeleton.jsx` + shimmer CSS; Dashboard
  spinneri o'rniga (past-tarmoq uchun funksional). Test: 294/294 PASS.
- **F3** (287 inline style → tokenlar) — **ATAYLAB keyinга**: katta mexanik ish,
  pilot qiymati past. Yangi komponentlar (Skeleton) allaqachon token-klass bilan.
- **F5** — `docs/README.md` uzuq havolalari tozalandi; `LAUNCH-PLAN.md`, `PRIVACY.md`
  qo'shildi.
- [ ] 3 pilot ustoz bilan 90 daqiqalik haqiqiy dars (R1 darvozasi)

---

## Ataylab qilinmaydigan

| Nima | Sabab |
|---|---|
| Mehmon ovoz berishida `identity` dedup | Har join yangi anonim identity — bu anonim mehmon modelining tabiiy natijasi. Hisobsiz hal bo'lmaydi. DB darajasida bitta identity = bitta ovoz kafolatlangan |
| `onCleared` da sessiyani to'xtatish (mobil) | Fon rejimida darsni davom ettirishni buzardi. To'g'ri ilgak — `onTaskRemoved` (bajarildi) |

## Tarix

Yopilgan ishlar git tarixida. Eng yiriklari:

- **2026-07-28** — 7 agentli production auditining barcha kod bandlari
  (C-1, C-2, M1–M20 + minorlar). Batafsil: shu sanadagi commitlar
- **2026-07-25/26** — mobil ilova (mentor, Android) qurildi va qurilmada sinaldi
- **2026-07-24** — MVP tugadi, load test (25/25 ishtirokchi, 184 ms)
- **2026-07-23** — senior code-review tuzatishlari (`CLAUDE.md` da qisqacha)
