import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useApp } from '../store/app'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { EMAIL_ENABLED } from '../lib/features'
import { toast } from '../lib/toast'

export function Auth() {
  const navigate = useNavigate()
  const { doLogin, doRegister } = useApp()
  const [tab, setTab] = useState('login')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setLoading(true)
    try {
      if (tab === 'login') await doLogin(email, password)
      else await doRegister(fullName, email, password)
      toast.success(tab === 'login' ? 'Xush kelibsiz!' : "Ro'yxatdan o'tdingiz!")
      navigate('/app', { replace: true })
    } catch (err) {
      toast.error(errorText(err, 'Kirish amalga oshmadi'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="auth">
      <div className="auth__side">
        <div className="row gap-3" style={{ fontWeight: 800, fontSize: 20 }}>
          <div className="brand-logo">D</div> Darsly
        </div>
        <div>
          <div style={{ fontSize: 30, fontWeight: 800, lineHeight: 1.25, marginBottom: 16 }}>
            "Zoom'dan ko'chib o'tganimizga afsuslanmadim — hammasi bitta joyda."
          </div>
          <div className="row gap-3">
            <div className="brand-logo" style={{ width: 40, height: 40 }}>A</div>
            <div className="text-2" style={{ fontSize: 13.5 }}>Aziz Karimov, Matematika o'qituvchisi</div>
          </div>
        </div>
        <div />
      </div>

      <div className="auth__form">
        <div style={{ width: '100%', maxWidth: 380 }}>
          <div className="tabs">
            <button className={`tab ${tab === 'login' ? 'active' : ''}`} onClick={() => setTab('login')}>
              Kirish
            </button>
            <button className={`tab ${tab === 'register' ? 'active' : ''}`} onClick={() => setTab('register')}>
              Ro'yxatdan o'tish
            </button>
          </div>

          <form onSubmit={submit} className="col gap-4">
            {tab === 'register' && (
              <Field
                label="To'liq ism"
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="Ism Familiya"
                required
                minLength={2}
              />
            )}
            <Field
              label="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="email@misol.uz"
              required
            />
            <Field
              label="Parol"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              minLength={tab === 'register' ? 8 : undefined}
            />
            <Button type="submit" size="lg" loading={loading} className="full" style={{ marginTop: 8 }}>
              {tab === 'login' ? 'Kirish' : "Ro'yxatdan o'tish"}
            </Button>
          </form>

          {tab === 'login' && EMAIL_ENABLED && (
            <p className="muted" style={{ textAlign: 'center', fontSize: 13, marginTop: 12 }}>
              <a onClick={() => navigate('/forgot-password')} style={{ cursor: 'pointer', fontWeight: 600 }}>
                Parolni unutdingizmi?
              </a>
            </p>
          )}

          <p className="muted" style={{ textAlign: 'center', fontSize: 13, marginTop: 20 }}>
            Havola orqali qo'shilmoqchimisiz?{' '}
            <a onClick={() => navigate('/')} style={{ cursor: 'pointer', fontWeight: 600 }}>
              Bosh sahifa
            </a>
          </p>
        </div>
      </div>
    </div>
  )
}
