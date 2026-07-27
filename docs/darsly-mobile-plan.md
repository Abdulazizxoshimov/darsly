# Darsly Mobile (Mentor) — To'liq Reja

> **Maqsad:** Ustoz planshetdan (Android) darsni olib boradi va **ekranini ulashadi**.
> O'quvchilar hech narsa o'rnatmaydi — ular avvalgidek **web brauzerda** ko'radi.
> Natija: side-load qilinadigan **APK** (`Darsly Mentor`).

Holat: **reja** (2026-07-25). Kod hali yozilmagan.

---

## 1. Nega native ilova kerak (muammoning ildizi)

Ekran ulashish brauzerda `navigator.mediaDevices.getDisplayMedia()` orqali ishlaydi.
**Android Chrome va iPadOS Safari bu API'ni bermaydi** — bu platforma cheklovi, bizning
kodimizdagi xato emas. Frontend buni allaqachon bilib, ogohlantirish chiqaradi:

```js
// frontend/src/livekit/Controls.jsx:61-72
const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent)
if (!screenOn && (isMobile || !navigator.mediaDevices?.getDisplayMedia)) {
  return toast.error("Ekran ulashish telefon/planshetda ishlamaydi — kompyuterda oching yoki 'Doska'dan foydalaning")
}
```

Android'da ekran olishning **yagona** yo'li — OS'ning `MediaProjection` API'si, u esa
faqat native ilovaga beriladi. Shuning uchun **mentor uchun native Android ilova** —
to'g'ri va yagona yechim. Bu "workaround" emas, sanoat standarti (Zoom, Meet, Teams
mobil ilovalari ham xuddi shunday qiladi).

**Muhim yaxshi xabar:** server tomonda deyarli hech narsa o'zgarmaydi. LiveKit ekran
ulashishni oddiy ikkinchi video-trek (`Track.Source.SCREEN_SHARE`) sifatida qabul qiladi,
va web frontend uni **allaqachon** ko'rsatishga tayyor:

```js
// frontend/src/livekit/Stage.jsx:11-20 — screen share avtomatik "speaker" ko'rinishga o'tadi
const screenSharer = participants.find((p) => p.getTrackPublication(Track.Source.ScreenShare)...)
```

Ya'ni: **planshetdan ekran ulashilishi bilan barcha web-o'quvchilar uni ko'radi — frontendga o'zgarish shart emas.**

---

## 2. Qamrov (scope)

### v1 ga KIRADI — "Darsly Mentor" Android ilovasi
| # | Imkoniyat | Izoh |
|---|-----------|------|
| 1 | Login / logout / avtomatik token yangilash | mavjud `/auth/*` |
| 2 | Mening darslarim ro'yxati + yangi dars yaratish | `/lessons` |
| 3 | Darsga host bo'lib kirish | `POST /lessons/:id/token` |
| 4 | **Ekran ulashish (MediaProjection)** | ⭐ ilovaning asosiy sababi |
| 5 | Kamera + mikrofon (old/orqa kamera almashtirish) | LiveKit Android SDK |
| 6 | Kutish xonasi: real-time so'rov → Qabul / Rad | WS `/api/v1/ws?token=` |
| 7 | Ishtirokchilar: mute, chiqarish, gapirishga ruxsat | `/lessons/:id/participants/*` |
| 8 | Chat (web bilan to'liq mos) | data-channel + `/lessons/:id/chat` |
| 9 | Reaksiya va "qo'l ko'tarish"ni ko'rish | data-channel |
| 10 | Yozib olish: start / stop | `/lessons/:id/recording/*` |
| 11 | Darsni yakunlash | `POST /lessons/:id/end` |

### v1 ga KIRMAYDI (ataylab)
- **O'quvchi ilovasi** — o'quvchilar web'da qoladi (asosiy talab shu). Keyingi bosqich.
- **iOS/iPad** — iOS'da ekran ulashish butunlay boshqa mexanizm (ReplayKit Broadcast
  Upload Extension, alohida process, App Store majburiy). Alohida bosqich (§12).
- **Doska (whiteboard)** — ekran ulashish uni almashtiradi. Planshetda qalam bilan
  chizish keyingi versiyada juda mos keladi, lekin v1 ni cho'zmaydi.
