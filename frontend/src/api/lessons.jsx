import { api } from './api'

export function listLessons({ status, page, limit = 50, search } = {}) {
  const qs = new URLSearchParams()
  if (status) qs.set('status', status)
  if (page) qs.set('page', page)
  if (limit) qs.set('limit', limit)
  if (search) qs.set('search', search)
  return api.list(`/lessons?${qs.toString()}`)
}

export function getLesson(id) {
  return api.get(`/lessons/${id}`)
}

export function createLesson(body) {
  return api.post('/lessons', body)
}

export function updateLesson(id, body) {
  return api.patch(`/lessons/${id}`, body)
}

export function deleteLesson(id) {
  return api.del(`/lessons/${id}`)
}

// Host LiveKit tokeni — xonani ochadi.
export function getHostToken(id) {
  return api.post(`/lessons/${id}/token`)
}

export function endLesson(id) {
  return api.post(`/lessons/${id}/end`)
}
