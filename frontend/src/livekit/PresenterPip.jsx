import { memo, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useEffect as useLayoutSync, useRef as useScrollRef, useState as useDraft } from 'react'
import { Hand, MessageSquare, Mic, MicOff, MonitorX, Send } from 'lucide-react'

// Suzuvchi oyna (Document Picture-in-Picture).
//
// ## Muammo
// Ustoz ekranini ulashganda brauzerdan chiqib PDF/kod/GeoGebra oynasiga o'tadi.
// Shu payt o'quvchi qo'l ko'tarsa yoki reaksiya yuborsa — ustoz buni KO'RMAYDI.
// Dars ichidagi eng muhim signal aynan eng muhim paytda yo'qoladi.
//
// ## Yechim
// Document PiP — brauzer beradigan, har doim boshqa oynalar USTIDA turadigan
// kichik oyna. Unga React portali orqali qo'l navbati va reaksiyalar chiqariladi.
//
// ## Nega `<video>` PiP emas
// Klassik PiP faqat videoni ko'chira oladi; bizga esa jonli ro'yxat va tugmalar
// kerak. Document PiP aynan shu uchun mavjud.
export const PIP_SUPPORTED = typeof window !== 'undefined' && 'documentPictureInPicture' in window

/**
 * Uslublarni PiP hujjatiga ko'chiradi.
 *
 * PiP oynasi ALOHIDA hujjat — u asosiy sahifaning CSS'ini meros olmaydi va
 * usiz panel uslubsiz "yalang'och HTML" bo'lib ko'rinadi.
 *
 * `adoptedStyleSheets` o'rniga tugunlarni nusxalaymiz: u ancha sodda va ikkala
 * muhitda ham ishlaydi — dev'da Vite CSS'ni `<style>` sifatida, production'da
 * `<link>` sifatida beradi. `cssRules` o'qish esa cross-origin varaqda
 * `SecurityError` beradi va uni ushlab o'tirish kerak bo'lardi.
 */
function copyStyles(pipDoc) {
  for (const node of document.querySelectorAll('style, link[rel="stylesheet"]')) {
    pipDoc.head.appendChild(node.cloneNode(true))
  }
}

export function PresenterPip({ open, onClose, onOpenFailed, width = 340, height = 520, children }) {
  const [pipWin, setPipWin] = useState(null)
  // Callback'lar ref orqali: ular har renderda yangi bo'ladi, effekt esa faqat
  // `open` o'zgarganda ishlashi kerak (aks holda oyna yopilib-ochilib turardi).
  const onCloseRef = useRef(onClose)
  const onFailRef = useRef(onOpenFailed)
  // "Latest ref" naqshi. ESLint render paytida ref'ga yozishni ogohlantiradi
  // (u concurrent rejimda xavfsiz emas), lekin bu yerda ATAYLAB: effektga
  // ko'chirilsa ref joriy render effekti uchun eskirgan bo'lib qolardi va
  // oyna yopilganda eski callback chaqirilardi. Ilova concurrent
  // xususiyatlarini (Suspense bilan o'tuvchi render) ishlatmaydi.
  /* eslint-disable react-hooks/refs */
  onCloseRef.current = onClose
  onFailRef.current = onOpenFailed
  /* eslint-enable react-hooks/refs */

  useEffect(() => {
    if (!open || !PIP_SUPPORTED) return undefined

    let cancelled = false
    let win = null
    // O'ZIMIZ yopayotganimizni belgilaydi: `pagehide` bunda ham chiqadi, lekin
    // uni "ustoz yopdi" deb hisoblash xato bo'lardi (holat 'dismissed' ga
    // tushib, keyingi ulashishda oyna ochilmay qolardi).
    let selfClosing = false

    const handleHide = () => {
      if (!selfClosing) onCloseRef.current?.()
    }

    ;(async () => {
      try {
        win = await window.documentPictureInPicture.requestWindow({ width, height })
      } catch {
        // Odatda `NotAllowedError`: Document PiP foydalanuvchi harakatini talab
        // qiladi va ekran ulashish `await`idan keyin bu "ruxsat" sarflangan
        // bo'lishi mumkin. Bu XATO emas — chaqiruvchi qo'lda ochish tugmasini
        // ko'rsatadi (bir bosish = kafolatlangan harakat).
        if (!cancelled) onFailRef.current?.()
        return
      }
      if (cancelled) {
        win.close()
        return
      }
      copyStyles(win.document)
      win.document.title = 'Jonly — dars signallari'
      win.document.body.classList.add('pip-body')
      win.addEventListener('pagehide', handleHide)
      setPipWin(win)
    })()

    return () => {
      cancelled = true
      setPipWin(null)
      if (win) {
        selfClosing = true
        win.removeEventListener('pagehide', handleHide)
        win.close()
      }
    }
  }, [open, width, height])

  if (!pipWin) return null
  return createPortal(children, pipWin.document.body)
}

