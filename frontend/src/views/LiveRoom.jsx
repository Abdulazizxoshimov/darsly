import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { RoomEvent } from 'livekit-client'
import { Grid3x3, Loader2, SquareUser, Wifi, WifiOff } from 'lucide-react'
import { getHostToken, getLesson, endLesson } from '../api/lessons'
import { chatHistory, sendChat } from '../api/chat'
import { startRecording, stopRecording } from '../api/recordings'
import { errorText } from '../api/api'
import { useRoom, roomParticipants } from '../livekit/useRoom'
import { decodeData, encodeData } from '../livekit/messaging'
import { Stage } from '../livekit/Stage'
import { Whiteboard } from '../livekit/Whiteboard'
import { RoomRail } from '../livekit/RoomRail'
import { Controls } from '../livekit/Controls'
import { ReactionsOverlay } from '../livekit/ReactionsOverlay'
import { ChatPanel } from '../panels/ChatPanel'
import { ParticipantsPanel } from '../panels/ParticipantsPanel'
import { PollsPanel } from '../panels/PollsPanel'
import { useWaiting } from '../store/data'
import { roomSession } from '../lib/roomSession'
import { Button } from '../components/Button'
import { toast } from '../lib/toast'

// Token oladi (host: API; guest: sessiya), keyin RoomStage'ni yuklaydi.
export function LiveRoom({ mode }) {
  const params = useParams()
  const navigate = useNavigate()
  const isHost = mode === 'host'
  const lessonId = isHost ? params.id : undefined
  const slug = params.slug // guest join-slug — qayta ulanish yo'li

  const [token, setToken] = useState(isHost ? null : roomSession.get().room?.token || null)
  const [title, setTitle] = useState(isHost ? 'Jonli dars' : roomSession.get().room?.lesson.title || 'Jonli dars')
  const [loading, setLoading] = useState(isHost)
  const [error, setError] = useState(null)

  useEffect(() => {
    if (!isHost || !lessonId) return
    Promise.all([getHostToken(lessonId), getLesson(lessonId).catch(() => null)])
      .then(([rt, lesson]) => {
        setToken(rt)
        if (lesson) setTitle(lesson.title)
      })
      .catch((e) => setError(errorText(e, 'Xonaga ulanib bo‘lmadi')))
      .finally(() => setLoading(false))
  }, [isHost, lessonId])

  async function leave() {
    if (isHost && lessonId) {
      try {
        await endLesson(lessonId)
        toast.info('Dars yakunlandi')
      } catch (e) {
        toast.error(errorText(e))
      }
      navigate('/app', { replace: true })
    } else {
      roomSession.clear()
      navigate(slug ? `/r/${slug}` : '/', { replace: true })
    }
  }

  if (loading)
    return (
      <div className="center-shell">
        <div className="page-loader">
          <Loader2 size={28} style={{ animation: 'spin 0.7s linear infinite', color: 'var(--accent)' }} />
          <p>Xonaga ulanmoqda…</p>
        </div>
      </div>
    )

  if (error || !token)
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

  return <RoomStage isHost={isHost} lessonId={lessonId} title={title} roomToken={token} onLeave={leave} />
}

let seq = 0

