// O'QUVCHI tomoni: brauzerdan darsga qo'shiladi, kechikishni o'lchaydi.
//
// Ikki mustaqil o'lchov (biri ikkinchisini tekshiradi):
//  1. "Shishadan-shishagacha" — ustoz ekranida XOST soatiga sinxronlangan
//     millisekundli raqam turadi; biz qabul qilingan kadrni suratga olamiz va
//     o'sha lahzadagi xost vaqtidan ayiramiz. Bu foydalanuvchi HAQIQATAN
//     sezadigan kechikish (kamera→kodlash→tarmoq→bufer→dekod→ekran).
//  2. WebRTC `getStats()` — jitter-bufer, RTT, dekod vaqti, fps, rezolyutsiya.
//     Bu (1) ni tushuntiradi: kechikish qayerda tug'ilayotganini ko'rsatadi.
import { chromium } from '@playwright/test'
import fs from 'node:fs'

const BASE = process.env.BASE || 'https://app.169.58.104.245.sslip.io'
const SLUG = process.env.SLUG
const NAME = process.env.NAME || 'Talaba Brauzer'
const OUT = process.env.OUT || '/tmp/student'
const SECS = Number(process.env.SECS || 45)
if (!SLUG) { console.error('SLUG kerak'); process.exit(1) }
fs.mkdirSync(OUT, { recursive: true })

const browser = await chromium.launch({
  args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream', '--autoplay-policy=no-user-gesture-required'],
})
const ctx = await browser.newContext({ permissions: ['camera', 'microphone'], viewport: { width: 1600, height: 1000 } })
// RTCPeerConnection'ni sahifa yuklanishidan OLDIN ushlaymiz — ilovaning
// ichki obyektlariga (`window.__lkRoom` kabi) tayanmaymiz, chunki ular
// bo'lmasligi mumkin va o'lchov ilova tuzilishiga bog'lanib qolardi.
await ctx.addInitScript(() => {
  const Orig = window.RTCPeerConnection
  window.__pcs = []
  window.RTCPeerConnection = function (...args) {
    const pc = new Orig(...args)
    window.__pcs.push(pc)
    return pc
  }
  window.RTCPeerConnection.prototype = Orig.prototype
})
const page = await ctx.newPage()
page.on('console', (m) => { if (m.type() === 'error') console.log('  [brauzer xato]', m.text().slice(0, 160)) })

console.log(`→ ${BASE}/r/${SLUG}`)
await page.goto(`${BASE}/r/${SLUG}`, { waitUntil: 'networkidle' })
await page.locator('input').first().fill(NAME)
await page.getByRole('button', { name: /qo'shilish/i }).click()
await page.waitForFunction(() => location.pathname.includes('/room'), null, { timeout: 40000 })
console.log('✓ xonaga kirdi')

// Ustozning ekran ulashish treki kelishini kutamiz: eng katta o'lchamli,
// haqiqatan kadr oqayotgan <video>.
await page.waitForFunction(() => {
  const vs = [...document.querySelectorAll('video')]
  return vs.some((v) => v.videoWidth > 100 && !v.paused && v.currentTime > 0)
}, null, { timeout: 90000 })
console.log('✓ video oqmoqda')
await page.waitForTimeout(6000) // bufer barqarorlashsin

// ── 1) Shishadan-shishagacha ──────────────────────────────────────────────
// Har namunada: kadrni canvas'ga ko'chirib PNG qilamiz va O'SHA lahzadagi
// xost vaqtini yozamiz. Raqamni keyin ko'z bilan (yoki OCR) o'qiymiz.
const samples = []
for (let i = 0; i < 8; i++) {
  const shot = await page.evaluate(() => {
    const vs = [...document.querySelectorAll('video')].filter((v) => v.videoWidth > 100)
    if (!vs.length) return null
    const v = vs.sort((a, b) => b.videoWidth * b.videoHeight - a.videoWidth * a.videoHeight)[0]
    const c = document.createElement('canvas')
    c.width = v.videoWidth; c.height = v.videoHeight
    c.getContext('2d').drawImage(v, 0, 0)
    return { t: Date.now(), data: c.toDataURL('image/png'), w: v.videoWidth, h: v.videoHeight }
  })
  if (shot) {
    fs.writeFileSync(`${OUT}/frame_${i}_${shot.t}.png`, Buffer.from(shot.data.split(',')[1], 'base64'))
    samples.push({ i, t: shot.t, w: shot.w, h: shot.h })
  }
  await page.waitForTimeout(1200)
}
console.log(`✓ ${samples.length} kadr olindi (video ${samples[0]?.w}x${samples[0]?.h})`)
fs.writeFileSync(`${OUT}/samples.json`, JSON.stringify(samples, null, 2))

// ── 2) WebRTC statistikasi ────────────────────────────────────────────────
const stats = await page.evaluate(async (secs) => {
  const pcs = window.__pcs || []
  if (!pcs.length) return { err: 'RTCPeerConnection topilmadi' }
  const out = []
  const t0 = Date.now()
  while (Date.now() - t0 < secs * 1000) {
    for (const pc of pcs) {
      let rep
      try { rep = await pc.getStats() } catch { continue }
      let rtt = null
      rep.forEach((r) => {
        // Tarmoq RTT — nomzod juftidan (eng ishonchli manba).
        if (r.type === 'candidate-pair' && r.state === 'succeeded' && r.currentRoundTripTime != null) {
          rtt = r.currentRoundTripTime * 1000
        }
      })
      rep.forEach((r) => {
        if (r.type === 'inbound-rtp' && r.kind === 'video' && r.framesDecoded > 0) {
          out.push({
            t: Date.now(), rttMs: rtt,
            w: r.frameWidth, h: r.frameHeight, fps: r.framesPerSecond,
            jbDelay: r.jitterBufferDelay, jbCount: r.jitterBufferEmittedCount,
            decode: r.totalDecodeTime, decoded: r.framesDecoded,
            dropped: r.framesDropped, nack: r.nackCount, pli: r.pliCount,
            bytes: r.bytesReceived, jitter: r.jitter,
          })
        }
      })
    }
    await new Promise((r) => setTimeout(r, 1000))
  }
  return { out }
}, Math.min(SECS, 30))
fs.writeFileSync(`${OUT}/stats.json`, JSON.stringify(stats, null, 2))
console.log('✓ statistika yozildi:', stats.err || `${stats.out?.length || 0} namuna`)

await page.screenshot({ path: `${OUT}/page.png`, fullPage: false })
await browser.close()
console.log('TUGADI →', OUT)
