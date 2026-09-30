import { refreshAccess, tokenStore } from '../api/api'
import { jwtExpired } from './jwt'

// WebSocket bazaviy URL. Dev'da VITE_API_URL bo'sh → joriy origin (Vite proxy /api ws:true).
function wsBase() {
  const base = import.meta.env.VITE_API_URL
  if (base) return base.replace(/^http/, 'ws')
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${window.location.host}`
}

/** Qayta ulanish kutishiga qo'shiladigan tasodifiy ulush (yuqori chegara). */
export const JITTER_MS = 1_000
/**
 * Ilova darajasidagi "ping" oralig'i. Brauzer WS ping-kadrini yubora olmaydi;
 * server o'zi ping yuboradi va o'lik klientni aniqlaydi (`websocket.go`), bu
 * xabar esa ORALIQ vositalar (Caddy/NAT) uchun — jim ulanishni uzmasin.
 * Server uni o'qib tashlab yuboradi.
 */
export const HEARTBEAT_MS = 25_000
/** Ketma-ket shuncha muvaffaqiyatsiz urinishdan keyin token yangilanib ko'riladi. */
const REFRESH_AFTER_FAILURES = 2

/**
 * Qayta ulanuvchi kanal — authed va mehmon kanallarining UMUMIY mexanizmi
 * (avval ikkalasi ~90% bir xil kod bilan alohida yozilgan edi).
 *
 *   · eksponensial backoff (1s·2^n, `maxDelayMs` gacha) + jitter — ommaviy
 *     uzilishda hamma bir soniyada birga urilmasin ("reconnect bo'roni");
 *   · `online` va tab ko'rinishi qaytganda kutmasdan darhol qayta ulanish
 *     (fon tabda brauzer taymerlarni sekinlashtiradi — mentor tabga qaytganda
 *     admit so'rovlarini darhol ko'rishi kerak);
 *   · heartbeat — oraliq proksi/NAT jim ulanishni yopib qo'ymasin;
 *   · `beforeReconnect(failures)` — urinishdan oldin tekshiruv; `false`
 *     qaytarsa kanal butunlay to'xtaydi (masalan sessiya bekor qilingan).
 */
function createChannel({ url, maxDelayMs, onEvent, beforeReconnect }) {
  let ws = null
  let failures = 0
  let closed = false
  let timer = null
  let heartbeat = null

  function stopHeartbeat() {
    if (heartbeat) clearInterval(heartbeat)
    heartbeat = null
  }
  function startHeartbeat() {
    stopHeartbeat()
    heartbeat = setInterval(() => {
      if (ws && ws.readyState === 1) ws.send('{"type":"ping"}')
    }, HEARTBEAT_MS)
  }

  function open() {
    if (closed) return
    const u = url()
    if (!u) return
    ws = new WebSocket(u)
    ws.onopen = () => {
      failures = 0
      startHeartbeat()
    }
    ws.onmessage = (ev) => {
      try {
        onEvent(JSON.parse(ev.data))
      } catch {
        /* e'tiborsiz */
      }
    }
    ws.onclose = () => {
      stopHeartbeat()
      if (closed) return
      schedule()
    }
    ws.onerror = () => ws && ws.close()
  }

  function schedule() {
    const delay = Math.min(maxDelayMs, 1000 * 2 ** failures) + Math.floor(Math.random() * JITTER_MS)
    failures += 1
    timer = setTimeout(reconnect, delay)
  }

  async function reconnect() {
    timer = null
    if (closed) return
    if (beforeReconnect && (await beforeReconnect(failures)) === false) {
      closed = true
      return
    }
    open()
  }

  // Tarmoq/ko'rinish qaytdi — kutishni tashlab darhol urinamiz. Ochiq yoki
  // ochilayotgan ulanish bo'lsa tegmaymiz.
  function wake() {
    if (closed || !timer) return
    if (ws && (ws.readyState === 0 || ws.readyState === 1)) return
    clearTimeout(timer)
    reconnect()
  }
  const onVisible = () => document.visibilityState === 'visible' && wake()
  window.addEventListener('online', wake)
  document.addEventListener('visibilitychange', onVisible)

  open()

  return () => {
    closed = true
    window.removeEventListener('online', wake)
    document.removeEventListener('visibilitychange', onVisible)
    if (timer) clearTimeout(timer)
    stopHeartbeat()
    if (ws) ws.close()
  }
}

// Authed foydalanuvchi real-time kanali (/api/v1/ws?token=).
// onEvent(msg) chaqiriladi. Xabar: { type, room?, payload, created_at }.
//
// Brauzer 401 ni ko'rmaydi (upgrade'dan oldingi HTTP javobi `close` kodi
// 1006 bo'lib keladi), shuning uchun token MUDDATI o'zimizda tekshiriladi:
// muddati o'tgan bo'lsa yoki ketma-ket ikki urinish yiqilsa access token
// `refreshAccess` bilan (HTTP qatlami bilan BIR XIL, dedup'langan yo'l)
// yangilanadi. Sessiya bekor qilingan/o'lgan bo'lsa kanal to'xtaydi — keyingi
// HTTP so'rov foydalanuvchini o'zi chiqaradi.
export function connectRealtime(onEvent) {
  return createChannel({
    url: () => {
      const token = tokenStore.access
      return token ? `${wsBase()}/api/v1/ws?token=${encodeURIComponent(token)}` : null
    },
    maxDelayMs: 15000,
    onEvent,
    beforeReconnect: async (failures) => {
      const token = tokenStore.access
      if (!token) return false
      if (!jwtExpired(token) && failures < REFRESH_AFTER_FAILURES) return true
      const r = await refreshAccess()
      if (r.access || r.network) return true
      return false
    },
  })
}

// Guest kutish-xonasi kanali (/api/v1/ws/waitingroom?request_id=).
// Cap 10s — authed'dan past: kutayotgan mehmon admit'ni kech olmasin.
export function connectWaitingRoom(requestId, onEvent) {
  return createChannel({
    url: () => `${wsBase()}/api/v1/ws/waitingroom?request_id=${encodeURIComponent(requestId)}`,
    maxDelayMs: 10000,
    onEvent,
  })
}
