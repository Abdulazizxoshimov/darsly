// Xona ekranining SOF mantig'i — React'siz, LiveKit SDK'siz.
//
// Nega alohida fayl: bu yerdagi qarorlar (nima qachon qayta render bo'ladi, aloqa
// indikatori nima deydi, qo'l navbati qanday tartiblanadi) mahsulot qoidalari.
// Ular komponent ichida yashirilsa faqat brauzerda, qo'lda tekshiriladi — ya'ni
// amalda umuman tekshirilmaydi. Bu yerda esa ular oddiy testga tushadi.
// (Mobil ilovadagi `ScreenAudioPolicy`/`MediaTuning` bilan bir xil yondashuv.)

import { ConnectionError, ConnectionErrorReason, ConnectionQuality, DisconnectReason } from 'livekit-client'

// ─── useRoom klassifikatorlari (sof — React'siz, shuning uchun shu yerda) ─────
// Bu uch qaror `useRoom` ichida yashiringan edi va faqat brauzerda, real
// LiveKit hodisasi bilan tekshirilardi — ya'ni amalda hech qachon. Ular
// mahsulot/xavfsizlik qoidalari, shuning uchun `useRoom` xatti-harakatini
// o'zgartirmasdan bu yerga ko'chirildi va `roomLogic.test.js` da sinaladi.

/**
 * Kamera (video) publish ruxsati bormi. `permissions.canPublishSources` —
 * LiveKit protokol enum'i (CAMERA=1); bo'sh ro'yxat = BARCHA manbalar (host).
 * O'quvchida default FAQAT mikrofon — kamera ustoz ruxsatidan keyin qo'shiladi
 * (backend `studentVideoSources`). Ehtiyot uchun string ('camera') ham qabul.
 *
 * Xavfsizlik-tegishli: bu funksiya "o'quvchi kamerasini yoqa oladimi?" savoliga
 * javob beradi; noto'g'ri `true` — ustoz ruxsat bermagan holda kamera yonishi.
 */
export function canPublishCameraOf(perms) {
  if (!perms?.canPublish) return false
  const src = perms.canPublishSources
  if (!src || src.length === 0) return true // bo'sh = hammasi (host)
  return src.some((s) => s === 1 || s === 'camera')
}

/**
 * Uzilish "yakuniy"mi (server xona yopdi / chiqarib yubordi) yoki vaqtinchalik
 * (tarmoq uzildi — LiveKit o'zi qayta ulanadi). Avval HAR QANDAY `disconnected`
 * guest'ni sessiyasi bilan bosh sahifaga uloqtirardi — vaqtinchalik uzilishda ham.
 */
export function endedByServer(reason) {
  return (
    reason === DisconnectReason.ROOM_DELETED ||
    reason === DisconnectReason.ROOM_CLOSED ||
    reason === DisconnectReason.PARTICIPANT_REMOVED ||
    reason === DisconnectReason.DUPLICATE_IDENTITY
  )
}

/**
 * Uzilish sababi → UI holati. Xona ICHIDAGI chiqarib yuborish (kick) va sessiya
 * dublikati (boshqa qurilmada kirildi) bir-biridan ajratiladi — foydalanuvchiga
 * aniq sabab ko'rsatiladi. Vaqtinchalik uzilishda `null` (banner chiqmaydi).
 *   'removed'      — ustoz chiqarib yubordi (PARTICIPANT_REMOVED)
 *   'duplicate'    — xuddi shu identity boshqa joyda ulandi (DUPLICATE_IDENTITY)
 *   'room_deleted' — xona yopildi (ROOM_DELETED / ROOM_CLOSED)
 */
export function endedReason(reason) {
  if (!endedByServer(reason)) return null
  if (reason === DisconnectReason.PARTICIPANT_REMOVED) return 'removed'
  if (reason === DisconnectReason.DUPLICATE_IDENTITY) return 'duplicate'
  return 'room_deleted'
}

