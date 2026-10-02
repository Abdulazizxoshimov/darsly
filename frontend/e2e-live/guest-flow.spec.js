// Full-flow: O'quvchi / guest oqimi (public /r/:slug → backend → DB), loginsiz.
//
// Audit dizayni: dars `scheduled` (host kirmagan) bo'lsa guest submit → join POST
// EMAS, klient "hali boshlanmagan" kutish ekranini ko'rsatadi; join/parol/kutish-xona
// navigatsiyasi dars `live` bo'lganда (LiveKit host kirgач) ishlaydi. Shuning uchun
// bu yerda scheduled-oldi oqim tekshiriladi; live guest-join → LiveKit kerak (GAP,
// docs/E2E-FULLFLOW.md — sintetik host bilan keyin).
import { test, expect, api } from './fixtures.js'

function field(page, label) {
  return page
    .locator('.field', { has: page.locator('label.field__label', { hasText: label }) })
    .locator('input, textarea')
    .first()
}

test.describe("O'quvchi / guest oqimi (live)", () => {
  test('havola preview (DB) → ism bilan → "hali boshlanmagan" kutish', async ({ page, mentorAccount }) => {
    const lesson = await api.seedLesson(mentorAccount, {
      title: `Guest dars ${Date.now().toString(36)}`,
      is_waiting_room_enabled: false,
    })
    await page.goto(`/r/${lesson.join_slug}`)
    // Preview backend→DB'dan: dars nomi + mentor ismi.
    await expect(page.getByRole('heading', { name: lesson.title })).toBeVisible({ timeout: 15000 })

    await field(page, 'Ismingiz').fill("Aziz O'quvchi")
    await page.getByRole('button', { name: /Dars boshlanishini kutish|Darsga qo'shilish/i }).click()
    await expect(page.getByText(/hali boshlanmagan/i)).toBeVisible({ timeout: 15000 })
  })

  test('kutish xonali dars → preview "Kutish xonasi yoqilgan" belgisi', async ({ page, mentorAccount }) => {
    const lesson = await api.seedLesson(mentorAccount, {
      title: `WR dars ${Date.now().toString(36)}`,
      is_waiting_room_enabled: true,
    })
    await page.goto(`/r/${lesson.join_slug}`)
    await expect(page.getByRole('heading', { name: lesson.title })).toBeVisible({ timeout: 15000 })
    // Kutish xonasi yoqilganini preview ko'rsatadi (backend LessonPublic).
    await expect(page.getByText(/Kutish xonasi yoqilgan/i)).toBeVisible()
  })

  test('parolli dars → preview "Parol" maydonini ko\'rsatadi (has_passcode)', async ({ page, mentorAccount }) => {
    const lesson = await api.seedLesson(mentorAccount, {
      title: `Parol dars ${Date.now().toString(36)}`,
      passcode: '1234',
    })
    await page.goto(`/r/${lesson.join_slug}`)
    await expect(page.getByRole('heading', { name: lesson.title })).toBeVisible({ timeout: 15000 })
    // has_passcode=true → "Parol" maydoni ko'rinadi (backend LessonPublic.has_passcode).
    await expect(field(page, 'Parol')).toBeVisible()
  })
})
