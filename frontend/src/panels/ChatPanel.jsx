import { memo, useEffect, useRef, useState } from 'react'
import { Lock, Send, X } from 'lucide-react'

// Dars chati. Xabarlar SERVERDA saqlanadi (tarix, yozuv, moderatsiya) va
// LiveKit data-channel orqali real-vaqtda yetkaziladi.
//
// Shaxsiy xabar: qabul qiluvchi tanlansa xabar FAQAT ikki tomonga yetkaziladi
// (server `destination_identities` bilan yuboradi), ya'ni maxfiylik klientning
// "ko'rsatmaslik" xushmuomalaligiga tayanmaydi.
export const ChatPanel = memo(function ChatPanel({ entries, participants, localId, onSend, onClose }) {
  const [text, setText] = useState('')
  const [to, setTo] = useState('') // '' = hammaga
  const endRef = useRef(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  // Tanlangan qabul qiluvchi xonadan chiqib ketsa — "hammaga"ga qaytamiz,
  // aks holda xabar yo'q odamga ketardi (va jimgina yo'qolardi).
  const others = participants.filter((p) => p.identity !== localId)
  useEffect(() => {
    if (to && !others.some((p) => p.identity === to)) setTo('')
  }, [to, others])

  function submit(e) {
    e.preventDefault()
    const body = text.trim()
    if (!body) return
    onSend(body, to)
    setText('')
  }

  const toName = others.find((p) => p.identity === to)?.name
  const nameOf = (id) => participants.find((p) => p.identity === id)?.name || id

  // Shaxsiy xabar yorlig'i render paytida hisoblanadi — ism jonli yangilanadi.
  const label = (m) => {
    if (!m.toIdentity) return null
    return m.self ? `→ ${nameOf(m.toIdentity)}` : 'sizga shaxsiy'
  }

  return (
    <div className="panel">
      <div className="panel__head">
        <h3 className="h2">Chat</h3>
        <button className="icon-btn" onClick={onClose} aria-label="Yopish">
          <X size={20} />
        </button>
      </div>

      <div className="chat-list">
        {entries.length === 0 && (
          <p className="muted" style={{ textAlign: 'center', fontSize: 14, marginTop: 24 }}>Hali xabar yo'q</p>
        )}
        {entries.map((m) => {
          const dm = label(m)
          return (
            <div key={m.id} className={`chat-msg ${m.self ? 'self' : ''}`}>
              <span className="chat-msg__name">
                {m.self ? 'Siz' : m.name}
                {dm && (
                  <span className="chat-msg__dm">
                    <Lock size={10} /> {dm}
                  </span>
                )}
              </span>
              <div className={`chat-bubble ${dm ? 'chat-bubble--dm' : ''}`}>{m.body}</div>
            </div>
          )
        })}
        <div ref={endRef} />
      </div>

      <form onSubmit={submit} className="chat-input">
        {others.length > 0 && (
          <select
            className="chat-to"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            aria-label="Kimga"
            title={to ? `Shaxsiy xabar: ${toName}` : 'Hammaga'}
          >
            <option value="">Hammaga</option>
            {others.map((p) => (
              <option key={p.identity} value={p.identity}>
                {p.name}
              </option>
            ))}
          </select>
        )}
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={to ? `${toName}ga shaxsiy…` : 'Xabar yozing…'}
          maxLength={2000}
        />
        <button type="submit" className="btn btn--icon" aria-label="Yuborish">
          <Send size={18} />
        </button>
      </form>
    </div>
  )
})
