import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppProvider, bootstrapRetryDelay, useApp } from './app'
import { ApiError, tokenStore } from '../api/api'

// Bootstrap (`me()`) tarmoq sababli yiqilsa: token QOLADI va so'rov o'zi qayta
// uriniladi. Avval qayta urinish yo'q edi — `authed=true, user=null` holat
// sahifa qo'lda yangilanmaguncha qolib ketardi.

vi.mock('../api/auth', () => ({
  me: vi.fn(),
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
}))
vi.mock('../lib/ws', () => ({ connectRealtime: () => () => {} }))
vi.mock('../lib/toast', () => ({ toast: { error: vi.fn(), info: vi.fn(), success: vi.fn() } }))

import { me } from '../api/auth'
import { toast } from '../lib/toast'

function Probe() {
  const { user, ready, authed } = useApp()
  return (
    <div>
      <span data-testid="ready">{String(ready)}</span>
      <span data-testid="authed">{String(authed)}</span>
      <span data-testid="user">{user ? user.full_name : '-'}</span>
    </div>
  )
}

function setup() {
  const qc = new QueryClient()
  render(
    <QueryClientProvider client={qc}>
      <AppProvider>
        <Probe />
      </AppProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  localStorage.clear()
  tokenStore.set({ access_token: 'a', refresh_token: 'r' })
})
afterEach(() => {
  vi.useRealTimers()
  localStorage.clear()
})

describe('bootstrapRetryDelay', () => {
  it('3s → 6s → 12s → 24s → 30s (cap)', () => {
    expect([0, 1, 2, 3, 4, 9].map(bootstrapRetryDelay)).toEqual([3000, 6000, 12000, 24000, 30000, 30000])
  })
})

describe('AppProvider — bootstrap', () => {
  it('tarmoq xatosidan keyin token qoladi, ilova tayyor va me() QAYTA so‘raladi', async () => {
    me.mockRejectedValueOnce(new TypeError('Failed to fetch')).mockResolvedValueOnce({ id: 'u1', full_name: 'Aziz' })
    setup()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(screen.getByTestId('ready').textContent).toBe('true')
    expect(screen.getByTestId('authed').textContent).toBe('true')
    expect(screen.getByTestId('user').textContent).toBe('-')
    expect(toast.error).toHaveBeenCalledTimes(1)
    expect(tokenStore.access).toBe('a')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000)
    })
    expect(me).toHaveBeenCalledTimes(2)
    expect(screen.getByTestId('user').textContent).toBe('Aziz')
  })

  it('`online` hodisasi kutishni tashlab darhol qayta so‘raydi', async () => {
    me.mockRejectedValueOnce(new TypeError('Failed to fetch')).mockResolvedValueOnce({ id: 'u1', full_name: 'Aziz' })
    setup()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })
    await act(async () => {
      window.dispatchEvent(new Event('online'))
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(me).toHaveBeenCalledTimes(2)
    expect(screen.getByTestId('user').textContent).toBe('Aziz')
  })

  // 401 — sessiya rostan o'lgan (refresh allaqachon urinilgan): token tozalanadi,
  // qayta urinish YO'Q.
  it('401 da token tozalanadi va qayta urinilmaydi', async () => {
    me.mockRejectedValue(new ApiError('UNAUTHORIZED', 'x', 401))
    setup()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60_000)
    })
    expect(me).toHaveBeenCalledTimes(1)
    expect(tokenStore.access).toBeNull()
    expect(screen.getByTestId('authed').textContent).toBe('false')
  })

  it('token yo‘q bo‘lsa me() umuman so‘ralmaydi', async () => {
    tokenStore.clear()
    setup()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(me).not.toHaveBeenCalled()
    expect(screen.getByTestId('ready').textContent).toBe('true')
  })
})
