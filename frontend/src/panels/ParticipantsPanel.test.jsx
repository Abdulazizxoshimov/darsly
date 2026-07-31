import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

// Moderatsiya API'lari — tarmoqsiz. Panel ularni to'g'ridan-to'g'ri chaqiradi.
vi.mock('../api/room', () => ({
  muteAll: vi.fn().mockResolvedValue(null),
  muteParticipant: vi.fn().mockResolvedValue(null),
  removeParticipant: vi.fn().mockResolvedValue(null),
  allowSpeak: vi.fn().mockResolvedValue(null),
  revokeSpeak: vi.fn().mockResolvedValue(null),
}))

// Kutish xonasi API'si — panel uni React Query orqali chaqiradi.
vi.mock('../api/waitingroom', () => ({
  listWaiting: vi.fn().mockResolvedValue([]),
  admitWaiting: vi.fn().mockResolvedValue(null),
  rejectWaiting: vi.fn().mockResolvedValue(null),
  admitAllWaiting: vi.fn().mockResolvedValue({ total: 0, admitted: 0, failed: 0 }),
  getWaitingStatus: vi.fn(),
}))

vi.mock('../lib/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}))

import { muteAll, removeParticipant } from '../api/room'
import { admitAllWaiting, listWaiting } from '../api/waitingroom'
import { toast } from '../lib/toast'
import { ParticipantsPanel } from './ParticipantsPanel'

beforeEach(() => {
  vi.clearAllMocks()
  listWaiting.mockResolvedValue([])
  admitAllWaiting.mockResolvedValue({ total: 0, admitted: 0, failed: 0 })
})

// Panel React Query'ga tayanadi (kutish xonasi ro'yxati). Testda tarmoqqa
// chiqmasligi uchun qayta urinishlar o'chiriladi.
function wrap(ui) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{ui}</QueryClientProvider>
}

const P = (identity, name, over = {}) => ({
  identity,
  name,
  isLocal: false,
  speaking: false,
  micMuted: true,
  camTrack: null,
  screenTrack: null,
  canPublish: false,
  ...over,
})

function setup(over = {}) {
  const onLowerHand = vi.fn()
  const onLowerAllHands = vi.fn()
  const onClose = vi.fn()
  const props = {
    participants: [P('me', 'Ustoz', { isLocal: true }), P('u1', 'Ali'), P('u2', 'Vali')],
    isHost: true,
    lessonId: 'l1',
    localId: 'me',
    raisedHands: new Map(),
    onLowerHand,
    onLowerAllHands,
    onClose,
    ...over,
  }
  render(wrap(<ParticipantsPanel {...props} />))
  return { onLowerHand, onLowerAllHands, onClose }
}

