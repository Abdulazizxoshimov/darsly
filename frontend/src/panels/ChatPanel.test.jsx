import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
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

  // Tezlik cheklovi xabarni JIMGINA yutib yuborardi: maydon tozalanardi,
  // xabar esa hech qayerga bormasdi va foydalanuvchi buni faqat javob
  // kelmaganidan bilardi.
  it('yuborilmagan xabar maydonda QOLADI va sabab ko‘rsatiladi', async () => {
    const user = userEvent.setup()
    const onSend = vi.fn().mockReturnValue(false) // tezlik cheklovi
    setup({ onSend })

    const input = screen.getByRole('textbox')
    await user.type(input, 'Salom')
    await user.click(screen.getByLabelText('Yuborish'))

    expect(input).toHaveValue('Salom')
    expect(screen.getByRole('alert')).toHaveTextContent(/juda tez/i)
  })

  it('tarix yuklanmasa «xabar yo‘q» DEMAYDI, sababni aytadi', async () => {
    const user = userEvent.setup()
    const onRetryHistory = vi.fn()
    setup({ historyError: true, onRetryHistory })

    expect(screen.queryByText("Hali xabar yo'q")).not.toBeInTheDocument()
    expect(screen.getByText(/eski xabarlarni yuklab bo‘lmadi/i)).toBeInTheDocument()

    await user.click(screen.getByText('Qayta urinish'))
    expect(onRetryHistory).toHaveBeenCalled()
  })
})

// ── Fayl ulashish ───────────────────────────────────────────────────────────
// MAHSULOT QOIDALARI:
//  · yaroqsiz fayl (hajm/tur) SERVERGA UMUMAN ketmaydi — 20 MB'ni sekin mobil
//    internetda yuklab, so'ng 400 olish foydalanuvchi uchun eng yomon holat;
//  · sabab panel ICHIDA qoladi (toast 4 soniyada yo'qoladi va uzoq yuklashdan
//    keyin ekranga qaytgan foydalanuvchi uni ko'rmasdi);
//  · progress ko'rinadi — busiz "ilova qotdi" taassuroti paydo bo'ladi.

// happy-dom'da `File.size` mazmundan hisoblanadi — katta faylni SOXTA
// o'lchov bilan yasaymiz (20 MB xotirada ushlab turmaslik uchun).
function fakeFile(name, type, size = 1024) {
  const f = new File(['x'], name, { type })
  Object.defineProperty(f, 'size', { value: size })
  return f
}

function pick(file) {
  fireEvent.change(screen.getByTestId('chat-file-input'), { target: { files: [file] } })
}

