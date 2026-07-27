import { api } from './api'

export function listNotifications(unread = false) {
  return api.list(`/notifications?unread=${unread}&limit=50`)
}
export function unreadCount() {
  return api.get('/notifications/unread-count').then((d) => d.count)
}
export function markRead(id) {
  return api.post(`/notifications/${id}/read`)
}
export function markAllRead() {
  return api.post('/notifications/read-all')
}
