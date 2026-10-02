// Full-flow: Admin panel — foydalanuvchilar CRUD (UI → backend → DB)
import { test, expect } from './fixtures.js'

test.describe('Admin — foydalanuvchilar (live)', () => {
  test('admin foydalanuvchilar sahifasini ko\'radi', async ({ adminPage }) => {
    await adminPage.goto('/app/users')
    await expect(adminPage.getByRole('heading', { name: 'Foydalanuvchilar' })).toBeVisible()
  })

  test('yangi foydalanuvchi yaratish → ro\'yxatda (DB) → o\'chirish', async ({ adminPage }) => {
    const sfx = Date.now().toString(36)
    const name = `QA User ${sfx}`
    const email = `qauser+${sfx}@darsly.uz`

    await adminPage.goto('/app/users')
    await adminPage.getByRole('button', { name: /Foydalanuvchi qo'shish/i }).click()

    // CreateUserModal.
    await adminPage.getByPlaceholder('Ism Familiya').fill(name)
    await adminPage.getByPlaceholder('email@misol.uz').fill(email)
    await adminPage.getByPlaceholder('Kamida 8 belgi').fill('parol12345')
    await adminPage.getByLabel('Rol').selectOption('mentor')
    await adminPage.locator('button[type="submit"]').click()

    // POST /users → jadvalда (DB) ko'rinadi.
    await expect(adminPage.getByText(email)).toBeVisible({ timeout: 15000 })

    // O'chirish (window.confirm) → yo'qoladi.
    adminPage.on('dialog', (d) => d.accept())
    await adminPage.getByRole('button', { name: `${name}ni o'chirish` }).click()
    await expect(adminPage.getByText(email)).toHaveCount(0, { timeout: 15000 })
  })

  test('mentor admin sahifasiga kira olmaydi (RBAC)', async ({ mentorPage }) => {
    await mentorPage.goto('/app/users')
    // Mentor admin emas → sahifa ko'rsatilmaydi (redirect yoki ruxsat yo'q).
    await expect(mentorPage.getByRole('heading', { name: 'Foydalanuvchilar' })).toHaveCount(0, { timeout: 8000 })
  })
})
