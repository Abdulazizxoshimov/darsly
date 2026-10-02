// Full-flow: XONA ICHIDAGI ACTION'lar — so'rovnoma, moderatsiya, reject/admit-all, reaksiya.
// Har test: host (asosiy page) + guest(lar) (alohida kontekst) real LiveKit xonasida.
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

// Host xonaga kiradi (dars LIVE bo'ladi).
async function hostEnter(page, mentorAccount, lesson) {
  await api.uiLogin(page, mentorAccount.email, mentorAccount.password)
  await page.goto(`/app/lesson/${lesson.id}/room`)
  await expect(page.getByLabel('Chat').first()).toBeVisible({ timeout: 40_000 })
}

// Guest havola orqali to'g'ridan xonaga kiradi (WR o'chiq, dars live).
async function guestJoin(browser, lesson, name) {
  const { ctx, page } = await newGuest(browser)
  await page.goto(`/r/${lesson.join_slug}`)
  await field(page, 'Ismingiz').fill(name)
  await page.locator('button[type="submit"]').click()
  await expect(page).toHaveURL(new RegExp(`/r/${lesson.join_slug}/room`), { timeout: 25_000 })
  await expect(page.getByLabel('Chat').first()).toBeVisible({ timeout: 40_000 })
  return { ctx, page }
}

