import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Controls } from './Controls'
import { SCREEN_PUBLISH } from './mediaTuning'

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
