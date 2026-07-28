import { ScreenSharePresets } from 'livekit-client'

/**
 * Media sifati sozlamalari — **past internetli hududlar uchun**.
 *
 * Bu fayl `mobile/.../data/livekit/MediaTuning.kt` ning web ekvivalenti va
 * ATAYLAB u bilan bir xil qarorlarni oshkora takrorlaydi. Ikki klient bir xil
 * xulq ko'rsatishi shart: aks holda "ustoz telefondan ulashsa ko'rinadi,
 * kompyuterdan ulashsa ko'rinmaydi" degan tushuntirib bo'lmaydigan farq chiqadi
 * (bu farq loyihada haqiqatan bor edi — mobil sozlangan, web default holicha qolgan).
 *
 * ## Barcha raqamlar SDK'dan o'qilgan, taxmin emas
 * `livekit-client@2.6.4` (`dist/livekit-client.esm.mjs`) dagi `ScreenSharePresets`:
 *
 * | Preset        | O'lcham   | FPS | Bitrate   |
 * |---------------|-----------|-----|-----------|
 * | `h360fps3`    | 640×360   | 3   | 200 kbps  |
 * | `h360fps15`   | 640×360   | 15  | 400 kbps  |
 * | `h720fps5`    | 1280×720  | 5   | 800 kbps  |
 * | `h720fps15`   | 1280×720  | 15  | 1.5 Mbps  |
 * | `h1080fps15`  | 1920×1080 | 15  | 2.5 Mbps  |
 *
 * Bu qiymatlar Android SDK'sidagilar bilan aynan bir xil — ya'ni ikki platforma
 * bir xil qatlamlarni e'lon qiladi va SFU tomonda ular bir xil ko'rinadi.
 */

/** Ekran ulashishning asosiy (yuqori) qatlami — matn keskinligi shu yerda. */
export const SCREEN_HIGH = ScreenSharePresets.h720fps15

/**
 * Ekran ulashishning **past** qatlami — zaif internet uchun.
 *
 * 640×360 @ 3 fps / 200 kbps. Uch kadr sekundiga slayd va kod uchun yetarli:
 * kontent statik, ya'ni fps emas, **o'qilishi** muhim. 200 kbps kuchsiz 3G'da ham
 * o'tadi (qurilma sinovida operator kanali ~300 KB/s o'lchangan).
 */
export const SCREEN_LOW = ScreenSharePresets.h360fps3

/**
 * Xona uchun publish default'lari.
 *
 * ## Ekran ulashish — SIMULCAST bilan (bu yerdagi asosiy qaror)
 * Bitta qatlam bilan e'lon qilingan ekran o'quvchi 1.5 Mbit/s ni ko'tara olmasa
 * pastroq sifatga **tusha olmaydi** — u ekranni umuman ko'rmaydi. Ekran ulashish
 * esa bu mahsulotning asosiy mazmuni. `screenShareSimulcastLayers` bilan qatlamlarni
 * o'zimiz tanlaymiz: yuqorisi tegilmagan 720p/1.5 Mbps, pastkisi 200 kbps —
 * ya'ni yaxshi internetdagi o'quvchi avvalgidek 720p oladi, yomonidagi esa
 * ekranni ko'rmay qolish o'rniga 360p oladi.
 *
 * ## `maintain-resolution` — matn uchun hal qiluvchi
 * Kanal torayganda WebRTC nimadandir voz kechishi kerak: **o'lchamdan** yoki
 * **kadr chastotasidan**. Matn uchun javob aniq — o'lcham saqlanadi, fps tushadi.
 * SDK ekran uchun default'da ham shuni tanlaydi (`options.d.ts` izohi), lekin biz
 * uni OSHKORA yozamiz: bu qaror mahsulotga tegishli, SDK default'i esa ertaga
 * o'zgarishi mumkin va buni hech kim sezmasdi.
 *
 * ## Ovoz: RED va DTX
 * · **RED** (RFC 2198) — har paketda oldingi paket nusxasi ketadi; paket yo'qolsa
 *   ovoz uzilmaydi. Narxi ~2× audio bitrate, lekin audio umumiy oqimning kichik qismi.
 * · **DTX** — jim paytda deyarli hech narsa yuborilmaydi; ustoz gapirmayotganda
 *   kanal ekran uchun bo'shaydi.
 * SDK'da ikkalasi ham default yoniq, lekin **jimgina**: kimdir kelajakda
 * `publishDefaults`ni boshqa sabab bilan qayta yozsa ular beixtiyor o'chib ketardi
 * va buni faqat yomon tarmoqda — aynan eng muhim paytda — sezish mumkin bo'lardi.
 *
 * ## Kodek ATAYLAB o'zgartirilmagan (vp8 — SDK default'i)
 * VP9/AV1 past bitrate'da sifat beradi, lekin: (a) SVC'da simulcast o'chadi —
 * yuqoridagi ikki qatlamli yechim yo'qoladi; (b) arzon Android telefonlarda
 * dekodlash CPU'ni yeydi (bizning asosiy auditoriya). O'lchovsiz almashtirilmaydi.
 * O'zgartirmoqchi bo'lsangiz — avval `tests/load` bilan o'lchang.
 */
export const PUBLISH_DEFAULTS = {
  simulcast: true,
  screenShareEncoding: SCREEN_HIGH.encoding,
  screenShareSimulcastLayers: [SCREEN_LOW],
  degradationPreference: 'maintain-resolution',
  red: true,
  dtx: true,
}

/**
 * Ekran ulashishni qo'lga olish sozlamalari.
 *
 * · `audio: true` — brauzer tab/tizim ovozini ham ulashish imkonini beradi
 *   (ustoz video yoki animatsiya ko'rsatganda ovozsiz qolmasin). Mobil ilovada bu
 *   allaqachon bor (`ScreenAudioPlan`), webda yo'q edi.
 * · `contentHint: 'text'` — enkoderga "bu matn" deydi: keskinlik fps'dan muhimroq.
 * · `systemAudio: 'include'` / `surfaceSwitching: 'include'` — Chrome tanlash
 *   oynasida tegishli variantlarni ko'rsatadi (boshqa brauzerlarda e'tiborsiz).
 * · `selfBrowserSurface: 'exclude'` — ustoz tasodifan darsning o'z tabini ulashib,
 *   cheksiz "oyna ichida oyna" effektini yasab qo'ymasligi uchun.
 */
export const SCREEN_CAPTURE = {
  audio: true,
  contentHint: 'text',
  systemAudio: 'include',
  surfaceSwitching: 'include',
  selfBrowserSurface: 'exclude',
}
