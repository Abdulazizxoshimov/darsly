import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import { CreateLessonModal } from './CreateLessonModal'

// MAHSULOT QOIDASI: «Dars yaratish» = TEZKOR dars (Zoom "New meeting" kabi).
//  · formada tavsif va boshlanish vaqti YO'Q;
//  · so'rov tanasida scheduled_at/description/duration_min YUBORILMAYDI;
//  · yozib olish DEFAULT YONIQ;
//  · muvaffaqiyatdan so'ng ustoz DARHOL xonaga o'tadi (Boshlash oqimi bilan bir xil).

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
      <MemoryRouter initialEntries={['/app']}>
        <CreateLessonModal open onClose={onClose} />
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
  createLesson.mockResolvedValue({ id: 'l77', title: 'Test', status: 'scheduled' })
})

describe('CreateLessonModal (tezkor dars)', () => {
  it('formada tavsif va vaqt maydonlari YO‘Q — faqat nom, parol, toggle’lar', () => {
    setup()
    expect(screen.getByPlaceholderText('Masalan: Kvadrat tenglamalar')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('4–20 belgi')).toBeInTheDocument()
    // Tavsif ham, datetime ham bo'lmasligi kerak
    expect(screen.queryByPlaceholderText('Dars haqida qisqacha')).not.toBeInTheDocument()
    expect(document.querySelector('input[type="datetime-local"]')).toBeNull()
  })

  it('yozib olish va kutish xonasi DEFAULT YONIQ', () => {
    setup()
    const switches = screen.getAllByRole('switch')
    expect(switches).toHaveLength(2)
    for (const s of switches) expect(s).toHaveAttribute('aria-checked', 'true')
  })

  it('scheduled_at YUBORILMAYDI, bo‘sh parol ham yuborilmaydi', async () => {
    const user = userEvent.setup()
    setup()
    await user.type(screen.getByPlaceholderText('Masalan: Kvadrat tenglamalar'), 'Algebra')
    await user.click(screen.getByRole('button', { name: /yaratish va boshlash/i }))

    await waitFor(() => expect(createLesson).toHaveBeenCalledTimes(1))
    const body = createLesson.mock.calls[0][0]
    expect(body.title).toBe('Algebra')
    expect(body.is_recording_enabled).toBe(true)
    expect(body.is_waiting_room_enabled).toBe(true)
    expect(body).not.toHaveProperty('scheduled_at')
    expect(body).not.toHaveProperty('description')
    expect(body).not.toHaveProperty('duration_min')
    expect(body.passcode).toBeUndefined()
  })

  it('muvaffaqiyatdan so‘ng DARHOL xonaga o‘tadi', async () => {
    const user = userEvent.setup()
    const { onClose } = setup()
    await user.type(screen.getByPlaceholderText('Masalan: Kvadrat tenglamalar'), 'Algebra')
    await user.click(screen.getByRole('button', { name: /yaratish va boshlash/i }))

    await waitFor(() =>
      expect(screen.getByTestId('location')).toHaveTextContent('/app/lesson/l77/room'),
    )
    expect(onClose).toHaveBeenCalled()
  })

  it('toggle o‘chirilsa qiymat false ketadi, parol kiritilsa yuboriladi', async () => {
    const user = userEvent.setup()
    setup()
    await user.type(screen.getByPlaceholderText('Masalan: Kvadrat tenglamalar'), 'Algebra')
    await user.type(screen.getByPlaceholderText('4–20 belgi'), 'sirli12')
    // Birinchi switch — yozib olish
    await user.click(screen.getAllByRole('switch')[0])
    await user.click(screen.getByRole('button', { name: /yaratish va boshlash/i }))

    await waitFor(() => expect(createLesson).toHaveBeenCalledTimes(1))
    const body = createLesson.mock.calls[0][0]
    expect(body.is_recording_enabled).toBe(false)
    expect(body.passcode).toBe('sirli12')
  })

  it('xato bo‘lsa xonaga O‘TMAYDI va modal ochiq qoladi', async () => {
    createLesson.mockRejectedValue(new Error('server down'))
    const user = userEvent.setup()
    const { onClose } = setup()
    await user.type(screen.getByPlaceholderText('Masalan: Kvadrat tenglamalar'), 'Algebra')
    await user.click(screen.getByRole('button', { name: /yaratish va boshlash/i }))

    await waitFor(() => expect(createLesson).toHaveBeenCalled())
    expect(screen.getByTestId('location')).toHaveTextContent('/app')
    expect(onClose).not.toHaveBeenCalled()
  })
})
