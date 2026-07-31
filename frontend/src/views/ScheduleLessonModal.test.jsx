import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import { ScheduleLessonModal } from './ScheduleLessonModal'

// MAHSULOT QOIDASI: «Dars rejalashtirish» (Jadval bo'limi) — to'liq forma:
//  · boshlanish vaqti BU YERDA majburiy;
//  · yaratilgach xonaga KIRMAYDI — dars jadvalda ko'rinadi.

vi.mock('../api/lessons', () => ({
  listLessons: vi.fn(),
  getLesson: vi.fn(),
  createLesson: vi.fn(),
  updateLesson: vi.fn(),
  deleteLesson: vi.fn(),
  getHostToken: vi.fn(),
  endLesson: vi.fn(),
}))

import { createLesson } from '../api/lessons'

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="location">{loc.pathname}</div>
}

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/app/schedule']}>
        <ScheduleLessonModal open onClose={onClose} />
        <Routes>
          <Route path="*" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return { onClose }
}

beforeEach(() => {
  vi.clearAllMocks()
  createLesson.mockResolvedValue({ id: 'l88', title: 'Test', status: 'scheduled' })
})

describe('ScheduleLessonModal (rejalashtirish)', () => {
  it('boshlanish vaqti maydoni bor va MAJBURIY', () => {
    setup()
    const dt = document.querySelector('input[type="datetime-local"]')
    expect(dt).not.toBeNull()
    expect(dt).toBeRequired()
  })

  it('scheduled_at ISO formatda yuboriladi, xonaga O‘TMAYDI', async () => {
    const user = userEvent.setup()
    const { onClose } = setup()
    await user.type(screen.getByPlaceholderText('Masalan: Kvadrat tenglamalar'), 'Geometriya')
    const dt = document.querySelector('input[type="datetime-local"]')
    await user.type(dt, '2026-08-01T10:00')
    await user.click(screen.getByRole('button', { name: /^rejalashtirish$/i }))

    await waitFor(() => expect(createLesson).toHaveBeenCalledTimes(1))
    const body = createLesson.mock.calls[0][0]
    expect(body.title).toBe('Geometriya')
    expect(body.scheduled_at).toBe(new Date('2026-08-01T10:00').toISOString())
    expect(body.duration_min).toBe(60)
    expect(body.is_recording_enabled).toBe(true)
    expect(body.is_waiting_room_enabled).toBe(true)

    // Xonaga navigatsiya YO'Q — jadval sahifasida qoladi
    expect(screen.getByTestId('location')).toHaveTextContent('/app/schedule')
    expect(onClose).toHaveBeenCalled()
  })

  it('xato bo‘lsa modal yopilmaydi', async () => {
    createLesson.mockRejectedValue(new Error('server down'))
    const user = userEvent.setup()
    const { onClose } = setup()
    await user.type(screen.getByPlaceholderText('Masalan: Kvadrat tenglamalar'), 'Geometriya')
    await user.type(document.querySelector('input[type="datetime-local"]'), '2026-08-01T10:00')
    await user.click(screen.getByRole('button', { name: /^rejalashtirish$/i }))

    await waitFor(() => expect(createLesson).toHaveBeenCalled())
    expect(onClose).not.toHaveBeenCalled()
  })
})
