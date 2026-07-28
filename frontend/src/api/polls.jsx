import { api } from './api'

// Host
export function listPolls(lessonId) {
  return api.get(`/lessons/${lessonId}/polls`)
}
export function createPoll(lessonId, question, options) {
  return api.post(`/lessons/${lessonId}/polls`, { question, options })
}
export function closePoll(pollId) {
  return api.post(`/polls/${pollId}/close`)
}
// Ochiq (guest — LiveKit room token bilan)
export function votePoll(pollId, token, option_index) {
  return api.post(`/polls/${pollId}/vote`, { token, option_index }, { auth: false })
}
// Natijalar endi ochiq EMAS — xonada bo'lganlik isboti (room-token) kerak.
export function pollResults(pollId, token) {
  return api.get(`/polls/${pollId}/results?token=${encodeURIComponent(token)}`, { auth: false })
}
