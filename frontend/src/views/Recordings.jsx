import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Download, Film, Loader2, PlayCircle, Send, Trash2 } from 'lucide-react'
import { qk, useLessons } from '../store/data'
import { downloadRecording, listRecordings } from '../api/recordings'
import { errorText } from '../api/api'
import { PageLoader } from '../components/Spinner'
import { mapLimited } from '../lib/batch'
import { safeUrl } from '../lib/url'
import {
  formatBytes,
  formatDuration,
  formatExpiry,
  expiresInDays,
  EXPIRY_WARN_DAYS,
  RECORDING_STATUS_UZ,
} from '../lib/format'
import { toast } from '../lib/toast'

// Backend'da «hamma yozuvlar» endpoint'i yo'q — har dars alohida so'raladi.
// Avval bu har qator o'z `useQuery`si bilan 100 tagacha PARALLEL so'rov edi;
// mentor limiti 30 so'rov/s (burst 60), ya'ni sahifa o'zini 429 ga urar va
// jadvalning yarmi «yuklab bo'lmadi» bo'lib chiqardi. Endi BITTA so'rov
// cheklangan parallellik va tezlik bilan hammasini yig'adi (`lib/batch`).
export const RECORDINGS_CONCURRENCY = 4
export const RECORDINGS_MIN_INTERVAL_MS = 60 // ≤ ~16 so'rov/s — limitdan ancha past

/** Yozuvi bo'lishi MUMKIN bo'lgan darslar: yozuv yoqilgan va boshlangan. */
export function lessonsWithRecordings(lessons) {
  return (lessons || []).filter((l) => l.is_recording_enabled)
}
function mayHaveRecordings(l) {
  return l.status === 'live' || l.status === 'ended'
}

// Dars → { recs } yoki { error }. Boshlanmagan dars so'ralmaydi (yozuvi bo'lmaydi).
export async function fetchRecordingsByLesson(lessons, fetchOne = listRecordings) {
  const targets = lessons.filter(mayHaveRecordings)
  const results = await mapLimited(targets, (l) => fetchOne(l.id), {
    concurrency: RECORDINGS_CONCURRENCY,
    minIntervalMs: RECORDINGS_MIN_INTERVAL_MS,
  })
  const byLesson = {}
  for (const l of lessons) byLesson[l.id] = { recs: [] }
  targets.forEach((l, i) => {
    const r = results[i]
    byLesson[l.id] = r.ok ? { recs: r.value } : { error: r.error }
  })
  return byLesson
}

function useRecordingsByLesson(lessons) {
  const ids = lessons.map((l) => l.id)
  return useQuery({
    queryKey: [...qk.recordings, 'by-lessons', ids],
    queryFn: () => fetchRecordingsByLesson(lessons),
    enabled: ids.length > 0,
  })
}

// Yozuvlar — jadval: dars · sana · davomiylik · hajm · holat · yuklab olish.
export function Recordings() {
  const { data, isLoading, isError } = useLessons({ limit: 100 })
  const recLessons = useMemo(() => lessonsWithRecordings(data?.data), [data])
  const byLesson = useRecordingsByLesson(recLessons)

  return (
    <div className="page">
      <h1 className="h1" style={{ marginBottom: 4 }}>Yozuvlar</h1>
      {/* Saqlash muddati MAHSULOT qoidasi (30 kun) — foydalanuvchi buni yozuv
          yo'qolgandan KEYIN emas, ro'yxatning boshida bilishi kerak. */}
      <p className="text-2" style={{ fontSize: 14, marginBottom: 24 }}>
        Darslarning saqlangan video yozuvlari. Har bir yozuv 30 kun saqlanadi — muddat tugagach
        avtomatik o‘chadi, shuning uchun kerakli darsni oldindan yuklab oling.
      </p>
      {isLoading || (recLessons.length > 0 && byLesson.isLoading) ? (
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
                <LessonRows key={l.id} lesson={l} entry={byLesson.data?.[l.id]} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function LessonRows({ lesson, entry }) {
  const recs = entry?.recs || []

  /*
    Yozuvi yo'q dars ham jadvalda ko'rinadi va NEGA bo'shligi yozib qo'yiladi —
    aks holda "yozib olish yoqilgan-u, yozuv qani?" degan savol nosozlikdek tuyuladi.
  */
  if (entry?.error || !recs.length) {
    return (
      <tr>
        <td className="table__title">{lesson.title}</td>
        <td colSpan={6} className="muted" style={{ fontSize: 13 }}>
          {entry?.error
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
    // Yangi oyna bizning sahifamizga (`opener`) qaytib ta'sir qila olmasin.
    if (win) win.opener = null
    setDownloading(true)
    try {
      const dl = await downloadRecording(rec.id)
      const url = safeUrl(dl?.url)
      if (!url) throw new Error('yaroqsiz havola')
      if (win) win.location.href = url
      else window.location.href = url // popup bloklangan bo'lsa shu tabda
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
  // `archived` — `expired` dan BUTUNLAY boshqacha: fayl serverdan ketgan,
  // lekin Telegramda turibdi va qaytarib olinadi. Tiklash arxiv sahifasida
  // bo'ladi (u yerda progress ko'rinadi va tayyor bo'lgach pleyer ochiladi).
  const archived = rec.status === 'archived'
  const inProgress = rec.status === 'recording' || rec.status === 'processing' || rec.status === 'restoring'

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
      <td className="table__meta">{ready ? formatBytes(rec.size_bytes) : '—'}</td>
      <td>
        {expired ? (
          <span className="expiry expiry--gone">O‘chirilgan</span>
        ) : archived ? (
          <span className="expiry">Telegramda saqlangan</span>
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
          {(rec.status === 'processing' || rec.status === 'restoring') && (
            <Loader2 size={14} style={{ animation: 'spin 0.7s linear infinite' }} />
          )}
          {archived && <Send size={14} />}
          {expired && <Trash2 size={14} />}
          {RECORDING_STATUS_UZ[rec.status] || rec.status}
        </span>
      </td>
      <td>
        <div className="row-actions">
          {ready ? (
            <>
              {/* Arxivda video chat bilan yonma-yon ko'riladi — yuklab olishdan
                  ko'ra ko'p ishlatiladigan harakat, shuning uchun oldinda. */}
              <Link className="btn btn--secondary btn--sm" to={`/app/lesson/${rec.lesson_id}/archive`}>
                <PlayCircle size={15} /> Ko‘rish
              </Link>
              <button className="btn btn--ghost btn--sm" onClick={download} disabled={downloading}>
                {downloading ? (
                  <Loader2 size={15} style={{ animation: 'spin 0.7s linear infinite' }} />
                ) : (
                  <Download size={15} />
                )}
                Yuklab olish
              </button>
            </>
          ) : archived ? (
            // Tiklash SHU YERDA boshlanmaydi: u yuzlab megabayt trafik va
            // 30-60 soniya kutish — arxiv sahifasida progress bilan ko'rsatiladi.
            <Link className="btn btn--secondary btn--sm" to={`/app/lesson/${rec.lesson_id}/archive`}>
              <Send size={15} /> Arxivdan tiklash
            </Link>
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
