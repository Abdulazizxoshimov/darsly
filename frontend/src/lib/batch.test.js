import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mapLimited } from './batch'

// Yozuvlar sahifasi 100 darsni bir vaqtda so'rab o'zini 429 ga urardi.
// Bu yerdagi chegaralar: bir vaqtda ≤ N so'rov va soniyasiga ≤ 1000/min ta.

describe('mapLimited', () => {
  it('bir vaqtda concurrency’dan ko‘p so‘rov YUBORMAYDI', async () => {
    let inflight = 0
    let peak = 0
    const resolvers = []
    const fn = vi.fn(
      () =>
        new Promise((res) => {
          inflight += 1
          peak = Math.max(peak, inflight)
          resolvers.push(() => {
            inflight -= 1
            res('ok')
          })
        }),
    )
    const p = mapLimited([1, 2, 3, 4, 5, 6, 7], fn, { concurrency: 3 })
    await Promise.resolve()
    expect(fn).toHaveBeenCalledTimes(3)
    while (resolvers.length) {
      resolvers.shift()()
      await Promise.resolve()
      await Promise.resolve()
    }
    const out = await p
    expect(peak).toBe(3)
    expect(out).toHaveLength(7)
    expect(out.every((r) => r.ok)).toBe(true)
  })

  it('natija KIRISH tartibida, xato boshqalarini to‘xtatmaydi', async () => {
    const out = await mapLimited(
      ['a', 'b', 'c'],
      async (x, i) => {
        if (x === 'b') throw new Error('yo‘q')
        await new Promise((r) => setTimeout(r, (3 - i) * 5)) // teskari tartibda tugaydi
        return x.toUpperCase()
      },
      { concurrency: 3 },
    )
    expect(out).toEqual([
      { ok: true, value: 'A' },
      { ok: false, error: expect.any(Error) },
      { ok: true, value: 'C' },
    ])
  })

  it('bo‘sh ro‘yxat → bo‘sh natija (worker ochilmaydi)', async () => {
    const fn = vi.fn()
    expect(await mapLimited([], fn)).toEqual([])
    expect(fn).not.toHaveBeenCalled()
  })

  describe('minIntervalMs — barqaror tezlik chegarasi', () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => vi.useRealTimers())

    // Bug: parallellik 4 bo'lsa ham javob tez kelsa (lokal tarmoq) soniyasiga
    // 100+ so'rov ketardi — burst (60) tugagach 429.
    it('so‘rovlar boshlanishi orasida kamida minIntervalMs bo‘ladi', async () => {
      const starts = []
      const fn = vi.fn(async () => {
        starts.push(Date.now())
      })
      const p = mapLimited([1, 2, 3, 4, 5], fn, { concurrency: 4, minIntervalMs: 50 })
      await vi.advanceTimersByTimeAsync(1000)
      await p
      expect(starts).toHaveLength(5)
      for (let i = 1; i < starts.length; i++) expect(starts[i] - starts[i - 1]).toBeGreaterThanOrEqual(50)
    })
  })
})
