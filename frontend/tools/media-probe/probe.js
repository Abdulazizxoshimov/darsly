// Brauzer ichidagi o'lchov zondi. Playwright (`run.mjs`) shu API'ni chaqiradi.
//
// Bir sahifa — bitta rol: `startPublisher` yoki `startSubscriber`.
// Nashr qiluvchi canvas'dan ekran-ulashish trekini yasaydi (haqiqiy getDisplayMedia
// o'rniga: deterministik va CI'da takrorlanadigan), obunachi esa dekodlangan
// kadrni ASL kadr bilan solishtiradi.

import { LocalVideoTrack, Room, RoomEvent, Track, VideoQuality } from 'livekit-client'
import { ROI_TOP, compare, readMarker, renderScene } from './scene.js'
// ISHLAB CHIQARISHDAGI sozlamalarning O'ZI — qo'lda ko'chirilgan nusxa emas.
// "Oldin/keyin" jadvali aynan yetkazib beriladigan qiymatlarni o'lchashi shart,
// aks holda hisobot kod bilan jimgina ajralib ketadi.
import { PUBLISH_DEFAULTS, SCREEN_PUBLISH } from '../../src/livekit/mediaTuning.js'

const state = { room: null, track: null, timer: null, sub: null }
let logLines = []
const log = (...a) => {
  logLines.push(a.join(' '))
}

async function connect(url, token) {
  const room = new Room({ adaptiveStream: false, dynacast: false })
  await room.connect(url, token, { autoSubscribe: true })
  state.room = room
  return room
}

/**
 * Nashr qiluvchi: canvas → ekran-ulashish treki.
 * cfg: {url, token, width, height, fps, mode, codec, scalabilityMode, simulcast,
 *       maxBitrate, layers:[{w,h,bitrate,fps}], contentHint, degradationPreference}
 */
async function startPublisher(cfg) {
  const room = await connect(cfg.url, cfg.token)

  const canvas = document.createElement('canvas')
  canvas.width = cfg.width
  canvas.height = cfg.height
  document.body.appendChild(canvas)
  const ctx = canvas.getContext('2d', { alpha: false })

  let frameIndex = 0
  const draw = () => {
    renderScene(ctx, cfg.width, cfg.height, frameIndex & 0xfff, cfg.mode || 'scroll')
    frameIndex++
  }
  draw()
  // captureStream(fps) canvas O'ZGARGANDA kadr oladi — shuning uchun chizish
  // sur'ati aynan fps bo'lishi kerak (rAF 60 Hz bilan emas).
  const stream = canvas.captureStream(cfg.fps)
  state.timer = setInterval(draw, 1000 / cfg.fps)

  const mst = stream.getVideoTracks()[0]
  // contentHint — enkoderga "bu matn" signali: keskinlik ravon harakatdan muhimroq.
  mst.contentHint = cfg.contentHint ?? 'text'

  const track = new LocalVideoTrack(mst, undefined, true)
  state.track = track

  // `production: true` — `src/livekit/mediaTuning.js` dagi HAQIQIY qiymatlar bilan.
  const opts = cfg.production
    ? {
        ...PUBLISH_DEFAULTS,
        ...SCREEN_PUBLISH,
        source: Track.Source.ScreenShare,
        videoCodec: 'vp8',
        backupCodec: false,
      }
    : {
    source: Track.Source.ScreenShare,
    videoCodec: cfg.codec || 'vp8',
    simulcast: !!cfg.simulcast,
    screenShareEncoding: {
      maxBitrate: cfg.maxBitrate,
      maxFramerate: cfg.fps,
      priority: 'high',
    },
    degradationPreference: cfg.degradationPreference || 'maintain-resolution',
    backupCodec: cfg.backupCodec ?? false,
  }
  if (cfg.scalabilityMode) opts.scalabilityMode = cfg.scalabilityMode
  if (!cfg.production && cfg.layers) {
    opts.screenShareSimulcastLayers = cfg.layers.map((l) => ({
      width: l.w,
      height: l.h,
      encoding: { maxBitrate: l.bitrate, maxFramerate: l.fps, priority: 'medium' },
      resolution: { width: l.w, height: l.h, frameRate: l.fps },
    }))
  }
  const pub = await room.localParticipant.publishTrack(track, opts)
  log('published', pub.trackSid, 'codec', pub.trackInfo?.codecs?.map((c) => c.mimeType).join('/'))

  const sender = track.sender
  const params = sender?.getParameters()
  return {
    sid: pub.trackSid,
    encodings: params?.encodings?.map((e) => ({
      rid: e.rid,
      maxBitrate: e.maxBitrate,
      maxFramerate: e.maxFramerate,
      scaleResolutionDownBy: e.scaleResolutionDownBy,
      scalabilityMode: e.scalabilityMode,
    })),
    degradationPreference: params?.degradationPreference,
  }
}

