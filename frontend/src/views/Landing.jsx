import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Check, Link2 } from 'lucide-react'
import { Button } from '../components/Button'
import BrandMark from '../components/BrandMark'

const FEATURES = [
  'Dars yozib olish va yuklab olish',
  'Kutish xonasi va host boshqaruvi',
  "So'rovnomalar va bildirishnomalar",
  'Ilova o‘rnatmasdan — to‘g‘ridan-to‘g‘ri brauzerda',
]

export function Landing() {
  const navigate = useNavigate()
  const [joinUrl, setJoinUrl] = useState('')

  function goJoin() {
    const m = joinUrl.trim().match(/([A-Za-z0-9_-]{6,})\/?$/)
    if (m) navigate(`/r/${m[1]}`)
  }

  return (
    <div style={{ minHeight: '100%', overflow: 'auto' }}>
      <div className="landing-nav">
        <div className="row gap-3" style={{ fontWeight: 800, fontSize: 20 }}>
          <BrandMark /> jonly
        </div>
        <div className="row gap-4 text-2" style={{ fontSize: 14 }}>
          <Button variant="ghost" size="sm" onClick={() => navigate('/auth')}>
            Kirish
          </Button>
        </div>
      </div>

      <div className="hero">
        <div>
          <div className="pill">
            <span className="dot" /> Zoom'ga to'liq muqobil, o'zimizning video tizim
          </div>
          <h1 style={{ margin: '20px 0' }}>
            Jonli darslarni <span style={{ color: 'var(--accent-light)' }}>bir platformada</span> o'tkazing
          </h1>
          <p className="text-2" style={{ fontSize: 18, lineHeight: 1.6, maxWidth: 520, marginBottom: 32 }}>
            Video-chat, yozib olish, kutish xonasi va so'rovnomalar — barchasi brauzerda, ilova o'rnatmasdan.
          </p>
          <div className="row gap-3 wrap" style={{ marginBottom: 40 }}>
            <Button size="lg" onClick={() => navigate('/auth')}>
              Boshlash
            </Button>
            <Button variant="ghost" size="lg" onClick={() => navigate('/auth')}>
              Ustoz sifatida kirish
            </Button>
          </div>

          <div
            className="row gap-2"
            style={{
              background: 'var(--card)',
              border: '1px solid var(--border-strong)',
              borderRadius: 12,
              padding: '6px 6px 6px 16px',
              maxWidth: 440,
            }}
          >
            <Link2 size={16} color="var(--text-3)" />
            <input
              value={joinUrl}
              onChange={(e) => setJoinUrl(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && goJoin()}
              placeholder="jonly.uz/r/... havolasini joylashtiring"
              style={{ flex: 1, background: 'none', border: 'none', color: 'var(--text)', fontSize: 13.5, outline: 'none' }}
            />
            <Button variant="secondary" size="sm" onClick={goJoin}>
              Kirish
            </Button>
          </div>

          <div className="text-2" style={{ fontSize: 13.5, marginTop: 24, maxWidth: 440, lineHeight: 1.6 }}>
            Yangi platforma — birinchi ustozlar uchun boshlash bepul. Havola orqali
            o'quvchilar ro'yxatdan o'tmasdan qo'shiladi.
          </div>
        </div>

        <div className="card card--pad">
          <div className="row gap-3" style={{ marginBottom: 20 }}>
            <BrandMark size={56} />
            <div>
              <div style={{ fontWeight: 700, fontSize: 16 }}>Jonly</div>
              <div className="text-2" style={{ fontSize: 13 }}>Video-dars platformasi</div>
            </div>
          </div>
          <div className="text-2" style={{ fontSize: 13, fontWeight: 700, marginBottom: 10 }}>Platformada mavjud:</div>
          <div className="col gap-2">
            {FEATURES.map((f) => (
              <div key={f} className="row gap-2" style={{ fontSize: 13.5, color: 'var(--text-bright)' }}>
                <Check size={15} color="var(--success)" /> {f}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
