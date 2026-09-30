import { createContext, useContext, useEffect, useMemo, useState, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { tokenStore, setUnauthorizedHandler, ApiError, errorText } from '../api/api'
import { me, login as apiLogin, register as apiRegister, logout as apiLogout } from '../api/auth'
import { connectRealtime } from '../lib/ws'
import { setLogoutReason } from '../lib/logoutReason'
import { toast } from '../lib/toast'
import { qk } from './data'

const AppContext = createContext(null)

// Bootstrap (`me()`) tarmoq sababli yiqilsa qayta urinish oralig'i: 3s → 6s →
// 12s → 24s → 30s (cap). Avval qayta urinish UMUMAN yo'q edi: `authed=true`,
// `user=null` holat qolib ketardi va foydalanuvchi sahifani qo'lda
// yangilamaguncha rol/ism yo'q "yarim kirgan" ilovani ko'rardi.
export const BOOTSTRAP_RETRY_BASE_MS = 3_000
export const BOOTSTRAP_RETRY_MAX_MS = 30_000
export function bootstrapRetryDelay(attempt) {
  return Math.min(BOOTSTRAP_RETRY_MAX_MS, BOOTSTRAP_RETRY_BASE_MS * 2 ** attempt)
}

export function AppProvider({ children }) {
  const qc = useQueryClient()
  const [user, setUser] = useState(null)
  const [ready, setReady] = useState(!tokenStore.isAuthed) // token yo'q → darrov tayyor

  // 401 → sessiyani tozalash (RequireAuth /auth ga yo'naltiradi).
  // Sabab (`session_revoked` / `expired`) sessionStorage'ga yoziladi va login
  // sahifasi uni ko'rsatadi — bu yerda to'g'ridan-to'g'ri ko'rsatib bo'lmaydi,
  // chunki keyingi qatorda sahifa butunlay qayta yuklanadi.
  useEffect(() => {
    setUnauthorizedHandler((reason) => {
      tokenStore.clear()
      setLogoutReason(reason)
      setUser(null)
      if (!window.location.pathname.startsWith('/auth')) window.location.href = '/auth'
    })
  }, [])

  // Boshlang'ich: token bo'lsa foydalanuvchini yuklaymiz.
  useEffect(() => {
    if (!tokenStore.isAuthed) {
      // Token yo'q — ilova bir martalik yuklanishida darhol "tayyor" (bootstrap).
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bir martalik bootstrap
      setReady(true)
      return
    }
    let alive = true
    let timer = null
    let attempt = 0

    const load = () => {
      me()
        .then((u) => {
          if (alive) setUser(u)
        })
        .catch((e) => {
          if (!alive) return
          // Sessiyani FAQAT token haqiqatan rad etilganda tozalaymiz.
          //
          // 401 `api.jsx` da allaqachon refresh bilan bir marta qayta
          // urinilgan; bu yerga yetgan bo'lsa sessiya rostan o'lgan.
          if (e instanceof ApiError && e.status === 401) {
            tokenStore.clear()
            return
          }
          // Tarmoq/server xatosi — token saqlanadi, foydalanuvchi xabardor
          // qilinadi (faqat birinchi marta) va so'rov o'zi qayta uriniladi.
          // Ilova esa ochilaveradi (`ready`): kutish ekranida qotib qolmaydi.
          if (attempt === 0) toast.error(errorText(e, 'Serverga ulanib bo‘lmadi — qayta urinilmoqda'))
          timer = setTimeout(load, bootstrapRetryDelay(attempt))
          attempt += 1
        })
        .finally(() => {
          if (alive) setReady(true)
        })
    }
    // Internet qaytishi bilan kutmasdan urinamiz.
    const onOnline = () => {
      if (!timer) return
      clearTimeout(timer)
      timer = null
      load()
    }
    window.addEventListener('online', onOnline)
    load()

    return () => {
      alive = false
      window.removeEventListener('online', onOnline)
      if (timer) clearTimeout(timer)
    }
  }, [])

  // Real-time kanal — faqat authed bo'lganda.
  useEffect(() => {
    if (!user) return
    const disconnect = connectRealtime((msg) => {
      switch (msg.type) {
        case 'notification':
          qc.invalidateQueries({ queryKey: qk.notifications })
          qc.invalidateQueries({ queryKey: qk.unreadCount })
          if (msg.payload && msg.payload.title) toast.info(msg.payload.title)
          break
        case 'waiting_room.request':
          qc.invalidateQueries({ queryKey: qk.waiting })
          toast.info("Kutish xonasiga yangi so'rov")
          break
        default:
          break
      }
    })
    return disconnect
  }, [user, qc])

  const doLogin = useCallback(async (email, password) => {
    await apiLogin(email, password)
    const u = await me()
    setUser(u)
    return u
  }, [])

  const doRegister = useCallback(async (full_name, email, password) => {
    await apiRegister(full_name, email, password)
    const u = await me()
    setUser(u)
    return u
  }, [])

  const doLogout = useCallback(async () => {
    await apiLogout()
    setUser(null)
    qc.clear()
  }, [qc])

  // Memo: har renderda yangi obyekt bo'lsa kontekstning HAR BIR iste'molchisi
  // (butun daraxt) qayta render bo'lardi — jonli xonada ham.
  const value = useMemo(
    () => ({
      user,
      setUser,
      authed: !!user || tokenStore.isAuthed,
      ready,
      doLogin,
      doRegister,
      doLogout,
    }),
    [user, ready, doLogin, doRegister, doLogout],
  )

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp AppProvider ichida ishlatilishi kerak')
  return ctx
}
