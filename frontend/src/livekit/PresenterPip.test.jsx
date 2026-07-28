import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PresenterPanel } from './PresenterPip'

// Suzuvchi oyna paneli — ustoz ekran ulashganda ko'radigan YAGONA oyna.
// Shuning uchun undagi har bir amal ishlashi kritik: agar "to'xtatish" tugmasi
// jim qolsa, ustoz ulashishni to'xtatish uchun boshqa oynani qidirishga majbur.

function setup(over = {}) {
  const onToggleMic = vi.fn()
  const onStopShare = vi.fn()
  const onLowerHand = vi.fn()
  const onSendChat = vi.fn()
  const props = {
    hands: [],
    reactions: [],
    chat: [],
    micOn: true,
    onToggleMic,
    onStopShare,
    onLowerHand,
    onSendChat,
    ...over,
  }
  render(<PresenterPanel {...props} />)
  return { onToggleMic, onStopShare, onLowerHand, onSendChat }
}

describe('PresenterPanel (suzuvchi oyna)', () => {
  it('qo‘l navbati raqamlangan holda chiqadi', () => {
    setup({
      hands: [
        { identity: 'u1', name: 'Ali' },
        { identity: 'u2', name: 'Vali' },
      ],
    })
    expect(screen.getByText("Qo'l ko'targanlar (2)")).toBeInTheDocument()
    expect(screen.getByText('Ali')).toBeInTheDocument()
    expect(screen.getByText('Vali')).toBeInTheDocument()
  })

  it('qo‘l bo‘lmasa ham bo‘lim tushuntirish bilan qoladi', () => {
    // Bo'sh ro'yxat "hech kim so'ramadi" degan MA'LUMOT — uni yashirish
    // ustozni "ishlayaptimi?" degan shubhaga qoldirardi.
    setup()
    expect(screen.getByText("Hozircha yo'q")).toBeInTheDocument()
  })

  it('qo‘lni tushirish identity bilan chaqiriladi', async () => {
    const user = userEvent.setup()
    const { onLowerHand } = setup({ hands: [{ identity: 'u1', name: 'Ali' }] })
    await user.click(screen.getByTitle('Tushirish'))
    expect(onLowerHand).toHaveBeenCalledWith('u1')
  })

  it('chat xabarlari ko‘rinadi va shaxsiysi belgilanadi', () => {
    setup({
      chat: [
        { id: 'm1', name: 'Ali', body: 'Savolim bor', self: false, toIdentity: null },
        { id: 'm2', name: 'Ustoz', body: 'Javob', self: true, toIdentity: 'u1' },
      ],
    })
    expect(screen.getByText('Savolim bor')).toBeInTheDocument()
    expect(screen.getByText('Javob')).toBeInTheDocument()
    expect(screen.getByText('shaxsiy')).toBeInTheDocument()
  })

  it('suzuvchi oynadan javob yuborish mumkin', async () => {
    const user = userEvent.setup()
    const { onSendChat } = setup()

    await user.type(screen.getByPlaceholderText('Javob yozing…'), 'Ha, eshityapman')
    await user.click(screen.getByLabelText('Yuborish'))

    expect(onSendChat).toHaveBeenCalledWith('Ha, eshityapman')
  })

  it('bo‘sh javob yuborilmaydi', async () => {
    const user = userEvent.setup()
    const { onSendChat } = setup()
    await user.click(screen.getByLabelText('Yuborish'))
    expect(onSendChat).not.toHaveBeenCalled()
  })

  it('mikrofon holati tugmada rost ko‘rsatiladi', async () => {
    const user = userEvent.setup()
    const { onToggleMic } = setup({ micOn: false })
    // Ovozsiz bo'lsa tugma "Ovozsiz" deydi — ustoz gapirib turib
    // "nega meni eshitishmayapti" degan holatga tushmasligi uchun.
    const btn = screen.getByText('Ovozsiz')
    await user.click(btn)
    expect(onToggleMic).toHaveBeenCalled()
  })

  it('ulashishni to‘xtatish tugmasi ishlaydi', async () => {
    const user = userEvent.setup()
    const { onStopShare } = setup()
    await user.click(screen.getByText("To'xtatish"))
    expect(onStopShare).toHaveBeenCalled()
  })

  it('reaksiya bo‘lmasa bo‘lim chizilmaydi (kichik oynada joy tejaladi)', () => {
    setup()
    expect(screen.queryByText(/👍/)).not.toBeInTheDocument()
  })

  it('reaksiyalar kim yuborgani bilan chiqadi', () => {
    setup({ reactions: [{ id: 1, emoji: '👍', name: 'Ali' }] })
    expect(screen.getByText('Ali')).toBeInTheDocument()
  })
})
