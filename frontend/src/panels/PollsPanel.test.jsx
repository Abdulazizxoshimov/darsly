import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PollsPanel } from './PollsPanel'

// MAHSULOT QOIDASI №7 — so'rovnoma natijasi IKKI rejimda:
//  · mentor_only (default) — natija faqat ustozda, e'lon qilib bo'lmaydi;
//  · public — o'quvchi ko'radi, LEKIN faqat ustoz «E'lon qilish» bosgach.
//
// O'quvchi tomonda eng muhim qoida: e'lon qilinmagan natija o'rniga BO'SH
// diagramma chizilmasligi kerak. «0 ovoz» bilan «ko'rsatilmaydi» ni farqlab
// bo'lmasa o'quvchi noto'g'ri xulosa chiqaradi.

const hooks = vi.hoisted(() => ({
  polls: vi.fn(),
  results: vi.fn(),
  create: vi.fn(),
  close: vi.fn(),
  publish: vi.fn(),
}))

vi.mock('../store/data', () => ({
  usePolls: () => ({ data: hooks.polls() }),
  usePollResults: (...a) => ({ data: hooks.results(...a) }),
  useCreatePoll: () => ({ mutateAsync: hooks.create, isPending: false }),
  useClosePoll: () => ({ mutateAsync: hooks.close, isPending: false }),
  usePublishPoll: () => ({ mutateAsync: hooks.publish, isPending: false }),
}))

vi.mock('../api/polls', async (orig) => ({
  ...(await orig()),
  votePoll: vi.fn().mockResolvedValue(null),
}))

import { votePoll } from '../api/polls'

const ROOM_TOKEN = { token: 'rt', ws_url: 'ws://x', identity: 'i', role: 'participant' }

function setup(over = {}) {
  const onBroadcastPoll = vi.fn()
  render(
    <PollsPanel
      isHost={false}
      lessonId="l1"
      roomToken={ROOM_TOKEN}
      guestActivePoll={null}
      publishedResults={null}
      votedPollId={null}
      onVoted={() => {}}
      onBroadcastPoll={onBroadcastPoll}
      onClose={() => {}}
      {...over}
    />,
  )
  return { onBroadcastPoll }
}

const POLL_PUBLIC = {
  id: 'p1',
  question: 'Tushunarli bo‘ldimi?',
  options: ['Ha', "Yo'q"],
  is_active: true,
  results_visibility: 'public',
  results_published_at: null,
}
const POLL_PRIVATE = { ...POLL_PUBLIC, id: 'p2', results_visibility: 'mentor_only' }

beforeEach(() => {
  vi.clearAllMocks()
  hooks.polls.mockReturnValue([])
  hooks.results.mockReturnValue(undefined)
  hooks.publish.mockResolvedValue({ poll: POLL_PUBLIC, counts: [7, 3], total: 10 })
})

describe('PollsPanel — ustoz', () => {
  it('yaratishda natija ko‘rinuvchanligi TANLANADI va default yopiq tomon', async () => {
    const user = userEvent.setup()
    hooks.create.mockResolvedValue({ ...POLL_PUBLIC, results_visibility: 'mentor_only' })
    const { onBroadcastPoll } = setup({ isHost: true })

    await user.click(screen.getByText("Yangi so'rovnoma"))
    const vis = screen.getByLabelText("Natija kimga ko'rinadi")
    expect(vis).toHaveValue('mentor_only') // default — yopiq tomon

    await user.type(screen.getByPlaceholderText('Savol'), 'Tushunarlimi?')
    await user.type(screen.getByPlaceholderText('Variant 1'), 'Ha')
    await user.type(screen.getByPlaceholderText('Variant 2'), "Yo'q")
    await user.selectOptions(vis, 'public')
    await user.click(screen.getByText('Boshlash'))

    await waitFor(() => expect(hooks.create).toHaveBeenCalled())
    expect(hooks.create.mock.calls[0][0].resultsVisibility).toBe('public')
    // O'quvchiga rejim ham aytiladi — ovoz bergach qanday matn ko'rsatishni
    // shu belgilaydi.
    expect(onBroadcastPoll.mock.calls[0][1].results_visibility).toBeDefined()
  })

  it('public so‘rovnomada «E‘lon qilish» tugmasi bor', async () => {
    const user = userEvent.setup()
    hooks.polls.mockReturnValue([POLL_PUBLIC])
    setup({ isHost: true })

    const btn = screen.getByText("Natijani e'lon qilish")
    expect(screen.getByText("Natija hali e'lon qilinmagan")).toBeInTheDocument()

    await user.click(btn)
    await waitFor(() => expect(hooks.publish).toHaveBeenCalledWith({ lessonId: 'l1', pollId: 'p1' }))
  })

  it('mentor_only so‘rovnomada e‘lon qilish tugmasi YO‘Q', () => {
    hooks.polls.mockReturnValue([POLL_PRIVATE])
    setup({ isHost: true })
    expect(screen.queryByText("Natijani e'lon qilish")).not.toBeInTheDocument()
    expect(screen.getByText('Natija faqat sizda')).toBeInTheDocument()
  })

  it('e‘lon qilingan so‘rovnomada tugma qayta ko‘rsatilmaydi', () => {
    hooks.polls.mockReturnValue([{ ...POLL_PUBLIC, results_published_at: '2026-07-31T09:00:00Z' }])
    setup({ isHost: true })
    expect(screen.queryByText("Natijani e'lon qilish")).not.toBeInTheDocument()
    expect(screen.getByText("Natija e'lon qilingan")).toBeInTheDocument()
  })
})

