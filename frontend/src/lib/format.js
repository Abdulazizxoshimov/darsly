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
}
