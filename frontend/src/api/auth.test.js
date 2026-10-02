import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { logout } from './auth'
import { tokenStore } from './api'

// Logout kontrakti: server joriy sessiyani access token bo'yicha HAR DOIM
// bekor qiladi; `refresh_token` ixtiyoriy. Refresh yo'q bo'lsa tana yuborilmaydi.

beforeEach(() => {
  localStorage.clear()
})
afterEach(() => vi.unstubAllGlobals())

describe('logout', () => {
  it('refresh bor — tanada yuboriladi, so‘ng lokal sessiya tozalanadi', async () => {
    tokenStore.set({ access_token: 'a', refresh_token: 'r' })
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await logout()
    const [, init] = fetchMock.mock.calls[0]
    expect(init.method).toBe('POST')
    expect(init.headers.Authorization).toBe('Bearer a')
    expect(JSON.parse(init.body)).toEqual({ refresh_token: 'r' })
    expect(tokenStore.access).toBeNull()
  })

  // Bug: refresh yo'q bo'lganda `{refresh_token:null}` ketardi — endi tana
  // umuman yo'q, server access token'dagi sessiyani o'zi o'chiradi.
  it('refresh yo‘q — bo‘sh tana, faqat access token bilan', async () => {
    localStorage.setItem('jonly.access', 'a')
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await logout()
    const [, init] = fetchMock.mock.calls[0]
    expect(init.body).toBeUndefined()
    expect(init.headers['Content-Type']).toBeUndefined()
    expect(init.headers.Authorization).toBe('Bearer a')
    expect(tokenStore.access).toBeNull()
  })

  it('server xatosida ham lokal sessiya tozalanadi', async () => {
    tokenStore.set({ access_token: 'a', refresh_token: 'r' })
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    await expect(logout()).resolves.toBeUndefined()
    expect(tokenStore.access).toBeNull()
  })
})
