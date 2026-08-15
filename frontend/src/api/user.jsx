import { api } from './api'

export function updateProfile(body) {
  return api.put('/users/me', body)
}
export function changePassword(current_password, new_password) {
  return api.put('/users/me/password', { current_password, new_password })
}

// Hisobni o'chirishni SO'RASH — mentor o'zi o'chirmaydi, admin tasdiqlab o'chiradi.
// Faqat web'da (mobil ilovada bu imkoniyat yo'q).
export function requestAccountDeletion() {
  return api.post('/users/me/request-deletion')
}

/* -------- Admin (RBAC: faqat admin) -------- */

// Ro'yxat `UserShort` qaytaradi: {id, full_name, email, avatar_url?, color}.
export function listUsers({ page, limit = 100, role } = {}) {
  const qs = new URLSearchParams()
  if (page) qs.set('page', page)
  if (limit) qs.set('limit', limit)
  if (role) qs.set('role', role)
  return api.list(`/users?${qs.toString()}`)
}

// body: {full_name, email, password, role?} — backend `CreateUserReq`.
export function createUser(body) {
  return api.post('/users', body)
}

// body: {full_name?, role?, ...} — backend `UpdateUserReq`.
export function updateUser(id, body) {
  return api.put(`/users/${id}`, body)
}

export function deactivateUser(id) {
  return api.post(`/users/${id}/deactivate`)
}
export function activateUser(id) {
  return api.post(`/users/${id}/activate`)
}

export function deleteUser(id) {
  return api.del(`/users/${id}`)
}
