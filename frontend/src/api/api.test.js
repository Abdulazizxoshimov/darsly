import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, errorText, filenameFromDisposition, setUnauthorizedHandler, tokenStore } from './api'
import { consumeLogoutReason, setLogoutReason } from '../lib/logoutReason'

// SESSIYA BEKOR QILINISHI (`SESSION_REVOKED`) — bitta akkaunt = bitta faol
// sessiya mahsulot qoidasining klient tomoni.
//
// Ikki narsa muhim:
//  1. Bekor qilingan sessiyada refresh urinishi MA'NOSIZ — u ham 401 beradi.
//     Bekorga urinish foydalanuvchini kutdiradi va serverga ortiqcha yuk.
//  2. Sabab YO'QOLMASLIGI kerak: chiqarish sahifani butunlay qayta yuklaydi,
//     shuning uchun u sessionStorage orqali login sahifasiga olib o'tiladi.

const json = (body, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

let reasons

beforeEach(() => {
  reasons = []
  localStorage.clear()
  sessionStorage.clear()
  tokenStore.set({ access_token: 'old-access', refresh_token: 'old-refresh' })
  setUnauthorizedHandler((r) => reasons.push(r))
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('401 boshqaruvi', () => {
  it('SESSION_REVOKED da refresh UMUMAN urinilmaydi', async () => {
    const fetchMock = vi.fn().mockResolvedValue(json({ code: 'SESSION_REVOKED', message: 'revoked' }, 401))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.get('/lessons')).rejects.toMatchObject({ code: 'SESSION_REVOKED', status: 401 })

    // Faqat asl so'rov — `/auth/refresh` ga chiqilmagan.
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(reasons).toEqual(['session_revoked'])
  })

  it('access muddati tugaganda refresh qilinadi va so‘rov TAKRORLANADI', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json({ code: 'TOKEN_EXPIRED', message: 'expired' }, 401))
      .mockResolvedValueOnce(json({ data: { access_token: 'new-access', refresh_token: 'new-refresh' } }))
      .mockResolvedValueOnce(json({ data: [{ id: 'l1' }] }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.get('/lessons')).resolves.toEqual([{ id: 'l1' }])
    expect(tokenStore.access).toBe('new-access')
    expect(reasons).toEqual([]) // chiqarilmadi
  })

  it('refresh ham SESSION_REVOKED bersa sabab SAQLANADI', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json({ code: 'TOKEN_EXPIRED', message: 'expired' }, 401))
      .mockResolvedValueOnce(json({ code: 'SESSION_REVOKED', message: 'revoked' }, 401))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.get('/lessons')).rejects.toMatchObject({ code: 'SESSION_REVOKED' })
    expect(reasons).toEqual(['session_revoked'])
  })

  it('refresh oddiy sabab bilan yiqilsa «expired» deyiladi', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json({ code: 'TOKEN_EXPIRED', message: 'expired' }, 401))
      .mockResolvedValueOnce(json({ code: 'TOKEN_INVALID', message: 'bad' }, 401))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.get('/lessons')).rejects.toMatchObject({ code: 'UNAUTHORIZED' })
    expect(reasons).toEqual(['expired'])
  })
})

// Sabab sahifa qayta yuklanishidan OMON chiqishi kerak — u yagona kanal
// (sessionStorage) orqali login sahifasiga o'tadi va BIR MARTA o'qiladi.
describe('logoutReason kanali', () => {
  it('saqlanadi, o‘zbekcha matnga aylanadi va bir martalik', () => {
    setLogoutReason('session_revoked')
    expect(consumeLogoutReason()).toMatch(/boshqa qurilmada kirildi/i)
    expect(consumeLogoutReason()).toBeNull()
  })

  it('noma’lum sabab saqlanmaydi (bo‘sh banner chiqmasin)', () => {
    setLogoutReason('nimadir')
    expect(consumeLogoutReason()).toBeNull()
  })
})

// ── Fayl javobi (chat transkripti) ──────────────────────────────────────────
//
// Transkript endpoint'i JWT talab qiladi, ya'ni uni oddiy `<a href>` bilan
// ochib bo'lmaydi. Fayl `api.blob` orqali — 401/refresh zanjiri bilan BIR XIL
// yo'ldan — olinadi, nomini esa SERVER aytadi.
describe('api.blob — himoyalangan fayl yuklab olish', () => {
  it('blob va Content-Disposition’dagi fayl nomini qaytaradi', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response('salom', {
          status: 200,
          headers: {
            'Content-Type': 'text/plain; charset=utf-8',
            'Content-Disposition': 'attachment; filename="algebra-chat-2026-08-01.txt"',
          },
        }),
      ),
    )
    const res = await api.blob('/lessons/l1/chat/transcript?format=txt')
    expect(res.filename).toBe('algebra-chat-2026-08-01.txt')
    expect(await res.blob.text()).toBe('salom')
  })

  // Header o'qilmasa nom BO'SH qaytadi — chaqiruvchi o'zining mazmunli
  // zaxira nomini qo'ya olsin (`'fayl'` truthy bo'lib uni bo'g'ib qo'yardi).
  it('Content-Disposition yo‘q bo‘lsa nom bo‘sh qaytadi', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('x', { status: 200 })))
    const res = await api.blob('/lessons/l1/chat/transcript')
    expect(res.filename).toBe('')
  })

  it('RFC 5987 (filename*=UTF-8) shaklini ham o‘qiydi', () => {
    expect(
      filenameFromDisposition("attachment; filename*=UTF-8''dars%2Dchat.txt"),
    ).toBe('dars-chat.txt')
  })

  it('xato javobda fayl emas, ApiError beradi', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(json({ code: 'BAD_REQUEST', message: 'format must be txt or html' }, 400)),
    )
    await expect(api.blob('/lessons/l1/chat/transcript?format=pdf')).rejects.toMatchObject({
      code: 'BAD_REQUEST',
    })
  })
})

