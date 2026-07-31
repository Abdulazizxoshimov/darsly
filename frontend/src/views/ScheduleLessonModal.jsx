import { useState } from 'react'
import { useCreateLesson } from '../store/data'
import { errorText } from '../api/api'
import { Modal } from '../components/Modal'
import { Field } from '../components/Field'
import { Button } from '../components/Button'
import { Toggle } from '../components/Toggle'
import { toast } from '../lib/toast'

// «Dars rejalashtirish» — Jadval sahifasidagi to'liq forma: nom, tavsif,
// boshlanish vaqti (BU YERDA majburiy), davomiylik, parol, toggle'lar.
// Yaratilgach xonaga KIRMAYDI — dars jadvalda ko'rinadi.
// Tezkor dars alohida: Darslar sahifasidagi CreateLessonModal.
export function ScheduleLessonModal({ open, onClose }) {
  const create = useCreateLesson()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [duration, setDuration] = useState(60)
  const [passcode, setPasscode] = useState('')
  // Yozib olish DEFAULT YONIQ — yoqishni unutish qaytarib bo'lmaydigan
  // yo'qotish (dars o'tib ketdi), o'chirishni unutish esa tuzatiladi.
  const [recording, setRecording] = useState(true)
  // Kutish xonasi DEFAULT O'CHIQ (PRODUCT.md) — server default'i bilan bir xil.
  const [waitingRoom, setWaitingRoom] = useState(false)
  // Zoom modeli ovoz sozlamalari (server default'lari ham true):
  // kirganda mute — sinf shovqin bilan boshlanmasin; o'zi ochish ruxsati —
  // o'quvchi gapirmoqchi bo'lsa mikrofonini o'zi yoqadi.
  const [muteOnEntry, setMuteOnEntry] = useState(true)
  const [allowSelfUnmute, setAllowSelfUnmute] = useState(true)

  function reset() {
    setTitle('')
    setDescription('')
    setScheduledAt('')
    setDuration(60)
    setPasscode('')
  }

  async function submit(e) {
    e.preventDefault()
    try {
      await create.mutateAsync({
        title,
        description: description || undefined,
        scheduled_at: new Date(scheduledAt).toISOString(),
        duration_min: Number(duration),
        passcode: passcode || undefined,
        is_recording_enabled: recording,
        is_waiting_room_enabled: waitingRoom,
        mute_on_entry: muteOnEntry,
        allow_self_unmute: allowSelfUnmute,
      })
      toast.success('Dars rejalashtirildi')
      reset()
      onClose()
    } catch (err) {
      toast.error(errorText(err, 'Dars rejalashtirilmadi'))
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Dars rejalashtirish" width={520}>
      <form onSubmit={submit} className="col gap-4">
        <Field
          label="Dars nomi"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Masalan: Kvadrat tenglamalar"
          required
          minLength={2}
        />
        <Field
          label="Tavsif (ixtiyoriy)"
          textarea
          rows={2}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Dars haqida qisqacha"
        />
        <div className="row gap-3">
          <Field
            label="Sana va vaqt"
            type="datetime-local"
            value={scheduledAt}
            onChange={(e) => setScheduledAt(e.target.value)}
            required
          />
          <Field
            label="Davomiyligi (daqiqa)"
            type="number"
            min={5}
            max={1440}
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
          />
        </div>
        <Field
          label="Parol (ixtiyoriy)"
          value={passcode}
          onChange={(e) => setPasscode(e.target.value)}
          placeholder="4–20 belgi"
          minLength={4}
          maxLength={20}
        />
        <div className="col gap-2" style={{ marginTop: 4 }}>
          <Toggle label="Yozib olish yoqilsin" checked={recording} onChange={setRecording} />
          <Toggle label="Kutish xonasi yoqilsin" checked={waitingRoom} onChange={setWaitingRoom} />
          <Toggle label="Kirganda mikrofon o'chiq bo'lsin" checked={muteOnEntry} onChange={setMuteOnEntry} />
          <Toggle label="O'quvchi mikrofonni o'zi yoqa olsin" checked={allowSelfUnmute} onChange={setAllowSelfUnmute} />
        </div>
        <div className="row gap-3" style={{ marginTop: 8 }}>
          <Button type="button" variant="ghost" onClick={onClose} className="grow">
            Bekor qilish
          </Button>
          <Button type="submit" loading={create.isPending} className="grow">
            Rejalashtirish
          </Button>
        </div>
      </form>
    </Modal>
  )
}
