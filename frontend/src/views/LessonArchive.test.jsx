import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { LessonArchive } from './LessonArchive'

// Arxiv sahifasining uchta mahsulot qoidasi shu yerda qulflanadi:
//  1. Chatdagi vaqtni bosganda video O'SHA daqiqaga sakraydi (sahifaning butun ma'nosi);
//  2. `archived` → tiklash → poll → `ready` oqimi oxirigacha ishlaydi va
//     pleyer O'ZI paydo bo'ladi (foydalanuvchi hech nima bosmaydi);
//  3. Chat va materiallar video holatidan MUSTAQIL — video yo'q bo'lsa ham ko'rinadi.

vi.mock('../api/archive', () => ({
  getLessonArchive: vi.fn(),
  getChatTranscript: vi.fn(),
  adaptArchive: (x) => x,
}))
vi.mock('../api/recordings', () => ({
  getRecording: vi.fn(),
  restoreRecording: vi.fn(),
  listRecordings: vi.fn(),
  downloadRecording: vi.fn(),
  startRecording: vi.fn(),
  stopRecording: vi.fn(),
}))
vi.mock('../api/chat', async (orig) => ({
  ...(await orig()),
  deleteChatMessage: vi.fn().mockResolvedValue(null),
}))

import { getLessonArchive, getChatTranscript } from '../api/archive'
import { getRecording, restoreRecording } from '../api/recordings'

const LESSON = {
  id: 'l2',
  title: 'Geometriya — uchburchaklar',
  duration_min: 60,
  started_at: '2026-07-01T09:00:00Z',
  status: 'ended',
}

const CHAT = [
  {
    id: 'ac1', sender_identity: 'host', sender_name: 'Aziz Karimov',
    body: 'Assalomu alaykum', is_private: false, to_identity: null, file: null,
    created_at: '2026-07-01T09:00:12Z', offset_sec: 12,
  },
  {
    id: 'ac2', sender_identity: 'guest_1', sender_name: 'Ali Valiyev',
    body: 'Ustoz, savol bor', is_private: false, to_identity: null, file: null,
    created_at: '2026-07-01T09:02:05Z', offset_sec: 125,
  },
  {
    id: 'ac3', sender_identity: 'host', sender_name: 'Aziz Karimov',
    body: 'Marhamat', is_private: true, to_identity: 'guest_1', file: null,
    created_at: '2026-07-01T09:02:20Z', offset_sec: 140,
  },
]

const MATERIALS = [
  { name: 'uchburchaklar.pdf', size: 184_320, mime: 'application/pdf', url: 'blob:x/uchburchaklar.pdf', created_at: '2026-07-01T09:40:00Z' },
]

const archive = (recording) => ({ lesson: LESSON, recording, chat: CHAT, materials: MATERIALS })

const READY = {
  id: 'r3', status: 'ready', duration_sec: 3600, size_bytes: 196_608_000,
  url: 'blob:x/archive.mp4', expires_at: null,
}
const ARCHIVED = {
  id: 'r3', status: 'archived', duration_sec: 3600, size_bytes: 196_608_000,
  url: null, expires_at: null,
}

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/app/lesson/l2/archive']}>
        <Routes>
          <Route path="/app/lesson/:id/archive" element={<LessonArchive />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return qc
}

beforeEach(() => {
  vi.clearAllMocks()
  // happy-dom `play()` ni amalga oshirmagan — pleyerni ijro etib bo'lmaydi,
  // lekin `currentTime` oddiy xossa sifatida ishlaydi (sakrash aynan shu).
  HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined)
})