- **So'rovnoma (poll) yaratish** — ko'rish bor, yaratish v1.1.
- **Play Store** — v1 to'g'ridan-to'g'ri APK (side-load).

---

## 3. Texnologiya tanlovi

**Tavsiya: Native Android — Kotlin + Jetpack Compose + `io.livekit:livekit-android`**

| Mezon | Kotlin (native) | Flutter | React Native |
|---|---|---|---|
| Ekran ulashish ishonchliligi | ⭐ **Eng yuqori** — LiveKit Android SDK asl manba | Yaxshi (wrapper) | O'rtacha (wrapper, Expo dev-client + config plugin kerak) |
| Android 14/15 foreground-service qoidalari | To'liq nazorat | Wrapper kutish kerak | Wrapper kutish kerak |
| Mavjud React kodini qayta ishlatish | Yo'q (~300 satr API qatlami qayta yoziladi) | Yo'q | Qisman (api/, messaging.js) |
| iOS uchun keyin | 0% | ~70% (lekin ekran ulashish baribir alohida) | ~70% (xuddi shunday) |
| Build/CI murakkabligi | Eng past (Gradle → APK) | O'rtacha | Eng yuqori (JS bundle + native) |
| 90 daqiqalik darsda batareya/barqarorlik | Eng yaxshi | Yaxshi | O'rtacha |

**Nega Kotlin:**
1. Ilovaning **butun mavjud bo'lish sababi** — ekran ulashish. U eng ishonchli
   bo'lishi shart. LiveKit Android SDK — Flutter/RN pluginlari o'rab olgan asl SDK;
   uni to'g'ridan-to'g'ri ishlatish orada bir qatlam xatolik kamaytiradi.
2. Cross-platform'ning asosiy foydasi — iOS. Lekin iOS'da ekran ulashish kodi
   **baribir** noldan (ReplayKit extension) yoziladi, ya'ni Flutter/RN bu yerda
   deyarli hech narsa tejamaydi.
3. Ilova kichik: 7 ta ekran, mentor-only. Qayta ishlatilmaydigan JS ~300 satr.
4. Android 14 (API 34) va 15 (API 35) `mediaProjection` foreground-service qoidalarini
   qattiqlashtirdi — bu yerda wrapper yangilanishini kutib qolmaslik muhim.