/** Obunachi: ulanadi, birinchi kadrni kutadi, har kadrda sifatni o'lchaydi. */
async function startSubscriber(cfg) {
  const t0 = Date.now()
  const room = await connect(cfg.url, cfg.token)
  const connectedAt = Date.now()

  const video = document.createElement('video')
  video.autoplay = true
  video.muted = true
  video.playsInline = true
  document.body.appendChild(video)

  const w = cfg.width
  const h = cfg.height
  const dec = document.createElement('canvas')
  dec.width = w
  dec.height = h
  const dctx = dec.getContext('2d', { alpha: false, willReadFrequently: true })
  const ref = document.createElement('canvas')
  ref.width = w
  ref.height = h
  const rctx = ref.getContext('2d', { alpha: false, willReadFrequently: true })

  state.sub = {
    t0,
    lastIndex: -1,
    timer: null,
    firstFrameMs: null,
    connectMs: connectedAt - t0,
    samples: [],
    markerFails: 0,
    frames: 0,
  }

  const onFrame = () => {
    if (!video.videoWidth) return
    state.sub.frames++
    if (state.sub.firstFrameMs === null) state.sub.firstFrameMs = Date.now() - t0
    // Dekodlangan kadrni ASL o'lchamga kattalashtiramiz: tomoshabin uni aynan
    // shunday (to'liq ekranda) ko'radi, demak sifat ham shu masshtabda o'lchanadi.
    dctx.drawImage(video, 0, 0, w, h)
    const dimg = dctx.getImageData(0, 0, w, h)
    const m = readMarker(dimg)
    if (!m) {
      state.sub.markerFails++
      return
    }
    if (m.frameIndex === state.sub.lastIndex) return // ayni kadr ikki marta o'lchanmasin
    state.sub.lastIndex = m.frameIndex
    const latency = ((Date.now() % 60000) - m.tsMod + 60000) % 60000
    renderScene(rctx, w, h, m.frameIndex, cfg.mode || 'scroll')
    const rimg = rctx.getImageData(0, 0, w, h)
    const q = compare(dimg, rimg, w, h)
    state.sub.samples.push({
      t: Date.now() - t0,
      frameIndex: m.frameIndex,
      latency,
      psnr: q.psnr,
      sharp: q.sharp,
      vw: video.videoWidth,
      vh: video.videoHeight,
    })
  }

  let attached = false
  const onTrack = async (track) => {
    if (track.kind !== 'video' || attached) return
    attached = true
    track.attach(video)
    try {
      await video.play()
    } catch (e) {
      log('play() rad etildi:', String(e))
    }
    // Qatlamni MAJBURAN tanlash — aks holda SFU o'zi eng yuqorisini beradi va
    // "zaif o'quvchi nimani ko'radi" degan savol o'lchanmay qoladi.
    const q = { low: VideoQuality.LOW, medium: VideoQuality.MEDIUM, high: VideoQuality.HIGH }[cfg.quality]
    if (q !== undefined) {
      for (const p of room.remoteParticipants.values()) {
        const pub = p.getTrackPublication(Track.Source.ScreenShare)
        if (pub?.setVideoQuality) pub.setVideoQuality(q)
      }
    }
    // `requestVideoFrameCallback` har DEKODLANGAN kadrda aniq bir marta chaqiriladi —
    // namuna olish uchun eng to'g'ri manba. Lekin u kompozitor bo'lmagan muhitda
    // (headless) ba'zan umuman uyg'onmaydi, shuning uchun taymer zaxira sifatida
    // qoladi; ikkalasi bir kadrni ikki marta o'lchamasligi uchun kadr raqami
    // takrorlansa namuna tashlab yuboriladi.
    if (video.requestVideoFrameCallback) {
      const loop = () => {
        onFrame()
        video.requestVideoFrameCallback(loop)
      }
      video.requestVideoFrameCallback(loop)
    }
    state.sub.timer = setInterval(onFrame, 1000 / 30)
  }

  room.on(RoomEvent.TrackSubscribed, onTrack)
  // MUHIM: `autoSubscribe` bilan trek ULANISH PAYTIDA obuna bo'lib ulguradi va
  // `TrackSubscribed` biz tinglashni boshlagunimizcha o'tib ketadi (nashr qiluvchi
  // allaqachon efirda — aynan haqiqiy stsenariy). Shu sabab mavjud treklar
  // qo'lda ham ko'rib chiqiladi.
  for (const p of room.remoteParticipants.values()) {
    for (const pub of p.trackPublications.values()) {
      if (pub.track) await onTrack(pub.track)
    }
  }
  return { connectMs: state.sub.connectMs }
}

