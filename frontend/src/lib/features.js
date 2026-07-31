// Feature-bayroqlar (build vaqtida Vite env'dan). Backend sozlamasi bilan mos bo'lishi kerak.
//
// EMAIL_ENABLED: backend'dagi EMAIL_ENABLED bilan mos. SMTP o'chiq bo'lganda (default)
// parol-tiklash xatlari jimgina tashlanadi — shuning uchun "Parolni unutdim" oqimi
// UI'da yashiriladi. SMTP yoqilganda `VITE_EMAIL_ENABLED=true` qilinsa oqim qaytadi.
export const EMAIL_ENABLED = import.meta.env.VITE_EMAIL_ENABLED === 'true'

// Brauzer jonli darsni umuman ololadimi.
//
// Nega kerak: qo'llab-quvvatlanmagan brauzerda (eski iOS WebView, ba'zi
// in-app brauzerlar — Telegram/Instagram ichidagi) `room.connect()` tushunarsiz
// xato bilan yiqiladi va foydalanuvchi "Xonaga ulanib bo'lmadi" degan umumiy
// xabarni ko'radi. U esa muammo O'ZIDA emas, BRAUZERDA ekanini bilmaydi va
// qayta-qayta urinaveradi.
//
// Bu yerda ATAYLAB `livekit-client` ning `isBrowserSupported()` si emas, xom
// tekshiruv ishlatiladi: `features.js` ilovaning asosiy chunk'ida, LiveKit esa
// faqat xona chunk'ida (~150KB gzip). Tekshiruv uchun butun SDK'ni tortish
// noto'g'ri savdo bo'lardi.
export function isLiveRoomSupported({ needsPublish = true } = {}) {
  if (typeof window === 'undefined') return true // SSR/test — bloklamaymiz
  const rtcOk = typeof window.RTCPeerConnection === 'function'
  if (!needsPublish) {
    // O'quvchi uchun `mediaDevices` shart emas — gate YUMSHOQ (Zoom modelida
    // o'quvchi publish qila oladi, lekin qilmasa ham dars ko'rish/eshitish
    // to'liq ishlaydi). Bu ataylab: `navigator.mediaDevices` XAVFSIZ BO'LMAGAN
    // originda (http + IP/domen, localhost emas) brauzer tomonidan UMUMAN
    // berilmaydi — dev/LAN sinovida o'quvchi shu tekshiruvda noto'g'ri
    // "brauzer eski" xabariga urilardi (2026-07-30). Bunday holatda o'quvchi
    // XONAGA KIRADI, faqat mikrofon/kamera tugmalari «HTTPS kerak» izohi
    // bilan o'chiq turadi (`Controls.mediaOk`).
    return rtcOk
  }
  return (
    rtcOk &&
    typeof navigator !== 'undefined' &&
    !!navigator.mediaDevices &&
    typeof navigator.mediaDevices.getUserMedia === 'function'
  )
}
