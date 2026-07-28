import { Download, Film, Loader2 } from 'lucide-react'
import { useLessons, useRecordings } from '../store/data'
import { downloadRecording } from '../api/recordings'
import { errorText } from '../api/api'
import { PageLoader } from '../components/Spinner'
import { formatSize, formatDuration, RECORDING_STATUS_UZ } from '../lib/format'
import { toast } from '../lib/toast'

export function Recordings() {
  const { data, isLoading, isError } = useLessons({ limit: 100 })
  const recLessons = (data?.data || []).filter((l) => l.is_recording_enabled)

  return (
    <div className="page" style={{ maxWidth: 820 }}>
      <h1 className="h1" style={{ marginBottom: 24 }}>Yozuvlar</h1>
      {isLoading ? (
        <div style={{ height: 260 }}>
          <PageLoader />
        </div>
      ) : isError ? (
        <div className="empty">
          <p className="text-2" style={{ fontSize: 14 }}>Yozuvlarni yuklab bo'lmadi. Qayta urinib ko'ring.</p>
        </div>
      ) : !recLessons.length ? (
        <div className="empty">
          <Film size={32} color="var(--text-3)" style={{ marginBottom: 12 }} />
          <p className="text-2" style={{ fontSize: 14 }}>Yozib olingan dars yo'q</p>
        </div>
      ) : (
        <div className="col gap-4" style={{ gap: 20 }}>
          {recLessons.map((l) => (
            <LessonRecordings key={l.id} lesson={l} />
          ))}
        </div>
      )}
    </div>
  )
}

function LessonRecordings({ lesson }) {
  const { data: recs = [], isLoading } = useRecordings(lesson.id)
  if (isLoading) return null
  return (
    <div>
      <h3 className="text-2" style={{ fontSize: 14, fontWeight: 700, marginBottom: 8 }}>{lesson.title}</h3>
      {/*
        Yozuvi yo'q dars AVVAL butunlay yashirilardi (`!recs.length` → null). Natijada
        yozib olish yoqilgan darslar bor-u, hech biriniki hali tayyor bo'lmasa —
        sahifada faqat "Yozuvlar" sarlavhasi turardi va foydalanuvchi buni nosozlik
        deb o'ylardi. Endi dars ko'rinadi va NEGA bo'shligi yozib qo'yiladi.
      */}
      {recs.length ? (
        <div className="col gap-2">
          {recs.map((r) => (
            <RecordingRow key={r.id} rec={r} />
          ))}
        </div>
      ) : (
        <div className="card" style={{ padding: 14 }}>
          <div className="muted" style={{ fontSize: 13 }}>
            {lesson.status === 'live'
              ? 'Dars davom etmoqda — yozuv dars yakunlangach tayyor bo‘ladi.'
              : lesson.status === 'scheduled'
                ? 'Dars hali boshlanmagan.'
                : 'Bu darsda yozuv saqlanmagan.'}
          </div>
        </div>
      )}
    </div>
  )
}

function RecordingRow({ rec }) {
  async function download() {
    try {
      const dl = await downloadRecording(rec.id)
      window.open(dl.url, '_blank')
    } catch (e) {
      toast.error(errorText(e, 'Yuklab olib bo‘lmadi'))
    }
  }
  const ready = rec.status === 'ready'
  return (
    <div className="row gap-3 card" style={{ padding: 14 }}>
      <div className="empty__icon" style={{ width: 40, height: 40, marginBottom: 0 }}>
        <Film size={20} color="var(--accent-light)" />
      </div>
      <div className="grow">
        <div style={{ fontSize: 14, fontWeight: 500 }}>{new Date(rec.started_at).toLocaleString('uz')}</div>
        <div className="muted" style={{ fontSize: 12 }}>
          {ready ? `${formatDuration(rec.duration_sec)} · ${formatSize(rec.size_bytes)}` : RECORDING_STATUS_UZ[rec.status]}
        </div>
      </div>
      {ready ? (
        <button className="btn btn--secondary btn--sm" onClick={download}>
          <Download size={16} /> Yuklab olish
        </button>
      ) : (
        <span
          className="row gap-2"
          style={{ fontSize: 13, color: rec.status === 'failed' ? 'var(--danger)' : 'var(--text-3)' }}
        >
          {(rec.status === 'recording' || rec.status === 'processing') && (
            <Loader2 size={14} style={{ animation: 'spin 0.7s linear infinite' }} />
          )}
          {RECORDING_STATUS_UZ[rec.status]}
        </span>
      )}
    </div>
  )
}
