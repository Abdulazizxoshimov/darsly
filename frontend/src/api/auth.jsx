import { api, tokenStore } from './api'

export function register(full_name, email, password) {
  return api.post('/auth/register', { full_name, email, password }, { auth: false }).then((t) => {
    tokenStore.set(t)
    return t
  })
}

export function login(email, password) {
  return api.post('/auth/login', { email, password }, { auth: false }).then((t) => {
    tokenStore.set(t)
    return t
  })
}

export function me() {
  return api.get('/auth/me')
}

export async function logout() {
  const refresh_token = tokenStore.refresh
  try {
    await api.post('/auth/logout', { refresh_token })
  } catch {
    /* lokal sessiyani baribir tozalaymiz */
  }
  tokenStore.clear()
}

export function forgotPassword(email) {
  return api.post('/auth/forgot-password', { email }, { auth: false })
}

export function resetPassword(token, new_password) {
  return api.post('/auth/reset-password', { token, new_password }, { auth: false })
}
