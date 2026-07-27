import { http, HttpResponse } from 'msw'

// MSW mock handler'lari — backend tayyor bo'lmasa frontend'ni alohida ishlatish/test uchun.
// Faqat VITE_USE_MOCK=true bo'lganda ishga tushadi (main.jsx). Real xatti-harakatga yaqin.

const now = () => new Date().toISOString()
const TOKENS = { access_token: 'mock-access', refresh_token: 'mock-refresh' }

const USER = {
  id: 'u1',
  email: 'mentor@darsly.uz',
  full_name: 'Aziz Karimov',
  color: '#6c5ce7',
  role: 'mentor',
  timezone: 'Asia/Tashkent',
  language: 'uz',
  is_active: true,
  created_at: now(),
  updated_at: now(),
}

let lessons = [
  {
    id: 'l1',
    mentor_id: 'u1',
    title: 'Kvadrat tenglamalar — guruh darsi',
    description: 'Kirish darsi',
    scheduled_at: now(),
    duration_min: 60,
    join_slug: 'demo123',
    has_passcode: false,
    is_locked: false,
    is_recording_enabled: true,
    is_waiting_room_enabled: true,
    status: 'scheduled',
    created_at: now(),
    updated_at: now(),
  },
]

const B = '/api/v1'
const ok = (data, status = 200) => HttpResponse.json({ data }, { status })

export const handlers = [
  http.post(`${B}/auth/register`, () => ok(TOKENS, 201)),
  http.post(`${B}/auth/login`, () => ok(TOKENS)),
  http.post(`${B}/auth/refresh`, () => ok(TOKENS)),
  http.post(`${B}/auth/logout`, () => new HttpResponse(null, { status: 204 })),
  http.get(`${B}/auth/me`, () => ok(USER)),

  http.get(`${B}/lessons`, () =>
    HttpResponse.json({ data: lessons, total: lessons.length, page: 1, limit: 50, total_pages: 1 }),
  ),
  http.post(`${B}/lessons`, async ({ request }) => {
    const body = await request.json()
    const l = {
      id: 'l' + (lessons.length + 1),
      mentor_id: 'u1',
      join_slug: 'slug' + (lessons.length + 1),
      has_passcode: !!body.passcode,
      is_locked: false,
      status: 'scheduled',
      created_at: now(),
      updated_at: now(),
      duration_min: body.duration_min || 60,
      ...body,
    }
    lessons = [l, ...lessons]
    return ok(l, 201)
  }),
  http.get(`${B}/lessons/:id`, ({ params }) => {
    const l = lessons.find((x) => x.id === params.id)
    return l ? ok(l) : HttpResponse.json({ code: 'NOT_FOUND', message: 'topilmadi' }, { status: 404 })
  }),

  http.get(`${B}/joinlink/:slug`, ({ params }) =>
    ok({
      id: 'l1',
      title: 'Kvadrat tenglamalar — guruh darsi',
      mentor_name: 'Aziz Karimov',
      status: 'live',
      has_passcode: params.slug === 'locked',
      is_waiting_room_enabled: true,
    }),
  ),
  http.post(`${B}/joinlink/:slug`, () =>
    ok({
      lesson: { id: 'l1', title: 'Kvadrat tenglamalar', mentor_name: 'Aziz Karimov', status: 'live', has_passcode: false, is_waiting_room_enabled: true },
      next_step: 'waiting_room',
      request_id: 'req1',
    }),
  ),
  http.get(`${B}/waitingroom/:id/status`, () => ok({ request_id: 'req1', status: 'pending' })),

  http.get(`${B}/notifications`, () => HttpResponse.json({ data: [], total: 0, page: 1, limit: 50, total_pages: 0 })),
  http.get(`${B}/notifications/unread-count`, () => ok({ count: 0 })),
]
