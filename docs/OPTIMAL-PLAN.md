# Darsly — mavjud funksiyalarni ideal holatga keltirish rejasi

> Sana: 2026-07-27 · Qamrov: backend + web frontend + mobil
> Maqsad: **yangi funksiya qo'shmaslik**. Bor funksiyalarni Zoom darajasida (yoki undan yaxshiroq)
> ishlaydigan holatga keltirish + ikki talab: (1) o'quvchi gapira olishi, (2) qo'l/reaksiya ekran
> ulashish paytida ham ustozga ko'rinishi.
> Har bir band `fayl:satr` dalili bilan. Baholar: **mehnat** = bitta dasturchi uchun taxminiy vaqt.

---

## 0. Bir sahifalik xulosa

Uch qatlamning holati bir xil emas:

| Qatlam | Baho | Bir jumlada |
|---|---|---|
| Backend (Go) | **8/10** | Arxitektura toza, poyga holatlari to'g'ri yechilgan. Kamchiliklar — nuqtaviy, tizimli emas. |
| Mobil (Kotlin) | **8/10** | Media sozlamalari professional, 165 test. Kamchilik — dars ichidagi ijtimoiy qatlam (chat/qo'l/reaksiya) umuman yo'q. |
| Web (React) | **5/10** | Eng zaif bo'g'in: 0 test, media default sozlamada, o'lik kod, holat faqat brauzer xotirasida. |

**Yagona ildiz sabab.** Dars ichidagi holat — qo'l ko'tarish, chat, reaksiya — **serverda mavjud emas**.
Hammasi LiveKit data-channel'da uchadi. Shundan kelib chiqadigan oqibatlar zanjiri:

- kech kirgan hech narsani ko'rmaydi (qo'l ham, chat ham);
- qayta ulanish holatni o'chiradi;
- ustoz qo'lni tushira olmaydi, navbat tartibi yo'q;
- o'quvchi chati yozuvga ham, tarixga ham tushmaydi;
- moderatsiya va rate-limit qilib bo'lmaydi (spam vektori ochiq);
- mobil ilova bu oqimga umuman ulanmagan.

Shu bitta bo'shliqni yopish (`roomstate` domeni) — quyidagi 4 muammoni bir vaqtda hal qiladi va
ikkala yangi talab uchun ham poydevor bo'ladi.

**Tavsiya etilgan tartib:** F1 (server-truth) → F2 (web media + PiP) → F3 (mobil data-channel) →
F4 (barqarorlik/xavfsizlik qarzlari) → F5 (test qarzi).

---

## 1. Ikki yangi talab — texnik yechim

### 1.1 O'quvchi gapira olishi

**Hozir nima bor (ishlaydi):**
- `livekit/token.go:19` — participant tokenida `CanPublish=false` (webinar modeli).
- `room.go:SetSpeakPermission` → `participant.go:SetParticipantPublish` — token qayta chiqarmasdan
  huquq beradi. Manbalar aniq ro'yxatlangan (`studentPublishSources` = CAMERA+MICROPHONE,
  SCREEN_SHARE ataylab yo'q) — **to'g'ri va puxta**.
- `POST /lessons/:id/participants/:identity/allow-speak` va `revoke-speak` — bor.
- Web: `ParticipantsPanel.jsx:100` da tugma; `LiveRoom.jsx:277` `ParticipantPermissionsChanged`
  hodisasini ushlab toast ko'rsatadi; `Controls.jsx:36` `canPublish` bo'yicha mikrofon ochiladi.

**Ya'ni asosiy mexanizm allaqachon ishlaydi.** Yetishmayotgani — **oqim** (UX + davomiylik):

| # | Bo'shliq | Dalil | Yechim |
|---|---|---|---|
| S-1 | O'quvchining "so'rovi" (qo'l) o'tkinchi — ustoz panelni ochmasa yoki kech ulansa ko'rmaydi | `LiveRoom.jsx:409` | Qo'l → server holati (§1.3) |
| S-2 | Navbat tartibi yo'q — kim birinchi ko'targani bilinmaydi | `raisedHands` = `Set` | Server holatida `raised_at` bilan tartiblangan ro'yxat |
| S-3 | Ruxsat berilgach qo'l avtomatik tushmaydi → ro'yxat axlatga to'ladi | — | `allow-speak` → server qo'lni tushiradi (bitta amalda) |
| S-4 | Ustoz qo'lni majburan tushira olmaydi | — | `POST /lessons/:id/hands/:identity/lower` + "Hammasini tushirish" |
| S-5 | Ruxsat berilganda o'quvchida faqat toast chiqadi — tugmani o'zi topishi kerak | `LiveRoom.jsx:283` | Toast'da **"Mikrofonni yoqish"** tugmasi (bir bosishda efirga) |
| S-6 | Ruxsat bekor qilinganda o'quvchi bilmaydi (jim o'chadi) | — | `canPublish` true→false da toast + trek to'xtaganini ko'rsatish |
| S-7 | Dars tugab qayta ulanilsa ruxsat yo'qoladi (LiveKit holati xonada) | — | Ruxsatlar Redis'da (`room:speakers:<lessonID>`), qayta ulanishda tiklanadi |
| S-8 | Mobil ilovada ishtirokchi boshqaruvi umuman yo'q | `ui/room/` da panel yo'q | Mobil ishtirokchilar paneli (§4) |

