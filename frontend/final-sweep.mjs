// YAKUNIY OQIM SINOVI — mentor + 2 o'quvchi bir vaqtda (Zoom modeli).
// Ishga tushirish: node final-sweep.mjs
import { chromium } from '@playwright/test'

const BASE = process.env.SWEEP_BASE || 'http://localhost:3000'
const API = process.env.SWEEP_API || 'http://localhost:8087'
const res = []
const ok = (n) => { res.push(`PASS  ${n}`); console.log(`PASS  ${n}`) }
const bad = (n, e) => { res.push(`FAIL  ${n} — ${String(e).slice(0, 160)}`); console.log(`FAIL  ${n} — ${String(e).slice(0, 160)}`) }
async function step(n, fn) { try { await fn(); ok(n) } catch (e) { bad(n, e) } }

// ── API yordamchilari ──
async function api(path, { method = 'GET', token, body } = {}) {
  const r = await fetch(`${API}/api/v1${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await r.text()
  let json = null
  try { json = JSON.parse(text) } catch { /* bo'sh javob */ }
  return { status: r.status, json }
}

// ⚠️ TARTIB MUHIM: «bitta faol sessiya» siyosati tufayli har login oldingisini
// bekor qiladi. Shu sabab avval BRAUZER kiradi, keyin API chaqiruvlari AYNI
// o'sha sessiya tokeni bilan yuriladi (haqiqiy foydalanuvchi kabi).
const browser = await chromium.launch({ args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream'] })
const mCtx = await browser.newContext({ permissions: ['camera', 'microphone'] })
const mentor = await mCtx.newPage()
await step('mentor login (web)', async () => {
  await mentor.goto(`${BASE}/auth`, { waitUntil: 'networkidle' })
  await mentor.locator('input').first().fill('admin@darsly.uz')
  await mentor.locator('input[type="password"]').fill('Admin12345')
  await mentor.getByRole('button', { name: /^kirish$/i }).click()
  await mentor.waitForURL('**/app**', { timeout: 15000 })
})
const TOKEN = await mentor.evaluate(() => localStorage.getItem('darsly.access'))
if (!TOKEN) { console.log('FAIL  sessiya tokeni olinmadi'); process.exit(1) }
ok('sessiya tokeni brauzerdan olindi')

// Sinov darsi
const created = await api('/lessons', { method: 'POST', token: TOKEN, body: { title: 'Final sweep darsi' } })
const LESSON = created.json?.data
await step('dars yaratildi (tezkor)', async () => { if (!LESSON?.id) throw new Error(JSON.stringify(created.json)) })
await step('yangi audio-siyosat maydonlari bor', async () => {
  if (LESSON.mute_on_entry !== true || LESSON.allow_self_unmute !== true) throw new Error(JSON.stringify(LESSON))
})
await step('mentor xonaga kirdi (dars live bo\'ldi)', async () => {
  await mentor.goto(`${BASE}/app/lesson/${LESSON.id}/room`, { waitUntil: 'networkidle' })
  await mentor.waitForTimeout(12000)
  const l = await api(`/lessons/${LESSON.id}`, { token: TOKEN })
  if (l.json?.data?.status !== 'live') throw new Error('status: ' + l.json?.data?.status)
})

// ── O'QUVCHILAR ──
const slug = LESSON.join_slug
const students = []
for (const name of ['Talaba Bir', 'Talaba Ikki']) {
  const c = await browser.newContext({ permissions: ['camera', 'microphone'] })
  const p = await c.newPage()
  await step(`o'quvchi «${name}» qo'shildi`, async () => {
    await p.goto(`${BASE}/r/${slug}`, { waitUntil: 'networkidle' })
    await p.locator('input').first().fill(name)
    await p.getByRole('button', { name: /qo'shilish/i }).click()
    await p.waitForURL('**/room**', { timeout: 20000 })
    await p.waitForTimeout(8000)
  })
  students.push({ name, page: p })
}

await step('xonada 3 ishtirokchi ko\'rinadi (API roster)', async () => {
  const r = await api(`/lessons/${LESSON.id}/participants`, { token: TOKEN })
  const n = (r.json?.data || []).length
  if (n < 3) throw new Error(`roster: ${n}`)
})

await step('Mute All (allow_self_unmute=false) qo\'llandi', async () => {
  const r = await api(`/lessons/${LESSON.id}/mute-all`, { method: 'POST', token: TOKEN, body: { allow_self_unmute: false } })
  if (r.status !== 204) throw new Error('status ' + r.status)
  const l = await api(`/lessons/${LESSON.id}`, { token: TOKEN })
  if (l.json?.data?.allow_self_unmute !== false) throw new Error('bayroq yangilanmadi')
})

await step('chat: mentor xabar yubordi va tarixda ko\'rinadi', async () => {
  const send = await api(`/lessons/${LESSON.id}/chat`, { method: 'POST', token: TOKEN, body: { body: 'Sweep sinov xabari' } })
  if (send.status >= 400) throw new Error('yuborish ' + send.status)
  const hist = await api(`/lessons/${LESSON.id}/chat`, { token: TOKEN })
  const arr = hist.json?.data?.messages || hist.json?.data || []
  if (!JSON.stringify(arr).includes('Sweep sinov xabari')) throw new Error('tarixda yo\'q')
})

// Ban: birinchi o'quvchini doimiy chiqarish
await step('ban scope=mentor (doimiy) ishladi', async () => {
  const r = await api(`/lessons/${LESSON.id}/participants`, { token: TOKEN })
  const guest = (r.json?.data || []).find((p) => /talaba bir/i.test(p.name || p.display_name || ''))
  if (!guest) throw new Error('o\'quvchi topilmadi')
  const rem = await api(`/lessons/${LESSON.id}/participants/${guest.identity}/remove`, {
    method: 'POST', token: TOKEN, body: { scope: 'mentor' },
  })
  if (rem.status !== 204) throw new Error('remove ' + rem.status)
  const bl = await api('/blocklist', { token: TOKEN })
  if (!JSON.stringify(bl.json?.data || []).toLowerCase().includes('talaba bir')) throw new Error('blocklistda yo\'q')
})

await step('ban qilingan ism qayta kira olmaydi (403)', async () => {
  const j = await fetch(`${API}/api/v1/joinlink/${slug}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ guest_name: 'Talaba Bir' }),
  })
  if (j.status !== 403) throw new Error('status ' + j.status)
})

