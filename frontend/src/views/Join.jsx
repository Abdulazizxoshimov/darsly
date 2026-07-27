import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { AlertTriangle, Lock, Users } from 'lucide-react'
import { useJoinPreview } from '../store/data'
import { joinLink } from '../api/join'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { PageLoader } from '../components/Spinner'
import { roomSession } from '../lib/roomSession'
import { toast } from '../lib/toast'

export function Join() {
  const { slug } = useParams()
  const navigate = useNavigate()
  const { data: lesson, isLoading, isError } = useJoinPreview(slug)
  const [name, setName] = useState('')
  const [passcode, setPasscode] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setSubmitting(true)
    try {
      const resp = await joinLink(slug, { guest_name: name, passcode: passcode || undefined })
      if (resp.next_step === 'waiting_room' && resp.request_id) {
        roomSession.setPending({ requestId: resp.request_id, lesson: resp.lesson, guestName: name })
        navigate(`/r/${slug}/waiting`)
      } else if (resp.next_step === 'join' && resp.room) {
        roomSession.setRoom({ token: resp.room, lesson: resp.lesson, guestName: name })
        navigate(`/r/${slug}/room`)
      }
    } catch (err) {
      toast.error(errorText(err, "Qo'shilib bo'lmadi"))
    } finally {
      setSubmitting(false)
    }
  }

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
            {lesson.is_waiting_room_enabled ? "Kutish xonasiga o'tish" : "Darsga qo'shilish"}
          </Button>
        </form>
      </div>
    </div>
  )
}
