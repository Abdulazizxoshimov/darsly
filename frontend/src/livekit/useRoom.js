import { useEffect, useRef, useState, useCallback } from 'react'
import { ConnectionState, DisconnectReason, Room, RoomEvent, Track } from 'livekit-client'
import { CAMERA_PUBLISH, PUBLISH_DEFAULTS } from './mediaTuning'
import { localSignature, participantSignature, qualityLabel } from './roomLogic'
import { isHostParticipant } from './messaging'

// livekit-client'ga to'g'ridan-to'g'ri ulanadigan hook.
//
// ## Nega "snapshot", nega to'g'ridan-to'g'ri SDK obyekti emas
// Avvalgi versiya har LiveKit hodisasida `setTick(n+1)` qilardi va bolalar SDK
// obyektlarini render paytida o'qirdi. Oqibati: `ActiveSpeakersChanged` (~0.5 s da bir)
// butun daraxtni — 571 qatorlik xona ekranini VA 768 qatorlik doskani — qayta
// render qilardi. Ustoz doskaga chizayotganda bu seziladigan lag berardi.
//
// Endi hodisada faqat **immutable snapshot** yasaladi va uning IMZOSI eskisi bilan
// solishtiriladi. Imzo o'zgarmasa — `setState` umuman chaqirilmaydi, ya'ni React
// hech narsa qilmaydi. O'zgarsa — faqat snapshot'ga bog'liq bolalar (memo bilan)
// yangilanadi; doska va boshqaruv paneli tegilmaydi.
//
// Past-internet: adaptiveStream + dynacast + simulcast (`mediaTuning.js` — mobil bilan bir xil).

/** Bitta ishtirokchining render uchun kerak bo'ladigan holati (immutable). */
function snapshotOne(p, isLocal) {
  const cam = p.getTrackPublication(Track.Source.Camera)
  const screen = p.getTrackPublication(Track.Source.ScreenShare)
  const mic = p.getTrackPublication(Track.Source.Microphone)
  return {
    identity: p.identity,
    name: p.name || p.identity,
    isLocal,
    speaking: p.isSpeaking,
    micMuted: !mic || mic.isMuted,
    // Trek obyekti mutable, lekin uning IDENTITETI barqaror — `attach()` uchun aynan
    // shu obyekt kerak, shuning uchun snapshot'ga havola sifatida kiritiladi.
    camTrack: cam && cam.videoTrack && !cam.isMuted ? cam.videoTrack : null,
    screenTrack: screen && screen.videoTrack && !screen.isMuted ? screen.videoTrack : null,
    canPublish: p.permissions?.canPublish ?? false,
    // Ustozmi — token metadata'sidan (backend imzolagan, soxtalab bo'lmaydi).
    // Galereya tartibi (ustoz birinchi sahifada) shu bayroqqa tayanadi.
    isHost: isHostParticipant(p),
  }
}

function snapshotAll(room) {
  const out = [snapshotOne(room.localParticipant, true)]
  for (const p of room.remoteParticipants.values()) out.push(snapshotOne(p, false))
  return out
}

/** Local media holati — `Controls` shunga bog'lanadi (ishtirokchilar ro'yxatiga emas). */
function localState(room) {
  const lp = room.localParticipant
  return {
    identity: lp.identity,
    name: lp.name || lp.identity,
    micOn: lp.isMicrophoneEnabled,
    camOn: lp.isCameraEnabled,
    screenOn: lp.isScreenShareEnabled,
    canPublish: lp.permissions?.canPublish ?? false,
  }
}

/**
 * Uzilish sababi → nima qilish kerakligi.
 *
 * Bu farq muhim: avval HAR QANDAY `disconnected` guest'ni sessiyasi bilan birga
 * bosh sahifaga uloqtirardi — vaqtinchalik tarmoq uzilishida ham. Endi "xona
 * yopildi / chiqarib yuborildi" (yakuniy) va "aloqa uzildi" (qayta ulanadi)
 * bir-biridan ajratiladi.
 */
function endedByServer(reason) {
  return (
    reason === DisconnectReason.ROOM_DELETED ||
    reason === DisconnectReason.ROOM_CLOSED ||
    reason === DisconnectReason.PARTICIPANT_REMOVED ||
    reason === DisconnectReason.DUPLICATE_IDENTITY
  )
}