describe('Dars arxivi — vaqt bo‘yicha sakrash', () => {
  it('chat vaqtini bosganda video shu soniyaga o‘tadi', async () => {
    const user = userEvent.setup()
    getLessonArchive.mockResolvedValue(archive(READY))
    setup()

    const video = await screen.findByTestId('archive-video')
    expect(video.currentTime).toBe(0)

    // «2:05» = 125 soniya — Ali Valiyevning savoli.
    await user.click(screen.getByRole('button', { name: /^2:05/ }))
    expect(video.currentTime).toBe(125)

    await user.click(screen.getByRole('button', { name: /^0:12/ }))
    expect(video.currentTime).toBe(12)
  })

  it('joriy vaqtga mos xabar ajratib ko‘rsatiladi', async () => {
    getLessonArchive.mockResolvedValue(archive(READY))
    setup()

    const video = await screen.findByTestId('archive-video')
    // 130-soniyada faol xabar — 125-soniyadagi («Ustoz, savol bor»), 140 emas.
    video.currentTime = 130
    video.dispatchEvent(new Event('timeupdate'))

    const msgs = await screen.findAllByTestId('archive-msg')
    const active = msgs.filter((m) => m.classList.contains('is-active'))
    expect(active).toHaveLength(1)
    expect(within(active[0]).getByText('Ustoz, savol bor')).toBeInTheDocument()
  })

  it('video yo‘q bo‘lsa vaqt tugmasi o‘chirilgan (yolg‘on va’da bo‘lmasin)', async () => {
    getLessonArchive.mockResolvedValue(archive(ARCHIVED))
    setup()
    expect(await screen.findByRole('button', { name: /^2:05/ })).toBeDisabled()
  })
})

describe('Dars arxivi — yozuv holatlari', () => {
  it('`archived` → tiklash → poll → `ready` oqimi pleyerni o‘zi ochadi', async () => {
    const user = userEvent.setup()
    getLessonArchive.mockResolvedValue(archive(ARCHIVED))
    restoreRecording.mockResolvedValue({ status: 'restoring', poll_after_s: 1 })
    getRecording.mockResolvedValue({ id: 'r3', status: 'restoring' })
    setup()

    expect(await screen.findByText('Bu dars 30 kundan eski')).toBeInTheDocument()
    expect(screen.queryByTestId('archive-video')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Videoni tiklash/ }))
    // TanStack v5 mutatsiya funksiyasiga ikkinchi argument (kontekst) uzatadi.
    expect(restoreRecording.mock.calls[0][0]).toBe('r3')

    // Tiklash boshlandi — spinner va sabab ko'rinadi, tugma yo'q.
    expect(await screen.findByText('Telegramdan yuklanmoqda…')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Videoni tiklash/ })).not.toBeInTheDocument()

    // Poll `ready` deydi → arxiv qayta so'raladi va endi havola bilan keladi.
    getRecording.mockResolvedValue({ id: 'r3', status: 'ready' })
    getLessonArchive.mockResolvedValue(archive(READY))

    expect(await screen.findByTestId('archive-video', {}, { timeout: 5000 })).toBeInTheDocument()
    expect(screen.queryByText('Telegramdan yuklanmoqda…')).not.toBeInTheDocument()
  }, 10_000)

  // Poll `ready` deyishi bilan arxivning yangi javobi kelishi ORASIDA havola
  // hali yo'q. O'sha oraliqda «Yozuvda xatolik» ko'rsatish yolg'on bo'lardi.
  it('`ready`, lekin havola hali kelmagan oraliqda xato EMAS, kutish ko‘rinadi', async () => {
    getLessonArchive.mockResolvedValue(archive({ ...READY, url: null }))
    setup()
    expect(await screen.findByText('Video ochilmoqda…')).toBeInTheDocument()
    expect(screen.queryByText('Yozuvda xatolik')).not.toBeInTheDocument()
  })

  it('`processing` holatida pleyer emas, tayyorlanish xabari ko‘rinadi', async () => {
    getLessonArchive.mockResolvedValue(archive({ ...ARCHIVED, status: 'processing' }))
    getRecording.mockResolvedValue({ id: 'r3', status: 'processing' })
    setup()
    expect(await screen.findByText('Yozuv tayyorlanmoqda…')).toBeInTheDocument()
    expect(screen.queryByTestId('archive-video')).not.toBeInTheDocument()
  })

  it('`expired` holatida tiklash tugmasi YO‘Q — Telegram nusxasi ham yo‘q', async () => {
    getLessonArchive.mockResolvedValue(archive({ ...ARCHIVED, status: 'expired' }))
    setup()
    expect(await screen.findByText('Video muddati tugagan')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Videoni tiklash/ })).not.toBeInTheDocument()
  })

  it('yozuv umuman yo‘q bo‘lsa sabab aytiladi', async () => {
    getLessonArchive.mockResolvedValue(archive(null))
    setup()
    expect(await screen.findByText('Bu darsda video yozilmagan')).toBeInTheDocument()
  })
})

