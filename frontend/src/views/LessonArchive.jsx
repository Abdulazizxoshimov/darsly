import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowLeft,
  Clock,
  Download,
  FileDown,
  Film,
  FolderOpen,
  Loader2,
  MessageSquare,
  RotateCcw,
  Trash2,
  VideoOff,
} from 'lucide-react'
import {
  useLessonArchive,
  useRecordingStatus,
  useRestoreRecording,
  useDeleteChatMessage,
} from '../store/data'
import { getChatTranscript } from '../api/archive'
import { errorText } from '../api/api'
import { saveBlob } from '../lib/download'
import { Button } from '../components/Button'
import { PageLoader, Spinner } from '../components/Spinner'
import { ChatFileCard } from '../components/ChatFileCard'
import {
  formatBytes,
  formatDateTime,
  formatDuration,
  formatExpiry,
  formatOffset,
  formatSize,
  expiresInDays,
} from '../lib/format'
import { toast } from '../lib/toast'

// ── Dars arxivi (PRODUCT.md «Dars arxivi va Telegram saqlash», ish №20) ──────
//
// Zoom'dan farqi va butun sahifaning MA'NOSI: video va chat YONMA-YON turadi,
// chatdagi vaqtni bosganda video o'sha daqiqaga sakraydi. Ustoz "Ali qayerda
// savol bergan edi?" degan savolga bir bosishda javob topadi — yozuvni
// boshdan ko'rib chiqmasdan.
//
// Ikkinchi qat'iy qoida: CHAT VA MATERIALLAR HAR DOIM KO'RINADI. Video
// serverdan o'chgan bo'lishi mumkin (30 kun retention → Telegram arxivi),
// chat esa DB'da — u yo'qolmaydi. Shuning uchun video holati chat qismini
// hech qachon yashirmaydi.

// Yozuv holatining "kutish"lari — shu holatlarda poll qilinadi.
// `recording` ham shu yerda: dars endi tugagan bo'lsa yozuv bir necha soniya
// shu holatda turadi va u ham o'zi yangilanishi kerak (izchillik).
const PENDING = new Set(['recording', 'processing', 'restoring'])

// Poll CHEKSIZ bo'lmasligi kerak. Backend ishchisi yiqilsa yoki fayl juda
// katta bo'lsa status `restoring`/`processing` da qotib qolishi mumkin —
// bunda klient soatlab har 5 soniyada so'rov yuborib, foydalanuvchi esa
// aylanuvchi spinnerdan boshqa hech nima ko'rmasdi. Muddat tugagach poll
// to'xtaydi va sabab bilan «qayta urinish» taklif qilinadi.
const MAX_WAIT_MS = 5 * 60_000

/**
 * `active` bo'lgandan `ms` o'tgach `true` bo'ladi; `active` o'chsa yoki
 * `resetKey` o'zgarsa (qayta urinish) noldan boshlanadi.
 *
 * Effekt TANASIDA `setState` yo'q (faqat taymer va tozalash) — shuning uchun
 * "cascading render" ogohlantirishi ham chiqmaydi.
 */
function useElapsed(active, ms, resetKey = 0) {
  const [fired, setFired] = useState(false)
  useEffect(() => {
    if (!active) return undefined
    const t = setTimeout(() => setFired(true), ms)
    return () => {
      clearTimeout(t)
      setFired(false)
    }
  }, [active, ms, resetKey])
  return active && fired
}

export function LessonArchive() {
  const { id } = useParams()
  // `key` — boshqa darsga o'tilganda BUTUN holat nolga qaytadi. Router
  // komponentni qayta ishlatadi va usiz oldingi darsning pleyer vaqti,
  // «tiklash boshlandi» bayrog'i va poll oralig'i yangi darsga o'tib ketardi.
  return <ArchiveView key={id} lessonId={id} />
}

