# Darsly / Jonly — QA (Python) test rejasi

> **Maqsad:** butun mahsulotni (backend API, web frontend, mobil) tashqaridan,
> mijoz ko'zi bilan sinovdan o'tkazuvchi **Python** test batareyasi. Git'da turadi,
> GitHub Actions'da **jadval bo'yicha avtomatik** ishlaydi va yiqilsa **darhol xabar** yuboradi.
>
> Holat: **BACKEND + WEB/MOBIL KONTRAKT + E2E YOZILDI VA YASHIL (2026-09-30).**
> Qarorlar 13-bo'limda. Jonli local backendga qarshi butun suite: **195 passed + 13 skipped**
> (skip = LiveKit SFU kerak bo'lgan room-token yo'llari — `requires_livekit` bilan TOZA skip),
> **25 test fayli**. Qamrov: backend 18 domen (5-bo'lim), web kontrakt, mobil kontrakt, E2E oqim.
> **CI (P2) yozildi:** `.github/workflows/qa.yml` — push/PR/kunlik CI runnerda ephemeral
> backend (docker-compose.dev.yml + `go run`) ko'tarib butun Python suite'ni yugurtiradi
> (LiveKit testlari avto-skip). Telegram alert SHARTLI (`QA_TG_BOT_TOKEN`/`QA_TG_CHAT_ID`
> sirlari qo'shilsa yoqiladi). Prod-monitor (P6) job'i deploy bo'lgach yoqiladi (izohli shablon).
> Keyingi: Telegram sirlarini qo'shish + prod deploy → prod-monitor, keyin Espresso/Appium (P7).
> Muallif: QA. Sana: 2026-09-21 (reja) · 2026-09-30 (resurslar + backend + web/mobil + e2e + CI).

> ### ✅ P1 backend qamrovi (2026-09-30) — jonli backendda yashil
> `backend/` (20 fayl): health_ops, auth, appconfig, users, lessons, joinlink,
> room_token, waitingroom, roomstate, chat, polls, recording, archive, notifications,
> blocklist, telegram, webhooks, ws, rbac, ratelimit.
> - Har domen: ijobiy (2xx + `schemas` kontrakt), auth (401), RBAC (403), validatsiya (400),
>   IDOR/ownership (begona mentor → 403/404), konflikt (409) — mos joyda.
> - **Webhook imzo** (kritik): to'g'ri imzo → 200, buzilgan tana / noto'g'ri sekret / imzosiz
>   → 401 (LiveKit webhook JWT `lib/livekit_sign.py` bilan — SFU KERAK EMAS).
> - **LiveKit chegarasi:** host-token/admit/vote/roomstate-ijobiy room-token talab qiladi →
>   LiveKit ishlaganda ishlaydi. Darsni haqiqiy **`live`** qilish (admit-all empty→200,
>   recording start→201) real LiveKit **ishtirokchisini** talab qiladi — bu black-box HTTP
>   chegarasi (mavjud Go `tests/load/` va qo'lda qoplanadi; GAP sifatida hujjatlashtirildi).

> ### 📦 Resurs holati (2026-09-30) — test yozishga TAYYOR
> Backend/mobil audit tugagach test yozuvchi shu asboblarni tayyor topadi:
> - **`lib/client.py`** — httpx wrapper: `{data}`/`{code,message}` konverti, `expect(status)`.
> - **`lib/auth.py`** — seed-admin login, `create_user(role=...)` (ochiq registratsiya yopiq muhitga moslashgan).
> - **`lib/factories.py`** — `Factory`: user/dars yaratadi va teardown'da **avto-tozalaydi** (yetim ma'lumot yo'q).
> - **`lib/schemas.py`** — barcha entity uchun kontrakt-validatorlar (`validate(data, LESSON)`, `list_of(...)`); `additionalProperties` ochiq (backend maydon qo'shsa sinmaydi, olib tashlasa ushlaydi).
> - **`lib/ws.py`** — WebSocket ulanish + `recv_json` (`?token=` / `?request_id=`).
> - **`lib/rooms.py`** — host token / joinlink preview+join / admit (guest room-token oqimi).
> - **`conftest.py`** — fixture'lar: `client`, `admin`, `user`, `mentor`, `as_user`, `as_mentor`, `factory`, `lesson`, `ws_url`, `qa_profile`; backend o'chiq bo'lsa aniq sabab bilan **skip**; **prod-monitor profilida destruktiv testlar avto-skip**.
> - **`requirements.txt` + `.venv`** — pytest, httpx, jsonschema, websockets, pytest-html/xdist/rerunfailures o'rnatildi va tekshirildi (`pytest --collect-only` toza, harness import ✓).
> - **`.env.example` / `.gitignore` / `pytest.ini`** (markerlar: smoke/destructive/mentor/ws/slow) tayyor.
>
> **Qolgani (backend/mobil tayyor bo'lgach):** 5-bo'limdagi endpoint inventari bo'yicha test
> modullarini yozish (P1→), so'ng `qa.yml` CI + cron monitoring (P2/P6). Bular ATAYLAB
> kutilyapti — kontrakt audit paytida o'zgarishi mumkin, shuning uchun test yozish keyinga.

> ### ⚡ Tez boshlash (keyin, davom etilganda)
> ```bash
> cd backend && docker compose -f docker-compose.dev.yml up -d && go run ./cmd   # backend
> cd QA && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt     # QA muhiti
> cp .env.example .env                                                            # sozlash
> QA_BASE_URL=http://localhost:8087 .venv/bin/python -m pytest -v                 # ishga tushirish
> ```

---

## 0. Bu qatlam NEGA kerak (mavjud testlardan farqi)

Loyihada allaqachon kuchli test bazasi bor va uni **takrorlamaymiz**:

| Qatlam | Bor narsa | Turi |
|--------|-----------|------|
| Backend | 75 fayl / ~497 funksiya (`internal/apitests` real Postgres'ga, unit) | oq-quti, kod ichida, Go |
| Frontend | 28 vitest + 3 Playwright (MSW mock) | komponent + mock E2E, JS |
| Mobil | 535 JVM unit test (parser/repo/pure-logic) | oq-quti, JVM |

**Bu testlar bir kamchilikni qoplamaydi:** ular **jonli, ishlab turgan tizimni** (deploy
qilingan server, real Postgres+Redis+MinIO+LiveKit, real tarmoq) tashqaridan tekshirmaydi.
Backend apitests real DB'ga tegadi, lekin CI konteynerida — deploy qilingan `sslip.io`
serveriga emas. Frontend/mobil testlari esa API'ni **mock** qiladi.

Python QA qatlamining vazifasi aynan shu bo'shliq:

1. **Black-box API kontrakt testi** — har bir HTTP endpoint jonli serverga real so'rov bilan; status, JSON shakli, auth, RBAC, validatsiya, xato formati.
2. **Kontrakt regressiyasi** — backend web/mobil kutgan javob shaklini buzmaganini kafolatlash (frontend `api/*.jsx` va mobil `DarslyApi.kt` kutgan maydonlar).
3. **E2E flow** — register → dars → joinlink → waiting room → recording → notification, boshdan-oxir, HTTP darajasida.
4. **Synthetic monitoring** — jadval bo'yicha (cron) jonli serverni tekshirib turish: "hozir prod ishlayaptimi?"; yiqilsa xabar.

---

## 1. Qamrov chegarasi — Python NIMANI testlaydi, NIMANI yo'q

Halol bo'lish uchun (senior QA sifatida) har qatlam uchun to'g'ri asbob:

| Sinov turi | Asbob | Python? | Izoh |
|------------|-------|:-------:|------|
| REST API (barcha endpoint) | pytest + httpx | ✅ | Asosiy deliverable |
| API kontrakt (web/mobil shakli) | pytest + jsonschema | ✅ | Client kutgan maydonlarni tekshiradi |
| E2E flow (HTTP) | pytest | ✅ | Ko'p-endpointli oqim |
| WebSocket (ws, waitingroom) | pytest + websockets | ✅ | admit → guest token push |
| Web brauzer E2E | Playwright **for Python** | ✅ | Real brauzer; mavjud JS Playwright bilan bir-birini to'ldiradi |
| Rate-limit / negativ / xavfsizlik | pytest | ✅ | 401/403/409/429, IDOR, RBAC |
| Yuklama (smoke perf) | pytest + (locust ixtiyoriy) | ⚠️ | Yengil; og'ir LiveKit yuklamasi `tests/load/` Go'da qoladi |
| **Mobil UI (Compose)** | Espresso/Appium | ❌ | Python bilan **emas** — bo'shliq sifatida hujjatlashtiriladi |
| **LiveKit media leg** (real kamera/SFU) | Go SDK / qo'lda | ❌ | HTTP orqali emas; mavjud `tests/load/` va qo'lda |

> **Muhim:** "mobil test" bu yerda mobil **UI**ni haydash emas (Python qila olmaydi) —
> mobil ilova bog'liq bo'lgan **backend endpoint'lar kontraktini** tekshirish. Mobil UI
> uchun alohida Espresso/Appium bosqichi kerak (hozir yo'q — 6-bo'limda gap bor).

---

## 2. Papka tuzilmasi

```
QA/
├── README.md                  # shu reja
├── pyproject.toml             # pytest, httpx, playwright, ... (yoki requirements.txt)
├── pytest.ini / conftest.py   # global fixture'lar (base_url, tokenlar, tozalash)
├── .env.example               # QA_BASE_URL, QA_WS_URL, test hisob ma'lumotlari
├── lib/                       # umumiy yordamchilar (mijozlar takrorlamasin)
│   ├── client.py              # httpx wrapper: auth, refresh, envelope ochish
│   ├── auth.py                # register/login helper, rol berish
│   ├── factories.py           # test ma'lumot yaratish (lesson, user) + cleanup
│   ├── schemas.py             # jsonschema/pydantic modellar (javob shakli)
│   └── ws.py                  # WebSocket yordamchi
│
├── backend/                   # ⭐ ASOSIY — har endpoint black-box
│   ├── test_health_ops.py     # /health /ready /metrics /swagger
│   ├── test_auth.py           # register/login/refresh/logout/me/forgot/reset
│   ├── test_users.py          # CRUD, IDOR, admin-only, parol
│   ├── test_lessons.py        # dars CRUD, ownership
│   ├── test_joinlink.py       # slug preview/join, passcode, lock
│   ├── test_room_token.py     # host/participant token, end
│   ├── test_waitingroom.py    # admit/reject/status/admit-all + WS
│   ├── test_roomstate.py      # hand/reaction/state (room-token)
│   ├── test_chat.py           # host + room chat, upload, transcript, delete
│   ├── test_polls.py          # create/publish/close/vote/results
│   ├── test_recording.py      # start/stop/download/local/restore
│   ├── test_archive.py        # lesson archive
│   ├── test_notifications.py  # list/unread/read/read-all
│   ├── test_blocklist.py      # list/unblock, ban scope
│   ├── test_telegram.py       # link/status/unlink
│   ├── test_appconfig.py      # min_version
│   ├── test_webhooks.py       # livekit imzo (ijobiy/salbiy)
│   ├── test_ws.py             # /ws, /ws/waitingroom auth
│   ├── test_rbac.py           # student mentor route'larida 403; admin cheklovi
│   └── test_ratelimit.py      # 429, timing, enumeration himoyasi
│
├── frontend/                  # web mijoz nuqtai nazari
│   ├── contract/              # web kutgan JSON shaklini tekshirish (pytest)
│   │   └── test_web_contract.py
│   └── e2e/                   # Playwright-Python (real brauzer, jonli/staging)
│       ├── test_auth_flow.py
│       ├── test_lesson_create.py
│       ├── test_join_flow.py
│       └── test_waitingroom_flow.py
│
├── mobile/                    # mobil mijoz nuqtai nazari (faqat KONTRAKT)
│   ├── test_mobile_contract.py   # DarslyApi.kt kutgan endpoint+shakl
│   ├── test_mentor_only.py       # admin login rad etilishi (backend tomondan)
│   └── GAPS.md                   # Espresso/Appium bo'shlig'i hujjati
│
└── e2e/                       # cross-cutting to'liq oqim (HTTP+WS)
    ├── test_full_lesson_flow.py  # register→dars→join→WS admit→chat→recording→end
    └── test_guest_flow.py        # joinlink→waiting→admit→room-token bilan chat/vote
```

---

## 3. Texnologiya tanlovi

- **Python 3.12+**, `pytest` — test runner.
- **httpx** — sinxron/asinxron HTTP mijoz (retry, timeout, cookie yo'q — JWT bilan).
- **pydantic v2** yoki **jsonschema** — javob shaklini validatsiya (kontrakt).
- **websockets** — `/ws` va `/ws/waitingroom` uchun.
- **playwright** (Python) — `frontend/e2e/` real brauzer.
- **pytest-xdist** — parallel; **pytest-rerunfailures** — flaky retry (monitoring uchun).
- **allure-pytest** yoki **pytest-html** — HTML hisobot (artefakt).
- Ixtiyoriy: **locust** — yengil yuklama; **tenacity** — polling.

Barchasi `QA/pyproject.toml`da pin qilinadi (ci.yml'dagi golangci-lint kabi — `latest` emas).

---

## 4. Muhit va test-ma'lumot strategiyasi (⚠️ eng muhim qaror)

Black-box test **ma'lumot yaratadi va o'chiradi** — prod'ni ifloslantirmasligi shart.
Uch profil (`QA_BASE_URL` env bilan tanlanadi):

| Profil | Target | Qamrov | Ma'lumot |
|--------|--------|--------|----------|
| **local** | `http://localhost:8087` | to'liq (destruktiv ham) | erkin yaratish/o'chirish |
| **staging/ephemeral** | CI'da `docker-compose` bilan ko'tarilgan yangi backend | to'liq E2E + destruktiv | har run toza DB |
| **prod-monitor** | `https://app.169.58.104.245.sslip.io` (yoki `jonly.uz`) | faqat **smoke** (o'qish + o'z-ustidan tozalanadigan) | maxsus QA hisob, self-cleanup |

> **⚠️ Muhitdan aniqlangan haqiqat (2026-09-21):** local `.env`da
> `ALLOW_OPEN_REGISTRATION=false` — ya'ni `POST /auth/register` **403** qaytaradi
> ("Open registration is disabled"). Shuning uchun test foydalanuvchilari **ochiq
> register orqali EMAS**, balki **seed admin** (`admin@darsly.uz` / `Admin12345`,
> `.env`da `SEED_ADMIN_*`) bilan **admin API** (`POST /api/v1/users` — `{email,
> password, full_name, role}`) orqali yaratiladi. `lib/auth.py` aynan shunday
> ishlaydi (`admin_client()` → `create_user(role=...)`). Register endpointining
> o'zi esa `test_auth.py`da **403 qaytishi** (yoki bayroq yoqilса 201) sifatida
> tekshiriladi — muhitga moslashuvchan.

Qoidalar:
1. **Prod'ga destruktiv test yo'q.** Prod'da faqat: `/health`, `/ready`, login (maxsus QA hisob), `/auth/me`, dars yaratish→**darhol o'chirish**, joinlink preview. Boshqa foydalanuvchi ma'lumotiga tegmaydi.
2. **To'liq E2E faqat local/ephemeral'da.** CI'da har run yangi Postgres+Redis+MinIO ko'tariladi (ci.yml'dagi `services:` naqshi) → izsiz.
3. **Test hisoblari:** `factories.py` register orqali yaratadi; mentor huquqi kerak bo'lsa — local/ephemeral'da to'g'ridan-to'g'ri DB UPDATE (apitests kabi) yoki seed admin (`admin@darsly.uz`). Prod'da oldindan yaratilgan `qa+monitor@...` hisob (parol GitHub Secret'da).
4. **Har test o'zidan keyin tozalaydi** (`fixture` teardown): yaratilgan lesson/user `DELETE`. E2E'da unikal email (`qa+{uuid}@darsly.uz`).
5. **Sirlar** faqat GitHub Secrets / `.env` (git'ga tushmaydi) — `.env.example` shablon bo'ladi.

---

## 5. Backend API test inventari (endpoint × tekshiruv)

Har endpoint uchun kamida: **ijobiy** (2xx + shakl), **auth** (401), **RBAC** (403 mos rolda),
**validatsiya** (400), va tegishli bo'lsa **konflikt** (409) / **topilmadi** (404).

### 5.1 Ops / infratuzilma
- `GET /health` → 200 `{status:ok}`; `GET /ready` → 200/503; `GET /metrics` (prod'da token); `/swagger` prod'da yo'q.

### 5.2 Auth (`/api/v1/auth`)
- register: 201 TokenPair; 409 mavjud email; 400 zaif parol; open-registration o'chsa 403.
- login: 200; 401 xato parol; **429** rate-limit; **enumeration**: yo'q user ham bir xil vaqt (timing).
- refresh: 200 rotatsiya; 401 revoked/eski token.
- logout: 204; keyin refresh 401 (sessiya o'chgan).
- forgot/reset: har doim 204 (enumeration yo'q); reset xato token → 400/401.
- `/auth/me`: 200; 401 tokensiz.

### 5.3 Users (`/api/v1/users`)
- me: GET/PUT (rol o'zgartirilsa e'tiborga olinmaydi — **privilege escalation** testi), parol o'zgartirish, self-delete, request-deletion + cancel.
- admin CRUD: list/create/update/activate/deactivate/delete → student/mentor'da **403**, admin'da 2xx.
- `GET /users/:id`: **IDOR** — o'zganikini so'raganda 403 (admin bo'lmasa).
- `PUT /users/:id/password`: mentor ruxsati, sessiyalar bekor bo'lishi.

### 5.4 Lessons (`/api/v1/lessons`) — mentor
- CRUD; **ownership**: boshqa mentor darsiga 403/404; student'da 403; validatsiya (title, duration 5–1440).

### 5.5 Room / video
- `POST /:id/token` (host) → 200 RoomToken (`ws_url`, `token`); egasi bo'lmasa 403.
- `POST /:id/end` → 204; participants list; mute-all/mute/remove(scope lesson|mentor)/allow-speak/revoke-speak; hands lower/lower-all.

### 5.6 Waiting room
- `GET /:id/waitingroom` (mentor); `admit` → 200 token; `reject` → 204; **admit ikki marta → 409** (atomik TransitionFromPending); `admit-all` partial; `GET /waitingroom/:id/status` public.

### 5.7 Room-state (room-token, guest)
- hand/reaction/state: to'g'ri token → 204/200; **tokensiz 401**; noto'g'ri/boshqa room token → 401; validatsiya (emoji 1–16).

### 5.8 Chat
- host: history (cursor, limit≤50), send (1–2000), upload (≤20MB, tur cheklovi, rate-limit), transcript (txt/html), delete.
- room-token: send/history/upload; tokensiz 401; private DM (`to`).

### 5.9 Polls
- create (2–10 variant), publish (mentor_only bo'lsa 400), close, vote (guest room-token), results (published bo'lgunча student 403, host doim 200).

### 5.10 Recording
- start (Egress)/stop/download (presigned URL ishlashi), list, local-start→upload-url→complete oqimi, restore (202).

### 5.11 Archive / Telegram / Notifications / Blocklist / App-config / Webhook
- archive: video+chat+material bitta javobda, `Cache-Control: no-store`.
- telegram: status/link (kod)/unlink.
- notifications: list (unread filtr), unread-count, read, read-all.
- blocklist: list, unblock; ban scope `mentor` doimiy.
- app-config: min_version.
- **webhook**: to'g'ri imzo → 200; **noto'g'ri imzo → 401** (kritik xavfsizlik testi); egress_ended → recording ready.

### 5.12 WebSocket
- `/ws?token=`: JWT bilan ulanadi, tokensiz 401; ulanганda pending snapshot keladi.
- `/ws/waitingroom?request_id=`: mavjud request bilan ulanadi; **admit → guest real-time token oladi** (race: guest kech ulansa ham Redis'dan oladi).

### 5.13 RBAC / rate-limit matritsasi
- `test_rbac.py`: har mentor-only route'ga student token bilan → 403 (jadval-asosli, parametrlangan).
- `test_ratelimit.py`: auth 15/40 limit, 429; monitoring uchun bu testlar prod'da **o'chiriladi** (limitni buzmaslik uchun).

---

## 6. Frontend qatlami

**6.1 Kontrakt (`frontend/contract/`)** — Python:
- Frontend `src/api/*.jsx` chaqiradigan har endpoint javobida web kutgan **maydonlar** borligini tekshiradi (masalan `joinlink` javobida `next_step`, `roomToken.ws_url`, `unread-count` da `count`). Backend shaklni buzsa web sindan oldin ushlaydi.

**6.2 Brauzer E2E (`frontend/e2e/`)** — Playwright-Python, jonli/staging'ga:
- register → dashboard; login → xato parol; dars yaratish → `/app/lesson/:id/room` ga o'tish (LiveKit media legigacha, media'ni test qilmaydi); joinlink preview + name/passcode; waiting room admit→room o'tishi.
- Bu mavjud JS Playwright'ni (MSW mock) **to'ldiradi**: JS mock ustida deterministik, bu esa **real backend** ustida. Ikkalasi maqsadi har xil.

---

## 7. Mobil qatlami

Python mobil UI'ni hayday olmaydi, shuning uchun:
- **7.1 Kontrakt (`mobile/test_mobile_contract.py`)** — `DarslyApi.kt` va `AuthRefreshApi.kt` kutgan har endpoint+javob shaklini jonli backendga tekshiradi (envelope `{data}`, `ListEnvelope.total`, 204 typeless, `ws_url`).
- **7.2 Mentor-only (`test_mentor_only.py`)** — admin login → backend `/auth/me` role='admin' bo'lsa mobil rad etadi; **backend tomondan** admin/student cheklovi to'g'ri ekanini tasdiqlaydi (mobil kodida student guard yo'qligi — QA bo'shlig'i sifatida GAPS.md'da).
- **7.3 GAPS.md** — Espresso/Appium bilan qoplanishi kerak bo'lgan (hozir yo'q) UI/instrumentation testlar ro'yxati: login oqimi, room ekrani, recording xizmati, forced-update gate.

---

## 8. Cross-cutting E2E (`QA/e2e/`)
1. **To'liq dars oqimi:** mentor register → login → dars yaratish → host token → guest joinlink orqali kiradi → (waiting room bo'lsa) mentor WS orqali so'rov oladi → admit → guest token oladi → guest room chatga yozadi / poll ovoz beradi → mentor recording start/stop → mentor `/end` → notification tekshiruvi. Faqat local/ephemeral.
2. **Guest oqimi:** joinlink preview → join (passcode) → waiting → admit (WS) → room-token bilan hand/reaction/chat/vote.

---

## 9. CI/CD va jadval (avtomatik ishlash)

Yangi workflow: **`.github/workflows/qa.yml`** (mavjud `ci.yml`'ga tegmaymiz).

Uch xil ishga tushish:

1. **Push/PR'da** → `local/ephemeral` profil: CI'da `services:` bilan Postgres+Redis+MinIO ko'taril­adi, backend `go run` bilan ishga tushadi, Python QA to'liq batareya ishlaydi. Regressiyani merge'dan oldin ushlaydi.
2. **Jadval (cron) bo'yicha** → `prod-monitor` profil: `schedule: cron` (masalan har 30 daq yoki `*/15`) deploy qilingan serverga faqat smoke. Bu — synthetic monitoring.
3. **`workflow_dispatch`** → qo'lda, profil tanlab.

```yaml
# eskiz
on:
  push:
  pull_request:
  schedule:
    - cron: "*/30 * * * *"   # prod smoke har 30 daqiqa
  workflow_dispatch:
    inputs: { profile: { default: "prod-monitor" } }
```

**Xabar (alerting):** yiqilsa —
- **Telegram** (loyihada allaqachon bot infratuzilmasi bor — eng mos) yoki
- GitHub Issue avtomatik ochish, yoki
- Actions email/Slack.

`schedule` jobida `if: failure()` bilan bot orqali xabar yuboriladi (chat_id, token — Secret).

---

## 10. Hisobot (reporting)
- Har run: `pytest-html`/allure HTML hisobot → Actions artefakt (ci.yml'dagi playwright-report kabi).
- Monitoring run natijasi: qisqa xulosa (o'tdi/yiqildi + qaysi endpoint) Telegram xabarida.
- Ixtiyoriy: kunlik "uptime" jamlanmasi.

---

## 11. Bosqichlar (rollout)

| Bosqich | Ish | Natija |
|---------|-----|--------|
| **P0** ✅ | `lib/` skeleti + `conftest.py` + `test_health_ops.py` + `test_auth.py` + local profil | birinchi yashil test lokalda |
| **P1** ✅ | `backend/` to'liq inventar (5-bo'lim) + RBAC/negativ | 18 domen, 162 pass + 8 LiveKit-skip |
| **P3** ✅ | `e2e/` cross-cutting + WS testlari | to'liq oqim (real-time reject WS yashil) |
| **P4** ✅ | `frontend/contract/test_web_contract.py` | web mijoz kutgan maydonlar (NOSTANDART bayroqlar) |
| **P5** ✅ | `mobile/` kontrakt + `test_mentor_only.py` + GAPS.md | mobil mijoz kontrakti |
| **P2** ✅ | `.github/workflows/qa.yml` — push/PR/kunlik ephemeral backend | CI'da avtomatik (ci.yml tegilmadi) |
| **P6** | `schedule` cron + Telegram alert | synthetic monitoring jonli |
| **P7** *(kelajak)* | Espresso/Appium mobil UI, locust yuklama | UI/perf bo'shliqlari |

---

## 12. Qabul qilingan qarorlar (2026-09-21)

Foydalanuvchi bilan kelishilgan yakuniy qarorlar:

1. **Boshlash profili:** ✅ **local** (`http://localhost:8087`) — hozir shundan. Prod-monitor keyingi bosqich.
2. **Alert kanali:** ✅ **Telegram** (loyihada bot infratuzilmasi bor — eng arzon/qulay).
3. **Cron chastotasi:** ✅ **24 soatda bir** (kunlik) — `cron: "0 2 * * *"` (ci.yml bilan bir xil vaqt).
4. **QA hisob:** ✅ **`qa+monitor@darsly.uz`** yaratiladi (parol GitHub Secret'da). Local'da seed admin (`admin@darsly.uz`) ishlatiladi.
5. **Frontend E2E:** ✅ **Playwright QO'SHILMAYDI.** Sabab: og'ir, flaky va frontendda allaqachon JS Playwright bor (takror bo'lardi). Frontend qamrovi = **API kontrakt testi** (web kutgan JSON shaklini tekshirish). Mobil ham xuddi shunday — faqat kontrakt.

> Natijada `frontend/e2e/` (Playwright-Python) **rejadan chiqarildi**; `frontend/contract/`
> qoladi. 1-bo'limdagi jadvalda "Web brauzer E2E" endi mavjud JS Playwright zimmasida.

---

## 13. Ilova — aniqlangan kontrakt (kod'dan tasdiqlangan)

Testlar aynan shu shakllarga tayanadi (`backend/api/http_status/response.go` + `internal/entity`):

**Javob konverti:**
- Success (200/201/202): `{"data": <obj>}`
- List: `{"data": [...], "total", "page", "limit", "total_pages"}`  (bo'sh ro'yxat `[]`, `null` emas)
- Xato: `{"code": "BAD_REQUEST|UNAUTHORIZED|FORBIDDEN|...", "message": "<xavfsiz matn>"}`
- 204: tanasiz

**Auth (`internal/entity/auth.go`):**
- `TokenPair` = `{"access_token", "refresh_token"}`
- `RegisterReq` = `{"full_name"(2-255), "email", "password"(8-72)}`
- `LoginReq` = `{"email", "password"}`
- Ochiq registratsiya: `GET /api/v1/app-config` → `data.allow_open_registration` (hozir `false`)

**Admin user yaratish (`internal/entity/user.go`):**
- `CreateUserReq` = `{"email", "password"(8-72), "full_name"(2-255), "role": admin|mentor|student|guest}`
- Faqat **admin** roli chaqira oladi (`POST /api/v1/users` → 201 `{data: User}`)

**Seed admin (local `.env`):** `admin@darsly.uz` / `Admin12345` (startupda avtomatik yaratiladi).

**Ops endpointlari (`/api/v1` ostida EMAS):** `/health` (200 `{status:ok}`), `/ready` (200/503),
`/metrics` (non-prod ochiq), `/swagger/*` (faqat non-prod).

> Davom etilganda **P1** (backend to'liq inventar, 5-bo'lim) dan boshlanadi — P0 skeleti tayyor.
