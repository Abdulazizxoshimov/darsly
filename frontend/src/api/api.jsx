// Jonly API client — fetch wrapper.
// - base URL .env (VITE_API_URL); dev'da bo'sh → Vite proxy /api → :8087
// - har so'rovga JWT (Authorization: Bearer)
// - 401 → refresh bilan bir marta yangilash; bo'lmasa logout + /auth
// - backend xato formatini ({code,message}) parse qiladi, o'zbekcha matnga map

import { setLogoutReason } from '../lib/logoutReason'

const BASE = (import.meta.env.VITE_API_URL || '') + '/api/v1'

const ACCESS_KEY = 'jonly.access'
const REFRESH_KEY = 'jonly.refresh'

// Rebrand (Darsly → Jonly) dan oldingi kalitlar. Bir marta ko'chiriladi —
// aks holda deploy'dan keyin HAR BIR foydalanuvchi tizimdan chiqib ketardi.
const LEGACY_KEYS = { 'darsly.access': ACCESS_KEY, 'darsly.refresh': REFRESH_KEY }
function migrateLegacyKeys() {
  try {
    for (const [from, to] of Object.entries(LEGACY_KEYS)) {
      const v = localStorage.getItem(from)
      if (v !== null) {
        if (localStorage.getItem(to) === null) localStorage.setItem(to, v)
        localStorage.removeItem(from)
      }
    }
  } catch {
    /* storage yopiq — eski sessiya shunchaki tiklanmaydi */
  }
}
migrateLegacyKeys()

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
  // Bitta akkaunt = bitta faol sessiya. Odatda «boshqa qurilmada kirildi»
  // degani (logout/parol tiklash ham shu kodni beradi — harakat bir xil).
  SESSION_REVOKED: 'Boshqa qurilmada kirildi — qaytadan kiring',
  // Klient tomonda yasaladi (`REQUEST_TIMEOUT_MS`), backend kodi emas.
  TIMEOUT: 'Internet sekin — so‘rov bajarilmadi, qayta urinib ko‘ring',
  NETWORK: 'Serverga ulanib bo‘lmadi',
  ABORTED: 'Bekor qilindi',
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
// `reason`: 'session_revoked' | 'expired' — login sahifasi shuni ko'rsatadi.
let onUnauthorized = (reason) => {
  tokenStore.clear()
  setLogoutReason(reason)
  if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/auth')) {
    window.location.href = '/auth'
  }
}
export function setUnauthorizedHandler(fn) {
  onUnauthorized = fn
}

// Qaytadi: `{ access, revoked, network }`
//   access  — yangi access token (muvaffaqiyat) yoki null;
//   revoked — sessiya BEKOR qilingan (boshqa qurilmada kirilgan): qayta
//             urinishning ma'nosi yo'q, foydalanuvchiga aniq sabab aytiladi;
//   network — serverga YETIB BORMADI (offline, timeout, 5xx): sessiya haqida
//             hech narsa ma'lum emas. Avval bu ham "expired" deb hisoblanib
//             foydalanuvchi tizimdan chiqarilardi — metro/lift uzilishida
//             dars boshlanishida parol so'rash eng yomon payt.
async function refreshTokens() {
  const refresh_token = tokenStore.refresh
  if (!refresh_token) return { access: null, revoked: false, network: false }
  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token }),
    })
    if (res.status >= 500) return { access: null, revoked: false, network: true }
    if (!res.ok) {
      let code = ''
      try {
        code = (await res.json())?.code || ''
      } catch {
        /* bo'sh body */
      }
      return { access: null, revoked: code === 'SESSION_REVOKED', network: false }
    }
    const json = await res.json()
    tokenStore.set(json.data)
    return { access: json.data.access_token, revoked: false, network: false }
  } catch {
    return { access: null, revoked: false, network: true }
  }
}

// Bir vaqtda ko'p 401 → faqat bitta refresh so'rovi. WS kanali ham shu
// yo'ldan yangilaydi (`lib/ws.js`) — ikkita parallel refresh bo'lmasin.
let refreshing = null
export function refreshAccess() {
  if (!refreshing) {
    refreshing = refreshTokens().finally(() => {
      refreshing = null
    })
  }
  return refreshing
}