describe('Dars arxivi — chat va materiallar video holatidan mustaqil', () => {
  // Chat DB'da yashaydi va yo'qolmaydi; video esa 30 kundan keyin serverdan
  // ketadi. Shuning uchun video holati chat qismini HECH QACHON yashirmaydi.
  for (const status of ['archived', 'expired', 'processing', 'failed']) {
    it(`\`${status}\` holatida ham chat va materiallar ko‘rinadi`, async () => {
      getLessonArchive.mockResolvedValue(archive({ ...ARCHIVED, status }))
      getRecording.mockResolvedValue({ id: 'r3', status })
      setup()
      expect(await screen.findByText('Ustoz, savol bor')).toBeInTheDocument()
      expect(screen.getByText('Assalomu alaykum')).toBeInTheDocument()
      expect(screen.getByText('uchburchaklar.pdf')).toBeInTheDocument()
    })
  }

  it('yozuv umuman yo‘q bo‘lganda ham chat to‘liq ko‘rinadi', async () => {
    getLessonArchive.mockResolvedValue(archive(null))
    setup()
    expect(await screen.findByText('Marhamat')).toBeInTheDocument()
    expect(screen.getByText('shaxsiy')).toBeInTheDocument()
  })

  it('chat bo‘sh bo‘lsa aniq xabar, materiallar bo‘limi baribir bor', async () => {
    getLessonArchive.mockResolvedValue({ lesson: LESSON, recording: READY, chat: [], materials: [] })
    setup()
    expect(await screen.findByText('Bu darsda chat yozilmagan')).toBeInTheDocument()
    expect(screen.getByText('Bu darsda fayl ulashilmagan.')).toBeInTheDocument()
  })
})

describe('Dars arxivi — chat transkripti', () => {
  it('TXT va HTML alohida so‘raladi', async () => {
    const user = userEvent.setup()
    getLessonArchive.mockResolvedValue(archive(READY))
    getChatTranscript.mockResolvedValue({ blob: new Blob(['x']), filename: 'dars-chat.txt' })
    setup()

    await user.click(await screen.findByRole('button', { name: /TXT sifatida yuklab olish/ }))
    expect(getChatTranscript).toHaveBeenCalledWith('l2', 'txt')

    await user.click(screen.getByRole('button', { name: /HTML sifatida yuklab olish/ }))
    expect(getChatTranscript).toHaveBeenCalledWith('l2', 'html')
  })

  it('chat bo‘sh bo‘lsa yuklab olish o‘chirilgan', async () => {
    getLessonArchive.mockResolvedValue({ lesson: LESSON, recording: READY, chat: [], materials: [] })
    setup()
    expect(await screen.findByRole('button', { name: /TXT sifatida yuklab olish/ })).toBeDisabled()
    expect(screen.getByRole('button', { name: /HTML sifatida yuklab olish/ })).toBeDisabled()
  })
})

describe('Dars arxivi — xato holati', () => {
  it('so‘rov yiqilsa qayta urinish tugmasi bilan xabar chiqadi', async () => {
    getLessonArchive.mockRejectedValue(new Error('network'))
    setup()
    expect(await screen.findByText(/Dars arxivini yuklab bo‘lmadi/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Qayta urinish/ })).toBeInTheDocument()
  })
})

// Backend Telegramdan TIKLANGAN nusxa uchun ham asl `ended_at + 30 kun` ni
// yuborishi mumkin — u allaqachon o'tgan sana. Ishlayotgan pleyer yonida
// «Muddati tugagan» deb yozish qarama-qarshilik bo'lardi.
describe('Dars arxivi — muddat izohi', () => {
  it('kelajakdagi muddat ko‘rsatiladi', async () => {
    getLessonArchive.mockResolvedValue(
      archive({ ...READY, expires_at: new Date(Date.now() + 12 * 86_400_000).toISOString() }),
    )
    setup()
    expect(await screen.findByText(/12 kundan keyin o‘chadi/)).toBeInTheDocument()
  })

  it('o‘tib ketgan muddat UMUMAN ko‘rsatilmaydi', async () => {
    getLessonArchive.mockResolvedValue(
      archive({ ...READY, expires_at: new Date(Date.now() - 5 * 86_400_000).toISOString() }),
    )
    setup()
    await screen.findByTestId('archive-video')
    expect(screen.queryByText(/Muddati tugagan/)).not.toBeInTheDocument()
    expect(screen.queryByText(/Telegram arxivida saqlanib qoladi/)).not.toBeInTheDocument()
  })

  it('muddat yo‘q bo‘lsa izoh chiqmaydi', async () => {
    getLessonArchive.mockResolvedValue(archive({ ...READY, expires_at: null }))
    setup()
    await screen.findByTestId('archive-video')
    expect(screen.queryByText(/Telegram arxivida saqlanib qoladi/)).not.toBeInTheDocument()
  })
})

