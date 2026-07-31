import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { AlertTriangle, CalendarClock, CheckCircle2, Lock, Users } from 'lucide-react'
import { useJoinPreview } from '../store/data'
import { joinLink } from '../api/join'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { PageLoader, Spinner } from '../components/Spinner'
import { roomSession } from '../lib/roomSession'
import { formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'

// Dars boshlanishini kutishda status so'rovi oralig'i (5-8 s talab qilingan).
const START_POLL_MS = 6000

export function Join() {
  const { slug } = useParams()
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [passcode, setPasscode] = useState('')
  const [submitting, setSubmitting] = useState(false)
  // Dars-oldi kutish rejimi: o'quvchi qo'shilmoqchi, lekin dars hali `scheduled`.
  // Bu rejimda preview har START_POLL_MS da qayta so'raladi; `live` bo'lgach
  // avto-kirish (`autoJoinRef` — bitta marta).
  const [waitingStart, setWaitingStart] = useState(false)
  // POST `next_step:"lesson_ended"` qaytardi (preview hali eski bo'lishi mumkin).
  const [endedFromJoin, setEndedFromJoin] = useState(false)
  const autoJoinRef = useRef(false)

  const { data: lesson, isLoading, isError } = useJoinPreview(slug, {
    refetchInterval: waitingStart ? START_POLL_MS : undefined,
  })

  const lessonEnded =
    endedFromJoin || lesson?.status === 'ended' || lesson?.status === 'cancelled'

  async function doJoin() {
    setSubmitting(true)
    try {
      const resp = await joinLink(slug, { guest_name: name, passcode: passcode || undefined })
      if (resp.next_step === 'lesson_ended') {
        setWaitingStart(false)
        setEndedFromJoin(true)
      } else if (resp.next_step === 'waiting_room' && resp.request_id) {
        roomSession.setPending({ requestId: resp.request_id, lesson: resp.lesson, guestName: name })
        navigate(`/r/${slug}/waiting`)
      } else if (resp.next_step === 'join' && resp.room) {
        roomSession.setRoom({ token: resp.room, lesson: resp.lesson, guestName: name })
        navigate(`/r/${slug}/room`)
      }
    } catch (err) {
      // Avto-kirish yiqilsa kutish rejimidan chiqamiz — o'quvchi xatoni ko'rib
      // qo'lda qayta urinsin (masalan parol o'zgargan bo'lishi mumkin).
      setWaitingStart(false)
      autoJoinRef.current = false
      toast.error(errorText(err, "Qo'shilib bo'lmadi"))
    } finally {
      setSubmitting(false)
    }
  }

  async function submit(e) {
    e.preventDefault()
    // Dars hali boshlanmagan — token so'ramaymiz (backend baribir bo'sh xonaga
    // kiritardi yoki kutish so'rovi ochilib qolardi), kutish sahifasini ochamiz.
    if (lesson?.status === 'scheduled') {
      setWaitingStart(true)
      return
    }
    await doJoin()
  }

  // Kutish rejimida dars `live` bo'ldi — avto-kirish (bir marta).
  useEffect(() => {
    if (!waitingStart || lesson?.status !== 'live' || autoJoinRef.current) return
    autoJoinRef.current = true
    doJoin()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [waitingStart, lesson?.status])

  if (isLoading)
    return (
      <div className="center-shell">
        <PageLoader label="Dars yuklanmoqda…" />
      </div>
    )

  if (isError || !lesson)
    return (
      <div className="center-shell">
        <div style={{ textAlign: 'center', maxWidth: 360 }}>
          <div className="empty__icon" style={{ background: 'var(--warning-soft)', color: 'var(--warning)', margin: '0 auto 16px' }}>
            <AlertTriangle size={28} />
          </div>
          <h2 className="h1" style={{ marginBottom: 6 }}>Yaroqsiz havola</h2>
          <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
            Bu dars mavjud emas yoki havola muddati o'tgan.
          </p>
          <Button variant="ghost" onClick={() => navigate('/')}>Bosh sahifa</Button>
        </div>
      </div>
    )

  // Yakunlangan/bekor qilingan dars (№3): havola "o'lik sahifa" emas — holatni
  // aniq aytamiz. Kirish YO'Q (backend ham token bermaydi).
  if (lessonEnded)
    return (
      <div className="center-shell col center">
        <div className="empty__icon" style={{ background: 'var(--success-soft)', color: 'var(--success)' }}>
          <CheckCircle2 size={28} />
        </div>
        <h2 className="h1" style={{ marginBottom: 6 }}>
          {lesson.status === 'cancelled' ? 'Dars bekor qilingan' : 'Dars yakunlangan'}
        </h2>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 20, textAlign: 'center', maxWidth: 360 }}>
          «{lesson.title}» darsi {lesson.status === 'cancelled' ? 'bekor qilingan' : 'allaqachon tugagan'}.
          Yangi havolani ustozingizdan so'rang.
        </p>
        <Button variant="ghost" onClick={() => navigate('/')}>Bosh sahifa</Button>
      </div>
    )

  // Dars-oldi kutish: dars hali boshlanmagan, status kuzatilmoqda.
  if (waitingStart)
    return (
      <div className="center-shell col center">
        <div
          className="row center"
          style={{ width: 64, height: 64, borderRadius: '50%', background: 'var(--accent-soft)', marginBottom: 24 }}
        >
          {submitting ? <Spinner size={32} /> : <CalendarClock size={30} color="var(--accent-light)" />}
        </div>
        <h2 className="h1" style={{ marginBottom: 6 }}>Dars boshlanishini kuting</h2>
        <p className="text-2" style={{ fontSize: 14, textAlign: 'center', maxWidth: 360, marginBottom: 4 }}>
          «{lesson.title}» hali boshlanmagan
          {lesson.scheduled_at ? ` — rejada ${formatDateTime(lesson.scheduled_at)}` : ''}. Ustoz darsni
          boshlashi bilan avtomatik kirasiz.
        </p>
        <p className="muted" style={{ fontSize: 13, marginBottom: 20 }}>Bu oyna ochiq turishi kerak…</p>
        <Button variant="ghost" onClick={() => setWaitingStart(false)}>Orqaga</Button>
      </div>
    )

  return (
    <div className="center-shell">
      <div className="card card--pad" style={{ width: '100%', maxWidth: 420 }}>
        <div className="row gap-2" style={{ color: 'var(--accent-lighter)', fontSize: 13, fontWeight: 600, marginBottom: 16 }}>
          <Users size={16} /> Darsga qo'shilish
        </div>
        <h1 className="h1" style={{ marginBottom: 4 }}>{lesson.title}</h1>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 24 }}>
          {lesson.mentor_name}
          {lesson.is_waiting_room_enabled && ' · Kutish xonasi yoqilgan'}
        </p>
        <form onSubmit={submit} className="col gap-4">
          <Field label="Ismingiz" value={name} onChange={(e) => setName(e.target.value)} placeholder="Ismingizni kiriting" required minLength={2} />
          {lesson.has_passcode && (
            <Field
              label="Parol"
              value={passcode}
              onChange={(e) => setPasscode(e.target.value)}
              placeholder="Dars paroli"
              icon={<Lock size={16} />}
              required
            />
          )}
          <Button type="submit" size="lg" loading={submitting} className="full" style={{ marginTop: 4 }}>
            {lesson.status === 'scheduled'
              ? 'Dars boshlanishini kutish'
              : lesson.is_waiting_room_enabled
                ? "Kutish xonasiga o'tish"
                : "Darsga qo'shilish"}
          </Button>
        </form>
      </div>
    </div>
  )
}