> **Diqqat (xavfsizlik):** `allow-speak` kamera huquqini ham beradi. Dars uchun ko'pincha faqat
> mikrofon kerak. `SetSpeakPermission`ga `sources []string` parametri qo'shilsin
> (default: faqat MICROPHONE, "kamera bilan" — alohida tanlov).

### 1.2 Qo'l va reaksiya — ekran ulashish paytida ko'rinishi

Muammoning mohiyati platformaga qarab **butunlay boshqacha**:

**A) Web — ustoz brauzerdan boshqa oynaga o'tadi.**
Sahifa ichida hammasi joyida: `ReactionsOverlay` va `RoomRail` `.stage` ustida turadi
(`LiveRoom.jsx:528-529`), ekran ulashilganda ham ko'rinadi. Muammo — **brauzer ko'rinmay qoladi**.

*Yechim: Document Picture-in-Picture API* (`documentPictureInPicture.requestWindow()`, Chrome 116+).
Har doim ustida turadigan kichik oyna: ko'tarilgan qo'llar ro'yxati + uchayotgan reaksiyalar +
mikrofon/ekran to'xtatish tugmalari. Ekran ulashish boshlanganda **avtomatik ochiladi**, tugaganda yopiladi.
- Qo'llab-quvvatlamaydigan brauzer (Firefox/Safari) → zaxira: `Notification` API (ruxsat bilan) +
  sahifaga qaytganda to'plangan hodisalar ko'rsatiladi.
- **Muhim tafsilot:** PiP oynasiga ko'chirilgan DOM tugunlari asosiy hujjatning CSS'ini olmaydi —
  `styles.css` ni PiP hujjatiga nusxalash kerak (`adoptedStyleSheets`).

**B) Mobil — ustoz butunlay boshqa ilovada.**
`MediaProjection` ekranni **bor holicha** yozib beradi: ekranga chiqarilgan har qanday overlay
(bubble) **o'quvchilarga ham ko'rinadi**. Shuning uchun tartib:

1. **Birinchi qadam (arzon, ruxsatsiz, ulashuvga tushmaydi):** foreground bildirishnoma matnini
   jonli yangilash — `LessonNotifications.build(ctx, text)` allaqachon `text` parametrini qabul qiladi
   (`service/LessonNotifications.kt:41`), lekin hech kim uzatmayapti. Matn: `✋ 3 · 👍 Ali`.
   Qo'l ko'tarilganda bir marta **vibratsiya** (bildirishnoma `IMPORTANCE_LOW` — ovoz chiqarmaydi).
2. **Ikkinchi qadam (ixtiyoriy):** `SYSTEM_ALERT_WINDOW` bilan kichik, chekkadagi bubble.
   Ulashuvda ko'rinishini hisobga olib: minimal o'lcham, ismsiz (faqat son), ustoz uni yashira olsin.
3. Ilovaga qaytganda — to'plangan hodisalar `RoomScreen`da.

**Ikkala platformada ham dastlabki shart bitta:** hodisa manbai ishonchli bo'lishi kerak, ya'ni §1.3.

### 1.3 Poydevor: `roomstate` domeni (server-truth)

CLAUDE.md dagi 12-qadamli naqsh bo'yicha yangi domen. **Yangi mahsulot funksiyasi emas** —
bor funksiyalarning ostiga haqiqat manbaini qo'yish.

**Nima serverda saqlanadi:**

| Ma'lumot | Joyi | Sabab |
|---|---|---|
| Ko'tarilgan qo'llar | Redis hash `room:hands:<lessonID>` (identity → `{name, raised_at}`), TTL 12s | Tez, tartiblanadi, dars tugagach o'z-o'zidan o'ladi. `HSet/HGetAll` allaqachon bor (`redis.go:32-33`) |
| So'zga ruxsat berilganlar | Redis set `room:speakers:<lessonID>` | Qayta ulanishda tiklash (S-7) |
| Chat (barcha ishtirokchi) | Postgres `chat_messages` (allaqachon bor) | Tarix + yozuv + moderatsiya |
| Reaksiyalar | **Saqlanmaydi** | O'tkinchi tabiat (Zoom ham saqlamaydi). Faqat relay + rate-limit |

**Transport.** Broadcast — LiveKit data-channel (`SendData`, `participant.go:13`), WebSocket emas:
guest'da JWT yo'q, LiveKit esa allaqachon ulangan va past kechikishli. Klient → server yo'nalishi
esa HTTP: guest'ni **room-token** bilan autentifikatsiya qilamiz — bu naqsh so'rovnomada allaqachon
ishlatilgan va sinovdan o'tgan (`v1/poll.go:VotePoll` + `livekit.VerifyToken` + `tokenRoom` tekshiruvi
`poll.go:74`). Ya'ni yangi autentifikatsiya mexanizmi ixtiro qilinmaydi.

