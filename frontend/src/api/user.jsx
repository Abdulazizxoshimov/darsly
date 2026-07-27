import { api } from './api'

export function updateProfile(body) {
  return api.put('/users/me', body)
}
export function changePassword(current_password, new_password) {
  return api.put('/users/me/password', { current_password, new_password })
}
