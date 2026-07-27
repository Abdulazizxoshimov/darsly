import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { MailCheck, MailX } from 'lucide-react'
import { forgotPassword } from '../api/auth'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { EMAIL_ENABLED } from '../lib/features'
import { toast } from '../lib/toast'

export function ForgotPassword() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setLoading(true)
    try {
      await forgotPassword(email)
      setSent(true)
    } catch (err) {
      toast.error(errorText(err))
    } finally {
      setLoading(false)
    }
  }

  // Email o'chiq bo'lsa oqim ishlamaydi — tushunarli xabar (jimgina "yuborildi" demaslik)
  if (!EMAIL_ENABLED)
    return (
      <div className="center-shell">
        <div className="card card--pad" style={{ width: '100%', maxWidth: 400 }}>
          <div className="col center" style={{ textAlign: 'center' }}>
            <div className="empty__icon" style={{ background: 'var(--warning-soft)', color: 'var(--warning)' }}>
              <MailX size={28} />
            </div>
            <h2 className="h1" style={{ marginBottom: 6 }}>Parol tiklash hozircha o‘chiq</h2>
            <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
              Email orqali parol tiklash vaqtincha mavjud emas. Parolingizni tiklash uchun ustoz yoki administrator bilan bog‘laning.
            </p>
            <Button variant="ghost" onClick={() => navigate('/auth')}>Kirishga qaytish</Button>
          </div>
        </div>
      </div>
    )

  return (
    <div className="center-shell">
      <div className="card card--pad" style={{ width: '100%', maxWidth: 400 }}>
        {sent ? (
          <div className="col center" style={{ textAlign: 'center' }}>
            <div className="empty__icon" style={{ background: 'var(--success-soft)', color: 'var(--success)' }}>
              <MailCheck size={28} />
            </div>
            <h2 className="h1" style={{ marginBottom: 6 }}>Havola yuborildi</h2>
            <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
              Agar bu email ro'yxatdan o'tgan bo'lsa, parolni tiklash havolasi yuborildi.
            </p>
            <Button variant="ghost" onClick={() => navigate('/auth')}>Kirishga qaytish</Button>
          </div>
        ) : (
          <>
            <h1 className="h1" style={{ marginBottom: 4 }}>Parolni tiklash</h1>
            <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
              Email manzilingizni kiriting — tiklash havolasini yuboramiz.
            </p>
            <form onSubmit={submit} className="col gap-4">
              <Field
                label="Email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="email@misol.uz"
                required
              />
              <Button type="submit" size="lg" loading={loading} className="full">
                Havola yuborish
              </Button>
            </form>
            <p className="muted" style={{ textAlign: 'center', fontSize: 13, marginTop: 16 }}>
              <a onClick={() => navigate('/auth')} style={{ cursor: 'pointer', fontWeight: 600 }}>Kirishga qaytish</a>
            </p>
          </>
        )}
      </div>
    </div>
  )
}
