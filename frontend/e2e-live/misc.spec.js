// Full-flow: Yozuvlar sahifasi · parol o'zgartirish (DB) · Telegram (o'chiq → yashirin)
import { test, expect, api } from './fixtures.js'

function field(page, label) {
  return page
    .locator('.field', { has: page.locator('label.field__label', { hasText: label }) })
    .locator('input, textarea')
    .first()
}

test.describe('Misc sahifalar (live)', () => {
  test('yozuvlar sahifasi ochiladi', async ({ mentorPage }) => {
    await mentorPage.goto('/app/recordings')
    await expect(mentorPage.getByRole('heading', { name: 'Yozuvlar' })).toBeVisible()
  })

  test("parol o'zgartirish → DB (yangi ishlaydi, eski 401)", async ({ mentorPage, mentorAccount }) => {
    const newPw = 'yangiparol12345'
    await mentorPage.goto('/app/profile')
    await field(mentorPage, 'Joriy parol').fill(mentorAccount.password)
    await field(mentorPage, 'Yangi parol').fill(newPw)
    await mentorPage.getByRole('button', { name: "O'zgartirish" }).click()

    // PUT /users/me/password → DB. Yangi parol bilan login ishlashini kutamiz (timing).
    await expect
      .poll(
        async () => {
          const r = await api.jsonFetch('/api/v1/auth/login', {
            method: 'POST',
            body: { email: mentorAccount.email, password: newPw },
          })
          return r.status
        },
        { timeout: 15000 },
      )
      .toBe(200)

    // Eski parol endi ishlamaydi.
    const old = await api.jsonFetch('/api/v1/auth/login', {
      method: 'POST',
      body: { email: mentorAccount.email, password: mentorAccount.password },
    })
    expect(old.status).toBe(401)
  })

  test('Telegram bo\'limi o\'chiq integratsiyada ko\'rsatilmaydi', async ({ mentorPage }) => {
    await mentorPage.goto('/app/profile')
    // Profil yuklandi (parol bo'limi bor), lekin Telegram bo'limi (enabled:false) YO'Q.
    await expect(mentorPage.getByRole('heading', { name: "Parolni o'zgartirish" })).toBeVisible()
    await expect(mentorPage.getByRole('heading', { name: /Telegram/i })).toHaveCount(0)
  })
})
