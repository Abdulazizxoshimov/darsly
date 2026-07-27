import { useState } from 'react'
import { Bell, BellOff, Calendar, CheckCheck, Info, Users } from 'lucide-react'
import { useNotifications, useMarkRead, useMarkAllRead } from '../store/data'
import { Button } from '../components/Button'
import { PageLoader } from '../components/Spinner'

const TYPE_ICON = { lesson_reminder: Calendar, waiting_room: Users, system: Info }

export function Notifications() {
  const [unreadOnly, setUnreadOnly] = useState(false)
  const { data, isLoading, isError } = useNotifications(unreadOnly)
  const markRead = useMarkRead()
  const markAll = useMarkAllRead()

  return (
    <div className="page" style={{ maxWidth: 760 }}>
      <div className="row between" style={{ marginBottom: 24 }}>
        <h1 className="h1">Bildirishnomalar</h1>
        <div className="row gap-2">
          <Button variant={unreadOnly ? 'primary' : 'ghost'} size="sm" onClick={() => setUnreadOnly((v) => !v)}>
            <BellOff size={16} /> O'qilmagan
          </Button>
          <Button variant="secondary" size="sm" onClick={() => markAll.mutate()} loading={markAll.isPending}>
            <CheckCheck size={16} /> Barchasini o'qildi
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div style={{ height: 260 }}>
          <PageLoader />
        </div>
      ) : isError ? (
        <div className="empty">
          <p className="text-2" style={{ fontSize: 14 }}>Bildirishnomalarni yuklab bo'lmadi.</p>
        </div>
      ) : !data?.data?.length ? (
        <div className="empty">
          <Bell size={32} color="var(--text-3)" style={{ marginBottom: 12 }} />
          <p className="text-2" style={{ fontSize: 14 }}>Bildirishnomalar yo'q</p>
        </div>
      ) : (
        <div className="col gap-2">
          {data.data.map((n) => {
            const Icon = TYPE_ICON[n.type] || Info
            const unread = !n.read_at
            return (
              <div
                key={n.id}
                className={`notif-row ${unread ? 'unread' : ''}`}
                style={{ opacity: unread ? 1 : 0.7 }}
                onClick={() => unread && markRead.mutate(n.id)}
              >
                <div className="notif-row__icon">
                  <Icon size={20} />
                </div>
                <div className="grow">
                  <div className="row gap-2">
                    <span style={{ fontWeight: 600, fontSize: 14 }}>{n.title}</span>
                    {unread && <span className="dot" style={{ background: 'var(--accent)' }} />}
                  </div>
                  <p className="text-2" style={{ fontSize: 13, marginTop: 2 }}>{n.body}</p>
                  <p className="muted" style={{ fontSize: 12, marginTop: 4 }}>
                    {new Date(n.created_at).toLocaleString('uz')}
                  </p>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
