import { api } from './api'

// Host boshqaruvi (mentor). identity — LiveKit ishtirokchi id.
// Eslatma: ishtirokchilar ro'yxati jonli xonada LiveKit `room` obyektidan olinadi;
// bu funksiyalar host moderatsiyasi (mute/kick/speak) uchun REST amallar.
export function muteParticipant(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/mute`)
}
// scope: 'lesson' (faqat shu dars — default) | 'mentor' (mentorning HAMMA
// darslaridan doimiy blok, qora ro'yxatga tushadi).
export function removeParticipant(lessonId, identity, scope = 'lesson') {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/remove`, { scope })
}
// allowSelfUnmute berilsa bayroq bir yo'la yangilanadi (Zoom checkbox'i);
// undefined bo'lsa body yuborilmaydi — bayroq o'zgarmaydi.
export function muteAll(lessonId, allowSelfUnmute) {
  return api.post(
    `/lessons/${lessonId}/mute-all`,
    allowSelfUnmute === undefined ? undefined : { allow_self_unmute: allowSelfUnmute },
  )
}
export function allowSpeak(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/allow-speak`)
}
export function revokeSpeak(lessonId, identity) {
  return api.post(`/lessons/${lessonId}/participants/${encodeURIComponent(identity)}/revoke-speak`)
}
