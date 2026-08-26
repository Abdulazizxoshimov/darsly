import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Controls } from './Controls'
import { SCREEN_PUBLISH, CAMERA_PUBLISH } from './mediaTuning'
import { toast } from '../lib/toast'

// Ekran ulashish oldidan chiqadigan eslatma.
//
// Nega bu test kritik: «Butun ekran» tanlansa suzuvchi chat oynasi HAM efirga
// tushadi va o'quvchilar ustozning shaxsiy yozishmalarini ko'radi. Buni kod
// to'sa olmaydi (OS cheklovi) — yagona himoya shu eslatma. Agar u jimgina
// yo'qolib qolsa, buni faqat maxfiylik buzilgandan keyin bilib qolamiz.

function setup(over = {}) {
  const setScreenShareEnabled = vi.fn().mockResolvedValue(undefined)
  const room = {
    localParticipant: {
      setScreenShareEnabled,
      setMicrophoneEnabled: vi.fn().mockResolvedValue(undefined),
      setCameraEnabled: vi.fn().mockResolvedValue(undefined),
    },
  }
  render(
    <Controls
      room={room}
      local={{ identity: 'u1', micOn: true, camOn: false, screenOn: false, canPublish: true }}
      isHost
      recording={false}
      onToggleRecord={vi.fn()}
      panel="none"
      setPanel={vi.fn()}
      handRaised={false}
      onToggleHand={vi.fn()}
      onReaction={vi.fn()}
      onLeave={vi.fn()}
      unreadChat={0}
      waitingCount={0}
      whiteboardOn={false}
      onToggleBoard={vi.fn()}
      dataSaver={false}
      onToggleDataSaver={vi.fn()}
      {...over}
    />,
  )
  return { setScreenShareEnabled }
}

describe('Controls — ekran ulashish eslatmasi', () => {
  beforeEach(() => {
    localStorage.clear()
    // Kompyuter brauzeri: getDisplayMedia bor (mobil tekshiruvidan o'tsin).
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(), getDisplayMedia: vi.fn() },
    })
  })
  afterEach(() => localStorage.clear())

  it('«Ekran» bosilganda AVVAL eslatma chiqadi, ulashish boshlanmaydi', async () => {
    const user = userEvent.setup()
    const { setScreenShareEnabled } = setup()

    await user.click(screen.getByLabelText('Ekran'))

    expect(screen.getByText('Bitta oynani ulashing')).toBeInTheDocument()
    expect(setScreenShareEnabled).not.toHaveBeenCalled()
  })

  it('tasdiqlangach ulashish boshlanadi', async () => {
    const user = userEvent.setup()
    const { setScreenShareEnabled } = setup()

    await user.click(screen.getByLabelText('Ekran'))
    await user.click(screen.getByText('Tushunarli, davom etish'))

    expect(setScreenShareEnabled).toHaveBeenCalledWith(true, expect.anything(), SCREEN_PUBLISH)
  })

  it('bekor qilinsa ulashish BOSHLANMAYDI', async () => {
    const user = userEvent.setup()
    const { setScreenShareEnabled } = setup()

    await user.click(screen.getByLabelText('Ekran'))
    await user.click(screen.getByText('Bekor qilish'))

    expect(setScreenShareEnabled).not.toHaveBeenCalled()
    expect(screen.queryByText('Bitta oynani ulashing')).not.toBeInTheDocument()
  })

  it('«boshqa eslatilmasin» belgilansa keyingi darsda ham chiqmaydi', async () => {
    const user = userEvent.setup()
    const first = setup()
    await user.click(screen.getByLabelText('Ekran'))
    await user.click(screen.getByLabelText('Boshqa eslatilmasin'))
    await user.click(screen.getByText('Tushunarli, davom etish'))
    expect(first.setScreenShareEnabled).toHaveBeenCalledTimes(1)

    // Yangi dars = yangi mount. Belgi localStorage'da (sessiyada emas), ya'ni
    // o'rgangan ustoz eslatmani boshqa ko'rmaydi.
    cleanup()
    const second = setup()
    await user.click(screen.getByLabelText('Ekran'))
    expect(screen.queryByText('Bitta oynani ulashing')).not.toBeInTheDocument()
    expect(second.setScreenShareEnabled).toHaveBeenCalledWith(true, expect.anything(), SCREEN_PUBLISH)
  })

  it('eslatma o‘chirilgan bo‘lsa modal umuman chiqmaydi', async () => {
    localStorage.setItem('jonly.ui.prefs', JSON.stringify({ shareWindowTip: true }))
    const user = userEvent.setup()
    const { setScreenShareEnabled } = setup()

    await user.click(screen.getByLabelText('Ekran'))

    expect(screen.queryByText('Bitta oynani ulashing')).not.toBeInTheDocument()
    expect(setScreenShareEnabled).toHaveBeenCalledWith(true, expect.anything(), SCREEN_PUBLISH)
  })

  it('ulashishni TO‘XTATISH eslatmasiz, darhol ishlaydi', async () => {
    const user = userEvent.setup()
    const { setScreenShareEnabled } = setup({
      local: { identity: 'u1', micOn: true, camOn: false, screenOn: true, canPublish: true },
    })

    await user.click(screen.getByLabelText('Ekran'))

    expect(screen.queryByText('Bitta oynani ulashing')).not.toBeInTheDocument()
    expect(setScreenShareEnabled).toHaveBeenCalledWith(false, undefined, SCREEN_PUBLISH)
  })
})

