import { createContext, useContext, useEffect, useState, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { tokenStore, setUnauthorizedHandler } from '../api/api'
import { me, login as apiLogin, register as apiRegister, logout as apiLogout } from '../api/auth'
import { connectRealtime } from '../lib/ws'
import { toast } from '../lib/toast'

const AppContext = createContext(null)

export function AppProvider({ children }) {
  const qc = useQueryClient()
  const [user, setUser] = useState(null)
  const [ready, setReady] = useState(!tokenStore.isAuthed) // token yo'q → darrov tayyor

  // 401 → sessiyani tozalash (RequireAuth /auth ga yo'naltiradi).
  useEffect(() => {
    setUnauthorizedHandler(() => {
      tokenStore.clear()
      setUser(null)
      if (!window.location.pathname.startsWith('/auth')) window.location.href = '/auth'
    })
  }, [])

  // Boshlang'ich: token bo'lsa foydalanuvchini yuklaymiz.
  useEffect(() => {
    let alive = true
    if (tokenStore.isAuthed && !user) {
      me()
        .then((u) => alive && setUser(u))
        .catch(() => alive && tokenStore.clear())
        .finally(() => alive && setReady(true))
    } else {
      setReady(true)
    }
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Real-time kanal — faqat authed bo'lganda.
  useEffect(() => {
    if (!user) return
    const disconnect = connectRealtime((msg) => {
      switch (msg.type) {
        case 'notification':
          qc.invalidateQueries({ queryKey: ['notifications'] })
          qc.invalidateQueries({ queryKey: ['unread-count'] })
          if (msg.payload && msg.payload.title) toast.info(msg.payload.title)
          break
        case 'waiting_room.request':
          qc.invalidateQueries({ queryKey: ['waiting'] })
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

  const value = {
    user,
    setUser,
    authed: !!user || tokenStore.isAuthed,
    ready,
    doLogin,
    doRegister,
    doLogout,
  }

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp AppProvider ichida ishlatilishi kerak')
  return ctx
}