**Yangi endpointlar:**

```
POST /api/v1/rooms/:lessonID/hand        {token, raised}      → 204   (ochiq, room-token)
POST /api/v1/rooms/:lessonID/reaction    {token, emoji}       → 204   (ochiq, rate-limit 1/2s)
GET  /api/v1/rooms/:lessonID/state?token=…                    → snapshot (qo'llar+speakerlar)
POST /api/v1/lessons/:id/hands/:identity/lower                → 204   (host, protected)
POST /api/v1/lessons/:id/hands/lower-all                      → 204   (host, protected)
POST /api/v1/rooms/:lessonID/chat        {token, body, to?}   → 201   (ochiq, room-token; to = DM)
GET  /api/v1/rooms/:lessonID/chat?token=…&before=…            → tarix (hammaga)
```

**Migratsiya `000011`:** `chat_messages.to_identity TEXT NULL` + qisman indeks
(`WHERE to_identity IS NOT NULL`). Shaxsiy xabar shu bitta ustun bilan yopiladi; `SendData`ga
`destination_identities` beriladi (LiveKit qo'llab-quvvatlaydi).

**Mehnat:** backend 2.5–3 kun · web 2 kun · mobil 1.5 kun.

---

## 2. Backend — hozirgi holat va ish ro'yxati

### 2.1 Kuchli tomonlar (tegilmasin)

- `waitingroom`: `TransitionFromPending` atomik (`WHERE status='pending'` + RowsAffected) — TOCTOU yo'q.
- `room.ensureRoomOnce`: `singleflight` + Redis flag — 500 ta parallel join'da bitta `CreateRoom`.
- `recording`: yozuv `track_published` webhook'ida boshlanadi (`webhook.go:44`) — egressning 5 daqiqalik
  kutish timeout'i chetlab o'tilgan; DB xatosida egress to'xtatiladi (yetim yo'q); `EndLesson` da
  tartib to'g'ri (avval yozuv, keyin xona).
- `token/jwt.go`: refresh rotatsiyasi CAS (Lua) + grace oynasi — mobil 4G uzilishida sessiya o'lmaydi.
- WS Hub: 256-bufer + sekin klientni tashlash (`websocket.go:233`) + Redis fan-out — ko'p instansga tayyor.
- Indekslar joyida: `chat_messages(lesson_id, created_at)`, qisman indekslar, trgm.

### 2.2 Qo'shilishi kerak

| # | Ish | Fayl | Mehnat |
|---|---|---|---|
| B-N1 | `roomstate` domeni (§1.3) — entity, redis repo, usecase, 5 handler, routelar, policy.csv | yangi `usecase/roomstate/` | 2 kun |
| B-N2 | Chat: room-token bilan ishtirokchi xabari + tarix hammaga + `to_identity` (DM) | `usecase/chat/`, `postgres/chat.go`, migration 000011 | 1 kun |
| B-N3 | `SetSpeakPermission(sources)` — default faqat mikrofon | `livekit/participant.go:22` | 2 soat |
| B-N4 | `allow-speak` qo'lni avtomatik tushirsin (bitta usecase amalida) | `usecase/room/room.go` | 1 soat |
| B-N5 | Ban ro'yxati: kick qilingan qaytib kira olmasin | `usecase/room`, `joinlink` | 3 soat |

### 2.3 Optimallashtirish / tuzatish

| # | Daraja | Muammo | Dalil | Mehnat |
|---|---|---|---|---|
| B-1 | 🔴 | `GET /polls/:id/results` **autentifikatsiyasiz ochiq** — poll ID bilan har kim natijani ko'radi | `router.go:210` | 40 daq |
| B-2 | 🔴 | `poll`/`chat` da `shared.ValidateID` yo'q → `"abc"` UUID ustuniga yetadi → **auth'siz 500 generatori** | `poll.go:47,68` | 30 daq |
| B-3 | 🔴 | Rol pasaytirilganda/user o'chirilganda sessiya tirik qoladi; `Rotate` rolni eski claim'dan meros qiladi | `user.go:119-131,244-257`, `jwt.go:230` | 1 kun |
| B-4 | 🟠 | `MuteAll` ishtirokchilar bo'ylab **ketma-ket** HTTP — 100 kishida ~100 chaqiruv, 10s timeout bilan qulash xavfi | `room.go:MuteAll` | 2 soat (errgroup, 8 parallel) |
| B-5 | 🟠 | Kick ≠ ban: token 6 soat yaroqli + `auto_create` → chiqarilgan qaytib kiradi va yopilgan xonani tiklaydi | `livekit.yaml` + token TTL | 3 soat |
| B-6 | 🟠 | Redis yiqilsa auth **fail-open** — bekor qilingan tokenlar qayta ishlaydi | `jwt.go:139-145` | 3 soat |
| B-7 | 🟠 | Join-parol lockout **butun dars uchun global** → istalgan kishi 5 ta noto'g'ri parol bilan hamma o'quvchini bloklaydi | `joinlink.go:19-24` | 2 soat (IP+slug kaliti) |
| B-8 | 🟠 | `EnsureForRoom` **har `track_published`da** DB so'rovi qiladi — 30 ta ishtirokchi = 60 so'rov | `webhook.go:44` → `recording.go:57` | 1 soat (Redis SetNX guard) |
| B-9 | 🟠 | `internal/worker` **0% test** — dedup/retry mantiqi sinovsiz | — | 1 kun |
| B-10 | 🟠 | SMTP `ctx`ni e'tiborsiz qoldiradi → worker abadiy osilishi mumkin; cheksiz requeue (poison message) | `email/email.go:105`, `worker/email.go:68` | 4 soat |
| B-11 | 🟠 | CI Postgres portini `5432`ga map qiladi, `testutil` `5442` kutadi → **50+ integratsiya testi jimgina skip** | `ci.yml` ↔ `testutil/pg.go:43` | 20 daq |
| B-12 | 🟡 | `refresh_tokens` jadvali cheksiz o'sadi (tozalash chaqirilmaydi) | `postgres/auth.go` | 2 soat |
| B-13 | 🟡 | `WithTx` — 0 ta chaqiruv joyi; `reminder.tick`da claim'dan keyin `Notify` yiqilsa eslatma yo'qoladi | `worker/reminder.go:50` | 3 soat |
| B-14 | 🟡 | Casbin `model.conf` nisbiy yo'l — noto'g'ri cwd'da hamma endpoint 503 | `pkg/casbin` | 1 soat |
| B-15 | 🟡 | `users` email qidiruvida to'liq skan (trgm indeks faqat `full_name`da) | `000002:22-24` | 30 daq |