describe('PollsPanel — o‘quvchi', () => {
  it('faol so‘rovnoma yo‘qligi tushuntiriladi', () => {
    setup()
    expect(screen.getByText("Hozircha faol so'rovnoma yo'q")).toBeInTheDocument()
  })

  it('ovoz berish serverga yuboriladi va holat XONAGA ko‘tariladi', async () => {
    const user = userEvent.setup()
    const onVoted = vi.fn()
    setup({ guestActivePoll: POLL_PUBLIC, onVoted })

    await user.click(screen.getByText('Ha'))
    await waitFor(() => expect(votePoll).toHaveBeenCalledWith('p1', 'rt', 0))
    // Holat panelda EMAS: panel yopilib qayta ochilsa ham ovoz "esda qoladi".
    await waitFor(() => expect(onVoted).toHaveBeenCalledWith('p1'))
  })

  it('ovoz berilgach natija E‘LON QILINMAGUNCHA ko‘rsatilmaydi', () => {
    setup({ guestActivePoll: POLL_PUBLIC, votedPollId: 'p1' })

    expect(screen.getByText('Ovozingiz qabul qilindi')).toBeInTheDocument()
    expect(screen.getByText("Natijani ustoz hali e'lon qilmagan.")).toBeInTheDocument()
    // Bo'sh diagramma CHIZILMAYDI.
    expect(document.querySelector('.poll-bar-track')).toBeNull()
  })

  it('mentor_only rejimda sabab boshqacha — natija umuman ochilmaydi', () => {
    setup({ guestActivePoll: POLL_PRIVATE, votedPollId: POLL_PRIVATE.id })
    expect(screen.getByText("Natijani faqat ustoz ko'radi.")).toBeInTheDocument()
  })

  it('ovoz xatosi ANIQ sabab bilan ko‘rsatiladi (umumiy «so‘rovda xatolik» emas)', async () => {
    const user = userEvent.setup()
    votePoll.mockRejectedValueOnce(Object.assign(new Error('poll is closed'), { code: 'BAD_REQUEST' }))
    setup({ guestActivePoll: POLL_PUBLIC })
    await user.click(screen.getByText('Ha'))
    expect(await screen.findByRole('alert')).toHaveTextContent(/yopilgan/i)
  })

  it('yopilgan so‘rovnoma ekrandan YO‘QOLMAYDI (savol ko‘rinib turadi)', () => {
    // Ustoz odatda avval ovozni yopib, keyin natijani e'lon qiladi.
    setup({ guestActivePoll: { ...POLL_PUBLIC, is_active: false } })
    expect(screen.getByText('Tushunarli bo‘ldimi?')).toBeInTheDocument()
    expect(screen.getByText('So‘rovnoma yopilgan')).toBeInTheDocument()
  })

  it('poll_published kelganda natija darhol chiqadi (qo‘shimcha so‘rovsiz)', () => {
    setup({
      guestActivePoll: POLL_PUBLIC,
      publishedResults: { poll: POLL_PUBLIC, counts: [7, 3], total: 10 },
    })
    expect(screen.getByText('10 ovoz')).toBeInTheDocument()
    expect(screen.getByText('70%')).toBeInTheDocument()
    // Natija serverdan data-channel bilan keladi — HTTP so'rov yoqilmaydi.
    expect(hooks.results).toHaveBeenCalled()
    const opts = hooks.results.mock.calls.at(-1)[2]
    expect(opts.enabled).toBe(false)
  })
})
