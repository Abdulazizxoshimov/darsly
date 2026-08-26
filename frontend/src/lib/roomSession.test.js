import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { roomSession } from './roomSession'

// F-5 — guest sessiya ko'prigi (Join → WaitingRoom → LiveRoom).
//
// sessionStorage'da yashaydi: F5 (sahifa yangilash) omon qoladi, tab yopilsa
// tozalanadi. Private rejim / bloklangan storage'da tashlanmasligi (throw)
// KRITIK — aks holda mehmon oqimi butunlay yiqiladi.

beforeEach(() => sessionStorage.clear())
afterEach(() => {
  sessionStorage.clear()
  vi.restoreAllMocks()
})

describe('roomSession — o‘qish/yozish', () => {
  // Bug: bo'sh sessiyada get() null/undefined qaytarsa, `.pending` o'qish yiqiladi.
  it('bo‘sh holatda {pending:null, room:null} qaytaradi', () => {
    expect(roomSession.get()).toEqual({ pending: null, room: null })
  })

  // Bug: setPending room'ni o'chirib yuborsa (yoki aksincha), keyingi qadam
  // token/lesson'ni yo'qotadi — mehmon "sessiya topilmadi" ga tushadi.
  it('setPending room’ni saqlab qoladi, setRoom pending’ni saqlab qoladi', () => {
    roomSession.setPending({ requestId: 'req1', lesson: { id: 'l1' } })
    roomSession.setRoom({ token: 'tok', lesson: { id: 'l1' } })
    const s = roomSession.get()
    expect(s.pending).toEqual({ requestId: 'req1', lesson: { id: 'l1' } })
    expect(s.room).toEqual({ token: 'tok', lesson: { id: 'l1' } })
  })

  it('F5 (yangi o‘qish) saqlangan qiymatni qaytaradi', () => {
    roomSession.setPending({ requestId: 'req9' })
    expect(roomSession.get().pending).toEqual({ requestId: 'req9' })
  })

  it('clear() ikkalasini ham tozalaydi', () => {
    roomSession.setPending({ requestId: 'x' })
    roomSession.setRoom({ token: 't' })
    roomSession.clear()
    expect(roomSession.get()).toEqual({ pending: null, room: null })
  })

  // Bug: buzuq JSON (masalan qisman yozilgan) parse'da throw qilsa oq ekran.
  it('buzuq JSON — bo‘sh holatga qaytadi (yiqilmaydi)', () => {
    sessionStorage.setItem('darsly.roomSession', '{buzuq')
    expect(roomSession.get()).toEqual({ pending: null, room: null })
  })
})

describe('roomSession — private rejim / bloklangan storage', () => {
  // Bug: private rejimda getItem throw qilsa (ba'zi brauzerlar), get() butun
  // ilovani yiqitadi. Try/catch bo'sh holatga tushishi kerak.
  it('getItem throw qilsa get() bo‘sh holat qaytaradi', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError')
    })
    expect(roomSession.get()).toEqual({ pending: null, room: null })
  })

  // Bug: setItem throw qilsa (kvota/private) set* jimgina o'tishi kerak —
  // ilova buzilmaydi (in-memory qolmaydi, lekin mehmon oqimi to'xtamaydi).
  it('setItem throw qilsa setPending jimgina o‘tadi (throw yo‘q)', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })
    expect(() => roomSession.setPending({ requestId: 'x' })).not.toThrow()
    expect(() => roomSession.clear()).not.toThrow()
  })
})
