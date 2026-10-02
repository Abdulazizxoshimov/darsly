// Full-flow: Darslar (brauzer UI → backend → DB → UI qayta o'qiydi)
import { test, expect } from './fixtures.js'

test.describe('Darslar (live)', () => {
  test('dars yaratish → xonaga o\'tadi + ro\'yxatda (DB) ko\'rinadi', async ({ mentorPage }) => {
    const title = `E2E dars ${Date.now().toString(36)}`

    // 1) Dashboard'да "Dars yaratish" → modal.
    await mentorPage.getByRole('button', { name: /Dars yaratish/i }).first().click()
    // Modal ochildi — title input ko'rinadi.
    await expect(mentorPage.getByPlaceholder('Masalan: Kvadrat tenglamalar')).toBeVisible()

    // 2) Nom + yaratish.
    await mentorPage.getByPlaceholder('Masalan: Kvadrat tenglamalar').fill(title)
    await mentorPage.getByRole('button', { name: /Yaratish va boshlash/i }).click()

    // 3) Yaratildi → DARHOL host xonasiga o'tadi (/app/lesson/:id/room) = DB'да yaratildi.
    await expect(mentorPage).toHaveURL(/\/app\/lesson\/[^/]+\/room/, { timeout: 20000 })

    // 4) Dashboard'ga qaytamiz → ro'yxat backend→DB'dan qayta o'qiladi → dars ko'rinadi.
    await mentorPage.goto('/app')
    await expect(mentorPage.getByText(title)).toBeVisible({ timeout: 15000 })
  })

  test('dars yaratish — nom bo\'sh bo\'lsa yaratilmaydi', async ({ mentorPage }) => {
    await mentorPage.getByRole('button', { name: /Dars yaratish/i }).first().click()
    await expect(mentorPage.getByPlaceholder('Masalan: Kvadrat tenglamalar')).toBeVisible()
    // Nom bo'sh → "Yaratish va boshlash" bosilsa ham xonaga o'tmaydi (validatsiya).
    await mentorPage.getByRole('button', { name: /Yaratish va boshlash/i }).click()
    await expect(mentorPage).not.toHaveURL(/\/room/, { timeout: 4000 })
  })
})
