# To'liq-flow E2E test rejasi va qamrovi (mobil + web → backend → DB)

> **Maqsad:** loyihadagi HAR action uchun uchdan-uchga test — mijoz (brauzer/ilova)
> → backend → **Postgres/Redis (DBgacha)**. Mavjud mock/unit testlardan farqi: bu
> HAQIQIY stack bilan ishlaydi.
>
> Holat: **ENGINE TAYYOR (2026-10-02).** Web live-E2E harness qurildi va 8 flow yashil.
> Qolgan flow'lar shu naqsh bo'yicha qo'shiladi.

## Arxitektura
| Qatlam | Asbob | Qayerda ishlaydi |
|--------|-------|------------------|
| **Web full-flow** | Playwright → Vite(proxy) → real backend → DB | GitHub runner YOKI server (arzon, tez) |
| **Mobil full-flow** | Android instrumented (Compose + Hilt) → real backend → DB | **Self-hosted server** (emulator/qurilma) — push'да |
| **Media (6 mobil + web video/whiteboard)** | QO'LDA (idrok sifati) | foydalanuvchi, har push |

- Web: `frontend/playwright.live.config.js` + `frontend/e2e-live/` · ishga tushirish: `npm run test:e2e:live` (backend ishlab turishi kerak)
- Fixtures: `frontend/e2e-live/fixtures.js` — admin API orqali test hisob yaratish/tozalash, UI login, `mentorPage`/`studentAccount`/`admin`.
- ⚠️ «bitta akkaunt = bitta sessiya»: UI sessiyasi faolligida o'sha akkaunt bilan API login QILMA (UI'ni bekor qiladi). DB tasdiqlashni UI-reflektsiya yoki boshqa hisob bilan qil.

---

## WEB qamrovi (Playwright live) — **22 test YASHIL** (8 spec, ~32s)

> ⚠️ E2E backend **yuqori auth rate-limit** bilan ishga tushirilsin (ko'p test-login):
> `RATE_LIMIT_LOGIN_IP_EMAIL/EMAIL/IP` + `RATE_LIMIT_REFRESH_*` + `RATE_LIMIT_AUTH_IP`
> katta qiymat. (Rate-limit xulqi QA `test_ratelimit.py` da ALOHIDA sinaladi.)

### ✅ Tayyor (22 test)
- **Auth** (auth.spec): login · noto'g'ri parol · logout
- **Darslar** (lessons + lessons-manage): yaratish→xona→ro'yxat(DB) · bo'sh-nom · tahrirlash→DB · o'chirish→DB · rejalashtirish→Jadval(DB)
- **Akkaunt** (account + misc): profil ism→DB→UI · parol o'zgartirish→DB(yangi 200/eski 401) · bildirishnoma · blocklist · yozuvlar · Telegram-yashirin(o'chiq)
- **Admin** (admin-users): userlar sahifa · yaratish→DB→o'chirish · mentor RBAC 403
- **Guest** (guest-flow): havola preview(DB) · kutish-xona belgisi · parol maydoni(has_passcode)
- **Arxiv/chat** (archive-chat): chat-tarixi modal · dars arxivi sahifasi

### ⬜ Qolgan = JONLI XONA ichki (live dars + ko'pincha 2-ishtirokchi kerak)
Bular LiveKit + `live` dars (host kirgан) talab qiladi — sintetik ishtirokchi (Go LiveKit SDK) bilan avtomatlashtiriladi, yoki qo'lда:
- Xona ichki chat yuborish · so'rovnoma (yaratish→ovoz→natija) · moderatsiya (mute/kick/ban/speak) · qo'l/reaksiya · kutish-xona admit/admit-all/reject (guest bilan juft) · havola nusxalash (clipboard)

### 🙋 Qo'lда (web media)
- Jonli xona video/audio render · ekran ulashish · whiteboard chizish — idrok sifati

---

## MOBIL qamrovi (Android instrumented) — ~48 action

### ⬜ Skelet + flow'lar (keyingi ish)
- `app/src/androidTest/` (hozir 0) — Hilt test runner + Compose test rule + ephemeral backend helper (web fixtures'ga o'xshash)
- Flow'lar web mentor-qismiga oyna: login · dars CRUD · kutish xonasi · chat · poll · moderatsiya · bildirishnoma · blocklist · arxiv · profil · telegram
- Ishga tushirish: ulangan qurilma yoki emulyator + ishlayotgan backend

### 🙋 Qo'lда (6 mobil media — foydalanuvchi har push)
- Kamera on/off · mikrofon · kamera almashtirish · ekran ulashish · ikki-tomon video/audio

> Media MEXANIKASINI (track publish/subscribe, kadr keldi) keyinchalik sintetik
> ishtirokchi (Go LiveKit SDK, `backend/tests/load/`) bilan avtomatlashtirsa bo'ladi;
> SIFAT idroki esa odamда qoladi.

---

## CI (server kelganда)
```
Har push:
├── GitHub runner (tez):   backend unit+apitest(→DB) · QA Python(→DB) · mobil unit · web Playwright LIVE(→DB) · lint
└── Self-hosted server:    mobil instrumented E2E(→DB) · LiveKit real-media mexanikasi
Foydalanuvchi qo'lда:       6 mobil media + web media/whiteboard idroki
```
- Serverni **GitHub self-hosted runner** qilib ro'yxatга olish (`runs-on: [self-hosted, android]`), KVM+emulator+Docker+to'liq stack.
- Web live-E2E GitHub runner'да ham ishlaydi (emulyator shart emas) — `qa.yml`ga qo'shsa bo'ladi (ephemeral backend bilan).
