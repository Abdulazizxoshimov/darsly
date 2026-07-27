import { useState } from 'react'
import { useCreateLesson } from '../store/data'
import { errorText } from '../api/api'
import { Modal } from '../components/Modal'
import { Field } from '../components/Field'
import { Button } from '../components/Button'
import { toast } from '../lib/toast'

function Toggle({ label, checked, onChange }) {
  return (
    <div className="toggle" onClick={() => onChange(!checked)}>
      <span style={{ fontWeight: 500 }}>{label}</span>
      <span className={`toggle__track ${checked ? 'on' : ''}`}>
        <span className="toggle__knob" />
      </span>
    </div>
  )
}

export function CreateLessonModal({ open, onClose }) {
  const create = useCreateLesson()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [duration, setDuration] = useState(60)
  const [passcode, setPasscode] = useState('')
  const [recording, setRecording] = useState(true)
  const [waitingRoom, setWaitingRoom] = useState(true)

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
        scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        duration_min: Number(duration),
        passcode: passcode || undefined,
        is_recording_enabled: recording,
        is_waiting_room_enabled: waitingRoom,
      })
      toast.success('Dars yaratildi')
      reset()
      onClose()
    } catch (err) {
      toast.error(errorText(err, 'Dars yaratilmadi'))
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Yangi dars" width={520}>
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
        </div>
        <div className="row gap-3" style={{ marginTop: 8 }}>
          <Button type="button" variant="ghost" onClick={onClose} className="grow">
            Bekor qilish
          </Button>
          <Button type="submit" loading={create.isPending} className="grow">
            Yaratish
          </Button>
        </div>
      </form>
    </Modal>
  )
}
