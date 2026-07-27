import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Check, Link2 } from 'lucide-react'
import { Button } from '../components/Button'

const FEATURES = [
  'Dars yozib olish va yuklab olish',
  'Kutish xonasi va host boshqaruvi',
  "So'rovnomalar va bildirishnomalar",
]

const STATS = [
  ['540+', "faol o'quvchi"],
  ['4.9 / 5', "o'rtacha reyting"],
  ['1200+', "o'tkazilgan dars"],
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
          <div className="brand-logo">D</div> Darsly
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
              placeholder="darsly.uz/r/... havolasini joylashtiring"
              style={{ flex: 1, background: 'none', border: 'none', color: 'var(--text)', fontSize: 13.5, outline: 'none' }}
            />
            <Button variant="secondary" size="sm" onClick={goJoin}>
              Kirish
            </Button>
          </div>

          <div className="row gap-4" style={{ gap: 36, marginTop: 32 }}>
            {STATS.map(([v, l]) => (
              <div key={l}>
                <div style={{ fontSize: 26, fontWeight: 800 }}>{v}</div>
                <div className="muted" style={{ fontSize: 13 }}>{l}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="card card--pad">
          <div className="row gap-3" style={{ marginBottom: 20 }}>
            <div className="brand-logo" style={{ width: 64, height: 64, fontSize: 24 }}>A</div>
            <div>
              <div style={{ fontWeight: 700, fontSize: 16 }}>Aziz Karimov</div>
              <div className="text-2" style={{ fontSize: 13 }}>Matematika o'qituvchisi · 8 yillik tajriba</div>
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