// ── Xato javob shakllari (403/404/409/5xx) ───────────────────────────────────
//
// Har biri o'zbekcha matnga to'g'ri xaritalanishi kerak: umumiy "Xatolik" yetarli
// emas — foydalanuvchi ruxsat yo'qmi, topilmadimi yoki server yiqildimi bilishi
// shart. 409 ALOHIDA muhim: backend uni ikki marta admit (double-admit) da beradi.
describe('xato javob → ApiError + errorText mapping', () => {
  const stub = (body, status) => vi.stubGlobal('fetch', vi.fn().mockResolvedValue(json(body, status)))

  it('403 FORBIDDEN → status va «ruxsatingiz yo‘q» matni', async () => {
    stub({ code: 'FORBIDDEN', message: 'forbidden' }, 403)
    const err = await api.get('/lessons/l1').catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ code: 'FORBIDDEN', status: 403 })
    expect(errorText(err)).toMatch(/ruxsatingiz yo‘q/i)
  })

  it('404 NOT_FOUND → status va «Topilmadi» matni', async () => {
    stub({ code: 'NOT_FOUND', message: 'not found' }, 404)
    const err = await api.get('/lessons/nope').catch((e) => e)
    expect(err).toMatchObject({ code: 'NOT_FOUND', status: 404 })
    expect(errorText(err)).toMatch(/topilmadi/i)
  })

  // ⭐ Double-admit: mentor bir mehmonni ikki marta kiritsa backend 409 beradi.
  // Bug: 409 umumiy xato deb ko'rsatilsa, mentor "nega ishlamadi?" deb qayta bosadi.
  it('409 CONFLICT (double-admit) → status va «allaqachon mavjud» matni', async () => {
    stub({ code: 'CONFLICT', message: 'already admitted' }, 409)
    const err = await api.post('/waitingroom/req1/admit').catch((e) => e)
    expect(err).toMatchObject({ code: 'CONFLICT', status: 409 })
    expect(errorText(err)).toMatch(/allaqachon mavjud/i)
  })

  it('500 INTERNAL_ERROR → «Serverda xatolik» matni', async () => {
    stub({ code: 'INTERNAL_ERROR', message: 'boom' }, 500)
    const err = await api.get('/lessons').catch((e) => e)
    expect(err).toMatchObject({ code: 'INTERNAL_ERROR', status: 500 })
    expect(errorText(err)).toMatch(/serverda xatolik/i)
  })

  // Bug: 5xx bo'sh tana (gateway/proxy) parse'da yiqilsa — foydalanuvchi hech
  // qanday xato ko'rmaydi. Bo'sh tanada INTERNAL_ERROR ga tushishi kerak.
  it('502 bo‘sh tana → INTERNAL_ERROR ga tushadi (yiqilmaydi)', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 502 })))
    const err = await api.get('/lessons').catch((e) => e)
    expect(err).toMatchObject({ code: 'INTERNAL_ERROR', status: 502 })
  })

  // Bug: non-401 xatoda refresh urinilsa — bekorga so'rov va noto'g'ri logout.
  it('403/404/409/5xx da refresh UMUMAN urinilmaydi', async () => {
    const fetchMock = vi.fn().mockResolvedValue(json({ code: 'CONFLICT', message: 'x' }, 409))
    vi.stubGlobal('fetch', fetchMock)
    await api.post('/waitingroom/req1/admit').catch(() => {})
    expect(fetchMock).toHaveBeenCalledTimes(1) // /auth/refresh ga chiqilmagan
  })

  // Bug: tarmoq uzilishi (fetch TypeError) noaniq "Xatolik" deb ko'rsatilsa,
  // foydalanuvchi internetini emas, ilovani ayblaydi.
  it('tarmoq xatosi (TypeError) → «Serverga ulanib bo‘lmadi»', () => {
    const netErr = new TypeError('Failed to fetch')
    expect(errorText(netErr)).toMatch(/ulanib bo‘lmadi/i)
  })

  it('noma’lum kod → err.message yoki fallback ishlatiladi', () => {
    expect(errorText(new ApiError('WEIRD_CODE', 'maxsus xabar', 400))).toBe('maxsus xabar')
    expect(errorText({}, 'zaxira')).toBe('zaxira')
  })
})
