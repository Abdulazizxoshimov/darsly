import { api, ApiError } from './api'

// Chat ikki yo'l bilan ishlaydi va bu ataylab:
//
//  · XONA yo'li (`/rooms/...`) — LiveKit room-token bilan. Host ham, o'quvchi ham
//    dars ichida shundan foydalanadi: bitta kod yo'li, bitta xatti-harakat.
//  · HOST yo'li (`/lessons/...`) — JWT bilan. Dars TUGAGANDAN keyin ham tarixni
//    o'qish (va moderatsiya qilish) uchun kerak — room-token faqat dars davomida yaroqli.
//
// Real-vaqt yetkazish serverdan LiveKit data-channel orqali keladi — bu
// funksiyalarning javobi faqat optimistik ko'rsatish uchun.

// ── Xona yo'li (dars ichida) ────────────────────────────────────────────────

export function roomChatHistory(lessonId, token, { before, limit } = {}) {
  const qs = new URLSearchParams({ token })
  if (before) qs.set('before', before)
  if (limit) qs.set('limit', limit)
  return api.get(`/rooms/${lessonId}/chat?${qs.toString()}`, { auth: false })
}

// to — qabul qiluvchi identity (shaxsiy xabar). Bo'sh/undefined = hammaga.
export function roomChatSend(lessonId, token, body, to) {
  return api.post(`/rooms/${lessonId}/chat`, { token, body, to: to || '' }, { auth: false })
}

// Fayl — xona yo'li. Token QUERY'da: shunda server 20 MB tanani o'qimasdan
// TURIB yaroqsiz tokenni rad etadi (kontraktda ham shu tavsiya etilgan).
export function roomChatUpload(lessonId, token, file, { body, to, onProgress, signal } = {}) {
  const fd = new FormData()
  fd.append('file', file)
  if (body) fd.append('body', body)
  if (to) fd.append('to', to)
  return api.upload(`/rooms/${lessonId}/chat/upload?token=${encodeURIComponent(token)}`, fd, {
    auth: false,
    onProgress,
    signal,
  })
}

// ── Host yo'li (dars tashqarisida — tarix va moderatsiya) ───────────────────

export function chatHistory(lessonId) {
  return api.get(`/lessons/${lessonId}/chat`)
}

// Moderatsiya — xabarni o'chirish (faqat dars egasi).
// Xabar tarixdan BUTUNLAY yo'qoladi (qabrtosh qoldirilmaydi), jonli xonadagilar
// esa data-channel'dagi `chat_deleted` orqali xabardor bo'ladi.
export function deleteChatMessage(lessonId, messageId) {
  return api.del(`/lessons/${lessonId}/chat/${messageId}`)
}

// ── Fayl cheklovlari (server bilan AYNAN bir xil) ──────────────────────────
//
// Serverda ham tekshiriladi; bu yerdagi nusxa faqat foydalanuvchini 20 MB'ni
// bekorga yuklab, so'ng 400 olishdan qutqarish uchun. Ro'yxat o'zgarsa
// backend `chat` usecase'i bilan BIRGA o'zgartiriladi.
export const MAX_FILE_BYTES = 20 * 1024 * 1024

// Chegara MATNI ham shu qiymatdan chiqadi. Avval "20 MB" uch joyda qo'lda
// yozilgan edi — chegara o'zgarsa matnlar jimgina yolg'onga aylanardi.
export const MAX_FILE_LABEL = `${MAX_FILE_BYTES / 1024 / 1024} MB`

export const ALLOWED_EXTENSIONS = [
  'jpg', 'jpeg', 'png', 'gif', 'webp',
  'pdf', 'docx', 'xlsx', 'pptx', 'doc', 'xls', 'ppt', 'txt', 'csv',
]

export const FILE_ACCEPT = ALLOWED_EXTENSIONS.map((e) => `.${e}`).join(',')

/** Fayl qabul qilinmasa — o'zbekcha sabab, aks holda `null`. */
export function fileRejectReason(file) {
  if (!file) return 'Fayl tanlanmadi'
  if (file.size > MAX_FILE_BYTES) return `Fayl juda katta — eng ko‘pi ${MAX_FILE_LABEL}`
  if (file.size === 0) return 'Fayl bo‘sh'
  const ext = file.name.split('.').pop()?.toLowerCase() || ''
  if (!ALLOWED_EXTENSIONS.includes(ext)) {
    return 'Bu turdagi fayl qabul qilinmaydi — rasm, PDF, Office yoki matn fayli yuboring'
  }
  return null
}

/**
 * Yuklash xatosini o'zbekcha matnga aylantiradi.
 *
 * Umumiy `errorText` bu yerda YARAMAYDI: u `BAD_REQUEST` ni «So'rovda xatolik»
 * deb ko'rsatadi va foydalanuvchi nima qilishni bilmay qoladi. Server esa aniq
 * sababni aytadi (hajm / tur mos emas) — uni tarjima qilib beramiz.
 */
export function uploadErrorText(err) {
  const code = err instanceof ApiError ? err.code : ''
  if (code === 'ABORTED') return 'Yuklash bekor qilindi'
  if (code === 'RATE_LIMITED') return 'Juda tez yubordingiz — bir daqiqada 5 tagacha fayl'
  if (code === 'TIMEOUT') return 'Internet sekin — fayl yuborilmadi, qayta urinib ko‘ring'
  if (code === 'NETWORK') return 'Serverga ulanib bo‘lmadi — fayl yuborilmadi'
  if (code === 'FORBIDDEN') return 'Bu darsga fayl yuborishga ruxsatingiz yo‘q'

  const msg = String(err?.message || '')
  if (/too large/i.test(msg)) return `Fayl juda katta — eng ko‘pi ${MAX_FILE_LABEL}`
  if (/does not match its extension/i.test(msg))
    return 'Fayl mazmuni kengaytmasiga mos emas — faylni tekshiring'
  if (/unsupported|not allowed|file type|extension/i.test(msg))
    return 'Bu turdagi fayl qabul qilinmaydi — rasm, PDF, Office yoki matn fayli yuboring'
  // Backend matni: `file is required (multipart field "file", max 20 MB)`.
  // U IKKI holatda chiqadi — fayl umuman yuborilmaganda VA tana hajmi
  // `MaxBytesReader` chegarasidan oshganda (o'shanda `FormFile` xato beradi).
  // Foydalanuvchi uchun ehtimoli yuqori sabab — hajm, shuning uchun ikkalasi
  // ham aytiladi.
  if (/file is required|no file|missing file/i.test(msg))
    return `Fayl yuborilmadi — uni qayta tanlang (hajmi ${MAX_FILE_LABEL} dan oshmasin)`
  return 'Faylni yuborib bo‘lmadi — qayta urinib ko‘ring'
}
