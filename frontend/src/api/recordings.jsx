import { api } from './api'

export function startRecording(lessonId) {
  return api.post(`/lessons/${lessonId}/recording/start`)
}
export function stopRecording(recordingId) {
  return api.post(`/recordings/${recordingId}/stop`)
}
// Backend bo'sh ro'yxatni `null` qaytaradi — chaqiruvchi har doim massiv oladi.
export function listRecordings(lessonId) {
  return api.get(`/lessons/${lessonId}/recordings`).then((items) => (Array.isArray(items) ? items : []))
}
export function downloadRecording(recordingId) {
  return api.get(`/recordings/${recordingId}/download`)
}

// Bitta yozuv holati — `processing`/`restoring` ni poll qilish uchun.
// Ro'yxatni (`/lessons/:id/recordings`) har 5 soniyada so'rash isrof bo'lardi.
export function getRecording(recordingId) {
  return api.get(`/recordings/${recordingId}`)
}

// Telegram arxividan qaytarib olish. 202 bilan DARHOL qaytadi — yuklab olish
// 30-60 soniya davom etadi va uni HTTP so'rovida ushlab turib bo'lmaydi.
// Idempotent: ikki marta bosilsa ham bitta yuklash ketadi; fayl allaqachon
// serverda bo'lsa `{status:"ready"}` qaytadi (bu XATO emas).
export async function restoreRecording(recordingId) {
  const r = await api.post(`/recordings/${recordingId}/restore`)
  return {
    status: r?.status || 'restoring',
    // Server qancha kutishni O'ZI aytadi (thundering-herd bo'lmasin).
    poll_after_s: Number(r?.poll_after_s) || 5,
  }
}