// F-3 — moderatsiya gating (server enforce qiladi; UI tugmani o'chirib SABABINI aytadi).
describe('Controls — mikrofon/kamera moderatsiya gating', () => {
  beforeEach(() => {
    localStorage.clear()
    // Xavfsiz origin: mediaDevices bor (mediaOk=true), aks holda barcha tugmalar
    // "HTTPS kerak" bilan o'chib qolib gatingni yashirib qo'yardi.
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(), getDisplayMedia: vi.fn() },
    })
  })
  afterEach(() => localStorage.clear())

  // Bug: allow_self_unmute=false darsda o'quvchi o'zini och(a olmaydi)sa ham
  // tugma faol ko'rinsa — bosadi, server qayta mute qiladi, u sababni bilmaydi.
  it('o‘quvchi + selfUnmuteBlocked + mic o‘chiq → mikrofon tugmasi O‘CHIQ va sababi tooltipda', () => {
    setup({
      isHost: false,
      selfUnmuteBlocked: true,
      local: { identity: 's1', micOn: false, camOn: false, screenOn: false, canPublish: true },
    })
    const mic = screen.getByLabelText('Mentor ruxsat bermagan — mikrofonni ustoz ochadi')
    expect(mic).toBeDisabled()
  })

  // Bug: mute qilish (mikrofon YOQIQ holatda) hech qachon bloklanmasligi kerak —
  // aks holda o'quvchi o'zini jimlata olmaydi.
  it('mic YOQIQ bo‘lsa selfUnmuteBlocked ta’sir qilmaydi (mute doim mumkin)', () => {
    setup({
      isHost: false,
      selfUnmuteBlocked: true,
      local: { identity: 's1', micOn: true, camOn: false, screenOn: false, canPublish: true },
    })
    expect(screen.getByLabelText('Mikrofon')).not.toBeDisabled()
  })

  // Bug: ustoz o'ziga ovoz siyosatini qo'llamasligi kerak — u xonani boshqaradi.
  it('HOST uchun selfUnmuteBlocked mikrofonni bloklamaydi', () => {
    setup({
      isHost: true,
      selfUnmuteBlocked: true,
      local: { identity: 'h1', micOn: false, camOn: false, screenOn: false, canPublish: true },
    })
    expect(screen.getByLabelText('Mikrofon')).not.toBeDisabled()
  })

  // Bug: ustoz kamera ruxsati bermagan o'quvchida kamera tugmasi faol ko'rinsa,
  // bosilganda LiveKit tushunarsiz xato beradi va o'quvchi NIMA qilishni bilmaydi.
  it('kamera ruxsati yo‘q o‘quvchida kamera tugmasi O‘CHIQ va «qo‘l ko‘taring» deydi', () => {
    setup({
      isHost: false,
      local: {
        identity: 's1', micOn: false, camOn: false, screenOn: false,
        canPublish: true, canPublishCamera: false,
      },
    })
    const cam = screen.getByLabelText('Ustoz kamera uchun ruxsat bermagan — qo‘l ko‘taring')
    expect(cam).toBeDisabled()
  })

  // Bug: ustoz kamera bergach (canPublishCamera=true) tugma ochilishi kerak.
  it('kamera ruxsati berilgach kamera tugmasi ochiladi', () => {
    setup({
      isHost: false,
      local: {
        identity: 's1', micOn: false, camOn: false, screenOn: false,
        canPublish: true, canPublishCamera: true,
      },
    })
    expect(screen.getByLabelText('Kamera')).not.toBeDisabled()
  })
})

