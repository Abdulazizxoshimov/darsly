import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { connectRealtime, connectWaitingRoom } from './ws'
import { tokenStore } from '../api/api'

// F-2 — WS avtomatik qayta ulanish (eksponensial backoff).
//
// Nega kritik: bu jadval NOTO'G'RI bo'lsa ikki yomon holat bo'ladi —
//  · backoff yo'q/juda tez → uzilishda server "reconnect bo'roni" yeydi;
//  · ceiling yo'q → uzoq uzilishdan keyin qayta ulanish daqiqalab kutdiradi
//    (2^n cheksiz o'sadi) va real-time kanal amalda o'ladi.
// Timer'lar soxta — jadvalni deterministik o'lchaymiz, real kutmaymiz.

// happy-dom'da WebSocket bor, lekin u haqiqiy tarmoqqa uriladi. Uni to'liq
// nazorat qilinadigan stub bilan almashtiramiz: har instansiyani ro'yxatga
// olamiz va `close`/`open` ni QO'LDA chaqiramiz.
let sockets
class MockWS {
  constructor(url) {
    this.url = url
    this.readyState = 0
    this.onopen = null
    this.onmessage = null
    this.onclose = null
    this.onerror = null
    this.closeCalls = 0
    sockets.push(this)
  }
  close() {
    this.closeCalls += 1
    this.readyState = 3
  }
}

beforeEach(() => {
  sockets = []
  vi.useFakeTimers()
  vi.stubGlobal('WebSocket', MockWS)
  // wsBase() joriy origin'ni ishlatadi — happy-dom'da mavjud.
  localStorage.setItem('darsly.access', 'tok-123')
})

afterEach(() => {
  vi.runOnlyPendingTimers()
  vi.useRealTimers()
  vi.unstubAllGlobals()
  localStorage.clear()
})

describe('connectRealtime — backoff jadvali', () => {
  // Bug: token URL'ga to'g'ri kodlanmasa (yoki umuman qo'shilmasa) authed
  // kanal 401 bilan yopiladi va foydalanuvchi real-time push'larni olmaydi.
  it('birinchi ulanishda token URL’ga kodlanadi', () => {
    connectRealtime(vi.fn())
    expect(sockets).toHaveLength(1)
    expect(sockets[0].url).toContain('/api/v1/ws?token=tok-123')
  })

  // Bug: token bo'lmasa ham WS ochilsa — ma'nosiz 401 loop.
  it('token yo‘q bo‘lsa ulanmaydi', () => {
    localStorage.clear()
    connectRealtime(vi.fn())
    expect(sockets).toHaveLength(0)
  })

  // Bug: backoff bo'lmasa (yoki 2^n emas) uzilishda server "reconnect bo'roni"
  // yeydi. Jadval AYNAN 1s, 2s, 4s, 8s bo'lishi kerak (ceiling'gacha).
  it('uzilishlarda 1s → 2s → 4s → 8s eksponensial kutadi', () => {
    connectRealtime(vi.fn())
    const reconnectAfter = (ms) => {
      const before = sockets.length
      sockets[before - 1].onclose()
      // Kutish tugamaguncha yangi socket OCHILMAYDI…
      vi.advanceTimersByTime(ms - 1)
      expect(sockets).toHaveLength(before)
      // …tugagach ochiladi.
      vi.advanceTimersByTime(1)
      expect(sockets).toHaveLength(before + 1)
    }
    reconnectAfter(1000) // 1000 * 2^0
    reconnectAfter(2000) // 2^1
    reconnectAfter(4000) // 2^2
    reconnectAfter(8000) // 2^3
  })

  // Bug: ceiling bo'lmasa 2^n cheksiz o'sadi va uzoq uzilishdan keyin qayta
  // ulanish daqiqalab kutdiradi — kanal amalda o'ladi. Cap = 15s.
  it('kutish 15s bilan cheklanadi (ceiling)', () => {
    connectRealtime(vi.fn())
    // 2^4=16s > 15s bo'lgunicha ko'p marta uzamiz.
    for (let i = 0; i < 6; i++) {
      const before = sockets.length
      sockets[before - 1].onclose()
      vi.advanceTimersByTime(15000)
      expect(sockets.length).toBeGreaterThanOrEqual(before + 1)
    }
  })

  // Bug: muvaffaqiyatli ulanish backoff'ni nolga qaytarmasa, keyingi uzilish
  // darhol emas, avvalgi katta kutish bilan boshlanadi.
  it('onopen backoff’ni tiklaydi — keyingi uzilish yana 1s dan boshlanadi', () => {
    connectRealtime(vi.fn())
    sockets[0].onclose()
    vi.advanceTimersByTime(1000) // 1-socket qayta ochildi (idx 1)
    sockets[1].onopen() // ulanish tikladi → retries=0
    sockets[1].onclose()
    // Yana 1s (2s emas) da ochilishi kerak.
    vi.advanceTimersByTime(999)
    expect(sockets).toHaveLength(2)
    vi.advanceTimersByTime(1)
    expect(sockets).toHaveLength(3)
  })

  // Bug: buzuq JSON kelsa onEvent throw qilib butun kanalni yiqitmasin.
  it('buzuq xabar e’tiborsiz qoldiriladi, to‘g‘risi onEvent’ga uzatiladi', () => {
    const onEvent = vi.fn()
    connectRealtime(onEvent)
    expect(() => sockets[0].onmessage({ data: '{buzuq' })).not.toThrow()
    expect(onEvent).not.toHaveBeenCalled()
    sockets[0].onmessage({ data: JSON.stringify({ type: 'x' }) })
    expect(onEvent).toHaveBeenCalledWith({ type: 'x' })
  })

  // Bug: stop() dan keyin ham qayta ulanish davom etsa — sahifadan chiqqan
  // foydalanuvchi uchun "arvoh" WS'lar ochilaveradi (resurs sızması).
  it('stop() dan keyin qayta ulanish TO‘XTAYDI', () => {
    const stop = connectRealtime(vi.fn())
    stop()
    expect(sockets[0].closeCalls).toBe(1)
    sockets[0].onclose() // stop dan keyingi close hech nima qilmasin
    vi.advanceTimersByTime(60000)
    expect(sockets).toHaveLength(1)
  })

  // Bug: onerror'da socket yopilmasa, "yarim ochiq" ulanish osilib qoladi.
  it('onerror socketni yopadi (yarim ochiq ulanish qolmasin)', () => {
    connectRealtime(vi.fn())
    sockets[0].onerror()
    expect(sockets[0].closeCalls).toBe(1)
  })
})

