// Darsly API client — fetch wrapper.
// - base URL .env (VITE_API_URL); dev'da bo'sh → Vite proxy /api → :8087
// - har so'rovga JWT (Authorization: Bearer)
// - 401 → refresh bilan bir marta yangilash; bo'lmasa logout + /auth
// - backend xato formatini ({code,message}) parse qiladi, o'zbekcha matnga map

const BASE = (import.meta.env.VITE_API_URL || '') + '/api/v1'

const ACCESS_KEY = 'darsly.access'
const REFRESH_KEY = 'darsly.refresh'

export const tokenStore = {
  get access() {
    return localStorage.getItem(ACCESS_KEY)
  },
  get refresh() {
    return localStorage.getItem(REFRESH_KEY)
  },
  set({ access_token, refresh_token }) {
    localStorage.setItem(ACCESS_KEY, access_token)
    localStorage.setItem(REFRESH_KEY, refresh_token)
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
  get isAuthed() {
    return !!localStorage.getItem(ACCESS_KEY)
  },
}

// Backend xato kodini o'zbekcha matnga map qiladi.
const ERROR_UZ = {
  UNAUTHORIZED: 'Email yoki parol noto‘g‘ri',
  FORBIDDEN: 'Bu amalga ruxsatingiz yo‘q',
  NOT_FOUND: 'Topilmadi',
  BAD_REQUEST: 'So‘rovda xatolik',
  CONFLICT: 'Bu ma’lumot allaqachon mavjud',
  INTERNAL_ERROR: 'Serverda xatolik yuz berdi',
  VALIDATION_ERROR: 'Kiritilgan ma’lumotlar noto‘g‘ri',
  // Backend kodlari bilan aynan mos bo'lishi shart (internal/pkg/errors, api/middleware).
  RATE_LIMITED: 'Juda ko‘p urinish — biroz kuting',
  AUTHZ_UNAVAILABLE: 'Xizmat vaqtincha ishlamayapti — birozdan so‘ng urinib ko‘ring',
  TOKEN_EXPIRED: 'Sessiya tugadi — qaytadan kiring',
  TOKEN_INVALID: 'Sessiya yaroqsiz — qaytadan kiring',
}

export class ApiError extends Error {
  constructor(code, message, status) {
    super(message)
    this.code = code
    this.status = status
  }
}

// Xatoni foydalanuvchiga ko'rsatiladigan o'zbekcha matnga aylantiradi.
export function errorText(err, fallback = 'Xatolik yuz berdi') {
  if (err instanceof ApiError) {
    return ERROR_UZ[err.code] || err.message || fallback
  }
  if (err && err.name === 'TypeError') return 'Serverga ulanib bo‘lmadi'
  return fallback
}

// Logout callback — AppContext o'rnatadi (aylanma import'siz).
let onUnauthorized = () => {
  tokenStore.clear()
  if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/auth')) {
    window.location.href = '/auth'
  }
}
export function setUnauthorizedHandler(fn) {
  onUnauthorized = fn
}

// Bir vaqtda ko'p 401 → faqat bitta refresh so'rovi.
let refreshing = null
async function refreshTokens() {
  const refresh_token = tokenStore.refresh
  if (!refresh_token) return null
  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token }),
    })
    if (!res.ok) throw new Error('refresh failed')
    const json = await res.json()
    tokenStore.set(json.data)
    return json.data.access_token
  } catch {
    return null
  }
}

async function parseError(res) {
  let code = 'INTERNAL_ERROR'
  let message = 'Xatolik'
  try {
    const body = await res.json()
    code = body.code || code
    message = body.message || message
  } catch {
    /* bo'sh body */
  }
  return new ApiError(code, message, res.status)
}

// Asosiy so'rov. auth=false — public endpoint (token qo'shilmaydi).
async function request(method, path, { body, auth = true, raw = false, _retried = false } = {}) {
  const headers = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (auth && tokenStore.access) headers.Authorization = `Bearer ${tokenStore.access}`

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  // 401 → refresh + qayta urinish (bir marta)
  if (res.status === 401 && auth && !_retried) {
    refreshing = refreshing ?? refreshTokens()
    const newToken = await refreshing
    refreshing = null
    if (newToken) {
      return request(method, path, { body, auth, raw, _retried: true })
    }
    onUnauthorized()
    throw new ApiError('UNAUTHORIZED', 'Sessiya tugadi', 401)
  }

  if (res.status === 204) return null
  if (!res.ok) throw await parseError(res)

  const json = await res.json()
  if (raw) return json // { data, total, page, limit, total_pages }
  return json.data // { data: T } → T
}

export const api = {
  get: (path, opts) => request('GET', path, opts),
  post: (path, body, opts) => request('POST', path, { body, ...opts }),
  patch: (path, body, opts) => request('PATCH', path, { body, ...opts }),
  put: (path, body, opts) => request('PUT', path, { body, ...opts }),
  del: (path, opts) => request('DELETE', path, opts),
  // ro'yxat konverti ({data,total,...}) uchun
  list: (path, opts) => request('GET', path, { raw: true, ...opts }),
}
