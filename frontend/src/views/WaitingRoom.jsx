import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Ban } from 'lucide-react'
import { useWaitingStatus } from '../store/data'
import { connectWaitingRoom } from '../lib/ws'
import { roomSession } from '../lib/roomSession'
import { Button } from '../components/Button'
import { Spinner } from '../components/Spinner'

// Guest kutish ekrani. WS real-time push (admit/reject) + status polling (fallback).
export function WaitingRoom() {
  const { slug } = useParams()
  const navigate = useNavigate()
  const pending = roomSession.get().pending
  const requestId = pending?.requestId
  const [rejected, setRejected] = useState(false)

  const { data, error } = useWaitingStatus(requestId, !!requestId && !rejected)
  // «So'rov topilmadi» FAQAT haqiqiy 404 da. Avval HAR QANDAY xato shunday
  // ko'rsatilardi: status endpoint'i IP bo'yicha 20 so'rov/s bilan cheklangan
  // va katta sinf (bitta NAT) 429 olganda hamma «so'rov muddati o'tgan» degan
  // yolg'onni ko'rardi. 429/tarmoq xatosida kutishda qolamiz — poll o'zi
  // (sekinroq) davom etadi va WS kanali ham ishlayveradi.
  const notFound = error?.status === 404

  function admit(room) {
    roomSession.setRoom({ token: room, lesson: pending.lesson, guestName: pending.guestName, join: pending.join })
    navigate(`/r/${slug}/room`, { replace: true })
  }

  // WS real-time (tezroq yo'l)
  useEffect(() => {
    if (!requestId) return
    return connectWaitingRoom(requestId, (msg) => {
      if (msg.type === 'waiting_room.admitted' && msg.payload) admit(msg.payload)
      else if (msg.type === 'waiting_room.rejected') setRejected(true)
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestId])

  // Polling fallback
  useEffect(() => {
    if (!data) return
    // Server polling natijasi (tashqi ma'lumot) → UI holatiga sinxronlash (ataylab).
    if (data.status === 'admitted' && data.room) admit(data.room)
    // eslint-disable-next-line react-hooks/set-state-in-effect -- polling natijasini UI'ga ko'chirish
    else if (data.status === 'rejected') setRejected(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  if (!pending)
    return (
      <div className="center-shell col center">
        <p className="text-2">Sessiya topilmadi.</p>
        <Button variant="ghost" style={{ marginTop: 16 }} onClick={() => navigate(`/r/${slug}`)}>
          Qaytadan urinish
        </Button>
      </div>
    )

  if (notFound)
    return (
      <div className="center-shell col center">
        <div className="empty__icon" style={{ background: 'var(--warning-soft)', color: 'var(--warning)' }}>
          <Ban size={28} />
        </div>
        <h2 className="h1" style={{ marginBottom: 6 }}>So'rov topilmadi</h2>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 20, textAlign: 'center', maxWidth: 320 }}>
          Kutish so'rovi muddati o'tgan yoki bekor qilingan.
        </p>
        <Button variant="ghost" onClick={() => navigate(`/r/${slug}`)}>Qaytadan urinish</Button>
      </div>
    )

  if (rejected)
    return (
      <div className="center-shell col center">
        <div className="empty__icon" style={{ background: 'var(--danger-soft)', color: 'var(--danger)' }}>
          <Ban size={28} />
        </div>
        <h2 className="h1" style={{ marginBottom: 6 }}>Kirish rad etildi</h2>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 20, textAlign: 'center', maxWidth: 320 }}>
          Ustoz sizning so'rovingizni rad etdi.
        </p>
        <Button variant="ghost" onClick={() => navigate('/')}>Bosh sahifa</Button>
      </div>
    )

  return (
    <div className="center-shell col center">
      <div
        className="row center"
        style={{ width: 64, height: 64, borderRadius: '50%', background: 'var(--accent-soft)', marginBottom: 24 }}
      >
        <Spinner size={32} />
      </div>
      <h2 className="h1" style={{ marginBottom: 6 }}>Kutish xonasidasiz</h2>
      <p className="text-2" style={{ fontSize: 14, textAlign: 'center', maxWidth: 320, marginBottom: 4 }}>
        "{pending.lesson.title}" darsiga qo'shilish uchun ustoz tasdiqini kutmoqdasiz.
      </p>
      {/* Vaqtinchalik xato (429/tarmoq): kutish davom etadi, lekin sababi aytiladi. */}
      {error && (
        <p className="muted" style={{ fontSize: 13 }}>
          Server bilan aloqa vaqtincha uzildi — qayta urinilmoqda…
        </p>
      )}
      <p className="muted" style={{ fontSize: 13 }}>Bu oyna ochiq turishi kerak…</p>
    </div>
  )
}
