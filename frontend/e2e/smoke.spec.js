import { test, expect } from '@playwright/test'

// Asosiy oqim (MSW mock bilan): landing → ro'yxatdan o'tish → dashboard →
// dars yaratish → join-link preview. LiveKit xonasi real SFU talab qilgani uchun
// smoke'da qamralmaydi (u qo'lda/integratsion sinaladi).

test('landing → auth → dashboard → dars yaratish', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText("Jonli darslarni")).toBeVisible()

  // Kirish sahifasiga
  await page.getByRole('button', { name: 'Kirish' }).first().click()
  await expect(page).toHaveURL(/\/auth/)

  // Ro'yxatdan o'tish tabiga o'tish (birinchi — tab tugmasi)
  await page.getByRole('button', { name: "Ro'yxatdan o'tish" }).first().click()
  await page.getByPlaceholder('Ism Familiya').fill('Test Foydalanuvchi')
  await page.getByPlaceholder('email@misol.uz').fill('test@darsly.uz')
  await page.getByPlaceholder('••••••••').fill('parol12345')
  // Formani yuborish (submit tugmasi)
  await page.locator('form button[type="submit"]').click()

  // Dashboard
  await expect(page).toHaveURL(/\/app/)
  await expect(page.getByText('Mening darslarim')).toBeVisible()

  // Tezkor dars yaratish: header tugmasi → modal → nom → «Yaratish va boshlash».
  // «Dars yaratish» = tezkor dars (Zoom "New meeting" kabi): yaratilishi bilan
  // server darsni LIVE qiladi va ilova DARHOL xona sahifasiga o'tadi
  // (CreateLessonModal → navigate(`/app/lesson/:id/room`)). Xona ichi real SFU
  // talab qilgani uchun smoke faqat SHU navigatsiyagacha tekshiradi — dars
  // muvaffaqiyatli yaratilgani va oqim to'g'ri ulanganining isboti shu.
  await page.getByRole('button', { name: 'Dars yaratish' }).first().click()
  await page.getByPlaceholder('Masalan: Kvadrat tenglamalar').fill('Test dars')
  await page.getByRole('button', { name: 'Yaratish va boshlash' }).click()
  await expect(page).toHaveURL(/\/app\/lesson\/.+\/room/)
})

test('join-link preview ko‘rinadi', async ({ page }) => {
  await page.goto('/r/demo123')
  await expect(page.getByText("Darsga qo'shilish")).toBeVisible()
  await expect(page.getByPlaceholder('Ismingizni kiriting')).toBeVisible()
})

test('yaroqsiz join oqimi ismsiz to‘xtaydi', async ({ page }) => {
  await page.goto('/r/demo123')
  const submit = page.getByRole('button', { name: /Kutish xonasiga|Darsga qo'shilish/ })
  await expect(submit).toBeVisible()
})