/**
 * PiP ichidagi panel — ustozga ekran ulashish paytida KERAK bo'ladigan minimum.
 *
 * Ataylab kam narsa: oyna kichik va ustozning diqqati asosan ulashilayotgan
 * kontentda. Shuning uchun faqat (a) kim qo'l ko'tardi, (b) qanday reaksiya
 * keldi, (c) CHAT — o'quvchi savolini o'qish va qisqa javob berish,
 * (d) mikrofon va ulashishni to'xtatish.
 *
 * Yuqoridagi chiziqli ramka — "ekran ulashilmoqda" belgisi. PiP oynasi har doim
 * boshqa oynalar ustida turgani uchun bu ramka ustoz qaysi ilovada bo'lishidan
 * qat'i nazar ko'rinadi: brauzer sahifasi ichidagi ramka esa aynan ulashish
 * paytida (ustoz boshqa oynada bo'lganda) ko'rinmay qoladi.
 */
export const PresenterPanel = memo(function PresenterPanel({
  hands,
  reactions,
  chat,
  micOn,
  onToggleMic,
  onStopShare,
  onLowerHand,
  onSendChat,
}) {
  const [draft, setDraft] = useDraft('')
  const endRef = useScrollRef(null)

  // Yangi xabar kelganda pastga suramiz — ustoz oxirgi savolni ko'rsin.
  useLayoutSync(() => {
    endRef.current?.scrollIntoView({ block: 'end' })
  }, [chat.length])

  function submit(e) {
    e.preventDefault()
    const body = draft.trim()
    if (!body) return
    onSendChat(body)
    setDraft('')
  }

  return (
    <div className="pip pip--sharing">
      <div className="pip__section">
        <div className="pip__title">
          <Hand size={13} /> Qo'l ko'targanlar ({hands.length})
        </div>
        {hands.length === 0 ? (
          <p className="pip__empty">Hozircha yo'q</p>
        ) : (
          <ul className="pip__list">
            {hands.map((h, i) => (
              <li key={h.identity}>
                <span className="rail-num">{i + 1}</span>
                <span className="truncate">{h.name}</span>
                <button className="pip__x" onClick={() => onLowerHand(h.identity)} title="Tushirish">
                  ×
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {reactions.length > 0 && (
        <div className="pip__section">
          <div className="pip__reactions">
            {reactions.map((r) => (
              <span key={r.id} className="pip__reaction">
                {r.emoji} <span className="truncate">{r.name}</span>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Chat — ekran ulashish paytida ustoz uchun eng muhim kanal: o'quvchi
          savolni ovoz bilan emas, yozib beradi. Bu yerda oxirgi xabarlar va
          qisqa javob maydoni bor; to'liq tarix asosiy oynada qoladi. */}
      <div className="pip__section pip__section--grow pip__chat">
        <div className="pip__title">
          <MessageSquare size={13} /> Chat
        </div>
        {chat.length === 0 ? (
          <p className="pip__empty">Hali xabar yo'q</p>
        ) : (
          <div className="pip__msgs">
            {chat.map((m) => (
              <div key={m.id} className={`pip__msg ${m.self ? 'pip__msg--self' : ''}`}>
                <span className="pip__msg-name">
                  {m.self ? 'Siz' : m.name}
                  {m.toIdentity && <span className="pip__msg-dm">shaxsiy</span>}
                </span>
                {/* Suzuvchi oyna kichkina — fayl KARTOCHKASI sig'maydi, lekin
                    "kimdir fayl yubordi" signali yo'qolmasligi kerak: aks holda
                    ekran ulashayotgan ustoz uchun xabar BO'SH ko'rinardi. */}
                <span className="pip__msg-body">
                  {m.file ? `📎 ${m.file.name}${m.body ? ` — ${m.body}` : ''}` : m.body}
                </span>
              </div>
            ))}
            <div ref={endRef} />
          </div>
        )}
        <form className="pip__reply" onSubmit={submit}>
          <input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="Javob yozing…"
            maxLength={2000}
          />
          <button type="submit" aria-label="Yuborish">
            <Send size={14} />
          </button>
        </form>
      </div>

      <div className="pip__actions">
        <button className={`pip__btn ${micOn ? '' : 'pip__btn--danger'}`} onClick={onToggleMic}>
          {micOn ? <Mic size={16} /> : <MicOff size={16} />}
          {micOn ? 'Mikrofon' : 'Ovozsiz'}
        </button>
        <button className="pip__btn pip__btn--danger" onClick={onStopShare}>
          <MonitorX size={16} /> To'xtatish
        </button>
      </div>
    </div>
  )
})