/**
 * `room.connect()` xatosi → nima qilish kerak.
 *   'auth'    — server tokenni rad etdi (muddati o'tgan/yaroqsiz): qayta urinish
 *               BEFOYDA, avval token yangilanadi (`lib/roomToken`).
 *   'network' — server yetib bo'lmadi / timeout / WS uzildi: chegaralangan
 *               backoff bilan qayta urinish mumkin.
 *   null      — biz o'zimiz bekor qildik (effekt tozalandi): hech narsa qilinmaydi.
 *
 * Avval xato butunlay yutilardi va ekran har 5 soniyada AYNI eskirgan token
 * bilan «Dars davom etmoqda» deb abadiy urinardi.
 */
export function connectFailure(err) {
  if (err instanceof ConnectionError) {
    if (err.reason === ConnectionErrorReason.NotAllowed) return 'auth'
    if (err.reason === ConnectionErrorReason.Cancelled || err.reason === ConnectionErrorReason.LeaveRequest) {
      return null
    }
  }
  return 'network'
}

/** Avtomatik qayta ulanishlar soni — shundan keyin faqat qo'lda. */
export const MAX_AUTO_RECONNECT = 6
const RECONNECT_BASE_MS = 5_000
const RECONNECT_MAX_MS = 60_000

/**
 * Tarmoq uzilishida `attempt`-urinishgacha kutish (5s → 10s → 20s → 40s → 60s…).
 * `MAX_AUTO_RECONNECT` dan keyin `null` — avtomatik urinish TO'XTAYDI:
 * uzoq uzilishda serverga (va batareyaga) abadiy urib turmaymiz.
 */
export function reconnectDelay(attempt) {
  if (attempt >= MAX_AUTO_RECONNECT) return null
  return Math.min(RECONNECT_MAX_MS, RECONNECT_BASE_MS * 2 ** attempt)
}

/**
 * Kech kirgan ishtirokchiga FAOL so'rovnomani takrorlash xabari (host yuboradi).
 * `open` xabari kirishdan oldin ketgan bo'lsa o'quvchi bo'sh panel ko'rardi
 * va ovoz bera olmasdi. Faol so'rovnoma bo'lmasa `null`.
 */
export function pollReplayMessage(polls) {
  const active = (polls || []).find((p) => p && p.is_active)
  if (!active) return null
  return {
    kind: 'poll',
    action: 'open',
    poll: {
      id: active.id,
      question: active.question,
      options: active.options,
      results_visibility: active.results_visibility,
    },
  }
}

/**
 * Media publish natijasi → ko'rsatiladigan xato sinfi. Kamera va mikrofon
 * ALOHIDA kuzatiladi: kamera rad etilib mikrofon ishlashi odatiy holat va bunda
 * "kamera yoqilmadi" deb aniq aytish kerak, umumiy "xatolik" emas. Ikkalasi ham
 * ishlagan bo'lsa `null` (banner yo'q).
 */
export function mediaErrorClass(camFailed, micFailed) {
  if (camFailed && micFailed) return 'both'
  if (camFailed) return 'camera'
  if (micFailed) return 'mic'
  return null
}

/**
 * Ishtirokchilar snapshot'ining "mazmun imzosi".
 *
 * `useRoom` har LiveKit hodisasida snapshot yasaydi va imzoni eskisi bilan
 * solishtiradi; imzo bir xil bo'lsa React umuman xabardor qilinmaydi.
 *
 * ⚠️ Bu funksiyaga UI ko'rsatadigan HAR BIR maydon kirishi shart. Kirmay qolgan
 * maydon "ekranda eskirgan qiymat" degani bo'ladi va uni brauzerda tutish qiyin —
 * shuning uchun `roomLogic.test.js` da har maydon uchun alohida holat bor.
 */
export function participantSignature(items) {
  let s = ''
  for (const i of items) {
    s += `${i.identity}|${i.name}|${i.speaking ? 1 : 0}${i.micMuted ? 1 : 0}${i.canPublish ? 1 : 0}${
      i.canPublishCamera ? 1 : 0
    }${i.isHost ? 1 : 0}|${i.camTrack?.sid || '-'}|${i.screenTrack?.sid || '-'};`
  }
  return s
}

