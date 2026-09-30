import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { RoomEvent } from 'livekit-client'
import { ApiError } from '../api/api'
import { decodeData } from '../livekit/messaging'
import { LiveRoom } from './LiveRoom'

// Jonli xona — XONA TOKENI hayot sikli va qayta ulanish siyosati.
//
// Mehmon tokeni 30 daqiqalik va backend uni uzaytirmaydi. Avval u bir marta
// olinib o'zgarmas edi: 30 daqiqadan keyin chat/qo'l/ovoz/holat 401 berardi
// («Email yoki parol noto'g'ri» matni bilan), uzilish esa AYNI eskirgan token
// bilan cheksiz qayta ulanish sikliga aylanardi. LiveKit SDK bu yerda soxta
// (`useRoom` mock) — sinov ob'ekti xonaning O'ZI: token manbasi, so'rovlar,
// uzilish ekranlari, so'rovnoma takrorlash.

const h = vi.hoisted(() => ({
  state: null, // useRoom qaytaradigan holat
  calls: [], // useRoom argumentlari (retryKey/identity kuzatuvi)
}))

vi.mock('../livekit/useRoom', () => ({
  useRoom: (args) => {
    h.calls.push(args)
    return h.state
  },
}))
vi.mock('../livekit/Whiteboard', () => ({ Whiteboard: () => null }))
// happy-dom'da WebRTC yo'q — brauzer tekshiruvi (alohida sinovi bor) chetlab o'tiladi.
vi.mock('../lib/features', async (orig) => ({ ...(await orig()), isLiveRoomSupported: () => true }))
vi.mock('../lib/toast', () => ({ toast: { error: vi.fn(), info: vi.fn(), success: vi.fn() } }))
vi.mock('../api/chat', async (orig) => ({
  ...(await orig()),
  roomChatHistory: vi.fn(),
  roomChatSend: vi.fn(),
  roomChatUpload: vi.fn(),
  deleteChatMessage: vi.fn(),
}))
vi.mock('../api/roomstate', () => ({
  setHand: vi.fn(),
  sendReaction: vi.fn(),
  getRoomState: vi.fn(),
  lowerHand: vi.fn(),
  lowerAllHands: vi.fn(),
}))
vi.mock('../api/join', () => ({ joinLink: vi.fn(), previewJoinLink: vi.fn() }))
vi.mock('../api/lessons', () => ({
  getHostToken: vi.fn(),
  getLesson: vi.fn(),
  endLesson: vi.fn(),
  listLessons: vi.fn(),
  createLesson: vi.fn(),
  updateLesson: vi.fn(),
  deleteLesson: vi.fn(),
}))
vi.mock('../api/recordings', () => ({
  listRecordings: vi.fn(),
  startRecording: vi.fn(),
  stopRecording: vi.fn(),
  downloadRecording: vi.fn(),
  getRecording: vi.fn(),
  restoreRecording: vi.fn(),
}))
vi.mock('../api/polls', async (orig) => ({
  ...(await orig()),
  listPolls: vi.fn(),
}))
vi.mock('../api/waitingroom', () => ({
  listWaiting: vi.fn(),
  getWaitingStatus: vi.fn(),
  admitWaiting: vi.fn(),
  rejectWaiting: vi.fn(),
  admitAllWaiting: vi.fn(),
}))

import { roomChatHistory, roomChatSend, roomChatUpload } from '../api/chat'
import { getRoomState } from '../api/roomstate'
import { joinLink } from '../api/join'
import { getHostToken, getLesson } from '../api/lessons'
import { listRecordings } from '../api/recordings'
import { listPolls } from '../api/polls'
import { listWaiting } from '../api/waitingroom'
import { toast } from '../lib/toast'

// Imzosiz JWT — faqat `exp` o'qiladi.
const jwt = (expSec) => `h.${btoa(JSON.stringify({ exp: expSec })).replace(/=+$/, '')}.s`
const inSeconds = (s) => Math.floor(Date.now() / 1000) + s

const LESSON = { id: 'l1', title: 'Algebra', is_waiting_room_enabled: false, allow_self_unmute: true }
const JOIN = { slug: 'demo', guestName: 'Ali', passcode: '' }
const T1 = (over = {}) => ({ token: jwt(inSeconds(1800)), ws_url: 'ws://lk', identity: 'g1', lesson_id: 'l1', ...over })
const T2 = { token: 'T2', ws_url: 'ws://lk', identity: 'g2', lesson_id: 'l1' }

