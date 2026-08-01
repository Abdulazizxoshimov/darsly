import { memo, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Hand, MessageSquare, Mic, MicOff, MonitorX, Send } from 'lucide-react'

// Suzuvchi oyna (Document Picture-in-Picture).
//
// ## Muammo
// Ustoz ekranini ulashganda brauzerdan chiqib PDF/kod/GeoGebra oynasiga o'tadi.
// Shu payt o'quvchi qo'l ko'tarsa yoki savol yozsa — ustoz buni KO'RMAYDI.
// Dars ichidagi eng muhim signal aynan eng muhim paytda yo'qoladi.
//
// ## Yechim
// Document PiP — brauzer beradigan, har doim boshqa oynalar USTIDA turadigan
// kichik oyna. Unga React portali orqali IKKI mustaqil signal chiqariladi:
// chat (yangi xabar sanog'i bilan) va qo'l ko'targanlar (o'z sanog'i bilan).
//
// ## Nega kompakt (~236×96) default
// Oyna ustozning ish ekranini (slayd, kod, PDF) TO'SADI. Shuning uchun sukut
// holatida u faqat SIGNAL: "3 yangi xabar", "2 qo'l". Ustoz o'zi kerak deb
// bilganda chat panelini ochadi va oyna kengayadi — ya'ni joyni ustoz emas,
// vaziyat so'raydi.
//
// ## Nega `<video>` PiP emas
// Klassik PiP faqat videoni ko'chira oladi; bizga esa jonli ro'yxat va tugmalar
// kerak. Document PiP aynan shu uchun mavjud.
export const PIP_SUPPORTED = typeof window !== 'undefined' && 'documentPictureInPicture' in window

/** Kompakt (sukut) o'lchami — ikki signal tugmasi + mikrofon/to'xtatish qatori. */
// Balandlik hisobi (aynan sig'sin, ichki scroll paydo bo'lmasin):
// 8+8 padding + 2+2 ramka + 40 signal qatori + 6 oraliq + 34 boshqaruv = 100 (+4 zaxira).
export const PIP_COMPACT = { width: 236, height: 104 }

/** Qo'l ro'yxatining bitta qatori (px) — oyna balandligi shunga qarab o'sadi. */
const HAND_ROW = 28
/** Ro'yxatda ko'rsatiladigan maksimal qator; ortig'i scroll bilan. */
const HAND_ROWS_MAX = 5

const clamp = (v, lo, hi) => Math.min(Math.max(v, lo), hi)

/**
 * Badge matni: 9 dan ortig'i «9+».
 *
 * Aniq raqam (nuqta emas) ataylab: "xabar bor" bilan "12 ta xabar bor" ustoz
 * uchun ikki xil qaror. Lekin uch xonali son kichik tugmani buzadi, shuning
 * uchun 9+ da to'xtaydi — bu yerda aniqlik emas, "ko'p" degan signal yetarli.
 */
export function badgeText(n) {
  return n > 9 ? '9+' : String(n)
}

/**
 * Suzuvchi oyna o'lchami — SOF funksiya (testga tushadi).
 *
 * Uch holat:
 *   · kompakt — faqat ikki signal tugmasi va boshqaruv (ekranni deyarli to'smaydi);
 *   · qo'l ro'yxati ochiq — oyna faqat BO'YIGA o'sadi (eni o'zgarmaydi, ya'ni
 *     ustozning ko'z odati buzilmaydi), qatorlar soni bo'yicha, 5 qatorda to'xtaydi;
 *   · chat ochiq — ekranning ~1/4 eni (asoschi talabi), lekin 320 dan tor va 420
 *     dan keng emas: torida xabar o'qilmaydi, kengida ish ekrani yopiladi.
 */
export function pipWindowSize({ chatOpen = false, handsOpen = false, handCount = 0, screen } = {}) {
  const sc = screen || (typeof window !== 'undefined' ? window.screen : null)
  if (chatOpen) {
    const sw = sc?.availWidth || 1280
    const sh = sc?.availHeight || 800
    return {
      width: clamp(Math.round(sw * 0.25), 320, 420),
      height: clamp(Math.round(sh * 0.62), 380, 560),
    }
  }
  if (handsOpen) {
    const rows = clamp(handCount, 1, HAND_ROWS_MAX)
    return { width: PIP_COMPACT.width, height: PIP_COMPACT.height + 16 + rows * HAND_ROW }
  }
  return { ...PIP_COMPACT }
}

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

