import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { UsersView } from './Users'

// MAHSULOT QOIDALARI (admin bo'limi):
//  · faqat `admin` roli ko'radi — boshqa rol URL tersa /app ga qaytadi
//    (backend RBAC baribir 403 beradi, bu UI darajasidagi himoya);
//  · ro'yxat backend `UserShort` maydonlaridan chiqadi (rol ro'yxatda YO'Q);
//  · admin o'zini o'chira olmaydi;
//  · yangi hisob backend `CreateUserReq` shaklida ketadi.

let mockUser = { id: 'u1', full_name: 'Admin Aka', role: 'admin', color: '#19d3a2' }
vi.mock('../store/app', () => ({
  useApp: () => ({ user: mockUser }),
}))

vi.mock('../api/user', () => ({
  updateProfile: vi.fn(),
  changePassword: vi.fn(),
  listUsers: vi.fn(),
  createUser: vi.fn(),
  deleteUser: vi.fn(),
}))

import { listUsers, createUser, deleteUser } from '../api/user'

const USERS = {
  data: [
    { id: 'u1', full_name: 'Admin Aka', email: 'admin@darsly.uz', color: '#19d3a2' },
    { id: 'u2', full_name: 'Malika Yusupova', email: 'malika@darsly.uz', color: '#6c5ce7' },
  ],
  total: 2,
  page: 1,
  limit: 100,
  total_pages: 1,
}

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/app/users']}>
        <Routes>
          <Route path="/app/users" element={<UsersView />} />
          <Route path="/app" element={<div data-testid="dashboard">dashboard</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockUser = { id: 'u1', full_name: 'Admin Aka', role: 'admin', color: '#19d3a2' }
  listUsers.mockResolvedValue(USERS)
  createUser.mockResolvedValue({ id: 'u3', full_name: 'Yangi Ustoz', email: 'y@darsly.uz' })
  deleteUser.mockResolvedValue(null)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('UsersView (admin)', () => {
  it('admin ro‘yxatni jadvalda ko‘radi', async () => {
    setup()
    expect(await screen.findByText('Malika Yusupova')).toBeInTheDocument()
    expect(screen.getByText('malika@darsly.uz')).toBeInTheDocument()
    expect(screen.getByText('(siz)')).toBeInTheDocument()
  })

  it('admin BO‘LMAGAN rol /app ga qaytariladi va so‘rov ketmaydi', async () => {
    mockUser = { id: 'u9', full_name: 'Oddiy Ustoz', role: 'mentor' }
    setup()
    expect(await screen.findByTestId('dashboard')).toBeInTheDocument()
    expect(listUsers).not.toHaveBeenCalled()
  })

  it('yangi foydalanuvchi CreateUserReq shaklida yaratiladi', async () => {
    const user = userEvent.setup()
    setup()
    await user.click(await screen.findByRole('button', { name: /foydalanuvchi qo'shish/i }))
    await user.type(screen.getByPlaceholderText('Ism Familiya'), 'Yangi Ustoz')
    await user.type(screen.getByPlaceholderText('email@misol.uz'), 'yangi@darsly.uz')
    await user.type(screen.getByPlaceholderText('Kamida 8 belgi'), 'parol12345')
    await user.click(screen.getByRole('button', { name: /^yaratish$/i }))
    // TanStack v5 mutationFn'ga ikkinchi (kontekst) argument ham uzatadi —
    // shuning uchun faqat birinchi argument tekshiriladi.
    await waitFor(() => expect(createUser).toHaveBeenCalledTimes(1))
    expect(createUser.mock.calls[0][0]).toEqual({
      full_name: 'Yangi Ustoz',
      email: 'yangi@darsly.uz',
      password: 'parol12345',
      role: 'mentor',
    })
  })

  it('o‘chirish tasdiq bilan deleteUser ni chaqiradi', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const user = userEvent.setup()
    setup()
    await screen.findByText('Malika Yusupova')
    await user.click(screen.getByRole('button', { name: /malika yusupovani o'chirish/i }))
    await waitFor(() => expect(deleteUser).toHaveBeenCalledTimes(1))
    expect(deleteUser.mock.calls[0][0]).toBe('u2')
  })

  it('tasdiq RAD etilsa o‘chirilmaydi', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    const user = userEvent.setup()
    setup()
    await screen.findByText('Malika Yusupova')
    await user.click(screen.getByRole('button', { name: /malika yusupovani o'chirish/i }))
    expect(deleteUser).not.toHaveBeenCalled()
  })

  it('admin O‘ZINI o‘chira olmaydi (tugma o‘chirilgan)', async () => {
    setup()
    await screen.findByText('(siz)')
    expect(screen.getByRole('button', { name: /admin akani o'chirish/i })).toBeDisabled()
  })
})