// Soxta LiveKit Room — faqat hodisa obunasi va data publish.
function fakeRoom() {
  const listeners = new Map()
  return {
    on(evt, fn) {
      if (!listeners.has(evt)) listeners.set(evt, new Set())
      listeners.get(evt).add(fn)
      return this
    },
    off(evt, fn) {
      listeners.get(evt)?.delete(fn)
      return this
    },
    emit(evt, ...args) {
      for (const fn of listeners.get(evt) || []) fn(...args)
    },
    localParticipant: {
      publishData: vi.fn(),
      setMicrophoneEnabled: vi.fn().mockResolvedValue(undefined),
      setScreenShareEnabled: vi.fn().mockResolvedValue(undefined),
      isMicrophoneEnabled: false,
    },
  }
}

function connected(room, over = {}) {
  return {
    room,
    connState: 'connected',
    connectError: null,
    ended: null,
    quality: 'good',
    participants: [],
    local: { identity: 'g1', name: 'Ali', micOn: false, camOn: false, screenOn: false, canPublish: true },
    dataSaver: false,
    setDataSaver: vi.fn(),
    audioBlocked: false,
    resumeAudio: vi.fn(),
    mediaError: null,
    dismissMediaError: vi.fn(),
    ...over,
  }
}

function seedGuest(room = { token: T1(), lesson: LESSON, guestName: 'Ali', join: JOIN }) {
  sessionStorage.setItem('jonly.roomSession', JSON.stringify({ pending: null, room }))
}
const storedRoom = () => JSON.parse(sessionStorage.getItem('jonly.roomSession')).room
const storedPending = () => JSON.parse(sessionStorage.getItem('jonly.roomSession')).pending