---

## 3. Web frontend — hozirgi holat va ish ro'yxati

### 3.1 Kuchli tomonlar

- `Whiteboard.jsx` (768 qator): en-fit + vertikal scroll modeli, PDF fon, chunk'lab uzatish,
  kech kirganga snapshot, follow-presenter — dars stsenariysi uchun Zoom annotation'idan qulayroq.
- `api.jsx`: 401 → bitta parallel refresh (`refreshing` guard), xato kodlari o'zbekchaga map qilingan.
- `ws.js`: eksponensial backoff bilan avtomatik qayta ulanish.
- Dizayn tizimi to'liq tokenlashtirilgan (`styles.css`).

### 3.2 Qo'shilishi kerak

| # | Ish | Fayl | Mehnat |
|---|---|---|---|
| W-N1 | `useRoomState` hook: server snapshot (`GET /rooms/:id/state`) + data-channel merge; qo'l/chat shundan | yangi `src/livekit/useRoomState.js` | 1 kun |
| W-N2 | Guest chati: `POST /rooms/:id/chat` (room-token) + tarix hammaga + DM tanlovi | `panels/ChatPanel.jsx`, `api/chat.jsx` | 1 kun |
| W-N3 | **Document PiP oynasi** (§1.2A): qo'llar + reaksiyalar + mic/stop tugmalari; ekran ulashishda avtomatik | yangi `src/livekit/PresenterPip.jsx` | 1 kun |
| W-N4 | Qo'l navbati UI: tartiblangan ro'yxat, "tushirish", "hammasini tushirish", ruxsat berish bitta bosishda | `panels/ParticipantsPanel.jsx` | 4 soat |
| W-N5 | Ruxsat toast'ida **"Mikrofonni yoqish"** tugmasi (S-5) | `views/LiveRoom.jsx:283` | 1 soat |

### 3.3 Optimallashtirish / tuzatish

