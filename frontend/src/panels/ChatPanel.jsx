import { useEffect, useRef, useState } from 'react'
import { Send, X } from 'lucide-react'

export function ChatPanel({ entries, onSend, onClose }) {
  const [text, setText] = useState('')
  const endRef = useRef(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  function submit(e) {
    e.preventDefault()
    const body = text.trim()
    if (!body) return
    onSend(body)
    setText('')
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
        {entries.map((m) => (
          <div key={m.id} className={`chat-msg ${m.self ? 'self' : ''}`}>
            <span className="chat-msg__name">{m.self ? 'Siz' : m.name}</span>
            <div className="chat-bubble">{m.body}</div>
          </div>
        ))}
        <div ref={endRef} />
      </div>
      <form onSubmit={submit} className="chat-input">
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Xabar yozing…"
          maxLength={2000}
        />
        <button type="submit" className="btn btn--icon" aria-label="Yuborish">
          <Send size={18} />
        </button>
      </form>
    </div>
  )
}