> Agar keyinchalik **o'quvchi ilovasi** ham kerak bo'lsa (unda ekran ulashish shart emas,
> faqat ko'rish), o'shanda Flutter/RN mantiqiy — chunki u iOS+Android'da bir xil va
> murakkab native qismi yo'q. Ya'ni: **mentor = native, student = cross-platform** —
> to'g'ri taqsimot.

**Minimal talablar:** `minSdk 26` (Android 8.0), `targetSdk 35`, Compose BOM, Kotlin 2.x.
Planshet uchun `landscape` asosiy, lekin `sw600dp` layout bilan telefon ham ishlaydi.

---

## 4. Arxitektura

Backend'dagi Clean Architecture ruhini saqlaymiz:

```
app/
├── ui/                  Compose ekranlar + ViewModel (StateFlow)
│   ├── auth/  lessons/  room/  waiting/  settings/
├── domain/              use-case'lar (RoomController, WaitingRoomController…)
├── data/
│   ├── api/             Retrofit + Moshi — backend REST (docs/api-contract.md)
│   ├── ws/              OkHttp WebSocket — /api/v1/ws (auto-reconnect + backoff)
│   ├── livekit/         LiveKit Room wrapper, ScreenShareManager, DataChannel
│   └── store/           EncryptedSharedPreferences (token), DataStore (sozlama)
└── service/
    └── LessonForegroundService  (mediaProjection + microphone type)
```

**Qoidalar** (backend CLAUDE.md bilan bir xil ruh):
- Compose ekranlarida biznes logika yo'q — faqat state + event.
- Barcha tarmoq xatosi bitta `ApiError(code, message)` ga aylanadi (backend konverti
  `{code, message}` bilan bir xil).
- LiveKit `Room` obyekti **faqat** `LessonForegroundService`da yashaydi — Activity
  yo'q qilinsa ham (ekran o'chdi, boshqa ilovaga o'tdi) dars uzilmaydi. Bu eng muhim
  arxitektura qarori: ustoz ekranini ulashib **boshqa ilovani ochadi** (PDF, brauzer),
  ya'ni bizning Activity fon'ga tushadi.

---

## 5. Ekranlar

| # | Ekran | Mazmun |
|---|-------|--------|
| 1 | Login | email + parol; "Meni eslab qol"; server URL (debug build'da o'zgartiriladi) |
| 2 | Darslar | ro'yxat (jonli / rejalashtirilgan / tugagan), pull-to-refresh, qidiruv |
| 3 | Dars yaratish | sarlavha, vaqt, parol, kutish xonasi toggle, join-havolani ulashish |
| 4 | **Jonli xona** | asosiy ekran (quyida) |
| 5 | Kutish xonasi (bottom sheet) | so'rovlar ro'yxati, Qabul / Rad, "hammasini qabul qil" |
| 6 | Ishtirokchilar (bottom sheet) | mute / chiqarish / gapirishga ruxsat, qo'l ko'targanlar tepada |
| 7 | Chat (bottom sheet) | tarix + yangi xabar |
| 8 | Sozlamalar | profil, chiqish, ilova versiyasi, log yuborish |

### Jonli xona (4) — planshet landscape
```
┌──────────────────────────────────────────────────────┐
│ ● JONLI  Matematika 9-sinf     👥 24   ⏺ 12:03  [⋮]  │
├──────────────────────────────┬───────────────────────┤
│                              │  Kutish xonasi   (2)  │
│   Ekran ulashish preview     │  ─────────────────    │
│   yoki kamera                │  Ali Valiyev          │
│   (ulashilayotganda kichik   │   [Qabul] [Rad]       │
│    "SIZ EKRAN ULASHYAPSIZ"   │  ─────────────────    │
│    banner)                   │  Chat / Reaksiya      │
│                              │  ✋ 3 qo'l ko'tarilgan │
├──────────────────────────────┴───────────────────────┤
│  🎤   📹   [ 🖥 EKRAN ULASHISH ]   ⏺   💬   👥   ⏹  │
└──────────────────────────────────────────────────────┘
```
- Ekran ulashilayotganda **suzuvchi mini-panel** (`TYPE_APPLICATION_OVERLAY`) chiqadi:
  ⏹ to'xtatish, 🎤 mute, 👥 kutayotganlar soni. Ustoz boshqa ilovada bo'lganda ham
  darsni boshqara oladi. Bu — mahsulotni "haqiqiy" qiladigan detal.
- Notification'da ham xuddi shu tugmalar (overlay ruxsati berilmasa — fallback).

---

## 6. ⭐ Ekran ulashish — texnik tafsilot (yadro)

Bu ilovaning butun sababi, shuning uchun eng aniq yozilgan qism.

### 6.1 Manifest
```xml
<uses-permission android:name="android.permission.INTERNET"/>
<uses-permission android:name="android.permission.CAMERA"/>
<uses-permission android:name="android.permission.RECORD_AUDIO"/>
<uses-permission android:name="android.permission.FOREGROUND_SERVICE"/>
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_MEDIA_PROJECTION"/>
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_MICROPHONE"/>
<uses-permission android:name="android.permission.POST_NOTIFICATIONS"/>   <!-- API 33+ -->
<uses-permission android:name="android.permission.SYSTEM_ALERT_WINDOW"/>  <!-- suzuvchi panel -->

<service
    android:name=".service.LessonForegroundService"
    android:exported="false"
    android:foregroundServiceType="mediaProjection|microphone|camera" />
```

> LiveKit Android SDK o'zining `ScreenCaptureService`ini ham beradi. Biz **o'z
> servisimizni** ishlatamiz, chunki unga dars sessiyasi (Room, WS, timer) ham
> bog'lanadi. Implementatsiya paytida SDK'ning joriy hujjatidan aniq class nomi va
> tavsiya etilgan yondashuvni tekshiring — SDK 2.x da o'zgargan bo'lishi mumkin.

### 6.2 Oqim (Android 14/15 qoidalariga mos tartib — TARTIB MUHIM)
```
1. Ustoz [Ekran ulashish] bosadi
2. MediaProjectionManager.createScreenCaptureIntent() → tizim dialogi
   ("Darsly butun ekranni yozib olsinmi?")  ← foydalanuvchi tasdiqlaydi
3. onActivityResult(resultCode, data) olinadi
   ⚠️ Android 14+ : foreground service SHU QADAMDAN KEYIN startForeground qilinishi shart.
      Aks holda SecurityException / darhol o'ldiriladi.
4. startForegroundService(LessonForegroundService, intent-extras: resultCode+data)
   → servis darhol startForeground(id, notification, FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION)
5. room.localParticipant.setScreenShareEnabled(true, mediaProjectionPermissionResultData)
6. LiveKit SCREEN_SHARE trek chiqaradi → SFU → barcha web-o'quvchilar ko'radi
```

### 6.3 Kod eskizi
```kotlin
// ui/room/ScreenShareLauncher.kt
private val projectionLauncher = registerForActivityResult(
    ActivityResultContracts.StartActivityForResult()
) { result ->
    if (result.resultCode != Activity.RESULT_OK || result.data == null) {
        vm.onScreenShareCancelled()      // bekor qilish — XATO emas, jim o'tamiz
        return@registerForActivityResult
    }
    LessonForegroundService.startScreenShare(this, result.resultCode, result.data!!)
}

fun requestScreenShare() {
    val mpm = getSystemService(MediaProjectionManager::class.java)
    projectionLauncher.launch(mpm.createScreenCaptureIntent())
}
```
```kotlin
// service/LessonForegroundService.kt (muhim qismi)
override fun onStartCommand(intent: Intent, flags: Int, startId: Int): Int {
    startForeground(NOTIF_ID, buildNotification(), FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION)
    val data: Intent = intent.getParcelableExtra(EXTRA_PROJECTION_DATA)!!
    scope.launch {
        runCatching { room.localParticipant.setScreenShareEnabled(true, data) }
            .onFailure { reportScreenShareFailed(it) }   // Sentry + UI toast
    }
    return START_NOT_STICKY   // qayta tiklashda proyeksiya baribir yaroqsiz
}
```

### 6.4 Sifat sozlamalari (planshet + O'zbekiston interneti)
| Parametr | Qiymat | Sabab |
|---|---|---|
| Rezolyutsiya | 1280×720 gacha kichraytirish | 1600p tablet ekrani → bitrate portlaydi |
| FPS | 15 (slayd/PDF) · 24 (video ko'rsatilsa) | matn uchun 15 yetarli, trafik 40% kam |
| Bitrate | 800–1500 kbps adaptiv | LiveKit simulcast + `degradationPreference` |
| Content hint | `TEXT`/`DETAIL` | matn keskinligi harakat silliqligidan muhimroq |
| Simulcast | screen-share uchun **o'chirilgan** | matn past qatlamda o'qilmaydi; adaptiv bitrate afzal |

### 6.5 Chekka holatlar (albatta test qilinsin)
| Holat | Kutilgan xatti-harakat |
|---|---|
| Ustoz tizim dialogini bekor qildi | Toast yo'q, tugma o'chgan holatga qaytadi |
| Ustoz boshqa ilovaga o'tdi (PDF ochdi) | Ulashish davom etadi, suzuvchi panel ko'rinadi |
| Telefon qo'ng'irog'i keldi | Mikrofon avtomatik mute, ulashish davom etadi, qo'ng'iroq tugagach unmute so'raladi |
| Ekran o'chdi / qulflandi | Ulashish davom etadi (foreground service), audio ham |
| Wi-Fi → 4G ga o'tdi | LiveKit ICE restart; UI'da "Qayta ulanmoqda…" 3 soniyadan ortiq bo'lsa |
| Batareya optimizatsiyasi servisni o'ldirdi | Onboarding'da "cheklanmagan batareya" so'raladi; o'lsa — qayta ulanish taklifi |
| Android 15: proyeksiya har sessiyada qayta tasdiqlanadi | Bir dars = bir tasdiq; qayta ulanishda qayta so'raladi (UI shunga tayyor bo'lsin) |
| Ekranda maxfiy bildirishnoma chiqdi | Onboarding: "Bezovta qilmang" rejimini yoqish tavsiyasi |
| Kamera + ekran bir vaqtda | LiveKit ikkalasini ham publish qiladi; web'da screen asosiy, kamera kichik |

---

## 7. Backend o'zgarishlari (aniq, fayl bilan)

Yaxshi xabar: **oz.** Lekin ikkitasi **majburiy** — ularsiz ilova production'da ishlamaydi.

### 🔴 B1 — MAJBURIY: WebSocket `Origin` tekshiruvi native klientni bloklaydi
`backend/internal/infrastructure/websocket/websocket.go:27-34`:
```go
CheckOrigin: func(r *http.Request) bool {
    if allowed == "" { return !isProd }
    origin := r.Header.Get("Origin")
    return strings.EqualFold(origin, allowed)   // ← native klient Origin YUBORMAYDI → false
},
```
Native OkHttp WebSocket `Origin` header yubormaydi (u brauzer tushunchasi). Production'da
`FRONTEND_BASE_URL` o'rnatilgan → mobil ilova **har doim 403 oladi** → kutish xonasi
push'lari kelmaydi.

**Tuzatish (xavfsiz):**
```go
origin := r.Header.Get("Origin")
// Native mijozlar (Android/iOS) Origin yubormaydi. Brauzer esa WS uchun Origin'ni
// HAR DOIM yuboradi — ya'ni bo'sh Origin brauzerdan kelolmaydi, CSWSH xavfi yo'q.
// Autentifikatsiya baribir ?token= orqali bajariladi.
if origin == "" { return true }
return strings.EqualFold(origin, allowed)
```
Muqobil (qattiqroq): `X-Darsly-Client: android` headerini talab qilish + bo'sh Origin.

### 🔴 B2 — MAJBURIY: refresh-token rotatsiyasi mobil uzilishda hammani chiqarib yuboradi
`internal/pkg/token/jwt.go:146-152` — refresh **qayta ishlatilsa** → `RevokeAllUserSessions`.
Mobil tarmoqda klassik ssenariy: ilova refresh yuboradi → server rotatsiya qiladi →
**javob yo'lda yo'qoladi** → ilova o'sha eski refresh bilan qayta uradi → server buni
"token o'g'irlandi" deb hisoblaydi → **ustoz web'da ham, planshetda ham chiqib ketadi.
Dars o'rtasida.**

**Ikki tomonlama tuzatish:**
1. *Klientda (majburiy):* refresh **single-flight** — bir vaqtda faqat bitta so'rov
   (`Mutex`), qolgan barcha 401 lar shu natijani kutadi. Retry **yangi** token bilan.
2. *Backendda (tavsiya):* rotatsiyadan keyin eski `jti` ni darhol o'chirmasdan
   ~30 soniyalik "grace" oynasida yangi juftlikka qayta ishora qiluvchi kalit
   qoldirish — takroriy so'rov o'sha juftlikni qaytaradi, sessiya o'lmaydi.
   (`Rotate` ichida: `SETEX refresh:grace:<old_jti> 30 <new_refresh>`.)

### 🟡 B3 — Tavsiya: yozuvda ekran ulashish kichkina ko'rinadi
`internal/infrastructure/livekit/egress.go:26` — `Layout: "grid"`. Grid'da ekran ulashish
oddiy plitka bo'lib qoladi; darsni qayta ko'rgan o'quvchi slaydlardagi matnni o'qiy olmaydi.
→ `"speaker"` (yoki `"single-speaker"`) ga o'zgartirilsin — LiveKit ekran ulashishni
avtomatik asosiy oynaga oladi. Bir satrlik o'zgarish, katta effekt.

### 🟡 B4 — Tavsiya: ilova versiyasini majburiy yangilash
`GET /api/v1/app/version` → `{android_min: "1.0.0", android_latest: "1.2.0", apk_url: "..."}`
Side-load'da Play Store avtomatik yangilamaydi. Buzuq versiya tarqalib ketsa, uni
to'xtatishning yagona yo'li shu. **12 qadamli naqsh talab qilmaydi** — bitta handler yetarli.

### 🟢 B5 — Ixtiyoriy (v1.1): FCM push
Hozir eslatmalar faqat WS orqali (`notification` domeni + reminder worker) → ilova
yopiq bo'lsa ustoz darsni o'tkazib yuboradi. FCM qo'shish uchun to'liq 12-qadamli naqsh:
`device_tokens` domeni + migratsiya + `POST /devices` + reminder worker'da FCM yuborish.
~1 kun. v1 da o'rniga: Android `AlarmManager` bilan lokal eslatma (server o'zgarishsiz).

