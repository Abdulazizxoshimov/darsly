// Ekran-ulashish SIFATI va KECHIKISHini o'lchaydigan haydovchi (Playwright).
//
// Nima uchun: "kechikish ko'p, sifat past" — bu his-tuyg'u. Uni RAQAMGA aylantirmasdan
// hech qanday sozlamani o'zgartirib bo'lmaydi (loyiha qoidasi: LiveKit API'lari va
// natijalari TAXMIN QILINMAYDI). Bu vosita har konfiguratsiya uchun quyidagini beradi:
//   · PSNR va keskinlik nisbati (matn o'qiladimi)
//   · glass-to-glass kechikish (chizilgandan ko'ringungacha)
//   · haqiqiy bitreyt / o'lcham / fps / enkoder cheklovi sababi
//   · ICE yo'li (host/srflx/relay) — relay kechikishni oshiradi
//
// Ishga tushirish (frontend dev-server 3000 va LiveKit 7880 ishlab turishi kerak):
//   node tools/media-probe/run.mjs --out /tmp/quality.json
//   node tools/media-probe/run.mjs --only vp9 --secs 20
//
// Standart LiveKit dev kalitlari `services/livekit/livekit.local.yaml` dan olinadi.

import crypto from 'node:crypto'
import fs from 'node:fs'
import { chromium } from 'playwright'

const arg = (name, def) => {
  const i = process.argv.indexOf('--' + name)
  return i > -1 ? process.argv[i + 1] : def
}
const LK_URL = arg('lk', 'ws://127.0.0.1:7880')
const API_KEY = arg('key', 'devkey')
const API_SECRET = arg('secret', 'secret_at_least_32_characters_long_000000')
const PAGE = arg('page', 'http://localhost:3000/tools/media-probe/index.html')
const SECS = Number(arg('secs', 16))
const ONLY = arg('only', '')
const OUT = arg('out', '/tmp/media-probe.json')

/** LiveKit access token — HS256 JWT (server-sdk'siz, sof crypto bilan). */
function token(room, identity, canPublish) {
  const now = Math.floor(Date.now() / 1000)
  const header = { alg: 'HS256', typ: 'JWT' }
  const payload = {
    exp: now + 3600,
    nbf: now - 10,
    iss: API_KEY,
    sub: identity,
    name: identity,
    video: { room, roomJoin: true, canPublish, canSubscribe: true, canPublishData: true },
  }
  const b64 = (o) => Buffer.from(JSON.stringify(o)).toString('base64url')
  const data = `${b64(header)}.${b64(payload)}`
  const sig = crypto.createHmac('sha256', API_SECRET).update(data).digest('base64url')
  return `${data}.${sig}`
}

const SCREEN_720 = { width: 1280, height: 720, fps: 15 }
const SCREEN_1080 = { width: 1920, height: 1080, fps: 15 }

/**
 * Stsenariylar. Har biri BITTA savolga javob beradi — shuning uchun `q` maydoni bor.
 */
