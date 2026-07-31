import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ChatHistoryModal } from './ChatHistoryModal'

// MAHSULOT QOIDASI: chat DB'da saqlanadi va dars TUGAGANDAN keyin ham
// o'qilishi kerak (JWT yo'li: GET /lessons/:id/chat).
//  · xabarlar XRONOLOGIK tartibda (server kursori yangi→eski bersa ham);
//  · shaxsiy xabar belgilanadi;
//  · bo'sh/xato holatlar tushunarli matn bilan.

vi.mock('../api/chat', () => ({
  chatHistory: vi.fn(),
  sendChat: vi.fn(),
  roomChatHistory: vi.fn(),
  roomChatSend: vi.fn(),
  roomChatUpload: vi.fn(),
  chatUpload: vi.fn(),
  deleteChatMessage: vi.fn(),
}))

import { chatHistory, deleteChatMessage } from '../api/chat'

const LESSON = { id: 'l1', title: 'Kvadrat tenglamalar' }

const MESSAGES = [
  // Ataylab teskari (yangi → eski) — komponent xronologikka tartiblashi shart.
  {
    id: 'c2',
    lesson_id: 'l1',
    sender_identity: 'guest_1',
    sender_name: 'Ali Valiyev',
    body: 'Savolim bor edi',
    to_identity: null,
    created_at: '2026-07-30T10:05:00Z',
  },
  {
    id: 'c1',
    lesson_id: 'l1',
    sender_identity: 'host',
    sender_name: 'Aziz Karimov',
    body: 'Darsga xush kelibsiz!',
    to_identity: null,
    created_at: '2026-07-30T10:00:00Z',
  },
  {
    id: 'c3',
    lesson_id: 'l1',
    sender_identity: 'host',
    sender_name: 'Aziz Karimov',
    body: 'Faqat sizga javob',
    to_identity: 'guest_1',
    created_at: '2026-07-30T10:06:00Z',
  },
]

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={qc}>
      <ChatHistoryModal lesson={LESSON} onClose={onClose} />
    </QueryClientProvider>,
  )
  return { onClose }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('ChatHistoryModal', () => {
  it('xabarlar yuboruvchi nomi, vaqti va matni bilan ko‘rinadi', async () => {
    chatHistory.mockResolvedValue(MESSAGES)
    setup()
    expect(await screen.findByText('Darsga xush kelibsiz!')).toBeInTheDocument()
    expect(screen.getByText('Savolim bor edi')).toBeInTheDocument()
    expect(screen.getAllByText('Aziz Karimov').length).toBeGreaterThan(0)
    expect(screen.getByText('Ali Valiyev')).toBeInTheDocument()
    // Dars nomi ham modal sarlavhasi ostida
    expect(screen.getByText('Kvadrat tenglamalar')).toBeInTheDocument()
  })

  it('xabarlar XRONOLOGIK tartibda chiqadi', async () => {
    chatHistory.mockResolvedValue(MESSAGES)
    setup()
    await screen.findByText('Darsga xush kelibsiz!')
    const bodies = [...document.querySelectorAll('.chat-bubble')].map((n) => n.textContent)
    expect(bodies).toEqual(['Darsga xush kelibsiz!', 'Savolim bor edi', 'Faqat sizga javob'])
  })

  it('shaxsiy xabar "shaxsiy" belgisi bilan ajralib turadi', async () => {
    chatHistory.mockResolvedValue(MESSAGES)
    setup()
    await screen.findByText('Faqat sizga javob')
    expect(screen.getByText('shaxsiy')).toBeInTheDocument()
  })

  it('bo‘sh chat: "Bu darsda chat yozilmagan"', async () => {
    chatHistory.mockResolvedValue([])
    setup()
    expect(await screen.findByText('Bu darsda chat yozilmagan')).toBeInTheDocument()
  })

  it('xato holati tushunarli matn bilan', async () => {
    chatHistory.mockRejectedValue(new Error('down'))
    setup()
    expect(await screen.findByText(/chat tarixini yuklab bo'lmadi/i)).toBeInTheDocument()
  })

  // Moderatsiya dars TUGAGANDAN keyin ham kerak: buzg'unchi xabar ko'pincha
  // aynan shunda ko'zga tashlanadi, xonaga esa qaytib bo'lmaydi.
  it('xabar tasdiqdan keyin o‘chiriladi va ro‘yxatdan yo‘qoladi', async () => {
    const user = userEvent.setup()
    chatHistory.mockResolvedValue(MESSAGES)
    deleteChatMessage.mockResolvedValue(null)
    setup()
    await screen.findByText('Savolim bor edi')

    // Har xabar yonida o'chirish tugmasi (ustoz oynasi).
    const buttons = screen.getAllByLabelText("Xabarni o'chirish")
    await user.click(buttons[1]) // "Savolim bor edi"

    expect(deleteChatMessage).not.toHaveBeenCalled() // tasdiqsiz — yo'q
    await user.click(screen.getByText("Ha, o'chirish"))

    await waitFor(() => expect(deleteChatMessage).toHaveBeenCalledWith('l1', 'c2'))
    await waitFor(() => expect(screen.queryByText('Savolim bor edi')).not.toBeInTheDocument())
  })

  it('o‘chirishni bekor qilish API chaqirmaydi', async () => {
    const user = userEvent.setup()
    chatHistory.mockResolvedValue(MESSAGES)
    setup()
    await screen.findByText('Savolim bor edi')

    await user.click(screen.getAllByLabelText("Xabarni o'chirish")[0])
    await user.click(screen.getByText('Bekor qilish'))
    expect(deleteChatMessage).not.toHaveBeenCalled()
    expect(screen.getByText('Savolim bor edi')).toBeInTheDocument()
  })
})