### ✅ O'zgarish TALAB QILMAYDIGANLAR (tekshirildi)
- **Ekran ulashish grant'i** — `livekit/token.go`: host uchun `CanPublish: true`,
  manba cheklovi yo'q → SCREEN_SHARE ruxsat etilgan. ✅
- **Web frontend** — `Stage.jsx` screen share'ni allaqachon aniqlaydi va asosiy
  oynaga chiqaradi. ✅ **Nol o'zgarish.**
- **CORS** — native HTTP klientga taalluqli emas. ✅
- **Ko'p qurilma sessiyasi** — har login o'z `sid`ini oladi; ustoz web va planshetda
  bir vaqtda kira oladi. ✅ (B2 dagi grace tuzatilsa.)
- **RBAC** — `mentor` roli o'zgarmaydi. ✅

---

## 8. Web bilan moslik protokoli (data-channel)

Mobil ilova web o'quvchilar bilan **aynan bir xil** JSON sxemasini yuborishi shart,
aks holda chat/reaksiya bir tomonlama bo'lib qoladi. Manba haqiqat:
`frontend/src/livekit/messaging.js` + `views/LiveRoom.jsx`.

```jsonc
// Chat        (LiveRoom.jsx:395)
{ "kind": "chat",     "id": "<uuid>", "name": "Ustoz", "body": "matn", "ts": 1737800000000 }
// Reaksiya    (LiveRoom.jsx:402)
{ "kind": "reaction", "emoji": "👍", "name": "Ali" }
// Qo'l ko'tarish (LiveRoom.jsx:411)
{ "kind": "hand",     "raised": true, "identity": "<lk-identity>", "name": "Ali" }
// So'rovnoma  (LiveRoom.jsx:440)
{ "kind": "poll",     "action": "open|close", "poll": { ... } }
// Doska       (LiveRoom.jsx:297+) — v1 da mobil FAQAT e'tiborsiz qoldiradi
{ "kind": "wb",       "act": "stroke|clear|on|off|view|snapshot|bg_*", ... }
```
Qo'shimcha: backend host-chat'ni `entity.ChatMessage` shaklida (`kind` **siz**,
`sender_name` + `body` bilan) broadcast qiladi — mobil dekoder buni ham
`kind:"chat"` ga normalizatsiya qilishi kerak (`messaging.js:24-36` bilan bir xil).

