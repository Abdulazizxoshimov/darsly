// Full-flow: Dars tahrirlash / o'chirish / rejalashtirish (UI → backend → DB)
import { test, expect } from './fixtures.js'

// Field komponenti label'ни input'ga bog'lamaydi → konteyner orqali.
function field(page, label) {
  return page
    .locator('.field', { has: page.locator('label.field__label', { hasText: label }) })
    .locator('input, textarea')
    .first()
}

test.describe('Darslarni boshqarish (live)', () => {
  test('darsni tahrirlash → nom o\'zgaradi (DB → ro\'yxat)', async ({ mentorPageWithLesson }) => {
    const { page, lesson } = mentorPageWithLesson
    await page.goto('/app')
    await expect(page.getByText(lesson.title)).toBeVisible({ timeout: 15000 })

    await page.getByRole('button', { name: 'Tahrirlash' }).first().click()
    await expect(field(page, 'Dars nomi')).toBeVisible()

    const newTitle = `${lesson.title} TAHRIR`
    await field(page, 'Dars nomi').fill(newTitle)
    await page.getByRole('button', { name: /Saqla|Yangila/i }).first().click()

    // DB yangilandi → ro'yxat qayta o'qiladi → yangi nom ko'rinadi.
    await expect(page.getByText(newTitle)).toBeVisible({ timeout: 15000 })
  })

  test('darsni o\'chirish → ro\'yxatdan yo\'qoladi (DB)', async ({ mentorPageWithLesson }) => {
    const { page, lesson } = mentorPageWithLesson
    await page.goto('/app')
    await expect(page.getByText(lesson.title)).toBeVisible({ timeout: 15000 })

    await page.getByRole('button', { name: 'Tahrirlash' }).first().click()
    // O'chirish `window.confirm` ishlatadi → avtomatik tasdiqlaymiz.
    page.on('dialog', (d) => d.accept())
    await page.getByRole('button', { name: /o'chir/i }).first().click()

    // DELETE /lessons/:id → ro'yxatdan yo'qoladi.
    await expect(page.getByText(lesson.title)).toHaveCount(0, { timeout: 15000 })
  })

  test('dars rejalashtirish → Jadvalда (DB) paydo bo\'ladi', async ({ mentorPage }) => {
    const title = `Rejali dars ${Date.now().toString(36)}`
    await mentorPage.goto('/app/schedule')
    await mentorPage.getByRole('button', { name: /Dars rejalashtirish/i }).first().click()
    await expect(field(mentorPage, 'Dars nomi')).toBeVisible()

    await field(mentorPage, 'Dars nomi').fill(title)
    await field(mentorPage, 'Sana va vaqt').fill('2026-12-01T10:00')
    await mentorPage.getByRole('button', { name: /Rejalashtir|Saqla/i }).last().click()

    // Jadval ro'yxati backend→DB'dan → rejalashtirilган dars ko'rinadi.
    await expect(mentorPage.getByText(title)).toBeVisible({ timeout: 15000 })
  })
})