test.describe("Xona ichidagi action'lar (live)", () => {
  test("so'rovnoma to'liq oqimi: yaratish → guest ovoz → natija → e'lon → yopish", async ({ browser, page, mentorAccount }) => {
    test.setTimeout(150_000)
    const lesson = await api.seedLesson(mentorAccount, {
      title: `Poll ${Date.now().toString(36)}`,
      is_waiting_room_enabled: false,
    })
    await hostEnter(page, mentorAccount, lesson)
    const g = await guestJoin(browser, lesson, 'Poll Guest')

    // HOST: so'rovnoma yaratish (public — e'lon qilingach guest ko'radi).
    await page.getByLabel("So'rovnoma").first().click()
    await page.getByRole('button', { name: "Yangi so'rovnoma" }).click()
    await page.getByPlaceholder('Savol').fill('2+2 nechchi?')
    await page.getByPlaceholder('Variant 1').fill('3')
    await page.getByPlaceholder('Variant 2').fill('4')
    await page.locator('#poll-visibility').selectOption('public')
    await page.getByRole('button', { name: 'Boshlash' }).click()

    // GUEST: poll paneli avtomatik ochiladi (data-channel 'open') → ovoz beradi.
    await expect(g.page.getByText('2+2 nechchi?')).toBeVisible({ timeout: 20_000 })
    await g.page.locator('.poll-option', { hasText: '4' }).click()
    await expect(g.page.getByText('Ovozingiz qabul qilindi')).toBeVisible({ timeout: 15_000 })

    // HOST: natija keladi (3s refetch) — "1 ovoz" (DB'dan).
    await expect(page.getByText('1 ovoz')).toBeVisible({ timeout: 15_000 })

    // HOST: natijani E'LON qiladi → guest ResultBars ko'radi (server broadcast).
    await page.getByRole('button', { name: "Natijani e'lon qilish" }).click()
    await expect(page.getByText("Natija e'lon qilingan")).toBeVisible({ timeout: 15_000 })
    await expect(g.page.getByText('1 ovoz')).toBeVisible({ timeout: 20_000 })

    // HOST: so'rovnomani yopadi → holat "Yopiq".
    await page.getByRole('button', { name: 'Yopish va natija' }).click()
    await expect(page.getByText('Yopiq')).toBeVisible({ timeout: 15_000 })

    await g.ctx.close()
  })

  test('moderatsiya: videoga ruxsat → bekor → mute → chiqarib yuborish', async ({ browser, page, mentorAccount }) => {
    test.setTimeout(150_000)
    const lesson = await api.seedLesson(mentorAccount, {
      title: `Mod ${Date.now().toString(36)}`,
      is_waiting_room_enabled: false,
    })
    await hostEnter(page, mentorAccount, lesson)
    const g = await guestJoin(browser, lesson, 'Mod Guest')

    // HOST: ishtirokchilar paneli → guest qatori.
    await page.getByLabel('Ishtirokchilar').first().click()
    const row = page.locator('.p-row', { hasText: 'Mod Guest' })
    await expect(row).toBeVisible({ timeout: 20_000 })

    // Videoga ruxsat (LiveKit UpdateParticipant → guest kamera yoqa oladi).
    await row.hover()
    await row.getByTitle('Videoga ruxsat').click()
    await expect(page.getByText('Video yoqildi')).toBeVisible({ timeout: 15_000 })
    // GUEST tomonda kamera tugmasi faollashadi (canPublishCamera granted).
    await expect(g.page.getByLabel(/Kamera/).first()).toBeEnabled({ timeout: 20_000 })

    // Videoni bekor qilish.
    await row.hover()
    await row.getByTitle('Videoni bekor qilish').click()
    await expect(page.getByText('Video bekor qilindi')).toBeVisible({ timeout: 15_000 })

    // Mute (server-side track mute).
    await row.hover()
    await row.getByTitle('Mute').click()

    // Chiqarib yuborish (shu darsdan) → guest ro'yxatdan yo'qoladi.
    await row.hover()
    await row.getByTitle('Chiqarib yuborish').click()
    await expect(page.getByRole('heading', { name: 'Chiqarib yuborish' })).toBeVisible()
    await page.getByRole('button', { name: 'Shu darsdan' }).click()
    await expect(page.getByText('darsdan chiqarildi')).toBeVisible({ timeout: 15_000 })
    await expect(row).toHaveCount(0, { timeout: 25_000 })

    await g.ctx.close()
  })

  test("kutish xonasi: RAD ETISH + HAMMASINI KIRITISH (2 guest)", async ({ browser, page, mentorAccount }) => {
    test.setTimeout(150_000)
    const lesson = await api.seedLesson(mentorAccount, {
      title: `WR2 ${Date.now().toString(36)}`,
      is_waiting_room_enabled: true,
    })
    await hostEnter(page, mentorAccount, lesson)

    // Ikki guest so'rov yuboradi → /waiting.
    const g1 = await newGuest(browser)
    await g1.page.goto(`/r/${lesson.join_slug}`)
    await field(g1.page, 'Ismingiz').fill('Rad Guest')
    await g1.page.locator('button[type="submit"]').click()
    await expect(g1.page).toHaveURL(/\/waiting/, { timeout: 25_000 })

    const g2 = await newGuest(browser)
    await g2.page.goto(`/r/${lesson.join_slug}`)
    await field(g2.page, 'Ismingiz').fill('Kirit Guest')
    await g2.page.locator('button[type="submit"]').click()
    await expect(g2.page).toHaveURL(/\/waiting/, { timeout: 25_000 })

    // HOST: birinchisini RAD etadi → guest1 "Kirish rad etildi" (WS push).
    await page.getByLabel('Ishtirokchilar').first().click()
    await expect(page.getByText('Rad Guest')).toBeVisible({ timeout: 25_000 })
    await page
      .locator('.row', { has: page.getByText('Rad Guest') })
      .getByTitle('Rad etish')
      .first()
      .click()
    await expect(g1.page.getByText('Kirish rad etildi')).toBeVisible({ timeout: 25_000 })

    // HOST: qolganini HAMMASINI KIRITISH → guest2 xonaga kiradi.
    await expect(page.getByText('Kirit Guest')).toBeVisible({ timeout: 25_000 })
    await page.getByRole('button', { name: 'Hammasini kiritish' }).click()
    await page.getByRole('button', { name: 'Kiritish', exact: true }).click()
    await expect(page.getByText(/o‘quvchi kiritildi/)).toBeVisible({ timeout: 15_000 })
    await expect(g2.page).toHaveURL(new RegExp(`/r/${lesson.join_slug}/room`), { timeout: 30_000 })

    await g1.ctx.close()
    await g2.ctx.close()
  })

  test('reaksiya: guest emoji yuboradi → hostda ko‘rinadi', async ({ browser, page, mentorAccount }) => {
    test.setTimeout(120_000)
    const lesson = await api.seedLesson(mentorAccount, {
      title: `React ${Date.now().toString(36)}`,
      is_waiting_room_enabled: false,
    })
    await hostEnter(page, mentorAccount, lesson)
    const g = await guestJoin(browser, lesson, 'Emoji Guest')

    // GUEST: Reaksiya → palitra → birinchi emoji.
    await g.page.getByLabel('Reaksiya').first().click()
    const firstEmoji = g.page.locator('.reactions-pop button').first()
    const emoji = (await firstEmoji.textContent()).trim()
    await firstEmoji.click()

    // HOST: emoji overlay'da paydo bo'ladi (data-channel).
    await expect(page.getByText(emoji).first()).toBeVisible({ timeout: 20_000 })

    await g.ctx.close()
  })
})