function scenarios() {
  const list = []
  // ── 1. KODEK samaradorligi: bir xil bitreyt, bir xil o'lcham, boshqa kodek.
  //    Savol: past bitreytda MATN qaysi kodekda o'qiladi?
  for (const cap of [200, 400, 800, 1500]) {
    for (const codec of ['vp8', 'vp9', 'av1', 'h264']) {
      list.push({
        name: `codec-${codec}-${cap}k`,
        q: 'kodek samaradorligi',
        pub: {
          ...SCREEN_720,
          codec,
          maxBitrate: cap * 1000,
          simulcast: false,
          scalabilityMode: codec === 'vp9' || codec === 'av1' ? 'L1T3' : undefined,
          mode: 'scroll',
        },
      })
    }
  }
  // ── 2. O'LCHAM: 1080p slayd matni 720p ga nisbatan (bir xil bitreyt byudjeti).
  for (const codec of ['vp8', 'vp9']) {
    for (const cap of [800, 1500, 2500]) {
      list.push({
        name: `res1080-${codec}-${cap}k`,
        q: '1080p kerakmi',
        pub: {
          ...SCREEN_1080,
          codec,
          maxBitrate: cap * 1000,
          simulcast: false,
          scalabilityMode: codec === 'vp9' ? 'L1T3' : undefined,
          mode: 'scroll',
        },
      })
    }
  }
  // ── 3. STATIK slayd (harakatsiz) — real dars kontentining ko'p qismi shunday.
  for (const codec of ['vp8', 'vp9']) {
    list.push({
      name: `static-${codec}-400k`,
      q: 'statik slayd',
      pub: {
        ...SCREEN_720,
        codec,
        maxBitrate: 400000,
        simulcast: false,
        scalabilityMode: codec === 'vp9' ? 'L1T3' : undefined,
        mode: 'static',
      },
      subMode: 'static',
    })
  }
  // ── 4. HOZIRGI ishlab chiqarish sozlamasi (mediaTuning.js) — bazaviy nuqta.
  list.push({
    name: 'current-simulcast-high',
    q: 'hozirgi holat',
    pub: {
      ...SCREEN_720,
      codec: 'vp8',
      maxBitrate: 1500000,
      simulcast: true,
      layers: [{ w: 640, h: 360, bitrate: 200000, fps: 3 }],
      mode: 'scroll',
    },
  })
  list.push({
    name: 'current-simulcast-low',
    q: 'hozirgi holat (past qatlam)',
    pub: {
      ...SCREEN_720,
      codec: 'vp8',
      maxBitrate: 1500000,
      simulcast: true,
      layers: [{ w: 640, h: 360, bitrate: 200000, fps: 3 }],
      mode: 'scroll',
    },
    sub: { quality: 'low' },
  })
  // ── 5. TAKLIF: VP9 SVC L3T3_KEY — bitta oqim, uch fazoviy qatlam.
  list.push({
    name: 'proposed-vp9-svc-high',
    q: 'taklif',
    pub: {
      ...SCREEN_1080,
      codec: 'vp9',
      maxBitrate: 1500000,
      simulcast: false,
      scalabilityMode: 'L3T3_KEY',
      mode: 'scroll',
    },
  })
  list.push({
    name: 'proposed-vp9-svc-low',
    q: 'taklif (past qatlam)',
    pub: {
      ...SCREEN_1080,
      codec: 'vp9',
      maxBitrate: 1500000,
      simulcast: false,
      scalabilityMode: 'L3T3_KEY',
      mode: 'scroll',
    },
    sub: { quality: 'low' },
  })
  // ── 5b. HOZIRGI sozlama, LEKIN haqiqiy manba o'lchami bilan.
  //    Web'da `SCREEN_CAPTURE` da `resolution` berilmagan → brauzer 1080p gacha
  //    oladi va `screenShareEncoding` faqat BITREYTni belgilaydi, o'lchamni emas.
  //    Ya'ni ishlab chiqarishda yuqori qatlam 1080p. Taklif bilan solishtirish
  //    aynan shu manbada bo'lishi kerak (aks holda PSNR bazasi boshqacha bo'ladi).
  for (const qty of ['high', 'low']) {
    list.push({
      name: `current1080-${qty}`,
      q: 'hozirgi (1080p manba)',
      pub: {
        ...SCREEN_1080,
        codec: 'vp8',
        maxBitrate: 1500000,
        simulcast: true,
        layers: [{ w: 640, h: 360, bitrate: 200000, fps: 3 }],
        mode: 'scroll',
      },
      sub: { quality: qty },
    })
  }
  // ── 6. TAKLIF QILINGAN qatlam tuzilishi: 1080p manba, UCH qatlam.
  //    Hozirgi sozlamada 200 kbps va 1500 kbps orasida HECH NARSA yo'q — 600 kbps
  //    li o'quvchi 1500'ni ko'tarmaydi va 200 kbps/3 fps ga tushadi. O'rta qatlam
  //    aynan shu bo'shliqni yopadi.
  const proposedLayers = [
    { w: 640, h: 360, bitrate: 250000, fps: 10 },
    { w: 1280, h: 720, bitrate: 700000, fps: 15 },
  ]
  for (const qty of ['high', 'medium', 'low']) {
    list.push({
      name: `cand-3layer-${qty}`,
      q: 'taklif: 3 qatlam',
      pub: {
        ...SCREEN_1080,
        codec: 'vp8',
        maxBitrate: 1500000,
        simulcast: true,
        layers: proposedLayers,
        mode: 'scroll',
      },
      sub: { quality: qty },
    })
  }
  // ── 7. PAST qatlamning FPS'i: 3 fps "muzlagan ekran" hissini beradimi?
  for (const low of [
    { name: 'fps3-200k', w: 640, h: 360, bitrate: 200000, fps: 3 },
    { name: 'fps10-250k', w: 640, h: 360, bitrate: 250000, fps: 10 },
    { name: 'fps15-400k', w: 640, h: 360, bitrate: 400000, fps: 15 },
  ]) {
    list.push({
      name: `lowlayer-${low.name}`,
      q: 'past qatlam fps',
      pub: {
        ...SCREEN_1080,
        codec: 'vp8',
        maxBitrate: 1500000,
        simulcast: true,
        layers: [low, { w: 1280, h: 720, bitrate: 700000, fps: 15 }],
        mode: 'scroll',
      },
      sub: { quality: 'low' },
    })
  }
  // ── 8. ASOSIY SAVOL: kichik byudjetda O'LCHAMni saqlash yaxshimi yoki FPS'ni?
  //    1080p manbadan 360p ga tushish = 3× kichraytirish: 14 px matn ~5 px bo'ladi
  //    va HECH QANDAY bitreyt uni tiklamaydi. Shuning uchun past qatlamni
  //    "kamroq kadr, ko'proq piksel" tomonga suramizmi — o'lchab ko'ramiz.
  const ladders = {
    'A-current': [{ w: 640, h: 360, bitrate: 200000, fps: 3 }],
    'B-540p': [{ w: 960, h: 540, bitrate: 300000, fps: 5 }],
    'C-720p3': [{ w: 1280, h: 720, bitrate: 300000, fps: 3 }],
    'D-720p5': [{ w: 1280, h: 720, bitrate: 400000, fps: 5 }],
  }
  for (const [key, low] of Object.entries(ladders)) {
    list.push({
      name: `lowres-${key}`,
      q: 'past qatlam: o\'lcham yoki fps',
      pub: {
        ...SCREEN_1080,
        codec: 'vp8',
        maxBitrate: 1500000,
        simulcast: true,
        layers: [...low, { w: 1600, h: 900, bitrate: 800000, fps: 15 }],
        mode: 'scroll',
      },
      sub: { quality: 'low' },
    })
  }
  // O'rta qatlam variantlari (o'rtacha 4G / uydagi Wi-Fi)
  for (const [key, mid] of Object.entries({
    'M720': { w: 1280, h: 720, bitrate: 700000, fps: 15 },
    'M900': { w: 1600, h: 900, bitrate: 800000, fps: 12 },
    'M1080': { w: 1920, h: 1080, bitrate: 800000, fps: 8 },
  })) {
    list.push({
      name: `midres-${key}`,
      q: 'o\'rta qatlam',
      pub: {
        ...SCREEN_1080,
        codec: 'vp8',
        maxBitrate: 1500000,
        simulcast: true,
        layers: [{ w: 960, h: 540, bitrate: 300000, fps: 5 }, mid],
        mode: 'scroll',
      },
      sub: { quality: 'medium' },
    })
  }
  // ── 9. YAKUNIY nomzod narvon: uchala qatlam ham TO'LIQ o'lchamda, farq faqat
  //    kadr chastotasi va bitreytda. Savol — nashr qiluvchining protsessori
  //    (uchta 1080p enkod) buni ko'taradimi.
  for (const qty of ['high', 'medium', 'low']) {
    list.push({
      name: `final-${qty}`,
      q: 'yakuniy nomzod',
      pub: {
        ...SCREEN_1080,
        codec: 'vp8',
        maxBitrate: 1500000,
        simulcast: true,
        layers: [
          { w: 1920, h: 1080, bitrate: 350000, fps: 3 },
          { w: 1920, h: 1080, bitrate: 800000, fps: 8 },
        ],
        mode: 'scroll',
      },
      sub: { quality: qty },
    })
  }
  // ── 10. "KEYIN" o'lchovi: `src/livekit/mediaTuning.js` dagi HAQIQIY qiymatlar.
  //    Bu stsenariy hech qanday raqamni takrorlamaydi — u ishlab chiqarish
  //    modulini import qiladi, ya'ni hisobot kod bilan ajralib keta olmaydi.
  for (const qty of ['high', 'medium', 'low']) {
    list.push({
      name: `shipped-${qty}`,
      q: 'yetkazilgan sozlama',
      pub: { ...SCREEN_1080, production: true, mode: 'scroll', codec: 'vp8', maxBitrate: 1500000 },
      sub: { quality: qty },
    })
  }
  // ── 11. Eng past qatlamning fps'i va KECHIKISH. 3 fps da jitter-bufer kadrni
  //    uzoq ushlaydi (kadrlar orasi 333 ms) — bu "sekin" hissini beradi.
  for (const [name, fps, bitrate] of [
    ['3fps-300k', 3, 300000],
    ['5fps-350k', 5, 350000],
    ['8fps-400k', 8, 400000],
  ]) {
    list.push({
      name: `lowfps-${name}`,
      q: 'past qatlam kechikishi',
      pub: {
        ...SCREEN_1080,
        codec: 'vp8',
        maxBitrate: 1500000,
        simulcast: true,
        layers: [
          { w: 1920, h: 1080, bitrate, fps },
          { w: 1920, h: 1080, bitrate: 800000, fps: 8 },
        ],
        mode: 'scroll',
      },
      sub: { quality: 'low' },
    })
  }
  return ONLY ? list.filter((s) => s.name.includes(ONLY)) : list
}