| # | Daraja | Muammo | Dalil | Mehnat |
|---|---|---|---|---|
| W-1 | 🔴 | **Yozib olish tugmasi o'lik** — `recording` va `onToggleRecord` proplari `Controls`ga uzatilmagan; `REC` indikatori hech qachon yonmaydi | `LiveRoom.jsx:554-568` ↔ `Controls.jsx:22` | 15 daq |
| W-2 | 🔴 | **Har LiveKit hodisasida butun daraxt re-render**: `bump()` → `RoomStage` (571 qator) + `Whiteboard` (768 qator) qayta chiziladi. `ActiveSpeakersChanged` ~0.5s da bir keladi → doskada chizayotganda seziladigan lag | `useRoom.js:11`, `Stage/RoomRail` `React.memo`siz | 1 kun |
| W-3 | 🔴 | Media sozlamalari default: ekran uchun simulcast qatlami yo'q, `degradationPreference` yo'q, kodek tanlanmagan, ekran audiosi yo'q — mobilda yechilgan muammo webda ochiq | `useRoom.js:19`, `Controls.jsx:70` ↔ `MediaTuning.kt` | 1 kun |
| W-4 | 🔴 | Aloqa indikatori **soxta** — `ConnectionQuality` hodisasi umuman ishlatilmagan, "Yaxshi" har doim yoziladi | `LiveRoom.jsx:491` | 3 soat |
| W-5 | 🟠 | Guest `disconnected` bo'lsa **darhol bosh sahifaga uloqtiriladi** va sessiya o'chadi — vaqtinchalik uzilishda ham | `LiveRoom.jsx:268` | 3 soat (15s grace) |
| W-6 | 🟠 | "Tejamkor rejim" yo'q — zaif tarmoqda o'quvchi videoni qo'lda o'chira olmaydi | — | 4 soat |
| W-7 | 🟠 | Data-channel'da rate-limit yo'q — bitta o'quvchi emoji/chat spam qilsa hammaning ekrani to'ladi | `messaging.js` | 2 soat |
| W-8 | 🟠 | **0 ta test** (MSW handler'lari tayyor turibdi) | `src/test/` | 2 kun |
| W-9 | 🟡 | `strokesRef` cheksiz o'sadi — 90 daqiqalik darsda xotira va snapshot og'irlashadi (kech kirganga yuborish sekinlashadi) | `LiveRoom.jsx:110` | 3 soat |
| W-10 | 🟡 | `roomParticipants(room)` har renderda yangi massiv yasaydi | `useRoom.js:90` | 1 soat |
| W-11 | 🟡 | `ws.js` backoff'da jitter yo'q (server qayta ko'tarilganda hamma bir vaqtda uriladi); token muddati tugasa 401 loop | `ws.js:36` | 2 soat |
| W-12 | 🟡 | Tokenlar `localStorage`da + CSP/HSTS yo'q → bitta XSS = 30 kunlik refresh o'g'irlanadi | `api.jsx:11`, `Caddyfile` | 4 soat |

---

## 4. Mobil — hozirgi holat va ish ro'yxati

### 4.1 Kuchli tomonlar

- `MediaTuning.kt`: ekran uchun simulcast qatlamlari qo'lda tanlangan, `MAINTAIN_RESOLUTION`,
  RED+DTX oshkora — **va bularning bari testlar bilan qotirilgan**. Web'dan bir bosh baland.
- `ScreenAudioPlan`/`ScreenAudioPolicy`/`ScreenAudioMixer`: mute semantikasi Zoom'nikiga moslangan,
  sof mantiq JVM testida sinaladi.
- `RoomUiState`: `ConnectionQualityChanged` (M16) ulangan — **webda yo'q bo'lgan narsa mobilda bor**.
- `LessonService` + `LessonSessionHolder`: fon rejimi, Android 14 tip qoidalari to'g'ri qo'llangan.
- 165 test, qurilmada 25 mezon tasdiqlangan.

### 4.2 Qo'shilishi kerak

| # | Ish | Fayl | Mehnat |
|---|---|---|---|
| M-N1 | `RoomEvent.DataReceived` ulanishi + sof parser (`RealtimeParser` uslubida, test bilan) | `ui/room/RoomViewModel.kt:349`, yangi `data/livekit/RoomDataParser.kt` | 4 soat |
| M-N2 | **Ekran ulashishda ko'rinish (§1.2B):** bildirishnoma matnini jonli yangilash + qo'l kelganda vibratsiya | `service/LessonNotifications.kt:41` (`text` parametri ishlatilmayapti) | 4 soat |
| M-N3 | Ilova ichida qo'l navbati + reaksiya overlay | `ui/room/RoomStage.kt` | 4 soat |
| M-N4 | Ishtirokchilar paneli: mute / chiqarish / **so'zga ruxsat berish** (hozir mobilda umuman yo'q) | yangi `ui/room/ParticipantsSheet.kt` | 1 kun |
| M-N5 | Chat (kamida o'qish + yuborish) | yangi `ui/room/ChatSheet.kt` | 1 kun |
| M-N6 | (Ixtiyoriy, 2-qadam) `SYSTEM_ALERT_WINDOW` bubble | `AndroidManifest.xml` + yangi servis | 1 kun |

### 4.3 Optimallashtirish / tuzatish

| # | Daraja | Muammo | Dalil | Mehnat |
|---|---|---|---|---|
| M-1 | 🔴 | **C-11 — tarmoq almashuvi (Wi-Fi↔LTE) yiqilgan** — qurilma sinovidagi yagona muvaffaqiyatsiz mezon | `MOBILE-DEVICE-TEST.md` | 1 kun |
| M-2 | 🔴 | Release build: **imzo yo'q, R8 o'chiq, ABI split yo'q, 66 MB** | `app/build.gradle.kts` | 4 soat |
| M-3 | 🟠 | Dars holati o'zgarishi push qilinmaydi — web'da dars yakunlansa telefondagi ro'yxat eskirgan qoladi | backend WS 2 hodisa bilan cheklangan | B-N1 bilan birga |
| M-4 | 🟡 | `RoomViewModel` 539 qator — hodisa boshqaruvi alohida sinfga ajratilsin | `ui/room/RoomViewModel.kt` | 4 soat |
| M-5 | 🟡 | `TODO(R1)`: `Room` obyekti `LessonSessionHolder`da, servisda emas — Activity o'lsa dars 100% kafolatlanmaydi | `service/LessonService.kt:30` | 1 kun |
| M-6 | 🟡 | `TODO(R2)`: DI qo'lda (`Net.kt:14`) | `data/api/Net.kt` | 4 soat |

---

## 5. Bajarish tartibi

| Faza | Mazmun | Natija | Mehnat |
|---|---|---|---|
| **F1** | B-N1…B-N5, W-N1, W-N2, W-N4, W-N5, M-N1 | Qo'l/chat/ruxsat **serverda**; kech kirgan hammasini ko'radi; o'quvchi to'liq gapira oladi; chat yozuvga tushadi; DM ishlaydi | ~5 kun |
| **F2** | W-1, W-2, W-3, W-4, W-N3 | Web media mobil darajasiga chiqadi; PiP bilan ekran ulashishda qo'l/reaksiya ko'rinadi; o'lik kod yo'q; doskada lag yo'q | ~4 kun |
| **F3** | M-N2, M-N3, M-N4, M-1 | Mobil ustoz ulashish paytida hammasini ko'radi va boshqara oladi; tarmoq almashuvi omon | ~3 kun |
| **F4** | B-1…B-8, W-5, W-6, W-7, M-2 | Xavfsizlik va barqarorlik qarzlari yopiladi | ~4 kun |
| **F5** | B-9…B-15, W-8…W-12, M-4…M-6 | Test qarzi va ichki sifat | ~5 kun |

**Jami ~21 ish kuni.** F1+F2+F3 (12 kun) tugagach — sanab o'tilgan funksiyalarning barchasi
Zoom bilan bir xil ishonchlilikda ishlaydi; F4+F5 — uni shu holatda ushlab turish uchun.

---

## 6. Ataylab qilinmaydigan ishlar

- Breakout rooms, virtual fon, AI shovqin bosish, transkript — **yangi funksiya**, hozirgi maqsaddan tashqarida.
- Reaksiyalarni DB'ga saqlash — o'tkinchi tabiat, Zoom ham saqlamaydi; faqat relay + rate-limit.
- Chat uchun alohida WebSocket kanali — LiveKit data-channel allaqachon ulangan va guest'da JWT yo'q;
  ikkinchi kanal ortiqcha murakkablik bo'lardi.

---

## 7. BAJARILGAN ISHLAR (jonli holat)

> Bu bo'lim ish jarayonida yangilanadi. Har band tekshirilgan: build + test yashil.

### ✅ F2 — Web (build ✅ · 33 birlik test ✅)

| Band | Holat | Izoh |
|---|---|---|
| W-1 O'lik yozib olish tugmasi | ✅ | Proplar uzatildi; REC holati serverdan (`listRecordings`) o'qiladi |
| W-2 Re-render bo'roni | ✅ | Ishtirokchilar **immutable snapshot + imzo**; `Stage`/`RoomRail`/`Controls`/`ParticipantTile`/`Whiteboard`/`ChatPanel`/`ParticipantsPanel` — `memo` |
| W-3 Media sozlamalari | ✅ | `mediaTuning.js` — mobil `MediaTuning.kt` bilan bir xil qatlamlar; ekran audiosi + `contentHint: 'text'` |
| W-4 Soxta aloqa indikatori | ✅ | `ConnectionQuality` ulandi; "Ulandi" (o'lchanmagan) ≠ "Yaxshi" (o'lchangan) |
| W-5 Guest uzilishda uloqtirilishi | ✅ | `DisconnectReason` bo'yicha yakuniy/vaqtinchalik ajratildi; sessiya saqlanadi, 5 s da avtomatik qayta ulanish |
| W-6 Tejamkor rejim | ✅ | Kirish kameralari uziladi, ovoz+ekran qoladi |
| W-7 Spam himoyasi | ✅ | Klient throttle (reaksiya 1.5 s, chat 0.4 s) + server cheklovi |
| W-8 Testlar | ⏳ qisman | **0 → 33** birlik test (`roomLogic.js` sof mantiqqa ajratildi). Komponent/e2e testlari qoldi |
| W-N4 Qo'l navbati UI | ✅ | Raqamlangan navbat, "tushirish", "hammasini tushirish" |
| W-N5 Ruxsat UX | ✅ | Banner + "Mikrofonni yoqish" tugmasi (toast emas) |

### ✅ F1 — Server-haqiqat

**`roomstate` domeni** (12 birlik + 4 API test):
- Qo'l ko'tarish Redis'da; **navbat tartibi serverda** hisoblanadi; takroriy bosish o'rinni buzmaydi
- `/rooms/:lessonID/{hand,reaction,state}` (ochiq — room-token) + `/lessons/:id/hands/{lower,lower-all}` (host + RBAC)
- Reaksiya uchun server tomonda cheklov (2 s da 1)
- `room.Hands` (DIP): **ruxsat berilganda qo'l avtomatik tushadi**, dars tugaganda tozalanadi
- `RoomToken.lesson_id` qo'shildi (guest'ga kerak edi)

**Chat** (10 birlik + 1 haqiqiy-SQL test):
- **O'quvchi xabari ham saqlanadi** (avval faqat host); tarix **hammaga** yuklanadi
- **Shaxsiy xabar** (`to_identity`, migratsiya 000011): yetkazish `destination_identities` bilan — begona klientga **umuman bormaydi**
- Ko'rinuvchanlik filtri **SQL'da** (repo qatlamida), usecase'da emas
- Xona yo'lida 5 s / 5 xabar cheklovi

**Yo'l-yo'lakay yopilgan xavfsizlik teshiklari:**
- 🔴 B-1 `GET /polls/:id/results` butunlay ochiq edi → room-token talab qiladi
- 🔴 B-2 `poll`da `ValidateID` yo'q edi → autentifikatsiyasiz 500 generatori yopildi
- Status kodlari izchillashtirildi: token yo'q = **401** (avval 400/401 aralash edi)

**Test infratuzilmasi tuzatildi:** `FakeCache` hash'lari no-op edi (yolg'on qamrov) → haqiqiy xulq; `FakeChatRepo` endi ko'rinuvchanlik filtrini ham qo'llaydi.

