import { api } from './api'

// Host boshqaruvi (mentor). identity — LiveKit ishtirokchi id.
// Eslatma: ishtirokchilar ro'yxati jonli xonada LiveKit `room` obyektidan olinadi;
// bu funksiyalar host moderatsiyasi (mute/kick/speak) uchun REST amallar.
export function muteParticipant(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/mute`)
}
export function removeParticipant(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/remove`)
}
export function muteAll(lessonId) {
  return api.post(`/lessons/${lessonId}/mute-all`)
}
export function allowSpeak(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/allow-speak`)
}
export function revokeSpeak(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/revoke-speak`)
}
