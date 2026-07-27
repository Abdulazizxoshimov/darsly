import { useState } from 'react'
import { KeyRound, Save } from 'lucide-react'
import { useApp } from '../store/app'
import { useUpdateProfile, useChangePassword } from '../store/data'
import { errorText } from '../api/api'
import { Field } from '../components/Field'
import { Button } from '../components/Button'
import { Avatar } from '../components/Avatar'
import { toast } from '../lib/toast'

export function Profile() {
  const { user, setUser } = useApp()
  const updateProfile = useUpdateProfile()
  const changePassword = useChangePassword()

  const [fullName, setFullName] = useState(user?.full_name || '')
  const [timezone, setTimezone] = useState(user?.timezone || '')
  const [language, setLanguage] = useState(user?.language || 'uz')
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')

  async function saveProfile(e) {
    e.preventDefault()
    try {
      const updated = await updateProfile.mutateAsync({ full_name: fullName, timezone, language })
      setUser(updated)
      toast.success('Profil saqlandi')
    } catch (err) {
      toast.error(errorText(err))
    }
  }

  async function savePassword(e) {
    e.preventDefault()
    try {
      await changePassword.mutateAsync({ current, next })
      toast.success("Parol o'zgartirildi")
      setCurrent('')
      setNext('')
    } catch (err) {
      toast.error(errorText(err))
    }
  }

  return (
    <div className="page" style={{ maxWidth: 640 }}>
      <h1 className="h1" style={{ marginBottom: 24 }}>Profil</h1>

      <div className="card card--pad" style={{ marginBottom: 20 }}>
        <div className="row gap-4" style={{ marginBottom: 24 }}>
          <Avatar name={user?.full_name || '?'} color={user?.color} src={user?.avatar_url} size={64} />
          <div>
            <div style={{ fontWeight: 700, fontSize: 18 }}>{user?.full_name}</div>
            <div className="text-2" style={{ fontSize: 14 }}>{user?.email}</div>
            <div className="muted cap" style={{ fontSize: 12, marginTop: 2 }}>{user?.role}</div>
          </div>
        </div>
        <form onSubmit={saveProfile} className="col gap-4">
          <Field label="To'liq ism" value={fullName} onChange={(e) => setFullName(e.target.value)} minLength={2} />
          <div className="row gap-3">
            <Field label="Vaqt mintaqasi" value={timezone} onChange={(e) => setTimezone(e.target.value)} placeholder="Asia/Tashkent" />
            <Field label="Til" value={language} onChange={(e) => setLanguage(e.target.value)} placeholder="uz" maxLength={8} />
          </div>
          <div>
            <Button type="submit" loading={updateProfile.isPending}>
              <Save size={16} /> Saqlash
            </Button>
          </div>
        </form>
      </div>

      <div className="card card--pad">
        <div className="row gap-2" style={{ marginBottom: 16 }}>
          <KeyRound size={16} color="var(--accent-light)" />
          <h2 className="h2">Parolni o'zgartirish</h2>
        </div>
        <form onSubmit={savePassword} className="col gap-4">
          <Field label="Joriy parol" type="password" value={current} onChange={(e) => setCurrent(e.target.value)} required />
          <Field label="Yangi parol" type="password" value={next} onChange={(e) => setNext(e.target.value)} required minLength={8} />
          <div>
            <Button type="submit" variant="secondary" loading={changePassword.isPending}>
              O'zgartirish
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
