import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  CalendarClock,
  CheckCircle2,
  Copy,
  Film,
  Link2,
  ListVideo,
  Lock,
  MessageSquare,
  Pencil,
  Plus,
  Radio,
  UserCheck,
  Video,
} from 'lucide-react'
import { useApp } from '../store/app'
import { useLessons } from '../store/data'
import { Button } from '../components/Button'
import { StatusBadge } from '../components/Badge'
import { DashboardSkeleton } from '../components/Skeleton'
import { CreateLessonModal } from './CreateLessonModal'
import { EditLessonModal } from './EditLessonModal'
import { ChatHistoryModal } from './ChatHistoryModal'
import { formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'

// Jadval qatorlari uchun holat tartibi: jonli tepada, keyin rejalashtirilgan,
// so'ng tugagan/bekor — ustoz avval "hozir nima bo'layapti"ni ko'radi.
const STATUS_ORDER = { live: 0, scheduled: 1, ended: 2, cancelled: 3 }

function isToday(iso) {
  if (!iso) return false
  const d = new Date(iso)
  const n = new Date()
  return d.getFullYear() === n.getFullYear() && d.getMonth() === n.getMonth() && d.getDate() === n.getDate()
}

export function Dashboard() {
  const { user } = useApp()
  // Admin ham dars ochishi mumkin (mentor huquqlari meros) — faqat student ko'ra olmaydi.
  const isMentor = user?.role === 'mentor' || user?.role === 'admin'
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState(null)
  const [chatLesson, setChatLesson] = useState(null)
  const { data, isLoading, isError } = useLessons({ limit: 100 })

  const lessons = useMemo(() => {
    const list = [...(data?.data || [])]
    list.sort((a, b) => {
      const s = (STATUS_ORDER[a.status] ?? 9) - (STATUS_ORDER[b.status] ?? 9)
      if (s !== 0) return s
      return (b.scheduled_at ? Date.parse(b.scheduled_at) : 0) - (a.scheduled_at ? Date.parse(a.scheduled_at) : 0)
    })
    return list
  }, [data])

  // Statlar faqat API'da BOR ma'lumotdan hisoblanadi (lessons ro'yxati).
  const stats = useMemo(() => {
    const all = data?.data || []
    return {
      total: data?.total ?? all.length,
      today: all.filter((l) => isToday(l.scheduled_at) || isToday(l.started_at)).length,
      live: all.filter((l) => l.status === 'live').length,
      recorded: all.filter((l) => l.is_recording_enabled && l.status === 'ended').length,
    }
  }, [data])

  return (
    <div className="page">
      <div className="row between" style={{ marginBottom: 24 }}>
        <div>
          <h1 className="h1">{isMentor ? 'Mening darslarim' : 'Darslar'}</h1>
          <p className="text-2" style={{ fontSize: 14, marginTop: 4 }}>
            {isMentor ? 'Darslarni yarating va boshqaring' : 'Qatnashadigan darslaringiz'}
          </p>
        </div>
        {isMentor && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus size={18} /> Dars yaratish
          </Button>
        )}
      </div>

      {isLoading ? (
        <DashboardSkeleton />
      ) : isError ? (
        <div className="empty">
          <p className="text-2">Darslarni yuklab bo'lmadi. Qayta urinib ko'ring.</p>
        </div>
      ) : !lessons.length ? (
        <EmptyState isMentor={isMentor} onCreate={() => setCreateOpen(true)} />
      ) : (
        <>
          <div className="stats">
            <div className="stat">
              <span className="stat__label"><ListVideo size={14} /> Jami darslar</span>
              <span className="stat__value">{stats.total}</span>
            </div>
            <div className="stat">
              <span className="stat__label"><CalendarClock size={14} /> Bugungi darslar</span>
              <span className="stat__value">{stats.today}</span>
            </div>
            <div className={`stat ${stats.live > 0 ? 'stat--live' : ''}`}>
              <span className="stat__label"><Radio size={14} /> Jonli efirda</span>
              <span className="stat__value">{stats.live}</span>
            </div>
            <div className="stat">
              <span className="stat__label"><CheckCircle2 size={14} /> Yozuvli yakunlangan</span>
              <span className="stat__value">{stats.recorded}</span>
            </div>
          </div>

          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Dars</th>
                  <th>Holat</th>
                  <th>Vaqt</th>
                  <th>Davomiylik</th>
                  <th>Sozlamalar</th>
                  <th style={{ textAlign: 'right' }}>Amallar</th>
                </tr>
              </thead>
              <tbody>
                {lessons.map((l) => (
                  <LessonRow
                    key={l.id}
                    lesson={l}
                    isMentor={isMentor}
                    onEdit={() => setEditing(l)}
                    onChat={() => setChatLesson(l)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      <CreateLessonModal open={createOpen} onClose={() => setCreateOpen(false)} />
      {editing && <EditLessonModal lesson={editing} onClose={() => setEditing(null)} />}
      {chatLesson && <ChatHistoryModal lesson={chatLesson} onClose={() => setChatLesson(null)} />}
    </div>
  )
}

function LessonRow({ lesson, isMentor, onEdit, onChat }) {
  const navigate = useNavigate()
  const joinUrl = `${window.location.origin}/r/${lesson.join_slug}`

  function copyLink() {
    navigator.clipboard.writeText(joinUrl)
    toast.success('Havola nusxalandi')
  }

  async function share() {
    // Ulashish: qurilma share oynasi bo'lsa — o'sha, bo'lmasa clipboard.
    if (navigator.share) {
      try {
        await navigator.share({ title: lesson.title, url: joinUrl })
      } catch {
        /* foydalanuvchi bekor qildi */
      }
    } else {
      copyLink()
    }
  }

  return (
    <tr className={lesson.status === 'live' ? 'tr--live' : ''}>
      <td style={{ maxWidth: 340 }}>
        <div className="table__title truncate">{lesson.title}</div>
        {lesson.description && <div className="table__sub truncate">{lesson.description}</div>}
      </td>
      <td><StatusBadge status={lesson.status} /></td>
      <td className="table__meta">{formatDateTime(lesson.scheduled_at)}</td>
      <td className="table__meta">{lesson.duration_min} daq</td>
      <td>
        <div className="table__icons">
          {lesson.has_passcode && (
            <span title="Parol bilan himoyalangan" aria-label="Parol" style={{ display: 'flex' }}>
              <Lock size={15} />
            </span>
          )}
          {lesson.is_waiting_room_enabled && (
            <span title="Kutish xonasi yoqilgan" aria-label="Kutish xonasi" style={{ display: 'flex' }}>
              <UserCheck size={15} />
            </span>
          )}
          {lesson.is_recording_enabled && (
            <span title="Yozib olish yoqilgan" aria-label="Yozib olish" style={{ display: 'flex' }}>
              <Radio size={15} />
            </span>
          )}
          {!lesson.has_passcode && !lesson.is_waiting_room_enabled && !lesson.is_recording_enabled && (
            <span className="muted" style={{ fontSize: 13 }}>—</span>
          )}
        </div>
      </td>
      <td>
        <div className="row-actions">
          {isMentor ? (
            <>
              {/*
                Yakunlangan dars xonani OCHMAYDI: server `lesson is not active` (400)
                qaytaradi — o'rniga SHU darsning arxivi.

                Avval bu yerda ikkita o'xshash kirish nuqtasi bor edi: «Yozuvlar»
                (umumiy ro'yxatga olib borardi — ustoz o'z darsini yana qidirishi
                kerak edi) va alohida chat ikonkasi. Arxiv sahifasi ikkalasini
                ham o'z ichiga oladi: video, chat va materiallar bir joyda.
              */}
              {lesson.status === 'ended' ? (
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => navigate(`/app/lesson/${lesson.id}/archive`)}
                >
                  <Film size={15} /> Arxiv
                </Button>
              ) : lesson.status === 'cancelled' ? (
                <Button size="sm" variant="secondary" disabled>
                  Bekor qilingan
                </Button>
              ) : (
                <Button size="sm" onClick={() => navigate(`/app/lesson/${lesson.id}/room`)}>
                  <Video size={15} /> {lesson.status === 'live' ? 'Davom etish' : 'Boshlash'}
                </Button>
              )}
              <button className="tbl-btn" onClick={share} title="Ulashish" aria-label="Ulashish">
                <Link2 size={15} />
              </button>
              <button className="tbl-btn" onClick={copyLink} title="Havolani nusxalash" aria-label="Havolani nusxalash">
                <Copy size={15} />
              </button>
              {/* Yakunlangan darsda chat ARXIV sahifasida (video bilan yonma-yon
                  va vaqt bo'yicha bog'langan holda) — bu yerda takrorlanmaydi.
                  Davom etayotgan/rejadagi darsda esa arxiv hali yo'q, chat
                  tarixi va moderatsiya faqat shu modal orqali ochiladi. */}
              {lesson.status !== 'ended' && (
                <button className="tbl-btn" onClick={onChat} title="Chat tarixi" aria-label="Chat tarixi">
                  <MessageSquare size={15} />
                </button>
              )}
              <button className="tbl-btn" onClick={onEdit} title="Tahrirlash" aria-label="Tahrirlash">
                <Pencil size={15} />
              </button>
            </>
          ) : (
            <Button size="sm" onClick={() => navigate(`/r/${lesson.join_slug}`)}>
              <Link2 size={15} /> Qo'shilish
            </Button>
          )}
        </div>
      </td>
    </tr>
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
