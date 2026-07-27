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