describe('ParticipantsPanel', () => {
  it('ishtirokchilar soni sarlavhada ko‘rsatiladi', () => {
    setup()
    expect(screen.getByText('Ishtirokchilar (3)')).toBeInTheDocument()
  })

  it('qo‘l ko‘tarilmagan bo‘lsa navbat bo‘limi umuman chizilmaydi', () => {
    setup()
    expect(screen.queryByText(/Qo'l ko'targanlar/)).not.toBeInTheDocument()
  })

  it('qo‘l ko‘targanlar NAVBAT tartibida va raqamlangan holda chiqadi', () => {
    // Map insert tartibi = navbat (server `at` bo'yicha bergan tartib).
    const raisedHands = new Map([
      ['u2', { name: 'Vali', at: 100 }],
      ['u1', { name: 'Ali', at: 200 }],
    ])
    setup({ raisedHands })

    expect(screen.getByText("Qo'l ko'targanlar (2)")).toBeInTheDocument()
    const nums = screen.getAllByText(/^[12]$/).map((el) => el.textContent)
    expect(nums).toEqual(['1', '2'])
  })

  it('o‘zimizning qo‘limiz navbatda ko‘rsatilmaydi', () => {
    const raisedHands = new Map([['me', { name: 'Ustoz', at: 1 }]])
    setup({ raisedHands })
    expect(screen.queryByText(/Qo'l ko'targanlar/)).not.toBeInTheDocument()
  })

  it('xonada bo‘lmagan ishtirokchining qo‘li ko‘rsatilmaydi', () => {
    // Chiqib ketgan o'quvchining qo'li navbatda osilib qolmasligi kerak.
    const raisedHands = new Map([['chiqib_ketgan', { name: 'Yo‘q', at: 1 }]])
    setup({ raisedHands })
    expect(screen.queryByText(/Qo'l ko'targanlar/)).not.toBeInTheDocument()
  })

  it('qo‘lni tushirish tugmasi identity bilan chaqiriladi', async () => {
    const user = userEvent.setup()
    const raisedHands = new Map([['u1', { name: 'Ali', at: 1 }]])
    const { onLowerHand } = setup({ raisedHands })

    await user.click(screen.getByTitle("Qo'lni tushirish"))
    expect(onLowerHand).toHaveBeenCalledWith('u1')
  })

  it('"hammasini tushirish" tugmasi ishlaydi', async () => {
    const user = userEvent.setup()
    const raisedHands = new Map([['u1', { name: 'Ali', at: 1 }]])
    const { onLowerAllHands } = setup({ raisedHands })

    await user.click(screen.getByText('Hammasini tushirish'))
    expect(onLowerAllHands).toHaveBeenCalled()
  })

  it('so‘zga ruxsat berilgan o‘quvchida "ruxsat berish" tugmasi ko‘rsatilmaydi', () => {
    const raisedHands = new Map([['u1', { name: 'Ali', at: 1 }]])
    setup({
      raisedHands,
      participants: [P('me', 'Ustoz', { isLocal: true }), P('u1', 'Ali', { canPublish: true })],
    })
    // Navbatda faqat "tushirish" qoladi — ruxsat allaqachon berilgan.
    expect(screen.getByTitle("Qo'lni tushirish")).toBeInTheDocument()
    expect(screen.queryByTitle("So'zga ruxsat berish")).not.toBeInTheDocument()
  })

  it('o‘quvchi (host emas) moderatsiya tugmalarini ko‘rmaydi', () => {
    setup({ isHost: false })
    expect(screen.queryByText("Hammani o'chirish")).not.toBeInTheDocument()
    expect(screen.queryByTitle('Chiqarib yuborish')).not.toBeInTheDocument()
  })

  it('host o‘ziga moderatsiya tugmalarini ko‘rmaydi', () => {
    setup()
    // 2 ta o'quvchi bor → har biriga 3 tadan tugma; o'zimizga 0 ta.
    expect(screen.getAllByTitle('Chiqarib yuborish')).toHaveLength(2)
  })

  // ── «Hammani o'chirish» (Zoom andozasi: modal + checkbox) ──────────────────

  it('«Hammani o‘chirish» modal ochadi va default checkbox bilan allow_self_unmute=true yuboradi', async () => {
    const user = userEvent.setup()
    const onPolicyChanged = vi.fn()
    setup({ onPolicyChanged })

    await user.click(screen.getByText("Hammani o'chirish"))
    // Modal ochildi — checkbox (o'zi ocholmasin) default O'CHIQ.
    expect(screen.getByText("O'quvchilar o'zi ocholmasin")).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /^o'chirish$/i }))

    await waitFor(() => expect(muteAll).toHaveBeenCalledWith('l1', true))
    expect(onPolicyChanged).toHaveBeenCalledWith(true)
  })

  it('checkbox belgilansa allow_self_unmute=false yuboriladi', async () => {
    const user = userEvent.setup()
    const onPolicyChanged = vi.fn()
    setup({ onPolicyChanged })

    await user.click(screen.getByText("Hammani o'chirish"))
    await user.click(screen.getByRole('switch'))
    await user.click(screen.getByRole('button', { name: /^o'chirish$/i }))

    await waitFor(() => expect(muteAll).toHaveBeenCalledWith('l1', false))
    expect(onPolicyChanged).toHaveBeenCalledWith(false)
  })

  // ── Chiqarish tanlovi (№4: bir darslik / doimiy) ───────────────────────────

  it('chiqarishda tanlov dialogi: «Shu darsdan» scope=lesson yuboradi', async () => {
    const user = userEvent.setup()
    setup()

    await user.click(screen.getAllByTitle('Chiqarib yuborish')[0])
    await user.click(screen.getByRole('button', { name: 'Shu darsdan' }))

    await waitFor(() => expect(removeParticipant).toHaveBeenCalledWith('l1', 'u1', 'lesson'))
  })

  it('«Doimiy» tanlansa scope=mentor yuboriladi', async () => {
    const user = userEvent.setup()
    setup()

    await user.click(screen.getAllByTitle('Chiqarib yuborish')[1])
    await user.click(screen.getByRole('button', { name: /Doimiy/ }))

    await waitFor(() => expect(removeParticipant).toHaveBeenCalledWith('l1', 'u2', 'mentor'))
  })

  it('chiqarish darhol API chaqirmaydi — avval tanlov so‘raladi', async () => {
    const user = userEvent.setup()
    setup()
    await user.click(screen.getAllByTitle('Chiqarib yuborish')[0])
    expect(removeParticipant).not.toHaveBeenCalled()
  })

  // ── «Hammasini kiritish» (100–300 kishilik dars) ───────────────────────────

  const QUEUE = [
    { id: 'w1', lesson_id: 'l1', requester_name: 'Sardor', status: 'pending', created_at: '2026-07-31T09:00:00Z' },
    { id: 'w2', lesson_id: 'l1', requester_name: 'Nilufar', status: 'pending', created_at: '2026-07-31T09:00:05Z' },
  ]

  it('navbat BO‘SH bo‘lsa «Hammasini kiritish» umuman ko‘rinmaydi', async () => {
    setup()
    await waitFor(() => expect(listWaiting).toHaveBeenCalled())
    expect(screen.queryByText('Hammasini kiritish')).not.toBeInTheDocument()
  })

  it('navbat bo‘lsa tugma chiqadi, tasdiq NECHTA ekanini aytadi va API chaqiriladi', async () => {
    const user = userEvent.setup()
    listWaiting.mockResolvedValue(QUEUE)
    admitAllWaiting.mockResolvedValue({ total: 2, admitted: 2, failed: 0 })
    setup()

    await user.click(await screen.findByText('Hammasini kiritish'))
    // Tasdiqda son bor — bir bosishda butun navbat kirib qolmasin.
    expect(screen.getByText('2', { selector: 'strong' })).toBeInTheDocument()
    // Tasdiqdan OLDIN hech narsa yuborilmaydi.
    expect(admitAllWaiting).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: /^kiritish$/i }))
    // TanStack Query mutationFn'ga ikkinchi argument (kontekst) ham uzatadi —
    // shuning uchun faqat BIRINCHI argumentni tekshiramiz.
    await waitFor(() => expect(admitAllWaiting).toHaveBeenCalled())
    expect(admitAllWaiting.mock.calls[0][0]).toBe('l1')
    expect(toast.success).toHaveBeenCalledWith('2 o‘quvchi kiritildi')
  })

  it('qisman muvaffaqiyat ROSTINI aytadi (failed>0)', async () => {
    const user = userEvent.setup()
    listWaiting.mockResolvedValue(QUEUE)
    admitAllWaiting.mockResolvedValue({ total: 2, admitted: 1, failed: 1 })
    setup()

    await user.click(await screen.findByText('Hammasini kiritish'))
    await user.click(screen.getByRole('button', { name: /^kiritish$/i }))

    await waitFor(() =>
      expect(toast.info).toHaveBeenCalledWith('1 o‘quvchi kiritildi · 1 tasi kiritilmadi'),
    )
    expect(toast.success).not.toHaveBeenCalled()
  })

  it('o‘quvchi (host emas) kutish navbatini ham, tugmani ham ko‘rmaydi', async () => {
    listWaiting.mockResolvedValue(QUEUE)
    setup({ isHost: false })
    await waitFor(() => expect(screen.queryByText('Hammasini kiritish')).not.toBeInTheDocument())
    expect(listWaiting).not.toHaveBeenCalled()
  })

  // ── Qidiruv (300 kishilik darsda aniq odamni topish) ──────────────────────

  const many = (n) => Array.from({ length: n }, (_, i) => P(`u${i}`, i === 0 ? 'Alisher' : `O‘quvchi ${i}`))

  it('ro‘yxat 10 tadan kam bo‘lsa qidiruv maydoni CHIZILMAYDI', () => {
    setup({ participants: many(9) })
    expect(screen.queryByLabelText('Ishtirokchilarni qidirish')).not.toBeInTheDocument()
  })

  it('10 tadan boshlab qidiruv chiqadi va ism bo‘yicha filtrlaydi (katta-kichik harf farqsiz)', async () => {
    const user = userEvent.setup()
    setup({ participants: many(12) })

    const input = screen.getByLabelText('Ishtirokchilarni qidirish')
    await user.type(input, 'alisher')

    expect(screen.getByText('Alisher')).toBeInTheDocument()
    expect(screen.queryByText('O‘quvchi 5')).not.toBeInTheDocument()
    // Sarlavhada «topilgan/jami».
    expect(screen.getByText('Ishtirokchilar (1/12)')).toBeInTheDocument()
  })

  it('hech kim topilmasa bo‘sh holat ko‘rsatiladi', async () => {
    const user = userEvent.setup()
    setup({ participants: many(12) })

    await user.type(screen.getByLabelText('Ishtirokchilarni qidirish'), 'zzz')
    expect(screen.getByText('Hech kim topilmadi')).toBeInTheDocument()
    expect(screen.getByText('Ishtirokchilar (0/12)')).toBeInTheDocument()
  })
})
