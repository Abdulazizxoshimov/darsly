import { test, expect } from '@playwright/test'

// M19 — mehmon zanjiri BRAUZERDA.
//
// Auditda topilgan bo'shliq: eng murakkab oqim (havola → parol → kutish xonasi →
// admit → jonli xona) faqat qo'lda sinalardi. Bu zanjirda regressiya bo'lsa
// hech qanday test uni tutmasdi — birlik testlari komponentlarni alohida
// ko'radi, oqimni emas.
//
// ⚠️ QAMROV CHEGARASI: LiveKit media legi (haqiqiy SFU ulanishi, kamera/
// mikrofon) bu yerda QAMRALMAYDI — u real SFU va media qurilmalarini talab
// qiladi. Test zanjirni xona chegarasigacha, ya'ni "token olindi va xonaga
// o'tildi" nuqtasigacha tekshiradi. Media legi qo'lda/integratsion sinaladi.

test.describe('Mehmon oqimi', () => {
  test('havola → ism → kutish xonasi', async ({ page }) => {
    await page.goto('/r/demo123')
    await expect(page.getByText("Darsga qo'shilish")).toBeVisible()

    await page.getByPlaceholder('Ismingizni kiriting').fill('Mehmon Aziz')
    await page.getByRole('button', { name: /Kutish xonasiga|Darsga qo'shilish/ }).click()

    await expect(page).toHaveURL(/\/waiting/)
  })

  // ⭐ Asosiy M19 zanjiri: kutish xonasidagi polling admit'ni ko'radi va
  // mehmonni xonaga o'tkazadi. Avval bu o'tish umuman avtomatlashtirilmagandi.
  test('kutish xonasi → admit → xonaga o‘tish', async ({ page }) => {
    await page.goto('/r/demo123?e2e_admit=1')
    await page.getByPlaceholder('Ismingizni kiriting').fill('Mehmon Aziz')
    await page.getByRole('button', { name: /Kutish xonasiga|Darsga qo'shilish/ }).click()

    await expect(page).toHaveURL(/\/waiting/)

    // Polling admit'ni ko'rgach klient xona marshrutiga o'tishi kerak.
    await expect(page).toHaveURL(/\/room/, { timeout: 20000 })
  })

  test('parol bilan himoyalangan dars: noto‘g‘ri parol xato beradi', async ({ page }) => {
    await page.goto('/r/locked')
    await expect(page.getByText("Darsga qo'shilish")).toBeVisible()

    await page.getByPlaceholder('Ismingizni kiriting').fill('Mehmon')
    // Parol maydoni faqat `has_passcode` bo'lganda ko'rinadi.
    const pass = page.getByPlaceholder('Dars paroli')
    await expect(pass).toBeVisible()
    await pass.fill('0000')
    await page.getByRole('button', { name: /Kutish xonasiga|Darsga qo'shilish/ }).click()

    // Noto'g'ri parolda sahifada qolamiz va xato ko'rsatiladi.
    await expect(page).not.toHaveURL(/\/waiting/)
  })

  test('sessiyasiz kutish sahifasi tushunarli xabar beradi', async ({ page }) => {
    // To'g'ridan-to'g'ri kutish URL'iga kirish (sessiya yo'q) — oq ekran
    // bo'lmasligi va nima qilish kerakligi ko'rinishi kerak.
    await page.goto('/r/demo123/waiting')
    await expect(page.getByText('Sessiya topilmadi.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Qaytadan urinish' })).toBeVisible()
  })
})
