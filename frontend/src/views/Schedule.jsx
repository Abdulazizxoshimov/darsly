import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Calendar, Clock, Plus, Video } from 'lucide-react'
import { useApp } from '../store/app'
import { useLessons } from '../store/data'
import { Button } from '../components/Button'
import { StatusBadge } from '../components/Badge'
import { PageLoader } from '../components/Spinner'
import { CreateLessonModal } from './CreateLessonModal'
import { formatDay, formatTime } from '../lib/format'

export function Schedule() {
  const navigate = useNavigate()
  const { user } = useApp()
  const isMentor = user?.role === 'mentor'
  const [createOpen, setCreateOpen] = useState(false)
  const { data, isLoading, isError } = useLessons({ status: 'scheduled', limit: 100 })

  const lessons = [...(data?.data || [])].sort(
    (a, b) =>
      (a.scheduled_at ? Date.parse(a.scheduled_at) : Infinity) -
      (b.scheduled_at ? Date.parse(b.scheduled_at) : Infinity),
  )

  const groups = groupByDay(lessons)

  return (
    <div className="page" style={{ maxWidth: 760 }}>
      <div className="row between" style={{ marginBottom: 24 }}>
        <h1 className="h1">Jadval</h1>
        {isMentor && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus size={18} /> Yangi dars
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
      ) : !lessons.length ? (
        <div className="empty">
          <div className="empty__icon">
            <Calendar size={28} />
          </div>
          <p className="text-2" style={{ fontSize: 14 }}>Rejalashtirilgan dars yo'q</p>
        </div>
      ) : (
        <div className="col gap-4" style={{ gap: 24 }}>
          {groups.map(([day, items]) => (
            <div key={day}>
              <h3 className="text-2" style={{ fontSize: 13, fontWeight: 700, marginBottom: 10 }}>{day}</h3>
              <div className="col gap-2">
                {items.map((l) => (
                  <div key={l.id} className="row gap-4 card" style={{ padding: 16 }}>
                    <div className="col center" style={{ width: 56, flexShrink: 0 }}>
                      <Clock size={16} color="var(--text-3)" />
                      <span style={{ fontSize: 14, fontWeight: 700, marginTop: 2 }}>{formatTime(l.scheduled_at)}</span>
                    </div>
                    <div className="grow">
                      <div style={{ fontWeight: 600, fontSize: 14 }} className="truncate">{l.title}</div>
                      <div className="muted" style={{ fontSize: 12 }}>{l.duration_min} daqiqa</div>
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
          ))}
        </div>
      )}

      <CreateLessonModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </div>
  )
}

function groupByDay(lessons) {
  const map = new Map()
  for (const l of lessons) {
    const key = formatDay(l.scheduled_at)
    if (!map.has(key)) map.set(key, [])
    map.get(key).push(l)
  }
  return [...map.entries()]
}
