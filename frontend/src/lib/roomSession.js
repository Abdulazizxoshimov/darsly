// Guest oqimi uchun sessiya: Join → WaitingRoom → LiveRoom orasida LiveKit token va
// dars ma'lumotini uzatadi. sessionStorage'da saqlanadi — sahifa yangilansa (F5) omon
// qoladi, lekin tab yopilganda tozalanadi (guest sessiyasi shu tab bilan cheklangan).
//
// `room.join` — qayta kirish ma'lumotlari (slug, ism, parol). Token 30 daqiqalik
// va uni yangilash uchun `POST /joinlink` qayta chaqiriladi (`lib/roomToken.js`);
// parol shu tab doirasida, token bilan bir xil sezgirlik darajasida saqlanadi.
const KEY = 'jonly.roomSession'
const EMPTY = { pending: null, room: null }

function read() {
  try {
    const raw = sessionStorage.getItem(KEY)
    return raw ? JSON.parse(raw) : { ...EMPTY }
  } catch {
    // sessionStorage yo'q (private rejim) yoki buzuq JSON — bo'sh holat
    return { ...EMPTY }
  }
}

function write(state) {
  try {
    sessionStorage.setItem(KEY, JSON.stringify(state))
  } catch {
    // yozib bo'lmasa jimgina o'tamiz (in-memory qolmaydi, lekin ilova buzilmaydi)
  }
}

/**
 * @typedef {object} GuestRoom
 * @property {import('./roomToken').RoomToken} token
 * @property {{ id?: string, title: string, is_waiting_room_enabled?: boolean, allow_self_unmute?: boolean }} lesson
 * @property {string} [guestName]
 * @property {import('./roomToken').GuestJoin} [join]
 */

/**
 * Saqlangan xona yozuvi SHAKLI to'g'ri bo'lsagina qaytadi, aks holda `null`.
 *
 * Eski versiya yoki qo'lda buzilgan sessionStorage `room.lesson.title` ni
 * o'qishda throw qilib butun sahifani oq qilardi. Bu yerda tekshiriladigan
 * maydonlar — xona ekrani ishlashi uchun SHART bo'lganlari.
 */
export function readGuestRoom() {
  const room = read().room
  if (!room || typeof room !== 'object') return null
  const { token, lesson } = room
  if (!token || typeof token.token !== 'string' || typeof token.ws_url !== 'string') return null
  if (!lesson || typeof lesson !== 'object' || typeof lesson.title !== 'string') return null
  return room
}

export const roomSession = {
  get: () => read(),
  setPending: (pending) => write({ ...read(), pending }),
  setRoom: (room) => write({ ...read(), room }),
  // Yangilangan token — qolgan maydonlar (dars, ism, kirish ma'lumotlari) saqlanadi.
  setRoomToken: (token) => {
    const s = read()
    if (s.room) write({ ...s, room: { ...s.room, token } })
  },
  clear: () => write({ ...EMPTY }),
}
