import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { badgeText, PIP_COMPACT, PresenterPanel, pipWindowSize } from './PresenterPip'

// Suzuvchi oyna paneli — ustoz ekran ulashganda ko'radigan YAGONA oyna.
// Shuning uchun undagi har bir amal ishlashi kritik: agar "to'xtatish" tugmasi
// jim qolsa, ustoz ulashishni to'xtatish uchun boshqa oynani qidirishga majbur.
//
// Bu yerdagi asosiy shartnoma (asoschi talabi): chat va qo'l — IKKI MUSTAQIL
// signal. Ular bir-birining ichida ham, bir-birining sanog'ida ham emas.

function setup(over = {}) {
  const onToggleMic = vi.fn()
  const onStopShare = vi.fn()
  const onLowerHand = vi.fn()
  const onSendChat = vi.fn()
  const onToggleChat = vi.fn()
  const onToggleHands = vi.fn()
  const props = {
    hands: [],
    reactions: [],
    chat: [],
    unread: 0,
    chatOpen: false,
    handsOpen: false,
    micOn: true,
    onToggleChat,
    onToggleHands,
    onToggleMic,
    onStopShare,
    onLowerHand,
    onSendChat,
    ...over,
  }
  const utils = render(<PresenterPanel {...props} />)
  return { ...utils, onToggleMic, onStopShare, onLowerHand, onSendChat, onToggleChat, onToggleHands }
}

describe('badgeText', () => {
  it('sanoqni raqam bilan ko‘rsatadi (nuqta emas)', () => {
    // "Xabar bor" bilan "7 ta xabar bor" — ustoz uchun ikki xil qaror.
    expect(badgeText(1)).toBe('1')
    expect(badgeText(9)).toBe('9')
  })

  it('9 dan ortig‘i «9+» ga qisqaradi', () => {
    // Uch xonali son kichik tugmani buzardi; bu yerda "ko'p" signali yetarli.
    expect(badgeText(10)).toBe('9+')
    expect(badgeText(342)).toBe('9+')
  })
})

describe('pipWindowSize — oyna o‘lchami', () => {
  const screenSize = { availWidth: 1920, availHeight: 1080 }

  it('sukut holat KOMPAKT — ish ekranini deyarli to‘smaydi', () => {
    expect(pipWindowSize({ screen: screenSize })).toEqual(PIP_COMPACT)
    expect(PIP_COMPACT.height).toBeLessThan(120)
  })

  it('chat ochilganda ~1/4 ekran eniga kengayadi', () => {
    const s = pipWindowSize({ chatOpen: true, screen: screenSize })
    expect(s.width).toBe(420) // 1920/4 = 480 → yuqori chegara 420
    expect(s.height).toBeGreaterThan(PIP_COMPACT.height * 3)
  })

  it('kichik ekranda ham o‘qish mumkin bo‘lgan minimal en saqlanadi', () => {
    const s = pipWindowSize({ chatOpen: true, screen: { availWidth: 1024, availHeight: 640 } })
    expect(s.width).toBe(320) // 256 → pastki chegara 320
    expect(s.height).toBe(397) // 640*0.62 ≈ 397 (ekran bo'yidan chiqib ketmaydi)
  })

  it('qo‘l ro‘yxati faqat BO‘YIGA o‘sadi va 5 qatorda to‘xtaydi', () => {
    const one = pipWindowSize({ handsOpen: true, handCount: 1, screen: screenSize })
    const five = pipWindowSize({ handsOpen: true, handCount: 5, screen: screenSize })
    const many = pipWindowSize({ handsOpen: true, handCount: 40, screen: screenSize })
    expect(one.width).toBe(PIP_COMPACT.width)
    expect(one.height).toBeGreaterThan(PIP_COMPACT.height)
    expect(five.height).toBeGreaterThan(one.height)
    expect(many.height).toBe(five.height) // ortig'i scroll bilan
  })

  it('chat ochiq bo‘lsa qo‘l ro‘yxati o‘lchamni belgilamaydi', () => {
    const s = pipWindowSize({ chatOpen: true, handsOpen: true, handCount: 9, screen: screenSize })
    expect(s).toEqual(pipWindowSize({ chatOpen: true, screen: screenSize }))
  })
})

