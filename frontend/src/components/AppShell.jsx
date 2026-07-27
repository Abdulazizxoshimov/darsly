import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Bell, Calendar, Film, LayoutGrid, LogOut, Menu, User, X } from 'lucide-react'
import { useApp } from '../store/app'
import { useUnreadCount } from '../store/data'
import { Avatar } from './Avatar'

const NAV = [
  { to: '/app', label: 'Boshqaruv', icon: LayoutGrid, end: true },
  { to: '/app/schedule', label: 'Jadval', icon: Calendar },
  { to: '/app/recordings', label: 'Yozuvlar', icon: Film },
  { to: '/app/notifications', label: 'Bildirishnomalar', icon: Bell },
  { to: '/app/profile', label: 'Profil', icon: User },
]

export function AppShell() {
  const { user, doLogout } = useApp()
  const navigate = useNavigate()
  const location = useLocation()
  const { data: unread = 0 } = useUnreadCount()
  const [menuOpen, setMenuOpen] = useState(false)

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
          <div className="brand-logo">D</div>
          Darsly
        </div>
        <nav className="nav">
          {NAV.map(({ to, label, icon: Icon, end }) => (
            <NavLink key={to} to={to} end={end} className={({ isActive }) => `nav__item ${isActive ? 'active' : ''}`}>
              <Icon size={18} /> {label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className="main">
        <header className="topbar">
          <button className="hamburger" onClick={() => setMenuOpen(true)} aria-label="Menyu" aria-expanded={menuOpen}>
            <Menu size={22} />
          </button>
          <button className="bell" onClick={() => navigate('/app/notifications')} aria-label="Bildirishnomalar">
            <Bell size={20} />
            {unread > 0 && <span className="bell__dot">{unread > 9 ? '9+' : unread}</span>}
          </button>
          <div className="row gap-3">
            <Avatar name={user?.full_name || '?'} color={user?.color} src={user?.avatar_url} size={36} />
            <div className="topbar__user" style={{ textAlign: 'right', lineHeight: 1.2 }}>
              <div style={{ fontSize: 14, fontWeight: 700 }}>{user?.full_name}</div>
              <div className="muted cap" style={{ fontSize: 12 }}>{user?.role}</div>
            </div>
          </div>
          <button className="icon-btn" onClick={logout} title="Chiqish" style={{ color: 'var(--text-3)' }}>
            <LogOut size={20} />
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
                <div className="brand-logo">D</div>
                Darsly
              </div>
              <button className="icon-btn" onClick={() => setMenuOpen(false)} aria-label="Yopish">
                <X size={20} />
              </button>
            </div>
            <nav className="nav">
              {NAV.map(({ to, label, icon: Icon, end }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  onClick={() => setMenuOpen(false)}
                  className={({ isActive }) => `nav__item ${isActive ? 'active' : ''}`}
                >
                  <Icon size={18} /> {label}
                </NavLink>
              ))}
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