/** RTCPeerConnection statistikasi — LiveKit ichidagi transportlardan. */
async function collectStats() {
  const room = state.room
  if (!room) return null
  const out = { outbound: [], inbound: [], pair: null, transportType: null }
  const engine = room.engine
  const pcs = []
  const pub = engine?.pcManager?.publisher?.pc ?? engine?.publisher?.pc
  const sub = engine?.pcManager?.subscriber?.pc ?? engine?.subscriber?.pc
  if (pub) pcs.push(pub)
  if (sub) pcs.push(sub)
  for (const pc of pcs) {
    const report = await pc.getStats()
    const byId = new Map()
    report.forEach((r) => byId.set(r.id, r))
    report.forEach((r) => {
      if (r.type === 'outbound-rtp' && r.kind === 'video') {
        out.outbound.push({
          rid: r.rid,
          bytesSent: r.bytesSent,
          packetsSent: r.packetsSent,
          framesEncoded: r.framesEncoded,
          framesSent: r.framesSent,
          frameWidth: r.frameWidth,
          frameHeight: r.frameHeight,
          fps: r.framesPerSecond,
          targetBitrate: r.targetBitrate,
          qualityLimitationReason: r.qualityLimitationReason,
          qualityLimitationDurations: r.qualityLimitationDurations,
          encoderImplementation: r.encoderImplementation,
          totalEncodeTime: r.totalEncodeTime,
          keyFramesEncoded: r.keyFramesEncoded,
          codec: byId.get(r.codecId)?.mimeType,
          ts: r.timestamp,
        })
      }
      if (r.type === 'inbound-rtp' && r.kind === 'video') {
        out.inbound.push({
          bytesReceived: r.bytesReceived,
          packetsReceived: r.packetsReceived,
          packetsLost: r.packetsLost,
          jitter: r.jitter,
          framesDecoded: r.framesDecoded,
          framesDropped: r.framesDropped,
          frameWidth: r.frameWidth,
          frameHeight: r.frameHeight,
          fps: r.framesPerSecond,
          jitterBufferDelay: r.jitterBufferDelay,
          jitterBufferEmittedCount: r.jitterBufferEmittedCount,
          totalDecodeTime: r.totalDecodeTime,
          decoderImplementation: r.decoderImplementation,
          codec: byId.get(r.codecId)?.mimeType,
          ts: r.timestamp,
        })
      }
      if (r.type === 'candidate-pair' && r.state === 'succeeded' && r.nominated) {
        const local = byId.get(r.localCandidateId)
        const remote = byId.get(r.remoteCandidateId)
        out.pair = {
          rtt: r.currentRoundTripTime,
          availableOutgoingBitrate: r.availableOutgoingBitrate,
          availableIncomingBitrate: r.availableIncomingBitrate,
          localType: local?.candidateType,
          localProtocol: local?.protocol,
          remoteType: remote?.candidateType,
          relayProtocol: local?.relayProtocol,
        }
        out.transportType = `${local?.candidateType}/${remote?.candidateType}`
      }
    })
  }
  return out
}

function subResult() {
  return state.sub ? { ...state.sub, log: logLines } : { log: logLines }
}

async function stop() {
  if (state.timer) clearInterval(state.timer)
  if (state.sub?.timer) clearInterval(state.sub.timer)
  try {
    await state.room?.disconnect()
  } catch {
    /* ahamiyatsiz */
  }
  state.room = null
  logLines = []
}

window.probe = { startPublisher, startSubscriber, collectStats, subResult, stop, ROI_TOP }
