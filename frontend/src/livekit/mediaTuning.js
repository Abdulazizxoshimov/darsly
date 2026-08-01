import { AudioPresets, VideoPreset, VideoPresets } from 'livekit-client'

/**
 * Media sifati sozlamalari — **past internetli hududlar uchun**.
 *
 * Bu fayl `mobile/.../data/livekit/MediaTuning.kt` ning web ekvivalenti va
 * ATAYLAB u bilan bir xil qarorlarni oshkora takrorlaydi. Ikki klient bir xil
 * xulq ko'rsatishi shart: aks holda "ustoz telefondan ulashsa ko'rinadi,
 * kompyuterdan ulashsa ko'rinmaydi" degan tushuntirib bo'lmaydigan farq chiqadi.
 *
 * ## Barcha raqamlar O'LCHANGAN (taxmin emas)
 * Vosita: `tools/media-probe/` — nashr qiluvchi canvas'dan slayd/kod chizadi,
 * obunachi dekodlangan kadrni ASL kadr bilan solishtiradi (PSNR + qirra
 * energiyasi) va "shishadan-shishagacha" kechikishni o'lchaydi. To'liq jadval:
 * `docs/latency-quality-report.md`. SDK: `livekit-client@2.21.0`.
 */

/**
 * ⭐ EKRAN NARVONI — uchala qatlam ham MANBA O'LCHAMIDA, farq faqat kadr
 * chastotasi va bitreytda.
 *
 * ## Nima noto'g'ri edi (o'lchov bilan)
 * Avvalgi narvon ikki qatlamli edi: 1080p/1.5 Mbps va **640×360 / 3 fps / 200 kbps**.
 * Zaif tarmoqdagi o'quvchi pastki qatlamga tushardi va u yerda:
 *
 * | qatlam | o'lcham | PSNR | qirra energiyasi (1.0 = asl) |
 * |---|---|---|---|
 * | yuqori | 1920×1080 | 39.3 dB | 0.97 |
 * | **past (eski)** | **640×360** | **17.0 dB** | **0.34** |
 *
 * 0.34 degani — matn qirralarining **66 % i yo'q**. Ya'ni "sifat past" emas,
 * balki **matn umuman o'qilmaydi**. Sabab bitreyt EMAS: 1080p → 360p uch
 * barobar kichraytirish, 14 px shrift ~5 px ga aylanadi va uni hech qanday
 * bitreyt tiklamaydi. O'lchov buni tasdiqladi — o'sha 360p qatlamga bitreytni
 * 200 → 400 kbps ga ko'tarish natijani YAXSHILAMADI (0.34 → 0.39).
 *
 * ## Yechim: o'lchamni saqlab, KADRni qurbon qilish
 * Bir xil ~750 kbps byudjetda uch variant o'lchandi:
 *
 * | o'rta qatlam | PSNR | qirra energiyasi |
 * |---|---|---|
 * | 1280×720 @ 15 fps | 24.3 dB | 0.42 |
 * | 1600×900 @ 12 fps | 25.8 dB | 0.49 |
 * | **1920×1080 @ 8 fps** | **33.7 dB** | **0.93** |
 *
 * Slayd va kod uchun javob bir ma'noli: piksel > kadr. Shuning uchun uchala
 * qatlam ham manba o'lchamida qoladi, faqat fps va bitreyt kamayadi.
 *
 * ## Nega presetlar 1920×1080 deb yozilgan
 * SDK `scaleResolutionDownBy = max(1, min(manba) / min(preset))` deb hisoblaydi
 * (`publishUtils` — o'qib tekshirilgan). 1080 dan kichik ekranda nisbat 1 dan
 * kichik bo'lib, `max(1, …)` uni 1 ga qaytaradi — ya'ni **kichraytirish yo'q**.
 * 1440p/4K ulashilganda esa past ikki qatlam 1080p ga tushadi, bu esa aynan
 * kerakli xulq. Ya'ni "1920×1080" bu yerda cheklov emas, **ustki chegara**.
 *
 * ## Narxi (halol)
 * Uchta qatlam ikkitaga nisbatan nashr qiluvchining enkoder yukini ~0.20 →
 * ~0.40 yadroga oshiradi (o'lchangan, VP8 dasturiy enkoder). `dynacast: true`
 * bo'lgani uchun obunachisi yo'q qatlam to'xtatiladi — amaldagi narx bundan past.
 */
const SCREEN_CAP = { width: 1920, height: 1080 }

/**
 * Yuqori qatlam — yaxshi internet. 1.5 Mbps ATAYLAB oshirilmadi: 2.5 Mbps ga
 * ko'tarish PSNR'ni 39 → 45 dB qiladi, lekin qirra energiyasi allaqachon 0.97 —
 * ya'ni matn o'qilishi yaxshilanmaydi, faqat kanal sarflanadi (300 o'quvchi).
 */
