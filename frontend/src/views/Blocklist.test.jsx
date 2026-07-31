import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('../api/blocklist', () => ({
  listBlocklist: vi.fn(),
  unblock: vi.fn().mockResolvedValue(null),
}))

import { listBlocklist, unblock } from '../api/blocklist'
import { Blocklist } from './Blocklist'

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <Blocklist />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  unblock.mockResolvedValue(null)
})

describe('Blocklist (qora ro‘yxat)', () => {
  it('ro‘yxat jadvalda ko‘rsatiladi', async () => {
    listBlocklist.mockResolvedValue([
      { id: 'b1', identity: 'g1', display_name: 'Bezori Bola', created_at: '2026-07-30T10:00:00Z' },
    ])
    setup()
    expect(await screen.findByText('Bezori Bola')).toBeInTheDocument()
  })

  it('bo‘sh ro‘yxatda tushunarli bo‘sh holat', async () => {
    listBlocklist.mockResolvedValue([])
    setup()
    expect(await screen.findByText("Qora ro'yxat bo'sh")).toBeInTheDocument()
  })

  it('xatoda qayta urinish taklif qilinadi', async () => {
    listBlocklist.mockRejectedValue(new Error('down'))
    setup()
    expect(await screen.findByText(/yuklab bo'lmadi/i)).toBeInTheDocument()
  })

  it('blokdan chiqarish tasdiq bilan DELETE chaqiradi', async () => {
    listBlocklist.mockResolvedValue([
      { id: 'b1', identity: 'g1', display_name: 'Bezori Bola', created_at: '2026-07-30T10:00:00Z' },
    ])
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const user = userEvent.setup()
    setup()

    await user.click(await screen.findByTitle('Blokdan chiqarish'))
    // TanStack v5 mutationFn'ga kontekst argumentini ham uzatishi mumkin —
    // faqat birinchi argument tekshiriladi.
    await waitFor(() => expect(unblock).toHaveBeenCalledTimes(1))
    expect(unblock.mock.calls[0][0]).toBe('b1')
  })

  it('tasdiq rad etilsa DELETE chaqirilmaydi', async () => {
    listBlocklist.mockResolvedValue([
      { id: 'b1', identity: 'g1', display_name: 'Bezori Bola', created_at: '2026-07-30T10:00:00Z' },
    ])
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    const user = userEvent.setup()
    setup()

    await user.click(await screen.findByTitle('Blokdan chiqarish'))
    expect(unblock).not.toHaveBeenCalled()
  })
})
