# Zoom darajasidagi video — to'liq ish ro'yxati

> **Maqsad:** Yozuvni Zoom kabi qilish. Bu ro'yxat `lesson.mp4` (bizniki) va Zoom-sifatли
> reference (matematika darsi) solishtiruvига asoslangan — ikkalasi ham
> `~/Downloads/jonly-vs-reference/` da. Qaror sabablari: [`RECORDING-DECISIONS.md`](RECORDING-DECISIONS.md).

**Sana:** 2026-08-16 · **Codec qarori:** H.264 CRF ~26 (universal moslik — arxiv brauzerда
ochiladi; HEVC Firefox/eski qurilmada buzilardi, shuning uchun H.264).

---

## 🧪 So'nggi test natijasi (1.7.2, 2026-08-16 — gorizontal, YouTube bilan)

`lesson.mp4` 240 MB / 14 daq / 1280×720. O'lchandi:
- ✅ **Audio trek VALID** (AAC 48kHz) → **Telegram'да ochiladi** (audio-bug fixi ishladi).
- ✅ **Video GORIZONTAL, to'liq kadr** — letterbox deyarli yo'q, sifat yaxshi (Q1-B tasdiqlandi).
  Kadr: `~/Downloads/jonly-vs-reference/5-yangi-test-gorizontal.png`.
- ❌ **Yozuvда OVOZ YO'Q** — `mean_volume: -91 dB` = to'liq jimlik. Na ovoz, na YouTube.
  Jimlik-fix valid trek berди (Telegram uchun), lekin **haqiqiy dars ovozи yozilmaган**.
- ⚠️ **Video ~6 fps** — o'ynayotган video uchun sakrab ko'rinishi mumkin (publish 15fps, recorderда tushган).
- ✅ **Hajm** — 240MB/14min, VBR ishlayapti.

## Zoomni "yaxshi" qiladigan 6 o'lchov — biz qayerdamiz (1.7.2'дан keyin)