export function PresenterPip({
  open,
  onClose,
  onOpenFailed,
  width = PIP_COMPACT.width,
  height = PIP_COMPACT.height,
  children,
}) {
  const [pipWin, setPipWin] = useState(null)
  // Callback'lar va O'LCHAM ref orqali: ular har renderda yangi bo'ladi, oyna
  // ochish effekti esa faqat `open` o'zgarganda ishlashi kerak. Avval o'lcham
  // effekt deps'ida edi — chat ochilishi oynani YOPIB QAYTA OCHARDI (ya'ni
  // ustoz ko'z oldida oyna miltillardi va Document PiP "user gesture" talabi
  // tufayli u umuman qayta ochilmasligi ham mumkin edi).
  const onCloseRef = useRef(onClose)
  const onFailRef = useRef(onOpenFailed)
  const sizeRef = useRef({ width, height })
  // "Latest ref" naqshi. ESLint render paytida ref'ga yozishni ogohlantiradi
  // (u concurrent rejimda xavfsiz emas), lekin bu yerda ATAYLAB: effektga
  // ko'chirilsa ref joriy render effekti uchun eskirgan bo'lib qolardi va
  // oyna yopilganda eski callback chaqirilardi. Ilova concurrent
  // xususiyatlarini (Suspense bilan o'tuvchi render) ishlatmaydi.
  /* eslint-disable react-hooks/refs */
  onCloseRef.current = onClose
  onFailRef.current = onOpenFailed
  sizeRef.current = { width, height }
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
        win = await window.documentPictureInPicture.requestWindow({ ...sizeRef.current })
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
  }, [open])

  // Chat ochildi/yopildi yoki qo'l ro'yxati o'sdi — oyna YOPILMASDAN o'lchamini
  // o'zgartiradi. `resizeTo` ni brauzer rad etishi mumkin (masalan foydalanuvchi
  // oynani o'zi sudrab kattalashtirgan bo'lsa): bu halokat emas — panel ichki
  // scroll bilan baribir ishlaydi, shuning uchun jim o'tamiz.
  useEffect(() => {
    if (!pipWin || typeof pipWin.resizeTo !== 'function') return
    try {
      pipWin.resizeTo(width, height)
    } catch {
      /* brauzer rad etdi — panel scroll bilan ishlashda davom etadi */
    }
  }, [pipWin, width, height])

  if (!pipWin) return null
  return createPortal(children, pipWin.document.body)
}

/**
 * PiP ichidagi panel — ustozga ekran ulashish paytida KERAK bo'ladigan minimum.
 *
 * ## Ikki mustaqil signal (asoschi talabi)
 * Chat va qo'l ko'tarish — ikki BOSHQA hodisa turi va ular bir-birini
 * ko'mib qo'ymasligi kerak:
 *   · chat sanog'i — O'QILMAGAN xabar (panel ochilganda nolga tushadi);
 *   · qo'l sanog'i — HAQIQIY holat (faqat qo'l tushirilganda kamayadi;
 *     ro'yxatni ochib "o'qidim" deyish uni kamaytirmaydi, chunki qo'l hamon ko'tarilgan).
 * Shuning uchun qo'l ro'yxati chat panelining ICHIDA emas — u overlay'ning
 * o'zida, alohida mini-ro'yxat sifatida ochiladi.
 *
 * Panel BOSHQARILADIGAN (controlled): `chatOpen`/`handsOpen` yuqorida —
 * chunki aynan shu ikki holat OYNA O'LCHAMINI belgilaydi, o'lchamni esa
 * `PresenterPip` egallaydi.
 *
 * Chetidagi chiziqli ramka — "ekran ulashilmoqda" belgisi. PiP oynasi har doim
 * boshqa oynalar ustida turgani uchun bu ramka ustoz qaysi ilovada bo'lishidan
 * qat'i nazar ko'rinadi.
 */
