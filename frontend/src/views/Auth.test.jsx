import { describe, expect, it, vi, beforeEach } from 'vitest'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { Auth } from './Auth'
import { setLogoutReason } from '../lib/logoutReason'

// MAHSULOT QOIDASI: ochiq ro'yxatdan o'tish server bayrog'i bilan boshqariladi.
//  · `allow_open_registration=false` → "Ro'yxatdan o'tish" UMUMAN ko'rinmaydi,
//    o'rniga "Administrator bilan bog'laning" matni;
//  · `true` → tab'lar ko'rinadi va register oqimi ishlaydi;
//  · bayroq kelmagan/xato bo'lsa ham xavfsiz tomonga (yashirish) og'adi.

const doLogin = vi.fn()
const doRegister = vi.fn()
vi.mock('../store/app', () => ({
  useApp: () => ({ doLogin, doRegister }),
}))

vi.mock('../api/app', () => ({
  getAppConfig: vi.fn(),
}))

import { getAppConfig } from '../api/app'

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/auth']}>
        <Auth />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  sessionStorage.clear()
  doLogin.mockResolvedValue({ id: 'u1' })
  doRegister.mockResolvedValue({ id: 'u1' })
})

describe('Auth — ochiq registratsiya bayrog‘i', () => {
  it('allow_open_registration=false → register tab YO‘Q, admin matni BOR', async () => {
    getAppConfig.mockResolvedValue({ allow_open_registration: false })
    setup()
    await waitFor(() => expect(getAppConfig).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /ro'yxatdan o'tish/i })).not.toBeInTheDocument()
    expect(screen.getByText(/administrator bilan bog'laning/i)).toBeInTheDocument()
  })

  it('app-config XATO bo‘lsa ham register ko‘rsatilmaydi (xavfsiz default)', async () => {
    getAppConfig.mockRejectedValue(new Error('down'))
    setup()
    await waitFor(() => expect(getAppConfig).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /ro'yxatdan o'tish/i })).not.toBeInTheDocument()
    // Kirish formasi baribir ishlaydi
    expect(screen.getByPlaceholderText('email@misol.uz')).toBeInTheDocument()
  })

  it('allow_open_registration=true → tab ko‘rinadi va register oqimi ishlaydi', async () => {
    getAppConfig.mockResolvedValue({ allow_open_registration: true })
    const user = userEvent.setup()
    setup()
    const tab = await screen.findByRole('button', { name: /ro'yxatdan o'tish/i })
    await user.click(tab)
    await user.type(screen.getByPlaceholderText('Ism Familiya'), 'Test Ustoz')
    await user.type(screen.getByPlaceholderText('email@misol.uz'), 'test@darsly.uz')
    await user.type(screen.getByPlaceholderText('••••••••'), 'parol12345')
    // Formadagi submit tugmasi ham "Ro'yxatdan o'tish" — oxirgisini olamiz
    const submitButtons = screen.getAllByRole('button', { name: /ro'yxatdan o'tish/i })
    await user.click(submitButtons[submitButtons.length - 1])
    await waitFor(() =>
      expect(doRegister).toHaveBeenCalledWith('Test Ustoz', 'test@darsly.uz', 'parol12345'),
    )
    expect(doLogin).not.toHaveBeenCalled()
  })

  it('sessiya BEKOR qilinganda sabab ko‘rsatiladi («boshqa qurilmada kirildi»)', async () => {
    // MAHSULOT QOIDASI: bitta akkaunt = bitta faol sessiya. `SESSION_REVOKED`
    // ni «sessiya tugadi» deb ko'rsatish adashtiradi — foydalanuvchi internetni
    // yoki parolni ayblab, akkaunti ulashilganini bilmay qoladi.
    setLogoutReason('session_revoked')
    getAppConfig.mockResolvedValue({ allow_open_registration: false })
    setup()
    expect(screen.getByRole('alert')).toHaveTextContent(/boshqa qurilmada kirildi/i)
  })

  it('sabab BIR MARTA ko‘rsatiladi — sahifa yangilansa qaytmaydi', async () => {
    setLogoutReason('session_revoked')
    getAppConfig.mockResolvedValue({ allow_open_registration: false })
    setup()
    expect(screen.getByRole('alert')).toBeInTheDocument()
    cleanup()
    setup()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('sababsiz kirishda banner umuman chiqmaydi', async () => {
    getAppConfig.mockResolvedValue({ allow_open_registration: false })
    setup()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('login oqimi doLogin ni chaqiradi', async () => {
    getAppConfig.mockResolvedValue({ allow_open_registration: false })
    const user = userEvent.setup()
    setup()
    await user.type(screen.getByPlaceholderText('email@misol.uz'), 'mentor@darsly.uz')
    await user.type(screen.getByPlaceholderText('••••••••'), 'parol12345')
    await user.click(screen.getByRole('button', { name: /kirish/i }))
    await waitFor(() => expect(doLogin).toHaveBeenCalledWith('mentor@darsly.uz', 'parol12345'))
    expect(doRegister).not.toHaveBeenCalled()
  })
})
