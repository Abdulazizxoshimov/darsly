import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useApp } from '../store/app'
import { PageLoader } from './Spinner'

// Autentifikatsiya talab qiluvchi route'lar uchun guard.
export function RequireAuth() {
  const { authed, ready } = useApp()
  const location = useLocation()

  if (!ready) return <PageLoader label="Yuklanmoqda…" />
  if (!authed) return <Navigate to="/auth" state={{ from: location }} replace />
  return <Outlet />
}
