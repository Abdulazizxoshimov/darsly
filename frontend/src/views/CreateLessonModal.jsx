import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCreateLesson } from '../store/data'
import { errorText } from '../api/api'
import { Modal } from '../components/Modal'
import { Field } from '../components/Field'
import { Button } from '../components/Button'
import { Toggle } from '../components/Toggle'
import { toast } from '../lib/toast'

// «Dars yaratish» = TEZKOR dars (Zoom'dagi "New meeting" kabi).
// Forma minimal: nom, parol, kutish xonasi, yozib olish — tavsif va vaqt YO'Q.
// Yaratilgach ustoz DARHOL xonaga kiradi: navigatsiya «Boshlash» tugmasi bilan
// bir xil oqim — host tokenni xona sahifasi (LiveRoom) o'zi oladi va dars
// avtomatik live bo'ladi (backend shunday ishlaydi).
// Rejalashtirilgan dars alohida: Jadval sahifasidagi ScheduleLessonModal.
export function CreateLessonModal({ open, onClose }) {
  const navigate = useNavigate()
  const create = useCreateLesson()
  const [title, setTitle] = useState('')
  const [passcode, setPasscode] = useState('')
  // Yozib olish DEFAULT YONIQ — yoqishni unutish qaytarib bo'lmaydigan
  // yo'qotish (dars o'tib ketdi), o'chirishni unutish esa tuzatiladi.
  const [recording, setRecording] = useState(true)
  // Kutish xonasi DEFAULT O'CHIQ (PRODUCT.md): tezkor darsda ustoz odatda
  // havolani o'zi tarqatadi va har kirganni qo'lda tasdiqlashni xohlamaydi.
  // Server default'i ham `false` — uch joyda (DB / web / mobil) zid qiymat bor edi.
  const [waitingRoom, setWaitingRoom] = useState(false)

  function reset() {
    setTitle('')
    setPasscode('')
  }

  async function submit(e) {
    e.preventDefault()
    try {
      // scheduled_at YUBORILMAYDI — bu tezkor dars, server hozirdan boshlaydi.
      const lesson = await create.mutateAsync({
        title,
        passcode: passcode || undefined,
        is_recording_enabled: recording,
        is_waiting_room_enabled: waitingRoom,
      })
      toast.success('Dars yaratildi')
      reset()
      onClose()
      navigate(`/app/lesson/${lesson.id}/room`)
    } catch (err) {
      toast.error(errorText(err, 'Dars yaratilmadi'))
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Dars yaratish" width={440}>
      <form onSubmit={submit} className="col gap-4">
        <p className="text-2" style={{ fontSize: 13, marginTop: -4 }}>
          Dars darhol boshlanadi — keyinroqqa rejalashtirish Jadval bo'limida.
        </p>
        <Field
          label="Dars nomi"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Masalan: Kvadrat tenglamalar"
          required
          minLength={2}
        />
        <Field
          label="Parol (ixtiyoriy)"
          value={passcode}
          onChange={(e) => setPasscode(e.target.value)}
          placeholder="4–20 belgi"
          minLength={4}
          maxLength={20}
        />
        <div className="col gap-2" style={{ marginTop: 4 }}>
          {/*
            Yozib olish DEFAULT YONIQ, lekin majburiy emas. Yoqilgan bo'lsa
            dars boshlanishi bilan server yozuvni O'ZI boshlaydi (backend
            `room.HostToken` → `recording.EnsureRecording`) — ustoz xona
            ichida hech narsa bosmaydi.
          */}
          <Toggle label="Yozib olish yoqilsin" checked={recording} onChange={setRecording} />
          <Toggle label="Kutish xonasi yoqilsin" checked={waitingRoom} onChange={setWaitingRoom} />
        </div>
        <div className="row gap-3" style={{ marginTop: 8 }}>
          <Button type="button" variant="ghost" onClick={onClose} className="grow">
            Bekor qilish
          </Button>
          <Button type="submit" loading={create.isPending} className="grow">
            Yaratish va boshlash
          </Button>
        </div>
      </form>
    </Modal>
  )
}
