import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../api/api'
import {
  RENEW_LEAD_MS,
  RoomTokenError,
  createRoomTokenSource,
  fromJoinError,
  fromJoinResponse,
  guestMinter,
  isFatalRoomTokenError,
  isRoomAuthError,
  renewDelay,
  roomErrorText,
} from './roomToken'
import { jwtExpired, jwtExpiry } from './jwt'

// Mehmon tokeni 30 daqiqalik va backend uni uzaytirmaydi. Bu yerdagi qoidalar
// tokenni MUDDATIDAN OLDIN va 401 da BIR MARTA yangilashni ta'minlaydi —
// aks holda 30 daqiqadan keyin chat/qo'l/ovoz jimgina o'lardi.

vi.mock('../api/join', () => ({ joinLink: vi.fn() }))
vi.mock('../api/lessons', () => ({ getHostToken: vi.fn() }))

import { joinLink } from '../api/join'

// Imzosiz JWT — faqat `exp` o'qiladi (imzo tekshirilmaydi).
const jwt = (payload) => {
  const b64 = btoa(JSON.stringify(payload)).replace(/=+$/, '').replace(/\+/g, '-').replace(/\//g, '_')
  return `eyJhbGciOiJIUzI1NiJ9.${b64}.sig`
}

const NOW = Date.parse('2026-08-01T10:00:00Z')

beforeEach(() => vi.clearAllMocks())

describe('jwtExpiry / renewDelay', () => {
  it('JWT `exp` ni ms da o‘qiydi (base64url)', () => {
    expect(jwtExpiry(jwt({ exp: 1_800_000_000, sub: 'g/1+?' }))).toBe(1_800_000_000_000)
  })

  // Bug: buzuq token throw qilsa xona ekrani oq bo'lardi.
  it('buzuq/exp‘siz token → null (throw yo‘q)', () => {
    expect(jwtExpiry('buzuq')).toBeNull()
    expect(jwtExpiry(jwt({ sub: 'x' }))).toBeNull()
    expect(jwtExpiry(undefined)).toBeNull()
  })

  it('jwtExpired: o‘tgan → true, kelajak → false, exp yo‘q → false (serverga qoldiramiz)', () => {
    expect(jwtExpired(jwt({ exp: (NOW - 1) / 1000 }), NOW)).toBe(true)
    expect(jwtExpired(jwt({ exp: (NOW + 60_000) / 1000 }), NOW)).toBe(false)
    expect(jwtExpired('x.y.z', NOW)).toBe(false)
  })

  it('yangilash muddat tugashidan LEAD oldin rejalashtiriladi', () => {
    const exp = (NOW + 30 * 60_000) / 1000
    expect(renewDelay(jwt({ exp }), NOW)).toBe(30 * 60_000 - RENEW_LEAD_MS)
  })

  // Bug: muddati o'tib ketgan token uchun manfiy kutish setTimeout'da 0 ga
  // tushadi — bu to'g'ri, lekin aniq kafolatlangan bo'lsin.
  it('muddati o‘tgan token → 0 (darhol)', () => {
    expect(renewDelay(jwt({ exp: (NOW - 1000) / 1000 }), NOW)).toBe(0)
  })

  it('exp o‘qilmasa null — rejalashtirilmaydi', () => {
    expect(renewDelay('x.y.z', NOW)).toBeNull()
  })
})

describe('fromJoinResponse / fromJoinError', () => {
  it('next_step=join → token', () => {
    const room = { token: 't', ws_url: 'ws://x', identity: 'g2' }
    expect(fromJoinResponse({ next_step: 'join', room })).toBe(room)
  })

  // Backend cheklovi: kutish xonasida qayta join YANGI so'rov yaratadi —
  // klient buni bilishi va mehmonni kutish sahifasiga qaytarishi kerak.
  it('waiting_room → RoomTokenError(waiting_room) requestId bilan', () => {
    let err
    try {
      fromJoinResponse({ next_step: 'waiting_room', request_id: 'req7', lesson: { id: 'l1' } })
    } catch (e) {
      err = e
    }
    expect(err).toBeInstanceOf(RoomTokenError)
    expect(err.kind).toBe('waiting_room')
    expect(err.requestId).toBe('req7')
  })

  it('lesson_ended / waiting_for_host → yakuniy turlar', () => {
    expect(() => fromJoinResponse({ next_step: 'lesson_ended' })).toThrow(expect.objectContaining({ kind: 'ended' }))
    expect(() => fromJoinResponse({ next_step: 'waiting_for_host' })).toThrow(
      expect.objectContaining({ kind: 'not_live' }),
    )
  })

  it('403 sabablari ajratiladi: qulf / chiqarilgan / boshqa', () => {
    expect(fromJoinError(new ApiError('FORBIDDEN', 'lesson is locked by the host', 403)).kind).toBe('locked')
    expect(fromJoinError(new ApiError('FORBIDDEN', "you have been removed from this mentor's lessons", 403)).kind).toBe(
      'removed',
    )
    expect(fromJoinError(new ApiError('FORBIDDEN', 'too many failed attempts', 403)).kind).toBe('forbidden')
  })

  // Bug: 5xx/429/tarmoq YAKUNIY deb qaralsa, bir lahzalik uzilish mehmonni
  // darsdan butunlay chiqarib yuborardi.
  it('5xx / RATE_LIMITED / tarmoq → network (vaqtinchalik)', () => {
    expect(fromJoinError(new ApiError('INTERNAL_ERROR', 'x', 502)).kind).toBe('network')
    expect(fromJoinError(new ApiError('RATE_LIMITED', 'x', 429)).kind).toBe('network')
    expect(fromJoinError(new ApiError('TIMEOUT', 'x', 0)).kind).toBe('network')
    expect(fromJoinError(new TypeError('Failed to fetch')).kind).toBe('network')
    expect(isFatalRoomTokenError(fromJoinError(new TypeError('x')))).toBe(false)
    expect(isFatalRoomTokenError(new RoomTokenError('ended', 'x'))).toBe(true)
  })

  it('404 → ended (dars o‘chirilgan)', () => {
    expect(fromJoinError(new ApiError('NOT_FOUND', 'lesson', 404)).kind).toBe('ended')
  })
})

describe('roomErrorText', () => {
  // Bug: xona ichidagi 401 «Email yoki parol noto'g'ri» deb chiqardi —
  // mehmon parol kiritmagan ham.
  it('xona 401 i parol haqida EMAS, sessiya haqida gapiradi', () => {
    expect(roomErrorText(new ApiError('UNAUTHORIZED', 'invalid room token', 401))).toMatch(/sessiyasi tugadi/i)
    expect(isRoomAuthError(new ApiError('UNAUTHORIZED', 'x', 401))).toBe(true)
    expect(isRoomAuthError(new ApiError('FORBIDDEN', 'x', 403))).toBe(false)
  })

  it('RoomTokenError turi bo‘yicha matn', () => {
    expect(roomErrorText(new RoomTokenError('removed', 'x'))).toMatch(/chiqarishgan/i)
    expect(roomErrorText(new RoomTokenError('waiting_room', 'x'))).toMatch(/tasdig‘i/i)
  })

  it('boshqa xatolar umumiy matnga tushadi', () => {
    expect(roomErrorText(new ApiError('RATE_LIMITED', 'x', 429))).toMatch(/ko‘p urinish/i)
    expect(roomErrorText({}, 'zaxira')).toBe('zaxira')
  })
})

describe('createRoomTokenSource', () => {
  const T1 = { token: 't1', ws_url: 'ws://x', identity: 'g1' }
  const T2 = { token: 't2', ws_url: 'ws://x', identity: 'g2' }

  it('run: 401 da tokenni yangilab BIR MARTA takrorlaydi', async () => {
    const mint = vi.fn().mockResolvedValue(T2)
    const persist = vi.fn()
    const src = createRoomTokenSource({ initial: T1, mint, persist })
    const call = vi
      .fn()
      .mockRejectedValueOnce(new ApiError('UNAUTHORIZED', 'invalid room token', 401))
      .mockResolvedValueOnce('ok')

    await expect(src.run((t) => call(t.token))).resolves.toBe('ok')
    expect(call.mock.calls).toEqual([['t1'], ['t2']])
    expect(mint).toHaveBeenCalledTimes(1)
    expect(persist).toHaveBeenCalledWith(T2)
    expect(src.current()).toBe(T2)
  })

  // Bug: ikkinchi 401 da yana yangilansa — cheksiz join sikli (har biri yangi
  // identity bilan).
  it('run: yangilangan token ham 401 bersa xato YUQORIGA chiqadi (sikl yo‘q)', async () => {
    const mint = vi.fn().mockResolvedValue(T2)
    const src = createRoomTokenSource({ initial: T1, mint })
    const err = new ApiError('UNAUTHORIZED', 'x', 401)
    await expect(src.run(() => Promise.reject(err))).rejects.toBe(err)
    expect(mint).toHaveBeenCalledTimes(1)
  })

  it('run: 401 bo‘lmagan xato yangilashsiz o‘tkaziladi', async () => {
    const mint = vi.fn()
    const src = createRoomTokenSource({ initial: T1, mint })
    await expect(src.run(() => Promise.reject(new ApiError('RATE_LIMITED', 'x', 429)))).rejects.toMatchObject({
      status: 429,
    })
    expect(mint).not.toHaveBeenCalled()
  })

  // Bug: chat, qo'l va holat so'rovlari bir vaqtda 401 olsa uchta join
  // ketardi — uchta identity.
  it('renew: bir vaqtdagi chaqiruvlar BITTA so‘rovga birlashadi', async () => {
    let resolve
    const mint = vi.fn(() => new Promise((r) => (resolve = r)))
    const src = createRoomTokenSource({ initial: T1, mint })
    const listener = vi.fn()
    src.subscribe(listener)

    const a = src.renew()
    const b = src.renew()
    expect(mint).toHaveBeenCalledTimes(1)
    resolve(T2)
    await expect(Promise.all([a, b])).resolves.toEqual([T2, T2])
    expect(listener).toHaveBeenCalledTimes(1)

    // Tugagach keyingi renew yangi so'rov ochadi.
    src.renew()
    expect(mint).toHaveBeenCalledTimes(2)
  })

  it('renew yiqilsa joriy token o‘zgarmaydi va xato uzatiladi', async () => {
    const mint = vi.fn().mockRejectedValue(new RoomTokenError('ended', 'x'))
    const src = createRoomTokenSource({ initial: T1, mint })
    await expect(src.renew()).rejects.toMatchObject({ kind: 'ended' })
    expect(src.current()).toBe(T1)
  })
})

describe('guestMinter', () => {
  it('saqlangan kirish ma’lumotlari bilan qayta join qiladi', async () => {
    joinLink.mockResolvedValue({ next_step: 'join', room: { token: 't2', ws_url: 'ws://x', identity: 'g2' } })
    const mint = guestMinter({ slug: 'demo', guestName: 'Ali', passcode: '1234' })
    await expect(mint()).resolves.toMatchObject({ token: 't2' })
    expect(joinLink).toHaveBeenCalledWith('demo', { guest_name: 'Ali', passcode: '1234' })
  })

  it('parolsiz darsda passcode yuborilmaydi', async () => {
    joinLink.mockResolvedValue({ next_step: 'join', room: { token: 't2', ws_url: 'ws://x' } })
    await guestMinter({ slug: 'demo', guestName: 'Ali', passcode: '' })()
    expect(joinLink).toHaveBeenCalledWith('demo', { guest_name: 'Ali', passcode: undefined })
  })

  // Eski sessiya (kirish ma'lumotlari saqlanmagan) — yangilash imkonsiz, lekin
  // bu ANIQ yakuniy xato, cheksiz urinish emas.
  it('kirish ma’lumotlari yo‘q → forbidden (so‘rov yuborilmaydi)', async () => {
    await expect(guestMinter(undefined)()).rejects.toMatchObject({ kind: 'forbidden' })
    expect(joinLink).not.toHaveBeenCalled()
  })

  it('API xatosi RoomTokenError ga aylanadi', async () => {
    joinLink.mockRejectedValue(new ApiError('FORBIDDEN', 'lesson is locked by the host', 403))
    await expect(guestMinter({ slug: 'demo', guestName: 'Ali' })()).rejects.toMatchObject({ kind: 'locked' })
  })
})
