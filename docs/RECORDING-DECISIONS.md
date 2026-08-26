# Yozuv (recording) bo'yicha qarorlar jurnali

> **Maqsad:** Video/yozuv sifati va korimlilik bo'yicha qabul qilingan qarorlar,
> **bekor qilingan variantlar** va ularning **sabablari**. Bu jurnal — bekor
> qilingan planga keyinchalik "aslida u optimal edi" deb qaytib qolmaslik uchun.
> Agar bir qarorga qayta qaytmoqchi bo'lsangiz — avval shu yerdagi "Nega rad etildi"
> ni o'qing; sabab hali kuchda bo'lsa, qaytmang.

**Sana:** 2026-08-16 · **Kontekst:** 73 daqiqalik test yozuvi (`lesson.mp4`, 992 MB)
boshqa platforma dars yozuvi (reference, matematika darsi) bilan solishtirildi.

---

## Test natijalari (o'lchangan faktlar)

| | Bizniki (Jonly) `lesson.mp4` | Reference (matematika darsi) |
|---|---|---|
| O'lcham | 1280×720 (16:9) | 1280×800 (16:10) |
| FPS | ~29.5 | 25 |
| Video bitrate | 1.89 Mbps (xom) | 0.11 Mbps (Telegram siqqan) |
| Audio | ❌ yo'q / buzuq (`[0][0][0][0]`) | ✅ AAC stereo 48 kHz |
| Davomiylik | 73 daq | 57 daq |
| Hajm | 992 MB (xom) | 78 MB (Telegram) |
| Kontent joylashuvi | portret → markazда tor, ~65% qora | to'liq ekran |

Fayllar: `~/Downloads/jonly-vs-reference/` (1-yonma-yon, 4-sifat-xom-vs-crf26).

---

## Q1 — Korimlilik (letterbox / aspect)

**Muammo:** Test yozuvида kontent markazда tor chiziqда, ikki yon katta qora
(letterbox). Sabab: recorder chiqishни **doim gorizontal 1280×720** qiladi, planshet
esa **portret** edi → tik ekran keng kadrга sig'ib qoladi.

### ❌ Rad etilgan variant 1: "O'lchamни Zoomга (1280×800) tenglashtirish"
**Nega rad etildi:** Reference to'liq ekran ko'ringani 800 vs 720 tufayli EMAS — u
**gorizontal, to'liq ekran** yozilgan. Faqat `OUT_H`ни 800 qilish portret
letterboxни **umuman hal qilmaydi** — yon qoralar baribir qoladi. Bu kosmetik
o'zgarish, asl sababга (portret → gorizontal kadr) tegmaydi.
**Qaytmaslik sharti:** doim kuchда — o'lcham raqami muammoning sababi emas.

