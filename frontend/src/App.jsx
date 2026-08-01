import { lazy, Suspense } from 'react'
import { Routes, Route } from 'react-router-dom'
import { RequireAuth } from './components/RequireAuth'
import { AppShell } from './components/AppShell'
import { PageLoader } from './components/Spinner'
import { Landing } from './views/Landing'
import { Auth } from './views/Auth'
import { ForgotPassword } from './views/ForgotPassword'
import { ResetPassword } from './views/ResetPassword'
import { Join } from './views/Join'
import { WaitingRoom } from './views/WaitingRoom'
import { Dashboard } from './views/Dashboard'
import { Schedule } from './views/Schedule'
import { Recordings } from './views/Recordings'
import { LessonArchive } from './views/LessonArchive'
import { Notifications } from './views/Notifications'
import { Profile } from './views/Profile'
import { UsersView } from './views/Users'
import { Blocklist } from './views/Blocklist'
import { NotFound } from './views/NotFound'

// Jonli xona LiveKit'ni tortadi — faqat xonaga kirilganda yuklanadi.
const LiveRoom = lazy(() => import('./views/LiveRoom').then((m) => ({ default: m.LiveRoom })))

function LiveRoomLazy(props) {
  return (
    <Suspense fallback={<PageLoader label="Yuklanmoqda…" />}>
      <LiveRoom {...props} />
    </Suspense>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Landing />} />
      <Route path="/auth" element={<Auth />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />

      {/* Guest join oqimi (ochiq) */}
      <Route path="/r/:slug" element={<Join />} />
      <Route path="/r/:slug/waiting" element={<WaitingRoom />} />
      <Route path="/r/:slug/room" element={<LiveRoomLazy mode="guest" />} />

      {/* Himoyalangan */}
      <Route element={<RequireAuth />}>
        <Route path="/app/lesson/:id/room" element={<LiveRoomLazy mode="host" />} />
        <Route path="/app" element={<AppShell />}>
          <Route index element={<Dashboard />} />
          <Route path="schedule" element={<Schedule />} />
          <Route path="recordings" element={<Recordings />} />
          {/* O'tgan dars sahifasi: video + chat + materiallar. Shell ICHIDA —
              bu ko'rish sahifasi, jonli xona emas. */}
          <Route path="lesson/:id/archive" element={<LessonArchive />} />
          <Route path="blocklist" element={<Blocklist />} />
          <Route path="notifications" element={<Notifications />} />
          <Route path="profile" element={<Profile />} />
          {/* Faqat admin — komponent ichida rol tekshiriladi (redirect) */}
          <Route path="users" element={<UsersView />} />
        </Route>
      </Route>

      <Route path="*" element={<NotFound />} />
    </Routes>
  )
}