export const SCREEN_HIGH = new VideoPreset({
  ...SCREEN_CAP,
  maxBitrate: 1_500_000,
  maxFramerate: 15,
  priority: 'high',
})

/**
 * O'rta qatlam — o'rtacha 4G / uy Wi-Fi'si (~1 Mbps). Eski narvonda bu qatlam
 * YO'Q EDI: 200 kbps va 1500 kbps orasida hech narsa bo'lmagani uchun 600 kbps
 * li o'quvchi to'g'ridan-to'g'ri o'qib bo'lmaydigan 360p ga tushardi.
 */
export const SCREEN_MID = new VideoPreset({
  ...SCREEN_CAP,
  maxBitrate: 800_000,
  maxFramerate: 8,
  priority: 'high',
})

/**
 * Past qatlam — zaif 3G/4G. 3 fps slayd va kod uchun yetarli (kontent statik),
 * muhimi — matn O'QILADI: qirra energiyasi 0.83 (eski narvonda 0.34).
 */
export const SCREEN_LOW = new VideoPreset({
  ...SCREEN_CAP,
  maxBitrate: 300_000,
  maxFramerate: 3,
  priority: 'high',
})

/**
 * KAMERA qatlamlari — ekrandan ATAYLAB past.
 *
 * Mahsulot qoidasi (docs/PRODUCT.md): kanal torayganda birinchi bo'lib KAMERA
 * qurbon bo'ladi, ekran (slayd/misol) va ovoz saqlanadi. Kamera 360p dan
 * yuqoriga chiqmaydi: dars mazmuni ekranda, kamera esa "gapirayotgan odam"
 * konteksti — u uchun HD trafik sarflash noto'g'ri savdo.
 */
export const CAMERA_HIGH = VideoPresets.h360
/**
 * Kameraning past qatlami. `priority: 'low'` AYNAN shu qatlamda turishi shart —
 * pastdagi izohga qarang (Chrome faqat 0-indeksdagi qatlam ustuvorligini oladi).
 * O'lchamlar SDK'ning `VideoPresets.h180` idan aynan ko'chirildi.
 */
export const CAMERA_LOW = new VideoPreset({
  width: 320,
  height: 180,
  maxBitrate: 160_000,
  maxFramerate: 20,
  priority: 'low',
})

/**
 * Xona uchun publish default'lari.
 *
 * ## Trek ustuvorligi: ovoz > ekran > kamera — bu avval ISHLAMASDI
 * SDK kodidan o'qildi (taxmin emas), `encodingsFromPresets`:
 *
 *     canSetPriority = (browser === 'Firefox' && os !== 'iOS') || idx === 0
 *
 * Ya'ni Chrome'da ustuvorlik FAQAT eng past qatlamdan (`idx === 0`) olinadi.
 * Avvalgi kodda `priority: 'high'` asosiy (eng yuqori) qatlamga qo'yilgan edi —
 * u Chrome'da **jimgina tashlab yuborilardi**, va "ekran kameradan ustun" degan
 * mahsulot qoidasi amalda umuman qo'llanmasdi. Endi ustuvorlik narvonning
 * BIRINCHI presetida ([SCREEN_LOW] va [CAMERA_LOW]) turadi.
 *
 * ## Ovoz: RED va DTX
 * · **RED** (RFC 2198) — har paketda oldingi paket nusxasi ketadi; paket yo'qolsa
 *   ovoz uzilmaydi. Narxi ~2× audio bitrate, lekin audio umumiy oqimning kichik qismi.
 * · **DTX** — jim paytda deyarli hech narsa yuborilmaydi; ustoz gapirmayotganda
 *   kanal ekran uchun bo'shaydi.
 * SDK'da ikkalasi ham default yoniq, lekin **jimgina**: kimdir kelajakda
 * `publishDefaults`ni boshqa sabab bilan qayta yozsa ular beixtiyor o'chib ketardi.
 *
 * ## Kodek ATAYLAB o'zgartirilmagan (vp8 — SDK default'i) — ENDI O'LCHOV BILAN
 * VP9/AV1 past bitreytda yaxshiroq deb hisoblanadi. Ekran-ulashish kontentida
 * bu **tasdiqlanmadi** (`tools/media-probe`, 1280×720 manba, bir xil bitreyt):
 *
 * | bitreyt | VP8 | VP9 | AV1 | H.264 |
 * |---|---|---|---|---|
 * | 200 kbps | 29.1 dB / 14 fps | 26.8 dB / **1 fps** | 27.3 dB / 3 fps | 36.2 dB / 15 fps |
 * | 400 kbps | 39.4 dB | 38.9 dB | 40.3 dB | 39.2 dB |
 * | 1500 kbps | 43.8 dB (346 kbps sarfladi) | 44.0 dB (**1499 kbps sarfladi**) | 42.9 dB | 39.1 dB |
 *
 * Uch sabab bilan VP8 qoladi: (a) VP9 past bitreytda kadr chastotasini 1 fps ga
 * tushiradi — ekran muzlagandek ko'rinadi; (b) VP9/AV1 kechikishni oshiradi
 * (p50 86–234 ms, VP8 esa 56–106 ms); (c) VP9 kanalni to'liq yeydi, VP8 esa
 * kerak bo'lmagan bitreytni olmaydi (1.5 Mbps ruxsatdan 346 kbps ishlatdi) —
 * 300 o'quvchili darsda bu to'g'ridan-to'g'ri server kanalini tejaydi.
 * SVC'da simulcast ham o'chadi, ya'ni yuqoridagi uch qatlamli narvon yo'qoladi.
 */