// F-3 — mediaFailText: getUserMedia xatosini TUZATILADIGAN o'zbekcha matnga aylantirish.
describe('Controls — media xato matni (mediaFailText)', () => {
  beforeEach(() => {
    localStorage.clear()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(), getDisplayMedia: vi.fn() },
    })
    vi.spyOn(toast, 'error').mockImplementation(() => {})
  })
  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  function setupWithMic(rejectErr) {
    const setMicrophoneEnabled = vi.fn().mockRejectedValue(rejectErr)
    const room = {
      localParticipant: {
        setMicrophoneEnabled,
        setCameraEnabled: vi.fn().mockResolvedValue(undefined),
        setScreenShareEnabled: vi.fn().mockResolvedValue(undefined),
      },
    }
    render(
      <Controls
        room={room}
        local={{ identity: 'h1', micOn: true, camOn: false, screenOn: false, canPublish: true, canPublishCamera: true }}
        isHost recording={false} onToggleRecord={vi.fn()} panel="none" setPanel={vi.fn()}
        handRaised={false} onToggleHand={vi.fn()} onReaction={vi.fn()} onLeave={vi.fn()}
        unreadChat={0} waitingCount={0} whiteboardOn={false} onToggleBoard={vi.fn()}
        dataSaver={false} onToggleDataSaver={vi.fn()}
      />,
    )
  }

  // Bug: umumiy "yoqib bo'lmadi" ruxsat, band qurilma va yo'q qurilmani
  // ajratmasa, foydalanuvchi qaysi harakatni qilishni bilmaydi.
  it('NotAllowedError → «ruxsat berilmagan» (qulf belgisidan ruxsat bering)', async () => {
    const user = userEvent.setup()
    setupWithMic(Object.assign(new Error('denied'), { name: 'NotAllowedError' }))
    await user.click(screen.getByLabelText('Mikrofon'))
    expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/ruxsat berilmagan/i))
  })

  it('NotReadableError → «band — boshqa dasturni yoping»', async () => {
    const user = userEvent.setup()
    setupWithMic(Object.assign(new Error('busy'), { name: 'NotReadableError' }))
    await user.click(screen.getByLabelText('Mikrofon'))
    expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/band/i))
  })

  it('NotFoundError → «topilmadi — qurilma ulanganini tekshiring»', async () => {
    const user = userEvent.setup()
    setupWithMic(Object.assign(new Error('none'), { name: 'NotFoundError' }))
    await user.click(screen.getByLabelText('Mikrofon'))
    expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/topilmadi/i))
  })

  it('noma’lum xato → umumiy «yoqib bo‘lmadi»', async () => {
    const user = userEvent.setup()
    setupWithMic(new Error('weird'))
    await user.click(screen.getByLabelText('Mikrofon'))
    expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/yoqib bo/i))
  })

  // Bug: kamera xatosi ham (mikrofon emas) o'z matni bilan chiqishi kerak —
  // "Kamera band", "Mikrofon band" emas.
  it('kamera xatosida matn «Kamera» bilan boshlanadi', async () => {
    const user = userEvent.setup()
    const setCameraEnabled = vi.fn().mockRejectedValue(Object.assign(new Error('x'), { name: 'NotReadableError' }))
    const room = {
      localParticipant: {
        setCameraEnabled,
        setMicrophoneEnabled: vi.fn().mockResolvedValue(undefined),
        setScreenShareEnabled: vi.fn().mockResolvedValue(undefined),
      },
    }
    render(
      <Controls
        room={room}
        local={{ identity: 'h1', micOn: true, camOn: false, screenOn: false, canPublish: true, canPublishCamera: true }}
        isHost recording={false} onToggleRecord={vi.fn()} panel="none" setPanel={vi.fn()}
        handRaised={false} onToggleHand={vi.fn()} onReaction={vi.fn()} onLeave={vi.fn()}
        unreadChat={0} waitingCount={0} whiteboardOn={false} onToggleBoard={vi.fn()}
        dataSaver={false} onToggleDataSaver={vi.fn()}
      />,
    )
    await user.click(screen.getByLabelText('Kamera'))
    expect(setCameraEnabled).toHaveBeenCalledWith(true, undefined, CAMERA_PUBLISH)
    expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/^Kamera band/i))
  })
})

// Yozib olish start/stop UI. (E2E darajasida SFU xonasi ichida — u yerda useRoom
// haqiqiy LiveKit ulanishini talab qiladi, shuning uchun qamrov chegarasidan
// tashqarida; bu yerda TUGMA mantig'i komponent darajasida sinaladi.)
describe('Controls — yozib olish tugmasi (start/stop)', () => {
  beforeEach(() => {
    localStorage.clear()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn(), getDisplayMedia: vi.fn() },
    })
  })
  afterEach(() => localStorage.clear())

  // Bug: yozib olish tugmasi o'quvchiga ko'rinsa, u ishlamaydigan tugmani bosadi
  // (backend baribir 403 beradi) — faqat host ko'rishi kerak.
  it('o‘quvchiga yozib olish tugmasi KO‘RINMAYDI', () => {
    setup({ isHost: false, local: { identity: 's1', micOn: true, canPublish: true } })
    expect(screen.queryByLabelText(/Yozib olish|to‘xtatish/i)).not.toBeInTheDocument()
  })

  // Bug: recording=false da tugma «to'xtatish» deb tursa, host yozuv ketayotgan
  // deb o'ylaydi (aslida yo'q).
  it('host + recording=false → «Yozib olish», bosilganda onToggleRecord chaqiriladi', async () => {
    const user = userEvent.setup()
    const onToggleRecord = vi.fn()
    setup({ isHost: true, recording: false, onToggleRecord })
    const btn = screen.getByLabelText('Yozib olish')
    await user.click(btn)
    expect(onToggleRecord).toHaveBeenCalledTimes(1)
  })

  // Bug: recording=true da tugma «Yozib olish» deb tursa, host yozuvni to'xtata
  // olmaydi (yoki ikkinchi yozuv boshlaydi deb qo'rqadi).
  it('host + recording=true → «Yozilmoqda — to‘xtatish» (danger holati)', () => {
    setup({ isHost: true, recording: true })
    expect(screen.getByLabelText('Yozilmoqda — to‘xtatish')).toBeInTheDocument()
    expect(screen.queryByLabelText('Yozib olish')).not.toBeInTheDocument()
  })
})
