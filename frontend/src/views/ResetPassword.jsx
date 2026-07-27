import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { MailX } from 'lucide-react'
import { resetPassword } from '../api/auth'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { EMAIL_ENABLED } from '../lib/features'
import { toast } from '../lib/toast'

// Emaildagi havola: /reset-password?token=<...>
export function ResetPassword() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get('token') || ''
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setLoading(true)
    try {
      await resetPassword(token, password)
      toast.success("Parol yangilandi — kirishingiz mumkin")
      navigate('/auth', { replace: true })
    } catch (err) {
      toast.error(errorText(err, 'Havola yaroqsiz yoki muddati o‘tgan'))
    } finally {
      setLoading(false)
    }
  }

  // Email o'chiq va token ham yo'q bo'lsa oqim mavjud emas — tushunarli xabar
  if (!EMAIL_ENABLED && !token)
    return (
      <div className="center-shell">
        <div className="card card--pad" style={{ width: '100%', maxWidth: 400 }}>
          <div className="col center" style={{ textAlign: 'center' }}>
            <div className="empty__icon" style={{ background: 'var(--warning-soft)', color: 'var(--warning)' }}>
              <MailX size={28} />
            </div>
            <h2 className="h1" style={{ marginBottom: 6 }}>Parol tiklash hozircha o‘chiq</h2>
            <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
              Email orqali parol tiklash vaqtincha mavjud emas. Administrator bilan bog‘laning.
            </p>
            <Button variant="ghost" onClick={() => navigate('/auth')}>Kirishga qaytish</Button>
          </div>
        </div>
      </div>
    )

  return (
    <div className="center-shell">
      <div className="card card--pad" style={{ width: '100%', maxWidth: 400 }}>
        <h1 className="h1" style={{ marginBottom: 4 }}>Yangi parol</h1>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>Yangi parolingizni kiriting.</p>
        {!token ? (
          <div className="col center" style={{ textAlign: 'center' }}>
            <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>Havola yaroqsiz — token topilmadi.</p>
            <Button variant="ghost" onClick={() => navigate('/auth')}>Kirishga qaytish</Button>
          </div>
        ) : (
          <form onSubmit={submit} className="col gap-4">
            <Field
              label="Yangi parol"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              minLength={8}
            />
            <Button type="submit" size="lg" loading={loading} className="full">
              Parolni yangilash
            </Button>
          </form>
        )}
      </div>
    </div>
  )
}
