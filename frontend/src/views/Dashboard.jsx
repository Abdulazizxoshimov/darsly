import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Calendar, Copy, Link2, Lock, Plus, Radio, Video } from 'lucide-react'
import { useApp } from '../store/app'
import { useLessons } from '../store/data'
import { Button } from '../components/Button'
import { StatusBadge } from '../components/Badge'
import { PageLoader } from '../components/Spinner'
import { CreateLessonModal } from './CreateLessonModal'
import { formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'

export function Dashboard() {
  const { user } = useApp()
  const isMentor = user?.role === 'mentor'
  const [createOpen, setCreateOpen] = useState(false)
  const { data, isLoading, isError } = useLessons({ limit: 50 })

  return (
    <div className="page">
      <div className="row between" style={{ marginBottom: 28 }}>
        <div>
          <h1 className="h1">{isMentor ? 'Mening darslarim' : 'Darslar'}</h1>
          <p className="text-2" style={{ fontSize: 14, marginTop: 4 }}>
            {isMentor ? 'Darslarni yarating va boshqaring' : 'Qatnashadigan darslaringiz'}
          </p>
        </div>
        {isMentor && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus size={18} /> Yangi dars
          </Button>
        )}
      </div>

      {isLoading ? (
        <div style={{ height: 260 }}>
          <PageLoader label="Darslar yuklanmoqda…" />
        </div>
      ) : isError ? (
        <div className="empty">
          <p className="text-2">Darslarni yuklab bo'lmadi. Qayta urinib ko'ring.</p>
        </div>
      ) : !data?.data?.length ? (
        <EmptyState isMentor={isMentor} onCreate={() => setCreateOpen(true)} />
      ) : (
        <div className="grid-cards">
          {data.data.map((l) => (
            <LessonCard key={l.id} lesson={l} isMentor={isMentor} />
          ))}
        </div>
      )}

      <CreateLessonModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </div>
  )
}

function LessonCard({ lesson, isMentor }) {
  const navigate = useNavigate()

  function copyLink() {
    navigator.clipboard.writeText(`${window.location.origin}/r/${lesson.join_slug}`)
    toast.success('Havola nusxalandi')
  }

  return (
    <div className="lesson-card">
      <div className="row between" style={{ marginBottom: 12 }}>
        <StatusBadge status={lesson.status} />
        <div className="row gap-2 muted">
          {lesson.has_passcode && <Lock size={14} />}
          {lesson.is_recording_enabled && <Radio size={14} />}
        </div>
      </div>
      <h3 style={{ fontSize: 16, fontWeight: 700, lineHeight: 1.3, marginBottom: 6 }}>{lesson.title}</h3>
      {lesson.description && (
        <p className="text-2" style={{ fontSize: 13, marginBottom: 12, overflow: 'hidden' }}>
          {lesson.description}
        </p>
      )}
      <div className="row gap-2 muted" style={{ fontSize: 13, margin: '12px 0 16px', marginTop: 'auto' }}>
        <Calendar size={14} /> {formatDateTime(lesson.scheduled_at)} · {lesson.duration_min} daq
      </div>
      <div className="row gap-2">
        {isMentor ? (
          <>
            <Button size="sm" className="grow" onClick={() => navigate(`/app/lesson/${lesson.id}/room`)}>
              <Video size={16} /> {lesson.status === 'live' ? 'Davom etish' : 'Boshlash'}
            </Button>
            <Button size="sm" variant="secondary" onClick={copyLink} title="Havolani nusxalash">
              <Copy size={16} />
            </Button>
          </>
        ) : (
          <Button size="sm" className="grow" onClick={() => navigate(`/r/${lesson.join_slug}`)}>
            <Link2 size={16} /> Qo'shilish
          </Button>
        )}
      </div>
    </div>
  )
}

function EmptyState({ isMentor, onCreate }) {
  return (
    <div className="empty">
      <div className="empty__icon">
        <Video size={28} />
      </div>
      <h3 className="h2" style={{ marginBottom: 4 }}>Hali dars yo'q</h3>
      <p className="text-2" style={{ fontSize: 14, marginBottom: 20, maxWidth: 320 }}>
        {isMentor
          ? "Birinchi darsingizni yarating va o'quvchilarga havola yuboring."
          : 'Sizga hali dars biriktirilmagan.'}
      </p>
      {isMentor && (
        <Button onClick={onCreate}>
          <Plus size={18} /> Dars yaratish
        </Button>
      )}
    </div>
  )
}