export const PUBLISH_DEFAULTS = {
  simulcast: true,
  screenShareEncoding: SCREEN_HIGH.encoding,
  // Tartib: pastdan yuqoriga (SDK talabi). IKKITA qo'shimcha qatlam berilishi
  // uchala qatlamni yoqadi — bitta berilsa SDK faqat ikki qatlam yasaydi.
  screenShareSimulcastLayers: [SCREEN_LOW, SCREEN_MID],
  // Kamera: past qatlam bilan — zaif o'quvchi kamerani butunlay yo'qotmaydi.
  videoEncoding: CAMERA_HIGH.encoding,
  videoSimulcastLayers: [CAMERA_LOW],
  // Ovoz: nutq preseti (SDK 2.21.0 da 24 kbps) — musiqa emas, dars ovozi.
  audioPreset: AudioPresets.speech,
  red: true,
  dtx: true,
  // ⚠️ `degradationPreference` BU YERDA ATAYLAB YO'Q — pastdagi izohga qarang.
}

/**
 * Har trek uchun ALOHIDA degradatsiya siyosati.
 *
 * ## Nega xona darajasida bo'lmasligi kerak
 * Avval `publishDefaults.degradationPreference = 'maintain-resolution'` edi.
 * `publishDefaults` — XONA darajasidagi sozlama, ya'ni u KAMERAGA ham tushardi.
 * Natijada zaif tarmoqda kamera o'lchamini kamaytirish o'rniga **kadrlarni
 * tashlardi**: ustozning yuzi muzlab qolardi. Mobil ilova esa kamerada
 * `BALANCED` ishlatardi — ikki platforma bir-biriga zid xulq ko'rsatardi.
 *
 * Endi siyosat trek turiga bog'liq va ikkala platformada bir xil:
 * · ekran → `maintain-resolution` (matn o'qilishi shart, kadr qurbon bo'ladi)
 * · kamera → `balanced` (yuz uchun ravon harakat o'lchamdan muhimroq)
 */
export const SCREEN_PUBLISH = { degradationPreference: 'maintain-resolution' }
export const CAMERA_PUBLISH = { degradationPreference: 'balanced' }

/**
 * Ekran ulashishni qo'lga olish sozlamalari.
 *
 * · `audio: true` — brauzer tab/tizim ovozini ham ulashish imkonini beradi
 *   (ustoz video yoki animatsiya ko'rsatganda ovozsiz qolmasin).
 * · `contentHint: 'text'` — enkoderga "bu matn" deydi: keskinlik fps'dan muhimroq.
 * · `resolution` OSHKORA berilgan: SDK default'i ham 1080p, lekin matn sifati
 *   to'g'ridan-to'g'ri shu raqamga bog'liq bo'lgani uchun u tasodifan
 *   o'zgarib ketmasligi kerak. 1080p dan yuqorisi olinmaydi — 1440p/4K
 *   enkoder yukini bir necha barobar oshiradi, o'qilishga qo'shadigani esa oz.
 * · `systemAudio: 'include'` / `surfaceSwitching: 'include'` — Chrome tanlash
 *   oynasida tegishli variantlarni ko'rsatadi (boshqa brauzerlarda e'tiborsiz).
 * · `selfBrowserSurface: 'exclude'` — ustoz tasodifan darsning o'z tabini ulashib,
 *   cheksiz "oyna ichida oyna" effektini yasab qo'ymasligi uchun.
 */
export const SCREEN_CAPTURE = {
  audio: true,
  contentHint: 'text',
  resolution: { width: 1920, height: 1080, frameRate: 15 },
  systemAudio: 'include',
  surfaceSwitching: 'include',
  selfBrowserSurface: 'exclude',
}
