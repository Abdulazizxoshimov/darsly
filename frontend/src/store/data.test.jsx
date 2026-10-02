import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  qk,
  useEndLesson,
  useOpenRoom,
  useStartRecording,
  useStopRecording,
  useWaitingStatus,
  waitingPollInterval,
  WAITING_POLL_MS,
} from './data'

// Kesh bekor qilish — xona ichidagi amallar (xonani ochish, yakunlash, yozuv)
// Dashboard/Jadval/Yozuvlar keshini o'zgartiradi. Avval ular oddiy API
// chaqiruvi edi va `staleTime: 30s` tufayli ustoz xonadan qaytganda 30
// soniyagacha eski holatni ko'rardi («Rejalashtirilgan» deb turgan jonli dars).

vi.mock('../api/lessons', () => ({
  getHostToken: vi.fn().mockResolvedValue({ token: 't', ws_url: 'ws://x' }),
  endLesson: vi.fn().mockResolvedValue(null),
  listLessons: vi.fn(),
  getLesson: vi.fn(),
  createLesson: vi.fn(),
  updateLesson: vi.fn(),
  deleteLesson: vi.fn(),
}))
vi.mock('../api/recordings', () => ({
  startRecording: vi.fn().mockResolvedValue({ id: 'r1' }),
  stopRecording: vi.fn().mockResolvedValue(null),
  listRecordings: vi.fn(),
  getRecording: vi.fn(),
  downloadRecording: vi.fn(),
  restoreRecording: vi.fn(),
}))
vi.mock('../api/waitingroom', () => ({
  getWaitingStatus: vi.fn(),
  listWaiting: vi.fn(),
  admitWaiting: vi.fn(),
  rejectWaiting: vi.fn(),
  admitAllWaiting: vi.fn(),
}))

import { getWaitingStatus } from '../api/waitingroom'

let qc
const wrapper = ({ children }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>

// Bekor qilingan (invalidated) kalitlar ro'yxati — `staleTime` dan qat'i nazar
// keyingi mount/refetch'da qayta so'raladi.
const invalidated = () =>
  qc
    .getQueryCache()
    .getAll()
    .filter((q) => q.state.isInvalidated)
    .map((q) => q.queryKey)

beforeEach(() => {
  vi.clearAllMocks()
  qc = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 30_000 } } })
  // Keshni oldindan to'ldiramiz — bekor qilish aynan shularga tegishi kerak.
  qc.setQueryData(qk.lessonList({ limit: 100 }), { data: [] })
  qc.setQueryData(qk.lessonList({ status: 'scheduled' }), { data: [] })
  qc.setQueryData(qk.lesson('l1'), { id: 'l1', status: 'scheduled' })
  qc.setQueryData(qk.lesson('l2'), { id: 'l2' })
  qc.setQueryData(qk.recordingsOf('l1'), [])
  qc.setQueryData(qk.recordingsOf('l2'), [])
  qc.setQueryData(qk.recording('r1'), { id: 'r1', status: 'recording' })
  qc.setQueryData(qk.lessonArchive('l1'), { chat: [] })
})

describe('xona mutatsiyalari — kesh bekor qilish', () => {
  it('useOpenRoom: darslar ro‘yxati (hammasi) va SHU dars bekor qilinadi', async () => {
    const { result } = renderHook(() => useOpenRoom(), { wrapper })
    await act(() => result.current.mutateAsync('l1'))
    const keys = invalidated()
    expect(keys).toContainEqual(qk.lessonList({ limit: 100 }))
    expect(keys).toContainEqual(qk.lessonList({ status: 'scheduled' }))
    expect(keys).toContainEqual(qk.lesson('l1'))
    // Boshqa dars va yozuvlar TEGILMAYDI.
    expect(keys).not.toContainEqual(qk.lesson('l2'))
    expect(keys).not.toContainEqual(qk.recordingsOf('l1'))
  })

  it('useEndLesson: darslar + dars + yozuvlar + arxiv bekor qilinadi', async () => {
    const { result } = renderHook(() => useEndLesson(), { wrapper })
    await act(() => result.current.mutateAsync('l1'))
    const keys = invalidated()
    expect(keys).toContainEqual(qk.lessonList({ limit: 100 }))
    expect(keys).toContainEqual(qk.lesson('l1'))
    expect(keys).toContainEqual(qk.recordingsOf('l1'))
    expect(keys).toContainEqual(qk.recordingsOf('l2')) // yozuvlar sahifasi butunlay
    expect(keys).toContainEqual(qk.lessonArchive('l1'))
  })

  it('useStartRecording: faqat shu dars yozuvlari bekor qilinadi', async () => {
    const { result } = renderHook(() => useStartRecording(), { wrapper })
    await act(() => result.current.mutateAsync('l1'))
    const keys = invalidated()
    expect(keys).toContainEqual(qk.recordingsOf('l1'))
    expect(keys).not.toContainEqual(qk.recordingsOf('l2'))
    expect(keys).not.toContainEqual(qk.lesson('l1'))
  })

  it('useStopRecording: dars yozuvlari va yozuvning o‘zi bekor qilinadi', async () => {
    const { result } = renderHook(() => useStopRecording(), { wrapper })
    await act(() => result.current.mutateAsync({ recordingId: 'r1', lessonId: 'l1' }))
    const keys = invalidated()
    expect(keys).toContainEqual(qk.recordingsOf('l1'))
    expect(keys).toContainEqual(qk.recording('r1'))
  })
})

describe('useWaitingStatus — 429 da sekinlashadi, 404 da qayta urinmaydi', () => {
  it('poll oralig‘i: oddiy 2s, 429 dan keyin 6s', () => {
    expect(waitingPollInterval({ state: { error: null } })).toBe(WAITING_POLL_MS)
    expect(waitingPollInterval({ state: { error: { status: 429 } } })).toBe(WAITING_POLL_MS * 3)
    expect(waitingPollInterval({ state: { error: { status: 500 } } })).toBe(WAITING_POLL_MS)
  })

  // Bug: 404 (so'rov yo'q — dars yakunlangan) bir marta qayta urinilardi va
  // mehmon «topilmadi» ekranini kechroq ko'rardi; 429 esa DARHOL xato bo'lib
  // yolg'on «So'rov topilmadi» ga aylanardi.
  it('404 → xato darhol (retry yo‘q); 429 → bir marta qayta uriniladi', async () => {
    getWaitingStatus.mockRejectedValue(Object.assign(new Error('nf'), { status: 404 }))
    const { result } = renderHook(() => useWaitingStatus('req1'), { wrapper })
    await vi.waitFor(() => expect(result.current.isError).toBe(true))
    expect(getWaitingStatus).toHaveBeenCalledTimes(1)

    getWaitingStatus.mockClear()
    getWaitingStatus.mockRejectedValue(Object.assign(new Error('rl'), { status: 429 }))
    const r2 = renderHook(() => useWaitingStatus('req2'), { wrapper })
    await vi.waitFor(() => expect(r2.result.current.isError).toBe(true), { timeout: 5000 })
    expect(getWaitingStatus).toHaveBeenCalledTimes(2)
  }, 10000)
})
