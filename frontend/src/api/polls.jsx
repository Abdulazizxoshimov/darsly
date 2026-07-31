import { api, ApiError } from './api'

// So'rovnoma natijasi IKKI rejimda bo'ladi (kontrakt №7):
//   mentor_only (default) — natijani faqat ustoz ko'radi, e'lon qilib bo'lmaydi;
//   public                — o'quvchi ham ko'radi, LEKIN faqat ustoz «E'lon qilish»
//                           bosgach (`results_published_at`).
// Yopish ≠ e'lon qilish: `closePoll` faqat ovoz berishni to'xtatadi.
export const RESULTS_VISIBILITY = {
  MENTOR_ONLY: 'mentor_only',
  PUBLIC: 'public',
}

// Host
export function listPolls(lessonId) {
  return api.get(`/lessons/${lessonId}/polls`)
}
export function createPoll(lessonId, question, options, results_visibility = RESULTS_VISIBILITY.MENTOR_ONLY) {
  return api.post(`/lessons/${lessonId}/polls`, { question, options, results_visibility })
}
export function closePoll(pollId) {
  return api.post(`/polls/${pollId}/close`)
}
// Natijani e'lon qiladi (faqat `public` rejimda; mentor_only'da server 400 beradi).
// Idempotent: takroriy bosish e'lon vaqtini surmaydi. Server natijani xonaga
// data-channel orqali o'zi tarqatadi — klient qo'shimcha broadcast qilmaydi.
export function publishPoll(lessonId, pollId) {
  return api.post(`/lessons/${lessonId}/polls/${pollId}/publish`)
}

// Ochiq (guest — LiveKit room token bilan)
export function votePoll(pollId, token, option_index) {
  return api.post(`/polls/${pollId}/vote`, { token, option_index }, { auth: false })
}
// Natijalar endi ochiq EMAS — xonada bo'lganlik isboti (room-token) kerak.
// O'quvchida e'lon qilinmagan bo'lsa 403 qaytadi (bo'sh natija EMAS — «0 ovoz»
// bilan «ko'rsatilmaydi» ni farqlash uchun).
export function pollResults(pollId, token) {
  return api.get(`/polls/${pollId}/results?token=${encodeURIComponent(token)}`, { auth: false })
}

/**
 * Ovoz berish xatosini o'zbekcha SABABGA aylantiradi.
 *
 * Umumiy `errorText` bu yerda «So'rovda xatolik» deydi — o'quvchi esa ovozi
 * nega o'tmaganini bilmaydi va tugmani qayta-qayta bosadi. Eng ko'p uchraydigan
 * ikki holat aniq nomlanadi: so'rovnoma yopilgan va allaqachon ovoz berilgan.
 */
export function voteErrorText(err) {
  const code = err instanceof ApiError ? err.code : ''
  if (code === 'CONFLICT') return 'Siz bu so‘rovnomada allaqachon ovoz bergansiz'
  if (code === 'RATE_LIMITED') return 'Juda ko‘p urinish — biroz kuting'
  if (code === 'UNAUTHORIZED') return 'Xonaga ulanish uzildi — sahifani yangilang'
  if (/closed|not active/i.test(String(err?.message || '')))
    return 'So‘rovnoma yopilgan — ovoz qabul qilinmadi'
  if (code === 'BAD_REQUEST') return 'Ovoz qabul qilinmadi — so‘rovnoma yopilgan bo‘lishi mumkin'
  return 'Ovoz berib bo‘lmadi — qayta urinib ko‘ring'
}
