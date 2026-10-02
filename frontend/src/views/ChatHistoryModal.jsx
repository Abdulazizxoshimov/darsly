import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MessageSquare, Trash2 } from 'lucide-react'
import { chatHistory } from '../api/chat'
import { qk, useDeleteChatMessage } from '../store/data'
import { errorText } from '../api/api'
import { Modal } from '../components/Modal'
import { Spinner } from '../components/Spinner'
import { Button } from '../components/Button'
import { ChatFileCard } from '../components/ChatFileCard'
import { formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'

// Dars chat tarixi — dars TUGAGANDAN keyin ham o'qiladi (JWT yo'li:
// GET /lessons/:id/chat). Mahsulot sababi: chat DB'da saqlanadi, lekin xona
// yopilgach uni ko'rishning hech qanday yo'li yo'q edi.
//
// Moderatsiya shu yerda ham ishlaydi: buzg'unchi xabar ko'pincha dars
// tugagandan keyin ko'zga tashlanadi, xonaga esa qaytib bo'lmaydi.
export function ChatHistoryModal({ lesson, onClose }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: qk.chatHistory(lesson.id),
    queryFn: () => chatHistory(lesson.id),
  })
  const del = useDeleteChatMessage()
  const [confirmDel, setConfirmDel] = useState(null)

  // Server kursor tartibida (yangi → eski) berishi mumkin — o'qish uchun
  // xronologik (eski → yangi) tartiblaymiz.
  const messages = [...(data || [])].sort(
    (a, b) => Date.parse(a.created_at || 0) - Date.parse(b.created_at || 0),
  )

  async function doDelete() {
    if (!confirmDel) return
    try {
      await del.mutateAsync({ lessonId: lesson.id, messageId: confirmDel.id })
      setConfirmDel(null)
      toast.info("Xabar o'chirildi")
    } catch (e) {
      toast.error(errorText(e, 'Xabarni o‘chirib bo‘lmadi'))
    }
  }

  return (
    <Modal open onClose={onClose} title="Dars chati" width={480}>
      <p className="text-2 truncate" style={{ fontSize: 13, marginTop: -8, marginBottom: 16 }}>
        {lesson.title}
      </p>

      {isLoading ? (
        <div className="row center" style={{ height: 120, justifyContent: 'center' }}>
          <Spinner size={26} />
        </div>
      ) : isError ? (
        <p className="text-2" style={{ fontSize: 14, textAlign: 'center', padding: '32px 0' }}>
          Chat tarixini yuklab bo'lmadi. Qayta urinib ko'ring.
        </p>
      ) : !messages.length ? (
        <div className="col center" style={{ padding: '32px 0', gap: 10 }}>
          <div className="empty__icon" style={{ marginBottom: 0 }}>
            <MessageSquare size={24} />
          </div>
          <p className="text-2" style={{ fontSize: 14, margin: 0 }}>Bu darsda chat yozilmagan</p>
        </div>
      ) : (
        <div className="col gap-3" style={{ maxHeight: '55vh', overflowY: 'auto', paddingRight: 4 }}>
          {messages.map((m) => (
            <div key={m.id} className="chat-msg">
              <div className="chat-msg__name">
                {m.sender_name || m.sender_identity}
                {m.to_identity && <span className="chat-msg__dm">shaxsiy</span>}
                <span className="muted" style={{ marginLeft: 6 }}>{formatDateTime(m.created_at)}</span>
              </div>
              <div className="chat-msg__row">
                <button
                  className="chat-msg__del"
                  onClick={() => setConfirmDel(m)}
                  aria-label="Xabarni o'chirish"
                  title="Xabarni o'chirish"
                >
                  <Trash2 size={13} />
                </button>
                <div className={`chat-bubble ${m.to_identity ? 'chat-bubble--dm' : ''}`}>
                  {m.body && <span className="chat-bubble__text">{m.body}</span>}
                  {m.file && <ChatFileCard file={m.file} />}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {confirmDel && (
        <div className="confirm-inline">
          <p style={{ fontSize: 14, fontWeight: 600, margin: '0 0 4px' }}>Xabarni o'chirasizmi?</p>
          <p className="text-2" style={{ fontSize: 13, margin: '0 0 12px' }}>
            Xabar dars tarixidan butunlay o‘chadi. Bu amalni bekor qilib bo‘lmaydi.
          </p>
          <div className="row gap-2" style={{ justifyContent: 'flex-end' }}>
            <Button variant="ghost" size="sm" onClick={() => setConfirmDel(null)} disabled={del.isPending}>
              Bekor qilish
            </Button>
            <Button variant="danger" size="sm" onClick={doDelete} loading={del.isPending}>
              Ha, o'chirish
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}
