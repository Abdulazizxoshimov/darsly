// Live E2E uchun umumiy harness — test hisoblari (admin API orqali) + UI login +
// backend/DB tasdiqlash yordamchilari. Barcha web full-flow testlari shu fixture'larga tayanadi.
//
// MUHIM — «bitta akkaunt = bitta faol sessiya»: UI sessiyasi faolligida O'SHA
// akkaunt bilan API orqali qayta login QILMANG (UI sessiyasini bekor qiladi).
// DB tasdiqlashni UI orqali (ro'yxat qayta o'qiladi = DB round-trip) yoki boshqa
// (admin) hisob bilan qiling.
import { test as base, expect } from '@playwright/test'

const BACKEND = process.env.QA_BACKEND || 'http://localhost:8087'
const ADMIN_EMAIL = process.env.QA_SEED_ADMIN_EMAIL || 'admin@darsly.uz'
const ADMIN_PW = process.env.QA_SEED_ADMIN_PASSWORD || 'Admin12345'

async function jsonFetch(path, { method = 'GET', token, body } = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  const r = await fetch(`${BACKEND}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  return r
}

async function apiLogin(email, password) {
  const r = await jsonFetch('/api/v1/auth/login', { method: 'POST', body: { email, password } })
  if (!r.ok) throw new Error(`apiLogin ${email} → ${r.status}`)
  return (await r.json()).data.access_token
}

async function adminToken() {
  return apiLogin(ADMIN_EMAIL, ADMIN_PW)
}

function uniqEmail(prefix = 'qae2e') {
  return `${prefix}+${Date.now().toString(36)}${Math.floor(Math.random() * 1e6).toString(36)}@darsly.uz`
}

async function createAccount(adminTok, role) {
  const email = uniqEmail(role)
  const password = 'parol12345'
  const r = await jsonFetch('/api/v1/users', {
    method: 'POST',
    token: adminTok,
    body: { email, password, full_name: `QA ${role}`, role },
  })
  if (r.status !== 201) throw new Error(`createAccount ${role} → ${r.status} ${await r.text()}`)
  const data = (await r.json()).data
  return { id: data.id, email, password, role }
}

// Dars seed qiladi — mentor API tokeni bilan (UI-login'dan OLDIN chaqiriladi:
// token keyin ISHLATILMAYDI, shuning uchun UI sessiyasi uni bekor qilsa ham dars DB'да qoladi).
async function seedLesson(creds, overrides = {}) {
  const tok = await apiLogin(creds.email, creds.password)
  const body = {
    title: 'Seed dars',
    duration_min: 30,
    is_recording_enabled: false,
    is_waiting_room_enabled: false,
    ...overrides,
  }
  const r = await jsonFetch('/api/v1/lessons', { method: 'POST', token: tok, body })
  if (r.status !== 201) throw new Error(`seedLesson → ${r.status} ${await r.text()}`)
  return (await r.json()).data
}

async function deleteAccount(adminTok, id) {
  try {
    await jsonFetch(`/api/v1/users/${id}`, { method: 'DELETE', token: adminTok })
  } catch {
    /* cleanup — xato yutiladi */
  }
}

// UI orqali login (haqiqiy forma — /auth route'да). Login → /app (dashboard).
async function uiLogin(page, email, password) {
  await page.goto('/auth')
  await page.getByPlaceholder('email@misol.uz').fill(email)
  await page.getByPlaceholder('••••••••').fill(password)
  await page.locator('button[type="submit"]').click()
  await page.waitForURL(/\/app/, { timeout: 20000 })
}

export const test = base.extend({
  // Seed admin API tokeni (test hisoblarini yaratish/tozalash uchun).
  admin: async ({}, use) => {
    await use(await adminToken())
  },
  // Yangi mentor hisobi (yaratiladi → test → o'chiriladi). API login QILMAYDI.
  mentorAccount: async ({ admin }, use) => {
    const m = await createAccount(admin, 'mentor')
    await use(m)
    await deleteAccount(admin, m.id)
  },
  // Yangi student hisobi.
  studentAccount: async ({ admin }, use) => {
    const s = await createAccount(admin, 'student')
    await use(s)
    await deleteAccount(admin, s.id)
  },
  // UI'da mentor sifatida kirilган sahifa.
  mentorPage: async ({ page, mentorAccount }, use) => {
    await uiLogin(page, mentorAccount.email, mentorAccount.password)
    await use(page)
  },
  // UI'да ADMIN sifatida kirilган sahifa (seed admin). ⚠️ `admin` (API token)
  // fixture'i bilan BIR TESTDA ishlatmang — bir xil akkaunt, sessiya ziddiyati.
  adminPage: async ({ page }, use) => {
    await uiLogin(page, ADMIN_EMAIL, ADMIN_PW)
    await use(page)
  },
  // Mentor + oldindan DB'да 1 dars (seed API orqali) + UI login → {page, lesson}.
  mentorPageWithLesson: async ({ page, mentorAccount }, use) => {
    const lesson = await seedLesson(mentorAccount, { title: `Boshqaruv ${Date.now().toString(36)}` })
    await uiLogin(page, mentorAccount.email, mentorAccount.password)
    await use({ page, lesson })
  },
})

export { expect }
export const api = {
  BACKEND, jsonFetch, apiLogin, adminToken, createAccount, deleteAccount, uiLogin, seedLesson,
}
