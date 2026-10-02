// Full-flow: Auth (brauzer → backend → DB)
import { test, expect } from './fixtures.js'

test.describe('Auth (live)', () => {
  test('login muvaffaqiyatli → /app dashboard', async ({ mentorPage }) => {
    // mentorPage fixture UI orqali kirdi va /app ga o'tdi.
    await expect(mentorPage).toHaveURL(/\/app/)
    await expect(mentorPage.getByText(/Dars/i).first()).toBeVisible()
  })

  test("noto'g'ri parol → /auth da qoladi (login o'tmaydi)", async ({ page, mentorAccount }) => {
    await page.goto('/auth')
    await page.getByPlaceholder('email@misol.uz').fill(mentorAccount.email)
    await page.getByPlaceholder('••••••••').fill('notrealpw999')
    await page.locator('button[type="submit"]').click()
    // Login o'tmaydi → hamon /auth, email maydoni ko'rinadi.
    await expect(page).toHaveURL(/\/auth/)
    await expect(page.getByPlaceholder('email@misol.uz')).toBeVisible()
  })

  test('logout → /auth ga qaytadi', async ({ mentorPage }) => {
    // AppShell'да ko'rinadigan "Chiqish" nav-item (desktop sidebar).
    const navLogout = mentorPage.locator('.nav__item', { hasText: 'Chiqish' })
    const target = (await navLogout.count()) ? navLogout.first() : mentorPage.getByLabel('Chiqish').first()
    await target.click()
    await expect(mentorPage).toHaveURL(/\/auth/, { timeout: 15000 })
    await expect(mentorPage.getByPlaceholder('email@misol.uz')).toBeVisible()
  })
})