describe('PresenterPanel — kompakt overlay', () => {
  it('sukut holatda chat xabarlari ham qo‘l ro‘yxati ham chizilmaydi', () => {
    // Kompakt oyna — faqat signal. Ikkalasi ochilmasa joy egallamaydi.
    setup({
      chat: [{ id: 'm1', name: 'Ali', body: 'Savolim bor', self: false, toIdentity: null }],
      hands: [{ identity: 'u1', name: 'Ali' }],
    })
    expect(screen.queryByText('Savolim bor')).not.toBeInTheDocument()
    expect(screen.queryByPlaceholderText('Javob yozing…')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Tushirish')).not.toBeInTheDocument()
  })

  it('o‘qilmagan xabar sanog‘i chat tugmasida chiqadi', () => {
    setup({ unread: 3 })
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByLabelText('Chat, 3 yangi xabar')).toBeInTheDocument()
  })

  it('o‘qilmagan yo‘q bo‘lsa badge umuman yo‘q', () => {
    setup({ unread: 0 })
    expect(screen.getByLabelText('Chat')).toBeInTheDocument()
  })

  it('9 dan ortiq o‘qilmagan «9+» bo‘lib ko‘rsatiladi', () => {
    setup({ unread: 24 })
    expect(screen.getByText('9+')).toBeInTheDocument()
  })

  it('qo‘l sanog‘i CHATDAN ALOHIDA tugmada ko‘rsatiladi', () => {
    setup({ unread: 3, hands: [{ identity: 'u1', name: 'Ali' }, { identity: 'u2', name: 'Vali' }] })
    // Ikki mustaqil signal: 3 ta o'qilmagan xabar va 2 ta qo'l — qo'shilmaydi.
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByLabelText("Qo'l ko'targanlar, 2 kishi")).toBeInTheDocument()
  })

  it('qo‘l yo‘q bo‘lsa qo‘l badge‘i chizilmaydi', () => {
    setup()
    expect(screen.getByLabelText("Qo'l ko'targanlar, hech kim yo'q")).toBeInTheDocument()
  })

  it('chat tugmasi va qo‘l tugmasi HAR BIRI o‘z callback‘ini chaqiradi', async () => {
    const user = userEvent.setup()
    const { onToggleChat, onToggleHands } = setup({ unread: 2, hands: [{ identity: 'u1', name: 'Ali' }] })

    await user.click(screen.getByLabelText('Chat, 2 yangi xabar'))
    expect(onToggleChat).toHaveBeenCalledTimes(1)
    expect(onToggleHands).not.toHaveBeenCalled()

    await user.click(screen.getByLabelText("Qo'l ko'targanlar, 1 kishi"))
    expect(onToggleHands).toHaveBeenCalledTimes(1)
    expect(onToggleChat).toHaveBeenCalledTimes(1)
  })

  it('mikrofon va «to‘xtatish» kompakt holatda ham qo‘l ostida', async () => {
    const user = userEvent.setup()
    const { onToggleMic, onStopShare } = setup({ micOn: false })
    // Ovozsiz bo'lsa tugma "Ovozsiz" deydi — ustoz gapirib turib
    // "nega meni eshitishmayapti" degan holatga tushmasligi uchun.
    await user.click(screen.getByText('Ovozsiz'))
    expect(onToggleMic).toHaveBeenCalled()
    await user.click(screen.getByText('To‘xtatish'))
    expect(onStopShare).toHaveBeenCalled()
  })

  it('reaksiya bo‘lmasa chizilmaydi, bo‘lsa kim yuborgani bilan chiqadi', () => {
    const { unmount } = setup()
    expect(screen.queryByText(/👍/)).not.toBeInTheDocument()
    unmount()
    setup({ reactions: [{ id: 1, emoji: '👍', name: 'Ali' }] })
    expect(screen.getByText('Ali')).toBeInTheDocument()
  })
})

