import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { Recordings } from './Recordings'

// MAHSULOT QOIDASI №5 — yozuv 30 kun saqlanadi, so'ng MinIO'dan o'chadi.
// UI shundan kelib chiqadi:
//  · `ready` yozuvda «N kundan keyin o'chadi» ko'rinadi (3 kun qolganda ogohlantirish);
//  · `expired` yozuvda yuklab olish tugmasi YO'Q — u 400 qaytaradi, ya'ni
//    tugmani ko'rsatish yolg'on va'da bo'lardi.

vi.mock('../api/recordings', () => ({
  listRecordings: vi.fn(),
  downloadRecording: vi.fn(),
  startRecording: vi.fn(),
  stopRecording: vi.fn(),
}))
vi.mock('../api/lessons', () => ({
  listLessons: vi.fn(),
  getLesson: vi.fn(),
  createLesson: vi.fn(),
  updateLesson: vi.fn(),
  deleteLesson: vi.fn(),
  getHostToken: vi.fn(),
  endLesson: vi.fn(),
}))

import { listRecordings } from '../api/recordings'
import { listLessons } from '../api/lessons'

const DAY = 86_400_000
const LESSON = { id: 'l1', title: 'Kvadrat tenglamalar', is_recording_enabled: true, status: 'ended' }

const REC = (over) => ({
  id: 'r1',
  lesson_id: 'l1',
  status: 'ready',
  duration_sec: 3600,
  size_bytes: 524_288_000,
  started_at: '2026-07-01T10:00:00Z',
  ...over,
})

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      {/* Qatorlarda arxiv sahifasiga havola bor — Router shart. */}
      <MemoryRouter>
        <Recordings />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  listLessons.mockResolvedValue({ data: [LESSON], total: 1, page: 1, limit: 100, total_pages: 1 })
})

describe('Recordings — saqlanish muddati', () => {
  it('tayyor yozuvda «N kundan keyin o‘chadi» ko‘rinadi', async () => {
    listRecordings.mockResolvedValue([REC({ expires_at: new Date(Date.now() + 12 * DAY).toISOString() })])
    setup()
    expect(await screen.findByText('12 kundan keyin o‘chadi')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Yuklab olish/ })).toBeInTheDocument()
  })

  it('muddat yaqinlashganda ogohlantirish ko‘rinishi', async () => {
    listRecordings.mockResolvedValue([REC({ expires_at: new Date(Date.now() + 2 * DAY).toISOString() })])
    setup()
    const cell = await screen.findByText('2 kundan keyin o‘chadi')
    expect(cell).toHaveClass('expiry--soon')
  })

  it('`expired` yozuvda YUKLAB OLISH tugmasi yo‘q', async () => {
    listRecordings.mockResolvedValue([REC({ status: 'expired' })])
    setup()
    expect(await screen.findByText('Muddati tugagan')).toBeInTheDocument()
    expect(screen.getByText('O‘chirilgan')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Yuklab olish/ })).not.toBeInTheDocument()
  })

  it('muddati yo‘q yozuvda (processing) ustun bo‘sh qoladi', async () => {
    listRecordings.mockResolvedValue([REC({ status: 'processing' })])
    setup()
    expect(await screen.findByText('Tayyorlanmoqda…')).toBeInTheDocument()
    expect(screen.getByText('Kuting…')).toBeInTheDocument()
  })

  it('yozuvi yo‘q dars ham NEGA bo‘shligi bilan ko‘rsatiladi', async () => {
    listRecordings.mockResolvedValue([])
    setup()
    expect(await screen.findByText('Bu darsda yozuv saqlanmagan.')).toBeInTheDocument()
  })

  it('yuklab olish yangi tabda ochiladi', async () => {
    const user = userEvent.setup()
    const { downloadRecording } = await import('../api/recordings')
    downloadRecording.mockResolvedValue({ url: 'https://minio/x.mp4', expires_in_s: 3600 })
    // Oyna bosish KONTEKSTIDA darhol ochiladi (popup bloklanmasligi uchun),
    // presigned havola kelgach unga yo'naltiriladi.
    const win = { location: { href: '' }, close: vi.fn() }
    const open = vi.spyOn(window, 'open').mockImplementation(() => win)
    listRecordings.mockResolvedValue([REC({ expires_at: new Date(Date.now() + 12 * DAY).toISOString() })])
    setup()

    await user.click(await screen.findByRole('button', { name: /Yuklab olish/ }))
    expect(downloadRecording).toHaveBeenCalledWith('r1')
    expect(open).toHaveBeenCalledWith('', '_blank')
    await vi.waitFor(() => expect(win.location.href).toBe('https://minio/x.mp4'))
  })
})