export function useRoom({ wsUrl, token, publish, retryKey = 0 }) {
  const [room, setRoom] = useState(null)
  const [connState, setConnState] = useState('connecting') // connecting|connected|reconnecting|disconnected
  const [ended, setEnded] = useState(null) // null | 'room_deleted' | 'removed' | 'duplicate'
  const [quality, setQuality] = useState('unknown') // good|poor|lost|unknown
  const [participants, setParticipants] = useState([])
  const [local, setLocal] = useState(null)

  // Tejamkor rejim: kirish (remote) kamera oqimlarini butunlay uzadi.
  // Ekran ulashish va ovoz QOLADI — dars mazmuni aynan shularda.
  const [dataSaver, setDataSaver] = useState(false)
  // M9 — media nosozliklari ENDI KO'RINADI.
  //
  // Avval uchala holat ham `.catch(() => {})` bilan jimgina yutilardi va
  // foydalanuvchi uchun bu "dars buzildi" degani edi: ovoz eshitilmaydi yoki
  // kamera ko'rinmaydi, ekranda esa hech qanday izoh yo'q va nima qilish
  // kerakligi ham noma'lum. Bularning har biri TUZATILADIGAN holat, shuning
  // uchun ular UI'ga chiqariladi.
  //
  //   audioBlocked — brauzer avtomatik ijroni to'sdi (foydalanuvchi hali
  //                  sahifa bilan muloqot qilmagan). Yechim: bitta bosish.
  //   mediaError   — kamera/mikrofonga ruxsat berilmadi yoki qurilma band.
  const [audioBlocked, setAudioBlocked] = useState(false)
  const [mediaError, setMediaError] = useState(null) // null | 'camera' | 'mic' | 'both'
  const dataSaverRef = useRef(false)
  const roomRef = useRef(null)

  const sigRef = useRef('')
  const localSigRef = useRef('')

  // Snapshot'ni qayta hisoblaydi va FAQAT mazmun o'zgargan bo'lsa state'ni yangilaydi.
  const sync = useCallback(() => {
    const r = roomRef.current
    if (!r) return
    const items = snapshotAll(r)
    const sig = participantSignature(items)
    if (sig !== sigRef.current) {
      sigRef.current = sig
      setParticipants(items)
    }
    const l = localState(r)
    const lsig = localSignature(l)
    if (lsig !== localSigRef.current) {
      localSigRef.current = lsig
      setLocal(l)
    }
  }, [])

  // Tejamkor rejimni xonadagi mavjud treklarga qo'llaydi.
  const applyDataSaver = useCallback((on) => {
    const r = roomRef.current
    if (!r) return
    for (const p of r.remoteParticipants.values()) {
      const cam = p.getTrackPublication(Track.Source.Camera)
      // `setSubscribed` faqat RemoteTrackPublication'da bor.
      if (cam && typeof cam.setSubscribed === 'function') cam.setSubscribed(!on)
    }
  }, [])

  useEffect(() => {
    // Qayta urinishda holat "ulanmoqda"ga qaytadi — aks holda ekran eski
    // "disconnected" da qotib qolardi va urinish ko'rinmasdi.
    setConnState('connecting')
    setEnded(null)

    const r = new Room({
      adaptiveStream: true,
      dynacast: true,
      publishDefaults: PUBLISH_DEFAULTS,
    })
    roomRef.current = r

    const audioEls = new Map()

    const onState = (s) => {
      if (s === ConnectionState.Connected) setConnState('connected')
      else if (s === ConnectionState.Reconnecting) setConnState('reconnecting')
      else if (s === ConnectionState.Connecting) setConnState('connecting')
      else if (s === ConnectionState.Disconnected) setConnState('disconnected')
      sync()
    }

    const onTrackSubscribed = (track) => {
      if (track.kind === 'audio') {
        const el = track.attach()
        el.style.display = 'none'
        document.body.appendChild(el)
        audioEls.set(track.sid, el)
      }
      sync()
    }
    const onTrackUnsubscribed = (track) => {
      if (track.kind === 'audio') {
        track.detach().forEach((el) => el.remove())
        audioEls.delete(track.sid)
      }
      sync()
    }

    // Yangi ishtirokchi trek e'lon qilsa — tejamkor rejim unga ham qo'llanishi kerak.
    const onTrackPublished = (pub) => {
      if (
        dataSaverRef.current &&
        pub?.source === Track.Source.Camera &&
        typeof pub.setSubscribed === 'function'
      ) {
        pub.setSubscribed(false)
      }
      sync()
    }

    // Aloqa sifati: FAQAT o'zimiznikini ko'rsatamiz (boshqaning sifati bizga
    // ta'sir qilmaydi va foydalanuvchini chalg'itardi).
    const onQuality = (q, participant) => {
      if (participant?.identity === r.localParticipant?.identity) setQuality(qualityLabel(q))
    }

    // Uzilish sababi — "yakuniy" va "vaqtinchalik" ni ajratish uchun.
    const onDisconnected = (reason) => {
      if (!endedByServer(reason)) return
      if (reason === DisconnectReason.PARTICIPANT_REMOVED) setEnded('removed')
      else if (reason === DisconnectReason.DUPLICATE_IDENTITY) setEnded('duplicate')
      else setEnded('room_deleted')
    }

    r.on(RoomEvent.Disconnected, onDisconnected)
      .on(RoomEvent.ConnectionStateChanged, onState)
      .on(RoomEvent.ParticipantConnected, sync)
      .on(RoomEvent.ParticipantDisconnected, sync)
      .on(RoomEvent.TrackSubscribed, onTrackSubscribed)
      .on(RoomEvent.TrackUnsubscribed, onTrackUnsubscribed)
      .on(RoomEvent.TrackPublished, onTrackPublished)
      .on(RoomEvent.TrackUnpublished, sync)
      .on(RoomEvent.LocalTrackPublished, sync)
      .on(RoomEvent.LocalTrackUnpublished, sync)
      .on(RoomEvent.TrackMuted, sync)
      .on(RoomEvent.TrackUnmuted, sync)
      .on(RoomEvent.ActiveSpeakersChanged, sync)
      .on(RoomEvent.ParticipantMetadataChanged, sync)
      .on(RoomEvent.ParticipantPermissionsChanged, sync)
      .on(RoomEvent.ConnectionQualityChanged, onQuality)
      // Ovoz ijrosi holati keyin ham o'zgarishi mumkin (masalan ustoz
      // gapirgach yangi audio trek kelganda brauzer yana to'sadi), shuning
      // uchun boshlang'ich `startAudio()` dan tashqari bu hodisa ham kerak.
      .on(RoomEvent.AudioPlaybackStatusChanged, () => {
        setAudioBlocked(!r.canPlaybackAudio)
      })

    let cancelled = false
    ;(async () => {
      try {
        await r.connect(wsUrl, token)
        if (cancelled) return
        setRoom(r)
        setConnState('connected')
        sync()
        if (publish) {
          // Har biri ALOHIDA kuzatiladi: kamera rad etilib mikrofon
          // ishlashi odatiy holat va bunda "kamera yoqilmadi" deb aniq
          // aytish kerak, umumiy "xatolik" emas.
          let camFailed = false
          let micFailed = false
          try {
            // `CAMERA_PUBLISH` — kamerada `balanced` degradatsiya (mobil bilan
            // bir xil). Sabab `mediaTuning.js` da.
            await r.localParticipant.setCameraEnabled(true, undefined, CAMERA_PUBLISH)
          } catch {
            camFailed = true
          }
          try {
            await r.localParticipant.setMicrophoneEnabled(true)
          } catch {
            micFailed = true
          }
          if (cancelled) return
          if (camFailed && micFailed) setMediaError('both')
          else if (camFailed) setMediaError('camera')
          else if (micFailed) setMediaError('mic')
        }
        try {
          await r.startAudio()
        } catch {
          // Avtomatik ijro to'sildi — bu XATO EMAS, brauzer siyosati.
          // Foydalanuvchi bitta bosish bilan tiklaydi (`resumeAudio`).
          if (!cancelled) setAudioBlocked(true)
        }
        sync()
      } catch {
        if (!cancelled) setConnState('disconnected')
      }
    })()

    return () => {
      cancelled = true
      audioEls.forEach((el) => el.remove())
      r.removeAllListeners()
      r.disconnect()
      roomRef.current = null
    }
    // retryKey — "Qayta ulanish" bosilganda bu effekt qaytadan ishga tushadi.
  }, [wsUrl, token, publish, sync, retryKey])

  // Tejamkor rejim o'zgarganda mavjud obunalarga qo'llaymiz.
  useEffect(() => {
    dataSaverRef.current = dataSaver
    applyDataSaver(dataSaver)
  }, [dataSaver, applyDataSaver])

  // Foydalanuvchi bosgach ovozni tiklaydi (brauzer avtomatik ijro siyosati
  // aynan "foydalanuvchi harakati" ni talab qiladi).
  const resumeAudio = useCallback(async () => {
    const r = roomRef.current
    if (!r) return
    try {
      await r.startAudio()
      setAudioBlocked(false)
    } catch {
      // Kamdan-kam: qayta urinish ham to'silsa banner joyida qoladi.
    }
  }, [])

  // Media xatosi bannerini foydalanuvchi yopa olsin (ruxsat bermaslik ham
  // ongli tanlov bo'lishi mumkin — masalan faqat tinglash uchun kirgan).
  const dismissMediaError = useCallback(() => setMediaError(null), [])

  return {
    room,
    connState,
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
  }
}