// 401 dan keyingi YAGONA qaror yo'li (`request` ham, `upload` ham shu yerdan
// o'tadi — avval ikki nusxa edi va ular bir-biridan sekin farqlanib ketardi).
// Qaytadi: `true` — token yangilandi, so'rovni BIR MARTA takrorlash mumkin.
// Aks holda mos xatoni tashlaydi (kerak bo'lsa tizimdan chiqarib).
async function recoverUnauthorized(err) {
  // Sessiya BEKOR qilingan bo'lsa refresh ham 401 beradi — bekorga
  // urinmaymiz va sababni saqlab qolamiz (login sahifasi ko'rsatadi).
  if (err.code === 'SESSION_REVOKED') {
    onUnauthorized('session_revoked')
    throw err
  }
  const r = await refreshAccess()
  if (r.access) return true
  // Tarmoq: sessiya tirik bo'lishi mumkin — chiqarmaymiz, keyingi so'rov
  // o'zi qayta urinib ko'radi.
  if (r.network) throw new ApiError('NETWORK', 'Serverga ulanib bo‘lmadi', 0)
  onUnauthorized(r.revoked ? 'session_revoked' : 'expired')
  throw r.revoked
    ? new ApiError('SESSION_REVOKED', 'Boshqa qurilmada kirildi', 401)
    : new ApiError('UNAUTHORIZED', 'Sessiya tugadi', 401)
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

// So'rov muddati. `fetch` ning O'ZIDA timeout YO'Q: mobil tarmoq "yarim
// ochiq" holatda qolsa (signal bor, ma'lumot yo'q — metro, lift, zaif Wi-Fi)
// promise MINUTLAB osilib turadi. Foydalanuvchi uchun bu "tugma ishlamayapti"
// degani: spinner aylanaveradi va hech qanday xato ham chiqmaydi.
//
// 20 soniya — eng sekin real so'rovdan (yozuvlar ro'yxati) ancha uzun, lekin
// odam kutishga tayyor bo'lgan vaqtdan qisqa.
const REQUEST_TIMEOUT_MS = 20_000

// `Content-Disposition: attachment; filename="a-chat.txt"` → `a-chat.txt`.
//
// Fayl nomini SERVER aytadi (dars sarlavhasidan yasalgan, ASCII'ga tozalangan) —
// klient uni o'zi to'qisa transkript fayllari boshqa nom bilan tushar edi.
// `filename*=UTF-8''…` (RFC 5987) shakli ham qabul qilinadi.
export function filenameFromDisposition(header, fallback = 'fayl') {
  const h = String(header || '')
  const star = /filename\*=UTF-8''([^;]+)/i.exec(h)
  if (star) {
    try {
      return decodeURIComponent(star[1].trim())
    } catch {
      /* buzuq kodlash — oddiy shaklga tushamiz */
    }
  }
  const plain = /filename="?([^";]+)"?/i.exec(h)
  return plain ? plain[1].trim() : fallback
}

