import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, filenameFromDisposition, setUnauthorizedHandler, tokenStore } from './api'
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
