import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { AppShell } from './AppShell'

// MAHSULOT QOIDASI: "Foydalanuvchilar" bo'limi sidebar'da FAQAT admin'ga
// ko'rinadi. Mobil pastki tab yo'q — navigatsiya doimiy chap sidebar'da.

let mockUser = { id: 'u1', full_name: 'Aziz Karimov', role: 'mentor', color: '#19d3a2' }
vi.mock('../store/app', () => ({
  useApp: () => ({ user: mockUser, doLogout: vi.fn() }),
}))

vi.mock('../store/data', () => ({
  useUnreadCount: () => ({ data: 3 }),
}))

function setup() {
  render(
    <MemoryRouter initialEntries={['/app']}>
      <Routes>
        <Route path="/app" element={<AppShell />}>
          <Route index element={<div>ichki sahifa</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  mockUser = { id: 'u1', full_name: 'Aziz Karimov', role: 'mentor', color: '#19d3a2' }
})

describe('AppShell', () => {
  it('mentor: asosiy bo‘limlar bor, "Foydalanuvchilar" YO‘Q', () => {
    setup()
    // "Darslar" topbar sarlavhasida ham bor — shuning uchun link roli bilan.
    expect(screen.getByRole('link', { name: 'Darslar' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Jadval' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Yozuvlar' })).toBeInTheDocument()
    expect(screen.queryByText('Foydalanuvchilar')).not.toBeInTheDocument()
  })

  it('admin: "Foydalanuvchilar" bo‘limi ko‘rinadi', () => {
    mockUser = { ...mockUser, role: 'admin' }
    setup()
    expect(screen.getByText('Foydalanuvchilar')).toBeInTheDocument()
  })

  it('sidebar pastida foydalanuvchi kartasi (ism + rol)', () => {
    setup()
    expect(screen.getByText('Aziz Karimov')).toBeInTheDocument()
    expect(screen.getByText('Ustoz')).toBeInTheDocument()
  })

  it('o‘qilmagan bildirishnoma soni qo‘ng‘iroqda ko‘rinadi', () => {
    setup()
    expect(screen.getByText('3')).toBeInTheDocument()
  })
})