/** Bir sahifadagi maksimal plitka soni (Zoom andozasi: 3×3 galereya). */
export const GALLERY_PAGE_SIZE = 9

/**
 * Galereya tartibi: ustoz → o'zim → gapirayotganlar → qolganlar.
 *
 * Nega shunday: sahifalashda birinchi sahifa "muhimlar sahifasi" (Zoom andozasi) —
 * ustoz va hozir gapirayotganlar KO'RINIB turishi kerak, 3-sahifada yashirinib
 * qolmasligi kerak. Sort BARQAROR (Array.sort ES2019+ da stable), ya'ni bir xil
 * darajadagi ishtirokchilar kelish tartibini saqlaydi — plitkalar har hodisada
 * sakramaydi.
 */
export function galleryOrder(items) {
  const rank = (p) => (p.isHost ? 0 : p.isLocal ? 1 : p.speaking ? 2 : 3)
  return [...items].sort((a, b) => rank(a) - rank(b))
}

/**
 * Galereya sahifasi: so'ralgan sahifani [0..total-1] ga qisadi va shu sahifa
 * plitkalarini qaytaradi. Ishtirokchi chiqib ketib sahifa "bo'sh qolsa" ham
 * chegaradan chiqmaydi — oxirgi mavjud sahifa ko'rsatiladi.
 *
 * Faqat JORIY sahifa plitkalari render qilinadi — ko'rinmagan plitkaning video
 * elementi DOM'da bo'lmaydi va LiveKit adaptiveStream u trekka obuna bo'lmaydi
 * (trafik tejash sahifalashning asosiy foydalaridan biri).
 */
export function galleryPage(items, page, size = GALLERY_PAGE_SIZE) {
  const total = Math.max(1, Math.ceil(items.length / size))
  const p = Math.min(Math.max(0, page), total - 1)
  return { items: items.slice(p * size, p * size + size), page: p, total }
}

/** Local media holatining imzosi (boshqaruv paneli shunga bog'lanadi). */
export function localSignature(l) {
  return `${l.identity}|${l.micOn ? 1 : 0}${l.camOn ? 1 : 0}${l.screenOn ? 1 : 0}${l.canPublish ? 1 : 0}${l.canPublishCamera ? 1 : 0}`
}

/** LiveKit `ConnectionQuality` → UI uchun sodda daraja. */
export function qualityLabel(q) {
  if (q === ConnectionQuality.Excellent || q === ConnectionQuality.Good) return 'good'
  if (q === ConnectionQuality.Poor) return 'poor'
  if (q === ConnectionQuality.Lost) return 'lost'
  return 'unknown'
}

/**
 * Aloqa indikatori matni.
 *
 * Avval bu yozuv HAR DOIM "Yaxshi" edi (faqat ulanish holatiga qarardi), ya'ni
 * foydalanuvchiga aynan tarmoq yomonlashganda yolg'on ko'rsatardi. Endi sifat
 * `ConnectionQuality` dan keladi va "Ulandi" (hali o'lchanmagan) bilan "Yaxshi"
 * (o'lchangan) ataylab ajratilgan — bilmaslikni bilishdek ko'rsatmaymiz.
 */
export function linkView(reconnecting, quality) {
  if (reconnecting) return { label: 'Ulanmoqda…', bad: true, hint: 'Serverga qayta ulanmoqda' }
  if (quality === 'poor') return { label: 'Zaif', bad: true, hint: 'Internet zaif — tejamkor rejimni yoqing' }
  if (quality === 'lost') return { label: 'Uzildi', bad: true, hint: 'Aloqa yo‘qoldi' }
  if (quality === 'unknown') return { label: 'Ulandi', bad: false, hint: 'Sifat hali o‘lchanmadi' }
  return { label: 'Yaxshi', bad: false, hint: 'Aloqa barqaror' }
}

