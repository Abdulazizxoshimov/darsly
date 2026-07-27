# Darsly Mobile — Roadmap va Texnologiya Tavsiyasi

> **Holat: TASDIQLASH KUTILMOQDA.** Foydalanuvchi tasdiqlagach jamoa (Backend / Mobile /
> Frontend / QA agentlar) ishga tushadi.
> Sana: 2026-07-25 · Muallif: PM agent + Frontend inventarizatsiya + Backend audit (3 agent)
>
> Bu hujjat `docs/darsly-mobile-plan.md` ni **almashtiradi** (u tor qamrovli dastlabki eskiz edi).

---

## 0. Tasdiqlangan qarorlar (foydalanuvchidan)

| # | Savol | Javob | Ta'siri |
|---|-------|-------|---------|
| 1 | Kim foydalanadi? | **Faqat o'qituvchi (mentor)** | O'quvchilar web'da qoladi. Ilova qamrovi 2 barobar kichrayadi. |
| 2 | Platforma? | **Android birinchi, iOS keyin** | R1–R2 Android; iOS alohida reliz (R3). |
| 3 | Tarqatish? | **Avval APK (QR), keyin do'kon** | Majburiy yangilanish mexanizmi P0 bo'ladi. |
| 4 | Ekranda nima ko'rsatiladi? | **Zoom qila oladigan hamma narsa** | Ekran ulashish + **ekran audiosi** + video rejimi kerak. To'liq paritet maqsad. |
| 5 | Qurilmalar? | **Hamma Android telefon va planshetlar** | `minSdk 26` (Android 8+), telefon **va** planshet layout, keng test matritsasi. |

