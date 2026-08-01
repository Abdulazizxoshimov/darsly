// Mahsulot qat'iy qiymatlari — profilda SOZLANMAYDI.
// Sabab: ilova butunlay o'zbekcha va foydalanuvchilar O'zbekistonda. Erkin matn
// maydonlari bo'lganda noto'g'ri mintaqa dars vaqtlarini surib yuborardi.
export const DEFAULT_TIMEZONE = 'Asia/Tashkent'
export const DEFAULT_LANGUAGE = 'uz'

// Ism initsiallari: "Malika Yusupova" → "MY"
export function initials(name) {
  const parts = String(name || '').trim().split(/\s+/).filter(Boolean)
  if (!parts.length) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

export function formatDateTime(iso) {
  if (!iso) return 'Vaqt belgilanmagan'
  return new Date(iso).toLocaleString('uz', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function formatTime(iso) {
  if (!iso) return '--:--'
  return new Date(iso).toLocaleTimeString('uz', { hour: '2-digit', minute: '2-digit' })
}

export function formatDay(iso) {
  if (!iso) return 'Vaqt belgilanmagan'
  return new Date(iso).toLocaleDateString('uz', { weekday: 'long', day: 'numeric', month: 'long' })
}

export function formatSize(bytes) {
  if (!bytes) return '0 MB'
  return (bytes / 1024 / 1024).toFixed(1) + ' MB'
}

export function formatDuration(sec) {
  return Math.round((sec || 0) / 60) + ' daq'
}

// Pleyer vaqt belgisi: 125 → «2:05», 3725 → «1:02:05».
// Chat xabari yonidagi tugmada shu ko'rinishda chiqadi — foydalanuvchi uni
// videoning shkalasidagi vaqt bilan bir qarashda solishtiradi.
export function formatOffset(sec) {
  const t = Math.max(0, Math.floor(Number(sec) || 0))
  const h = Math.floor(t / 3600)
  const m = Math.floor((t % 3600) / 60)
  const s = t % 60
  const pad = (n) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`
}

// Chat fayllari uchun — `formatSize` dan farqi: kichik fayl «0.0 MB» emas,
// «84 KB» bo'lib chiqadi (rasm/hujjatlar odatda megabaytdan kichik).
export function formatBytes(bytes) {
  const b = Number(bytes) || 0
  if (b < 1024) return `${b} B`
  if (b < 1024 * 1024) return `${Math.round(b / 1024)} KB`
  return `${(b / 1024 / 1024).toFixed(1)} MB`
}

// ── Yozuv saqlanish muddati (retention, 30 kun) ─────────────────────────────
// Backend `expires_at` beradi (faqat `ready` yozuvda). Ustoz «yozuv qachongacha
// turadi» degan savolga bir qarashda javob topishi kerak — aks holda muhim
// darsni yuklab olishni unutib, keyin uni topmay qoladi.

/** `expires_at` gacha qolgan kun (yuqoriga yaxlitlangan) yoki `null`. */
export function expiresInDays(iso, now = Date.now()) {
  if (!iso) return null
  const ts = new Date(iso).getTime()
  if (Number.isNaN(ts)) return null
  return Math.ceil((ts - now) / 86_400_000)
}

export function formatExpiry(iso, now = Date.now()) {
  const d = expiresInDays(iso, now)
  if (d === null) return null
  // Muddat allaqachon o'tgan (fon ishchisi hali yetib bormagan) — "Bugun
  // o'chadi" deyish noto'g'ri va'da bo'lardi: fayl istalgan daqiqada ketadi.
  if (d < 0) return 'Muddati tugagan'
  if (d === 0) return 'Bugun o‘chadi'
  if (d === 1) return 'Ertaga o‘chadi'
  return `${d} kundan keyin o‘chadi`
}

// Muddat yaqinlashganda ogohlantirish rangi (server 3 kun qolganda
// bildirishnoma yuboradi — UI shu chegara bilan izchil bo'lsin).
export const EXPIRY_WARN_DAYS = 3

export const LESSON_STATUS_UZ = {
  scheduled: 'Rejalashtirilgan',
  live: 'Jonli',
  ended: 'Tugagan',
  cancelled: 'Bekor qilingan',
}

export const RECORDING_STATUS_UZ = {
  recording: 'Yozilmoqda…',
  processing: 'Tayyorlanmoqda…',
  ready: 'Tayyor',
  failed: 'Xatolik',
  // Saqlash muddati (30 kun) tugagan — fayl o'chirilgan, qator tarix uchun qoladi.
  expired: 'Muddati tugagan',
  // Telegram arxivi YOQILGAN bo'lsa 30 kundan keyin `expired` emas, `archived`
  // bo'ladi: fayl serverdan ketgan, lekin Telegramda turibdi va qaytarib
  // olinadi. Foydalanuvchi uchun bu «yo'qolgan» EMAS, «uzoqroqda».
  archived: 'Telegram arxivida',
  restoring: 'Tiklanmoqda…',
}

// Rol nomlari. Backend xom qiymat qaytaradi (`mentor`/`student`/`admin`) va u
// UI'da to'g'ridan-to'g'ri ko'rsatilardi — foydalanuvchi profilida inglizcha
// "mentor" deb turardi. Noma'lum rol uchun xom qiymat qoladi (yangi rol
// qo'shilsa bo'sh joy ko'rinmasin).
export const ROLE_UZ = {
  mentor: 'Ustoz',
  student: "O'quvchi",
  admin: 'Administrator',
  guest: 'Mehmon',
}

export function roleLabel(role) {
  return ROLE_UZ[role] || role || ''
}
