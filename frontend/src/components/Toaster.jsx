import { useEffect, useState } from 'react'
import { CheckCircle2, Info, X, XCircle } from 'lucide-react'
import { toast } from '../lib/toast'

const ICON = { success: CheckCircle2, error: XCircle, info: Info }

export function Toaster() {
  const [items, setItems] = useState([])
  useEffect(() => toast.subscribe(setItems), [])

  return (
    <div className="toaster">
      {items.map((t) => {
        const Icon = ICON[t.kind] || Info
        return (
          <div key={t.id} className={`toast toast--${t.kind}`}>
            <Icon size={20} className="toast__icon" />
            <p className="toast__msg">{t.message}</p>
            <button className="icon-btn" onClick={() => toast.dismiss(t.id)} aria-label="Yopish">
              <X size={16} />
            </button>
          </div>
        )
      })}
    </div>
  )
}
