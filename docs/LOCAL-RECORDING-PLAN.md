# Lokal (client-side) yozib olish — ish-reja

> Maqsad: **Zoom «local recording» kabi** — telefon dars vaqtida ekran+ovozni
> **to'liq sifatda** o'z xotirasiga yozadi, dars tugagach serverga yuklaydi.
> Undan keyingi zanjir (MinIO → transkod → Telegram → Arxiv → retention)
> **o'zgarmaydi**. Server egress (Chrome-composite, CPU'ga yetmaydi, 4 fps)
> o'rniga keladi.

Manba: 2 kod-xaritalash agenti + Android feasibility tadqiqoti (2026-08).

---

## 0. Nega bu — to'g'ri yechim

- **Jonli** oqim SFU orqali to'g'ridan (server kodlamaydi) → sifat yaxshi.
- **Yozuv** hozir server egress'да Chrome bilan kompozitsiya + kodlash → 4 yadroli VPS bo'g'iladi → **6449 kadr tashlandi → 4 fps / 6 kbps**.
- Telefon ekranни **allaqachon to'liq sifatда** ushlab turibdi (MediaProjection). O'shani lokal yozsak — sifat = jonli sifat, **server CPU yengil**, **tekin**.

---

## 1. Texnik asos (tadqiqotда tasdiqlangan)

### ✅ Video — ishonchli, imkoni bor
- MediaProjection'ни LiveKit SDK yaratadi, LEKIN **OCHIQ getterlar bor**:
  - `LocalVideoTrack.getCapturer()` (public) → `livekit.org.webrtc.ScreenCapturerAndroid`
  - `ScreenCapturerAndroid.getMediaProjection()` (public) → **xuddi shu** `MediaProjection`
- Bir MediaProjection'дан **ikkinchi `VirtualDisplay`** yasash Android'da ruxsat etilgan → uni o'z `MediaCodec` (H264) input `Surface`ига → `MediaMuxer` bilan mp4.
- SDK **audioда aynan shu reuse'ни qilyapti** (`ScreenAudioCapturer(MediaProjection,...)`) — tirik dalil.
- Ruxsatlar: `FOREGROUND_SERVICE_MEDIA_PROJECTION`, `RECORD_AUDIO`, FGS tiplari — **hammasi bor** (`AndroidManifest.xml:13-15,67-71`). Yangi ruxsat kerak emas.
- media3 kerak emas — platform `MediaCodec+MediaMuxer` yetarli.

### ⚠️ Audio — eng nozik qism (asosiy xavf)
- **Android call-ovozини (VOICE_COMMUNICATION) ekran-yozib olish orqали ushlashга RUXSAT BERMAYDI** → AudioPlaybackCapture bilan **o'quvchilар ovozини olib bo'lmaydi**.
- Telefonда mikrofon **bitta** va uни WebRTC ADM (LiveKit) egallaydi → parallel `AudioRecord`/`MediaRecorder(MIC)` **konflikt berishi mumkin**.
- Shuning uchun audio uchun uch yo'l:
  - **(a) Faqat ustoz ovozi** — eng oddiy; lekin ADM mic'ни egallagани uchun konflikt sinovда tekshirилиши SHART.
  - **(b) To'liq (ustoz + o'quvchilar)** — LiveKit'ning o'z audio quvurига (dekodlangan PCM) ulanish; SDK darajасидаgi ish, murakkab.
  - **(c) Video ovozsiz + ovoz alohida** — kam ma'noли.
- **QAROR + SPIKE kerak** (pastда 5-bosqich).

### 🟡 Dual-encoder xavfi
- Telefon bir vaqtда 2 marta kodlaydi: oqим (LiveKit) + fayl (MediaCodec). Kuchли telefon ko'taradi; zaif telefonда strain → **fallback (server egress)** kerak.

---

## 2. MOBIL o'zgarishlar (`mobile/`, fayl:qator)

