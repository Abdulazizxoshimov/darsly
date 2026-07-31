import { useState } from 'react'
import { AlertTriangle, Download, Film, Loader2, Trash2 } from 'lucide-react'
import { useLessons, useRecordings } from '../store/data'
import { downloadRecording } from '../api/recordings'
import { errorText } from '../api/api'
import { PageLoader } from '../components/Spinner'
import {
  formatSize,
  formatDuration,
  formatExpiry,
  expiresInDays,
  EXPIRY_WARN_DAYS,
  RECORDING_STATUS_UZ,
} from '../lib/format'
import { toast } from '../lib/toast'

// Yozuvlar — jadval: dars · sana · davomiylik · hajm · holat · yuklab olish.
export function Recordings() {
  const { data, isLoading, isError } = useLessons({ limit: 100 })
  const recLessons = (data?.data || []).filter((l) => l.is_recording_enabled)

  return (
    <div className="page">
      <h1 className="h1" style={{ marginBottom: 4 }}>Yozuvlar</h1>
      {/* Saqlash muddati MAHSULOT qoidasi (30 kun) — foydalanuvchi buni yozuv
          yo'qolgandan KEYIN emas, ro'yxatning boshida bilishi kerak. */}
      <p className="text-2" style={{ fontSize: 14, marginBottom: 24 }}>
        Darslarning saqlangan video yozuvlari. Har bir yozuv 30 kun saqlanadi — muddat tugagach
        avtomatik o‘chadi, shuning uchun kerakli darsni oldindan yuklab oling.
      </p>
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
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Dars</th>
                <th>Sana</th>
                <th>Davomiylik</th>
                <th>Hajm</th>
                <th>Saqlanish</th>
                <th>Holat</th>
                <th style={{ textAlign: 'right' }}>Yuklab olish</th>
              </tr>
            </thead>
            <tbody>
              {recLessons.map((l) => (
                <LessonRows key={l.id} lesson={l} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function LessonRows({ lesson }) {
  const { data: recs = [], isLoading, isError } = useRecordings(lesson.id)
  if (isLoading) return null

  /*
    Yozuvi yo'q dars ham jadvalda ko'rinadi va NEGA bo'shligi yozib qo'yiladi —
    aks holda "yozib olish yoqilgan-u, yozuv qani?" degan savol nosozlikdek tuyuladi.
  */
  if (isError || !recs.length) {
    return (
      <tr>
        <td className="table__title">{lesson.title}</td>
        <td colSpan={6} className="muted" style={{ fontSize: 13 }}>
          {isError
            ? "Bu dars yozuvlarini yuklab bo'lmadi."
            : lesson.status === 'live'
              ? 'Dars davom etmoqda — yozuv dars yakunlangach tayyor bo‘ladi.'
              : lesson.status === 'scheduled'
                ? 'Dars hali boshlanmagan.'
                : 'Bu darsda yozuv saqlanmagan.'}
        </td>
      </tr>
    )
  }

  return recs.map((r) => <RecordingRow key={r.id} rec={r} lessonTitle={lesson.title} />)
}

function RecordingRow({ rec, lessonTitle }) {
  const [downloading, setDownloading] = useState(false)

  async function download() {
    // Oyna DARHOL, bosish kontekstida ochiladi. `await` dan KEYIN ochilgan
    // `window.open` foydalanuvchi harakati bilan bog'lanmaydi va Safari/Firefox
    // uni popup deb bloklaydi — natijada tugma "hech nima qilmaydi".
    const win = window.open('', '_blank')
    setDownloading(true)
    try {
      const dl = await downloadRecording(rec.id)
      if (win) win.location.href = dl.url
      else window.location.href = dl.url // popup bloklangan bo'lsa shu tabda
    } catch (e) {
      win?.close()
      toast.error(errorText(e, 'Yuklab olib bo‘lmadi'))
    } finally {
      setDownloading(false)
    }
  }
  const ready = rec.status === 'ready'
  const failed = rec.status === 'failed'
  // Muddati tugagan yozuv: qator TARIX uchun qoladi, lekin fayl MinIO'dan
  // o'chirilgan — yuklab olish 400 beradi. Tugmani ko'rsatib, so'ng xato
  // chiqarish yolg'on va'da bo'lardi, shuning uchun u umuman ko'rsatilmaydi.
  const expired = rec.status === 'expired'
  const inProgress = rec.status === 'recording' || rec.status === 'processing'

  const days = ready ? expiresInDays(rec.expires_at) : null
  const expiryText = ready ? formatExpiry(rec.expires_at) : null
  const soon = days !== null && days <= EXPIRY_WARN_DAYS

  return (
    <tr>
      <td className="table__title" style={{ maxWidth: 300 }}>
        <span className="truncate" style={{ display: 'block' }}>{lessonTitle}</span>
      </td>
      <td className="table__meta">{new Date(rec.started_at).toLocaleString('uz')}</td>
      <td className="table__meta">{ready ? formatDuration(rec.duration_sec) : '—'}</td>
      {/* Muddati tugagan yozuvda HAJM ko'rsatilmaydi: fayl MinIO'dan
          o'chirilgan va "500 MB" raqami uni hali turibdi deb tushuntirardi. */}
      <td className="table__meta">{ready ? formatSize(rec.size_bytes) : '—'}</td>
      <td>
        {expired ? (
          <span className="expiry expiry--gone">O‘chirilgan</span>
        ) : expiryText ? (
          <span className={`expiry ${soon ? 'expiry--soon' : ''}`}>
            {soon && <AlertTriangle size={13} />}
            {expiryText}
          </span>
        ) : (
          <span className="muted" style={{ fontSize: 13 }}>—</span>
        )}
      </td>
      <td>
        <span
          className="row gap-2"
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: failed ? 'var(--danger)' : ready ? 'var(--success)' : 'var(--text-2)',
          }}
        >
          {rec.status === 'recording' && <span className="rec-dot" />}
          {rec.status === 'processing' && <Loader2 size={14} style={{ animation: 'spin 0.7s linear infinite' }} />}
          {expired && <Trash2 size={14} />}
          {RECORDING_STATUS_UZ[rec.status] || rec.status}
        </span>
      </td>
      <td>
        <div className="row-actions">
          {ready ? (
            <button className="btn btn--secondary btn--sm" onClick={download} disabled={downloading}>
              {downloading ? (
                <Loader2 size={15} style={{ animation: 'spin 0.7s linear infinite' }} />
              ) : (
                <Download size={15} />
              )}
              Yuklab olish
            </button>
          ) : expired ? (
            <span className="muted" style={{ fontSize: 13 }}>Muddati tugagan — fayl o‘chirilgan</span>
          ) : (
            <span className="muted" style={{ fontSize: 13 }}>{inProgress ? 'Kuting…' : '—'}</span>
          )}
        </div>
      </td>
    </tr>
  )
}
