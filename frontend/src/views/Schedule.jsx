import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Calendar, Clock, Lock, Plus, Radio, UserCheck, Video } from 'lucide-react'
import { useApp } from '../store/app'
import { useLessons } from '../store/data'
import { Button } from '../components/Button'
import { StatusBadge } from '../components/Badge'
import { PageLoader } from '../components/Spinner'
import { ScheduleLessonModal } from './ScheduleLessonModal'
import { formatDay, formatTime } from '../lib/format'

// Jadval — keng ekranda 2 ustun: chapda kunlar (soni bilan), o'ngda tanlangan
// kun darslari. Tor ekranda kunlar gorizontal lentaga yig'iladi (styles.css).
export function Schedule() {
  const navigate = useNavigate()
  const { user } = useApp()
  const isMentor = user?.role === 'mentor' || user?.role === 'admin'
  const [createOpen, setCreateOpen] = useState(false)
  const [selectedDay, setSelectedDay] = useState(null)
  const { data, isLoading, isError } = useLessons({ status: 'scheduled', limit: 100 })

  const groups = useMemo(() => {
    const lessons = [...(data?.data || [])].sort(
      (a, b) =>
        (a.scheduled_at ? Date.parse(a.scheduled_at) : Infinity) -
        (b.scheduled_at ? Date.parse(b.scheduled_at) : Infinity),
    )
    const map = new Map()
    for (const l of lessons) {
      const key = formatDay(l.scheduled_at)
      if (!map.has(key)) map.set(key, [])
      map.get(key).push(l)
    }
    return [...map.entries()]
  }, [data])

  // Tanlangan kun ro'yxatdan chiqib ketsa (dars o'chirildi) — fallback birinchi
  // (eng yaqin) kun. Bu hosila holat: effect'siz, render'da hisoblanadi.
  const active = groups.find(([day]) => day === selectedDay) || groups[0]

  return (
    <div className="page">
      <div className="row between" style={{ marginBottom: 24 }}>
        <div>
          <h1 className="h1">Jadval</h1>
          <p className="text-2" style={{ fontSize: 14, marginTop: 4 }}>Rejalashtirilgan darslar</p>
        </div>
        {isMentor && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus size={18} /> Dars rejalashtirish
          </Button>
        )}
      </div>

      {isLoading ? (
        <div style={{ height: 260 }}>
          <PageLoader />
        </div>
      ) : isError ? (
        <div className="empty">
          <p className="text-2" style={{ fontSize: 14 }}>Jadvalni yuklab bo'lmadi. Qayta urinib ko'ring.</p>
        </div>
      ) : !groups.length ? (
        <div className="empty">
          <div className="empty__icon">
            <Calendar size={28} />
          </div>
          <p className="text-2" style={{ fontSize: 14 }}>Rejalashtirilgan dars yo'q</p>
          {isMentor && (
            <Button style={{ marginTop: 16 }} onClick={() => setCreateOpen(true)}>
              <Plus size={18} /> Dars rejalashtirish
            </Button>
          )}
        </div>
      ) : (
        <div className="sched">
          <div className="sched__days">
            {groups.map(([day, items]) => (
              <button
                key={day}
                className={`day-btn ${day === active?.[0] ? 'active' : ''}`}
                onClick={() => setSelectedDay(day)}
              >
                <span className="truncate">{day}</span>
                <span className="day-btn__count">{items.length}</span>
              </button>
            ))}
          </div>

          <div className="col gap-2">
            {(active?.[1] || []).map((l) => (
              <div key={l.id} className="row gap-4 card" style={{ padding: '14px 18px' }}>
                <div className="col center" style={{ width: 56, flexShrink: 0 }}>
                  <Clock size={15} color="var(--text-3)" />
                  <span style={{ fontSize: 14, fontWeight: 700, marginTop: 2 }}>{formatTime(l.scheduled_at)}</span>
                </div>
                <div className="grow">
                  <div style={{ fontWeight: 700, fontSize: 14 }} className="truncate">{l.title}</div>
                  <div className="muted row gap-2" style={{ fontSize: 12.5, marginTop: 2 }}>
                    {l.duration_min} daqiqa
                    {l.has_passcode && <Lock size={13} />}
                    {l.is_waiting_room_enabled && <UserCheck size={13} />}
                    {l.is_recording_enabled && <Radio size={13} />}
                  </div>
                </div>
                <StatusBadge status={l.status} />
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() =>
                    isMentor ? navigate(`/app/lesson/${l.id}/room`) : navigate(`/r/${l.join_slug}`)
                  }
                >
                  <Video size={16} /> {isMentor ? 'Boshlash' : "Qo'shilish"}
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      <ScheduleLessonModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </div>
  )
}