**Mahsulot qoidasi (o'zgarmas):** o'quvchi uchun web hech qachon yopilmaydi. Havolani
bosgan o'quvchi hech qanday "ilovani o'rnating" devoriga urilmasligi shart.

---

## 1. Texnologiya tavsiyasi: **Kotlin + Jetpack Compose (native Android)**

### Nega native, Flutter/React Native emas

| Mezon | Kotlin native | Flutter | React Native |
|---|---|---|---|
| Ekran ulashish ishonchliligi | ⭐ **Eng yuqori** — LiveKit Android SDK asl manba | Yaxshi (wrapper) | O'rtacha (wrapper + Expo config plugin) |
| **Ekran audiosi** (`SCREEN_SHARE_AUDIO`) | To'liq nazorat (AudioPlaybackCapture) | Wrapper qo'llashini kutish kerak | Ko'pincha yo'q |
| Android 14/15 foreground-service qoidalari | To'g'ridan-to'g'ri | Wrapper yangilanishini kutish | Wrapper yangilanishini kutish |
| Eski qurilmalar (Android 8–10) | Eng yaxshi nazorat | Yaxshi | Og'irroq (JS bundle + Hermes) |
| Batareya / 90 daqiqalik dars | Eng yaxshi | Yaxshi | O'rtacha |
| iOS uchun keyin qayta ishlatish | 0% | ~65% | ~65% |

**Qaror sababi:** siz "Zoom qila oladigan hamma narsa" dedingiz. Zoom'ning mobil ekran
ulashishi — ekran + **tizim audiosi** + past latensiya + fon rejimida barqarorlik.
Bularning har biri Android'ning eng chuqur API'lariga (`MediaProjection`,
`AudioPlaybackCapture`, `ForegroundService` tiplari) tegadi. Wrapper qatlami bu yerda
qo'shimcha risk, foyda esa faqat iOS uchun — lekin iOS'da ekran ulashish (ReplayKit
Broadcast Extension) **baribir noldan yoziladi**, ya'ni wrapper deyarli hech narsa tejamaydi.

**Texnik parametrlar:**
- Kotlin 2.x · Jetpack Compose (Material 3) · `minSdk 26` (Android 8.0) · `targetSdk 35`
- LiveKit: `io.livekit:livekit-android` 2.x
- Tarmoq: Retrofit + OkHttp + Moshi · WS: OkHttp WebSocket
- Saqlash: EncryptedSharedPreferences (token) + DataStore (sozlama) + Room (offline kesh)
- Doska: Compose Canvas (yoki qattiq FPS talab bo'lsa `SurfaceView`)
- PDF: Android `PdfRenderer` (tizimga kiritilgan, kutubxona shart emas)
- Crash/telemetriya: Sentry Android

### Ekran layout'lari (barcha qurilmalar talabiga ko'ra)
| Qurilma | Layout | Izoh |
|---|---|---|
| Planshet landscape (asosiy) | 2 ustunli: sahna + o'ng panel | Zoom Tablet kabi |
| Planshet portret | 1 ustun + pastki sheet | |
| Telefon landscape | To'liq ekran sahna + suzuvchi boshqaruv | Dars ko'rish uchun eng yaxshi |
| Telefon portret | Ixcham: sahna yuqorida, boshqaruv pastda | |

---

## 2. Zoom bilan funksional paritet

Sizning talabingiz — "Zoom qila oladigan hamma narsa". Quyida halol solishtirish:
nima **allaqachon** bizda bor, nima qo'shiladi, nima **rejadan tashqarida** (va nega).

| Zoom imkoniyati | Bizning web'da | Mobil rejada | Reliz |
|---|---|---|---|
| Video/audio konferensiya | ✅ | ✅ | R1 |
| **Ekran ulashish** | ❌ (mobil'da imkonsiz) | ✅ **asosiy maqsad** | R1 |
| **Ekran audiosi** (video ko'rsatish) | ❌ | ✅ | R1 |
| Old/orqa kamera almashtirish | ❌ | ✅ | R1 |
| Fon rejimida davom etish | ❌ | ✅ | R1 |
| Kutish xonasi (waiting room) | ✅ | ✅ | R1 |
| Ishtirokchilar boshqaruvi (mute/remove) | ✅ | ✅ | R1 |
| Qo'l ko'tarish | ✅ | ✅ | R1 |
| Chat | ✅ | ✅ | R2 |
| Reaksiyalar (emoji) | ✅ | ✅ | R2 |
| Doska (whiteboard) | ✅ (kuchli: PDF fon, bosim) | ✅ + native qalam | R2 |
| So'rovnoma (poll) | ✅ | ✅ | R2 |
| Bulutga yozib olish | ✅ | ✅ | R2 |
| Push bildirishnoma | ❌ | ✅ | R2 |
| Suzuvchi mini-panel / PiP | ❌ | ✅ | R2 |
| Virtual fon / fon blur | ❌ | ⚠️ R4 | R4 |
| Breakout rooms | ❌ | ⚠️ R4 | R4 |
| Jonli subtitr / transkript | ❌ | ❌ rejada yo'q | — |
| Telefon orqali qo'ng'iroq (dial-in) | ❌ | ❌ rejada yo'q | — |

**Halol baho:** R1+R2 tugagach mobil ilova **o'qituvchi uchun Zoom bilan amalda teng**
bo'ladi. Virtual fon va breakout rooms — Zoom'da bor, bizda web'da ham yo'q; ular R4'ga
qo'yilgan, chunki ular mavjud bo'lmagan funksiyani noldan qurishni talab qiladi
(ML segmentatsiya modeli / ko'p-xonali arxitektura), bu esa asosiy maqsadni kechiktiradi.
Subtitr va dial-in rejadan tashqarida — ular alohida mahsulot yo'nalishi.

---

## 3. Backend to'siqlari (audit natijasi)

Backend arxitekturasi mustahkam va web uchun to'liq ishlaydi. Lekin **native klient uchun
9 ta bloker** topildi. Quyidagilar mobil ilovadan **mustaqil ravishda ham foydali** —
ya'ni bu ish behuda ketmaydi.

### 🔴 R0 — ilova ishlashi uchun MAJBURIY

| ID | Muammo | Fayl | Tuzatilmasa nima bo'ladi |
|---|--------|------|---------------------------|
| **BE-1** | WS `Origin` tekshiruvi native klientni bloklaydi | `websocket/websocket.go:27-34` | Ilova WS'ga **umuman ulana olmaydi** (403) → kutish xonasi va bildirishnoma real-time'i o'lik |
| **BE-2** | Refresh reuse-detection tarmoq uzilishida hamma sessiyani o'ldiradi | `pkg/token/jwt.go:142-154` | Mobil uzilishda **dars o'rtasida** planshet + web birdan chiqib ketadi |
| **BE-3** | Deploy'da TURN o'chirilgan | `deploy/server/livekit.yaml.example` (turn bloki yo'q) | ⚠️ **2026-07-25: 🔴→🟡 tushirildi.** Bu baho noto'g'ri edi. Haqiqiy qurilmada 4G (mobil internet, CGNAT) sinovi **muvaffaqiyatli** o'tdi: 720×1280, 9.6 MB media, ICE `srflx↔host` UDP, relay ishlatilmadi. Sabab: LiveKit — peer-to-peer emas, **ochiq IP'li SFU**; telefon chiquvchi UDP yuboradi, CGNAT mapping yaratadi, server javobni shu mappingга qaytaradi — CGNAT'ning katta qismida ishlaydi. TURN faqat **UDP butunlay to'silgan** tarmoqlarda kerak (maktab/ofis firewall'i), va u yerda ham avval `7881/tcp` ICE-TCP zaxirasi ishlaydi. TURN — oxirgi zaxira, bloker emas |
| **BE-4** | Xato javob shakli ikki xil: `{code,message}` vs `{error,code}` | `middleware/auth.go:26-29`, `rbac.go:27`, `rate_limit.go:85-88` | Klient bitta xato modeliga parse qila olmaydi; 401/403/429 da xabar `null` |
| **BE-5** | Ilova versiyasi / majburiy yangilash endpointi yo'q | — | Side-load'da buzuq versiyani **to'xtatishning yagona yo'li** yo'q |

### 🟡 R1–R2 — sifat va barqarorlik

| ID | Muammo | Fayl | Ta'siri |
|---|--------|------|---------|
| BE-6 | Presigned URL ichki `minio:9000` ga imzolanadi | `remote-setup.sh:49`, Caddyfile'da `files.*` yo'q | Yozuvni **yuklab bo'lmaydi** (web'da ham) |
| BE-7 | Deploy stack'da **egress konteyneri yo'q**, `livekit.yaml`da `redis:` yo'q | `deploy/server/docker-compose.yml` | Test serverida **yozib olish umuman ishlamaydi** |
| BE-8 | Egress layout `grid` | `livekit/egress.go:26` | ⚠️ **Bu da'vo qismab noto'g'ri edi** (2026-07-25 da tuzatildi): egress shabloni ekran ulashish publish qilinganda grid'ni **o'zi** speaker'ga almashtiradi — `livekit/egress:latest` obrazidagi kompilyatsiya qilingan shablon JS'i bilan tasdiqlangan. Baribir `speaker` ga o'tildi: ekran ulashishsiz ham ustoz asosiy oynada bo'lishi kerak (grid'da 25 talaba orasida teng plitka) va hujjatlashtirilmagan shablon xulqiga bog'liq qolmaslik uchun. Endi `LIVEKIT_EGRESS_LAYOUT` bilan sozlanadi |
| BE-9 | Push (FCM) infratuzilmasi yo'q | `notification.go:41-42` | Ilova yopiq bo'lsa kutish xonasi so'rovi va dars eslatmasi kelmaydi |
| BE-10 | Sessiya kaliti `Rotate` da uzaytirilmaydi | `pkg/token/jwt.go:127-212` | 30 kundan keyin **cheksiz refresh sikli** — hech qachon toza logout bo'lmaydi |
| BE-11 | CGNAT: auth 60/min/IP, WS 10rps/IP | `router.go:111,138-139` | Mobil operator NAT'i ortida ommaviy 429 |
| BE-12 | Kutish xonasi tokeni 15 daqiqada yo'qoladi, tiklash yo'q | `waitingroom.go:24,101,141-144` | Guest fon rejimidan qaytsa boshi berk ko'chada (web guest'ga ham tegadi) |
| BE-13 | `allow-speak` studentga **ekran ulashishni ham** beradi | `livekit/participant.go:24-35` | Kutilmagan: `CanPublishSources: [CAMERA, MICROPHONE]` bilan cheklash kerak |
| BE-14 | `docs/api-contract.md` eskirgan — 7 ta nomuvofiqlik | — | Mobil jamoa uchun yagona haqiqat manbai bo'lishi kerak |

### 🟢 Keyinroq (mentor-only qamrovda kerak emas)
Tizimga kirgan o'quvchi tushunchasi yo'q (`joinlink.go:102` — hamma `guest_<uuid>`),
student uchun dars endpointlari yo'q, guest chat/poll tarixi yo'q, fayl yuklash
endpointi yo'q, offline sync formati yo'q. Bular **o'quvchi ilovasi** qilinsa kerak
bo'ladi — hozirgi qamrovda emas.

---

## 4. Feature backlog (mentor · Android)

`P0` = bo'lmasa ilova ma'nosiz · `P1` = birinchi relizda kerak · `P2` = keyin
`✅` = web'da bor (portlanadi) · `🆕` = mobil uchun yangi · `⚠️` = bor, lekin mobilda sifat sakrashi

### Kirish va hisob
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M1 | Login + sessiyani saqlash | P0 | ✅ | |
| M2 | Jim token yangilash (**single-flight**) | P0 | ⚠️ | BE-2 bilan juft — bo'lmasa dars uziladi |
| M3 | Logout | P1 | ✅ | |
| M4 | Profil, parol o'zgartirish | P2 | ✅ | |
| M5 | Biometrik/PIN tez kirish | P2 | 🆕 | Dars oldidan 3 soniyada kirish |

### Darslar
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M6 | Darslar ro'yxati + pull-to-refresh + offline kesh | P0 | ✅ | |
| M7 | Dars yaratish | P1 | ✅ | |
| M8 | Join havolasini **native ulashish** (Telegram) | P1 | 🆕 | Real oqim Telegram orqali |
| M9 | Dars tahrirlash/o'chirish, jadval | P2 | ✅ | |
| M10 | Telefon kalendariga qo'shish | P2 | 🆕 | |

### ⭐ Jonli xona — media (yadro)
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M11 | Xonaga host bo'lib ulanish (LiveKit) | P0 | ✅ | |
| M12 | Kamera (old/orqa almashtirish) + mikrofon | P0 | ⚠️ | Orqa kamera bilan daftar ko'rsatish — yangi imkoniyat |
| M13 | **Ekran ulashish** (MediaProjection) | P0 | 🆕 | **Loyihaning sababi** |
| M14 | **Ekran audiosi** (AudioPlaybackCapture) | P0 | 🆕 | "Zoom kabi" talabidan — video ko'rsatish uchun |
| M15 | Fon rejimida davom etish (ForegroundService) | P0 | 🆕 | Ustoz PDF/GeoGebra ochsa dars uzilmasin |
| M16 | Ulanish sifati + avto qayta ulanish | P0 | ✅ | |
| M17 | Suzuvchi mini-panel (overlay) | P1 | 🆕 | Boshqa ilovada ham darsni boshqarish |
| M18 | Adaptiv sifat rejimi (matn/video/kam trafik) | P1 | 🆕 | 720p·15fps (matn) ↔ 24fps (video) |
| M19 | Ovoz yo'nalishi (dinamik/naushnik/BT) + qo'ng'iroqda avto-mute | P1 | 🆕 | |
| M20 | Gallery/Speaker ko'rinish | P1 | ✅ | |
| M21 | Picture-in-Picture | P2 | 🆕 | |

### Xona boshqaruvi
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M22 | Ishtirokchilar ro'yxati | P0 | ✅ | |
| M23 | So'zlashga ruxsat / bekor qilish | P0 | ✅ | Webinar modelida o'quvchi javob berishining yagona yo'li |
| M24 | Mute / chiqarish / hammani mute | P1 | ✅ | |
| M25 | Qo'l ko'targanlar paneli | P1 | ✅ | |
| M26 | Darsni yakunlash | P0 | ✅ | |

### Kutish xonasi
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M27 | Real-time so'rovlar + Qabul/Rad | P0 | ✅ | BE-1 siz ishlamaydi |
| M28 | "Hammasini qabul qil" | P1 | ⚠️ | 20 o'quvchi = 20 bosish muammosi |

### Muloqot
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M29 | **Web bilan protokol mosligi** (data-channel JSON) | P0 | ✅ | Bo'lmasa mobil ustoz va web o'quvchi bir-birini "eshitmaydi" |
| M30 | Chat (ikki tomonlama + tarix) | P1 | ✅ | |
| M31 | Qo'l ko'tarish / reaksiya | P1 | ✅ | |
| M32 | So'rovnoma yaratish/yopish/natija | P2 | ✅ | |

### Doska
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M33 | Doska: native qalam (bosim, palm-rejection, past latensiya) | P1 | ⚠️ | Protokol **allaqachon mobil-do'st** loyihalangan |
| M34 | PDF fon + sahifa navigatsiyasi | P1 | ✅ | `PdfRenderer` (tizim API'si) |
| M35 | Qurilmadan/Drive'dan fayl ochish | P1 | 🆕 | |
| M36 | Doska pro-asboblari (undo/redo, shakllar) | P2 | 🆕 | |

### Yozib olish va bildirishnoma
| ID | Feature | Prio | Web'da | Izoh |
|---|---------|------|--------|------|
| M37 | Yozib olish start/stop + REC indikatori | P1 | ✅ | Holat serverdan tiklanadi (web'dagi kamchilik tuzatiladi) |
| M38 | Yozuvlar ro'yxati + ulashish | P2 | ✅ | BE-6 siz ishlamaydi |
| M39 | Lokal eslatma (server o'zgarishsiz) | P1 | 🆕 | 80% qiymat, 0 backend ish |
| M40 | Push (FCM) | P1 | 🆕 | BE-9 talab qiladi |

### Ilova platformasi (ko'rinmas, lekin P0)
| ID | Feature | Prio | Izoh |
|---|---------|------|------|
| M41 | Onboarding: ruxsatlar + batareya optimizatsiyasini o'chirish | P0 | Bularsiz ekran ulashish/fon rejimi **jim o'ladi** |
| M42 | Majburiy yangilanish tekshiruvi | P0 | BE-5 talab qiladi |
| M43 | Telefon + planshet, portret + landscape layout | P0 | "Hamma qurilma" talabidan |
| M44 | Crash hisoboti (Sentry) + "Log yuborish" | P1 | Qo'llab-quvvatlash uchun |
| M45 | Offline / tarmoq yo'q holati | P1 | |
| M46 | Ekran ulashish telemetriyasi (boshlandi/xato/tugadi) | P1 | Muvaffaqiyat darajasini o'lchash |

---

## 5. Relizlar

### R0 — "Xavfni yopish" · ~1 kun
**Kiradi:** BE-1, BE-2, BE-3, BE-4, BE-5 + **spike**: bo'sh Android ilova + LiveKit SDK +
MediaProjection → test serveriga ulanib, web brauzerdagi o'quvchi ekranni ko'radi.

**Tayyor mezoni:**
1. Real qurilmadan ekran ulashildi, 3 xil brauzerda ko'rindi ✅
2. `Origin`siz WS ulanadi ✅
3. Parallel refresh sessiyani o'ldirmaydi ✅
4. 4G'dan (TURN orqali) media ulanadi ✅

> **Qat'iy qoida: R0 yashil bo'lmaguncha R1 boshlanmaydi.** Bu butun rejaning yagona
> katta noaniqligini eng arzon narxda yopadi.

#### R0 HOLATI (2026-07-25) — backend qismi ✅ TUGADI, QA TASDIQLADI

| Ish | Holat | Dalil |
|-----|-------|-------|
| BE-1 WS Origin | ✅ | Jonli 8 ssenariy: Origin'siz+token → 101, Origin'siz+tokensiz → 401, evil.com → 403 |
| BE-2 refresh grace | ✅ | Ayni juftlik idempotent qaytadi; 60s tashqarisida revoke saqlangan |
| BE-10 sessiya TTL | ✅ | Sessiya yo'q → `Rotate` xato (cheksiz sikl yopildi) |
| BE-4 xato konverti | ✅ | Web regressiyasi yo'q (2 agent mustaqil tasdiqladi) |
| BE-5 `/app-config` | ✅ | Ochiq, rate-limit, sir sizmaydi |
| BE-3/6/7 TURN+MinIO+egress | ✅ kod | **Deploy qilinmagan** — foydalanuvchi qarori |
| Android spike | ✅ | API 34 emulyator + haqiqiy server: `TrackPublished: SCREEN_SHARE` |

**QA topgan va tuzatilgan kritik xato — "sessiya forking":** `Rotate` da `Get`↔`TxPipelined`
orasi check-then-act edi (`MULTI/EXEC` `WATCH`siz = CAS emas). Jonli o'lchov: bitta refresh
token bilan 8 parallel so'rov → **8 ta mustaqil 30-kunlik refresh token**, ya'ni reuse-detektor
butunlay chetlab o'tilardi. Atomik Lua CAS bilan tuzatildi → 8 va 20 parallel so'rovda
**tirik JTI = 1**, hamma bir xil juftlik oladi. `-race -count=10` toza.

**Keyingi partiyada yakunlangan (QA ✅, 5 tsikl):**

| Ish | Natija |
|-----|--------|
| UUID validatsiyasi (`shared.ValidateID`) | 14 yo'lda 404, `22P02` = 0; **403 begona dars saqlangan** |
| BE-13 `allow-speak` manba cheklovi | Student ekran ulasha olmaydi; **host tokeni tegilmagan** |
| BE-8 egress layout | `speaker`, `LIVEKIT_EGRESS_LAYOUT` bilan sozlanadi |
| BE-12 kutish xonasi tokeni | Cache tugagach qayta chiqariladi (faqat `live` darsda) |
| BE-11 CGNAT rate-limit | Nishonli 60→10/min; taqsimlangan hujum: cheklanmagan→30/min |
| Validatsiya xatolari | Go struct/tag nomlari oshkor qilinmaydi |
| Host'ni nishonga olish guard'i | Ustoz o'z ekran ulashishini o'chira olmaydi |

**QA topgan 3 ta xato (agentlar o'zlari topa olmagan):**
1. **Sessiya forking** — `Rotate` da CAS yo'q edi → 8 parallel so'rov 8 ta mustaqil
   30-kunlik refresh token berardi, reuse-detektor chetlab o'tilardi. Lua CAS bilan yopildi.
2. **Soxta `sid` DoS** — BE-11 kiritgan: hujumchi imzosiz token bilan **qurbonning**
   rate-limit bucket'ini to'ldirib, uni tizimdan chiqarib yuborardi. Limiterda
   `sid` ni kalit qilishdan oldin imzo tekshiruvi bilan yopildi (muddati tugagan
   token ham rad etiladi — jonli tasdiqlangan).
3. **Host o'z ekran ulashishini o'chirishi** — BE-13 kiritgan yon ta'sir.

QA o'z taklifining (`sid`+IP kompozit kalit) agent yechimidan **kuchsizroq** ekanini tan oldi:
CGNAT ortida hujumchi va qurbon **ayni IP'da** bo'ladi (Ucell/Beeline — aynan maqsadli
auditoriya), ya'ni `sid`+IP hujumni to'smasdi. Imzo tekshiruvi ildizni yopadi.

**Ochiq (foydalanuvchiga bog'liq):**
- 4G/TURN orqali media — deploy kutmoqda
- Real qurilmada ekran brauzerda ko'rinishi — qurilma kutmoqda

**R1 ga qoldirilgan (🟢, bloklovchi emas):** mavjud bo'lmagan LiveKit ishtirokchisini
moderatsiya qilish 500 beradi (twirp `not_found` → `apperr.NotFound` qilinsin, pre-existing);
`rotateScript` Redis Cluster'da CROSSSLOT (bitta instansda muammo yo'q).

---

### R1 — "Ustoz cho'ntakda: jonli dars" · ~15 ish kuni
**Kiradi:** M1, M2, M3, M6, M7, M8, M11–M16, M18, **M20**, M22, M23, M26, M27, M29, M41, M42, M43, M46
(+M17 iloji bo'lsa)

> **QAROR (2026-07-26, foydalanuvchi tasdiqlagan):** M20 (video sahna) R2 dan **R1 ga
> ko'chirildi** va 5-blokka **xona redizayni** qo'shildi. Sabab: qurilma sinovida ko'rindiki,
> hozirgi xona ekrani — diagnostika paneli (matn qatorlari + jurnal), **video umuman
> ko'rsatilmaydi**. Zoom'da ustoz xonaga kirganda birinchi navbatda **odamlarni** ko'radi.
> 3 ta pilot ustozga "dasturchi ilovasi" berib bo'lmaydi. Batafsil: 5-blok tavsifi.

**Foydalanuvchi va'dasi:** *Ustoz planshet yoki telefondan to'liq dars o'tkazadi va
ekranini (audio bilan) ulashadi. Kompyuter kerak emas.*

**Tayyor mezoni (mahsulot darajasida):**
1. 3 ta real ustoz **90 daqiqalik haqiqiy darsni faqat mobil qurilmadan** uzilishsiz o'tkazdi
2. Ustoz dars o'rtasida boshqa ilovaga o'tdi — dars va ekran ulashish davom etdi
3. Waiting-room'dagi o'quvchi 3 soniyada qabul qilindi
4. Web'dagi o'quvchi ulashilgan ekranni **matn o'qiladigan sifatda** ko'rdi
5. Video ko'rsatilganda o'quvchi **ovozni ham** eshitdi
6. Wi-Fi ↔ 4G almashtirilganda dars tugamadi
7. 0 ta kritik crash; ekran ulashish muvaffaqiyati ≥ 95%

**Tarqatish:** imzolangan APK + QR-kod + "noma'lum manba" qo'llanmasi.

### R2 — "To'liq paritet + native ustunlik" · ~12 ish kuni
**Kiradi:** M4, M9, M17, M19, M24, M25, M28, M30, M31, M32, M33, M34, M35, M37, M38,
M39, M40, M44, M45 (+BE-6, BE-7, BE-8, BE-9, BE-10, BE-11, BE-13)

**Va'da:** *Ustoz web'ga umuman qaytmaydi — doska qalam bilan yaxshiroq, push eslatmalar
keladi, chat va yozuvlar joyida.*

**Tayyor mezoni:** ustozlarning ≥70% i darslarini ilovadan o'tkazadi; native doskaga
qalam bahosi web'nikidan yuqori (5 ustozdan so'rov); "kechikkan dars" holati 0 ga tushdi.
**Tarqatish:** Play Store internal testing (avtomatik yangilanish) + APK zaxira.

### R3 — "iOS/iPadOS" · ~12 ish kuni + App Store review
ReplayKit Broadcast Extension bilan ekran ulashish (alohida process, 50 MB xotira
chegarasi, Control Center'dan boshlanadi — **UX Android'dan farq qiladi**), maxfiylik
siyosati, akkaunt o'chirish, shikoyat/bloklash (Apple 1.2 talabi).

### R4 — "Zoom+ imkoniyatlari"
Virtual fon / blur, breakout rooms, PiP kengaytmasi, doska pro-asboblari, davomat hisoboti.

---

## 6. Jamoa va ish jarayoni

| Agent | Mas'uliyat | Huquq |
|---|---|---|
| **PM** | Backlog, ustuvorlik, koordinatsiya, hisobot, bloklanishni hal qilish | Reja + hisobot |
| **Backend** | BE-1…BE-14, web regressiyasiz | Backend kodi |
| **Mobile** | Android ilova (to'liq huquq: fayl, terminal, SDK, konfiguratsiya) | Cheklanmagan |
| **Frontend** | Web'ni "oltin standart" sifatida saqlash, regressiyani ushlash | Frontend kodi |
| **QA** | Har feature'ni funksional + regressiya + tarmoq/media testi | Faqat o'qish + test |

**Tsikl (o'zgarmas):**
```
PM vazifa beradi → Backend (kerak bo'lsa) → Mobile/Frontend kod yozadi
   → QA tekshiradi → ❌ bo'lsa qaytadi / ✅ bo'lsa PM hisobot beradi → keyingi feature
```
**QA "✅ tasdiqlandi" bermaguncha keyingi feature boshlanmaydi.**

**Qat'iy cheklov (barcha agentlar):** `gitlab.com` bilan bog'liq papka/repo/konfiguratsiyaga
**murojaat qilinmaydi**.

**PM o'zboshimchalik bilan qaror qilmaydi:** arxitektura o'zgarishi, qamrov kengayishi
yoki noaniq talab bo'lsa — foydalanuvchidan so'raydi.

---

## 7. Test matritsasi ("hamma qurilmalar" talabidan)

| Qurilma sinfi | Android | Nega majburiy |
|---|---|---|
| Samsung planshet (Tab A/S) | 13–14 | Asosiy maqsad qurilma, S Pen |
| **Xiaomi / Redmi** | 13–14 | MIUI fon servislarini agressiv o'ldiradi — **eng qattiq sinov** |
| Istalgan telefon | 15 | Android 15 proyeksiya qoidalari qattiqlashgan |
| Arzon/eski qurilma | 8–10 | `minSdk 26` chegarasi, kam RAM |
| Huawei (GMS'siz) | — | FCM ishlamaydi → push fallback tekshiruvi |

**Majburiy ssenariylar:** to'liq 90 daqiqalik dars · fon rejimi (PDF/GeoGebra ochish,
ekran qulflash) · Wi-Fi↔4G almashish · 500 kbps chegarash · telefon qo'ng'irog'i ·
har bir ruxsatni rad etish · batareya/qizish o'lchovi.

**Avtomatlashtirilgan:** unit (token single-flight, data-channel golden JSON — web
`messaging.js` bilan mosligi), Compose UI test, CI'da `assembleDebug` + testlar.

---

## 8. Risklar

| # | Risk | Ehtimol | Ta'sir | Yumshatish |
|---|------|---------|--------|------------|
| R-1 | Android 14/15 foreground-service qoidalari ulashishni o'ldiradi | O'rta | **Yuqori** | R0 spike; API 34 va 35 da real test |
| R-2 | MIUI/EMUI servisni o'ldiradi | Yuqori | O'rta | Onboarding'da avto-ishga tushirish; tez qayta ulanish |
| R-3 | Ustoz APK o'rnatmaydi ("noma'lum manba" qo'rquvi) | Yuqori | **Yuqori** | QR + video-qo'llanma; birinchi darsni birga o'rnatish; R2'da Play Store |
| R-4 | Trafik: 90 daq ekran ulashish ≈ 0.6–1 GB | O'rta | O'rta | M18 kam-trafik rejimi; Wi-Fi tavsiyasi; trafik hisoblagichi |
| R-5 | Batareya/qizish | O'rta | O'rta | 720p/15fps default; quvvatga ulash tavsiyasi; issiqlikda avto-pasaytirish |
| R-6 | **Maxfiylik hodisasi** — ekranda shaxsiy bildirishnoma efirga chiqadi | O'rta | **Yuqori** (obro') | Ulashish boshida "Bezovta qilmang" taklifi + 3 soniyalik ogohlantirish |
| R-7 | Buzuq versiya tarqalib ketishi | O'rta | Yuqori | M42 majburiy yangilanish — **R1'dan kesilmasin** |
| R-8 | "Zoom bilan bir xil" kutilmasi cheksiz kengayishi | Yuqori | Yuqori | §2 paritet jadvali kelishuv hujjati; yangi talab = yangi reliz, R1 kechiktirilmaydi |
| R-9 | Web'dagi ochiq nuqsonlar mobilga meros bo'ladi (BE-6, BE-7) | Yuqori | O'rta | R2'da yopiladi; aks holda ilovada "o'lik tugma" |
| R-10 | Fokus tarqalishi (Android + iOS + o'quvchi ilovasi) | Yuqori | Yuqori | Qat'iy ketma-ketlik R1→R2→R3; R4 faqat o'lchangan talab bilan |

---

## 9. Ochiq savollar (ishni bloklamaydi, lekin javob berilsa yaxshi)

1. **Pilot guruh:** nechta ustoz bilan sinaladi va "tayyor" ni kim tasdiqlaydi?
   (Bo'lmasa R1 mezoni sub'ektiv bo'lib qoladi.)
2. **Dars uzilishi hozir qanchalik tez-tez?** — barqarorlik ustuvorligini aniqlaydi.
3. **Yozib olish qanchalik ishlatiladi?** — agar har dars yozilsa, BE-6/BE-7/BE-8 R1'ga ko'chadi.
4. **Web ilova ustozlar uchun qo'llab-quvvatlanadimi?** — "ha" bo'lsa har funksiya ikki joyda.
5. **Ilova nomi va ikonkasi** — "Darsly Mentor" yoki bitta "Darsly"?

---

## 10. Keyingi qadam

Ushbu roadmap **tasdiqlangandan so'ng**:
1. PM `docs/mobile-acceptance-criteria.md` yaratadi (R1 mezonlari asosida)
2. Backend agent R0 blokerlarini (BE-1…BE-5) tuzatadi
3. Mobile agent R0 spike'ni bajaradi
4. QA R0 ni tasdiqlaydi → R1 boshlanadi

**Umumiy baho:** R0 ~1 kun · R1 ~15 kun · R2 ~12 kun · R3 (iOS) ~12 kun + review