| # | O'lchov | Zoom / reference | Bizniki (1.7.2) | Holat |
|---|---|---|---|---|
| 1 | Kadrni to'ldirish (16:9, qorasiz) | to'liq ekran | gorizontalда to'liq kadr | ✅ **Yechildi** (Q1-B) |
| 2 | Aniqlik (o'qishlilik) | 720–1080p | 720p, tiniq | ✅ yetarli (1080p opsion) |
| 3 | Samarali hajm | ixcham | ~17 MB/min VBR | 🟡 o'lchash → post-siqish kerakмi |
| 4 | Audio — trek valid | AAC | ✅ valid (Telegram ochadi) | ✅ **Tuzatildi** |
| 4b | **Audio — yozuvда OVOZ bor** | ovozли | ❌ **jimlik (-91dB)** | 🔴 **YANGI #1** |
| 5 | Audio — media (YouTube) ovozi | ichida | ❌ yozuvда yo'q + jonliда past | 🔴 |
| 6 | Silliq harakat | 25–30 fps | **~6 fps** 🔴 | 🔴 **YANGI** (kadr tushган) |

➡️ Video KO'RINISHI (aspect, sifat) endi **yaxshi**. Qolgan tanqidiy bo'shliqlar:
**yozuvда ovoz yo'q (4b/5)** va **past fps (6)**.

---

## Qilinadigan ishlar (ketma-ket)

### ✅ Allaqachon bor (kod yozilgan, kommit qilinmagan)
- **Letterbox tuval** — `LetterboxFit.kt` + `LocalRecorder.kt` (gorizontal 1280×720, test 7/7).
  → 16:9 mexanizmi tayyor; gorizontalда qora chegara qo'shmaydi (o'lchov #1 mexanizmi).
- **O'quvchi kamera opt-in** — ovoz erkin, kamera ustoz ruxsati bilan (video yuki nazoratда).

### 🔨 1. Audio bug — bo'sh/buzuq trek (o'lchov #4)
Ovoz manbai bo'lmaganда recorder `[0][0][0][0]` buzuq audio trek yozadi.
**Ish:** ovoz kelmasa — jim (silent) valid AAC trek yozish yoki audio treksiz yakunlash;
muxer boshlanishi audioga bog'lanib qolmasin.
**Natija:** har yozuv doim ochiladigan, valid audioli bo'ladi.

### 🔨 2. Kino/media ovozini yozish (o'lchov #5)
Hozir faqat ishtirokchilar mikrofoni yoziladi; device playback (kino/video) yozilmaydi.
**Ish:** `AudioPlaybackCapture` (MediaProjection bilan) qo'shib, mikser'ga ikkinchi manba
sifatida ulash. Ruxsat: MediaProjection allaqachon ekran ulashishда olingan.
**Natija:** mentor darsда kino qo'ysa — ovozи ham yozuvга tushadi.

### 🔨 3. Dars tugagач telefonда siqish (o'lchov #3 — asosiy yutuq)
Xom 992 MB → H.264 CRF 26 → **~100 MB (10× kichik), sifat saqlangan** (o'lchangan: 9%).
**Ish:** "To'xtatish"дан keyin fonда transcode (Android `MediaCodec` yoki tekshirilgan
kutubxona) → keyin **kichik fayl** serverга yuklanadi. App o'chsa — retry.
**Natija:** server transcode qilmaydi (cheksiz kengayadi), yuklash/saqlash 10× yengil.

### 🔨 4. Jonli bitrate CBR→VBR (o'lchov #3 — qo'shimcha)
Hozir qat'iy `bitrate=2_000_000` CBR — statik kadrда ham bitни isrof qiladi.
**Ish:** `KEY_BITRATE_MODE=VBR`, ~1.2 Mbps. Post-siqishni inkor qilmaydi — xom fayl
ham kichrayadi (batareya/joy tejaladi).
**Natija:** yozuv paytидаyoq fayl kichikroq.

### 🟡 5. (Opsion, testга bog'liq) 720p → 1080p
Faqat ovозли + gorizontal real testда zich matn (formulalar) o'qishга qiyin chiqsa.
Ko'r-ko'rona emas — o'lchovга asoslanган qaror.

### 📱 6. Foydalanish — gorizontal dars (o'lchov #1)
Ilova mentorни gorizontalга yo'naltiradi (ipucha/eslatma) — letterbox ishga tushmasin.
Kod emas, UX nudge.

---

## Test (kod tugagач)
1. **Ovозли gorizontal dars** — mikrofon yoniq + kino ovozли → audio + korimlilik.
2. **Vertikal + gorizontal** orientatsiya — 73 daq test faqat portret bo'ldi (adb uzilgan).
3. **Natija:** Arxivда hajm (~100 MB?), sifat (ko'z/SSIM), audio bor-yo'qligi, brauzerда ochilishi.

## Chiqarish
- APK build + o'rnatish. Kerak bo'lsa git commit.

---

## AUDIO — media (YouTube/kino) ovozi (2026-08-16 tadqiqoti)

**Topilma:** ekran/qurilma ovozi RAQAMLI ushlanadi (`ScreenAudioCapturer`/AudioPlaybackCapture) —
karnaydан emas, professional. Lekin ikki muammo:
- **Tartib bug'i (TUZATILDI v1.7.2):** ekran-ovoz faqat ulashish boshida + mikrofon yoniq bo'lsa
  ishga tushardi. Endi mik yoqilганда qayta chaqiriladi (`LessonSession.setMicrophoneEnabled`).
- **Media ovozi PAST (faqat YouTube'да, ovoz normal):** ikki sabab — (1) ekran gain `0.6`,
  (2) WebRTC noise-suppression/AGC standart YONIQ → media "shovqin" deб bostiriladi.

**Zoom yechimи (tadqiqot):** raqamli capture (biz ham ✓) + ovoz ALOHIDA oqim (bizда telefon bitta
ADM buferi → imkonsiz, mikrofonга mikslanadi) + faol-so'zlovchi DINAMIK ducking + "Original Sound"
(AGC/NS o'chirish) + stereo.

**Qilinadigan (media ovozini to'g'rilash):**
- 🔨 **A1. "Original Sound" ekvivalenti** — media ulashilганda noise-suppression + AGC o'chirish
  (media bostirilmasin). *Asosiy tuzatish.*
- 🔨 **A2. Ekran gain `0.6 → 0.9`** — tez, ovozга tegmaydi (alohida gain).
- 🔨 **A3. Dinamik ducking** — media 100%да, faqat ustoz gapirганда pasayadi (Zoom kabi).
- 🟡 **A4. (opsion) Stereo** — hozir mono (`CHANNELS=1`); musiqa/kino uchun.
- 🧪 **A5. Tekshirish:** yozuvга (LocalRecorder) ham YouTube ovozi tushadimi.

## YAKUNIY USTUVORLIK (1.7.2 test'дан keyin to'g'rilangan)

**✅ Tugadi va TASDIQLANDI (test bilan):**
- Audio-bug (trek valid) → **Telegram ochadi** ✅
- Video gorizontal, to'liq kadr, sifat yaxshi (Q1-B) ✅
- Ekran-ovoz tartib fixi (v1.7.2) · jonli VBR · Q4 o'quvchi-kamera (deploy) · APK 1.7.2 + mentor2 ✅

**🔴 1-GALДА — YOZUVДА OVOZ (eng muhim, yozuv jimlik chiqdi):**
- **REC-1.** Recorder audio manbaini ta'minlash: (a) mikrofon yozuv o'rtасiда yoqilса
  recorderга ulash (hozir faqat boshда ulanadi; boshда o'chiq bo'lsa — hech qachon), (b) ekran
  (YouTube) ovozини recorder mikser'iga berish (hozir u faqat jonli mik-trek ADM buferiga ketadi).
  → Natija: yozuvда ovozingiz ham, YouTube ham eshitiladi.

**🔴 2-GALДА — VIDEO FPS:**
- **VID-1.** ~6 fps sababини topib tuzatish (recorderда kadr tushishi / encoder bardoshi) → silliqroq.

**🟠 3-GALДА — JONLI ovoz sifati (viewer eshitadigan):**
- **A1.** ✅ (kod, v1.7.4) "Original Sound" — noise-suppression/AGC/highpass/typing o'chirildi
  (echo-cancel qoldi). `MediaTuning.audioCapture()` + RoomOptions. Media "shovqin" deб bostirilmaydi.
- **A2.** ✅ (kod, v1.7.4) Ekran gain `0.6 → 0.85`.
- **A3.** ⏸️ **Test'dан keyin** — dinamik ducking ko'r-ko'rona XAVFLI: telefonда media karnaydan
  chiqsa mikrofon uni akustik eshitadi → VAD "gapiryapti" deб yolg'on aniqlaydi → media doim
  ducking (teskari natija). Real ovoz + naushnik bilan tuning kerak. A1+A2 yetsa — kerak bo'lmasligi mumkin.

**🟡 4-GALДА — video hajm/aniqlik (o'lchovга bog'liq):**
- Real fayl hajmini kuzatish → post-siqish (item 3) kerakмi. Kerak bo'lsa 720p→1080p. Gorizontal UX eslatma.

**🧪 Test:** REC-1'дан keyin — mikrofon YONIQ + YouTube bilan yozib, ovoz + fps + Telegram tekshirish.

> **Eslatma:** avval "audio #1 = YouTube past" deб rejalashtirgandik. 1.7.2 test ko'rsatдики,
> haqiqiy #1 — yozuvда **umuman ovoz yo'q** (jimlik). "YouTube past" jonli oqim masalasi bo'lib,
> 3-galга tushди. Video ko'rinishi (aspect/sifat) esa kutilgandan yaxshi chiqди — yechilди.

## Zoomда bor, lekin biz ATAYLAB QILMAYMIZ (kelajakда qaytmaslik uchun)
- **Serverда gallery-kompozit / crop-pad** — bizda yozuv lokal, server 4-yadro.
  Serverда transcode = kengaymaydi. Biz telefonда 16:9 letterbox qilamiz (arzon).
  Batafsil: `RECORDING-DECISIONS.md` Q3 va Q5.
- **Ko'p-kamerali gallery yozuvi** — biz 1→N vebinar; yozuv mentor ekranини oladi,
  o'quvchi kamerasi kamdan-kam (opt-in). Gallery-normalize kerak emas.
