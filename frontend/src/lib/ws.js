import { tokenStore } from '../api/api'

// WebSocket bazaviy URL. Dev'da VITE_API_URL bo'sh → joriy origin (Vite proxy /api ws:true).
function wsBase() {
  const base = import.meta.env.VITE_API_URL
  if (base) return base.replace(/^http/, 'ws')
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${window.location.host}`
}

// Authed foydalanuvchi real-time kanali (/api/v1/ws?token=).
// Avtomatik qayta ulanish (exponential backoff, max 15s). onEvent(msg) chaqiriladi.
// Xabar: { type, room?, payload, created_at }.
export function connectRealtime(onEvent) {
  let ws = null
  let retries = 0
  let closed = false
  let timer = null

  function open() {
    const token = tokenStore.access
    if (!token || closed) return

    ws = new WebSocket(`${wsBase()}/api/v1/ws?token=${encodeURIComponent(token)}`)

    ws.onopen = () => {
      retries = 0
    }
    ws.onmessage = (ev) => {
      try {
        onEvent(JSON.parse(ev.data))
      } catch {
        /* e'tiborsiz */
      }
    }
    ws.onclose = () => {
      if (closed) return
      const delay = Math.min(15000, 1000 * 2 ** retries)
      retries += 1
      timer = setTimeout(open, delay)
    }
    ws.onerror = () => ws && ws.close()
  }

  open()

  return () => {
    closed = true
    if (timer) clearTimeout(timer)
    if (ws) ws.close()
  }
}

// Guest kutish-xonasi kanali (/api/v1/ws/waitingroom?request_id=).
export function connectWaitingRoom(requestId, onEvent) {
  let ws = null
  let retries = 0
  let closed = false
  let timer = null

  function open() {
    if (closed) return
    ws = new WebSocket(`${wsBase()}/api/v1/ws/waitingroom?request_id=${encodeURIComponent(requestId)}`)
    ws.onopen = () => {
      retries = 0
    }
    ws.onmessage = (ev) => {
      try {
        onEvent(JSON.parse(ev.data))
      } catch {
        /* e'tiborsiz */
      }
    }
    ws.onclose = () => {
      if (closed) return
      const delay = Math.min(10000, 1000 * 2 ** retries)
      retries += 1
      timer = setTimeout(open, delay)
    }
    ws.onerror = () => ws && ws.close()
  }
  open()
  return () => {
    closed = true
    if (timer) clearTimeout(timer)
    if (ws) ws.close()
  }
}