function ArchiveView({ lessonId: id }) {
  const qc = useQueryClient()
  const { data, isLoading, isError, error, refetch, isFetching } = useLessonArchive(id)

  const videoRef = useRef(null)
  // Video shkalasidagi joriy vaqt — chat xabarini ajratib ko'rsatish uchun.
  // Butun soniyagacha yaxlitlanadi: `timeupdate` sekundiga ~4 marta keladi va
  // har biri uchun qayta render qilish bekorchi ish bo'lardi.
  const [currentSec, setCurrentSec] = useState(0)

  const rec = data?.recording || null
  // `useMemo` — har renderda YANGI bo'sh massiv yasalsa quyidagi highlight
  // hisobi (`activeMsgId`) har safar qaytadan ishlab ketardi.
  const chat = useMemo(() => data?.chat || [], [data])
  const materials = data?.materials || []
  const lesson = data?.lesson || null

  // ── Yozuv holati: arxiv javobi + poll ────────────────────────────────────
  // Arxiv javobi bir marta keladi; `processing`/`restoring` esa vaqt bilan
  // o'zgaradi. Shuning uchun joriy holat ikki manbadan: poll javobi ustunroq.
  const [pollAfterS, setPollAfterS] = useState(5)
  const [restoreStarted, setRestoreStarted] = useState(false)
  const archiveStatus = rec?.status || null

  const wantPoll = PENDING.has(archiveStatus) || restoreStarted

  // Poll DEADLINE'i. Taymer `wantPoll` dan boshlanadi — POLL NATIJASIDAN emas:
  // aks holda «qancha kutdik» «nimani kutyapmiz» ga bog'lanib aylanma
  // bog'liqlik hosil bo'lardi. Muddat tugagach `enabled` o'chadi va so'rovlar
  // to'xtaydi (aks holda backend ishchisi yiqilganda klient soatlab
  // so'rayverardi). `attempt` — «qayta urinish» taymerni noldan boshlaydi.
  const [attempt, setAttempt] = useState(0)
  const stalled = useElapsed(wantPoll, MAX_WAIT_MS, attempt)

  const { data: polled, isError: pollError } = useRecordingStatus(rec?.id, {
    enabled: wantPoll && !stalled,
    intervalMs: pollAfterS * 1000,
  })
  // Kutish holati POLL javobidan hisoblanadi (u arxivnikidan yangiroq).
  const status = polled?.status || archiveStatus

  // Tiklash YIQILDI: backend `failRestore` da statusni `archived` ga qaytaradi.
  // Busiz mentor tugmani bosib, bir daqiqa kutib, AYNAN o'sha kartochkaga
  // qaytardi — bosilgan-bosilmagani bilinmasdi.
  const restoreFailed = restoreStarted && status === 'archived'

  // Tayyor bo'lgach PRESIGNED havola kerak — u faqat arxiv javobida keladi.
  // Poll `ready` deganini eshitganda arxivni bir marta qayta so'raymiz va
  // pleyer o'zi paydo bo'ladi (foydalanuvchi hech nima bosmaydi).
  useEffect(() => {
    if (polled?.status === 'ready' && archiveStatus && archiveStatus !== 'ready') {
      qc.invalidateQueries({ queryKey: ['lesson-archive', id] })
    }
  }, [polled?.status, archiveStatus, id, qc])

  const restore = useRestoreRecording()
  async function doRestore() {
    if (!rec?.id) return
    try {
      const res = await restore.mutateAsync(rec.id)
      setPollAfterS(res.poll_after_s)
      setRestoreStarted(true)
      setAttempt((a) => a + 1)
      if (res.status === 'ready') {
        // Fayl allaqachon serverda edi — bu XATO emas, shunchaki tez yo'l.
        qc.invalidateQueries({ queryKey: ['lesson-archive', id] })
      }
    } catch (e) {
      toast.error(errorText(e, 'Videoni tiklab bo‘lmadi'))
    }
  }

  // ── Presigned havolaning muddati ─────────────────────────────────────────
  // Havola 1 soatlik. Sahifa undan uzoq ochiq tursa pleyer JIMGINA qora
  // qolardi — hech qanday xabar yo'q, chat vaqtini bosish ham ishlamaydi.
  // Birinchi xatoda havolani yangilab ko'ramiz, ikkinchisida sabab aytamiz.
  const [urlBroken, setUrlBroken] = useState(false)
  const urlRetried = useRef(false)
  function handleVideoError() {
    if (!urlRetried.current) {
      urlRetried.current = true
      refetch()
      return
    }
    setUrlBroken(true)
  }
  function reloadVideo() {
    urlRetried.current = false
    setUrlBroken(false)
    setAttempt((a) => a + 1)
    refetch()
  }

  const canJump = status === 'ready' && !!rec?.url && !urlBroken

  // `useCallback` — bu funksiya har bir chat qatoriga tushadi va uning
  // identiteti o'zgarsa `memo` qilingan qatorlar baribir qayta chizilardi.
  const jumpTo = useCallback((sec) => {
    const v = videoRef.current
    if (!v) return
    v.currentTime = Math.max(0, sec)
    setCurrentSec(Math.floor(sec))
    // `play()` ba'zi brauzerlarda rad etilgan promise qaytaradi (avtoijro
    // siyosati) — uni ushlamasak konsolda ushlanmagan xato chiqadi.
    const p = v.play?.()
    if (p && typeof p.catch === 'function') p.catch(() => {})
  }, [])

  // Joriy vaqtga mos xabar — `offset_sec` <= currentSec bo'lgan ENG OXIRGISI.
  // Ro'yxat adapterda tartiblangan, shuning uchun oddiy chiziqli qidiruv yetarli.
  const activeMsgId = useMemo(() => {
    if (!canJump || !chat.length) return null
    let hit = null
    for (const m of chat) {
      if (m.offset_sec <= currentSec) hit = m.id
      else break
    }
    return hit
  }, [chat, currentSec, canJump])

  if (isLoading) {
    return (
      <div className="page">
        <div style={{ height: 320 }}>
          <PageLoader label="Arxiv yuklanmoqda…" />
        </div>
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="page">
        <BackLink />
        <div className="empty">
          <div className="empty__icon">
            <AlertTriangle size={26} />
          </div>
          {/* Serverning ROSTKI sababi ko'rsatiladi (403 «ruxsat yo'q»,
              TIMEOUT «internet sekin», …) — bitta umumiy matn ostida
              yashirilsa foydalanuvchi nima qilishni bilmay qolardi. */}
          <p className="text-2" style={{ fontSize: 14, marginBottom: 16 }}>
            {errorText(error, 'Dars arxivini yuklab bo‘lmadi')}
          </p>
          <p className="muted" style={{ fontSize: 13, marginTop: -8, marginBottom: 16 }}>
            Dars o‘chirilgan bo‘lishi yoki internet uzilgan bo‘lishi mumkin.
          </p>
          <Button variant="secondary" onClick={() => refetch()} loading={isFetching}>
            <RotateCcw size={16} /> Qayta urinish
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="page page--wide">
      <BackLink />

      <div className="arch-head">
        <div className="grow" style={{ minWidth: 0 }}>
          <h1 className="h1 truncate">{lesson?.title || 'Dars arxivi'}</h1>
          <div className="arch-head__meta">
            <span>{formatDateTime(lesson?.started_at || lesson?.scheduled_at)}</span>
            <span className="arch-head__dot" aria-hidden="true" />
            <span className="row gap-1">
              <Clock size={13} />
              {rec?.duration_sec
                ? formatDuration(rec.duration_sec)
                : `${lesson?.duration_min || 0} daq`}
            </span>
            {/* Hajm faqat fayl MAVJUD bo'lganda ko'rsatiladi (serverda yoki
                Telegramda). `expired`/`failed` da «180 MB» raqami yo'q faylni
                bor deb ko'rsatardi. */}
            {rec?.size_bytes > 0 && status !== 'expired' && status !== 'failed' && (
              <>
                <span className="arch-head__dot" aria-hidden="true" />
                <span>{formatSize(rec.size_bytes)}</span>
              </>
            )}
          </div>
        </div>
        <TranscriptDownload lessonId={id} hasChat={chat.length > 0} />
      </div>

      <div className="arch-grid">
        <section className="arch-video" aria-label="Dars videosi">
          <RecordingArea
            status={status}
            rec={rec}
            videoRef={videoRef}
            onTime={setCurrentSec}
            onRestore={doRestore}
            restoring={restore.isPending}
            restoreFailed={restoreFailed}
            telegramError={polled?.telegram_error || ''}
            stalled={stalled}
            pollError={pollError}
            urlBroken={urlBroken}
            onVideoError={handleVideoError}
            onReloadVideo={reloadVideo}
          />
        </section>

        <section className="arch-chat" aria-label="Dars chati">
          <div className="arch-chat__head">
            <MessageSquare size={15} />
            <span>Chat</span>
            <span className="arch-chat__count">{chat.length}</span>
          </div>
          <ChatTimeline
            lessonId={id}
            messages={chat}
            activeId={activeMsgId}
            canJump={canJump}
            onJump={jumpTo}
          />
        </section>
      </div>

      <MaterialsSection materials={materials} />
    </div>
  )
}

function BackLink() {
  return (
    <Link to="/app" className="back-link">
      <ArrowLeft size={15} /> Darslarga qaytish
    </Link>
  )
}

// ── Video maydoni — barcha yozuv holatlari ─────────────────────────────────
function RecordingArea({
  status,
  rec,
  videoRef,
  onTime,
  onRestore,
  restoring,
  restoreFailed,
  telegramError,
  stalled,
  pollError,
  urlBroken,
  onVideoError,
  onReloadVideo,
}) {
  if (!rec) {
    return (
      <ArchiveNotice icon={<VideoOff size={26} />} title="Bu darsda video yozilmagan">
        Yozib olish yoqilmagan yoki dars yozuvsiz o‘tgan. Chat va materiallar quyida saqlanib qolgan.
      </ArchiveNotice>
    )
  }

  // Havola o'lgan (1 soatlik presigned muddati tugagan) va yangilash ham
  // yordam bermagan. Jimgina qora pleyer o'rniga aniq sabab.
  if (urlBroken) {
    return (
      <ArchiveNotice icon={<AlertTriangle size={26} />} title="Video havolasi eskirdi" tone="bad">
        <span>Videoga havola 1 soat amal qiladi va u tugadi.</span>
        <Button variant="secondary" onClick={onReloadVideo} style={{ marginTop: 14 }}>
          <RotateCcw size={16} /> Qayta yuklash
        </Button>
      </ArchiveNotice>
    )
  }

  if (status === 'ready' && rec.url) {
    // DIQQAT: `expiresInDays` yo'q sana uchun `null` qaytaradi, `null >= 0`
    // esa JS'da `true` — shuning uchun tekshiruv aniq yozilgan.
    const days = expiresInDays(rec.expires_at)
    return (
      <>
        <video
          ref={videoRef}
          className="arch-video__player"
          src={rec.url}
          controls
          preload="metadata"
          playsInline
          data-testid="archive-video"
          onTimeUpdate={(e) => onTime(Math.floor(e.currentTarget.currentTime || 0))}
          onError={onVideoError}
        />
        {/* Muddat izohi FAQAT kelajakdagi sana uchun. Telegramdan tiklangan
            nusxada server hali ham asl `ended_at + 30 kun` ni yuborishi
            mumkin — u allaqachon o'tgan sana. «Muddati tugagan» deb yozib
            turib ishlayotgan pleyerni ko'rsatish qarama-qarshilik bo'lardi,
            shuning uchun bunday holatda izoh umuman chiqmaydi. */}
        {days !== null && days >= 0 && (
          <p className="arch-video__note">
            <Clock size={13} /> {formatExpiry(rec.expires_at)} — Telegram arxivida saqlanib qoladi.
          </p>
        )}
      </>
    )
  }

  // Poll `ready` dedi, lekin PRESIGNED havola hali yo'q: arxiv so'rovi endi
  // qayta ketdi. Bu bir necha yuz millisekundlik oraliq — usiz foydalanuvchi
  // shu payt «Yozuvda xatolik» degan YOLG'ON xabarni ko'rib qolardi.
  if (status === 'ready') {
    return (
      <ArchiveNotice icon={<Spinner size={26} />} title="Video ochilmoqda…" tone="wait">
        Yozuv tayyor — havola olinmoqda.
      </ArchiveNotice>
    )
  }

  // Kutish CHO'ZILIB KETDI (poll to'xtatildi) yoki holat so'rovining o'zi
  // yiqilyapti. Abadiy aylanuvchi spinner — eng yomon holat: foydalanuvchi
  // nima bo'layotganini ham, nima qilishni ham bilmaydi.
  if ((stalled || pollError) && (status === 'processing' || status === 'restoring')) {
    return (
      <ArchiveNotice icon={<AlertTriangle size={26} />} title="Kutilganidan uzoq davom etmoqda" tone="bad">
        <span>
          {status === 'restoring'
            ? 'Videoni Telegramdan tiklash odatda 30-60 soniya oladi, lekin javob kelmadi.'
            : 'Yozuvni tayyorlash cho‘zilib ketdi.'}{' '}
          Birozdan so‘ng qayta urinib ko‘ring.
        </span>
        <Button variant="secondary" onClick={onReloadVideo} style={{ marginTop: 14 }}>
          <RotateCcw size={16} /> Qayta tekshirish
        </Button>
      </ArchiveNotice>
    )
  }

  if (status === 'processing') {
    return (
      <ArchiveNotice icon={<Spinner size={26} />} title="Yozuv tayyorlanmoqda…" tone="wait">
        Video hali qayta ishlanmoqda. Bu bir necha daqiqa olishi mumkin — sahifa tayyor bo‘lgach
        o‘zi yangilanadi.
      </ArchiveNotice>
    )
  }

  if (status === 'restoring') {
    return (
      <ArchiveNotice icon={<Spinner size={26} />} title="Telegramdan yuklanmoqda…" tone="wait">
        <span>Video Telegram arxividan qaytarilmoqda — odatda 30-60 soniya. Sahifani yopmasangiz
        ham bo‘ladi: tayyor bo‘lgach pleyer o‘zi paydo bo‘ladi.</span>
        <RestoreProgress />
      </ArchiveNotice>
    )
  }

  // Tiklash urinishi YIQILDI — server statusni `archived` ga qaytardi.
  // Buni oddiy «Bu dars 30 kundan eski» kartochkasidan ajratish SHART:
  // aks holda tugma bosilgani ham, natija ham bilinmay qolardi.
  if (restoreFailed) {
    return (
      <ArchiveNotice icon={<AlertTriangle size={26} />} title="Videoni tiklab bo‘lmadi" tone="bad">
        <span>
          Telegram arxividan yuklab olish uzildi.{telegramError ? ` Sabab: ${telegramError}.` : ''} Video
          yo‘qolmagan — qayta urinib ko‘rishingiz mumkin.
        </span>
        <Button onClick={onRestore} loading={restoring} style={{ marginTop: 14 }}>
          <RotateCcw size={16} /> Qayta urinish
        </Button>
      </ArchiveNotice>
    )
  }

  if (status === 'archived') {
    return (
      <ArchiveNotice icon={<Film size={26} />} title="Bu dars 30 kundan eski">
        <span>
          Video serverdan o‘chirilgan, lekin Telegram arxivida butunligicha turibdi. Ko‘rish uchun
          uni qaytarib olish kerak — 30-60 soniya.
        </span>
        <Button onClick={onRestore} loading={restoring} style={{ marginTop: 14 }}>
          <RotateCcw size={16} /> Videoni tiklash
        </Button>
      </ArchiveNotice>
    )
  }

  if (status === 'expired') {
    return (
      <ArchiveNotice icon={<Trash2 size={26} />} title="Video muddati tugagan" tone="bad">
        Saqlash muddati (30 kun) tugagan va fayl serverdan o‘chirilgan. Telegram nusxasi ham yo‘q,
        shuning uchun uni tiklab bo‘lmaydi.
      </ArchiveNotice>
    )
  }

  if (status === 'recording') {
    return (
      <ArchiveNotice icon={<Spinner size={26} />} title="Dars hali yozilmoqda" tone="wait">
        Yozuv dars yakunlangach tayyorlanadi.
      </ArchiveNotice>
    )
  }

  return (
    <ArchiveNotice icon={<AlertTriangle size={26} />} title="Yozuvda xatolik" tone="bad">
      Videoni saqlab bo‘lmagan (yozib olish jarayoni uzilgan). Chat va materiallar quyida saqlanib
      qolgan.
    </ArchiveNotice>
  )
}

function ArchiveNotice({ icon, title, tone = '', children }) {
  return (
    <div className={`arch-notice ${tone ? `arch-notice--${tone}` : ''}`}>
      <div className="arch-notice__icon">{icon}</div>
      <h2 className="h2" style={{ marginBottom: 6 }}>{title}</h2>
      <div className="text-2 arch-notice__body">{children}</div>
    </div>
  )
}

// Tiklash progressi. Aniq foiz YO'Q — Telegram uni bermaydi va soxta foiz
// ("87%" deb turib yana kutish) ishonchni buzadi. O'rniga: harakatlanuvchi
// chiziq + O'TGAN vaqt, ya'ni faqat ROST ma'lumot.
function RestoreProgress() {
  const [sec, setSec] = useState(0)
  useEffect(() => {
    const t = setInterval(() => setSec((s) => s + 1), 1000)
    return () => clearInterval(t)
  }, [])
  return (
    <div className="arch-progress">
      <div className="arch-progress__bar">
        <span className="arch-progress__fill" />
      </div>
      <span className="arch-progress__time">{formatOffset(sec)}</span>
    </div>
  )
}

// ── Chat lentasi ────────────────────────────────────────────────────────────
//
// `memo` — BEKORCHI ISH EMAS: video ijro etilayotganda `currentSec` har
// soniyada yangilanadi va usiz butun lenta (uzoq darsda yuzlab xabar, har
// birida `ChatFileCard`) soniyada bir marta qaytadan chizilardi. Aslida esa
// faqat IKKI qator o'zgaradi — eskisi `is-active` ni yo'qotadi, yangisi oladi.
//
// Solishtiruvchi funksiyalarni ATAYLAB e'tiborsiz qoldiradi: ular
// `mutateAsync` (TanStack'da barqaror), `setConfirmId` (setState) va
// `lessonId` (sahifa `key={id}` bilan qayta yaratiladi) ustida yopiladi —
// ya'ni eskirgan closure xavfi yo'q, identiteti esa har renderda o'zgaradi.
const ArchMsg = memo(
  function ArchMsg({ m, isActive, canJump, confirmOpen, deleting, onJump, onAsk, onCancel, onConfirm }) {
    return (
      <li
        data-msg-id={m.id}
        className={`arch-msg ${isActive ? 'is-active' : ''}`}
        data-testid="archive-msg"
      >
        <div className="arch-msg__head">
          <button
            type="button"
            className="arch-msg__time"
            onClick={() => onJump(m.offset_sec)}
            disabled={!canJump}
            title={canJump ? 'Videoda shu daqiqaga o\u2018tish' : 'Video mavjud emas'}
            aria-label={`${formatOffset(m.offset_sec)} \u2014 videoda shu daqiqaga o\u2018tish`}
          >
            {formatOffset(m.offset_sec)}
          </button>
          <span className="arch-msg__name truncate">{m.sender_name}</span>
          {m.is_private && <span className="chat-msg__dm">shaxsiy</span>}
          <button
            type="button"
            className="arch-msg__del"
            onClick={() => onAsk(m.id)}
            aria-label={`Xabarni o'chirish \u2014 ${m.sender_name}`}
            title="Xabarni o'chirish"
          >
            <Trash2 size={13} />
          </button>
        </div>

        {m.body && <p className="arch-msg__body">{m.body}</p>}
        {m.file && <ChatFileCard file={m.file} />}

        {confirmOpen && (
          <div className="arch-msg__confirm">
            <span>Xabar butunlay o\u2018chadi.</span>
            <button
              type="button"
              className="arch-msg__confirm-yes"
              onClick={() => onConfirm(m.id)}
              disabled={deleting}
            >
              {deleting ? <Loader2 size={12} style={{ animation: 'spin 0.7s linear infinite' }} /> : null}
              O\u2018chirish
            </button>
            <button type="button" className="arch-msg__confirm-no" onClick={onCancel}>
              Bekor
            </button>
          </div>
        )}
      </li>
    )
  },
  (a, b) =>
    a.m === b.m &&
    a.isActive === b.isActive &&
    a.canJump === b.canJump &&
    a.confirmOpen === b.confirmOpen &&
    a.deleting === b.deleting,
)

function ChatTimeline({ lessonId, messages, activeId, canJump, onJump }) {
  const del = useDeleteChatMessage()
  const [confirmId, setConfirmId] = useState(null)
  const listRef = useRef(null)

  // Chat videoni KUZATADI: ijro davomida joriy xabar ko'rinish maydonidan
  // chiqib ketsa lenta o'ziga tortadi.
  //
  // `scrollIntoView` ATAYLAB ishlatilmagan — u barcha ota-elementlarni,
  // ya'ni sahifaning o'zini ham suradi. Tor ekranda (chat video ostida)
  // bu har xabarda sahifani sakratardi. Bu yerda faqat lentaning
  // `scrollTop` iga tegamiz va faqat element KO'RINMASA.
  useEffect(() => {
    const list = listRef.current
    if (!list || !activeId) return
    const el = list.querySelector(`[data-msg-id="${activeId}"]`)
    if (!el) return
    const top = el.offsetTop - list.offsetTop
    const outOfView = top < list.scrollTop || top + el.offsetHeight > list.scrollTop + list.clientHeight
    if (outOfView) list.scrollTop = Math.max(0, top - 12)
  }, [activeId])

  const onAsk = useCallback((messageId) => setConfirmId(messageId), [])
  const onCancel = useCallback(() => setConfirmId(null), [])

  // `mutateAsync` TanStack'da barqaror havola — `useCallback` ni har renderda
  // qaytadan yasashga sabab bo'lmaydi.
  const { mutateAsync } = del
  const onConfirm = useCallback(
    async (messageId) => {
      try {
        await mutateAsync({ lessonId, messageId })
        setConfirmId(null)
        toast.info("Xabar o'chirildi")
      } catch (e) {
        toast.error(errorText(e, 'Xabarni o\u2018chirib bo\u2018lmadi'))
      }
    },
    [mutateAsync, lessonId],
  )

  if (!messages.length) {
    return (
      <div className="arch-chat__empty">
        <MessageSquare size={22} color="var(--text-3)" />
        <p className="text-2" style={{ fontSize: 13.5, margin: 0 }}>Bu darsda chat yozilmagan</p>
      </div>
    )
  }

  return (
    <ol className="arch-chat__list" ref={listRef}>
      {messages.map((m) => (
        <ArchMsg
          key={m.id}
          m={m}
          isActive={m.id === activeId}
          canJump={canJump}
          confirmOpen={confirmId === m.id}
          deleting={confirmId === m.id && del.isPending}
          onJump={onJump}
          onAsk={onAsk}
          onCancel={onCancel}
          onConfirm={onConfirm}
        />
      ))}
    </ol>
  )
}

// ── Materiallar ─────────────────────────────────────────────────────────────
function MaterialsSection({ materials }) {
  return (
    <section className="arch-materials" aria-label="Dars materiallari">
      <div className="arch-materials__head">
        <FolderOpen size={16} color="var(--accent-light)" />
        <h2 className="h2">Materiallar</h2>
        {materials.length > 0 && <span className="arch-chat__count">{materials.length}</span>}
      </div>
      {!materials.length ? (
        <p className="text-2" style={{ fontSize: 13.5, margin: 0 }}>
          Bu darsda fayl ulashilmagan.
        </p>
      ) : (
        <ul className="arch-materials__list">
          {materials.map((f) => (
            <li key={`${f.name}-${f.created_at}`}>
              {/* Havolani imzolash serverda yiqilsa `url` BO'SH keladi.
                  `<a href="">` bosilganda joriy sahifani qayta yuklaydi —
                  jim va chalkash. Bunda kartochka bosilmaydigan bo'ladi va
                  sababi yozib qo'yiladi. */}
              {f.url ? (
                <a
                  className="arch-material"
                  href={f.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  title={`${f.name} — ochish`}
                >
                  <span className="arch-material__name truncate">{f.name}</span>
                  <span className="arch-material__meta">{formatBytes(f.size)}</span>
                  <Download size={15} />
                </a>
              ) : (
                <span className="arch-material arch-material--dead" title="Havolani olib bo‘lmadi">
                  <span className="arch-material__name truncate">{f.name}</span>
                  <span className="arch-material__meta">havola olinmadi</span>
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

// ── Chat transkripti (TXT / HTML) ───────────────────────────────────────────
//
// Endpoint himoyalangan — oddiy `<a href>` ishlamaydi (brauzer havolaga JWT
// qo'sha olmaydi). Fayl `api.blob` bilan olinadi va `saveBlob` saqlaydi.
function TranscriptDownload({ lessonId, hasChat }) {
  const [busy, setBusy] = useState('')

  async function download(format) {
    setBusy(format)
    try {
      const { blob, filename } = await getChatTranscript(lessonId, format)
      // `saveBlob` brauzer `createObjectURL` ni qo'llamasa `false` qaytaradi —
      // tekshirilmasa tugma jimgina hech nima qilmagandek ko'rinardi.
      if (!saveBlob(blob, filename || `dars-chat.${format}`)) {
        toast.error('Faylni saqlab bo\u2018lmadi \u2014 brauzeringiz buni qo\u2018llamaydi')
      }
    } catch (e) {
      toast.error(errorText(e, 'Chatni yuklab bo‘lmadi'))
    } finally {
      setBusy('')
    }
  }

  return (
    <div className="arch-dl">
      <span className="arch-dl__label">
        <FileDown size={14} /> Chatni yuklab olish
      </span>
      <div className="arch-dl__opts">
        {['txt', 'html'].map((f) => (
          <button
            key={f}
            type="button"
            className="arch-dl__btn"
            onClick={() => download(f)}
            disabled={!hasChat || !!busy}
            title={hasChat ? `Chat tarixini ${f.toUpperCase()} sifatida yuklab olish` : 'Chat bo‘sh'}
            aria-label={`Chat tarixini ${f.toUpperCase()} sifatida yuklab olish`}
          >
            {busy === f ? (
              <Loader2 size={13} style={{ animation: 'spin 0.7s linear infinite' }} />
            ) : null}
            {f.toUpperCase()}
          </button>
        ))}
      </div>
    </div>
  )
}