### ⏸️ Kechiktirilgan variant 2: "Dinamik tuval (letterboxни butunlay olib tashlash)"
Chiqish manba shakliga moslashadi (portret → tik video, gorizontal → keng).
**Nega hozir tanlanmadi (rad emas, kechiktirildi):**
- O'lcham har darsда o'zgaruvchan bo'ladi; portret video gorizontal pleyerда
  baribir yon qora oladi (qorani fayldan pleyerга ko'chiramiz, xolos).
- ⚠️ **Dars o'rtасiда burilish:** manba shakli o'zgarsa, video fayl o'lchamини
  o'rtaда o'zgartira olmaydi → aspectни qulflashга majbur bo'lamiz (= yana letterbox).
- Ko'proq kod, ko'proq chekka holat (toq piksel, encoder chegaralari).
**Qaytish sharti:** faqat mentorlar **ko'pincha portretда** (telefon selfi-kamera,
yuz ko'rsatib) o'tsa — o'sha holда bu optimal bo'ladi.

### ✅ Tanlangan: Variant B — dars GORIZONTALда o'tiladi
Aksar dars kontenti (doska, slayd, ekran ulashish) tabiiy gorizontal — reference
kabi. Gorizontalда bizni 1280×720 **allaqachon to'liq to'ladi, qorasiz, kod
o'zgarishisiz**. Ilova mentorни gorizontalга yo'naltiradi/tavsiya qiladi.
**Sabab:** eng arzon, eng mustahkam, bashoratli; letterbox umuman ishга tushmaydi.

### 🔧 Kod holati (2026-08-16 — shu session'da AMALGA OSHIRILDI)
Gorizontal 1280×720 tuval + letterbox recorder'da **allaqachon qurilgan**:
`mobile/.../data/livekit/LocalRecorder.kt` (encoder DOIM `OUT_W=1280×OUT_H=720`) +
`mobile/.../data/livekit/LetterboxFit.kt` (sof joylash hisobi — `cropAndScale` +
qora tuval ustiga blit; JVM test 7/7). Ya'ni letterbox — **xavfsizlik to'ri**:
portret yoki dars o'rtasida burilgan manba ham gorizontal faylga tushadi (pleyerda
to'liq ekran) va burilishda encoder sinmaydi. Q1-B (gorizontal dars) — shu tuval
ustidagi **foydalanish tavsiyasi**: kontent gorizontal bo'lsa letterbox qora chegara
qo'shmaydi. Ikkalasi zid emas — biri mexanizm, biri tavsiya.

### ❌ Rad etilgan tashxis: "FPS juda past (~2.5), video sirg'anadi/tirishadi"
Eng birinchi test (79s, portret, ilovaning **'Ekraningiz ulashilmoqda' placeholder
ekrani** — mentor haqiqiy doska/materialni ochmagan) ffprobe'da ~2.5 fps ko'rsatdi.
Bu **o'lchov artefakti**, throttle EMAS: kadrlararo interval tahlili — 162/193 interval
<100ms (hatto ~50 fps portlash), 12 tasi >1.5s (biri 21s). Statik ekran o'zgarmasa
yangi kadr kelmaydi (MediaProjection tabiati) → o'rtacha pastga tushadi; harakatli
kontentda fps to'liq (keyingi 73 min real test 29.5 fps berdi).
**Qaytmaslik sharti:** doim kuchda — "past fps"ni bug deb quvmang; statik/placeholder
kontent testi vakil emas. Sinovni real dars materiali (harakatli, gorizontal) bilan oling.

---

## Q2 — Video/kino ovozini yozish

**Muammo:** Recorder faqat ishtirokchilar mikrofonini yozadi, **device playback
(kino/video ovozi)ни ATAYLAB yozmaydi** (kod izohi: "AudioPlaybackCapture
call-ovozini ushlolmaydi"). "Darsда kino qo'yish" ssenariysiда video ovozsiz qoladi.

### ✅ Tanlangan: Device playback audio'ни miksga qo'shish (HA)
**Sabab:** mentor darsда video/kino qo'ysa, uning ovozи ham yozuvга tushishi kerak.

**Alohida bug (Tanlangan — tuzatiladi):** ovoz manbai bo'lmaganда recorder buzuq
`[0][0][0][0]` audio trek yozib qo'yadi (ba'zi pleyerlar buziladi). Yechim: jim
(silent) valid trek yozish yoki umuman audio trek yozmaslik.

### 🔧 Aniqlik (2026-08-16) — JONLI ekran-ovoz ALLAQACHON ushlanadi (tartib bug'i)
Item 2ни qayta ko'rдik: app'да **ekran/qurilma ovozini ushlash BOR** (`ScreenAudioCapturer`,
AudioPlaybackCapture, HONOR ABR-LX1'да o'lchangan). Dizayn (C-6): ovoz **mikrofon trekiga**
mikslanadi (telefonда bitta ADM buferi — mustaqil trek imkonsiz). **Ammo bug:** `startScreenAudio()`
(a) mikrofon TREKI bo'lса ishlaydi, (b) faqat ulashish boshida bir marta urinadi. Mik standart
**o'chiq** (kirishда) bo'lgani uchun ekranни ulashganда ekran-ovoz ishga tushmasdi — ustoz ovozi
o'tardi-yu, YouTube ovozi o'quvchiga bormasdi (bir trekда: ovoz o't­sa trek oqyapti, video ovozi
qo'shilmagan → capture boshlanmagan). **Tuzatildi (v1.7.2):** mik yoqilganда ekran faol bo'lsa
`startScreenAudio()` qayta chaqiriladi (`LessonSession.setMicrophoneEnabled`, repeat×6). Endi
tartib ahamiyatsiz. **Cheklov:** bu telefonда media-ovoz uchun mikrofon **yoniq** bo'lishi shart
(ADM bitta bufer). **Tekshiriladi:** yozuvга (LocalRecorder) ham YouTube ovozi tushadimi.

---

## Q3 — Siqish (compression): qayerда va qachon

**O'lchangan fakt (120s namuna, forest = og'ir kontent):**

| Variant | Hajm | vs xom | Sifat (ko'z / SSIM) |
|---|---|---|---|
| Xom (qurilma encoder, 2 Mbps CBR) | 28.9 MB | 100% | — |
| **H.264 CRF 26** | 2.7 MB | **9%** | farqsiz ✅ |
| H.265 CRF 28 | 2.2 MB | 7% | farqsiz ✅ |
| H.264 800k | 8.6 MB | 29% | farqsiz |

Butun dars: **992 MB → ~100 MB**, sifat saqlangan (11× kichik).
*(Eslatma: yutuqning bir qismi qora chegaralar toza siqilgani hisobiga; Q1-B bilan
kontent to'lgach ham real dars kontenti — slayd/doska — 5–10× kichrayadi.)*

### ❌ Rad etilgan variant 1: "Serverда siqish"
**Nega rad etildi:** server har dars uchun transcode qiladi → **CPU yuki yuqori**,
va avval xom **992 MB serverга yuklanadi** (mobil internet ko'tarmaydi). 100 mentor
bir vaqtда tugatsa server tiqiladi — kengaymaydi.
**Qaytmaslik sharti:** doim kuchда, agar server sig'imi muhim bo'lsa.

### ❌ Rad etilgan variant 2: "Faqat jonli (live) siqish"
Jonli MediaCodec CBR bitни isrof qiladi (statik kadrда ham 2 Mbps), CRF kabi
moslasha olmaydi. Post-optimizatsiya imkoni yo'q.
**Qaytmaslik sharti:** doim kuchда post-siqishга nisbatan.
*(Lekin jonli CBR→VBR ~1.2 Mbps — qo'shimcha arzon yutuq sifatida QOLADI, post-siqishни
inkor qilmaydi.)*

### ✅ Tanlangan: Dars tugagач, TELEFONДА siqish → keyin kichik faylни yuklash
**Sabab:**
- Server **hech qachon transcode qilmaydi** → CPU yuki **nol** → cheksiz kengayadi.
- Yuklash 10× yengil, saqlash 10× arzon.
- Yozuv baribir lokal (telefonда) — siqish uchun tabiiy joy o'sha.
**Yagona narx:** siqishни mentor telefoni bajaradi (~bir necha daqiqa, apparatли
HEVC encoder, fonда). App o'chib qolsa — siqish/yuklashни qayta tiklash (retry).

### ✅ HAQIQAT (2026-08-26 — test bilan aniqlandi): SERVER ALLAQACHON TRANSCODE QILADI
Avvalги taxminimiz — "server transcode qilmaydi, telefonда qilamiz" — **NOTO'G'RI edi.**
Backendда to'liq transcode pipeline BOR: `internal/worker/transcode.go` (`TranscodeWorker`),
`recordings.transcode_status` navbati (`ClaimTranscode` — atomik, BITTALAB).
- **x264 CRF 26, veryfast preset** — aynan biz tavsiya qilган usul. Sifat qat'iy saqlanadi,
  hajm mazmunga qarab kichrayadi. **Test (1.7.4): 61MB → 4MB (15×)**, stereo, faststart.
- `analyze` bosqichi qora chegaralarni (`cropdetect`) kesadi, jim/qora boshni trimlaydi, fps
  normallaydi.
- ⚠️ **Telefonда post-siqish (item 3) KERAK EMAS** — serverда x264 CRF bor (HW encoderдан yaxshiroq).
  Demodagi "server yuki" e'tirozi endi boshqacha hал qilinadi (pastга qarang).

### 🔧 Jonli sig'imni himoya (2026-08-26 — foydalanuvchi talabi): transcode CPU cheklovi
Muammо: transcode ffmpeg'да `-threads`/`nice` yo'q edi → bitta transcode 4 yadroni to'liq egallab,
jonli darsга (LiveKit SFU/API) ta'sir qilishi mumkin edi. Q3 dastlab "serverда transcode = kengaymaydi"
degan edi — endi transcode **BOR**, shuning uchun uni **cheklaymiz** (o'chirmaymiz):
- **`-threads 2`** (4 yadroдан) — LiveKit/API uchun 2 yadro DOIM bo'sh.
- **`nice -n 19` + `ionice -c 3`** — eng past CPU/IO ustuvorlik; ffmpeg faqat bo'sh resursни oladi.
- **1 worker, navbatли** (avvaldан) — 100 yozuv bir vaqtда emas, bittalab.
Natija: server band bo'lsa transcode sekinlashadi, **jonli dars sekinlashmaydi**. Sifat/hajm o'zgармаydi.
Kod: `internal/worker/transcode.go` (`Threads`, `NiceLevel`, `lowPriorityFFmpeg`).

---

## Q4 — O'quvchi kamerasi (video): opt-in

**Muammo:** o'quvchilar erkin kamera yoqsa, har video oqimi hammaga tarqaladi (N×N);
bir necha kamera 4-yadroli serverni to'ldirib dars sifati/kechikishini buzadi (video —
eng qimmat oqim). Bu YOZUV emas, jonli dars videosi bo'yicha qaror, lekin bevosita
video sifatiga taalluqli.

### ❌ Rad etilgan variant A: "Mikrofon HAM, kamera HAM ustoz ruxsati bilan"
**Nega rad etildi:** o'quvchi tushunmagan joyni **darhol OVOZda so'ray olishi** kerak —
har savol uchun ustozdan ruxsat so'rab o'tirish dars oqimini buzadi. Asoschi qarori:
ovoz erkin bo'lsin.
**Qaytmaslik sharti:** doim kuchda — ovoz arzon (~30 kbps + DTX), uni cheklashning
sig'im foydasi deyarli yo'q, UX zarari katta.

### ✅ Tanlangan: Mikrofon ERKIN, faqat KAMERA ustoz ruxsati bilan (Variant B)
O'quvchi mikrofonni O'ZI yoqadi (savolni darhol beradi); kamera uchun qo'l ko'taradi,
ustoz "Videoga ruxsat" beradi (bergач `ParticipantPermissionsChanged` klientda
kamera tugmasini yoqadi). **Sabab:** qimmat qism — video (N×N), ovoz arzon; xona toza
1→N vebinar bo'lib qoladi → sifat/kechikish barqaror, ovozli savol esa darhol.
**Kod:** `backend/.../livekit/token.go` (o'quvchi `CanPublish=true`, manba faqat
mikrofon), `participant.go` (`studentBaseSources`=[mic] / `studentVideoSources`=[kamera,mic]),
frontend `useRoom.js` `canPublishCamera`, `Controls.jsx` (kamera shu bayroqда), `ParticipantsPanel.jsx`
("Videoga ruxsat"). Backend + frontend testlar o'tdi. Ekran ulashish baribir faqat ustozda.

---

## Yakuniy tanlangan yo'nalish (qisqacha)

1. **Korimlilik:** gorizontal dars (Q1-B) — letterbox to'ri qurilgan, gorizontalда qora qo'shmaydi.
2. **Audio:** playback ovozини yozish + bo'sh-trek bug tuzatiladi.
3. **Siqish:** dars tugагач telefonда (post-Stop), serverда yuk yo'q.
4. **Qo'shimcha:** jonli CBR→VBR ~1.2 Mbps (arzon yutuq).
5. **O'quvchi videosi:** opt-in — mikrofon erkin, kamera ustoz ruxsati bilan (Q4).

## Ochiq / tasdiqlanadigan
- Codec tanlash: H.264 CRF 26 (universal moslik) yoki HEVC CRF 28 (kichikroq,
  apparatли encoder). Tavsiya: HEVC, fallback H.264.
- Q1 yakuniy tasdiq: mentorlar haqiqatда gorizontalда o'tadimi (portret bo'lsa — Q1
  dinamik tuvalга qaytiladi).

## Hali qilinmagan test
- Ovozли (mikrofon yoniq) dars testi.
- Vertikal + gorizontal orientatsiya (73 daq test faqat portret bo'ldi — adb uzilgan).
