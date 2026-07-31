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
- [ ] **DB backup** — hozir RPO ∞. `pg_dump` cron + MinIO'ga yuklash.
      ⚠️ Tiklashni **bir marta sinab ko'rish** shart: sinalmagan backup — backup emas
- [ ] **A0.1 — 2 brauzer + real kamera bilan uchdan-uchga video sinovi**
      (odam bilan; avtomatlashtirib bo'lmaydi)

## 🟠 Xavfsizlik / ishonchlilik

- [ ] **Sessiya bekor qilish Redis uzilganda fail-open** — Redis yiqilsa bekor
      qilingan token yana ishlaydi. Fail-closed'ga o'tish butun kirishni
      bloklaydi, shuning uchun qaror talab qiladi (deqradatsiya siyosati)
- [ ] **Login'da account-lockout yo'q** — faqat IP rate-limit. Bitta hisobga
      taqsimlangan parol tanlash himoyasiz (join-parolda bu allaqachon
      ikki darajali, `joinlink.go` naqshiga qara)
- [ ] `RevokeAllUserSessions` — teskari indeks yo'q, SCAN qoladi. Hozir
      pipeline bilan ~100× tezlashtirilgan; ~10⁶ kalitdan keyin qayta ko'riladi.
      Migratsiya rejasi `jwt.go` izohida (Redis Cluster hash-tag bilan birga)
- [ ] **Refresh token `localStorage`da** — TTL 30→14 kun qisqartirildi, lekin
      XSS oynasi yopilmadi. To'liq yechim: HttpOnly cookie (mobil API bilan
      birga ko'rib chiqilsin — u Keystore ishlatadi)
- [ ] Migratsiya rollback / backward-compat siyosati yo'q
- [ ] Redis prod persistence (AOF/RDB) yoqilmagan

## 🟡 Infratuzilma

- [ ] **CD yo'q** — deploy hali qo'lda (`DEPLOY-CHECKLIST.md`)
- [ ] Compose gigiyenasi: resurs limitlari, log rotatsiyasi, image tag'larini
      pin qilish (healthcheck'lar ✅ bajarildi)
- [ ] **TURN TLS/443** — ATAYLAB kechiktirilgan: sslip.io muhitida 443 Caddy'da
      band. Yoqish tartibi `deploy/server/livekit.yaml.example` da
- [ ] Runbook: tiklash, deploy, rollback, incident tartibi
- [ ] Masshtab (1000+ ishtirokchi) — talablar `DEPLOYMENT.md` §6 da
      (PgBouncer, multi-node LiveKit, Redis HA)

## 🔵 Mobil

- [ ] **Real domen** — hozir default staging (`sslip.io`). URL sozlanadigan
      (`-PdarslyApiUrl`), lekin domen hali yo'q
- [ ] **Nativ WebRTC yiqilishi** — `libjingle_peerconnection_so.so` · SIGABRT,
      tarmoq almashuvida (qurilma sinovida 3 takrordan 1 tasida). Telemetriya
      ulandi (Sentry NDK); **chastota ma'lum bo'lgach** `livekit-android`
      yangilash yoki `disconnect()` xulqini o'zgartirish hal qilinadi
- [ ] Ikonka — hozirgisi ishlaydigan brend variant; dizayn jamoasidan yakuniysi
- [ ] Play Store majburiylari: maxfiylik siyosati, Data Safety, **akkaunt o'chirish**

## 🟢 Sifat

- [ ] **2-brauzerli E2E** (mentor + guest, real LiveKit) — hozirgi suite
      mehmon zanjirini xona chegarasigacha qoplaydi; media legi qamralmagan
      (CI'da real SFU + media qurilmasi yo'q)
- [ ] `react-hooks/set-state-in-effect` — 6 ogohlantirish. Hammasi TASHQI
      holatni React holatiga ko'chirish; xonaning nozik joylariga tegadi,
      alohida ko'rib chiqilsin (`frontend/eslint.config.js` izohiga qara)
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