// Tiklash oqimining YIQILGAN yo'llari. Ular avval jimgina yo'qolardi:
// mentor tugmani bosib, kutib, aynan o'sha kartochkaga qaytardi.
describe('Dars arxivi — tiklash yiqilganda', () => {
  it('poll `archived` ga qaytsa aniq xato va «Qayta urinish» chiqadi', async () => {
    const user = userEvent.setup()
    getLessonArchive.mockResolvedValue(archive(ARCHIVED))
    restoreRecording.mockResolvedValue({ status: 'restoring', poll_after_s: 1 })
    getRecording.mockResolvedValue({ id: 'r3', status: 'restoring' })
    setup()

    await user.click(await screen.findByRole('button', { name: /Videoni tiklash/ }))
    await screen.findByText('Telegramdan yuklanmoqda…')

    // Server tiklashni uddalay olmadi → `archived` + sabab.
    getRecording.mockResolvedValue({
      id: 'r3',
      status: 'archived',
      telegram_error: 'file too large',
    })

    expect(await screen.findByText('Videoni tiklab bo‘lmadi', {}, { timeout: 5000 })).toBeInTheDocument()
    expect(screen.getByText(/file too large/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Qayta urinish/ })).toBeInTheDocument()
  }, 10_000)

  it('sabab bo‘lmasa ham xato kartochkasi ko‘rinadi', async () => {
    const user = userEvent.setup()
    getLessonArchive.mockResolvedValue(archive(ARCHIVED))
    restoreRecording.mockResolvedValue({ status: 'restoring', poll_after_s: 1 })
    getRecording.mockResolvedValue({ id: 'r3', status: 'archived' })
    setup()

    await user.click(await screen.findByRole('button', { name: /Videoni tiklash/ }))
    expect(await screen.findByText('Videoni tiklab bo‘lmadi', {}, { timeout: 5000 })).toBeInTheDocument()
  }, 10_000)
})

// Presigned havola 1 soatlik. Sahifa undan uzoq ochiq tursa pleyer JIMGINA
// qora qolardi — endi bir marta yangilanadi, so'ng sabab aytiladi.
describe('Dars arxivi — havola muddati tugaganda', () => {
  it('birinchi xatoda arxiv qayta so‘raladi, ikkinchisida sabab chiqadi', async () => {
    getLessonArchive.mockResolvedValue(archive(READY))
    setup()

    const video = await screen.findByTestId('archive-video')
    video.dispatchEvent(new Event('error'))
    // Birinchi urinish — jim yangilash, xato kartochkasi YO'Q.
    await vi.waitFor(() => expect(getLessonArchive.mock.calls.length).toBeGreaterThan(1))
    expect(screen.queryByText('Video havolasi eskirdi')).not.toBeInTheDocument()

    screen.getByTestId('archive-video').dispatchEvent(new Event('error'))
    expect(await screen.findByText('Video havolasi eskirdi')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Qayta yuklash/ })).toBeInTheDocument()
  })
})

describe('Dars arxivi — materiallar', () => {
  it('havolasi olinmagan fayl bosilmaydigan bo‘ladi (a href="" emas)', async () => {
    getLessonArchive.mockResolvedValue({
      lesson: LESSON,
      recording: READY,
      chat: CHAT,
      materials: [{ name: 'imzosiz.pdf', size: 1024, mime: 'application/pdf', url: '', created_at: '2026-07-01T09:40:00Z' }],
    })
    setup()
    expect(await screen.findByText('imzosiz.pdf')).toBeInTheDocument()
    expect(screen.getByText('havola olinmadi')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /imzosiz\.pdf/ })).not.toBeInTheDocument()
  })
})

describe('Dars arxivi — xato matni', () => {
  it('serverning ROSTKI sababi ko‘rsatiladi (umumiy matn ostida yashirilmaydi)', async () => {
    const { ApiError } = await import('../api/api')
    getLessonArchive.mockRejectedValue(new ApiError('FORBIDDEN', 'forbidden', 403))
    setup()
    expect(await screen.findByText('Bu amalga ruxsatingiz yo‘q')).toBeInTheDocument()
  })
})
