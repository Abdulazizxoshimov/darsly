// Full-flow: JONLI XONA — ikki kontekst (host + guest) bitta LiveKit xonasida.
// Talab: backend + LiveKit (node_ip LAN) ishlab turishi. Fake media (config launchOptions).
import { test, expect, api } from './fixtures.js'

function field(page, label) {
  return page
    .locator('.field', { has: page.locator('label.field__label', { hasText: label }) })
    .locator('input, textarea')
    .first()
}

async function newGuest(browser) {
  const ctx = await browser.newContext({
    baseURL: 'http://localhost:3000',
    permissions: ['camera', 'microphone'],
  })
  return { ctx, page: await ctx.newPage() }
}

test.describe('Jonli xona (live, 2 ishtirokchi)', () => {
  test("host kiradi → guest qo'shiladi → chat ikki tomonlama → qo'l", async ({ browser, page, mentorAccount }) => {
    test.setTimeout(120_000)
    const lesson = await api.seedLesson(mentorAccount, {
      title: `Live dars ${Date.now().toString(36)}`,
      is_waiting_room_enabled: false,
    })

    // HOST: login → xona (host token dars'ni LIVE qiladi, LiveKit'ga ulanadi).
    await api.uiLogin(page, mentorAccount.email, mentorAccount.password)
    await page.goto(`/app/lesson/${lesson.id}/room`)
    await expect(page.getByLabel('Chat').first()).toBeVisible({ timeout: 40_000 })

    // GUEST: alohida kontekst → havola → ism → dars LIVE → to'g'ridan join → xona.
    const { ctx, page: guest } = await newGuest(browser)
    await guest.goto(`/r/${lesson.join_slug}`)
    await field(guest, 'Ismingiz').fill('E2E Guest')
    await guest.locator('button[type="submit"]').click()
    await expect(guest).toHaveURL(new RegExp(`/r/${lesson.join_slug}/room`), { timeout: 25_000 })
    await expect(guest.getByLabel('Chat').first()).toBeVisible({ timeout: 40_000 })

    // CHAT: guest → host.
    await guest.getByLabel('Chat').first().click()
    await guest.getByPlaceholder('Xabar yozing…').fill('Salom ustoz!')
    await guest.getByLabel('Yuborish').click()
    await page.getByLabel('Chat').first().click()
    await expect(page.getByText('Salom ustoz!')).toBeVisible({ timeout: 20_000 })

    // CHAT: host → guest.
    await page.getByPlaceholder('Xabar yozing…').fill("Salom o'quvchi!")
    await page.getByLabel('Yuborish').click()
    await expect(guest.getByText("Salom o'quvchi!")).toBeVisible({ timeout: 20_000 })

    // QO'L: guest ko'taradi → host guest nomini ko'radi (ishtirokchi/qo'l).
    await guest.getByLabel("Qo'l").first().click()
    await expect(page.getByText('E2E Guest').first()).toBeVisible({ timeout: 20_000 })

    await ctx.close()
  })

  test("kutish xonasi: guest so'rov → host QABUL → guest xonaga kiradi", async ({ browser, page, mentorAccount }) => {
    test.setTimeout(120_000)
    const lesson = await api.seedLesson(mentorAccount, {
      title: `WR live ${Date.now().toString(36)}`,
      is_waiting_room_enabled: true,
    })

    // HOST xonada (dars live).
    await api.uiLogin(page, mentorAccount.email, mentorAccount.password)
    await page.goto(`/app/lesson/${lesson.id}/room`)
    await expect(page.getByLabel('Chat').first()).toBeVisible({ timeout: 40_000 })

    // GUEST: so'rov → /waiting.
    const { ctx, page: guest } = await newGuest(browser)
    await guest.goto(`/r/${lesson.join_slug}`)
    await field(guest, 'Ismingiz').fill('Kutuvchi Guest')
    await guest.locator('button[type="submit"]').click()
    await expect(guest).toHaveURL(/\/waiting/, { timeout: 25_000 })

    // HOST: Ishtirokchilar paneli → so'rov ko'rinadi → "Qabul qilish".
    await page.getByLabel('Ishtirokchilar').first().click()
    await expect(page.getByText('Kutuvchi Guest')).toBeVisible({ timeout: 25_000 })
    await page.getByTitle('Qabul qilish').first().click()

    // GUEST: real-time token (WS) → xonaga avtomatik o'tadi.
    await expect(guest).toHaveURL(new RegExp(`/r/${lesson.join_slug}/room`), { timeout: 30_000 })

    await ctx.close()
  })
})
