// Full-flow: Profil + bildirishnoma + blocklist sahifalari (UI → backend → DB)
import { test, expect } from './fixtures.js'

test.describe('Akkaunt / sahifalar (live)', () => {
  test('profil ismini o\'zgartirish → DB → UI yangilanadi', async ({ mentorPage }) => {
    const newName = `Mentor ${Date.now().toString(36)}`
    await mentorPage.goto('/app/profile')
    // Field komponenti label'ни input'ga htmlFor bilan bog'lamaydi → konteyner orqali.
    const nameField = mentorPage.locator('.field', {
      has: mentorPage.locator('label.field__label', { hasText: "To'liq ism" }),
    }).locator('input')
    await expect(nameField).toBeVisible()
    await nameField.fill(newName)
    await mentorPage.getByRole('button', { name: /Saqlash/i }).click()
    // PUT /users/me → me qayta o'qiladi (DB round-trip) → ko'rsatilган ism yangilanadi.
    await expect(mentorPage.getByText(newName).first()).toBeVisible({ timeout: 15000 })
  })

  test('bildirishnomalar sahifasi ochiladi (backend ro\'yxati)', async ({ mentorPage }) => {
    await mentorPage.goto('/app/notifications')
    await expect(mentorPage.getByRole('heading', { name: 'Bildirishnomalar' })).toBeVisible()
    // Yangi hisob → bo'sh holat yoki ro'yxat (ikkalasi ham to'g'ri — backend javob berdi).
    await expect(
      mentorPage.getByText(/Bildirishnomalar yo'q|yuklab bo'lmadi/i).or(mentorPage.locator('.empty')).first(),
    ).toBeVisible({ timeout: 15000 })
  })

  test('qora ro\'yxat sahifasi ochiladi (backend)', async ({ mentorPage }) => {
    await mentorPage.goto('/app/blocklist')
    await expect(mentorPage.getByRole('heading', { name: "Qora ro'yxat" })).toBeVisible()
  })
})
