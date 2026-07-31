import { api } from './api'

// Guest polling (ochiq) — WS real-time fallback.
export function getWaitingStatus(requestId) {
  return api.get(`/waitingroom/${requestId}/status`, { auth: false })
}

// Mentor tomoni
export function listWaiting(lessonId) {
  return api.get(`/lessons/${lessonId}/waitingroom`)
}

export function admitWaiting(requestId) {
  return api.post(`/waitingroom/${requestId}/admit`)
}

export function rejectWaiting(requestId) {
  return api.post(`/waitingroom/${requestId}/reject`)
}

// «Hammasini kiritish» — butun navbat BITTA so'rovda. 100–300 kishilik darsda
// har so'rovni alohida tugma bilan kiritish real emas edi (sekin internetda
// ustoz ro'yxatning yarmi kirib yarmi kirmagan holatni ko'rardi).
//
// Tanasi YO'Q. Javob HAR DOIM 200 — `{total,admitted,failed}`; qisman
// muvaffaqiyat (`failed>0`) NORMAL holat, xato emas: ro'yxat o'qilgandan keyin
// alohida admit/reject qilingan so'rov o'tkazib yuboriladi.
export function admitAllWaiting(lessonId) {
  return api.post(`/lessons/${lessonId}/waitingroom/admit-all`)
}
