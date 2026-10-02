import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { RoomEvent } from 'livekit-client'
import {
  AlertTriangle,
  Grid3x3,
  Loader2,
  Mic,
  PictureInPicture2,
  SquareUser,
  Volume2,
  Wifi,
  WifiOff,
  Zap,
} from 'lucide-react'
import { deleteChatMessage, roomChatHistory, roomChatSend, roomChatUpload } from '../api/chat'
import { errorText } from '../api/api'
import { useRoom } from '../livekit/useRoom'
import { acceptData, chatEntryFromServer, decodeData, encodeData } from '../livekit/messaging'
import * as roomstateApi from '../api/roomstate'
import {
  applyHandEvent,
  isChatVisible,
  linkView,
  nextPipState,
  pollReplayMessage,
  rateLimiter,
  reconnectDelay,
} from '../livekit/roomLogic'
import { PIP_SUPPORTED, PresenterPanel, PresenterPip, pipWindowSize } from '../livekit/PresenterPip'
import { Stage } from '../livekit/Stage'
import { Whiteboard } from '../livekit/Whiteboard'
import { RoomRail } from '../livekit/RoomRail'
import { Controls } from '../livekit/Controls'
import { ReactionsOverlay } from '../livekit/ReactionsOverlay'
import { ChatPanel } from '../panels/ChatPanel'
import { ParticipantsPanel } from '../panels/ParticipantsPanel'
import { PollsPanel } from '../panels/PollsPanel'
import {
  useEndLesson,
  useLesson,
  useOpenRoom,
  usePolls,
  useRecordings,
  useStartRecording,
  useStopRecording,
  useWaiting,
} from '../store/data'
import { readGuestRoom, roomSession } from '../lib/roomSession'
import {
  createRoomTokenSource,
  guestMinter,
  hostMinter,
  isFatalRoomTokenError,
  RENEW_RETRY_MS,
  renewDelay,
  roomErrorText,
} from '../lib/roomToken'
import { Button } from '../components/Button'
import { isLiveRoomSupported } from '../lib/features'
import { Modal } from '../components/Modal'
import { toast } from '../lib/toast'

// Token oladi (host: API; guest: sessiya), keyin RoomStage'ni yuklaydi.
export function LiveRoom({ mode }) {
  const params = useParams()
  const navigate = useNavigate()
  const isHost = mode === 'host'
  const lessonId = isHost ? params.id : undefined
  const slug = params.slug // guest join-slug — qayta ulanish yo'li

  // Mehmon: sessiyadagi xona yozuvi SHAKLI tekshirib o'qiladi (`readGuestRoom`)
  // — eskirgan/buzuq sessionStorage avval `room.lesson.title` da yiqilardi.
  const [guest] = useState(() => (isHost ? null : readGuestRoom()))
  const [token, setToken] = useState(isHost ? null : guest?.token || null)
  const [loading, setLoading] = useState(isHost)
  const [error, setError] = useState(null)

  // Host: token darsni serverda `live` qiladi — bu mutatsiya (`useOpenRoom`)
  // darslar keshini bekor qiladi, Dashboard ustoz qaytganda yangilangan bo'ladi.
  const { mutateAsync: openRoom } = useOpenRoom()
  const { mutateAsync: endLesson } = useEndLesson()
  // Dars obyekti — ovoz siyosati bayroqlari (`mute_on_entry`,
  // `allow_self_unmute`) shu yerdan boshlanadi. Host'da to'liq `Lesson` (API),
  // guest'da esa joinlink javobidagi `LessonPublic` — u ham AYNI ikki maydonni
  // beradi. Keyin siyosatni server yangilab turadi (roomstate + `policy`).
  const { data: hostLesson } = useLesson(lessonId)
  const lesson = isHost ? hostLesson || null : guest?.lesson || null
  const title = lesson?.title || 'Jonli dars'

  useEffect(() => {
    if (!isHost || !lessonId) return
    let alive = true
    openRoom(lessonId)
      .then((rt) => alive && setToken(rt))
      .catch((e) => alive && setError(errorText(e, 'Xonaga ulanib bo‘lmadi')))
      .finally(() => alive && setLoading(false))
    return () => {
      alive = false
    }
  }, [isHost, lessonId, openRoom])

  // Tokenning yagona manbai: muddatidan oldin va 401 da yangilanadi
  // (`lib/roomToken`). Mehmonda yangi token sessiyaga ham yoziladi (F5 uchun).
  const tokenSource = useMemo(() => {
    if (!token) return null
    return createRoomTokenSource({
      initial: token,
      mint: isHost ? hostMinter(lessonId) : guestMinter(guest?.join),
      persist: isHost ? undefined : roomSession.setRoomToken,
    })
  }, [token, isHost, lessonId, guest])

  // Haqiqiy yakunlash. Tasdiq `RoomStage` da so'raladi (u yerda ishtirokchilar
  // soni va yozuv holati bor — tasdiq matnini shular aniq qiladi), bu yerda
  // esa faqat bajariladi.
  const leave = useCallback(async () => {
    if (isHost && lessonId) {
      try {
        await endLesson(lessonId)
        toast.info('Dars yakunlandi')
      } catch (e) {
        toast.error(errorText(e))
      }
      navigate('/app', { replace: true })
      return
    }
    // O'quvchi uchun chiqish zararsiz — dars davom etadi, u qayta kira oladi.
    roomSession.clear()
    navigate(slug ? `/r/${slug}` : '/', { replace: true })
  }, [isHost, lessonId, navigate, slug, endLesson])

  // `beforeunload` ogohlantirishi `RoomStage` da: u faqat xonaga HAQIQATAN
  // ulanganda o'rnatilishi kerak, yuklanish yoki xato ekranida emas.

  if (loading)
    return (
      <div className="center-shell">
        <div className="page-loader">
          <Loader2 size={28} style={{ animation: 'spin 0.7s linear infinite', color: 'var(--accent)' }} />
          <p>Xonaga ulanmoqda…</p>
        </div>
      </div>
    )

  // Brauzer qo'llab-quvvatlanmasa — ulanishga URINMAYMIZ. Aks holda
  // `room.connect()` tushunarsiz xato bilan yiqilardi va foydalanuvchi
  // muammo brauzerda ekanini bilmasdan qayta-qayta urinaverardi.
  if (!isLiveRoomSupported({ needsPublish: isHost }))
    return (
      <div className="center-shell col center">
        <div className="empty__icon" style={{ background: 'var(--danger-soft)', color: 'var(--danger)' }}>
          <AlertTriangle size={28} />
        </div>
        <h2 className="h1" style={{ marginBottom: 6 }}>Brauzer qo‘llab-quvvatlamaydi</h2>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 20, textAlign: 'center', maxWidth: 360 }}>
          Jonli darsga kirish uchun Chrome, Safari yoki Firefox’ning yangi versiyasidan
          foydalaning. Havolani ilova ichidagi brauzerda (Telegram, Instagram) ochgan
          bo‘lsangiz, uni tashqi brauzerda oching.
        </p>
      </div>
    )

  if (error || !tokenSource)
    return (
      <div className="center-shell col center">
        <p style={{ color: 'var(--danger)', fontWeight: 600, marginBottom: 16 }}>
          {error || (isHost ? 'Token topilmadi' : 'Sessiya topilmadi — darsga qayta qo‘shiling')}
        </p>
        <Button variant="ghost" onClick={() => navigate(isHost ? '/app' : slug ? `/r/${slug}` : '/')}>
          {isHost ? 'Orqaga' : 'Darsga qayta qo‘shilish'}
        </Button>
      </div>
    )

  return (
    <RoomStage
      isHost={isHost}
      lessonId={lessonId}
      slug={slug}
      title={title}
      lesson={lesson}
      guest={guest}
      tokenSource={tokenSource}
      onLeave={leave}
    />
  )
}

let seq = 0

