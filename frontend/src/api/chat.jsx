import { api } from './api'

// Chat ikki yo'l bilan ishlaydi va bu ataylab:
//
//  · XONA yo'li (`/rooms/...`) — LiveKit room-token bilan. Host ham, o'quvchi ham
//    dars ichida shundan foydalanadi: bitta kod yo'li, bitta xatti-harakat.
//  · HOST yo'li (`/lessons/...`) — JWT bilan. Dars TUGAGANDAN keyin ham tarixni
//    o'qish uchun kerak (room-token faqat dars davomida yaroqli).
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

// ── Host yo'li (dars tashqarisida — tarix) ──────────────────────────────────

export function chatHistory(lessonId) {
  return api.get(`/lessons/${lessonId}/chat`)
}
export function sendChat(lessonId, body, to) {
  return api.post(`/lessons/${lessonId}/chat`, { body, to: to || '' })
}