describe('connectWaitingRoom — backoff jadvali', () => {
  // Bug: guest kanali token EMAS, request_id bilan ochiladi (ochiq endpoint).
  it('request_id URL’ga kodlanadi', () => {
    connectWaitingRoom('req 1', vi.fn())
    expect(sockets[0].url).toContain('/api/v1/ws/waitingroom?request_id=req%201')
  })

  // Bug: guest kanali ceiling'i 10s bo'lishi kerak (authed'dan farqli) — aks
  // holda kutayotgan mehmon admit'ni juda kech oladi.
  it('uzilishda 1s → 2s → 4s kutadi', () => {
    connectWaitingRoom('req1', vi.fn())
    const reconnectAfter = (ms) => {
      const before = sockets.length
      sockets[before - 1].onclose()
      vi.advanceTimersByTime(ms - 1)
      expect(sockets).toHaveLength(before)
      vi.advanceTimersByTime(1)
      expect(sockets).toHaveLength(before + 1)
    }
    reconnectAfter(1000)
    reconnectAfter(2000)
    reconnectAfter(4000)
  })

  // Bug: guest ceiling 15s bo'lsa (authed bilan bir xil), kutayotgan mehmon
  // admit'ni 5s kechroq oladi. Cap AYNAN 10s: 2^n hech qachon 10s'dan oshmaydi.
  it('kutish 10s bilan cheklanadi (authed 15s dan past)', () => {
    connectWaitingRoom('req1', vi.fn())
    // 2^4 = 16s > 10s bo'lguncha ko'p marta uzamiz; har safar 10s da ochilishi kerak.
    for (let i = 0; i < 6; i++) {
      const before = sockets.length
      sockets[before - 1].onclose()
      vi.advanceTimersByTime(10000)
      expect(sockets.length).toBeGreaterThanOrEqual(before + 1)
    }
  })

  it('stop() qayta ulanishni to‘xtatadi', () => {
    const stop = connectWaitingRoom('req1', vi.fn())
    stop()
    expect(sockets[0].closeCalls).toBe(1)
    vi.advanceTimersByTime(60000)
    expect(sockets).toHaveLength(1)
  })
})

// tokenStore import qilingani ishlatiladi — lint no-unused'ni tinchlantirish
// va token o'qish yo'li aynan shu store ekanini hujjatlash uchun.
it('connectRealtime tokenni tokenStore’dan oladi', () => {
  tokenStore.set({ access_token: 'AAA', refresh_token: 'r' })
  connectRealtime(vi.fn())
  expect(sockets[0].url).toContain('token=AAA')
})
