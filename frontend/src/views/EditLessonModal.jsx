import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { useDeleteLesson, useUpdateLesson } from '../store/data'
import { errorText } from '../api/api'
import { Modal } from '../components/Modal'
import { Field } from '../components/Field'
import { Button } from '../components/Button'
import { Toggle } from '../components/Toggle'
import { toast } from '../lib/toast'

// ISO → <input type="datetime-local"> qiymati (mahalliy vaqtda).
function toLocalInput(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Dars sozlamalarini tahrirlash (PATCH /lessons/:id) + o'chirish.
// Parol maydoni bo'sh qoldirilsa parol O'ZGARMAYDI; olib tashlash alohida toggle.
//
// Komponent faqat modal OCHIQ bo'lganda mount qilinadi (Dashboard:
// `{editing && <EditLessonModal …/>}`), shuning uchun forma holati to'g'ridan-
// to'g'ri lesson'dan boshlang'ich qiymat oladi — effect'siz.
export function EditLessonModal({ lesson, onClose }) {
  const update = useUpdateLesson()
  const del = useDeleteLesson()

  const [title, setTitle] = useState(lesson.title || '')
  const [description, setDescription] = useState(lesson.description || '')
  const [scheduledAt, setScheduledAt] = useState(toLocalInput(lesson.scheduled_at))
  const [duration, setDuration] = useState(lesson.duration_min || 60)
  const [passcode, setPasscode] = useState('')
  const [removePasscode, setRemovePasscode] = useState(false)
  const [recording, setRecording] = useState(!!lesson.is_recording_enabled)
  const [waitingRoom, setWaitingRoom] = useState(!!lesson.is_waiting_room_enabled)
  // Ovoz sozlamalari (Zoom modeli). `!== false` — eski javoblarda maydon
  // bo'lmasa server default'i (true) bilan mos qolamiz.
  const [muteOnEntry, setMuteOnEntry] = useState(lesson.mute_on_entry !== false)
  const [allowSelfUnmute, setAllowSelfUnmute] = useState(lesson.allow_self_unmute !== false)

  async function submit(e) {
    e.preventDefault()
    try {
      const body = {
        title,
        description: description || undefined,
        duration_min: Number(duration) || undefined,
        is_recording_enabled: recording,
        is_waiting_room_enabled: waitingRoom,
        mute_on_entry: muteOnEntry,
        allow_self_unmute: allowSelfUnmute,
      }
      if (scheduledAt) body.scheduled_at = new Date(scheduledAt).toISOString()
      if (removePasscode) body.remove_passcode = true
      else if (passcode) body.passcode = passcode
      await update.mutateAsync({ id: lesson.id, body })
      toast.success('Dars yangilandi')
      onClose()
    } catch (err) {
      toast.error(errorText(err, 'Darsni yangilab bo‘lmadi'))
    }
  }

  async function removeLesson() {
    if (!window.confirm(`«${lesson.title}» darsini o'chirasizmi? Bu amalni qaytarib bo'lmaydi.`)) return
    try {
      await del.mutateAsync(lesson.id)
      toast.success("Dars o'chirildi")
      onClose()
    } catch (err) {
      toast.error(errorText(err, 'Darsni o‘chirib bo‘lmadi'))
    }
  }

  return (
    <Modal open onClose={onClose} title="Darsni tahrirlash" width={520}>
      <form onSubmit={submit} className="col gap-4">
        <Field
          label="Dars nomi"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
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
        {!removePasscode && (
          <Field
            label={lesson.has_passcode ? "Yangi parol (bo'sh — o'zgarmaydi)" : 'Parol (ixtiyoriy)'}
            value={passcode}
            onChange={(e) => setPasscode(e.target.value)}
            placeholder="4–20 belgi"
            minLength={4}
            maxLength={20}
          />
        )}
        <div className="col gap-2">
          {lesson.has_passcode && (
            <Toggle label="Parolni olib tashlash" checked={removePasscode} onChange={setRemovePasscode} />
          )}
          <Toggle label="Yozib olish yoqilsin" checked={recording} onChange={setRecording} />
          <Toggle label="Kutish xonasi yoqilsin" checked={waitingRoom} onChange={setWaitingRoom} />
          <Toggle label="Kirganda mikrofon o'chiq bo'lsin" checked={muteOnEntry} onChange={setMuteOnEntry} />
          <Toggle label="O'quvchi mikrofonni o'zi yoqa olsin" checked={allowSelfUnmute} onChange={setAllowSelfUnmute} />
        </div>
        <div className="row gap-3" style={{ marginTop: 8 }}>
          <Button type="button" variant="danger" onClick={removeLesson} loading={del.isPending} title="Darsni o'chirish">
            <Trash2 size={16} />
          </Button>
          <span className="grow" />
          <Button type="button" variant="ghost" onClick={onClose}>
            Bekor qilish
          </Button>
          <Button type="submit" loading={update.isPending}>
            Saqlash
          </Button>
        </div>
      </form>
    </Modal>
  )
}
