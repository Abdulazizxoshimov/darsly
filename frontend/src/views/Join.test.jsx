import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'

// MAHSULOT QOIDALARI (№3, №16):
//  · yakunlangan dars havolasi JOIN QILMAYDI — «Dars yakunlangan» sahifasi;
//  · dars hali boshlanmagan bo'lsa — kutish sahifasi, token so'ralmaydi,
//    dars live bo'lganda avto-kirish.

vi.mock('../api/join', () => ({
  previewJoinLink: vi.fn(),
  joinLink: vi.fn(),
}))

import { previewJoinLink, joinLink } from '../api/join'
import { Join } from './Join'

const LESSON = {
  id: 'l1',
  title: 'Kvadrat tenglamalar',
  mentor_name: 'Aziz Karimov',
  scheduled_at: '2026-08-01T10:00:00Z',
  status: 'live',
  has_passcode: false,
  is_waiting_room_enabled: false,
}

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/r/demo123']}>
        <Routes>
          <Route path="/r/:slug" element={<Join />} />
          <Route path="/r/:slug/waiting" element={<div>WAITING_PAGE</div>} />
          <Route path="/r/:slug/room" element={<div>ROOM_PAGE</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  sessionStorage.clear()
})

describe('Join — dars holatlari', () => {
  it('yakunlangan dars: forma o‘rniga «Dars yakunlangan» sahifasi, kirish YO‘Q', async () => {
    previewJoinLink.mockResolvedValue({ ...LESSON, status: 'ended' })
    setup()

    expect(await screen.findByText('Dars yakunlangan')).toBeInTheDocument()
    expect(screen.queryByPlaceholderText('Ismingizni kiriting')).not.toBeInTheDocument()
    expect(joinLink).not.toHaveBeenCalled()
  })

  it('bekor qilingan dars alohida matn bilan ko‘rsatiladi', async () => {
    previewJoinLink.mockResolvedValue({ ...LESSON, status: 'cancelled' })
    setup()
    expect(await screen.findByText('Dars bekor qilingan')).toBeInTheDocument()
  })

  it('scheduled dars: submit token SO‘RAMAYDI — kutish sahifasi ochiladi', async () => {
    previewJoinLink.mockResolvedValue({ ...LESSON, status: 'scheduled' })
    const user = userEvent.setup()
    setup()

    await user.type(await screen.findByPlaceholderText('Ismingizni kiriting'), 'Ali')
    await user.click(screen.getByRole('button', { name: /kutish/i }))

    expect(await screen.findByText('Dars boshlanishini kuting')).toBeInTheDocument()
    // Token so'ralmagan — bo'sh xonaga kirish yoki behuda waiting-request yo'q.
    expect(joinLink).not.toHaveBeenCalled()
  })

  it('kutish sahifasida dars live bo‘lgach AVTO-KIRISH (joinLink chaqiriladi)', async () => {
    // 1-javob: scheduled (forma + kutish), keyingilari: live (poll natijasi).
    previewJoinLink
      .mockResolvedValueOnce({ ...LESSON, status: 'scheduled' })
      .mockResolvedValue({ ...LESSON, status: 'live' })
    joinLink.mockResolvedValue({
      lesson: LESSON,
      next_step: 'join',
      room: { token: 't', ws_url: 'ws://x', identity: 'g1', role: 'participant' },
    })
    const user = userEvent.setup()
    setup()

    await user.type(await screen.findByPlaceholderText('Ismingizni kiriting'), 'Ali')
    await user.click(screen.getByRole('button', { name: /kutish/i }))
    expect(await screen.findByText('Dars boshlanishini kuting')).toBeInTheDocument()

    // Kutish rejimida preview polling boshlanadi (refetchInterval). Testda
    // intervalni kutmaslik uchun query'ni qo'lda yangilaymiz — mount bo'lgan
    // kutish sahifasi `status:'live'`ni ko'rishi bilan avto-kirishi kerak.
    // (refetchInterval real vaqtda 6 s — bu yerda birinchi refetch'ni
    // react-query o'zi window focus'siz ham interval bilan qiladi.)
    await waitFor(() => expect(joinLink).toHaveBeenCalledWith('demo123', { guest_name: 'Ali', passcode: undefined }), {
      timeout: 8000,
    })
    expect(await screen.findByText('ROOM_PAGE', {}, { timeout: 3000 })).toBeInTheDocument()
  }, 15000)

  it('POST lesson_ended qaytarsa (preview eskirgan) — «Dars yakunlangan»', async () => {
    previewJoinLink.mockResolvedValue({ ...LESSON, status: 'live' })
    joinLink.mockResolvedValue({ lesson: { ...LESSON, status: 'ended' }, next_step: 'lesson_ended' })
    const user = userEvent.setup()
    setup()

    await user.type(await screen.findByPlaceholderText('Ismingizni kiriting'), 'Ali')
    await user.click(screen.getByRole('button', { name: /qo'shilish/i }))

    expect(await screen.findByText('Dars yakunlangan')).toBeInTheDocument()
  })
})
