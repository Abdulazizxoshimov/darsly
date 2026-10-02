import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { connectRealtime, connectWaitingRoom, HEARTBEAT_MS, JITTER_MS } from './ws'
import { refreshAccess, tokenStore } from '../api/api'

// F-2 — WS avtomatik qayta ulanish (eksponensial backoff + jitter).
//
// Nega kritik: bu jadval NOTO'G'RI bo'lsa ikki yomon holat bo'ladi —
//  · backoff yo'q/juda tez → uzilishda server "reconnect bo'roni" yeydi;
//  · ceiling yo'q → uzoq uzilishdan keyin qayta ulanish daqiqalab kutdiradi
//    (2^n cheksiz o'sadi) va real-time kanal amalda o'ladi.
// Timer'lar soxta — jadvalni deterministik o'lchaymiz, real kutmaymiz.

vi.mock('../api/api', async (orig) => ({
  ...(await orig()),
  refreshAccess: vi.fn(),
}))

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
    this.sent = []
    sockets.push(this)
  }
  send(data) {
    this.sent.push(data)
  }
  close() {
    this.closeCalls += 1
    this.readyState = 3
  }
}

// Muddati kelajakda bo'lgan / o'tgan imzosiz JWT (faqat `exp` o'qiladi).
const jwt = (expSec) => `h.${btoa(JSON.stringify({ exp: expSec })).replace(/=+$/, '')}.s`
const LIVE_TOKEN = jwt(Math.floor(Date.now() / 1000) + 3600)

// Uzilishdan keyin `ms` o'tgach yangi socket ochilishini tekshiradi.
async function expectReconnectAfter(ms) {
  const before = sockets.length
  sockets[before - 1].onclose()
  // Kutish tugamaguncha yangi socket OCHILMAYDI…
  await vi.advanceTimersByTimeAsync(ms - 1)
  expect(sockets).toHaveLength(before)
  // …tugagach ochiladi.
  await vi.advanceTimersByTimeAsync(1)
  expect(sockets).toHaveLength(before + 1)
}

beforeEach(() => {
  sockets = []
  vi.useFakeTimers()
  vi.stubGlobal('WebSocket', MockWS)
  // Jitter jadvalni deterministik o'lchash uchun o'chiriladi (alohida test bor).
  vi.spyOn(Math, 'random').mockReturnValue(0)
  refreshAccess.mockReset()
  refreshAccess.mockResolvedValue({ access: LIVE_TOKEN, revoked: false, network: false })
  // wsBase() joriy origin'ni ishlatadi — happy-dom'da mavjud.
  tokenStore.set({ access_token: LIVE_TOKEN, refresh_token: 'r' })
})

afterEach(() => {
  vi.runOnlyPendingTimers()
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  localStorage.clear()
})

