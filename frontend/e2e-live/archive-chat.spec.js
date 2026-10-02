// Full-flow: Dars arxivi + chat tarixi modal (UI → backend → DB)
import { test, expect } from './fixtures.js'

test.describe('Arxiv / chat tarixi (live)', () => {
  test('chat tarixi modal ochiladi (backend tarixi)', async ({ mentorPageWithLesson }) => {
    const { page, lesson } = mentorPageWithLesson
    await page.goto('/app')
    await expect(page.getByText(lesson.title)).toBeVisible({ timeout: 15000 })
    await page.getByRole('button', { name: 'Chat tarixi' }).first().click()
    // ChatHistoryModal — title "Dars chati" (GET /lessons/:id/chat).
    await expect(page.getByRole('heading', { name: 'Dars chati' })).toBeVisible({ timeout: 15000 })
  })

  test('dars arxivi sahifasi ochiladi (video+chat+material, DB)', async ({ mentorPageWithLesson }) => {
    const { page, lesson } = mentorPageWithLesson
    await page.goto(`/app/lesson/${lesson.id}/archive`)
    // GET /lessons/:id/archive → sahifa yuklanadi (dars nomi h1'да).
    await expect(page.getByRole('heading', { name: new RegExp(`${lesson.title}|Dars arxivi`) })).toBeVisible({
      timeout: 15000,
    })
  })
})