function RoomStage({ isHost, lessonId, slug, title, lesson, guest, tokenSource, onLeave }) {
  const navigate = useNavigate()
  const [retryKey, setRetryKey] = useState(0)
  // Joriy token — manbaga obuna. Yangilanganda server so'rovlari darhol yangi
  // tokenni oladi; LiveKit esa faqat `identity` o'zgarsa qayta ulanadi.
  const [roomToken, setRoomToken] = useState(() => tokenSource.current())
  useEffect(() => tokenSource.subscribe(setRoomToken), [tokenSource])
  const getToken = useCallback(() => tokenSource.current().token, [tokenSource])
  // Token yangilash YAKUNAN mumkin emas (dars tugagan, chiqarilgan, qulflangan…).
  const [fatal, setFatal] = useState(null)

  const {
    room,
    connState,
    connectError,
    ended,
    quality,
    participants,
    local,
    dataSaver,
    setDataSaver,
    audioBlocked,
    resumeAudio,
    mediaError,
    dismissMediaError,
  } = useRoom({
    wsUrl: roomToken.ws_url,
    getToken,
    identity: roomToken.identity,
    publish: isHost,
    retryKey,
  })

  // Yangilash yiqildi. Kutish xonasi darsida qayta `join` YANGI kutish so'rovi
  // yaratadi (backend cheklovi) — mehmon kutish sahifasiga qaytariladi va
  // ustoz tasdig'idan keyin xonaga qaytadi. Qolgan sabablar yakuniy ekran.
  const onRenewFailed = useCallback(
    (e) => {
      if (!isHost && e.kind === 'waiting_room') {
        roomSession.setPending({
          requestId: e.requestId,
          lesson: e.lesson || lesson,
          guestName: guest?.guestName,
          join: guest?.join,
        })
        toast.info(roomErrorText(e))
        navigate(`/r/${slug}/waiting`, { replace: true })
        return
      }
      setFatal(e)
    },
    [isHost, lesson, guest, navigate, slug],
  )

  // Xona so'rovi: `fn(token)`, 401 da tokenni yangilab BIR MARTA takrorlaydi.
  // Yangilash YAKUNAN yiqilsa (dars tugagan, chiqarilgan, kutish xonasi…)
  // shu yerdan markaziy ishlov beriladi — chaqiruvchi faqat o'tkinchi xatoni
  // (tarmoq, 429) toast qiladi (`notifyRoomError`), yakuniysini ekran ko'rsatadi.
  const run = useCallback(
    (fn) =>
      tokenSource.run(fn).catch((e) => {
        if (isFatalRoomTokenError(e)) onRenewFailed(e)
        throw e
      }),
    [tokenSource, onRenewFailed],
  )
  const notifyRoomError = useCallback((e, fallback) => {
    if (!isFatalRoomTokenError(e)) toast.error(roomErrorText(e, fallback))
  }, [])

  // Proaktiv yangilash — muddat tugashidan oldin. Tarmoq xatosida qayta
  // uriniladi, yakuniy xatoda ekran ko'rsatiladi. Kutish xonasi YOQILGAN darsda
  // jimgina yangilab bo'lmaydi (yuqoridagi sabab) — faqat talab bo'yicha (401).
  useEffect(() => {
    if (!isHost && lesson?.is_waiting_room_enabled) return
    const delay = renewDelay(roomToken.token)
    if (delay === null) return
    let alive = true
    let timer = null
    const attempt = () => {
      tokenSource.renew().catch((e) => {
        if (!alive) return
        if (isFatalRoomTokenError(e)) onRenewFailed(e)
        else timer = setTimeout(attempt, RENEW_RETRY_MS)
      })
    }
    timer = setTimeout(attempt, delay)
    return () => {
      alive = false
      clearTimeout(timer)
    }
  }, [tokenSource, roomToken.token, isHost, lesson?.is_waiting_room_enabled, onRenewFailed])

  // LiveKit tokenni rad etdi (`auth`) — bu qayta urinish bilan tuzalmaydi:
  // avval token yangilanadi, so'ng ulanish. Yangi token bilan ham rad etilsa
  // (masalan xona yopilgan) — yakuniy, cheksiz sikl yo'q.
  const renewedForConnectRef = useRef(false)
  useEffect(() => {
    if (connState === 'connected') renewedForConnectRef.current = false
  }, [connState])
  useEffect(() => {
    if (connectError !== 'auth') return
    if (renewedForConnectRef.current) {
      onRenewFailed(Object.assign(new Error('token rejected after renewal'), { kind: 'forbidden' }))
      return
    }
    let alive = true
    tokenSource
      .renew()
      .then(() => {
        if (!alive) return
        renewedForConnectRef.current = true
        setRetryKey((k) => k + 1)
      })
      .catch((e) => {
        if (!alive) return
        if (isFatalRoomTokenError(e)) onRenewFailed(e)
        // Tarmoq: quyidagi `Reconnect` ekrani backoff bilan o'zi qayta uradi.
      })
    return () => {
      alive = false
    }
  }, [connectError, tokenSource, onRenewFailed])

  // Tarmoq uzilishida avtomatik urinishlar CHEGARALANGAN (`reconnectDelay`):
  // ulanish tiklangach sanoq nolga qaytadi.
  const [reconnectAttempt, setReconnectAttempt] = useState(0)
  useEffect(() => {
    // LiveKit ulanish holati (tashqi) → urinish sanog'ini tiklash (ataylab).
    // eslint-disable-next-line react-hooks/set-state-in-effect -- ulanish holati sinxroni
    if (connState === 'connected') setReconnectAttempt(0)
  }, [connState])
  const retryConnect = useCallback(() => {
    setReconnectAttempt((n) => n + 1)
    setRetryKey((k) => k + 1)
  }, [])

  // Darsni yakunlash QAYTARIB BO'LMAYDI: xona o'chadi, hamma uziladi, yozuv
  // to'xtaydi. Bitta tasodifiy bosish 30 kishilik darsni tugatmasin — shuning
  // uchun ustozda tasdiq so'raladi. (Mobil ilovada tasdiq bor edi, webda yo'q
  // edi; endi ikkalasi izchil.) O'quvchining "Chiqish"i zararsiz — tasdiqsiz.
  const [confirmEnd, setConfirmEnd] = useState(false)
  const [ending, setEnding] = useState(false)

  const requestLeave = useCallback(() => {
    if (isHost) setConfirmEnd(true)
    else onLeave()
  }, [isHost, onLeave])

  const doEndLesson = useCallback(async () => {
    setEnding(true)
    try {
      await onLeave()
    } finally {
      setEnding(false)
      setConfirmEnd(false)
    }
  }, [onLeave])

  // Sahifani yopish/yangilash — ustoz tasodifan darsni tark etmasin
  // (F5 yoki tab'ni yopish tasdiq modalidan o'tmaydi). Brauzer o'z standart
  // matnini ko'rsatadi: maxsus matn zamonaviy brauzerlarda ruxsat etilmaydi.
  useEffect(() => {
    if (!isHost) return
    const onBeforeUnload = (e) => {
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [isHost])

  const [view, setView] = useState('gallery')
  const [panel, setPanel] = useState('none')
  const [chat, setChat] = useState([])
  const [reactions, setReactions] = useState([])
  // Qo'l ko'targanlar — Map (identity → {name, at}). Map insert tartibini saqlaydi,
  // ya'ni NAVBAT tartibi bepul keladi: kim birinchi so'ragan bo'lsa birinchi turadi.
  const [raisedHands, setRaisedHands] = useState(() => new Map())
  // Yozuv holati — serverdan o'qiladi (pastdagi effekt), taxmin qilinmaydi.
  const [recording, setRecording] = useState(false)
  const [recId, setRecId] = useState(null)
  const [guestPoll, setGuestPoll] = useState(null)
  // E'lon qilingan so'rovnoma natijasi — serverdan data-channel orqali
  // (`poll_published`). Natijaning O'ZI yuboriladi: 300 kishilik xonada har
  // biri alohida so'rov yuborsa bu 300 ta ortiqcha so'rov bo'lardi.
  const [publishedResults, setPublishedResults] = useState(null)
  // Ovoz berilgan so'rovnoma — holat PANEL EMAS, XONA darajasida. Panel
  // yopilib qayta ochilganda komponent unmount bo'ladi va lokal holat
  // yo'qolardi: o'quvchi variantlarni yana ko'rib, qayta ovoz berishga
  // urinardi (server 409 bilan rad etadi — ya'ni tushunarsiz xato).
  const [votedPollId, setVotedPollId] = useState(null)
  const [unreadChat, setUnreadChat] = useState(0)
  // Chat tarixi yuklanmadi (tarmoq) — panel buni aytadi va qayta urinish beradi.
  // `historyKey` — "Qayta urinish" bosilganda effektni qaytadan ishga tushiradi.
  const [historyError, setHistoryError] = useState(false)
  const [historyKey, setHistoryKey] = useState(0)
  // O'quvchiga so'zga ruxsat berildi — banner (toast emas: toast 4 soniyada
  // yo'qoladi va o'quvchi mikrofon tugmasini o'zi qidirib qolardi).
  const [speakGranted, setSpeakGranted] = useState(false)
  // Dars ovoz siyosati (Zoom modeli): allow_self_unmute=false bo'lsa o'quvchi
  // mikrofonni O'ZI ocholmaydi. Haqiqat manbai — SERVER (webhook enforce qiladi),
  // klient uni uch joydan oladi: kirish qiymati darsdan (host: `Lesson`,
  // guest: joinlink `LessonPublic`), keyin `GET /rooms/:id/state` bilan
  // tiklanadi va `policy` data-xabari bilan jonli yangilanadi.
  const [allowSelfUnmute, setAllowSelfUnmute] = useState(lesson?.allow_self_unmute !== false)
  // Suzuvchi oyna holati — mantiq `roomLogic.nextPipState` da (testlar ostida).
  const [pipState, setPipState] = useState('idle')
  // Suzuvchi oyna ICHIDAGI ikki mustaqil ochilish: chat paneli va qo'l ro'yxati.
  // Ular bu yerda (pastda emas) — chunki aynan shu ikkisi OYNA O'LCHAMINI
  // belgilaydi, o'lchamni esa `PresenterPip` ga prop sifatida uzatamiz.
  const [pipChatOpen, setPipChatOpen] = useState(false)
  const [pipHandsOpen, setPipHandsOpen] = useState(false)
  // Brauzer suzuvchi oynani qo'llab-quvvatlamasa ko'rsatiladigan tushuntirish
  // (Firefox/Safari) — ustoz "nega ochilmadi" deb qolmasin.
  const [pipHintOff, setPipHintOff] = useState(false)
  const chatIds = useRef(new Set())
  // `addChat` (data-channel hodisasi) suzuvchi oyna holatini RENDER'siz o'qishi
  // kerak — `panelRef` bilan bir xil sabab (aks holda obuna qayta o'rnatilardi).
  const pipOpenRef = useRef(false)
  const pipChatRef = useRef(false)
  // Joriy panel — `addChat` (data-channel hodisasi) uni render'siz o'qishi
  // kerak, aks holda har panel almashuvida qayta yaratilardi va u bilan birga
  // butun hodisa obunasi qayta o'rnatilardi.
  const panelRef = useRef('none')
  // Joriy so'rovnoma ID'si — data-channel hodisasi render'siz o'qiydi
  // (takroriy `open` — masalan qayta ulanishdan keyingi replay — bezovta qilmasin).
  const guestPollIdRef = useRef(null)

  // Tezlik cheklovchilari (klient tomon) — spamni birinchi bo'lib shu to'xtatadi.
  const allowReaction = useRef(rateLimiter(1500)).current
  const allowChat = useRef(rateLimiter(400)).current

  // Oq doska (whiteboard) holati
  const strokesRef = useRef([]) // manba: barcha stroke'lar {id,c,w,e,pts}
  const strokeMap = useRef(new Map()) // id -> stroke (O(1) qidiruv)
  const bgRef = useRef({ img: null, page: 0, total: 0, url: null }) // joriy PDF fon
  const bgRecvRef = useRef(new Map()) // id -> {chunks[], page, total} — kelayotgan fon bo'laklari
  const viewRef = useRef({ scrollY: 0 }) // en-fit doska viewport — faqat vertikal scrollY (render uchun)
  const followRef = useRef(null) // guest: host'ning oxirgi scrollY'i {scrollY}
  const lastViewRef = useRef(null) // host: oxirgi tarqatilgan scrollY {scrollY} (snapshot uchun)
  const [wbOn, setWbOn] = useState(false)
  const [wbSeq, setWbSeq] = useState(0) // inkremental render trigger
  const [wbClear, setWbClear] = useState(0) // to'liq qayta chizish trigger
  const [wbViewSeq, setWbViewSeq] = useState(0) // remote/snapshot view keldi → qayta chizish

  const { data: waitingData } = useWaiting(lessonId, isHost && !!lessonId)
  const waiting = waitingData || [] // backend bo'sh ro'yxatni null qaytaradi → null.length crash bo'lmasin
  // Host: faol so'rovnoma — kech kirganga takrorlash uchun (`pollReplayMessage`).
  // Panel bilan bitta kesh, qo'shimcha so'rov yo'q.
  const { data: hostPolls } = usePolls(isHost ? lessonId : undefined)
  // Yozuv holati — serverdan. Yozuv odatda AVTOMATIK boshlanadi (`track_published`
  // webhook'i), shuning uchun uni klientda taxmin qilib bo'lmaydi: ustoz REC
  // indikatorini ko'rmasa "yozilmayapti" deb o'ylaydi, bu esa yolg'on bo'lardi.
  // Yozuv media paydo bo'lgach boshlanadi — bir necha soniya kechikadi, shuning
  // uchun birinchi daqiqada tez-tez qayta so'raladi.
  const { data: hostRecordings } = useRecordings(isHost ? lessonId : undefined)
  const startRec = useStartRecording()
  const stopRec = useStopRecording()

  const localId = local?.identity || null
  const myHand = !!localId && raisedHands.has(localId)

  // Xona holati endpointlari uchun dars ID'si. Guest'da `lessonId` propи yo'q
  // (joinlink uni bermaydi), lekin room-token'da `lesson_id` bor — u xona nomidan
  // baribir kelib chiqadi, ya'ni yangi ma'lumot oshkor qilinmaydi.
  const roomLessonId = roomToken.lesson_id || lessonId || null

  // Kelgan/yuborilgan xabarni panel tushunadigan yozuvga aylantiradi.
  //
  // `toIdentity` XOM holda saqlanadi, tayyor matn emas: qabul qiluvchining ismi
  // render paytida ishtirokchilar ro'yxatidan olinadi. Aks holda ism xabar
  // kelgan ondagi holatda muzlab qolardi (masalan qayta ulanib ism o'zgartirgan
  // ishtirokchi eski nom bilan ko'rinardi).
  const toEntry = useCallback(
    (m) => ({
      id: m.id,
      name: m.name,
      body: m.body,
      self: m.senderIdentity === localId,
      toIdentity: m.toIdentity || null,
      // Biriktirilgan fayl {name,size,mime,url} — havola PRESIGNED va 1 soatlik,
      // shuning uchun u har javobda YANGI keladi va keshlanmaydi.
      file: m.file || null,
    }),
    [localId],
  )

  // O'qilmagan sanog'i chat KO'RINMAYOTGANDA o'sadi. "Ko'rinish" qoidasi
  // `roomLogic.isChatVisible` da (test ostida): ekran ulashilayotganda ustoz
  // brauzerda emas, shuning uchun suzuvchi oyna ochiq bo'lsa haqiqat manbai —
  // o'sha oynadagi chat, asosiy paneldagisi emas.
  const addChat = useCallback((e) => {
    if (chatIds.current.has(e.id)) return
    chatIds.current.add(e.id)
    setChat((prev) => [...prev, e])
    const visible = isChatVisible({
      panel: panelRef.current,
      pipOpen: pipOpenRef.current,
      pipChatOpen: pipChatRef.current,
    })
    if (!e.self && !visible) setUnreadChat((n) => n + 1)
  }, [])

  // Server javobidagi xabar (yuborish/fayl) → ekranga. Server o'zimizga ham
  // tarqatadi; `addChat` ID bo'yicha dublikatni kesadi.
  const addServerChat = useCallback(
    (m) => {
      const entry = chatEntryFromServer(m)
      if (entry) addChat(toEntry(entry))
    },
    [addChat, toEntry],
  )

  // Moderatsiya: xabar ro'yxatdan olib tashlanadi, LEKIN ID `chatIds` da
  // QOLADI — aks holda tarix qayta yuklanganda (yoki kech kelgan dublikatda)
  // o'chirilgan xabar ekranga qaytib chiqardi.
  const removeChat = useCallback((id) => {
    setChat((prev) => prev.filter((m) => m.id !== id))
  }, [])

  // dataURL'dan fon rasmini yuklaydi va to'liq qayta chizdiradi.
  // MUHIM: fon yuklash strokes'ni TOZALAMAYDI (snapshot fon+yozuvni birga yuboradi — kech kirgan guest
  // ikkalasini ko'rsin). Sahifa/PDF almashtirishda host ALOHIDA 'clear' yuboradi (hostSetBackground).
  const loadBg = useCallback((dataURL, page, total) => {
    const img = new Image()
    img.onload = () => {
      bgRef.current = { img, page, total, url: dataURL }
      setWbClear((n) => n + 1)
      setWbSeq((n) => n + 1)
    }
    img.onerror = () => toast.error('Fon rasmini ochib bo‘lmadi')
    img.src = dataURL
  }, [])

  // Kelgan (yoki local) doska xabarini strokesRef'ga qo'llaydi. Stabil — DataReceived deps buzilmasin.
  const applyWb = useCallback(
    (m) => {
      if (m.act === 'stroke') {
        let s = strokeMap.current.get(m.id)
        if (!s) {
          s = { id: m.id, c: m.c, w: m.w, e: m.e, pts: [] }
          strokeMap.current.set(m.id, s)
          strokesRef.current.push(s)
        }
        if (m.pts && m.pts.length) for (const p of m.pts) s.pts.push(p)
        setWbSeq((n) => n + 1)
      } else if (m.act === 'clear') {
        strokesRef.current = []
        strokeMap.current.clear()
        setWbClear((n) => n + 1)
      } else if (m.act === 'view') {
        followRef.current = { scrollY: m.scrollY || 0 }
        viewRef.current = { scrollY: m.scrollY || 0 }
        setWbViewSeq((n) => n + 1)
      } else if (m.act === 'on') {
        setWbOn(true)
      } else if (m.act === 'off') {
        setWbOn(false)
      } else if (m.act === 'snapshot') {
        if (m.i === 0) {
          strokesRef.current = []
          strokeMap.current.clear()
        }
        for (const s of m.strokes || []) {
          const cp = { id: s.id, c: s.c, w: s.w, e: s.e, pts: (s.pts || []).slice() }
          strokeMap.current.set(cp.id, cp)
          strokesRef.current.push(cp)
        }
        if (m.view) {
          followRef.current = { scrollY: m.view.scrollY || 0 }
          viewRef.current = { scrollY: m.view.scrollY || 0 }
          setWbViewSeq((n) => n + 1)
        }
        setWbOn(!!m.on)
        setWbClear((n) => n + 1)
        setWbSeq((n) => n + 1)
      } else if (m.act === 'bg_start') {
        bgRecvRef.current.set(m.id, { chunks: new Array(m.chunks), page: m.page, total: m.total })
      } else if (m.act === 'bg_chunk') {
        const buf = bgRecvRef.current.get(m.id)
        if (buf) buf.chunks[m.i] = m.data
      } else if (m.act === 'bg_end') {
        const buf = bgRecvRef.current.get(m.id)
        bgRecvRef.current.delete(m.id)
        if (buf) {
          const dataURL = buf.chunks.join('')
          loadBg(dataURL, buf.page, buf.total)
        }
      } else if (m.act === 'bg_clear') {
        bgRef.current = { img: null, page: 0, total: 0, url: null }
        setWbClear((n) => n + 1)
      }
    },
    [loadBg],
  )

  // Suzuvchi reaksiya. Yashash muddati CSS animatsiyasi bilan (3.4s) MOS
  // bo'lishi shart — qisqaroq bo'lsa emoji o'rtada g'oyib bo'lardi.
  const pushReaction = useCallback((emoji, name) => {
    const id = ++seq
    setReactions((prev) => [...prev, { id, emoji, name, left: 15 + ((id % 6) * 12) }])
    setTimeout(() => setReactions((prev) => prev.filter((r) => r.id !== id)), 3400)
  }, [])

  // Qo'l xabarini holatga qo'llaydi (local ham, remote ham shu yerdan o'tadi).
  // Mantiq `roomLogic.applyHandEvent` da — sof va test ostida.
  const applyHand = useCallback((m) => {
    setRaisedHands((prev) => applyHandEvent(prev, m))
  }, [])

  // Chat tarixi — HAMMAGA (avval faqat host yuklardi, o'quvchi bo'sh chat ko'rardi).
  // Server ommaviy xabarlar + faqat SHU ishtirokchining shaxsiy yozishmalarini beradi.
  useEffect(() => {
    if (!roomLessonId || !localId) return
    let alive = true
    run((t) => roomChatHistory(roomLessonId, t.token))
      .then((msgs) => {
        if (!alive) return
        setHistoryError(false)
        const entries = []
        for (const m of msgs || []) {
          const entry = chatEntryFromServer(m)
          if (!entry || chatIds.current.has(entry.id)) continue
          chatIds.current.add(entry.id)
          entries.push(toEntry(entry))
        }
        setChat((prev) => [...entries.reverse(), ...prev])
      })
      // Tarix kelmasa chat BO'SH ko'rinardi va bu "hali hech kim yozmagan"
      // degan YOLG'ON xabar edi. Endi farq aytiladi va qayta urinish beriladi.
      .catch(() => {
        if (alive) setHistoryError(true)
      })
    return () => {
      alive = false
    }
  }, [roomLessonId, run, localId, toEntry, historyKey])

  // Ustoz: yozuv holati ro'yxatdan (react-query keshi — start/stop mutatsiyalari
  // uni bekor qiladi, ya'ni indikator va Yozuvlar sahifasi bir manbadan).
  useEffect(() => {
    if (!isHost || !hostRecordings) return
    const active = hostRecordings.find((r) => r.status === 'recording')
    // Server holati (tashqi ma'lumot) → UI holatiga sinxronlash (ataylab).
    // eslint-disable-next-line react-hooks/set-state-in-effect -- yozuv holati sinxroni
    setRecording(!!active)
    setRecId(active ? active.id : null)
  }, [isHost, hostRecordings])

  // Data-channel qabul
  useEffect(() => {
    if (!room) return
    // `participant` — LiveKit tasdiqlagan yuboruvchi (payload ichidagi maydonlar
    // EMAS, ular soxtalashtiriladi). `acceptData` shu asosda filtrlaydi; izohi
    // `livekit/messaging.js` da. Shakl tekshiruvi `decodeData` da — buzuq
    // payload bu yerga yetib kelmaydi.
    const handler = (payload, participant) => {
      const msg = acceptData(decodeData(payload), participant)
      if (!msg) return
      if (msg.kind === 'chat') addChat(toEntry(msg))
      else if (msg.kind === 'chat_deleted') {
        // Xabar MAZMUNI yuborilmaydi va qabrtosh («o'chirilgan xabar»)
        // qoldirilmaydi: bo'sh joyni belgilash buzg'unchiga aynan u xohlagan
        // e'tiborni berardi. Xabar shunchaki yo'qoladi.
        removeChat(msg.id)
      } else if (msg.kind === 'reaction') {
        // O'z reaksiyamiz serverdan qaytib keladi — uni ikki marta ko'rsatmaymiz
        // (optimistik tarzda allaqachon chiqargan edik).
        if (msg.identity && msg.identity === localId) return
        pushReaction(msg.emoji, msg.name)
      }
      else if (msg.kind === 'hand') applyHand(msg)
      else if (msg.kind === 'poll' && msg.action === 'open') {
        // Takroriy `open` (qayta ulanishdan keyingi replay) — xabar/panel yo'q.
        const isNew = guestPollIdRef.current !== msg.poll.id
        setGuestPoll(msg.poll)
        if (!isHost && isNew) {
          setPanel('polls')
          toast.info("Yangi so'rovnoma")
        }
      } else if (msg.kind === 'poll' && msg.action === 'close') {
        // Yopilgan so'rovnoma EKRANDAN OLIB TASHLANMAYDI, faqat "yopiq" deb
        // belgilanadi. Ustoz odatda avval ovozni yopib, muhokamadan keyin
        // natijani e'lon qiladi — shu oraliqda savolni yo'qotsak o'quvchi
        // "Hozircha faol so'rovnoma yo'q" degan bo'sh ekranga qarab qolardi.
        setGuestPoll((p) => (p ? { ...p, is_active: false } : p))
      } else if (msg.kind === 'poll_published') {
        // Ustoz natijani e'lon qildi. Yopish ≠ e'lon qilish: natija AYNAN shu
        // hodisadan keyin ko'rinadi (`public` rejimda).
        setPublishedResults(msg.results)
        // Kech kirgan o'quvchi so'rovnomaning O'ZINI ko'rmagan bo'lishi mumkin
        // (host `open` xabarini u ulanishdan oldin yuborgan). Natija payload'ida
        // to'liq `poll` bor — savol va variantlarni shundan tiklaymiz, aks holda
        // panel bo'sh qolardi.
        if (msg.results.poll) setGuestPoll((cur) => cur || msg.results.poll)
        if (!isHost) {
          setPanel('polls')
          toast.info("So'rovnoma natijasi e'lon qilindi")
        }
      } else if (msg.kind === 'wb') {
        applyWb(msg)
      } else if (msg.kind === 'policy') {
        // SERVER ovoz siyosatini e'lon qildi (`mute-all` dan keyin). Host ham
        // qabul qiladi: dars sozlamasi boshqa qurilmadan (mobil ustoz ilovasi)
        // o'zgarsa web'dagi checkbox holati bilan farq qilib qolmasin.
        // Bu faqat UI ko'rsatkichi — server baribir har publish'da enforce qiladi.
        setAllowSelfUnmute(msg.allow_self_unmute !== false)
      }
    }
    room.on(RoomEvent.DataReceived, handler)
    return () => {
      room.off(RoomEvent.DataReceived, handler)
    }
  }, [room, addChat, removeChat, pushReaction, applyHand, isHost, applyWb, toEntry, localId])

  useEffect(() => {
    guestPollIdRef.current = guestPoll?.id || null
  }, [guestPoll])

  // O'quvchi: siyosat o'zgarganda xabar beramiz — tugma "jimgina" o'chib
  // qolsa foydalanuvchi buzilgan deb o'ylardi. Boshlang'ich qiymat KIRISH
  // holatidan olinadi: aks holda taqiq bilan kirgan o'quvchi xonaga
  // tushishi bilanoq "ustoz hozir o'chirdi" degan yolg'on xabarni ko'rardi.
  const prevAllowRef = useRef(allowSelfUnmute)
  useEffect(() => {
    if (isHost) return
    if (prevAllowRef.current && !allowSelfUnmute) {
      toast.info('Ustoz mikrofonlarni o‘chirdi — mikrofonni ustoz ochmaguncha yoqib bo‘lmaydi')
    } else if (!prevAllowRef.current && allowSelfUnmute) {
      toast.info('Endi mikrofonni o‘zingiz yoqishingiz mumkin')
    }
    prevAllowRef.current = allowSelfUnmute
  }, [allowSelfUnmute, isHost])

  // Panelni yopish — BARQAROR havola. Inline `() => setPanel('none')` har
  // renderda yangi funksiya bo'lgani uchun panellardagi `memo` umuman
  // ishlamasdi: doskadagi har chiziq chat panelini ham qayta chizardi.
  useEffect(() => {
    panelRef.current = panel
  }, [panel])

  const closePanel = useCallback(() => setPanel('none'), [])
  // Chat ochilganda o'qilmagan sanog'i nolga tushadi (avval bu effektda edi —
  // React uni "effektda setState" deb to'g'ri belgilagan: bu hodisa, sinxronizatsiya emas).
  const changePanel = useCallback((p) => {
    setPanel(p)
    if (p === 'chat') setUnreadChat(0)
  }, [])
  const retryHistory = useCallback(() => {
    setHistoryError(false)
    setHistoryKey((k) => k + 1)
  }, [])

  // Server xonani yopdi / ishtirokchi chiqarildi — bu YAKUNIY holat.
  // Vaqtinchalik tarmoq uzilishi bu yerga TUSHMAYDI (`useRoom.endedByServer`).
  useEffect(() => {
    if (!ended) return
    if (ended === 'removed') toast.info('Sizni darsdan chiqarishdi')
    else if (ended === 'duplicate') toast.info('Bu dars boshqa qurilmada ochildi')
    if (!isHost) {
      roomSession.clear()
      navigate(slug ? `/r/${slug}` : '/', { replace: true })
    } else {
      navigate('/app', { replace: true })
    }
  }, [ended, isHost, navigate, slug])

  // O'quvchi: host gapirishga ruxsat berganda banner ko'rsatamiz; olib tashlanganda xabar beramiz.
  const prevCanPublish = useRef(false)
  useEffect(() => {
    if (isHost || !local) return
    const now = !!local.canPublish
    if (now && !prevCanPublish.current) setSpeakGranted(true)
    if (!now && prevCanPublish.current) {
      setSpeakGranted(false)
      toast.info('Gapirish ruxsati olib tashlandi')
    }
    prevCanPublish.current = now
  }, [local, isHost])

  // Ekran ulashish boshlansa suzuvchi oyna o'zi ochiladi, tugasa yopiladi.
  // Faqat HOST uchun: o'quvchida ulashish ham, boshqariladigan signal ham yo'q.
  const sharing = !!local?.screenOn && isHost
  useEffect(() => {
    if (!PIP_SUPPORTED) return
    // Ekran-ulashish (tashqi holat) → suzuvchi oyna holatiga sinxronlash (ataylab).
    // eslint-disable-next-line react-hooks/set-state-in-effect -- ekran-ulashish holati sinxroni
    setPipState((st) => nextPipState(st, sharing ? 'share_start' : 'share_stop'))
    // Ulashish tugadi — oyna KOMPAKT holatga qaytadi: keyingi ulashish
    // ustozning ish ekranini darrov chorak ekranlik panel bilan yopmasin.
    if (!sharing) {
      pipChatRef.current = false
      setPipChatOpen(false)
      setPipHandsOpen(false)
    }
  }, [sharing])

  // Suzuvchi oyna ochiqligini ref'ga ko'chiramiz (`addChat` uni render'siz o'qiydi).
  useEffect(() => {
    pipOpenRef.current = pipState === 'open'
  }, [pipState])

  const publish = useCallback(
    (msg, identities) => {
      if (!room) return
      const opts = { reliable: true }
      if (identities) opts.destinationIdentities = identities
      room.localParticipant.publishData(encodeData(msg), opts)
    },
    [room],
  )

  // Ovoz siyosatini klient TARQATMAYDI: buni endi server qiladi (`mute-all`
  // dan keyin xonaga `policy` yuboradi), kech ulangan esa `getRoomState` dan
  // oladi. Avval xabarni faqat WEB ustoz klienti yuborardi — mobil ustoz
  // umuman yubormasdi va o'quvchi tugmasi yolg'on ko'rsatardi.

  // --- Oq doska: host hodisalari ---
  const hostDraw = useCallback(
    (segment) => {
      const msg = { kind: 'wb', act: 'stroke', ...segment }
      applyWb(msg)
      publish(msg)
    },
    [applyWb, publish],
  )
  const hostClear = useCallback(() => {
    const msg = { kind: 'wb', act: 'clear' }
    applyWb(msg)
    publish(msg)
  }, [applyWb, publish])
  const toggleBoard = useCallback(() => {
    const msg = { kind: 'wb', act: wbOn ? 'off' : 'on' }
    applyWb(msg)
    publish(msg)
  }, [wbOn, applyWb, publish])
  // Host vertikal scroll qildi — en doim fit bo'lgani uchun faqat scrollY tarqatiladi.
  const hostViewChange = useCallback(
    (v) => {
      lastViewRef.current = { scrollY: v.scrollY || 0 }
      publish({ kind: 'wb', act: 'view', scrollY: v.scrollY || 0 })
    },
    [publish],
  )

  // Fon rasmini (dataURL) ~12KB base64 bo'laklarga bo'lib data-channel orqali uzatadi.
  const sendBg = useCallback(
    (dataURL, page, total, identities) => {
      const BG_CHUNK = 12000
      const id = crypto.randomUUID()
      const chunks = Math.ceil(dataURL.length / BG_CHUNK)
      publish({ kind: 'wb', act: 'bg_start', id, page, total, chunks }, identities)
      for (let i = 0; i < chunks; i++) {
        publish({ kind: 'wb', act: 'bg_chunk', id, i, data: dataURL.slice(i * BG_CHUNK, (i + 1) * BG_CHUNK) }, identities)
      }
      publish({ kind: 'wb', act: 'bg_end', id }, identities)
    },
    [publish],
  )

  const hostSetBackground = useCallback(
    (dataURL, page, total) => {
      hostClear() // yangi sahifa/PDF — eski yozuvni tozala (local + broadcast) YANGI fondan OLDIN
      loadBg(dataURL, page, total) // local (host o'zi ko'rsin)
      sendBg(dataURL, page, total)
    },
    [hostClear, loadBg, sendBg],
  )

  const hostClearBackground = useCallback(() => {
    const msg = { kind: 'wb', act: 'bg_clear' }
    applyWb(msg)
    publish(msg)
  }, [applyWb, publish])

  // Kech qo'shilgan ishtirokchiga to'liq doska holatini chunk'lab yuboradi (faqat o'shanga).
  const sendSnapshot = useCallback(
    (identity) => {
      const bg = bgRef.current
      if (bg.url) sendBg(bg.url, bg.page, bg.total, [identity])
      const lv = lastViewRef.current
      if (lv) publish({ kind: 'wb', act: 'view', scrollY: lv.scrollY || 0 }, [identity])
      const all = strokesRef.current
      const CH = 25
      if (all.length === 0) {
        publish({ kind: 'wb', act: 'snapshot', i: 0, on: wbOn, strokes: [] }, [identity])
        return
      }
      const total = Math.ceil(all.length / CH)
      for (let i = 0; i < total; i++) {
        publish({ kind: 'wb', act: 'snapshot', i, on: wbOn, strokes: all.slice(i * CH, i * CH + CH) }, [identity])
      }
    },
    [publish, sendBg, wbOn],
  )

  // Host: yangi ishtirokchi ulanganda doska snapshot'i (doska yoniq bo'lsa) va
  // FAOL SO'ROVNOMA (bo'lsa). So'rovnoma `open` xabari kirishdan oldin ketgan
  // bo'lsa o'quvchi bo'sh panel ko'rar va ovoz bera olmasdi.
  //
  // Qo'llar bu yerda YUBORILMAYDI: ular endi server holati va har bir klient
  // xonaga kirganda `getRoomState` bilan o'zi oladi (pastdagi effekt). Host orqali
  // uzatish ustoz kech ulansa yoki brauzerini yangilasa ishlamay qolardi.
  useEffect(() => {
    if (!room || !isHost) return
    const onJoin = (p) => {
      if (wbOn) sendSnapshot(p.identity)
      const replay = pollReplayMessage(hostPolls)
      if (replay) publish(replay, [p.identity])
    }
    room.on(RoomEvent.ParticipantConnected, onJoin)
    return () => {
      room.off(RoomEvent.ParticipantConnected, onJoin)
    }
  }, [room, isHost, wbOn, sendSnapshot, hostPolls, publish])

  // Xonaga kirganda (va qayta ulanganda) ko'tarilgan qo'llar holatini serverdan
  // tiklaymiz — kech kirgan ham to'liq navbatni ko'radi.
  useEffect(() => {
    if (!room || !roomLessonId) return
    let alive = true
    const load = () =>
      run((t) => roomstateApi.getRoomState(roomLessonId, t.token))
        .then((st) => {
          if (!alive || !st) return
          const next = new Map()
          for (const h of st.hands || []) {
            next.set(h.identity, { name: h.name, at: Date.parse(h.raised_at) || 0 })
          }
          setRaisedHands(next)
          // Ovoz siyosati ham shu yerdan: `policy` data-xabarini o'tkazib
          // yuborgan (kech ulangan yoki qayta ulangan) klient uni FAQAT shundan
          // tiklaydi. Dars o'qilmasa server ruxsat beruvchi qiymat qaytaradi,
          // ya'ni mavjud bo'lmagan taqiqni ko'rsatib qo'ymaymiz.
          setAllowSelfUnmute(st.allow_self_unmute !== false)
          // O'quvchi yozuv holatini FAQAT shu yerdan biladi: `listRecordings`
          // mentor huquqini talab qiladi. Ustozda esa o'z manbasi bor
          // (yuqoridagi effekt), shuning uchun uni bu yerda ustiga yozmaymiz.
          if (!isHost) setRecording(!!st.recording)
        })
        .catch(() => {
          /* holat bo'lmasa xona baribir ishlaydi — bloklamaymiz */
        })
    load()
    // Yozuv dars o'rtasida boshlanishi/to'xtashi mumkin — o'quvchi buni
    // ko'rishi kerak (maxfiylik). Kamdan-kam so'raladi: bu indikator, oqim emas.
    const t = isHost ? null : setInterval(load, 30_000)
    return () => {
      alive = false
      if (t) clearInterval(t)
    }
  }, [room, roomLessonId, run, isHost])

  // Chat SERVER orqali: saqlanadi (tarix + yozuv), moderatsiya qilinadi va
  // shaxsiy xabar faqat ikki tomonga yetkaziladi. Host ham, o'quvchi ham AYNI
  // yo'ldan yuradi — avval ikki xil kod yo'li bor edi va o'quvchi xabari hech
  // qayerda saqlanmasdi.
  const sendChatNow = useCallback(
    async (body, to) => {
      try {
        const m = await run((t) => roomChatSend(roomLessonId, t.token, body, to))
        addServerChat(m)
      } catch (e) {
        notifyRoomError(e, 'Xabar yuborilmadi')
      }
    },
    [roomLessonId, run, addServerChat, notifyRoomError],
  )

  // `false` qaytishi = xabar YUBORILMADI (klient tomondagi tezlik cheklovi).
  // `ChatPanel` shunda matnni maydonda QOLDIRADI — avval u jimgina o'chib
  // ketardi va tez yozgan foydalanuvchi xabarini qaytadan terishga majbur
  // bo'lardi (eng bezovta qiluvchi turdagi yo'qotish: sababsiz).
  const handleSendChat = useCallback(
    (body, to) => {
      if (!roomLessonId || !allowChat()) return false
      void sendChatNow(body, to)
      return true
    },
    [allowChat, roomLessonId, sendChatNow],
  )

  // Fayl ham AYNI xona yo'lidan ketadi (host ham, o'quvchi ham) — bitta kod
  // yo'li, bitta xatti-harakat va bitta tezlik cheklovi. Xato YUQORIGA
  // uzatiladi: `ChatPanel` uni panel ichida ko'rsatadi (toast emas — 20 MB
  // yuklab, so'ng 4 soniyalik toast'ni o'tkazib yuborish oson).
  // `signal` — panelning «Bekor qilish» tugmasi; u XHR'gacha yetib borishi shart.
  const handleSendFile = useCallback(
    async (file, to, onProgress, signal) => {
      if (!roomLessonId) throw new Error('lesson yo‘q')
      const m = await run((t) => roomChatUpload(roomLessonId, t.token, file, { to, onProgress, signal }))
      addServerChat(m)
    },
    [roomLessonId, run, addServerChat],
  )

  // Xabarni o'chirish — FAQAT ustoz (JWT yo'li; `lessonId` faqat host'da bor).
  // Server hammaga `chat_deleted` tarqatadi; o'zimizda esa darhol olib
  // tashlaymiz — data-channel echo'siga qarab kutib turish "tugma ishlamadi"
  // taassurotini berardi.
  const handleDeleteChat = useCallback(
    async (id) => {
      if (!isHost || !lessonId) return
      try {
        await deleteChatMessage(lessonId, id)
        removeChat(id)
      } catch (e) {
        toast.error(errorText(e, 'Xabarni o‘chirib bo‘lmadi'))
      }
    },
    [isHost, lessonId, removeChat],
  )

  // Reaksiya SERVER orqali tarqatiladi: shunda tezlik cheklovi haqiqiy bo'ladi
  // (klient tomondagisini o'zgartirilgan klient chetlab o'tardi). O'zimizga
  // darhol ko'rsatamiz — server echo'si `identity` bo'yicha o'tkazib yuboriladi.
  const handleReaction = useCallback(
    (emoji) => {
      if (!allowReaction() || !roomLessonId) return
      pushReaction(emoji, local?.name || 'Men')
      run((t) => roomstateApi.sendReaction(roomLessonId, t.token, emoji)).catch(() => {
        /* 429 yoki tarmoq — emoji o'tkinchi, foydalanuvchini bezovta qilmaymiz */
      })
    },
    [allowReaction, roomLessonId, run, local, pushReaction],
  )

  // Qo'l — SERVER holati. Optimistik qo'llaymiz (tugma darhol javob bersin),
  // so'rov yiqilsa ORQAGA QAYTARAMIZ: aks holda o'quvchi qo'li ko'tarilgan deb
  // o'ylab kutib o'tirardi, ustoz esa uni ko'rmasdi — eng yomon nomuvofiqlik.
  const toggleHand = useCallback(async () => {
    if (!localId || !roomLessonId) return
    const raised = !myHand
    const name = local?.name || 'Men'
    applyHand({ identity: localId, name, raised, at: Date.now() })
    try {
      await run((t) => roomstateApi.setHand(roomLessonId, t.token, raised))
    } catch (e) {
      applyHand({ identity: localId, name, raised: !raised, at: Date.now() })
      notifyRoomError(e, raised ? "Qo'l ko'tarilmadi" : "Qo'l tushirilmadi")
    }
  }, [localId, roomLessonId, run, myHand, local, applyHand, notifyRoomError])

  const lowerHand = useCallback(
    async (identity) => {
      if (!lessonId) return
      try {
        await roomstateApi.lowerHand(lessonId, identity)
      } catch (e) {
        toast.error(errorText(e, "Qo'lni tushirib bo'lmadi"))
      }
    },
    [lessonId],
  )

  const lowerAllHands = useCallback(async () => {
    if (!lessonId) return
    try {
      await roomstateApi.lowerAllHands(lessonId)
    } catch (e) {
      toast.error(errorText(e, "Qo'llarni tushirib bo'lmadi"))
    }
  }, [lessonId])

  const toggleRecord = useCallback(async () => {
    if (!lessonId) return
    try {
      if (recording && recId) {
        await stopRec.mutateAsync({ recordingId: recId, lessonId })
        setRecording(false)
        setRecId(null)
        toast.info('Yozib olish to‘xtatildi')
      } else {
        const rec = await startRec.mutateAsync(lessonId)
        setRecording(true)
        setRecId(rec.id)
        toast.success('Yozib olish boshlandi')
      }
    } catch (e) {
      toast.error(errorText(e, 'Yozib olishda xatolik'))
    }
  }, [lessonId, recording, recId, startRec, stopRec])

  const broadcastPoll = useCallback((action, poll) => publish({ kind: 'poll', action, poll }), [publish])

  // Xabar `setState` yangilovchisining ICHIDA emas: StrictMode yangilovchini
  // ikki marta chaqiradi va toast ikki marta chiqardi.
  const toggleDataSaver = useCallback(() => {
    const next = !dataSaver
    setDataSaver(next)
    toast.info(next ? 'Tejamkor rejim: kameralar o‘chirildi, ovoz va ekran qoladi' : 'Tejamkor rejim o‘chirildi')
  }, [dataSaver, setDataSaver])

  const enableMic = useCallback(async () => {
    try {
      await room?.localParticipant.setMicrophoneEnabled(true)
      setSpeakGranted(false)
    } catch {
      toast.error('Mikrofonni yoqib bo‘lmadi')
    }
  }, [room])

  // Navbat ro'yxati — PiP paneli uchun (tartib serverdan keladi, Map saqlaydi).
  const handQueue = useMemo(
    () =>
      [...raisedHands.entries()]
        .filter(([id]) => id !== localId)
        .map(([identity, h]) => ({ identity, name: h.name })),
    [raisedHands, localId],
  )

  // PiP uchun oxirgi xabarlar. Panel ochilganda ekranning ~1/4 ini egallaydi,
  // ya'ni 8 ta xabar juda oz edi; to'liq tarix baribir asosiy oynada qoladi.
  const pipChat = useMemo(() => chat.slice(-50), [chat])

  // Suzuvchi oynadagi chat: ochilganda o'qilmagan sanog'i NOLGA tushadi (xabar
  // endi ko'z oldida). Qo'l ro'yxati esa yopiladi — ikkalasi birga ochilsa
  // kichik oynada ikki ro'yxat bir-birini siqib qo'yardi.
  const togglePipChat = useCallback(() => {
    const next = !pipChatRef.current
    pipChatRef.current = next
    setPipChatOpen(next)
    if (next) {
      setUnreadChat(0)
      setPipHandsOpen(false)
    }
  }, [])

  // Qo'l ro'yxati: badge SANOG'IGA tegmaydi — qo'l "o'qilmagan xabar" emas,
  // HOLAT. U faqat qo'l tushirilganda (yoki egasi tushirganda) kamayadi.
  const togglePipHands = useCallback(() => {
    setPipHandsOpen((v) => !v)
  }, [])

  const stopShare = useCallback(() => {
    room?.localParticipant.setScreenShareEnabled(false).catch(() => {})
  }, [room])

  const toggleMicQuick = useCallback(() => {
    const lp = room?.localParticipant
    if (!lp) return
    lp.setMicrophoneEnabled(!lp.isMicrophoneEnabled).catch(() => {})
  }, [room])

  const raisedHandsCount = useMemo(
    () => [...raisedHands.keys()].filter((id) => id !== localId).length,
    [raisedHands, localId],
  )

  // Suzuvchi oyna o'lchami — sof funksiya (`pipWindowSize`, test ostida).
  const pipSize = useMemo(
    () => pipWindowSize({ chatOpen: pipChatOpen, handsOpen: pipHandsOpen, handCount: handQueue.length }),
    [pipChatOpen, pipHandsOpen, handQueue.length],
  )

  // Yakuniy holat: token yangilab bo'lmaydi (dars tugagan / chiqarilgan /
  // qulflangan). Qayta urinish BEFOYDA — sabab aytiladi va chiqish beriladi.
  if (fatal)
    return (
      <div className="center-shell col center">
        <div className="empty__icon" style={{ background: 'var(--danger-soft)', color: 'var(--danger)' }}>
          <AlertTriangle size={28} />
        </div>
        <h2 className="h1" style={{ marginBottom: 6 }}>{roomErrorText(fatal)}</h2>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 20, textAlign: 'center', maxWidth: 340 }}>
          {fatal.kind === 'not_live'
            ? 'Ustoz darsni boshlashi bilan havola orqali qayta qo‘shilishingiz mumkin.'
            : 'Xonaga qayta ulanib bo‘lmaydi.'}
        </p>
        <Button onClick={onLeave}>{isHost ? 'Orqaga' : 'Chiqish'}</Button>
      </div>
    )

  // Ulanish yo'q: `auth` — token yangilanmoqda (yuqoridagi effekt), `network`
  // yoki ulangandan keyingi uzilish — backoff bilan qayta urinish.
  // Sessiya O'CHIRILMAYDI va sahifadan uloqtirilmaydi: "Qayta ulanish" bilan
  // darsga qaytiladi.
  if (connState === 'disconnected' && !ended) {
    if (connectError === 'auth')
      return (
        <div className="center-shell">
          <div className="page-loader">
            <Loader2 size={28} style={{ animation: 'spin 0.7s linear infinite', color: 'var(--accent)' }} />
            <p>Sessiya yangilanmoqda…</p>
          </div>
        </div>
      )
    // `key={retryKey}` — har urinishda sanoq qaytadan boshlansin.
    return (
      <Reconnect
        key={retryKey}
        delayMs={reconnectDelay(reconnectAttempt)}
        onRetry={retryConnect}
        onLeave={onLeave}
      />
    )
  }

  if (!room)
    return (
      <div className="center-shell">
        <div className="page-loader">
          <Loader2 size={28} style={{ animation: 'spin 0.7s linear infinite', color: 'var(--accent)' }} />
          <p>Xonaga ulanmoqda…</p>
        </div>
      </div>
    )

  const reconnecting = connState === 'reconnecting' || connState === 'connecting'
  const link = linkView(reconnecting, quality)

  return (
    <div className="room">
      <div className="room__header">
        <div className="row gap-3 grow">
          <span style={{ fontWeight: 700 }} className="truncate">{title}</span>
          {/* Yozuv indikatori ikkala tomonda ham ko'rinadi: ustozda o'z
              manbasidan, o'quvchida xona holatidan (`st.recording`). Avval
              faqat ustoz ko'rardi — o'quvchi o'zi yozilayotganini bilmasdi. */}
          {recording && (
            <span
              className="row gap-2"
              style={{ color: 'var(--danger)', fontSize: 12, fontWeight: 700 }}
              title="Bu dars yozib olinmoqda"
            >
              <span className="rec-dot" /> REC
            </span>
          )}
          {dataSaver && (
            <span className="row gap-2" style={{ color: 'var(--accent-light)', fontSize: 12, fontWeight: 700 }}>
              <Zap size={13} /> Tejamkor
            </span>
          )}
        </div>
        <div className="row gap-3">
          <span className={`conn ${link.bad ? 'conn--bad' : 'conn--good'}`} title={link.hint}>
            {link.bad ? <WifiOff size={16} /> : <Wifi size={16} />}
            {link.label}
          </span>
          {PIP_SUPPORTED && sharing && pipState !== 'open' && (
            <button
              className="icon-btn"
              style={{ width: 'auto', fontSize: 12, fontWeight: 600, gap: 4 }}
              onClick={() => setPipState((st) => nextPipState(st, 'user_open'))}
              title="Signallarni suzuvchi oynada ko'rsatish"
            >
              <PictureInPicture2 size={16} /> Suzuvchi oyna
            </button>
          )}
          <div className="view-toggle">
            <button className={view === 'gallery' ? 'active' : ''} onClick={() => setView('gallery')} title="Gallery">
              <Grid3x3 size={16} />
            </button>
            <button className={view === 'speaker' ? 'active' : ''} onClick={() => setView('speaker')} title="Speaker">
              <SquareUser size={16} />
            </button>
          </div>
          <span className="text-2" style={{ fontSize: 13 }}>{participants.length} ishtirokchi</span>
        </div>
      </div>

      {speakGranted && (
        <div className="speak-banner">
          <Mic size={16} />
          <span className="grow">Ustoz sizga gapirishga ruxsat berdi</span>
          <button className="btn btn--sm" onClick={enableMic}>Mikrofonni yoqish</button>
          <button className="icon-btn" onClick={() => setSpeakGranted(false)} aria-label="Yopish">×</button>
        </div>
      )}

      {/* Brauzer ovozni to'sdi — bu eng ko'p uchraydigan "hech narsa eshitilmayapti"
          sababi. Yopib bo'lmaydi: yopilsa foydalanuvchi darsni jimgina ko'rib
          o'tirardi va nima uchun eshitilmayotganini bilmasdi. */}
      {audioBlocked && (
        <div className="speak-banner speak-banner--warn">
          <Volume2 size={16} />
          <span className="grow">Brauzer ovozni to‘sdi — darsni eshitish uchun bosing</span>
          <button className="btn btn--sm" onClick={resumeAudio}>Ovozni yoqish</button>
        </div>
      )}

      {mediaError && (
        <div className="speak-banner speak-banner--warn">
          <AlertTriangle size={16} />
          <span className="grow">
            {mediaError === 'both'
              ? 'Kamera va mikrofon yoqilmadi — brauzer manzil satridagi ruxsatni tekshiring'
              : mediaError === 'camera'
                ? 'Kamera yoqilmadi — ruxsat berilmagan yoki boshqa dastur band qilgan'
                : 'Mikrofon yoqilmadi — ruxsat berilmagan yoki boshqa dastur band qilgan'}
          </span>
          <button className="icon-btn" onClick={dismissMediaError} aria-label="Yopish">×</button>
        </div>
      )}

      {/* Suzuvchi oyna Chrome/Edge xususiyati. Firefox/Safari'da u YO'Q — bu
          "xato" emas, lekin ustoz sababini bilmasa uni bizning nosozligimiz deb
          o'ylaydi. Shuning uchun ulashish boshlanganda bir marta tushuntiramiz. */}
      {sharing && !PIP_SUPPORTED && !pipHintOff && (
        <div className="speak-banner speak-banner--warn">
          <PictureInPicture2 size={16} />
          <span className="grow">
            Bu brauzer suzuvchi oynani qo‘llab-quvvatlamaydi — chat va qo‘l signallari faqat shu oynada
            ko‘rinadi. Chrome yoki Edge’da ular boshqa ilova ustida ham turadi.
          </span>
          <button className="icon-btn" onClick={() => setPipHintOff(true)} aria-label="Yopish">×</button>
        </div>
      )}

      <div className="room__body">
        <div className="stage">
          {wbOn ? (
            <Whiteboard
              strokesRef={strokesRef}
              bgRef={bgRef}
              viewRef={viewRef}
              followRef={followRef}
              seq={wbSeq}
              clearSeq={wbClear}
              viewSeq={wbViewSeq}
              canDraw={isHost}
              onHostDraw={hostDraw}
              onClear={hostClear}
              onViewChange={hostViewChange}
              onSetBackground={hostSetBackground}
              onClearBackground={hostClearBackground}
            />
          ) : (
            <Stage participants={participants} view={view} raisedHands={raisedHands} />
          )}
          <RoomRail participants={participants} isHost={isHost} raisedHands={raisedHands} localId={localId} />
          <ReactionsOverlay reactions={reactions} />
        </div>

        {panel === 'chat' && (
          <ChatPanel
            entries={chat}
            participants={participants}
            localId={localId}
            onSend={handleSendChat}
            onSendFile={handleSendFile}
            canDelete={isHost && !!lessonId}
            onDelete={handleDeleteChat}
            historyError={historyError}
            onRetryHistory={retryHistory}
            onClose={closePanel}
          />
        )}
        {panel === 'participants' && (
          <ParticipantsPanel
            participants={participants}
            isHost={isHost}
            lessonId={lessonId}
            localId={localId}
            raisedHands={raisedHands}
            allowSelfUnmute={allowSelfUnmute}
            onPolicyChanged={setAllowSelfUnmute}
            onLowerHand={lowerHand}
            onLowerAllHands={lowerAllHands}
            onClose={closePanel}
          />
        )}
        {panel === 'polls' && (
          <PollsPanel
            isHost={isHost}
            lessonId={lessonId}
            roomToken={roomToken}
            withRoomToken={run}
            guestActivePoll={guestPoll}
            publishedResults={publishedResults}
            votedPollId={votedPollId}
            onVoted={setVotedPollId}
            onBroadcastPoll={broadcastPoll}
            onClose={closePanel}
          />
        )}
      </div>

      <PresenterPip
        open={pipState === 'open'}
        width={pipSize.width}
        height={pipSize.height}
        onClose={() => setPipState((st) => nextPipState(st, 'user_close'))}
        onOpenFailed={() => {
          setPipState((st) => nextPipState(st, 'user_close'))
          // Document PiP foydalanuvchi harakatini talab qiladi va ekran tanlash
          // oynasidan keyin bu "ruxsat" sarflangan bo'lishi mumkin. Jim qolish
          // ustozni "signal oynasi qani?" degan savol bilan qoldirardi.
          toast.info('Suzuvchi oyna avtomatik ochilmadi — yuqoridagi «Suzuvchi oyna» tugmasini bosing')
        }}
      >
        <PresenterPanel
          hands={handQueue}
          reactions={reactions}
          chat={pipChat}
          unread={unreadChat}
          chatOpen={pipChatOpen}
          handsOpen={pipHandsOpen}
          onToggleChat={togglePipChat}
          onToggleHands={togglePipHands}
          micOn={!!local?.micOn}
          onToggleMic={toggleMicQuick}
          onStopShare={stopShare}
          onLowerHand={lowerHand}
          onSendChat={handleSendChat}
        />
      </PresenterPip>

      {/* Ekran ulashish belgisi — Zoom'dagi kabi chiziqli ramka. Ustoz brauzerga
          qaytganda ulashish DAVOM ETAYOTGANINI darhol ko'radi (eng ko'p uchraydigan
          xato: dars tugagach ulashishni to'xtatishni unutish). */}
      {sharing && (
        <div className="share-frame" aria-hidden="true">
          <span className="share-frame__tag">EKRAN ULASHILMOQDA</span>
        </div>
      )}

      <Controls
        room={room}
        local={local}
        isHost={isHost}
        recording={recording}
        onToggleRecord={toggleRecord}
        panel={panel}
        setPanel={changePanel}
        handRaised={myHand}
        onToggleHand={toggleHand}
        onReaction={handleReaction}
        onLeave={requestLeave}
        unreadChat={unreadChat}
        waitingCount={waiting.length}
        raisedHandsCount={raisedHandsCount}
        whiteboardOn={wbOn}
        onToggleBoard={toggleBoard}
        dataSaver={dataSaver}
        onToggleDataSaver={toggleDataSaver}
        selfUnmuteBlocked={!isHost && !allowSelfUnmute}
      />

      <Modal open={confirmEnd} onClose={() => !ending && setConfirmEnd(false)} title="Darsni yakunlaysizmi?" width={420}>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
          Xona yopiladi va barcha ishtirokchilar ({participants.length} kishi) darsdan chiqariladi.
          {recording && ' Yozib olish ham to‘xtatiladi.'} Bu amalni bekor qilib bo‘lmaydi.
        </p>
        <div className="row gap-3" style={{ justifyContent: 'flex-end' }}>
          <Button variant="ghost" onClick={() => setConfirmEnd(false)} disabled={ending}>
            Bekor qilish
          </Button>
          <Button variant="danger" onClick={doEndLesson} disabled={ending}>
            {ending ? 'Yakunlanmoqda…' : 'Ha, yakunlash'}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

// Uzilishdan keyingi ekran. MUHIM: guest sessiyasi TOZALANMAYDI — avvalgi
// versiya har uzilishda uni o'chirib, foydalanuvchini bosh sahifaga uloqtirardi
// va u darsga qaytish uchun havolani qaytadan qidirishga majbur bo'lardi.
//
// `delayMs` — avtomatik urinishgacha kutish (`roomLogic.reconnectDelay`);
// `null` — avtomatik urinishlar tugagan, faqat qo'lda.
function Reconnect({ delayMs, onRetry, onLeave }) {
  const [secs, setSecs] = useState(delayMs === null ? null : Math.round(delayMs / 1000))
  // Bir marta ishlash kafolati: `onRetry` har renderda yangi funksiya bo'lgani uchun
  // guard'siz effekt secs=0 da qayta-qayta ishga tushib CHEKSIZ SIKL yasardi
  // (retry → parent render → yangi onRetry → effekt → retry …).
  const fired = useRef(false)
  useEffect(() => {
    if (secs === null) return
    if (secs <= 0) {
      if (!fired.current) {
        fired.current = true
        onRetry()
      }
      return
    }
    const t = setTimeout(() => setSecs((s) => s - 1), 1000)
    return () => clearTimeout(t)
  }, [secs, onRetry])

  return (
    <div className="center-shell col center">
      <div className="empty__icon" style={{ background: 'var(--danger-soft)', color: 'var(--danger)' }}>
        <WifiOff size={28} />
      </div>
      <h2 className="h1" style={{ marginBottom: 6 }}>Aloqa uzildi</h2>
      <p className="text-2" style={{ fontSize: 14, marginBottom: 20, textAlign: 'center', maxWidth: 340 }}>
        {secs === null
          ? 'Avtomatik qayta ulanib bo‘lmadi — internetni tekshirib, qo‘lda urinib ko‘ring.'
          : `Dars davom etmoqda. ${secs > 0 ? `${secs} soniyadan so‘ng avtomatik qayta ulanamiz.` : 'Qayta ulanmoqda…'}`}
      </p>
      <div className="row gap-3">
        <Button onClick={onRetry}>Hoziroq qayta ulanish</Button>
        <Button variant="ghost" onClick={onLeave}>Chiqish</Button>
      </div>
    </div>
  )
}
