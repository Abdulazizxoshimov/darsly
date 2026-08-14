import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Bell, Calendar, Film, LayoutGrid, LogOut, Menu, ShieldOff, User, Users, X } from 'lucide-react'
import { useApp } from '../store/app'
import { useUnreadCount } from '../store/data'
import { Avatar } from './Avatar'
import { roleLabel } from '../lib/format'
import BrandMark from './BrandMark'

// Asosiy bo'limlar — doimiy chap sidebar'da. Bildirishnoma (topbar qo'ng'irog'i)
// va Profil (pastdagi foydalanuvchi kartasi) alohida kirish nuqtalariga ega,
// shuning uchun asosiy ro'yxatda takrorlanmaydi (mobil drawer'da esa bor).
const NAV = [
  { to: '/app', label: 'Darslar', icon: LayoutGrid, end: true },
  { to: '/app/schedule', label: 'Jadval', icon: Calendar },
  // «Arxiv» — o'tgan darslar (video + chat). Ilgari «Yozuvlar» deb nomlangan.
  { to: '/app/recordings', label: 'Arxiv', icon: Film },
  // Qora ro'yxat — mentorning doimiy bloklari (darsdan «Doimiy» chiqarilganlar).
  { to: '/app/blocklist', label: "Qora ro'yxat", icon: ShieldOff },
]

const ADMIN_NAV = [{ to: '/app/users', label: 'Foydalanuvchilar', icon: Users }]

// Bildirishnomalar bu yerda EMAS — u alohida bo'lim emas, faqat topbar
// qo'ng'irog'idan ochiladi (mobil va boshqa ilovalardagi naqsh).
const EXTRA_NAV = [{ to: '/app/profile', label: 'Profil', icon: User }]

function pageTitle(pathname, isAdmin) {
  // Arxiv sahifasi sidebar bo'limi EMAS (unga dars qatoridan kiriladi), lekin
  // topbar sarlavhasiz qolmasin.
  if (pathname.endsWith('/archive')) return 'Dars arxivi'
  // Bildirishnomalar — qo'ng'iroqdan ochiladi, nav ro'yxatida yo'q.
  if (pathname.endsWith('/notifications')) return 'Bildirishnomalar'
  const all = [...NAV, ...(isAdmin ? ADMIN_NAV : []), ...EXTRA_NAV]
  // Eng aniq (uzun) mos kelgan yo'l g'olib — '/app' hammaga mos kelmasin.
  const hit = all
    .filter((n) => (n.end ? pathname === n.to : pathname.startsWith(n.to)))
    .sort((a, b) => b.to.length - a.to.length)[0]
  return hit?.label || ''
}

function NavItems({ items, onNavigate }) {
  return items.map(({ to, label, icon: Icon, end }) => (
    <NavLink
      key={to}
      to={to}
      end={end}
      onClick={onNavigate}
      className={({ isActive }) => `nav__item ${isActive ? 'active' : ''}`}
    >
      <Icon size={18} /> {label}
    </NavLink>
  ))
}

export function AppShell() {
  const { user, doLogout } = useApp()
  const navigate = useNavigate()
  const location = useLocation()
  const { data: unread = 0 } = useUnreadCount()
  const [menuOpen, setMenuOpen] = useState(false)
  const isAdmin = user?.role === 'admin'

  async function logout() {
    setMenuOpen(false)
    await doLogout()
    navigate('/auth', { replace: true })
  }

  // Marshrut o'zgarganda mobil menyu yopilsin
  useEffect(() => {
    setMenuOpen(false)
  }, [location.pathname])

  // Menyu ochiqda Escape bilan yopish
  useEffect(() => {
    if (!menuOpen) return
    function onKey(e) {
      if (e.key === 'Escape') setMenuOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [menuOpen])

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="sidebar__brand">
          <span className="brand-logo--pulse" style={{ display: 'flex' }}>
            <BrandMark />
          </span>
          jonly
        </div>
        <nav className="nav">
          <NavItems items={NAV} />
          {isAdmin && (
            <>
              <div className="nav__section">Boshqaruv</div>
              <NavItems items={ADMIN_NAV} />
            </>
          )}
        </nav>
        <div className="sidebar__foot">
          <div
            className="sidebar__user"
            onClick={() => navigate('/app/profile')}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                navigate('/app/profile')
              }
            }}
            title="Profil"
          >
            <Avatar name={user?.full_name || '?'} color={user?.color} src={user?.avatar_url} size={36} />
            <div className="grow" style={{ lineHeight: 1.25 }}>
              <div className="sidebar__user-name truncate">{user?.full_name}</div>
              <div className="sidebar__user-role">{roleLabel(user?.role)}</div>
            </div>
            <button
              className="icon-btn"
              onClick={(e) => {
                e.stopPropagation()
                logout()
              }}
              title="Chiqish"
              aria-label="Chiqish"
            >
              <LogOut size={17} />
            </button>
          </div>
        </div>
      </aside>

      <div className="main">
        <header className="topbar">
          <button className="hamburger" onClick={() => setMenuOpen(true)} aria-label="Menyu" aria-expanded={menuOpen}>
            <Menu size={22} />
          </button>
          <div className="topbar__title">{pageTitle(location.pathname, isAdmin)}</div>
          <button className="bell" onClick={() => navigate('/app/notifications')} aria-label="Bildirishnomalar">
            <Bell size={20} />
            {unread > 0 && <span className="bell__dot">{unread > 9 ? '9+' : unread}</span>}
          </button>
          <button
            className="icon-btn"
            onClick={() => navigate('/app/profile')}
            title="Profil"
            aria-label="Profil"
            style={{ padding: 0 }}
          >
            <Avatar name={user?.full_name || '?'} color={user?.color} src={user?.avatar_url} size={32} />
          </button>
        </header>
        <main className="topbar__content">
          <Outlet />
        </main>
      </div>

      {/* Mobil navigatsiya — tashqariga bosilsa yopiladi */}
      {menuOpen && (
        <div className="drawer-scrim" onClick={() => setMenuOpen(false)} role="presentation">
          <aside className="drawer" onClick={(e) => e.stopPropagation()}>
            <div className="drawer__head">
              <div className="sidebar__brand" style={{ border: 'none', padding: 0, height: 'auto' }}>
                <BrandMark />
                jonly
              </div>
              <button className="icon-btn" onClick={() => setMenuOpen(false)} aria-label="Yopish">
                <X size={20} />
              </button>
            </div>
            <nav className="nav">
              <NavItems items={NAV} onNavigate={() => setMenuOpen(false)} />
              {isAdmin && <NavItems items={ADMIN_NAV} onNavigate={() => setMenuOpen(false)} />}
              <NavItems items={EXTRA_NAV} onNavigate={() => setMenuOpen(false)} />
              <button className="nav__item" onClick={logout} style={{ background: 'none', border: 'none', width: '100%', textAlign: 'left' }}>
                <LogOut size={18} /> Chiqish
              </button>
            </nav>
          </aside>
        </div>
      )}
    </div>
  )
}
