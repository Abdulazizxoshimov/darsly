import { api } from './api'

// Mentorning doimiy qora ro'yxati (№4): `remove scope:"mentor"` bilan bloklangan
// ishtirokchilar. Moslik ism bo'yicha (o'quvchida akkaunt yo'q — ongli cheklov).

// [BlocklistEntry]: {id, identity, display_name, created_at}
export function listBlocklist() {
  return api.get('/blocklist')
}

// Blokdan chiqarish (unban) — 204.
export function unblock(id) {
  return api.del(`/blocklist/${id}`)
}