**Qabul mezoni:** planshetdan yozilgan chat web'da chiqadi; web'dagi ✋ va 👍 planshetda
chiqadi; `wb` xabarlari mobil ilovani yiqitmaydi (jim tashlanadi).

WS xabar turlari (`websocket/waitingroom.go`):
`waiting_room.request` (mentorga) · `waiting_room.admitted` · `waiting_room.rejected`.

---

## 9. Bosqichlar

Bir dasturchi uchun baho. Har bosqich oxirida **ishlaydigan APK** bo'ladi.

| # | Bosqich | Ish | Kun | Qabul mezoni |
|---|---------|-----|-----|--------------|
| **M0** | Backend to'siqlari | B1 (WS Origin), B2 (refresh grace), B3 (egress layout) | **0.5** | `curl` bilan Origin'siz WS ulanadi; parallel refresh sessiyani o'ldirmaydi |
| **M1** | Skelet + Auth | Compose loyiha, Retrofit, EncryptedSharedPrefs, login/logout, single-flight refresh, xato konverti | **2** | Login → token saqlanadi → ilova qayta ochilganda kirgan holatda |
| **M2** | Darslar | Ro'yxat, yaratish, join-havolani ulashish, pull-to-refresh, offline/xato holatlari | **2** | Planshetda yaratilgan dars web'da ko'rinadi |
| **M3** | ⭐ Xona + ekran ulashish | LessonForegroundService, LiveKit ulanish, kamera/mik, **MediaProjection**, suzuvchi panel | **4** | **Planshetdan ekran ulashiladi, 3 xil brauzerdagi o'quvchi ko'radi; ustoz PDF ochsa ham davom etadi** |
| **M4** | Kutish xonasi + ishtirokchilar | WS auto-reconnect, admit/reject, mute/remove/allow-speak | **2** | Web'dan guest so'rov yuboradi → planshetda 1 soniyada chiqadi → Qabul → guest kiradi |
| **M5** | Chat + reaksiya + yozib olish | data-channel ikki tomonlama, chat tarixi, recording start/stop | **2** | §8 dagi qabul mezoni bajariladi |
| **M6** | Sayqal + release | Onboarding (ruxsatlar, batareya), Sentry, landscape/tablet layout, imzolangan APK, versiya tekshiruvi | **2** | Haqiqiy planshetda 90 daqiqalik dars uzilishsiz |
| | **Jami** | | **~14.5 kun** | |

