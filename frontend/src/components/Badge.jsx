import { LESSON_STATUS_UZ } from '../lib/format'

export function StatusBadge({ status }) {
  return (
    <span className={`badge badge--${status}`}>
      {status === 'live' && <span className="dot" style={{ background: 'currentColor', animation: 'pulseDot 1.6s infinite' }} />}
      {LESSON_STATUS_UZ[status] || status}
    </span>
  )
}

export function Badge({ children }) {
  return <span className="badge badge--neutral">{children}</span>
}
