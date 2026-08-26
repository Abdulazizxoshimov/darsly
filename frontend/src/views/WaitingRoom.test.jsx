import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { WaitingRoom } from './WaitingRoom'

// F-4 — kutish xonasi: WS real-time admit/reject + polling fallback reconciliation.
//
// Ikki manba (tez WS push va sekin 2s polling) BIR XIL holatga olib kelishi
// kerak. Nega kritik: agar polling admit'ni ko'rmasa, WS uzilganда mehmon
// abadiy kutadi; agar reject ikkala yo'ldan ham ishlamasa, rad etilgan mehmon
// spinnerni cheksiz ko'radi.

const h = vi.hoisted(() => ({
  status: { data: undefined, isError: false },
  wsHandler: null,
  cleanup: vi.fn(),
  setRoom: vi.fn(),
  pending: null,
}))

vi.mock('../store/data', () => ({ useWaitingStatus: () => h.status }))
vi.mock('../lib/ws', () => ({
  connectWaitingRoom: (id, cb) => {
    h.wsHandler = cb
    return h.cleanup
  },
}))
vi.mock('../lib/roomSession', () => ({
  roomSession: { get: () => ({ pending: h.pending }), setRoom: h.setRoom },
}))

const PENDING = { requestId: 'req1', lesson: { id: 'l1', title: 'Kvadrat tenglamalar' }, guestName: 'Aziz' }

function setup() {
  render(
    <MemoryRouter initialEntries={['/r/demo123/waiting']}>
      <Routes>
        <Route path="/r/:slug/waiting" element={<WaitingRoom />} />
        <Route path="/r/:slug/room" element={<div data-testid="in-room">xona</div>} />
        <Route path="/r/:slug" element={<div data-testid="join">join</div>} />
        <Route path="/" element={<div data-testid="home">home</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  h.status = { data: undefined, isError: false }
  h.wsHandler = null
  h.pending = PENDING
  h.cleanup.mockClear()
  h.setRoom.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('WaitingRoom — WS real-time yo‘li', () => {
  it('boshida kutish holati ko‘rinadi (dars sarlavhasi bilan)', () => {
    setup()
    expect(screen.getByText('Kutish xonasidasiz')).toBeInTheDocument()
    expect(screen.getByText(/Kvadrat tenglamalar/)).toBeInTheDocument()
  })

  // Bug: WS admit push token'ni sessiyaga yozib xonaga o'tkazmasa, mehmon
  // tasdiqlangan bo'lsa ham kutishда qoladi.
  it('WS admit → token sessiyaga yoziladi va xonaga o‘tiladi', () => {
    setup()
    act(() => h.wsHandler({ type: 'waiting_room.admitted', payload: 'ROOM-TOKEN-123' }))
    expect(h.setRoom).toHaveBeenCalledWith({
      token: 'ROOM-TOKEN-123',
      lesson: PENDING.lesson,
      guestName: 'Aziz',
    })
    expect(screen.getByTestId('in-room')).toBeInTheDocument()
  })

  // Bug: payload'siz admit (buzuq xabar) navigatsiyani ishga tushirsa, mehmon
  // token'siz xonaga o'tib LiveKit'da yiqiladi.
  it('WS admit payload‘siz kelsa xonaga O‘TILMAYDI', () => {
    setup()
    act(() => h.wsHandler({ type: 'waiting_room.admitted' }))
    expect(h.setRoom).not.toHaveBeenCalled()
    expect(screen.queryByTestId('in-room')).not.toBeInTheDocument()
  })

  // Bug: reject ko'rsatilmasa rad etilgan mehmon spinnerni cheksiz ko'radi.
  it('WS reject → «Kirish rad etildi» ko‘rinadi', () => {
    setup()
    act(() => h.wsHandler({ type: 'waiting_room.rejected' }))
    expect(screen.getByText('Kirish rad etildi')).toBeInTheDocument()
  })
})

describe('WaitingRoom — polling fallback', () => {
  // Bug: WS uzilib qolsa, polling admit'ni ko'rib xonaga o'tkazishi SHART —
  // aks holda mehmon abadiy kutadi.
  it('polling status=admitted → xonaga o‘tiladi', () => {
    h.status = { data: { status: 'admitted', room: 'POLL-TOKEN' }, isError: false }
    setup()
    expect(h.setRoom).toHaveBeenCalledWith({ token: 'POLL-TOKEN', lesson: PENDING.lesson, guestName: 'Aziz' })
    expect(screen.getByTestId('in-room')).toBeInTheDocument()
  })

  it('polling status=rejected → «Kirish rad etildi»', () => {
    h.status = { data: { status: 'rejected' }, isError: false }
    setup()
    expect(screen.getByText('Kirish rad etildi')).toBeInTheDocument()
  })

  // ⭐ "Ustoz darsni yakunladi, mehmon hali kutmoqda": status so'rovi 404/xato
  // beradi. Bug: bu holatda spinner qolib ketsa, mehmon dars tugaganini bilmaydi.
  it('status so‘rovi xato bo‘lsa (ustoz darsni yakunladi) «So‘rov topilmadi»', () => {
    h.status = { data: undefined, isError: true }
    setup()
    expect(screen.getByText("So'rov topilmadi")).toBeInTheDocument()
    expect(screen.getByText(/muddati o'tgan yoki bekor qilingan/i)).toBeInTheDocument()
  })
})

describe('WaitingRoom — sessiyasiz kirish', () => {
  // Bug: to'g'ridan-to'g'ri kutish URL'iga kelgan (sessiya yo'q) foydalanuvchi
  // oq ekran ko'rmasligi, balki nima qilishni bilishi kerak.
  it('pending yo‘q bo‘lsa «Sessiya topilmadi» va qaytish tugmasi', () => {
    h.pending = null
    setup()
    expect(screen.getByText('Sessiya topilmadi.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Qaytadan urinish' })).toBeInTheDocument()
  })
})