export const PresenterPanel = memo(function PresenterPanel({
  hands,
  reactions,
  chat,
  unread = 0,
  chatOpen = false,
  handsOpen = false,
  onToggleChat,
  onToggleHands,
  micOn,
  onToggleMic,
  onStopShare,
  onLowerHand,
  onSendChat,
}) {
  const [draft, setDraft] = useState('')
  const endRef = useRef(null)
  // Yangi qo'l ko'tarilganda qisqa puls. Nega kerak: sanoq 1→2 bo'lishi
  // ustozning periferik ko'rishida SEZILMAYDI, harakat esa seziladi —
  // bu overlay'ning butun ma'nosi (ustoz boshqa ilovada, ekranga tikilib turmaydi).
  const [pulse, setPulse] = useState(false)
  const prevHands = useRef(hands.length)

  useEffect(() => {
    const grew = hands.length > prevHands.current
    prevHands.current = hands.length
    if (!grew) return undefined
    setPulse(true)
    const t = setTimeout(() => setPulse(false), 1400)
    return () => clearTimeout(t)
  }, [hands.length])

  // Yangi xabar kelganda pastga suramiz — ustoz oxirgi savolni ko'rsin.
  useEffect(() => {
    if (chatOpen) endRef.current?.scrollIntoView({ block: 'end' })
  }, [chat.length, chatOpen])

  function submit(e) {
    e.preventDefault()
    const body = draft.trim()
    if (!body) return
    onSendChat(body)
    setDraft('')
  }

  const handCount = hands.length

  return (
    <div className={`pip pip--sharing ${chatOpen ? 'pip--wide' : 'pip--compact'}`}>
      <div className="pip__bar">
        <button
          type="button"
          className={`pip__sig ${chatOpen ? 'pip__sig--on' : ''}`}
          onClick={onToggleChat}
          aria-pressed={chatOpen}
          title={unread > 0 ? `Chat — ${unread} yangi xabar` : 'Chat'}
          aria-label={unread > 0 ? `Chat, ${unread} yangi xabar` : 'Chat'}
        >
          <MessageSquare size={15} />
          <span className="pip__sig-label">Chat</span>
          {unread > 0 && <span className="pip__count">{badgeText(unread)}</span>}
        </button>

        <button
          type="button"
          className={`pip__sig ${handsOpen ? 'pip__sig--on' : ''} ${pulse ? 'pip__sig--pulse' : ''}`}
          onClick={onToggleHands}
          aria-pressed={handsOpen}
          title={handCount > 0 ? `${handCount} kishi qo'l ko'tardi` : "Qo'l ko'targanlar"}
          aria-label={
            handCount > 0 ? `Qo'l ko'targanlar, ${handCount} kishi` : "Qo'l ko'targanlar, hech kim yo'q"
          }
        >
          <Hand size={15} />
          <span className="pip__sig-label">Qo‘l</span>
          {handCount > 0 && <span className="pip__count pip__count--hand">{badgeText(handCount)}</span>}
        </button>
      </div>

      {/* Qo'l ro'yxati — overlay'ning O'ZIDA (chat ichida emas). Oqim bo'ylab
          chiziladi, absolyut emas: kichik oynada absolyut qatlam kesilib qolardi,
          oyna balandligi esa `pipWindowSize` bilan aynan shu ro'yxatga qarab o'sadi. */}
      {handsOpen && (
        <div className="pip__drop">
          {handCount === 0 ? (
            <p className="pip__empty">Hozircha hech kim qo‘l ko‘tarmadi</p>
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
      )}

      {/* Chat — ekran ulashish paytida ustoz uchun eng muhim kanal: o'quvchi
          savolni ovoz bilan emas, yozib beradi. */}
      {chatOpen && (
        <div className="pip__chat">
          {chat.length === 0 ? (
            <p className="pip__empty">Hali xabar yo‘q</p>
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
      )}

      {/* Reaksiyalar — suzuvchi qatlam (pointer-events yo'q): ular o'z-o'zidan
          3.4s da yo'qoladi va kompakt oynaning BALANDLIGINI o'zgartirmasligi
          kerak, aks holda har emoji oynani sakratardi. */}
      {reactions.length > 0 && (
        <div className="pip__reactions" aria-hidden="true">
          {reactions.map((r) => (
            <span key={r.id} className="pip__reaction">
              {r.emoji} <span className="truncate">{r.name}</span>
            </span>
          ))}
        </div>
      )}

      <div className="pip__actions">
        <button className={`pip__btn ${micOn ? '' : 'pip__btn--danger'}`} onClick={onToggleMic}>
          {micOn ? <Mic size={16} /> : <MicOff size={16} />}
          {micOn ? 'Mikrofon' : 'Ovozsiz'}
        </button>
        <button className="pip__btn pip__btn--danger" onClick={onStopShare}>
          <MonitorX size={16} /> To‘xtatish
        </button>
      </div>
    </div>
  )
})