describe('PresenterPanel — qo‘l ro‘yxati (chatdan mustaqil)', () => {
  it('ro‘yxat overlay ICHIDA, chat panelisiz ochiladi', () => {
    setup({ handsOpen: true, hands: [{ identity: 'u1', name: 'Ali' }, { identity: 'u2', name: 'Vali' }] })
    expect(screen.getByText('Ali')).toBeInTheDocument()
    expect(screen.getByText('Vali')).toBeInTheDocument()
    // Chat paneli OCHILMAYDI — bu talabning o'zagi.
    expect(screen.queryByPlaceholderText('Javob yozing…')).not.toBeInTheDocument()
  })

  it('navbat raqamlangan (kim birinchi so‘ragan)', () => {
    setup({ handsOpen: true, hands: [{ identity: 'u1', name: 'Ali' }, { identity: 'u2', name: 'Vali' }] })
    expect(screen.getByText('1')).toBeInTheDocument()
    // "2" ham qator raqami, ham badge — ikkalasi ham bor bo'lishi kerak.
    expect(screen.getAllByText('2').length).toBeGreaterThan(0)
  })

  it('bo‘sh ro‘yxat tushuntirish bilan qoladi', () => {
    // Bo'sh ro'yxat "hech kim so'ramadi" degan MA'LUMOT — uni yashirish
    // ustozni "ishlayaptimi?" degan shubhaga qoldirardi.
    setup({ handsOpen: true })
    expect(screen.getByText('Hozircha hech kim qo‘l ko‘tarmadi')).toBeInTheDocument()
  })

  it('qo‘lni tushirish identity bilan chaqiriladi', async () => {
    const user = userEvent.setup()
    const { onLowerHand } = setup({ handsOpen: true, hands: [{ identity: 'u1', name: 'Ali' }] })
    await user.click(screen.getByTitle('Tushirish'))
    expect(onLowerHand).toHaveBeenCalledWith('u1')
  })

  it('ro‘yxat ochilishi qo‘l badge‘ini KAMAYTIRMAYDI', () => {
    // Qo'l — o'qilgan/o'qilmagan emas, HOLAT: ro'yxatni ko'rish qo'lni tushirmaydi.
    setup({ handsOpen: true, hands: [{ identity: 'u1', name: 'Ali' }, { identity: 'u2', name: 'Vali' }] })
    expect(screen.getByLabelText("Qo'l ko'targanlar, 2 kishi")).toBeInTheDocument()
  })
})

describe('PresenterPanel — chat paneli', () => {
  const CHAT = [
    { id: 'm1', name: 'Ali', body: 'Savolim bor', self: false, toIdentity: null },
    { id: 'm2', name: 'Ustoz', body: 'Javob', self: true, toIdentity: 'u1' },
  ]

  it('chat ochilganda tarix va kim yozgani ko‘rinadi', () => {
    setup({ chatOpen: true, chat: CHAT })
    expect(screen.getByText('Savolim bor')).toBeInTheDocument()
    expect(screen.getByText('Ali')).toBeInTheDocument()
    expect(screen.getByText('Javob')).toBeInTheDocument()
    expect(screen.getByText('shaxsiy')).toBeInTheDocument()
  })

  it('fayl xabari 📎 belgisi bilan ko‘rinadi (bo‘sh xabar bo‘lib qolmaydi)', () => {
    setup({ chatOpen: true, chat: [{ id: 'f1', name: 'Ali', body: '', self: false, file: { name: 'uy.pdf' } }] })
    expect(screen.getByText('📎 uy.pdf')).toBeInTheDocument()
  })

  it('xabar yo‘q bo‘lsa bo‘shligi aytiladi', () => {
    setup({ chatOpen: true })
    expect(screen.getByText('Hali xabar yo‘q')).toBeInTheDocument()
  })

  it('suzuvchi oynadan javob yuborish mumkin', async () => {
    const user = userEvent.setup()
    const { onSendChat } = setup({ chatOpen: true })
    await user.type(screen.getByPlaceholderText('Javob yozing…'), 'Ha, eshityapman')
    await user.click(screen.getByLabelText('Yuborish'))
    expect(onSendChat).toHaveBeenCalledWith('Ha, eshityapman')
  })

  it('bo‘sh javob yuborilmaydi', async () => {
    const user = userEvent.setup()
    const { onSendChat } = setup({ chatOpen: true })
    await user.click(screen.getByLabelText('Yuborish'))
    expect(onSendChat).not.toHaveBeenCalled()
  })

  it('chat ochiq bo‘lsa ham qo‘l signali tugmada qoladi', () => {
    // Ustoz chat o'qiyotganda qo'l ko'tarilsa ham ko'rishi shart.
    setup({ chatOpen: true, chat: CHAT, hands: [{ identity: 'u1', name: 'Ali' }] })
    expect(screen.getByLabelText("Qo'l ko'targanlar, 1 kishi")).toBeInTheDocument()
  })
})