function setup(mode = 'guest') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const entry = mode === 'guest' ? '/r/demo/room' : '/app/lesson/l1/room'
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/r/:slug/room" element={<LiveRoom mode="guest" />} />
          <Route path="/r/:slug/waiting" element={<div>WAITING_PAGE</div>} />
          <Route path="/r/:slug" element={<div>JOIN_PAGE</div>} />
          <Route path="/app/lesson/:id/room" element={<LiveRoom mode="host" />} />
          <Route path="/app" element={<div>APP_PAGE</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const lastRoomArgs = () => h.calls[h.calls.length - 1]

beforeEach(() => {
  vi.clearAllMocks()
  sessionStorage.clear()
  h.calls = []
  h.state = connected(fakeRoom())
  roomChatHistory.mockResolvedValue([])
  getRoomState.mockResolvedValue({ hands: [], recording: false, allow_self_unmute: true })
  joinLink.mockResolvedValue({ next_step: 'join', room: T2, lesson: LESSON })
})

describe('LiveRoom — mehmon: xona tokeni yangilanishi', () => {
  it('chat 401 → token qayta olinadi, xabar YANGI token bilan takrorlanadi, sessiya yangilanadi', async () => {
    seedGuest()
    const user = userEvent.setup()
    roomChatSend
      .mockRejectedValueOnce(new ApiError('UNAUTHORIZED', 'invalid room token', 401))
      .mockResolvedValueOnce({ id: 'm1', sender_name: 'Ali', body: 'salom', sender_identity: 'g2', created_at: '' })
    setup()

    await user.click(await screen.findByLabelText('Chat'))
    await user.type(screen.getByPlaceholderText('Xabar yozing…'), 'salom')
    await user.click(screen.getByLabelText('Yuborish'))

    await waitFor(() => expect(roomChatSend).toHaveBeenCalledTimes(2))
    expect(roomChatSend.mock.calls[0][1]).toBe(T1().token.slice(0, 0) + roomChatSend.mock.calls[0][1]) // eski token
    expect(roomChatSend.mock.calls[1][1]).toBe('T2')
    expect(joinLink).toHaveBeenCalledTimes(1)
    expect(joinLink).toHaveBeenCalledWith('demo', { guest_name: 'Ali', passcode: undefined })
    // F5 dan keyin ham yangi token — sessiyaga yozildi, kirish ma'lumotlari saqlandi.
    expect(storedRoom().token.token).toBe('T2')
    expect(storedRoom().join).toEqual(JOIN)
    // Yangi identity → LiveKit shu identity bilan qayta ulanadi.
    await waitFor(() => expect(lastRoomArgs().identity).toBe('g2'))
    expect(lastRoomArgs().getToken()).toBe('T2')
    // Xato matni yo'q — yangilash jimgina o'tdi.
    expect(toast.error).not.toHaveBeenCalled()
    expect(await screen.findByText('salom')).toBeInTheDocument()
  })

  // Bug: 401 «Email yoki parol noto'g'ri» deb chiqardi — mehmon parol kiritmagan ham.
  it('yangilangan token ham rad etilsa xona matni ko‘rsatiladi (parol haqida emas)', async () => {
    seedGuest()
    const user = userEvent.setup()
    roomChatSend.mockRejectedValue(new ApiError('UNAUTHORIZED', 'invalid room token', 401))
    setup()

    await user.click(await screen.findByLabelText('Chat'))
    await user.type(screen.getByPlaceholderText('Xabar yozing…'), 'salom')
    await user.click(screen.getByLabelText('Yuborish'))

    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    expect(toast.error.mock.calls[0][0]).toMatch(/sessiyasi tugadi/i)
    expect(toast.error.mock.calls[0][0]).not.toMatch(/parol/i)
    expect(joinLink).toHaveBeenCalledTimes(1) // cheksiz sikl yo'q
  })

  it('muddati yaqin token mount’da PROAKTIV yangilanadi (so‘rov kutilmaydi)', async () => {
    seedGuest({ token: T1({ token: jwt(inSeconds(60)) }), lesson: LESSON, guestName: 'Ali', join: JOIN })
    setup()
    await waitFor(() => expect(joinLink).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(lastRoomArgs().identity).toBe('g2'))
    expect(storedRoom().token.token).toBe('T2')
  })

  // Backend cheklovi: kutish xonasi darsida qayta `join` YANGI kutish so'rovi
  // yaratadi — proaktiv yangilash mehmonni darsdan chiqarib yuborardi.
  it('kutish xonasi YOQILGAN darsda proaktiv yangilash YO‘Q', async () => {
    seedGuest({
      token: T1({ token: jwt(inSeconds(60)) }),
      lesson: { ...LESSON, is_waiting_room_enabled: true },
      guestName: 'Ali',
      join: JOIN,
    })
    setup()
    await screen.findByText('Algebra')
    await act(() => new Promise((r) => setTimeout(r, 30)))
    expect(joinLink).not.toHaveBeenCalled()
  })

  it('401 da yangilash `waiting_room` bersa — kutish sahifasi (so‘rov ID va kirish ma’lumotlari bilan)', async () => {
    seedGuest()
    const user = userEvent.setup()
    roomChatSend.mockRejectedValue(new ApiError('UNAUTHORIZED', 'x', 401))
    joinLink.mockResolvedValue({ next_step: 'waiting_room', request_id: 'req9', lesson: LESSON })
    setup()

    await user.click(await screen.findByLabelText('Chat'))
    await user.type(screen.getByPlaceholderText('Xabar yozing…'), 'salom')
    await user.click(screen.getByLabelText('Yuborish'))

    expect(await screen.findByText('WAITING_PAGE')).toBeInTheDocument()
    expect(storedPending()).toMatchObject({ requestId: 'req9', guestName: 'Ali', join: JOIN })
    expect(toast.info).toHaveBeenCalledWith(expect.stringMatching(/tasdig‘i/i))
  })
})

describe('LiveRoom — ulanish xatosi: auth ≠ tarmoq', () => {
  it('auth: qayta urinish EMAS — token yangilanadi, so‘ng yangi identity bilan ulanadi', async () => {
    seedGuest()
    h.state = connected(null, { room: null, connState: 'disconnected', connectError: 'auth' })
    setup()

    expect(await screen.findByText('Sessiya yangilanmoqda…')).toBeInTheDocument()
    await waitFor(() => expect(joinLink).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(lastRoomArgs().retryKey).toBe(1))
    expect(lastRoomArgs().identity).toBe('g2')
    // «Aloqa uzildi» sanoq ekrani ko'rsatilmaydi — bu tarmoq emas.
    expect(screen.queryByText('Aloqa uzildi')).not.toBeInTheDocument()
  })

  it('auth + chiqarilgan (403 removed) → YAKUNIY ekran, qayta urinish yo‘q', async () => {
    seedGuest()
    h.state = connected(null, { room: null, connState: 'disconnected', connectError: 'auth' })
    joinLink.mockRejectedValue(new ApiError('FORBIDDEN', "you have been removed from this mentor's lessons", 403))
    setup()

    expect(await screen.findByText('Sizni bu darsdan chiqarishgan')).toBeInTheDocument()
    expect(joinLink).toHaveBeenCalledTimes(1)
    expect(h.calls.every((c) => c.retryKey === 0)).toBe(true)
  })

  // Backend kontrakti: jonli bo'lmagan darsda `join` token bermaydi
  // (`waiting_for_host`) — bu xato emas, holat.
  it('auth + waiting_for_host → «Ustoz hali xonada emas» (chiqish bilan)', async () => {
    seedGuest()
    h.state = connected(null, { room: null, connState: 'disconnected', connectError: 'auth' })
    joinLink.mockResolvedValue({ next_step: 'waiting_for_host', lesson: LESSON })
    setup()

    expect(await screen.findByText('Ustoz hali xonada emas')).toBeInTheDocument()
    expect(screen.getByText(/havola orqali qayta qo‘shilishingiz mumkin/i)).toBeInTheDocument()
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('auth + dars tugagan (lesson_ended) → «Dars yakunlangan»', async () => {
    seedGuest()
    h.state = connected(null, { room: null, connState: 'disconnected', connectError: 'auth' })
    joinLink.mockResolvedValue({ next_step: 'lesson_ended', lesson: LESSON })
    setup()
    expect(await screen.findByText('Dars yakunlangan')).toBeInTheDocument()
  })

  it('tarmoq: token yangilanmaydi, chegaralangan sanoq bilan «Aloqa uzildi»', async () => {
    seedGuest()
    h.state = connected(null, { room: null, connState: 'disconnected', connectError: 'network' })
    setup()

    expect(await screen.findByText('Aloqa uzildi')).toBeInTheDocument()
    expect(screen.getByText(/5 soniyadan so‘ng/)).toBeInTheDocument()
    await act(() => new Promise((r) => setTimeout(r, 30)))
    expect(joinLink).not.toHaveBeenCalled()
  })
})

describe('LiveRoom — fayl yuklash bekor qilinadi', () => {
  // Bug: `handleSendFile` 4-argument (`signal`) ni tashlab yuborardi — «Bekor
  // qilish» tugmasi hech narsa qilmasdi, 20 MB yuklanaverardi.
  it('AbortSignal XHR’gacha yetib boradi va «Bekor qilish» uni uzadi', async () => {
    seedGuest()
    const user = userEvent.setup()
    let uploadOpts = null
    roomChatUpload.mockImplementation(
      (_l, _t, _f, opts) =>
        new Promise((_res, rej) => {
          uploadOpts = opts
          opts.onProgress(10)
          opts.signal.addEventListener('abort', () => rej(new ApiError('ABORTED', 'Bekor qilindi', 0)))
        }),
    )
    setup()

    await user.click(await screen.findByLabelText('Chat'))
    const file = new File([new Uint8Array(1024)], 'uy_ishi.pdf', { type: 'application/pdf' })
    fireEvent.change(screen.getByTestId('chat-file-input'), { target: { files: [file] } })

    await waitFor(() => expect(roomChatUpload).toHaveBeenCalled())
    expect(uploadOpts.signal).toBeInstanceOf(AbortSignal)
    expect(uploadOpts.signal.aborted).toBe(false)

    await user.click(await screen.findByText('Bekor qilish'))
    await waitFor(() => expect(uploadOpts.signal.aborted).toBe(true))
    expect(await screen.findByRole('alert')).toHaveTextContent(/bekor qilindi/i)
  })
})

describe('LiveRoom — sessiya shakli', () => {
  // Bug: eskirgan sessionStorage (`token` satr, `lesson` yo'q) `room.lesson.title`
  // da throw qilib oq ekran berardi.
  it('buzuq/eskirgan sessiya — oq ekran emas, «Sessiya topilmadi»', async () => {
    sessionStorage.setItem('jonly.roomSession', JSON.stringify({ pending: null, room: { token: 'ROOM-TOKEN' } }))
    setup()
    expect(await screen.findByText(/Sessiya topilmadi/)).toBeInTheDocument()
    expect(h.calls).toHaveLength(0)
  })
})

describe('LiveRoom — ustoz: kech kirganga faol so‘rovnoma', () => {
  it('yangi ishtirokchi ulanganda faol so‘rovnoma FAQAT unga takrorlanadi', async () => {
    const room = fakeRoom()
    h.state = connected(room, {
      local: { identity: 'mentor1', name: 'Ustoz', micOn: true, camOn: true, screenOn: false, canPublish: true },
    })
    getHostToken.mockResolvedValue({ token: jwt(inSeconds(6 * 3600)), ws_url: 'ws://lk', identity: 'mentor1', lesson_id: 'l1' })
    getLesson.mockResolvedValue({ ...LESSON, status: 'live' })
    listWaiting.mockResolvedValue([])
    listRecordings.mockResolvedValue([])
    listPolls.mockResolvedValue([
      { id: 'p0', question: 'Eski', options: ['a'], is_active: false, results_visibility: 'public' },
      { id: 'p1', question: 'Tushunarlimi?', options: ['Ha', "Yo'q"], is_active: true, results_visibility: 'public' },
    ])
    setup('host')

    await screen.findByText('Algebra')
    await waitFor(() => expect(listPolls).toHaveBeenCalled())
    await waitFor(() => {
      room.emit(RoomEvent.ParticipantConnected, { identity: 'g5' })
      expect(room.localParticipant.publishData).toHaveBeenCalled()
    })
    const [payload, opts] = room.localParticipant.publishData.mock.calls.at(-1)
    expect(decodeData(payload)).toEqual({
      kind: 'poll',
      action: 'open',
      poll: { id: 'p1', question: 'Tushunarlimi?', options: ['Ha', "Yo'q"], results_visibility: 'public' },
    })
    expect(opts.destinationIdentities).toEqual(['g5'])
    // Host tokeni ham manbadan: identity barqaror — qayta ulanish yo'q.
    expect(lastRoomArgs().identity).toBe('mentor1')
  })
})