### 2.1 Yangi: `data/livekit/LocalRecorder.kt`
- Kirish: `MediaProjection` (SDK track'idан), o'lcham (`ScreenCaptureSize.forDisplay` qayta ishlatiladi), bitreyt (config), audio manba.
- Ish: 2-`VirtualDisplay` → `MediaCodec` (video/avc, `LOCAL_REC_BITRATE`≈0.8–2 Mbps, KEYFRAME interval) → input `Surface`; audio `MediaCodec`(aac) yoki `MediaRecorder`; `MediaMuxer` bilan mp4 (`+faststart` ekvivalenti — moov oldinга).
- Crash-xavfsizlik: davriy `muxer` segment/flush yoki `MediaRecorder` (o'zi finalize qiladi). Ilova yiqilса — oxirgi segment saqlanadi.
- Fayl: app-specific dir (`context.filesDir/recordings/<lessonId>.mp4`).

### 2.2 Ulanish nuqtalari
- **Start:** `ScreenShareController.start()` `onSuccess` (`ScreenShareController.kt:75-84`) — `startScreenAudio()` yonida `localRecorder.start(projection, ...)`.
  - Projection'ни olish: `session` → `room.localParticipant.getTrackPublication(Track.Source.SCREEN_SHARE)?.track as LocalScreencastVideoTrack` (naqsh `LessonSession.kt:296-297` da bor) → `.getCapturer() as ScreenCapturerAndroid` → `.getMediaProjection()`.
- **Stop/cleanup:** `LessonSession.stopScreenShare()` (`:225-229`), `onStop` lambda (`:215-219`), `RoomViewModel` teardown (`:566`) — recorder'ни SDK track'idан OLDIN to'xtatiб muxer'ni finalize qiling.
- **Reconnect:** `reconcileScreenShare()` (`:246`) / `restoreScreenShare()` (`RoomViewModel.kt:1174`) — yangi projection'да recorder'ни qaytadan bog'lash (cache qilmang — projection obyekti yangilanadi).
- **Foreground service:** `LessonService` mediaProjection tipi allaqachon faol — qo'shimcha kerak emas.

### 2.3 Fallback (zaif telefon)
- `MediaCodec` init/format sozlaмаsi muvaffaqiyatsiz bo'lsa yoki dual-encode qo'llab-quvvatlanmasa → `localRecorder` ishga tushmaydi va **dars `is_recording_enabled` egress yo'lига qaytadi** (server yozadi). Ya'ni lokal — «afzal», egress — «zaxira».
- Buni backend biladi: telefon "lokal yozuv boshlandi" deб backend'ga xabar bermaса, backend egress'ни boshlaydi (hozirgidek).

### 2.4 Yuklash (dars tugagach)
- Yangi `data/repo/RecordingUploadRepository.kt` + `DarslyApi` metodlari:
  - `POST lessons/{id}/recording/upload-url` → presigned PUT URL.
  - Telefon mp4'ни **to'g'ridan MinIO'ga** PUT (katta fayl — WorkManager bilan fonда, uzilса qayta).
  - `POST lessons/{id}/recording/complete` (object_key, duration, size, started_at, ended_at).
- Yuklangач lokal fayl o'chiriladi. WorkManager bilan ishonchli (ilova yopilса ham davom etadi).

### 2.5 UI
- «REC» indikatori lokal rejimда ham (`recordingEnabled`, `RoomViewModel.kt:144`).
- Dars tugagач: «Yozuv yuklanmoqda…» holati (kabinet/arxivда). Yuklanмаса — qayta urinish tugmasi.

---

## 3. BACKEND o'zgarishlar (`backend/`, fayl:qator)

**Muhim: undan keyingi zanjir manba-neytral** — `recordings` jadvали `status='ready'` + `object_key` bilan to'ldirilса, Telegram/retention/archive **o'zgаришsiz** ishlaydi.

### 3.1 MinIO — presigned PUT
- `internal/infrastructure/minio/minio.go:16-24` interfeysига `PresignedPutObject(ctx, key, ttl)` qo'shish (minio-go'да bor; `c.presign` public endpoint `minio.go:63-72`).

### 3.2 `egress_id` to'sig'ini yechish
- Hozir: `egress_id VARCHAR(64) NOT NULL` + UNIQUE (`migrations/000005_recordings.up.sql:7,18`).
- Yangi migratsiya: `egress_id` NULLABLE + UNIQUE'ни **partial** (`WHERE egress_id IS NOT NULL`).
- Yoki (kam ish): upload uchun sintetik `egress_id = "upload:"+recID`.
- `GetByEgressID/MarkReady/EnqueueTranscode/EnqueueTelegram/MarkFailed` — hozir `egress_id` WHERE (`postgres/recording.go:69,248,268,449,533`). **ID-asosli variantlar** qo'shish (yoki sintetik egress_id bilan qoldirib, hech narsa o'zgartirmaslik — eng kam ish).

