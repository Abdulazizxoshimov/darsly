import { api } from './api'

// Ochiq join-link oqimi (autentifikatsiyasiz).
export function previewJoinLink(slug) {
  return api.get(`/joinlink/${slug}`, { auth: false })
}

export function joinLink(slug, { guest_name, passcode }) {
  return api.post(`/joinlink/${slug}`, { guest_name, passcode }, { auth: false })
}
