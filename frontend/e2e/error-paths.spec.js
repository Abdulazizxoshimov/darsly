import { test, expect } from '@playwright/test'

// Xato yo'llari BRAUZERDA (MSW mock bilan). Muvaffaqiyatli oqim smoke.spec.js
// va guest-flow.spec.js da; bu yerda foydalanuvchi UMIDSIZLIKKA tushadigan
// yo'llar: noto'g'ri parol va kutish xonasidan rad etilish.
//
// ⚠️ QAMROV CHEGARASI: xona ichidagi UI (yozib olish start/stop tugmasi,
// media boshqaruvi) SFU'ning haqiqiy ulanishini talab qiladi va e2e'da
// QAMRALMAYDI — u komponent testlarida (Controls.test.jsx) sinaladi.

test.describe('Kirish xatosi', () => {
  // Bug: noto'g'ri parolda umumiy "Kirish amalga oshmadi" ko'rsatilsa, foydalanuvchi
  // parolini emas, tizimni ayblaydi. `UNAUTHORIZED` → «Email yoki parol noto'g'ri».
  test('noto‘g‘ri parol — aniq xato ko‘rsatiladi, sahifada qolamiz', async ({ page }) => {
    await page.goto('/auth')
    await page.getByPlaceholder('email@misol.uz').fill('mentor@darsly.uz')
    // Mock `wrongpass` parolida 401 qaytaradi (test/handlers.js).
    await page.getByPlaceholder('••••••••').fill('wrongpass')
    await page.locator('form button[type="submit"]').click()

    await expect(page.getByText(/Email yoki parol noto‘g‘ri/i)).toBeVisible()
    // /app ga O'TMAYDI — kirish sahifasida qolamiz.
    await expect(page).toHaveURL(/\/auth/)
  })

  // Bug: to'g'ri parolda ham xato ko'rsatilib qolsa — regressiya. Sanity juftligi.
  test('to‘g‘ri parol — dashboard ochiladi', async ({ page }) => {
    await page.goto('/auth')
    await page.getByPlaceholder('email@misol.uz').fill('mentor@darsly.uz')
    await page.getByPlaceholder('••••••••').fill('parol12345')
    await page.locator('form button[type="submit"]').click()
    await expect(page).toHaveURL(/\/app/)
  })
})

test.describe('Kutish xonasi — rad etish', () => {
  // Bug: rad etilgan mehmon reject signalini ikkala yo'ldan (WS/polling) ham
  // ololmasa spinnerni cheksiz ko'radi va dars boshlanmayotgandek tuyuladi.
  // `?e2e_reject=1` mock polling'ni darhol `rejected` qildiradi.
  test('rad etilgan mehmon «Kirish rad etildi» ko‘radi', async ({ page }) => {
    await page.goto('/r/demo123?e2e_reject=1')
    await page.getByPlaceholder('Ismingizni kiriting').fill('Rad Etilgan Mehmon')
    await page.getByRole('button', { name: /Kutish xonasiga|Darsga qo'shilish/ }).click()

    await expect(page).toHaveURL(/\/waiting/)
    await expect(page.getByText('Kirish rad etildi')).toBeVisible({ timeout: 20000 })
    // Bosh sahifaga qaytish yo'li ko'rinadi (mehmon tiqilib qolmaydi).
    await expect(page.getByRole('button', { name: 'Bosh sahifa' })).toBeVisible()
  })
})
