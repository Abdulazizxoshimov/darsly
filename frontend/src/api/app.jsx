import { api } from './api'

// Ochiq ilova konfiguratsiyasi (auth'siz).
// `allow_open_registration` — login sahifasi "Ro'yxatdan o'tish" bo'limini
// ko'rsatish-ko'rsatmaslikni shu bayroqdan biladi (server baribir o'zi bloklaydi).
export function getAppConfig() {
  return api.get('/app-config', { auth: false })
}