describe('connectRealtime — backoff jadvali', () => {
  // Bug: token URL'ga to'g'ri kodlanmasa (yoki umuman qo'shilmasa) authed
  // kanal 401 bilan yopiladi va foydalanuvchi real-time push'larni olmaydi.
  it('birinchi ulanishda token URL’ga kodlanadi', () => {
    connectRealtime(vi.fn())
    expect(sockets).toHaveLength(1)
    expect(sockets[0].url).toContain(`/api/v1/ws?token=${encodeURIComponent(LIVE_TOKEN)}`)
  })

  // Bug: token bo'lmasa ham WS ochilsa — ma'nosiz 401 loop.
  it('token yo‘q bo‘lsa ulanmaydi', () => {
    localStorage.clear()
    connectRealtime(vi.fn())
    expect(sockets).toHaveLength(0)
  })

  // Bug: backoff bo'lmasa (yoki 2^n emas) uzilishda server "reconnect bo'roni"
  // yeydi. Jadval AYNAN 1s, 2s, 4s, 8s bo'lishi kerak (ceiling'gacha).
  it('uzilishlarda 1s → 2s → 4s → 8s eksponensial kutadi', async () => {
    connectRealtime(vi.fn())
    await expectReconnectAfter(1000) // 1000 * 2^0
    await expectReconnectAfter(2000) // 2^1
    await expectReconnectAfter(4000) // 2^2
    await expectReconnectAfter(8000) // 2^3
  })

  // Bug: ceiling bo'lmasa 2^n cheksiz o'sadi va uzoq uzilishdan keyin qayta
  // ulanish daqiqalab kutdiradi — kanal amalda o'ladi. Cap = 15s.
  it('kutish 15s bilan cheklanadi (ceiling)', async () => {
    connectRealtime(vi.fn())
    // 2^4=16s > 15s bo'lgunicha ko'p marta uzamiz.
    for (let i = 0; i < 6; i++) {
      const before = sockets.length
      sockets[before - 1].onclose()
      await vi.advanceTimersByTimeAsync(15000)
      expect(sockets.length).toBeGreaterThanOrEqual(before + 1)
    }
  })

  // Bug: jitter'siz ommaviy uzilishda (server restart) HAMMA aynan 1s da
  // birga uriladi. Kutishga [0, JITTER_MS) tasodifiy ulush qo'shiladi.
  it('kutishga jitter qo‘shiladi (eng ko‘pi JITTER_MS)', async () => {
    Math.random.mockReturnValue(0.999)
    connectRealtime(vi.fn())
    const extra = Math.floor(0.999 * JITTER_MS)
    await expectReconnectAfter(1000 + extra)
  })

  // Bug: muvaffaqiyatli ulanish backoff'ni nolga qaytarmasa, keyingi uzilish
  // darhol emas, avvalgi katta kutish bilan boshlanadi.
  it('onopen backoff’ni tiklaydi — keyingi uzilish yana 1s dan boshlanadi', async () => {
    connectRealtime(vi.fn())
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(1000) // 1-socket qayta ochildi (idx 1)
    sockets[1].onopen() // ulanish tikladi → failures=0
    sockets[1].onclose()
    // Yana 1s (2s emas) da ochilishi kerak.
    await vi.advanceTimersByTimeAsync(999)
    expect(sockets).toHaveLength(2)
    await vi.advanceTimersByTimeAsync(1)
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
  it('stop() dan keyin qayta ulanish TO‘XTAYDI', async () => {
    const stop = connectRealtime(vi.fn())
    stop()
    expect(sockets[0].closeCalls).toBe(1)
    sockets[0].onclose() // stop dan keyingi close hech nima qilmasin
    await vi.advanceTimersByTimeAsync(60000)
    expect(sockets).toHaveLength(1)
  })

  // Bug: onerror'da socket yopilmasa, "yarim ochiq" ulanish osilib qoladi.
  it('onerror socketni yopadi (yarim ochiq ulanish qolmasin)', () => {
    connectRealtime(vi.fn())
    sockets[0].onerror()
    expect(sockets[0].closeCalls).toBe(1)
  })
})

describe('connectRealtime — tarmoq/ko‘rinish qaytganda darhol', () => {
  // Brauzer uzgan socket: holat CLOSED, so'ng `close` hodisasi.
  const drop = (s) => {
    s.readyState = 3
    s.onclose()
  }

  // Bug: 15s kutish o'rtasida internet qaytsa ham kutib o'tirardi; fon tabda
  // taymerlar sekinlashib, ustoz tabga qaytganda admit so'rovlari kech kelardi.
  it('`online` hodisasi kutishni tashlab darhol qayta ulaydi', async () => {
    connectRealtime(vi.fn())
    drop(sockets[0])
    await vi.advanceTimersByTimeAsync(1000)
    drop(sockets[1]) // ikkinchi uzilish — 2s kutish
    await vi.advanceTimersByTimeAsync(500)
    expect(sockets).toHaveLength(2)
    window.dispatchEvent(new Event('online'))
    await vi.advanceTimersByTimeAsync(0)
    expect(sockets).toHaveLength(3)
    // Eski taymer o'chirilgan — undan ikkinchi socket ochilmaydi.
    await vi.advanceTimersByTimeAsync(5000)
    expect(sockets).toHaveLength(3)
  })

  it('tab ko‘rinib qolganda ham (visibilitychange) darhol qayta ulaydi', async () => {
    connectRealtime(vi.fn())
    drop(sockets[0])
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(0)
    expect(sockets).toHaveLength(2)
  })

  // Kutish taymeri bo'lmasa (ulanish o'zi ochilmoqda) `online` hech nima qilmaydi.
  it('qayta ulanish kutilmayotgan bo‘lsa `online` e’tiborsiz', async () => {
    connectRealtime(vi.fn())
    window.dispatchEvent(new Event('online'))
    await vi.advanceTimersByTimeAsync(0)
    expect(sockets).toHaveLength(1)
  })

  // Bug: ulanish OCHIQ turganda `online` kelsa ikkinchi socket ochilmasin.
  it('ulanish ochiq bo‘lsa `online` hech nima qilmaydi', async () => {
    connectRealtime(vi.fn())
    sockets[0].readyState = 1
    sockets[0].onopen()
    window.dispatchEvent(new Event('online'))
    await vi.advanceTimersByTimeAsync(0)
    expect(sockets).toHaveLength(1)
  })
})

describe('connectRealtime — heartbeat', () => {
  // Oraliq proksi/NAT jim ulanishni yopmasin — ochiq ulanishda davriy ping.
  it('ochiq ulanishda har HEARTBEAT_MS da ping yuboriladi, yopilgach to‘xtaydi', async () => {
    connectRealtime(vi.fn())
    sockets[0].readyState = 1
    sockets[0].onopen()
    await vi.advanceTimersByTimeAsync(HEARTBEAT_MS * 2)
    expect(sockets[0].sent).toEqual(['{"type":"ping"}', '{"type":"ping"}'])
    sockets[0].readyState = 3
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(HEARTBEAT_MS)
    expect(sockets[0].sent).toHaveLength(2)
  })
})

describe('connectRealtime — token yangilash', () => {
  // Bug: access token muddati o'tgach WS 401 bilan yopiladi, brauzer esa
  // 401 ni ko'rmaydi (close 1006) — kanal o'lik token bilan abadiy urinardi.
  it('token muddati o‘tgan bo‘lsa qayta ulanishdan OLDIN refresh qilinadi', async () => {
    tokenStore.set({ access_token: jwt(1), refresh_token: 'r' })
    refreshAccess.mockImplementation(async () => {
      tokenStore.set({ access_token: LIVE_TOKEN, refresh_token: 'r2' })
      return { access: LIVE_TOKEN, revoked: false, network: false }
    })
    connectRealtime(vi.fn())
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(1000)
    expect(refreshAccess).toHaveBeenCalledTimes(1)
    expect(sockets[1].url).toContain(encodeURIComponent(LIVE_TOKEN))
  })

  it('muddati o‘tmagan token bilan birinchi uzilishda refresh SO‘RALMAYDI', async () => {
    connectRealtime(vi.fn())
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(1000)
    expect(refreshAccess).not.toHaveBeenCalled()
    expect(sockets).toHaveLength(2)
  })

  // Sessiya serverda o'lgan bo'lishi mumkin (token hali "yaroqli" ko'rinsa
  // ham) — ketma-ket ikki yiqilishdan keyin bir marta yangilab ko'riladi.
  it('ketma-ket ikki yiqilishdan keyin refresh urinib ko‘riladi', async () => {
    refreshAccess.mockResolvedValue({ access: LIVE_TOKEN, revoked: false, network: false })
    connectRealtime(vi.fn())
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(1000)
    sockets[1].onclose()
    await vi.advanceTimersByTimeAsync(2000)
    expect(refreshAccess).toHaveBeenCalledTimes(1)
    expect(sockets).toHaveLength(3)
  })

  // Bug: sessiya bekor qilingan (boshqa qurilmada kirildi) — kanal o'lik
  // refresh bilan abadiy urinmasin; HTTP qatlami foydalanuvchini o'zi chiqaradi.
  it('refresh sessiya o‘lganini aytsa kanal TO‘XTAYDI', async () => {
    tokenStore.set({ access_token: jwt(1), refresh_token: 'r' })
    refreshAccess.mockResolvedValue({ access: null, revoked: true, network: false })
    connectRealtime(vi.fn())
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(60000)
    expect(sockets).toHaveLength(1)
  })

  // Tarmoq sababli refresh yetib bormasa — sessiya haqida hech narsa ma'lum
  // emas, urinish davom etadi (internet qaytganda refresh ham o'tadi).
  it('refresh tarmoq xatosi bersa urinish davom etadi', async () => {
    tokenStore.set({ access_token: jwt(1), refresh_token: 'r' })
    refreshAccess.mockResolvedValue({ access: null, revoked: false, network: true })
    connectRealtime(vi.fn())
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(1000)
    expect(sockets).toHaveLength(2)
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
  it('uzilishda 1s → 2s → 4s kutadi', async () => {
    connectWaitingRoom('req1', vi.fn())
    await expectReconnectAfter(1000)
    await expectReconnectAfter(2000)
    await expectReconnectAfter(4000)
  })

  // Bug: guest ceiling 15s bo'lsa (authed bilan bir xil), kutayotgan mehmon
  // admit'ni 5s kechroq oladi. Cap AYNAN 10s: 2^n hech qachon 10s'dan oshmaydi.
  it('kutish 10s bilan cheklanadi (authed 15s dan past)', async () => {
    connectWaitingRoom('req1', vi.fn())
    // 2^4 = 16s > 10s bo'lguncha ko'p marta uzamiz; har safar 10s da ochilishi kerak.
    for (let i = 0; i < 6; i++) {
      const before = sockets.length
      sockets[before - 1].onclose()
      await vi.advanceTimersByTimeAsync(10000)
      expect(sockets.length).toBeGreaterThanOrEqual(before + 1)
    }
  })

  it('stop() qayta ulanishni to‘xtatadi', async () => {
    const stop = connectWaitingRoom('req1', vi.fn())
    stop()
    expect(sockets[0].closeCalls).toBe(1)
    await vi.advanceTimersByTimeAsync(60000)
    expect(sockets).toHaveLength(1)
  })

  // Mehmon kanalida sessiya yo'q — refresh UMUMAN chaqirilmaydi.
  it('mehmon kanali refresh qilmaydi', async () => {
    connectWaitingRoom('req1', vi.fn())
    sockets[0].onclose()
    await vi.advanceTimersByTimeAsync(1000)
    sockets[1].onclose()
    await vi.advanceTimersByTimeAsync(2000)
    expect(refreshAccess).not.toHaveBeenCalled()
  })
})
