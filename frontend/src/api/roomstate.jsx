import { api } from './api'

// Xona holati — qo'l ko'tarish va reaksiyalar.
//
// Ochiq endpointlar (`auth: false`) LiveKit ROOM-TOKEN bilan autentifikatsiya
// qilinadi: guest'da JWT yo'q, lekin imzolangan room-token'i bor va backend
// uning xonasi dars bilan mosligini tekshiradi.
//
// Tarqatish yo'li: server → LiveKit data-channel → barcha klientlar. Ya'ni bu
// funksiyalar hech narsa qaytarmaydi; natija `RoomEvent.DataReceived` orqali keladi.

export function setHand(lessonId, token, raised) {
  return api.post(`/rooms/${lessonId}/hand`, { token, raised }, { auth: false })
}

export function sendReaction(lessonId, token, emoji) {
  return api.post(`/rooms/${lessonId}/reaction`, { token, emoji }, { auth: false })
}

// Kech kirgan (yoki qayta ulangan) klient holatni shu bilan tiklaydi.
export function getRoomState(lessonId, token) {
  return api.get(`/rooms/${lessonId}/state?token=${encodeURIComponent(token)}`, { auth: false })
}

// Host amallari (JWT bilan).
export function lowerHand(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/hands/lower`, { identity })
}

export function lowerAllHands(lessonId) {
  return api.post(`/lessons/${lessonId}/hands/lower-all`)
}
