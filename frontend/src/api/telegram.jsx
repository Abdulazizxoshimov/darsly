import { api } from './api'

// Mentorni Telegram bilan bog'lash (kontrakt «Telegram»).
//
// `enabled:false` — serverda integratsiya UMUMAN sozlanmagan (bot tokeni yo'q).
// Bunda UI bo'limni ko'rsatmaydi: bog'lanmaydigan tugma faqat savol tug'diradi.

/** `GET /me/telegram` javobini ko'rinish modeliga aylantiradi (adapter). */
export function adaptTelegramStatus(raw) {
  return {
    enabled: !!raw?.enabled,
    linked: !!raw?.linked,
    username: raw?.telegram_username || '',
    linked_at: raw?.linked_at || null,
    chats: (raw?.chats || []).map((c) => ({
      chat_id: c.chat_id,
      title: c.title || 'Guruh',
      type: c.type || '',
      is_active: c.is_active !== false,
    })),
  }
}

export async function getTelegramStatus() {
  return adaptTelegramStatus(await api.get('/me/telegram'))
}

/** Bir martalik kod (15 daqiqa) + `t.me` deep-link. */
export async function startTelegramLink() {
  const raw = await api.post('/me/telegram/link')
  return {
    code: raw?.code || '',
    deep_link: raw?.deep_link || '',
    expires_in_s: Number(raw?.expires_in_s) || 0,
  }
}

export function unlinkTelegram() {
  return api.del('/me/telegram')
}
