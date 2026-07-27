// Guest oqimi uchun sessiya: Join → WaitingRoom → LiveRoom orasida LiveKit token va
// dars ma'lumotini uzatadi. sessionStorage'da saqlanadi — sahifa yangilansa (F5) omon
// qoladi, lekin tab yopilganda tozalanadi (guest sessiyasi shu tab bilan cheklangan).
const KEY = 'darsly.roomSession'
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

export const roomSession = {
  get: () => read(),
  setPending: (pending) => write({ ...read(), pending }),
  setRoom: (room) => write({ ...read(), room }),
  clear: () => write({ ...EMPTY }),
}