### ✅ F2 — Suzuvchi oyna (Document PiP)

Ustoz ekranini ulashib brauzerdan chiqib ketganda ham qo'l navbati va reaksiyalarni
ko'radi. `PresenterPip.jsx`: har doim ustida turadigan kichik oyna, React portali orqali.
- Ekran ulashish boshlanishi bilan **o'zi ochiladi**; ustoz yopsa qayta ochilmaydi,
  lekin **keyingi ulashishda yana ochiladi** — holat mashinasi `nextPipState` (6 test).
- Auto-ochilish rad etilsa (Document PiP foydalanuvchi harakatini talab qiladi va
  `await setScreenShareEnabled` uni sarflagan bo'lishi mumkin) — sarlavhada
  **"Suzuvchi oyna"** tugmasi chiqadi (bir bosish = kafolatlangan harakat).
- Uslublar PiP hujjatiga nusxalanadi (u asosiy sahifa CSS'ini meros olmaydi).
- Ichida: navbat + oxirgi reaksiyalar + mikrofon/to'xtatish + qo'lni tushirish.

### ✅ F3 — Mobil (329 test, 0 xato)

| Band | Holat | Izoh |
|---|---|---|
| M-N1 `DataReceived` | ✅ | `RoomDataParser` — sof, JVM testida (10 test). Notanish tur `null` → oldinga moslik |
| M-N2 Bildirishnomada signal | ✅ | `LessonNotifications.updateSignals` — `✋ 3 · 👍 Ali`; yangi qo'lda qisqa titrash |
| M-N3 Ilova ichida ko'rinish | ✅ | `SignalsOverlay` — sahna ustida navbat + reaksiyalar |
| Kech ulangan ustoz | ✅ | `GET /rooms/:id/state` bilan mavjud navbat tiklanadi (`loadRoomState`) |
| Navbat tartibi | ✅ | `HandQueue` — server `at` bo'yicha, uch klientda bir xil (9 test) |
| M-N4 Ishtirokchilar paneli | ⏳ | mute/chiqarish/so'zga ruxsat — hali yo'q |
| M-N5 Chat | ⏳ | mobil chat oynasi — hali yo'q |
| M-1 Tarmoq almashuvi (C-11) | ⏳ | qurilma sinovidagi yagona yiqilgan mezon |

> **Nega overlay emas, bildirishnoma:** `MediaProjection` ekranni bor holicha yozadi —
> suzuvchi oyna (`SYSTEM_ALERT_WINDOW`) o'quvchilarga ham ko'rinardi. Bildirishnoma esa
> ulashuvga tushmaydi va qo'shimcha ruxsat talab qilmaydi.

### ✅ Suzuvchi oynada CHAT + ulashish ramkasi

- PiP oynasiga chat qo'shildi: oxirgi xabarlar, shaxsiy belgisi va **tez javob maydoni**.
  Ustoz ekran ulashib boshqa ilovada bo'lganda o'quvchi savoliga to'g'ridan-to'g'ri javob bera oladi.
- **Ulashish ramkasi** (Zoom'dagi kabi): chiziqli, pulsatsiyalanuvchi chegara —
  webda xona oynasi ichida + PiP oynasi atrofida, mobilda sahna atrofida.
  > Brauzer sahifasi OS ekrani atrofiga chiza olmaydi (Zoom buni native ilova
  > bo'lgani uchun qiladi). Ustoz boshqa ilovada bo'lganda bu vazifani **PiP
  > oynasining ramkasi** bajaradi — u har doim ustida turadi.

### ✅ F4 — Xavfsizlik qarzlari yopildi

| # | Muammo | Yechim |
|---|---|---|
| B-3 | Rol pasaytirilganda/o'chirilganda sessiya tirik qolardi | Rol o'zgarsa va o'chirilsa **barcha sessiyalar bekor qilinadi** (3 test) |
| B-4 | `MuteAll` ketma-ket HTTP (100 kishi = 100 chaqiruv) | `errgroup` bilan **8 parallel**; bitta xato qolganini to'xtatmaydi |
| B-5 | Kick ≠ ban: chiqarilgan qaytib kirardi | Redis ban (6 soat) + `ParticipantToken` tekshiradi. **Avval ban, keyin uzish** (2 test) |
| B-6 | Redis yiqilsa auth **fail-open** | **Cheklangan fail-open**: uzilishda faqat 5 daqiqadan yangi tokenlar qabul qilinadi |

### ✅ Mobil — ishtirokchilar paneli va chat

Telefondan dars o'tayotgan ustoz endi webga muhtoj emas:
- **Ishtirokchilar paneli**: qo'l navbati (tepada, raqamlangan) + so'zga ruxsat, mute,
  mute-all, chiqarish, qo'lni tushirish
- **Chat paneli**: tarix + yuborish + shaxsiy xabar belgisi
- Boshqaruv panelida **nishonlar**: qo'l ko'targanlar soni va o'qilmagan xabarlar

### ✅ C-11 — tarmoq almashuvi (kod tomoni)

`LessonReconnectPolicy`: dastlabki uch urinish ~1.7 soniya ichida (tarmoq
almashuvida yangi interfeys odatda darhol tayyor), keyin backoff o'sadi
(muammo interfeys emas, kanal bo'lsa tez urinish zarar keltiradi). Umumiy oyna 60 s.

> ⚠️ **Cheklov:** bu 7 ta test siyosat MANTIG'INI qotiradi. C-11 mezonining o'zi —
> real qurilmada Wi-Fi↔LTE almashuvi — faqat qurilmada tasdiqlanadi va u hali
> bajarilmagan.

### ✅ W-8 — komponent testlari

Web: **0 → 69 test**. `happy-dom` + Testing Library (`jsdom` shu Node versiyasida
`ERR_REQUIRE_ESM` beradi). Qoplangan: `ChatPanel` (10), `ParticipantsPanel` (10),
`PresenterPanel` (10) + sof mantiq (39).

> Qolgani: Playwright e2e CI'da chaqirilmaydi (alohida infra ishi).

### ✅ F-1 — "yashil, lekin hech nima sinalmagan" holati YOPILDI

Ildiz sabab: CI Postgres'ni 5432'ga map qiladi, `testutil` esa 5442 kutadi (dev compose
porti). Ulanish yiqilardi → `t.Skip` → paket baribir `ok`. 50+ integratsiya testi
CI'da hech qachon ishlamagan.

Ikki tomondan tuzatildi:
1. **CI env:** `TEST_DB_PORT: 5432` (portlar mos keladi).
2. **`testutil.SkipOrFail`** — `TEST_REQUIRE_INFRA=1` bo'lganda skip **TAQIQLANADI**.
   Endi port yana adashsa CI qizil bo'ladi. Barcha infra-bog'liq skip'lar (Postgres,
   Redis — 8 joy) shu funksiyadan o'tadi.

Tekshirildi: infra bilan 21 paket ✅; `TEST_DB_PORT=9999` bilan CI rejimida **yiqiladi**
(aniq xabar bilan) — ya'ni mexanizm ishlaydi.

### ✅ CI to'liq qamrovga keltirildi

Avval CI faqat backend testlarini (aslida skip qilingan) va frontend **build**ini
bajarardi. Endi 4 job:

| Job | Nima qiladi |
|---|---|
| `backend` | build · vet · golangci-lint · `-race` test (**infra majburiy**) · coverage |
| `frontend` | `npm test` (69 birlik+komponent) · build |
| `e2e` | Playwright smoke (MSW mock ustida — backend shart emas) + yiqilganda hisobot artefakti |
| `mobile` | `testDebugUnitTest` (344 test) · `lintDebug` |

### ✅ C-11 — ikkala gipoteza bajarildi

`docs/MOBILE-DEVICE-TEST.md §5b` da yozilgan ikki taxmin tekshirildi, ikkalasi ham to'g'ri chiqdi:

1. **`RoomOptions.reconnectPolicy`** SDK'da bor (artefakt ochib tasdiqlandi) →
   `LessonReconnectPolicy`: dastlabki uch urinish ~1.7 s ichida, keyin backoff (7 test).
2. **`NetworkCallback`** → `NetworkMonitor` + `NetworkSwitchPolicy`: qaror TRANSPORT
   o'zgarishiga bog'landi (bir Wi-Fi'dan boshqasiga o'tishda uzib-ulash zarar keltiradi),
   Wi-Fi↔LTE da majburiy `disconnect()`+`connect()` (8 test). UI'da "Mobil internetga
   o'tildi" yozuvi — u aloqa yozuvidan ustun turadi.

> ⚠️ **Qoldiq:** bu 15 test qarorni qotiradi, qurilmadagi haqiqiy almashuvni emas.
> C-11 mezonining o'zi LTE qamrovi bor joyda sinalishi kerak — protokol
> `MOBILE-DEVICE-TEST.md §5c` da (6 qadam, kutilgan vaqtlar bilan).

### ⏭ Qolgan ishlar

1. **Qurilmada C-11 sinovi** — kod va protokol tayyor, LTE bor joyda bajarish kerak
2. **F5:** `worker` 0% qamrov, SMTP deadline, `refresh_tokens` tozalash