**Eng katta risk M3 da** — shuning uchun uni **birinchi kuni "spike" qilib** sinab ko'rish
tavsiya etiladi: bo'sh Activity + LiveKit SDK + MediaProjection → real qurilmadagi
test serverga ulanib, web'da ko'rinishini tasdiqlash. Yarim kunlik ish, butun rejaning
asosiy noaniqligini yopadi. **Bu tasdiqlanmaguncha qolgan ishni boshlamang.**

---

## 10. Test rejasi

### Qurilma matritsasi (minimal)
| Qurilma | Android | Nega |
|---|---|---|
| Samsung Galaxy Tab A9+ (yoki shunga o'xshash) | 14 | Asosiy maqsad qurilma |
| Xiaomi Redmi Pad / arzon planshet | 13 | MIUI agressiv batareya o'ldirishi — eng qattiq sinov |
| Istalgan telefon | 15 | Android 15 proyeksiya qoidalari |
| Eski planshet | 9–10 | `minSdk` chegarasi |

### Sinov ssenariylari
1. **To'liq dars (asosiy):** login → dars yaratish → jonli → ekran ulashish →
   3 ta web-o'quvchi (Chrome desktop, Chrome Android, Safari) → chat → qo'l ko'tarish →
   yozib olish → yakunlash → yozuvni yuklab olish. **Ekran ulashish yozuvda katta ko'rinsinmi — tekshirilsin (B3).**
2. **Yomon internet:** 4G → Wi-Fi almashish, 500 kbps chegarash, 5 soniya uzilish.
   Kutilgan: qayta ulanadi, ulashish tiklanadi, dars tugamaydi.
3. **Fon ssenariysi:** ekran ulashib PDF ochish, 10 daqiqa boshqa ilovada, ekranni qulflash.
4. **Batareya:** 90 daqiqa uzluksiz — quvvat sarfi va qizishni o'lchash.
5. **Ruxsat rad etish:** kamera/mikrofon/bildirishnoma/overlay — har birini rad etib,
   ilova yiqilmasligini tekshirish.
6. **Yuklama:** mavjud `backend/tests/load/` bilan 25+ ishtirokchi + planshetdan ekran ulashish.

### Avtomatlashtirilgan
- Unit: token store, refresh single-flight (parallel 401), data-channel encode/decode
  (web `messaging.js` bilan mos ekanini tasdiqlovchi **golden JSON** testlari).
- Instrumented: navigatsiya oqimi (Compose UI test).
- CI: `.github/workflows/` ga `android.yml` — `assembleDebug` + unit test har PR'da.
> Eslatma: backendda hozir 37 unit test bor; mobil ilova ham 0 dan boshlamasin.

---

## 11. Build va tarqatish

```bash
# Debug (test serverga qarab)
./gradlew assembleDebug   # → app-debug.apk

# Release (imzolangan)
./gradlew assembleRelease  # keystore: keystore.jks (git'ga TUSHMAYDI)
```

- **Keystore** `deploy/android/keystore.jks` — `.gitignore`da. Parol serverdagi sirlar
  bilan bir joyda. **Yo'qolsa yangilanish chiqara olmaysiz** — zaxira nusxasi shart.
- **Tarqatish (v1):** APK'ni serverga qo'yish →
  `https://app.194.163.139.242.sslip.io/download/darsly-mentor.apk` (Caddy'ga bitta
  `handle /download/*` bloki). Ustozga QR-kod beriladi.
- Ustoz "Noma'lum manbalardan o'rnatish" ruxsatini berishi kerak — onboarding'da
  skrinshotli qo'llanma.
- **Versiya:** `versionCode` monoton oshadi; ilova ochilganda B4 endpoint'ini tekshiradi,
  `android_min` dan past bo'lsa bloklovchi dialog.
- **Play Store (keyin):** internal testing track — avtomatik yangilanish uchun eng yaxshi.
  Talab: maxfiylik siyosati sahifasi + `MediaProjection`dan foydalanishni izohlash
  (Google buni tekshiradi — "video-dars uchun ekran ulashish" deb aniq yozilsin).

### Konfiguratsiya (build variants)
| | debug | release |
|---|---|---|
| API | `https://app.194.163.139.242.sslip.io` (o'zgartirsa bo'ladi) | production URL |
| LiveKit | `wss://livekit.194.163.139.242.sslip.io` | production wss |
| Log | to'liq | Sentry'ga |

---

## 12. Xavflar va yumshatish

| Xavf | Ehtimol | Ta'sir | Yumshatish |
|---|---|---|---|
| Android 14/15 foreground-service qoidalari ulashishni o'ldiradi | O'rta | **Yuqori** | M3 dan oldin spike; API 34 va 35 da real test |
| Xitoy ROM'lari (MIUI/EMUI) servisni o'ldiradi | Yuqori | O'rta | Onboarding'da "avtomatik ishga tushirish"ni yoqish; o'lganda tez qayta ulanish |
| B2 tuzatilmasa dars o'rtasida logout | O'rta | **Yuqori** | M0 da tuzatiladi + klientda single-flight |
| Planshet uzoq darsda qiziydi/tez o'chadi | O'rta | O'rta | 720p/15fps, quvvatga ulab ishlatish tavsiyasi |
| Ustoz maxfiy bildirishnomani efirga uzatadi | O'rta | O'rta | Ulashish boshlanishida "Bezovta qilmang"ni yoqish taklifi |
| APK'ni ishonchsiz manba deb blokirovka qilish | Past | Past | QR + qo'llanma; keyin Play Store |
| iOS/iPad talab qilinib qolishi | O'rta | Yuqori | v1 dan keyin alohida loyiha sifatida baholanadi (quyida) |

### iOS/iPad — keyingi bosqich uchun ogohlantirish
iPad'da ekran ulashish uchun **Broadcast Upload Extension** kerak: alohida process,
50 MB xotira chegarasi, App Group orqali ilova bilan aloqa, ilova ichidan
"Ekranni yozib olish" tugmasi emas — tizim Control Center'idan boshlanadi.
Baho: **+8–10 kun** va App Store ko'rib chiqishi majburiy (side-load yo'q).
Shuning uchun v1 Android'ga fokuslanadi — bu to'g'ri qaror.

---

## 13. Birinchi qadamlar (agar ma'qullansa)

1. **B1 tuzatilsin** (WS Origin) — 15 daqiqalik ish, ilovasiz ham foydali.
2. **M3 spike** — bo'sh Android loyihasi + LiveKit SDK + MediaProjection →
   test serveriga ulanish → web'da ekran ko'rinishini tasdiqlash. **Yarim kun.**
3. Spike ishlasa → M0…M6 bo'yicha yurish. Ishlamasa → Flutter/RN wrapper'ni sinash
   (reja tuzilishi bir xil qoladi, faqat §3 va §6 o'zgaradi).

---

**Xulosa:** yondashuv to'g'ri va texnik jihatdan sog'lom. Server arxitekturasi bunga
allaqachon tayyor — LiveKit uchun mobil ekran ulashish shunchaki yana bir publisher,
web o'quvchilar tomonida **nol o'zgarish**. Asosiy ish mobil ilovaning o'zida (~14 kun),
backendda esa faqat **ikkita majburiy tuzatish** (WS Origin va refresh grace) bor —
ikkalasi ham mobil ilovadan mustaqil ravishda ham foydali.