describe('ChatPanel — fayl ulashish', () => {
  it('onSendFile berilmasa biriktirish tugmasi ko‘rsatilmaydi', () => {
    setup()
    expect(screen.queryByLabelText('Fayl biriktirish')).not.toBeInTheDocument()
  })

  it('onSendFile berilsa biriktirish tugmasi chiqadi', () => {
    setup({ onSendFile: vi.fn() })
    expect(screen.getByLabelText('Fayl biriktirish')).toBeInTheDocument()
  })

  it('20 MB dan katta fayl yuborilmaydi — sabab panelda ko‘rinadi', async () => {
    const onSendFile = vi.fn()
    setup({ onSendFile })
    pick(fakeFile('katta.pdf', 'application/pdf', 21 * 1024 * 1024))
    expect(await screen.findByRole('alert')).toHaveTextContent(/20 MB/)
    expect(onSendFile).not.toHaveBeenCalled()
  })

  it('ruxsat etilmagan tur yuborilmaydi', async () => {
    const onSendFile = vi.fn()
    setup({ onSendFile })
    pick(fakeFile('virus.exe', 'application/octet-stream'))
    expect(await screen.findByRole('alert')).toHaveTextContent(/qabul qilinmaydi/i)
    expect(onSendFile).not.toHaveBeenCalled()
  })

  it('yaroqli fayl tanlangan qabul qiluvchi bilan uzatiladi va progress ko‘rinadi', async () => {
    const user = userEvent.setup()
    let resolveUpload
    const onSendFile = vi.fn(
      (file, to, onProgress) =>
        new Promise((res) => {
          onProgress(42)
          resolveUpload = res
        }),
    )
    setup({ onSendFile })

    await user.selectOptions(screen.getByLabelText('Kimga'), 'u1')
    const file = fakeFile('uy_ishi.pdf', 'application/pdf', 184_320)
    pick(file)

    await waitFor(() => expect(onSendFile).toHaveBeenCalled())
    expect(onSendFile.mock.calls[0][0]).toBe(file)
    expect(onSendFile.mock.calls[0][1]).toBe('u1')

    // Progress paneli: fayl nomi va foiz.
    expect(await screen.findByText('uy_ishi.pdf')).toBeInTheDocument()
    expect(screen.getByText('42%')).toBeInTheDocument()

    resolveUpload()
    await waitFor(() => expect(screen.queryByText('42%')).not.toBeInTheDocument())
  })

  it('yuklash davomida panel yopilmaydi, lekin BEKOR qilsa bo‘ladi', async () => {
    const user = userEvent.setup()
    let abortedSignal = null
    const onSendFile = vi.fn(
      (file, to, onProgress, signal) =>
        new Promise((_res, rej) => {
          onProgress(10)
          signal.addEventListener('abort', () => {
            abortedSignal = true
            rej(Object.assign(new Error('bekor'), { code: 'ABORTED' }))
          })
        }),
    )
    setup({ onSendFile })
    pick(fakeFile('katta.pdf', 'application/pdf', 5_000_000))

    // Holat panelda yashaydi — yopilsa progress ham, xato ham yo'qolardi.
    expect(await screen.findByText('10%')).toBeInTheDocument()
    expect(screen.getByLabelText('Yopish')).toBeDisabled()

    await user.click(screen.getByText('Bekor qilish'))
    await waitFor(() => expect(abortedSignal).toBe(true))
    await waitFor(() => expect(screen.getByLabelText('Yopish')).toBeEnabled())
  })

  it('server xatosi o‘zbekcha sabab bilan ko‘rsatiladi', async () => {
    const err = Object.assign(new Error('file is too large'), { code: 'BAD_REQUEST' })
    setup({ onSendFile: vi.fn().mockRejectedValue(err) })
    pick(fakeFile('katta.pdf', 'application/pdf'))
    expect(await screen.findByRole('alert')).toHaveTextContent(/20 MB/)
  })

  it('fayl kartochkasi nom va hajm bilan ko‘rinadi', () => {
    setup({
      entries: [
        {
          id: 'm1',
          name: 'Ali',
          body: '',
          self: false,
          toIdentity: null,
          file: { name: 'uy_ishi.pdf', size: 184_320, mime: 'application/pdf', url: 'https://x/y.pdf' },
        },
      ],
    })
    expect(screen.getByText('uy_ishi.pdf')).toBeInTheDocument()
    expect(screen.getByText('180 KB')).toBeInTheDocument()
  })

  it('rasm uchun kichik ko‘rish (thumbnail) chiqadi', () => {
    setup({
      entries: [
        {
          id: 'm1',
          name: 'Ali',
          body: '',
          self: false,
          toIdentity: null,
          file: { name: 'doska.png', size: 2048, mime: 'image/png', url: 'https://x/doska.png' },
        },
      ],
    })
    expect(screen.getByAltText('doska.png')).toBeInTheDocument()
  })
})

// ── Moderatsiya: xabarni o'chirish ─────────────────────────────────────────
// Xabar hammadan IZSIZ yo'qoladi va qaytarib bo'lmaydi — shuning uchun
// tasdiqsiz bajarilmaydi va faqat ustozga ko'rinadi.
describe('ChatPanel — xabarni o‘chirish', () => {
  const ENTRY = [{ id: 'm1', name: 'Ali', body: 'Yomon so‘z', self: false, toIdentity: null }]

  it('o‘chirish tugmasi o‘quvchida YO‘Q', () => {
    setup({ entries: ENTRY, onDelete: vi.fn(), canDelete: false })
    expect(screen.queryByLabelText("Xabarni o'chirish")).not.toBeInTheDocument()
  })

  it('tasdiqlanmaguncha o‘chirilmaydi', async () => {
    const user = userEvent.setup()
    const onDelete = vi.fn()
    setup({ entries: ENTRY, onDelete, canDelete: true })

    await user.click(screen.getByLabelText("Xabarni o'chirish"))
    expect(screen.getByText("Xabarni o'chirasizmi?")).toBeInTheDocument()
    expect(onDelete).not.toHaveBeenCalled()

    await user.click(screen.getByText('Bekor qilish'))
    expect(onDelete).not.toHaveBeenCalled()
  })

  it('tasdiqdan keyin xabar ID si bilan o‘chiriladi', async () => {
    const user = userEvent.setup()
    const onDelete = vi.fn().mockResolvedValue(undefined)
    setup({ entries: ENTRY, onDelete, canDelete: true })

    await user.click(screen.getByLabelText("Xabarni o'chirish"))
    await user.click(screen.getByText("Ha, o'chirish"))
    await waitFor(() => expect(onDelete).toHaveBeenCalledWith('m1'))
  })
})
