import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TelegramSection } from './TelegramSection'

vi.mock('../api/telegram', () => ({
  getTelegramStatus: vi.fn(),
  startTelegramLink: vi.fn(),
  unlinkTelegram: vi.fn(),
  adaptTelegramStatus: (x) => x,
}))

import { getTelegramStatus, startTelegramLink, unlinkTelegram } from '../api/telegram'

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <TelegramSection />
    </QueryClientProvider>,
  )
  return qc
}

const STATUS = (over) => ({ enabled: true, linked: false, username: '', linked_at: null, chats: [], ...over })

beforeEach(() => vi.clearAllMocks())

describe('Telegram bo‘limi', () => {
  it('server integratsiyasi sozlanmagan bo‘lsa bo‘lim UMUMAN ko‘rinmaydi', async () => {
    getTelegramStatus.mockResolvedValue(STATUS({ enabled: false }))
    setup()
    // Kutamiz — so'rov tugagach ham hech nima chizilmasligi kerak.
    await vi.waitFor(() => expect(getTelegramStatus).toHaveBeenCalled())
    expect(screen.queryByText('Telegram')).not.toBeInTheDocument()
  })

  it('holat so‘rovi yiqilsa profil sahifasi buzilmaydi (bo‘lim jim qoladi)', async () => {
    getTelegramStatus.mockRejectedValue(new Error('network'))
    setup()
    await vi.waitFor(() => expect(getTelegramStatus).toHaveBeenCalled())
    expect(screen.queryByText('Telegram')).not.toBeInTheDocument()
  })

  it('bog‘lanmagan holatda kod so‘raladi va `/start <kod>` ko‘rsatiladi', async () => {
    const user = userEvent.setup()
    getTelegramStatus.mockResolvedValue(STATUS())
    startTelegramLink.mockResolvedValue({
      code: 'JN7K2Q',
      deep_link: 'https://t.me/jonly_bot?start=JN7K2Q',
      expires_in_s: 900,
    })
    setup()

    await user.click(await screen.findByRole('button', { name: /Telegram bilan bog‘lash/ }))
    expect(await screen.findByText('/start JN7K2Q')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Telegramda ochish/ })).toHaveAttribute(
      'href',
      'https://t.me/jonly_bot?start=JN7K2Q',
    )
  })

  it('bot username noma’lum bo‘lsa faqat kod ko‘rsatiladi (havola yo‘q)', async () => {
    const user = userEvent.setup()
    getTelegramStatus.mockResolvedValue(STATUS())
    startTelegramLink.mockResolvedValue({ code: 'ABC123', deep_link: '', expires_in_s: 900 })
    setup()

    await user.click(await screen.findByRole('button', { name: /Telegram bilan bog‘lash/ }))
    expect(await screen.findByText('/start ABC123')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Telegramda ochish/ })).not.toBeInTheDocument()
  })

  it('bog‘langan holatda username va «Uzish» ko‘rinadi', async () => {
    const user = userEvent.setup()
    getTelegramStatus.mockResolvedValue(
      STATUS({
        linked: true,
        username: 'aziz_mentor',
        chats: [{ chat_id: -1001, title: 'Video darslar', type: 'group', is_active: true }],
      }),
    )
    unlinkTelegram.mockResolvedValue(null)
    setup()

    expect(await screen.findByText('@aziz_mentor')).toBeInTheDocument()
    expect(screen.getByText('Video darslar')).toBeInTheDocument()

    // Uzish — tasdiqdan keyin (tasodifiy bosish butun zanjirni buzmasin).
    await user.click(screen.getByRole('button', { name: /Uzish/ }))
    expect(screen.getByText('Telegramni uzasizmi?')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Ha, uzish/ }))
    await vi.waitFor(() => expect(unlinkTelegram).toHaveBeenCalled())
  })
})

describe('Telegram — kod hayotiy sikli', () => {
  it('bog‘lanish tugagach kod kartochkasi o‘rnini «Bog‘langan» egallaydi', async () => {
    const user = userEvent.setup()
    getTelegramStatus.mockResolvedValue(STATUS())
    startTelegramLink.mockResolvedValue({ code: 'JN7K2Q', deep_link: '', expires_in_s: 900 })
    setup()

    await user.click(await screen.findByRole('button', { name: /Telegram bilan bog‘lash/ }))
    expect(await screen.findByText('/start JN7K2Q')).toBeInTheDocument()

    // Mentor botga /start yubordi — keyingi poll `linked` deydi.
    getTelegramStatus.mockResolvedValue(STATUS({ linked: true, username: 'aziz_mentor' }))
    expect(await screen.findByText('@aziz_mentor', {}, { timeout: 8000 })).toBeInTheDocument()
    expect(screen.queryByText('/start JN7K2Q')).not.toBeInTheDocument()
  }, 12_000)

  it('kod muddati tugaganda ekrandan olinadi (o‘lik kod qolmasin)', async () => {
    const user = userEvent.setup()
    getTelegramStatus.mockResolvedValue(STATUS())
    // Muddat ataylab juda qisqa — mantiq server bergan qiymatga tayanadi.
    startTelegramLink.mockResolvedValue({ code: 'SHORT1', deep_link: '', expires_in_s: 0.05 })
    setup()

    await user.click(await screen.findByRole('button', { name: /Telegram bilan bog‘lash/ }))
    expect(await screen.findByText('/start SHORT1')).toBeInTheDocument()

    await vi.waitFor(() => expect(screen.queryByText('/start SHORT1')).not.toBeInTheDocument())
    expect(screen.getByRole('button', { name: /Telegram bilan bog‘lash/ })).toBeInTheDocument()
  })

  it('kod muddati SERVER qiymatidan chiqadi (qattiq kodlangan «15 daqiqa» emas)', async () => {
    const user = userEvent.setup()
    getTelegramStatus.mockResolvedValue(STATUS())
    startTelegramLink.mockResolvedValue({ code: 'ABC999', deep_link: '', expires_in_s: 300 })
    setup()

    await user.click(await screen.findByRole('button', { name: /Telegram bilan bog‘lash/ }))
    expect(await screen.findByText(/Kod 5 daqiqa amal qiladi/)).toBeInTheDocument()
  })
})