const pct = (arr, p) => {
  if (!arr.length) return null
  const s = [...arr].sort((a, b) => a - b)
  return s[Math.min(s.length - 1, Math.floor((s.length * p) / 100))]
}
const mean = (arr) => (arr.length ? arr.reduce((a, b) => a + b, 0) / arr.length : null)

async function runOne(browser, sc, i, total) {
  const room = `probe-${Date.now()}-${i}`
  const ctxPub = await browser.newContext()
  const ctxSub = await browser.newContext()
  const pubPage = await ctxPub.newPage()
  const subPage = await ctxSub.newPage()
  const errs = []
  for (const p of [pubPage, subPage]) {
    p.on('pageerror', (e) => errs.push(String(e)))
    p.on('console', (m) => {
      if (m.type() === 'error') errs.push(m.text())
    })
  }
  await pubPage.goto(PAGE)
  await subPage.goto(PAGE)
  await pubPage.waitForFunction(() => !!window.probe)
  await subPage.waitForFunction(() => !!window.probe)

  process.stdout.write(`[${i + 1}/${total}] ${sc.name} ... `)

  const pubInfo = await pubPage.evaluate(
    (cfg) => window.probe.startPublisher(cfg),
    { ...sc.pub, url: LK_URL, token: token(room, 'mentor', true) },
  )
  // Enkoder barqarorlashsin (birinchi soniyalarda bitreyt ramp-up qiladi)
  await pubPage.waitForTimeout(3000)

  const tJoin = Date.now()
  await subPage.evaluate(
    (cfg) => window.probe.startSubscriber(cfg),
    {
      url: LK_URL,
      token: token(room, 'oquvchi', false),
      width: sc.pub.width,
      height: sc.pub.height,
      mode: sc.pub.mode || 'scroll',
      ...(sc.sub || {}),
    },
  )

  const stats = async (p, who) => {
    const s = await p.evaluate(() => window.probe.collectStats())
    if (!s) throw new Error(`stats null (${who})`)
    return s
  }
  const s1pub = await stats(pubPage, 'pub1')
  await subPage.waitForTimeout(2000)
  const s1sub = await stats(subPage, 'sub1')
  await subPage.waitForTimeout(SECS * 1000)
  const s2pub = await stats(pubPage, 'pub2')
  const s2sub = await stats(subPage, 'sub2')
  const sub = await subPage.evaluate(() => window.probe.subResult())

  await pubPage.evaluate(() => window.probe.stop())
  await subPage.evaluate(() => window.probe.stop())
  await ctxPub.close()
  await ctxSub.close()

  // ── Bitreyt: ikki suratning FARQI (kumulyativ hisoblagichlar)
  const rate = (a, b, field) => {
    if (!a || !b) return null
    const dt = (b.ts - a.ts) / 1000
    return dt > 0 ? ((b[field] - a[field]) * 8) / dt : null
  }
  const outA = s1pub.outbound
  const outB = s2pub.outbound
  const layers = outB.map((b, idx) => {
    const a = outA.find((x) => x.rid === b.rid) || outA[idx]
    return {
      rid: b.rid || '-',
      w: b.frameWidth,
      h: b.frameHeight,
      fps: b.fps,
      bitrateKbps: Math.round((rate(a, b, 'bytesSent') || 0) / 1000),
      targetKbps: Math.round((b.targetBitrate || 0) / 1000),
      limit: b.qualityLimitationReason,
      encoder: b.encoderImplementation,
      codec: b.codec,
      msPerFrame:
        a && b.framesEncoded > a.framesEncoded
          ? +(((b.totalEncodeTime - a.totalEncodeTime) / (b.framesEncoded - a.framesEncoded)) * 1000).toFixed(2)
          : null,
    }
  })
  const inA = s1sub.inbound[0]
  const inB = s2sub.inbound[0]
  const jbA = inA ? inA.jitterBufferDelay / Math.max(1, inA.jitterBufferEmittedCount) : null
  const jbB = inB ? inB.jitterBufferDelay / Math.max(1, inB.jitterBufferEmittedCount) : null

  // Birinchi 3 s ni tashlab yuboramiz (enkoder/jitter-buffer ramp-up)
  const stable = sub.samples.filter((s) => s.t > (sub.firstFrameMs || 0) + 3000)
  const use = stable.length > 20 ? stable : sub.samples

  const r = {
    name: sc.name,
    q: sc.q,
    cfg: {
      codec: sc.pub.codec,
      cap: sc.pub.maxBitrate / 1000,
      src: `${sc.pub.width}x${sc.pub.height}@${sc.pub.fps}`,
      svc: sc.pub.scalabilityMode || null,
      simulcast: !!sc.pub.simulcast,
      mode: sc.pub.mode,
      subQuality: sc.sub?.quality || 'auto',
    },
    publisherEncodings: pubInfo.encodings,
    degradationPreference: pubInfo.degradationPreference,
    send: layers,
    recv: inB
      ? {
          codec: inB.codec,
          w: inB.frameWidth,
          h: inB.frameHeight,
          fps: inB.fps,
          bitrateKbps: Math.round((rate(inA, inB, 'bytesReceived') || 0) / 1000),
          packetsLost: inB.packetsLost - (inA?.packetsLost || 0),
          packetsReceived: inB.packetsReceived - (inA?.packetsReceived || 0),
          jitterMs: +(inB.jitter * 1000).toFixed(1),
          jitterBufferMs: +(((jbB * inB.jitterBufferEmittedCount - jbA * (inA?.jitterBufferEmittedCount || 0)) /
            Math.max(1, inB.jitterBufferEmittedCount - (inA?.jitterBufferEmittedCount || 0))) * 1000).toFixed(1),
          framesDropped: inB.framesDropped,
          decoder: inB.decoderImplementation,
          msPerFrame:
            inA && inB.framesDecoded > inA.framesDecoded
              ? +(((inB.totalDecodeTime - inA.totalDecodeTime) / (inB.framesDecoded - inA.framesDecoded)) * 1000).toFixed(2)
              : null,
        }
      : null,
    ice: { pub: s2pub.pair, sub: s2sub.pair, path: s2sub.transportType },
    join: {
      connectMs: sub.connectMs,
      firstFrameMs: sub.firstFrameMs,
      // "Qo'shilishdan ekranni ko'rgungacha" — foydalanuvchi sezadigan yagona raqam
      joinToFirstFrameMs: sub.t0 && sub.firstFrameMs !== null ? sub.t0 + sub.firstFrameMs - tJoin : null,
    },
    quality: {
      samples: use.length,
      markerFails: sub.markerFails,
      psnrP50: pct(use.map((s) => s.psnr), 50),
      psnrP10: pct(use.map((s) => s.psnr), 10),
      sharpP50: pct(use.map((s) => s.sharp), 50),
      latencyP50: pct(use.map((s) => s.latency), 50),
      latencyP95: pct(use.map((s) => s.latency), 95),
      latencyMean: mean(use.map((s) => s.latency)),
      renderFps: use.length / (SECS || 1),
    },
    errors: errs.slice(0, 5),
  }
  const fmt = (x, n = 1) => (x === null || x === undefined ? '—' : x.toFixed(n))
  console.log(
    `psnr ${fmt(r.quality.psnrP50)} dB · sharp ${fmt(r.quality.sharpP50, 2)} · ` +
      `lat p50 ${fmt(r.quality.latencyP50, 0)} ms · recv ${r.recv?.w}x${r.recv?.h}@${fmt(r.recv?.fps, 0)} ` +
      `${r.recv?.bitrateKbps} kbps · ttff ${r.join.joinToFirstFrameMs} ms · ice ${r.ice.path}`,
  )
  return r
}

const browser = await chromium.launch({
  args: [
    '--use-fake-ui-for-media-stream',
    '--use-fake-device-for-media-capture',
    '--autoplay-policy=no-user-gesture-required',
    '--disable-features=WebRtcHideLocalIpsWithMdns',
    '--enable-features=WebRTC-Vp9DependencyDescriptor',
  ],
})
const list = scenarios()
const results = []
for (let i = 0; i < list.length; i++) {
  try {
    results.push(await runOne(browser, list[i], i, list.length))
  } catch (e) {
    console.log('XATO:', list[i].name, String(e).slice(0, 200))
    results.push({ name: list[i].name, error: String(e).slice(0, 300) })
  }
}
await browser.close()
fs.writeFileSync(OUT, JSON.stringify(results, null, 2))
console.log('\nyozildi:', OUT)
