import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ShieldAlert } from 'lucide-react'
import { useApp } from '../store/app'
import { consumeLogoutReason } from '../lib/logoutReason'
import { useAppConfig } from '../store/data'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { EMAIL_ENABLED } from '../lib/features'
import { toast } from '../lib/toast'
import BrandMark from '../components/BrandMark'

export function Auth() {
  const navigate = useNavigate()
  const { doLogin, doRegister } = useApp()
  // Ochiq ro'yxatdan o'tish server bayrog'i bilan boshqariladi. Bayroq
  // aniq `true` bo'lmaguncha (yuklanmoqda/xato/false) forma KO'RSATILMAYDI —
  // server baribir o'zi bloklaydi, bu faqat UI signali.
  const { data: appConfig } = useAppConfig()
  const allowRegister = appConfig?.allow_open_registration === true

  // NEGA chiqarildik. Sabab bir martalik o'qiladi (`useState` initsializatori —
  // render'da emas: StrictMode ikki marta chaqirsa ham xabar yo'qolmasin).
  //
  // «Sessiya tugadi» deyish yetarli emas: `SESSION_REVOKED` odatda akkaunt
  // BOSHQA qurilmada ochilgani degani va foydalanuvchi buni bilishi kerak —
  // aks holda u internetni yoki ilovani ayblaydi, akkaunti ulashilganini emas.
  const [logoutReason] = useState(consumeLogoutReason)

  const [tab, setTab] = useState('login')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  const registering = allowRegister && tab === 'register'

  async function submit(e) {
    e.preventDefault()
    setLoading(true)
    try {
      if (registering) await doRegister(fullName, email, password)
      else await doLogin(email, password)
      toast.success(registering ? "Ro'yxatdan o'tdingiz!" : 'Xush kelibsiz!')
      navigate('/app', { replace: true })
    } catch (err) {
      toast.error(errorText(err, 'Kirish amalga oshmadi'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="auth-center">
      <div className="auth-card">
        <div className="auth-card__brand">
          <span className="brand-logo--pulse" style={{ display: 'flex' }}>
            <BrandMark size={48} />
          </span>
          jonly
        </div>

        {logoutReason && (
          <div className="auth-notice" role="alert">
            <ShieldAlert size={18} />
            <span>{logoutReason}</span>
          </div>
        )}

        {allowRegister && (
          <div className="tabs">
            <button className={`tab ${tab === 'login' ? 'active' : ''}`} onClick={() => setTab('login')}>
              Kirish
            </button>
            <button className={`tab ${tab === 'register' ? 'active' : ''}`} onClick={() => setTab('register')}>
              Ro'yxatdan o'tish
            </button>
          </div>
        )}

        <form onSubmit={submit} className="col gap-4">
          {registering && (
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
            minLength={registering ? 8 : undefined}
          />
          <Button type="submit" size="lg" loading={loading} className="full" style={{ marginTop: 8 }}>
            {registering ? "Ro'yxatdan o'tish" : 'Kirish'}
          </Button>
        </form>

        {!registering && EMAIL_ENABLED && (
          <p className="muted" style={{ textAlign: 'center', fontSize: 13, marginTop: 12 }}>
            <a onClick={() => navigate('/forgot-password')} style={{ cursor: 'pointer', fontWeight: 600 }}>
              Parolni unutdingizmi?
            </a>
          </p>
        )}

        {!allowRegister && (
          <p className="muted" style={{ textAlign: 'center', fontSize: 13, marginTop: 20 }}>
            Yangi hisob kerakmi? Administrator bilan bog'laning.
          </p>
        )}

        <p className="muted" style={{ textAlign: 'center', fontSize: 13, marginTop: 12 }}>
          Havola orqali qo'shilmoqchimisiz?{' '}
          <a onClick={() => navigate('/')} style={{ cursor: 'pointer', fontWeight: 600 }}>
            Bosh sahifa
          </a>
        </p>
      </div>
    </div>
  )
}
