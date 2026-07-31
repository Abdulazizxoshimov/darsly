import { useEffect, useId, useRef } from 'react'
import { X } from 'lucide-react'

export function Modal({ open, onClose, title, children, width = 480 }) {
  const titleId = useId()
  const boxRef = useRef(null)
  const returnFocusRef = useRef(null)

  useEffect(() => {
    if (!open) return
    const onKey = (e) => e.key === 'Escape' && onClose()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  // Fokusni modalga olib kirish va yopilgach QAYTARISH.
  //
  // Busiz skrin-rider foydalanuvchisi uchun modal umuman "ochilmaydi": fokus
  // ortdagi sahifada qoladi va o'qiladigan matn ham o'sha yerdan davom etadi.
  // Bu ayniqsa TASDIQ oynalarida xavfli — xabarni o'chirish yoki darsni
  // yakunlash so'ralayotganini bilmay qolish mumkin.
  useEffect(() => {
    if (!open) return
    returnFocusRef.current = document.activeElement
    // Modal ichidagi birinchi interaktiv element (odatda "Yopish" yoki tugma).
    const focusable = boxRef.current?.querySelector(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    )
    ;(focusable || boxRef.current)?.focus()
    return () => {
      const prev = returnFocusRef.current
      if (prev && typeof prev.focus === 'function') prev.focus()
    }
  }, [open])

  if (!open) return null
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        ref={boxRef}
        className="modal"
        style={{ width }}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        tabIndex={-1}
      >
        {title && (
          <div className="modal__head">
            <h3 className="h2" id={titleId}>{title}</h3>
            <button className="icon-btn" onClick={onClose} aria-label="Yopish">
              <X size={20} />
            </button>
          </div>
        )}
        <div className="modal__body">{children}</div>
      </div>
    </div>
  )
}