// Dars yakuni + yozuv
await step('dars yakunlandi', async () => {
  const r = await api(`/lessons/${LESSON.id}/end`, { method: 'POST', token: TOKEN })
  if (r.status >= 400) throw new Error('end ' + r.status)
})
await step('yakunlangan dars havolasi join bermaydi', async () => {
  const j = await fetch(`${API}/api/v1/joinlink/${slug}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ guest_name: 'Kech Qolgan' }),
  })
  const b = await j.json()
  if (b?.data?.next_step !== 'lesson_ended') throw new Error(JSON.stringify(b).slice(0, 120))
})
await step('yozuv tayyor holatga o\'tdi', async () => {
  for (let i = 0; i < 20; i++) {
    const r = await api(`/lessons/${LESSON.id}/recordings`, { token: TOKEN })
    const rec = (r.json?.data || [])[0]
    if (rec?.status === 'ready') return
    await new Promise((s) => setTimeout(s, 3000))
  }
  throw new Error('yozuv 60s ichida ready bo\'lmadi')
})
await step('yozuvda expires_at bor (30 kunlik retention)', async () => {
  const r = await api(`/lessons/${LESSON.id}/recordings`, { token: TOKEN })
  const rec = (r.json?.data || [])[0]
  if (!rec?.expires_at) throw new Error('expires_at yo\'q: ' + JSON.stringify(rec).slice(0, 120))
  const days = (new Date(rec.expires_at) - Date.now()) / 86400000
  if (days < 25 || days > 35) throw new Error('muddat g\'alati: ' + days.toFixed(1) + ' kun')
})

// ── 2-hafta: bitta faol sessiya ──
// ⚠️ Bu sinov JORIY sessiyani bekor qiladi — shuning uchun oxirida turadi va
// tozalash uchun YANGI token qaytaradi (aks holda tozalash 401 olib, sinov
// ma'lumotlari bazada qolib ketardi — 2026-07-31 da aynan shunday bo'ldi:
// qora ro'yxatdagi «Talaba Bir» keyingi yurishni buzdi).
let CLEAN_TOKEN = TOKEN
await step('bitta faol sessiya: ikkinchi login eskisini chiqaradi', async () => {
  const first = await api('/auth/login', { method: 'POST', body: { email: 'admin@darsly.uz', password: 'Admin12345' } })
  const oldRefresh = first.json?.data?.refresh_token
  if (!oldRefresh) throw new Error('birinchi login refresh bermadi')
  const second = await api('/auth/login', { method: 'POST', body: { email: 'admin@darsly.uz', password: 'Admin12345' } })
  CLEAN_TOKEN = second.json?.data?.access_token || TOKEN
  const r = await api('/auth/refresh', { method: 'POST', body: { refresh_token: oldRefresh } })
  if (r.status !== 401) throw new Error('eski refresh hali ishlayapti: ' + r.status)
  const code = r.json?.code
  if (code !== 'SESSION_REVOKED') throw new Error('kod: ' + code)
})


await browser.close()

// Tozalash — sessiya sinovidan keyingi YANGI token bilan.
await api(`/lessons/${LESSON.id}`, { method: 'DELETE', token: CLEAN_TOKEN })
const bl = await api('/blocklist', { token: CLEAN_TOKEN })
for (const b of bl.json?.data || []) {
  if (/talaba/i.test(b.display_name || '')) await api(`/blocklist/${b.id}`, { method: 'DELETE', token: CLEAN_TOKEN })
}

console.log('\n=== YAKUN ===')
console.log(`${res.filter((r) => r.startsWith('PASS')).length} PASS · ${res.filter((r) => r.startsWith('FAIL')).length} FAIL`)
res.filter((r) => r.startsWith('FAIL')).forEach((r) => console.log(r))