### 3.3 Usecase + handler + router
- `usecase/recording/recording.go`: yangi `RequestUpload(lessonID, mentorID)` — `object_key=recordings/<lesson>/<recID>.mp4` (`recording.go:266` formatи), `repo.Create` `status='uploading'` (yoki `recording`), presigned PUT qaytaradi.
- `CompleteUpload(recID, meta)` — MinIO `Head`(size) tekshiriб → `MarkReady`(ID bo'yicha) + `started_at/ended_at/duration/size` → **`EnqueueTranscode` + `EnqueueTelegram`** (mavjud `recording.go:581,598` mantig'i).
- Handler: `api/handlers/v1/recording.go` (namuna: `StartRecording`).
- Router: `api/router.go:314-316` yaqinида `POST /lessons/:id/recording/upload-url`, `POST /lessons/:id/recording/complete` (protected — auth+rbac avtomatik).

### 3.4 Lokal rejim (egress'ni o'chirish)
- Telefon lokal yozadigan bo'lsa egress boshlanmasin. Eng toza: `EnsureRecording` (`recording.go:114-141`) boshida "bu dars lokal rejimда → egress yo'q" tekshiruvi.
- Belgилаш: (a) mavjud `is_recording_enabled` ustiga qurish (lokal = klient upload qiladi, egress esa faqat fallbackда), yoki (b) `lessons.recording_mode ENUM('egress','local','auto')`. **`auto`** tavsiya: telefon ko'tarса lokal, aks holda egress.

### 3.5 Transkod (ixtiyoriy)
- Telefon mp4 allaqachon h264 — transkod SHART EMAS. Lekin crop/hajm uchun ishlatса bo'ladi. MVPда **o'tkazиб yuborish** mumkin (`content_offset_sec=0`).

---

## 4. WEB (keyingi bosqich)
- Brauzer `MediaRecorder` API bilan web-mentor ham lokal yozishi mumkin. Mobil MVP'дан keyin.

---

## 5. BOSQICHLAR (tavsiya)

### Bosqich 0 — SPIKE (avval! 1–2 kun) ⭐
Real telefonда faqat riskни sinash (to'liq feature emas):
1. Ekran ulashib turганда 2-`VirtualDisplay` + `MediaCodec` bilan lokal mp4 yozib ko'rish → **jonli oqim sifati pasayadimi?** (dual-encode).
2. Audio: mic'ни yozишга urиниб → **WebRTC ADM bilan konflikt bormi?** Ovoz chiqadimi?
3. O'lchoв: CPU/issiqlik/batareya; yozuv sifati (ffprobe).
**Natija:** dual-encode + audio ishlаsа → MVP'ga o'tamiz. Ishlamаsa → yo'lni qayta ko'ramiz (masalan kuchли server).

### Bosqich 1 — MVP
Video (to'liq sifat) + **faqat ustoz ovozi** + upload + backend upload yo'li + egress fallback. Ekran-ulashишли darslar. Bu — 90% foyda.

### Bosqich 2 — to'liq ovoz
Ustoz + o'quvчilar ovozини LiveKit audio quvуридан mikslash.

### Bosqich 3 — kamera-only darslar + web lokal yozuv.

---

## 6. QARORLAR (sizдан)
1. **SPIKE avval** — roziligmisiz? (2 kun, riskни tasdiqlaydi, keyin ishonch bilan quramiz.)
2. **Audio v1 = faqat ustoz ovozi** yetarlimi (Bosqich 1), keyin to'liq ovoz (Bosqich 2)? Yoki to'liq ovoz avvaldan shartmi?
3. **Rejim belgisi:** `auto` (telefon ko'tarса lokal, aks holда egress) — mosми?
