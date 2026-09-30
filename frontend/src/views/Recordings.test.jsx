import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { fetchRecordingsByLesson, Recordings, RECORDINGS_CONCURRENCY } from './Recordings'

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
  return render(
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

  // Bug: darslar ro'yxati yuklanayotganda bo'sh jadval ko'rsatilsa, foydalanuvchi
  // "yozuv yo'q" deb o'ylab ketadi (aslida hali kelmagan).
  it('darslar yuklanayotganda spinner ko‘rsatiladi', async () => {
    listLessons.mockReturnValue(new Promise(() => {}))
    const { container } = setup()
    await vi.waitFor(() => expect(container.querySelector('.page-loader')).toBeTruthy())
  })

  // Bug: darslar ro'yxati so'rovi yiqilsa aniq xato + qayta urinish ko'rsatilishi
  // kerak, jimgina bo'sh ro'yxat emas.
  it('darslar so‘rovi yiqilsa xato holati ko‘rsatiladi', async () => {
    listLessons.mockRejectedValue(new Error('down'))
    setup()
    expect(await screen.findByText(/Yozuvlarni yuklab bo'lmadi/i)).toBeInTheDocument()
  })

  // Bug: bitta darsning yozuvlar so'rovi yiqilsa, butun jadval emas, FAQAT o'sha
  // dars qatori nega bo'shligini yozib qo'yishi kerak.
  it('bir dars yozuvlari so‘rovi yiqilsa o‘sha qator sababini yozadi', async () => {
    listRecordings.mockRejectedValue(new Error('down'))
    setup()
    expect(await screen.findByText(/Bu dars yozuvlarini yuklab bo'lmadi/i)).toBeInTheDocument()
    // Dars sarlavhasi baribir ko'rinadi (qator TARIX uchun qoladi).
    expect(screen.getByText('Kvadrat tenglamalar')).toBeInTheDocument()
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
    // Yangi oyna bizning sahifamizga qaytib ta'sir qila olmasin.
    expect(win.opener).toBeNull()
  })

  // Bug: serverdan kelgan havola tekshiruvsiz `location.href` ga tushardi.
  it('xavfsiz bo‘lmagan sxemali havola OCHILMAYDI (oyna yopiladi, xato)', async () => {
    const user = userEvent.setup()
    const { downloadRecording } = await import('../api/recordings')
    downloadRecording.mockResolvedValue({ url: 'javascript:alert(1)' })
    const win = { location: { href: '' }, close: vi.fn() }
    vi.spyOn(window, 'open').mockImplementation(() => win)
    listRecordings.mockResolvedValue([REC({ expires_at: new Date(Date.now() + 12 * DAY).toISOString() })])
    setup()

    await user.click(await screen.findByRole('button', { name: /Yuklab olish/ }))
    await vi.waitFor(() => expect(win.close).toHaveBeenCalled())
    expect(win.location.href).toBe('')
  })
})

// ── N+1 → 429 ────────────────────────────────────────────────────────────────
// Backend'da «hamma yozuvlar» endpoint'i yo'q: har dars alohida so'raladi.
// Avval 100 dars = 100 parallel so'rov edi, mentor limiti esa 30 so'rov/s
// (burst 60) — sahifa o'zini o'zi 429 ga urardi.
describe('Recordings — so‘rovlar cheklangan parallellik bilan', () => {
  const lessonsN = (n, status = 'ended') =>
    Array.from({ length: n }, (_, i) => ({ id: `l${i}`, title: `Dars ${i}`, is_recording_enabled: true, status }))

  it('bir vaqtda RECORDINGS_CONCURRENCY dan ko‘p so‘rov ketmaydi', async () => {
    let inflight = 0
    let peak = 0
    const fetchOne = vi.fn(async () => {
      inflight += 1
      peak = Math.max(peak, inflight)
      await new Promise((r) => setTimeout(r, 5))
      inflight -= 1
      return []
    })
    const out = await fetchRecordingsByLesson(lessonsN(12), fetchOne)
    expect(fetchOne).toHaveBeenCalledTimes(12)
    expect(peak).toBeLessThanOrEqual(RECORDINGS_CONCURRENCY)
    expect(Object.keys(out)).toHaveLength(12)
  })

  // Boshlanmagan darsning yozuvi bo'lishi mumkin emas — so'rov UMUMAN ketmaydi
  // (100 dars ro'yxatida ko'pincha yarmi rejalashtirilgan).
  it('scheduled darslar so‘ralmaydi, lekin jadvalda qoladi', async () => {
    const fetchOne = vi.fn(async () => [])
    const out = await fetchRecordingsByLesson([...lessonsN(2, 'scheduled'), ...lessonsN(1, 'live')], fetchOne)
    expect(fetchOne).toHaveBeenCalledTimes(1)
    expect(out.l0).toEqual({ recs: [] })
  })

  it('bitta dars xatosi boshqalariga ta’sir qilmaydi', async () => {
    const fetchOne = vi.fn(async (id) => {
      if (id === 'l1') throw new Error('429')
      return [REC({ id: `r-${id}`, lesson_id: id })]
    })
    const out = await fetchRecordingsByLesson(lessonsN(3), fetchOne)
    expect(out.l0.recs).toHaveLength(1)
    expect(out.l1.error).toBeInstanceOf(Error)
    expect(out.l2.recs).toHaveLength(1)
  })

  // Bug: backend bo'sh ro'yxatni `null` qaytaradi — `recs.length` yiqilardi.
  it('null javob (bo‘sh ro‘yxat) sahifani yiqitmaydi', async () => {
    listRecordings.mockResolvedValue(null)
    setup()
    expect(await screen.findByText('Bu darsda yozuv saqlanmagan.')).toBeInTheDocument()
  })
})
