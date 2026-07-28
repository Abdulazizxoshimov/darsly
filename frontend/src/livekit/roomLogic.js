// Xona ekranining SOF mantig'i — React'siz, LiveKit SDK'siz.
//
// Nega alohida fayl: bu yerdagi qarorlar (nima qachon qayta render bo'ladi, aloqa
// indikatori nima deydi, qo'l navbati qanday tartiblanadi) mahsulot qoidalari.
// Ular komponent ichida yashirilsa faqat brauzerda, qo'lda tekshiriladi — ya'ni
// amalda umuman tekshirilmaydi. Bu yerda esa ular oddiy testga tushadi.
// (Mobil ilovadagi `ScreenAudioPolicy`/`MediaTuning` bilan bir xil yondashuv.)

import { ConnectionQuality } from 'livekit-client'

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
    s += `${i.identity}|${i.name}|${i.speaking ? 1 : 0}${i.micMuted ? 1 : 0}${i.canPublish ? 1 : 0}|${
      i.camTrack?.sid || '-'
    }|${i.screenTrack?.sid || '-'};`
  }
  return s
}

/** Local media holatining imzosi (boshqaruv paneli shunga bog'lanadi). */
export function localSignature(l) {
  return `${l.identity}|${l.micOn ? 1 : 0}${l.camOn ? 1 : 0}${l.screenOn ? 1 : 0}${l.canPublish ? 1 : 0}`
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
