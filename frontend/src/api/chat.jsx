import { api } from './api'

// Host: tarix + xabar yuborish (backend persist + LiveKit broadcast).
// Guest chat data-channel orqali (lib/liveroom-data.js), auth talab qilmaydi.
export function chatHistory(lessonId) {
  return api.get(`/lessons/${lessonId}/chat`)
}
export function sendChat(lessonId, body) {
  return api.post(`/lessons/${lessonId}/chat`, { body })
}