// Asosiy so'rov. auth=false — public endpoint (token qo'shilmaydi).
//
// `blob: true` — javob JSON emas, FAYL (chat transkripti). Bunday endpoint'ni
// oddiy `<a href>` bilan ochib bo'lmaydi: u himoyalangan va `Authorization`
// header'ini talab qiladi, brauzer esa havolaga header qo'sha olmaydi.
// Shuning uchun fayl shu yerdan — 401/refresh zanjiri bilan BIR XIL yo'ldan —
// olinadi va chaqiruvchi uni `saveBlob` bilan saqlaydi.
async function request(method, path, { body, auth = true, raw = false, blob = false, _retried = false } = {}) {
  const headers = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (auth && tokenStore.access) headers.Authorization = `Bearer ${tokenStore.access}`

  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS)
  let res
  try {
    res = await fetch(`${BASE}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: ctrl.signal,
    })
  } catch (e) {
    // Abort'ni tarmoq xatosidan ajratamiz — foydalanuvchiga ko'rsatiladigan
    // matn boshqacha ("sekin" va "ulanib bo'lmadi" bir narsa emas).
    if (e?.name === 'AbortError') {
      throw new ApiError('TIMEOUT', 'So‘rov juda uzoq davom etdi', 0)
    }
    throw e
  } finally {
    clearTimeout(timer)
  }

  // 401 → refresh + qayta urinish (bir marta)
  if (res.status === 401 && auth && !_retried) {
    await recoverUnauthorized(await parseError(res))
    return request(method, path, { body, auth, raw, blob, _retried: true })
  }

  if (res.status === 204) return null
  if (!res.ok) throw await parseError(res)

  if (blob) {
    return {
      blob: await res.blob(),
      // Zaxira nomi ATAYLAB bo'sh: header o'qilmasa (cross-origin'da server
      // `Access-Control-Expose-Headers` bermasa) chaqiruvchi O'ZINING mazmunli
      // nomini qo'ya olsin. `'fayl'` qaytarilsa u truthy bo'lib, chaqiruvchining
      // `filename || 'dars-chat.txt'` zaxirasi hech qachon ishlamas edi va fayl
      // kengaytmasiz tushardi.
      filename: filenameFromDisposition(res.headers?.get?.('Content-Disposition'), ''),
    }
  }

  const json = await res.json()
  if (raw) return json // { data, total, page, limit, total_pages }
  return json.data // { data: T } → T
}

// ── Fayl yuklash (multipart) ────────────────────────────────────────────────
//
// Nega `fetch` EMAS, `XMLHttpRequest`: fetch'da YUKLASH progressi umuman yo'q
// (`ReadableStream` request body brauzerlarda hali cheklangan). 20 MB fayl
// O'zbekistondagi mobil internetda bir necha o'nlab soniya ketadi — progresssiz
// bu "ilova qotib qoldi" degan taassurot beradi. Bu YAGONA XHR joyi va u shu
// yerdan tashqariga chiqmaydi: chaqiruvchi oddiy Promise ko'radi.
//
// Diqqat: `Content-Type` QO'LDA qo'yilmaydi — brauzer `multipart/form-data`
// chegarasini (boundary) o'zi yozadi, biz yozsak server tanani parse qila olmaydi.
const UPLOAD_TIMEOUT_MS = 120_000

function sendUpload(path, formData, { auth = true, onProgress, signal } = {}) {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) return reject(new ApiError('ABORTED', 'Bekor qilindi', 0))
    const xhr = new XMLHttpRequest()
    xhr.open('POST', `${BASE}${path}`)
    xhr.timeout = UPLOAD_TIMEOUT_MS
    if (auth && tokenStore.access) xhr.setRequestHeader('Authorization', `Bearer ${tokenStore.access}`)

    if (onProgress && xhr.upload) {
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) onProgress(Math.min(99, Math.round((e.loaded / e.total) * 100)))
      }
    }

    const onAbort = () => xhr.abort()
    signal?.addEventListener('abort', onAbort)
    const done = () => signal?.removeEventListener('abort', onAbort)

    xhr.onload = () => {
      done()
      let body = null
      try {
        body = JSON.parse(xhr.responseText)
      } catch {
        /* bo'sh yoki JSON bo'lmagan javob */
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        if (onProgress) onProgress(100)
        return resolve(body?.data ?? null)
      }
      reject(new ApiError(body?.code || 'INTERNAL_ERROR', body?.message || 'Xatolik', xhr.status))
    }
    xhr.onerror = () => {
      done()
      reject(new ApiError('NETWORK', 'Serverga ulanib bo‘lmadi', 0))
    }
    xhr.ontimeout = () => {
      done()
      reject(new ApiError('TIMEOUT', 'So‘rov juda uzoq davom etdi', 0))
    }
    xhr.onabort = () => {
      done()
      reject(new ApiError('ABORTED', 'Bekor qilindi', 0))
    }
    xhr.send(formData)
  })
}

// Yuklash ham `request()` BILAN BIR XIL 401 zanjiridan o'tadi.
//
// Busiz xatti-harakat izchil bo'lmasdi: access token muddati tugagan payt
// fayl yuborilsa foydalanuvchi "Faylni yuborib bo'lmadi" degan matnni ko'rib,
// o'lik sessiya bilan qolib ketardi — boshqa har bir so'rov esa o'zini
// jimgina yangilab ishlayverardi. `FormData` ni qayta yuborish xavfsiz:
// har urinishda yangi XHR ochiladi.
async function upload(path, formData, opts = {}) {
  const { auth = true, _retried = false } = opts
  try {
    return await sendUpload(path, formData, opts)
  } catch (e) {
    if (!(e instanceof ApiError) || e.status !== 401 || !auth || _retried) throw e
    await recoverUnauthorized(e)
    return upload(path, formData, { ...opts, _retried: true })
  }
}

export const api = {
  get: (path, opts) => request('GET', path, opts),
  post: (path, body, opts) => request('POST', path, { body, ...opts }),
  patch: (path, body, opts) => request('PATCH', path, { body, ...opts }),
  put: (path, body, opts) => request('PUT', path, { body, ...opts }),
  del: (path, opts) => request('DELETE', path, opts),
  // ro'yxat konverti ({data,total,...}) uchun
  list: (path, opts) => request('GET', path, { raw: true, ...opts }),
  // fayl javobi ({blob, filename}) — himoyalangan yuklab olish
  blob: (path, opts) => request('GET', path, { blob: true, ...opts }),
  // multipart (fayl) — progress bilan
  upload,
}
