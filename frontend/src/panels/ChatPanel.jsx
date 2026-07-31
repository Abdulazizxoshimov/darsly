import { memo, useEffect, useMemo, useRef, useState } from 'react'
import { AlertTriangle, Loader2, Lock, Paperclip, Send, Trash2, X } from 'lucide-react'
import { fileRejectReason, uploadErrorText, FILE_ACCEPT, MAX_FILE_LABEL } from '../api/chat'
import { ChatFileCard } from '../components/ChatFileCard'
import { Modal } from '../components/Modal'
import { Button } from '../components/Button'

// Dars chati. Xabarlar SERVERDA saqlanadi (tarix, yozuv, moderatsiya) va
// LiveKit data-channel orqali real-vaqtda yetkaziladi.
//
// Shaxsiy xabar: qabul qiluvchi tanlansa xabar FAQAT ikki tomonga yetkaziladi
// (server `destination_identities` bilan yuboradi), ya'ni maxfiylik klientning
// "ko'rsatmaslik" xushmuomalaligiga tayanmaydi.
//
// Fayl: mentor ham, o'quvchi ham yubora oladi (maks 20 MB). Yuklash progressi
// ko'rsatiladi — sekin mobil internetda progresssiz "ilova qotdi" taassuroti
// paydo bo'ladi va foydalanuvchi tugmani qayta-qayta bosadi.
//
// O'chirish (moderatsiya): FAQAT ustozda. Xabar hammadan izsiz yo'qoladi —
// shuning uchun tasodifiy bosishdan tasdiq bilan himoyalangan.
export const ChatPanel = memo(function ChatPanel({
  entries,
  participants,
  localId,
  onSend,
  onSendFile,
  canDelete = false,
  onDelete,
  historyError = false,
  onRetryHistory,
  onClose,
}) {
  const [text, setText] = useState('')
  const [to, setTo] = useState('') // '' = hammaga
  const [upload, setUpload] = useState(null) // {name, pct} — yuklanayotgan fayl
  const [fileError, setFileError] = useState(null)
  const [sendError, setSendError] = useState(null)
  const [confirmDel, setConfirmDel] = useState(null) // o'chiriladigan xabar
  const [deleting, setDeleting] = useState(false)
  const endRef = useRef(null)
  const fileRef = useRef(null)
  const abortRef = useRef(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  // Tanlangan qabul qiluvchi xonadan chiqib ketsa — "hammaga"ga qaytamiz,
  // aks holda xabar yo'q odamga ketardi (va jimgina yo'qolardi).
  //
  // `useMemo` SHART: `participants` har LiveKit hodisasida yangi massiv bo'ladi
  // va memosiz quyidagi effekt har renderda qayta ishlardi.
  const others = useMemo(
    () => participants.filter((p) => p.identity !== localId),
    [participants, localId],
  )
  useEffect(() => {
    if (to && !others.some((p) => p.identity === to)) setTo('')
  }, [to, others])

  function submit(e) {
    e.preventDefault()
    const body = text.trim()
    if (!body) return
    // `onSend` `false` qaytarsa xabar YUBORILMADI (klient tomondagi tezlik
    // cheklovi). Matnni tozalash — uni izsiz yo'qotish degani bo'lardi:
    // foydalanuvchi yozganini qaytadan terishga majbur bo'lardi.
    const accepted = onSend(body, to)
    if (accepted === false) {
      setSendError('Juda tez yubordingiz — bir soniyadan so‘ng urinib ko‘ring')
      return
    }
    setSendError(null)
    setText('')
  }

  async function pickFile(e) {
    const file = e.target.files?.[0]
    e.target.value = '' // bir xil faylni qayta tanlash ham hodisa bersin
    if (!file || !onSendFile) return
    setFileError(null)

    // Klient tomonda oldindan tekshiramiz: 20 MB'ni yuklab, so'ng serverdan
    // 400 olish — sekin internetda bir necha daqiqa isrof degani.
    const reason = fileRejectReason(file)
    if (reason) {
      setFileError(reason)
      return
    }

    const ctrl = new AbortController()
    abortRef.current = ctrl
    setUpload({ name: file.name, pct: 0 })
    try {
      await onSendFile(file, to, (pct) => setUpload((u) => (u ? { ...u, pct } : u)), ctrl.signal)
    } catch (err) {
      setFileError(uploadErrorText(err))
    } finally {
      abortRef.current = null
      setUpload(null)
    }
  }

  function cancelUpload() {
    abortRef.current?.abort()
  }

  async function doDelete() {
    if (!confirmDel || !onDelete) return
    setDeleting(true)
    try {
      await onDelete(confirmDel.id)
      setConfirmDel(null)
    } finally {
      setDeleting(false)
    }
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
        {/* Yuklash davomida panel yopilmaydi: uning holati (progress va xato)
            shu komponentda yashaydi va yopilsa foydalanuvchi fayl yetib
            bordimi yoki yo'qmi bilmay qolardi. Chiqish yo'li bor — «Bekor
            qilish» tugmasi. */}
        <button
          className="icon-btn"
          onClick={onClose}
          disabled={!!upload}
          aria-label="Yopish"
          title={upload ? 'Fayl yuklanmoqda — kuting yoki bekor qiling' : 'Yopish'}
        >
          <X size={20} />
        </button>
      </div>

      <div className="chat-list">
        {/* Tarix yuklanmagan bo'lsa "Hali xabar yo'q" YOLG'ON bo'ladi: xabarlar
            bor, biz ularni ololmadik. Farqni aytib, qayta urinish beramiz. */}
        {historyError && (
          <div className="chat-history-error">
            <AlertTriangle size={14} />
            <span className="grow">Eski xabarlarni yuklab bo‘lmadi</span>
            {onRetryHistory && (
              <button className="link-btn" onClick={onRetryHistory}>
                Qayta urinish
              </button>
            )}
          </div>
        )}
        {entries.length === 0 && !historyError && (
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
              <div className="chat-msg__row">
                {canDelete && onDelete && (
                  <button
                    className="chat-msg__del"
                    onClick={() => setConfirmDel(m)}
                    aria-label="Xabarni o'chirish"
                    title="Xabarni o'chirish"
                  >
                    <Trash2 size={13} />
                  </button>
                )}
                <div className={`chat-bubble ${dm ? 'chat-bubble--dm' : ''}`}>
                  {m.body && <span className="chat-bubble__text">{m.body}</span>}
                  {m.file && <ChatFileCard file={m.file} />}
                </div>
              </div>
            </div>
          )
        })}
        <div ref={endRef} />
      </div>

      {/* Yuklash progressi — foydalanuvchi qaysi fayl va qancha qolganini ko'radi. */}
      {upload && (
        <div className="chat-upload" role="status">
          <Loader2 size={14} style={{ animation: 'spin 0.7s linear infinite', flexShrink: 0 }} />
          <span className="chat-upload__name truncate">{upload.name}</span>
          <span className="chat-upload__pct">{upload.pct}%</span>
          {/* Bekor qilish MAJBURIY: 20 MB sekin mobil internetda bir necha
              daqiqa ketadi va noto'g'ri fayl tanlangan bo'lsa uni to'xtatishning
              boshqa yo'li yo'q edi. */}
          <button className="link-btn" onClick={cancelUpload}>
            Bekor qilish
          </button>
          <div className="chat-upload__track">
            <div className="chat-upload__fill" style={{ width: `${upload.pct}%` }} />
          </div>
        </div>
      )}

      {(fileError || sendError) && (
        <div className="chat-file-error" role="alert">
          <span className="grow">{fileError || sendError}</span>
          <button
            className="icon-btn"
            onClick={() => {
              setFileError(null)
              setSendError(null)
            }}
            aria-label="Xatoni yopish"
          >
            <X size={14} />
          </button>
        </div>
      )}

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
        {onSendFile && (
          <>
            <input
              ref={fileRef}
              type="file"
              accept={FILE_ACCEPT}
              onChange={pickFile}
              style={{ display: 'none' }}
              data-testid="chat-file-input"
            />
            <button
              type="button"
              className="btn btn--icon btn--secondary"
              onClick={() => fileRef.current?.click()}
              disabled={!!upload}
              aria-label="Fayl biriktirish"
              title={`Fayl biriktirish (maks ${MAX_FILE_LABEL})`}
            >
              <Paperclip size={18} />
            </button>
          </>
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

      <Modal
        open={!!confirmDel}
        onClose={() => !deleting && setConfirmDel(null)}
        title="Xabarni o'chirasizmi?"
        width={380}
      >
        <p className="text-2" style={{ fontSize: 14, marginBottom: 18 }}>
          Xabar barcha ishtirokchilardan va dars tarixidan butunlay o‘chadi. Bu amalni bekor qilib bo‘lmaydi.
        </p>
        <div className="row gap-3" style={{ justifyContent: 'flex-end' }}>
          <Button variant="ghost" onClick={() => setConfirmDel(null)} disabled={deleting}>
            Bekor qilish
          </Button>
          <Button variant="danger" onClick={doDelete} disabled={deleting}>
            {deleting ? 'O‘chirilmoqda…' : "Ha, o'chirish"}
          </Button>
        </div>
      </Modal>
    </div>
  )
})