/**
 * Qo'l ko'tarish hodisasini holatga qo'llaydi.
 *
 * Holat — `Map` (identity → {name, at}). Map insert tartibini saqlaydi, ya'ni
 * NAVBAT tartibi bepul keladi: kim birinchi so'ragan bo'lsa ro'yxatda birinchi turadi.
 *
 * Hech narsa o'zgarmasa **aynan o'sha Map** qaytariladi — React uchun bu "qayta
 * render kerak emas" degani (takroriy xabar butun xonani bezovta qilmasin).
 */
export function applyHandEvent(map, msg) {
  if (!msg) return map
  if (msg.act === 'lower_all') return map.size ? new Map() : map
  if (!msg.identity) return map

  const has = map.has(msg.identity)
  if (msg.raised) {
    if (has && map.get(msg.identity).name === (msg.name || msg.identity)) return map
    const next = new Map(map)
    next.set(msg.identity, { name: msg.name || msg.identity, at: msg.at || 0 })
    return next
  }
  if (!has) return map
  const next = new Map(map)
  next.delete(msg.identity)
  return next
}

/**
 * Oddiy tezlik cheklovchi (klient tomonda).
 *
 * Data-channel'da hech qanday cheklov yo'q edi: bitta o'quvchi emoji tugmasini
 * bosib turib butun xonaning ekranini to'ldira olardi. Server tomondagi cheklov
 * `roomstate` domeni bilan keladi; bu esa birinchi va eng arzon to'siq.
 *
 * `now` — ATAYLAB parametr: soatni funksiya ichida o'qish testni imkonsiz qilardi.
 */
export function rateLimiter(minIntervalMs, now = Date.now) {
  let last = -Infinity
  return function allow() {
    const t = now()
    if (t - last < minIntervalMs) return false
    last = t
    return true
  }
}

/**
 * Suzuvchi oyna (Document PiP) holat mashinasi.
 *
 * ## Nega mashina, nega oddiy `sharing && !closed` emas
 * Ikki talab bir-biriga qarama-qarshi:
 *   1) ekran ulashish boshlanishi bilan oyna O'ZI ochilsin (ustoz uni qidirmasin);
 *   2) ustoz oynani yopsa, u QAYTA OCHILMASIN (aks holda bezor qiladi).
 * Ikkinchisi birinchisini "o'chirib qo'ymasligi" kerak: keyingi ulashishda oyna
 * yana ochilishi lozim. Ya'ni "yopilgan" holat ulashish tugagach unutiladi.
 *
 * Holatlar: 'idle' (yopiq) · 'open' · 'dismissed' (ustoz yopgan — tegmaymiz)
 * Hodisalar: 'share_start' · 'share_stop' · 'user_close' · 'user_open'
 */
/**
 * "Chat ustozning KO'Z OLDIDAMI?" — o'qilmagan sanog'i shu javobga bog'liq.
 *
 * Ekran ulashilayotganda ustoz odatda brauzerda EMAS (PDF/kod/slayd oynasida),
 * ya'ni asosiy oynadagi ochiq chat paneli hech narsani ko'rsatmaydi. Shuning
 * uchun suzuvchi oyna ochiq bo'lsa yagona haqiqiy manba — o'sha oynadagi chat.
 *
 * Aks holda ikki xato bo'lardi:
 *   · PiP ochiq, asosiy panel ham 'chat' → badge umuman o'smaydi va ustoz
 *     yangi savolni butunlay o'tkazib yuboradi;
 *   · PiP chati ochiq turib badge o'sib borsa → ko'z oldidagi xabar uchun
 *     "o'qimadingiz" degan yolg'on signal.
 */
export function isChatVisible({ panel, pipOpen, pipChatOpen }) {
  if (pipOpen) return !!pipChatOpen
  return panel === 'chat'
}

export function nextPipState(state, event) {
  switch (event) {
    case 'share_start':
      // 'dismissed' dan ham ochiladi: yangi ulashish — yangi vaziyat.
      return 'open'
    case 'share_stop':
      return 'idle' // "yopgan edi" xotirasi shu yerda unutiladi
    case 'user_close':
      return state === 'open' ? 'dismissed' : state
    case 'user_open':
      return 'open'
    default:
      return state
  }
}