function RoomStage({ isHost, lessonId, title, roomToken, onLeave }) {
  const navigate = useNavigate()
  const { room, connState } = useRoom({ wsUrl: roomToken.ws_url, token: roomToken.token, publish: isHost })

  const [view, setView] = useState('gallery')
  const [panel, setPanel] = useState('none')
  const [chat, setChat] = useState([])
  const [reactions, setReactions] = useState([])
  const [raisedHands, setRaisedHands] = useState(new Set())
  const [myHand, setMyHand] = useState(false)
  const [recording, setRecording] = useState(false)
  const [recId, setRecId] = useState(null)
  const [guestPoll, setGuestPoll] = useState(null)
  const [unreadChat, setUnreadChat] = useState(0)
  const chatIds = useRef(new Set())

  // Oq doska (whiteboard) holati
  const strokesRef = useRef([]) // manba: barcha stroke'lar {id,c,w,e,pts}
  const strokeMap = useRef(new Map()) // id -> stroke (O(1) qidiruv)
  const bgRef = useRef({ img: null, page: 0, total: 0, url: null }) // joriy PDF fon
  const bgRecvRef = useRef(new Map()) // id -> {chunks[], page, total} — kelayotgan fon bo'laklari
  const viewRef = useRef({ scrollY: 0 }) // en-fit doska viewport — faqat vertikal scrollY (render uchun)
  const followRef = useRef(null) // guest: host'ning oxirgi scrollY'i {scrollY} (scale guest o'z enidan hisoblaydi)
  const lastViewRef = useRef(null) // host: oxirgi tarqatilgan scrollY {scrollY} (snapshot uchun)
  const [wbOn, setWbOn] = useState(false)
  const [wbSeq, setWbSeq] = useState(0) // inkremental render trigger
  const [wbClear, setWbClear] = useState(0) // to'liq qayta chizish trigger
  const [wbViewSeq, setWbViewSeq] = useState(0) // remote/snapshot view keldi → qayta chizish (guest ergashadi)

  const { data: waitingData } = useWaiting(lessonId, isHost && !!lessonId)
  const waiting = waitingData || [] // backend bo'sh ro'yxatni null qaytaradi → null.length crash bo'lmasin

  const addChat = useCallback((e) => {
    if (chatIds.current.has(e.id)) return
    chatIds.current.add(e.id)
    setChat((prev) => [...prev, e])
    if (!e.self) setUnreadChat((n) => n + 1)
  }, [])

  // dataURL'dan fon rasmini yuklaydi va to'liq qayta chizdiradi.
  // MUHIM: fon yuklash strokes'ni TOZALAMAYDI (snapshot fon+yozuvni birga yuboradi — kech kirgan guest
  // ikkalasini ko'rsin). Sahifa/PDF almashtirishda host ALOHIDA 'clear' yuboradi (hostSetBackground).
  // Stabil (deps []) — faqat stabil setter/ref'lardan foydalanadi.
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
  const applyWb = useCallback((m) => {
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
      // Host o'zining scrollY'ini yubordi. En doim fit — guest o'z en-fit scale'i bilan render qiladi.
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
      // Yangi fon uchun bufer ochamiz (i → data bo'laklari).
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
  }, [loadBg])

  const pushReaction = useCallback((emoji, name) => {
    const id = ++seq
    setReactions((prev) => [...prev, { id, emoji, name, left: 15 + ((id % 6) * 12) }])
    setTimeout(() => setReactions((prev) => prev.filter((r) => r.id !== id)), 2400)
  }, [])

  // Host chat tarixi
  useEffect(() => {
    if (!isHost || !lessonId || !room) return
    const localId = room.localParticipant.identity
    chatHistory(lessonId)
      .then((msgs) => {
        const entries = []
        for (const m of msgs || []) {
          if (chatIds.current.has(m.id)) continue
          chatIds.current.add(m.id)
          entries.push({ id: m.id, name: m.sender_name, body: m.body, self: m.sender_identity === localId })
        }
        setChat((prev) => [...entries.reverse(), ...prev])
      })
      .catch(() => {})
  }, [isHost, lessonId, room])

  // Data-channel qabul
  useEffect(() => {
    if (!room) return
    const handler = (payload) => {
      const msg = decodeData(payload)
      if (!msg) return
      if (msg.kind === 'chat') addChat({ id: msg.id, name: msg.name, body: msg.body, self: false })
      else if (msg.kind === 'reaction') pushReaction(msg.emoji, msg.name)
      else if (msg.kind === 'hand') {
        setRaisedHands((prev) => {
          const next = new Set(prev)
          if (msg.raised) next.add(msg.identity)
          else next.delete(msg.identity)
          return next
        })
      } else if (msg.kind === 'poll' && msg.action === 'open') {
        setGuestPoll(msg.poll)
        if (!isHost) {
          setPanel('polls')
          toast.info("Yangi so'rovnoma")
        }
      } else if (msg.kind === 'wb') {
        applyWb(msg)
      }
    }
    room.on(RoomEvent.DataReceived, handler)
    return () => {
      room.off(RoomEvent.DataReceived, handler)
    }
  }, [room, addChat, pushReaction, isHost, applyWb])

  // Chat panel ochilsa o'qilmagan nolga
  useEffect(() => {
    if (panel === 'chat') setUnreadChat(0)
  }, [panel])

  // Guest: server xonani yopsa (host yakunlasa) chiqadi
  useEffect(() => {
    if (!isHost && connState === 'disconnected' && room) {
      roomSession.clear()
      navigate('/', { replace: true })
    }
  }, [isHost, connState, room, navigate])

  // O'quvchi: host gapirishga ruxsat berganda (canPublish false→true) — mikrofon tugmasi
  // Controls'da faollashadi (useRoom bump orqali) + toast bilan xabar beriladi.
  useEffect(() => {
    if (!room || isHost) return
    const localId = room.localParticipant.identity
    const onPerm = (prev, participant) => {
      if (participant?.identity !== localId) return
      const canNow = participant.permissions?.canPublish
      if (canNow && !prev?.canPublish) toast.success('Sizga gapirishga ruxsat berildi')
    }
    room.on(RoomEvent.ParticipantPermissionsChanged, onPerm)
    return () => {
      room.off(RoomEvent.ParticipantPermissionsChanged, onPerm)
    }
  }, [room, isHost])

  function publish(msg, identities) {
    if (!room) return
    const opts = { reliable: true }
    if (identities) opts.destinationIdentities = identities
    room.localParticipant.publishData(encodeData(msg), opts)
  }

  // --- Oq doska: host hodisalari (plain funksiya — har render'da yangi room/wbOn ko'radi) ---
  function hostDraw(seg) {
    const msg = { kind: 'wb', act: 'stroke', ...seg }
    applyWb(msg)
    publish(msg)
  }
  function hostClear() {
    const msg = { kind: 'wb', act: 'clear' }
    applyWb(msg)
    publish(msg)
  }
  function toggleBoard() {
    const next = !wbOn
    const msg = { kind: 'wb', act: next ? 'on' : 'off' }
    applyWb(msg)
    publish(msg)
  }
  // Host vertikal scroll qildi (yoki doska ochildi) — viewRef Whiteboard'da yangilangan, faqat scrollY'ni tarqatamiz.
  // En doim fit bo'lgani uchun scale sinxronlanmaydi; guest o'z en-fit scale'i bilan render qiladi. (O'zimizga echo yo'q.)
  function hostViewChange(v) {
    lastViewRef.current = { scrollY: v.scrollY || 0 }
    publish({ kind: 'wb', act: 'view', scrollY: v.scrollY || 0 })
  }

  // Fon rasmini (dataURL) ~12KB base64 bo'laklarga bo'lib data-channel orqali uzatadi.
  // identities berilmasa — hammaga; berilsa — faqat o'sha ishtirokchiga (kech qo'shilgan guest).
  const BG_CHUNK = 12000
  function sendBg(dataURL, page, total, identities) {
    const id = crypto.randomUUID()
    const chunks = Math.ceil(dataURL.length / BG_CHUNK)
    publish({ kind: 'wb', act: 'bg_start', id, page, total, chunks }, identities)
    for (let i = 0; i < chunks; i++) {
      publish({ kind: 'wb', act: 'bg_chunk', id, i, data: dataURL.slice(i * BG_CHUNK, (i + 1) * BG_CHUNK) }, identities)
    }
    publish({ kind: 'wb', act: 'bg_end', id }, identities)
  }

  // Host PDF sahifasini fon qildi (sahifa almashtirish yoki yangi PDF): eski yozuvni ALOHIDA 'clear' bilan
  // tozalaymiz (local + broadcast), keyin yangi fonni qo'yamiz. loadBg endi strokes'ni tozalamaydi
  // (snapshot fon+yozuvni birga saqlaydi). Bir xil sahifada qolinsa bu chaqirilmaydi → yozuv saqlanadi.
  function hostSetBackground(dataURL, page, total) {
    hostClear() // yangi sahifa/PDF — eski yozuvni tozala (local + broadcast) YANGI fondan OLDIN
    loadBg(dataURL, page, total) // local (host o'zi ko'rsin) — chunk'larni qayta yig'ishga hojat yo'q
    sendBg(dataURL, page, total)
  }

  // Host PDF fonni olib tashladi.
  function hostClearBackground() {
    const msg = { kind: 'wb', act: 'bg_clear' }
    applyWb(msg)
    publish(msg)
  }

  // Kech qo'shilgan ishtirokchiga to'liq holatni chunk'lab yuboradi (faqat o'shanga).
  function sendSnapshot(identity) {
    if (!room) return
    // Fonni AVVAL yuboramiz — bg_end endi strokes'ni tozalamaydi. Keyin snapshot (i=0 strokes'ni almashtiradi)
    // → kech kirgan guest PDF fon + OLDINGI yozuvni ko'radi.
    const bg = bgRef.current
    if (bg.url) sendBg(bg.url, bg.page, bg.total, [identity])
    // Joriy scrollY — guest darhol ustoz turgan vertikal pozitsiyaga o'tsin (en avtomatik fit).
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
      const chunk = all.slice(i * CH, i * CH + CH)
      publish({ kind: 'wb', act: 'snapshot', i, on: wbOn, strokes: chunk }, [identity])
    }
  }

  // Host: yangi ishtirokchi ulanganda doska yoniq bo'lsa snapshot yuboradi.
  useEffect(() => {
    if (!room || !isHost) return
    const onJoin = (p) => {
      if (wbOn) sendSnapshot(p.identity)
    }
    room.on(RoomEvent.ParticipantConnected, onJoin)
    return () => {
      room.off(RoomEvent.ParticipantConnected, onJoin)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [room, isHost, wbOn])

  async function handleSendChat(body) {
    const name = room?.localParticipant.name || 'Men'
    if (isHost && lessonId) {
      try {
        const m = await sendChat(lessonId, body)
        addChat({ id: m.id, name: m.sender_name, body: m.body, self: true })
      } catch (e) {
        toast.error(errorText(e, 'Xabar yuborilmadi'))
      }
    } else {
      const id = `c${++seq}-${room?.localParticipant.identity}`
      publish({ kind: 'chat', id, name, body, ts: Date.now() })
      addChat({ id, name, body, self: true })
    }
  }

  function handleReaction(emoji) {
    const name = room?.localParticipant.name || 'Men'
    publish({ kind: 'reaction', emoji, name })
    pushReaction(emoji, name)
  }

  function toggleHand() {
    const raised = !myHand
    const localId = room?.localParticipant.identity
    const name = room?.localParticipant.name || 'Men'
    setMyHand(raised)
    publish({ kind: 'hand', raised, identity: localId, name })
    setRaisedHands((prev) => {
      const next = new Set(prev)
      if (raised) next.add(localId)
      else next.delete(localId)
      return next
    })
  }

  async function toggleRecord() {
    if (!lessonId) return
    try {
      if (recording && recId) {
        await stopRecording(recId)
        setRecording(false)
        setRecId(null)
        toast.info('Yozib olish to‘xtatildi')
      } else {
        const rec = await startRecording(lessonId)
        setRecording(true)
        setRecId(rec.id)
        toast.success('Yozib olish boshlandi')
      }
    } catch (e) {
      toast.error(errorText(e, 'Yozib olishda xatolik'))
    }
  }

  function broadcastPoll(action, poll) {
    publish({ kind: 'poll', action, poll })
  }

  if (!room) {
    // Ulanish uzilib qolsa (SFU'ga yetib bo'lmasa) cheksiz loaderда qolmaslik uchun.
    if (connState === 'disconnected')
      return (
        <div className="center-shell col center">
          <div className="empty__icon" style={{ background: 'var(--danger-soft)', color: 'var(--danger)' }}>
            <WifiOff size={28} />
          </div>
          <h2 className="h1" style={{ marginBottom: 6 }}>Xonaga ulanib bo'lmadi</h2>
          <p className="text-2" style={{ fontSize: 14, marginBottom: 20, textAlign: 'center', maxWidth: 320 }}>
            Video serverga ulanish uzildi. Internetni tekshirib, qayta urinib ko'ring.
          </p>
          <Button variant="ghost" onClick={onLeave}>Orqaga</Button>
        </div>
      )
    return (
      <div className="center-shell">
        <div className="page-loader">
          <Loader2 size={28} style={{ animation: 'spin 0.7s linear infinite', color: 'var(--accent)' }} />
          <p>Xonaga ulanmoqda…</p>
        </div>
      </div>
    )
  }

  const reconnecting = connState === 'reconnecting' || connState === 'connecting'
  const count = roomParticipants(room).length
  const localId = room.localParticipant.identity
  const raisedHandsCount = [...raisedHands].filter((id) => id !== localId).length

  return (
    <div className="room">
      <div className="room__header">
        <div className="row gap-3 grow">
          <span style={{ fontWeight: 700 }} className="truncate">{title}</span>
          {recording && (
            <span className="row gap-2" style={{ color: 'var(--danger)', fontSize: 12, fontWeight: 700 }}>
              <span className="rec-dot" /> REC
            </span>
          )}
        </div>
        <div className="row gap-3">
          <span className={`conn ${reconnecting ? 'conn--bad' : 'conn--good'}`}>
            {reconnecting ? <WifiOff size={16} /> : <Wifi size={16} />}
            {reconnecting ? 'Ulanmoqda…' : 'Yaxshi'}
          </span>
          <div className="view-toggle">
            <button className={view === 'gallery' ? 'active' : ''} onClick={() => setView('gallery')} title="Gallery">
              <Grid3x3 size={16} />
            </button>
            <button className={view === 'speaker' ? 'active' : ''} onClick={() => setView('speaker')} title="Speaker">
              <SquareUser size={16} />
            </button>
          </div>
          <span className="text-2" style={{ fontSize: 13 }}>{count} ishtirokchi</span>
        </div>
      </div>

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
            <Stage room={room} view={view} raisedHands={raisedHands} />
          )}
          <RoomRail room={room} isHost={isHost} raisedHands={raisedHands} />
          <ReactionsOverlay reactions={reactions} />
        </div>

        {panel === 'chat' && <ChatPanel entries={chat} onSend={handleSendChat} onClose={() => setPanel('none')} />}
        {panel === 'participants' && (
          <ParticipantsPanel
            room={room}
            isHost={isHost}
            lessonId={lessonId}
            raisedHands={raisedHands}
            onClose={() => setPanel('none')}
          />
        )}
        {panel === 'polls' && (
          <PollsPanel
            isHost={isHost}
            lessonId={lessonId}
            roomToken={roomToken}
            guestActivePoll={guestPoll}
            onBroadcastPoll={broadcastPoll}
            onClose={() => setPanel('none')}
          />
        )}
      </div>

      <Controls
        room={room}
        isHost={isHost}
        panel={panel}
        setPanel={setPanel}
        handRaised={myHand}
        onToggleHand={toggleHand}
        onReaction={handleReaction}
        recording={recording}
        onToggleRecord={toggleRecord}
        onLeave={onLeave}
        unreadChat={unreadChat}
        waitingCount={waiting.length}
        raisedHandsCount={raisedHandsCount}
        whiteboardOn={wbOn}
        onToggleBoard={toggleBoard}
      />
    </div>
  )
}
