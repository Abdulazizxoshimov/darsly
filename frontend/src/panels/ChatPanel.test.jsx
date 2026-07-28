import { describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChatPanel } from './ChatPanel'

// Chat panelining MAHSULOT qoidalari:
//  · shaxsiy xabar ommaviydan ko'rinib turadigan darajada farq qilishi kerak;
//  · qabul qiluvchi xonadan chiqib ketsa xabar yo'q odamga ketmasligi kerak;
//  · bo'sh xabar yuborilmasligi kerak.
// Bularning har biri brauzerda qo'lda tekshiriladigan, ya'ni amalda
// tekshirilmaydigan narsalar edi.

const P = (identity, name) => ({ identity, name })

function setup(over = {}) {
  const onSend = vi.fn()
  const onClose = vi.fn()
  const props = {
    entries: [],
    participants: [P('me', 'Ustoz'), P('u1', 'Ali'), P('u2', 'Vali')],
    localId: 'me',
    onSend,
    onClose,
    ...over,
  }
  const utils = render(<ChatPanel {...props} />)
  return { onSend, onClose, ...utils }
}

describe('ChatPanel', () => {
  it('bo‘sh chatda tushuntirish matni chiqadi', () => {
    setup()
    expect(screen.getByText("Hali xabar yo'q")).toBeInTheDocument()
  })

  it('o‘z xabari "Siz" deb belgilanadi', () => {
    setup({ entries: [{ id: 'm1', name: 'Ustoz', body: 'Salom', self: true, toIdentity: null }] })
    expect(screen.getByText('Siz')).toBeInTheDocument()
    expect(screen.getByText('Salom')).toBeInTheDocument()
  })

  it('shaxsiy xabar KIMGA ekani bilan belgilanadi', () => {
    setup({
      entries: [{ id: 'm1', name: 'Ustoz', body: 'Faqat sizga', self: true, toIdentity: 'u1' }],
    })
    // Yuboruvchi tomonda: "→ Ali"
    expect(screen.getByText('→ Ali')).toBeInTheDocument()
  })

  it('kelgan shaxsiy xabar "sizga shaxsiy" deb belgilanadi', () => {
    setup({
      entries: [{ id: 'm1', name: 'Ali', body: 'Savolim bor', self: false, toIdentity: 'me' }],
    })
    expect(screen.getByText('sizga shaxsiy')).toBeInTheDocument()
  })

  it('qabul qiluvchi ismi JONLI ro‘yxatdan olinadi', () => {
    // Ism xabar kelgan ondagi holatda muzlab qolmasligi kerak.
    const entries = [{ id: 'm1', name: 'Ustoz', body: 'x', self: true, toIdentity: 'u1' }]
    const { rerender } = setup({ entries })
    expect(screen.getByText('→ Ali')).toBeInTheDocument()

    rerender(
      <ChatPanel
        entries={entries}
        participants={[P('me', 'Ustoz'), P('u1', 'Ali Valiyev')]}
        localId="me"
        onSend={() => {}}
        onClose={() => {}}
      />,
    )
    expect(screen.getByText('→ Ali Valiyev')).toBeInTheDocument()
  })

  it('xabar yuborilganda qabul qiluvchi bilan birga uzatiladi', async () => {
    const user = userEvent.setup()
    const { onSend } = setup()

    await user.selectOptions(screen.getByLabelText('Kimga'), 'u1')
    // Placeholder tanlovga qarab o'zgaradi — matnga emas, ROLGA so'rov beramiz.
    await user.type(screen.getByRole('textbox'), 'Salom')
    await user.click(screen.getByLabelText('Yuborish'))

    expect(onSend).toHaveBeenCalledWith('Salom', 'u1')
  })

  it('bo‘sh yoki faqat probeldan iborat xabar yuborilmaydi', async () => {
    const user = userEvent.setup()
    const { onSend } = setup()

    await user.click(screen.getByLabelText('Yuborish'))
    await user.type(screen.getByRole('textbox'), '   ')
    await user.click(screen.getByLabelText('Yuborish'))

    expect(onSend).not.toHaveBeenCalled()
  })

  it('tanlangan qabul qiluvchi chiqib ketsa "Hammaga"ga qaytadi', async () => {
    const user = userEvent.setup()
    const participants = [P('me', 'Ustoz'), P('u1', 'Ali')]
    const { rerender } = setup({ participants })

    await user.selectOptions(screen.getByLabelText('Kimga'), 'u1')
    expect(screen.getByLabelText('Kimga')).toHaveValue('u1')

    // Ali xonadan chiqdi.
    rerender(
      <ChatPanel
        entries={[]}
        participants={[P('me', 'Ustoz')]}
        localId="me"
        onSend={() => {}}
        onClose={() => {}}
      />,
    )
    // Endi umuman boshqa ishtirokchi yo'q → tanlov ko'rsatilmaydi va
    // keyingi xabar hammaga ketadi (yo'q odamga emas).
    expect(screen.queryByLabelText('Kimga')).not.toBeInTheDocument()
  })

  it('o‘zimiz ro‘yxatda ko‘rsatilmaymiz (o‘zimizga yozish ma’nosiz)', () => {
    setup()
    const select = screen.getByLabelText('Kimga')
    expect(within(select).queryByText('Ustoz')).not.toBeInTheDocument()
    expect(within(select).getByText('Ali')).toBeInTheDocument()
  })

  it('yopish tugmasi ishlaydi', async () => {
    const user = userEvent.setup()
    const { onClose } = setup()
    await user.click(screen.getByLabelText('Yopish'))
    expect(onClose).toHaveBeenCalled()
  })
})
