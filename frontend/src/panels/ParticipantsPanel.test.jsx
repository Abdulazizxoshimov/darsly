import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ParticipantsPanel } from './ParticipantsPanel'

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
    expect(screen.queryByText('Hammani mute')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Chiqarib yuborish')).not.toBeInTheDocument()
  })

  it('host o‘ziga moderatsiya tugmalarini ko‘rmaydi', () => {
    setup()
    // 2 ta o'quvchi bor → har biriga 3 tadan tugma; o'zimizga 0 ta.
    expect(screen.getAllByTitle('Chiqarib yuborish')).toHaveLength(2)
  })
})
