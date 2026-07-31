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
    mute_on_entry: true,
    allow_self_unmute: true,
    status: 'scheduled',
    created_at: now(),
    updated_at: now(),
  },
]

// Mentorning doimiy qora ro'yxati (№4) — BlocklistEntry shakli.
let blocklist = [
  { id: 'b1', identity: 'guest_1', display_name: 'Bezori Bola', created_at: now() },
]

// So'rovnomalar — natija ko'rinuvchanligi ikki rejimda (kontrakt №7).
let polls = [
  {
    id: 'p1',
    lesson_id: 'l1',
    question: 'Mavzu tushunarli bo‘ldimi?',
    options: ['Ha', 'Qisman', "Yo'q"],
    is_active: true,
    created_at: new Date(Date.now() - 120_000).toISOString(),
    results_visibility: 'public',
    results_published_at: null,
  },
]

const pollResultsOf = (p) => {
  const counts = p.options.map((_, i) => (i === 0 ? 7 : i === 1 ? 3 : 1))
  return { poll: p, counts, total: counts.reduce((a, b) => a + b, 0) }
}

// Server tomonda ham qulflangan to'plam (`Controls.jsx: REACTIONS` bilan AYNAN bir xil).
const REACTION_SET = ['👍', '👏', '❤️', '😂', '😮', '🎉', '✋']

const DAY = 86_400_000
// Yozuvlar: muddati yaqin (2 kun), normal (25 kun) va o'chirilgan.
const recordings = [
  {
    id: 'r1',
    lesson_id: 'l1',
    egress_id: 'eg1',
    status: 'ready',
    duration_sec: 3600,
    size_bytes: 524_288_000,
    started_at: new Date(Date.now() - 28 * DAY).toISOString(),
    ended_at: new Date(Date.now() - 28 * DAY + 3_600_000).toISOString(),
    created_at: new Date(Date.now() - 28 * DAY).toISOString(),
    expires_at: new Date(Date.now() + 2 * DAY).toISOString(),
  },
  {
    id: 'r2',
    lesson_id: 'l1',
    egress_id: 'eg2',
    status: 'expired',
    duration_sec: 5400,
    size_bytes: 734_003_200,
    started_at: new Date(Date.now() - 40 * DAY).toISOString(),
    ended_at: new Date(Date.now() - 40 * DAY + 5_400_000).toISOString(),
    created_at: new Date(Date.now() - 40 * DAY).toISOString(),
  },
]

const B = '/api/v1'
const ok = (data, status = 200) => HttpResponse.json({ data }, { status })

// E2E boshqaruvi: `?e2e_admit=1` bo'lsa kutish xonasi bir necha so'rovdan keyin
// "admitted" ga o'tadi. Bu mock — real backendda holat mentor tasdig'i bilan
// o'zgaradi; bu yerda maqsad KLIENT zanjirini (polling → token → xonaga o'tish)
// brauzerda tekshirish.
const waitingAdmit =
  typeof window !== 'undefined' && new URLSearchParams(window.location.search).get('e2e_admit') === '1'
let waitingPolls = 0

// Admin foydalanuvchilar ro'yxati — backend `UserShort` shakli.
let users = [
  { id: 'u1', full_name: 'Aziz Karimov', email: 'mentor@darsly.uz', color: '#6c5ce7' },
  { id: 'u2', full_name: 'Malika Yusupova', email: 'malika@darsly.uz', color: '#19d3a2' },
]

export const handlers = [
  // Ochiq ilova konfiguratsiyasi — web bu yerdan `allow_open_registration`ni oladi.
  http.get(`${B}/app-config`, () =>
    ok({
      android: { min_version: '1.0.0', latest_version: '1.0.0', apk_url: '', force_update: false, release_notes: '' },
      allow_open_registration: true,
    }),
  ),

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
      // Server default'lari (berilmasa true) — CreateLessonReq bilan mos.
      mute_on_entry: body.mute_on_entry ?? true,
      allow_self_unmute: body.allow_self_unmute ?? true,
      ...body,
    }
    lessons = [l, ...lessons]
    return ok(l, 201)
  }),
  http.get(`${B}/lessons/:id`, ({ params }) => {
    const l = lessons.find((x) => x.id === params.id)
    return l ? ok(l) : HttpResponse.json({ code: 'NOT_FOUND', message: 'topilmadi' }, { status: 404 })
  }),
  http.patch(`${B}/lessons/:id`, async ({ params, request }) => {
    const l = lessons.find((x) => x.id === params.id)
    if (!l) return HttpResponse.json({ code: 'NOT_FOUND', message: 'topilmadi' }, { status: 404 })
    Object.assign(l, await request.json(), { updated_at: now() })
    return ok(l)
  }),

  // Joinlink preview. Maxsus slug'lar (klient oqimlarini brauzerda sinash uchun):
  //   locked → parolli · sched → hali boshlanmagan · ended → yakunlangan
  http.get(`${B}/joinlink/:slug`, ({ params }) =>
    ok({
      id: 'l1',
      title: 'Kvadrat tenglamalar — guruh darsi',
      mentor_name: 'Aziz Karimov',
      scheduled_at: now(),
      status: params.slug === 'sched' ? 'scheduled' : params.slug === 'ended' ? 'ended' : 'live',
      has_passcode: params.slug === 'locked',
      is_waiting_room_enabled: true,
    }),
  ),
  http.post(`${B}/joinlink/:slug`, async ({ params, request }) => {
    // Yakunlangan dars (№3): token ham, request ham YO'Q — faqat holat.
    if (params.slug === 'ended') {
      return ok({
        lesson: { id: 'l1', title: 'Kvadrat tenglamalar', mentor_name: 'Aziz Karimov', status: 'ended', has_passcode: false, is_waiting_room_enabled: true },
        next_step: 'lesson_ended',
      })
    }
    // Parol bilan himoyalangan dars: noto'g'ri parol → 401 (real backend kabi).
    if (params.slug === 'locked') {
      const body = await request.json().catch(() => ({}))
      if (body.passcode !== '1234') {
        return HttpResponse.json({ code: 'UNAUTHORIZED', message: 'invalid passcode' }, { status: 401 })
      }
    }
    return ok({
      lesson: { id: 'l1', title: 'Kvadrat tenglamalar', mentor_name: 'Aziz Karimov', status: 'live', has_passcode: false, is_waiting_room_enabled: true },
      next_step: 'waiting_room',
      request_id: 'req1',
    })
  }),

  // Host moderatsiyasi (Zoom ovoz nazorati, №11) — hammasi 204.
  http.post(`${B}/lessons/:id/mute-all`, () => new HttpResponse(null, { status: 204 })),
  http.post(`${B}/lessons/:id/participants/:identity/mute`, () => new HttpResponse(null, { status: 204 })),
  http.post(`${B}/lessons/:id/participants/:identity/remove`, () => new HttpResponse(null, { status: 204 })),
  http.post(`${B}/lessons/:id/participants/:identity/allow-speak`, () => new HttpResponse(null, { status: 204 })),
  http.post(`${B}/lessons/:id/participants/:identity/revoke-speak`, () => new HttpResponse(null, { status: 204 })),

  // Qora ro'yxat (№4).
  http.get(`${B}/blocklist`, () => ok(blocklist)),
  http.delete(`${B}/blocklist/:id`, ({ params }) => {
    blocklist = blocklist.filter((e) => e.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),
  // Kutish xonasi holati — E2E uchun BOSHQARILADIGAN.
  //
  // Avval har doim `pending` qaytardi, ya'ni "admit → jonli xona" zanjirini
  // brauzerda umuman sinab bo'lmasdi (auditdagi M19 bo'shlig'ining sababi).
  // Endi test `?e2e_admit=1` bilan sahifani ochib holatni o'zgartira oladi:
  // birinchi so'rov `pending`, keyingilari `admitted` + room-token.
  http.get(`${B}/waitingroom/:id/status`, () => {
    if (!waitingAdmit) return ok({ request_id: 'req1', status: 'pending' })
    waitingPolls += 1
    if (waitingPolls < 2) return ok({ request_id: 'req1', status: 'pending' })
    return ok({
      request_id: 'req1',
      status: 'admitted',
      room: {
        token: 'mock-room-token',
        ws_url: 'ws://localhost:7880',
        identity: 'guest_e2e',
        role: 'participant',
        lesson_id: 'l1',
      },
    })
  }),

  // Xona holati (o'quvchi yozuv indikatorini shu yerdan biladi).
  http.get(`${B}/rooms/:lessonID/state`, () => ok({ hands: [], recording: true })),
  http.get(`${B}/rooms/:lessonID/chat`, () => HttpResponse.json({ data: [] })),

  // Dars chat tarixi (JWT yo'li) — dars tugagach ham o'qiladi.
  http.get(`${B}/lessons/:id/chat`, ({ params }) =>
    HttpResponse.json({
      data:
        params.id === 'l1'
          ? [
              {
                id: 'c3',
                lesson_id: 'l1',
                sender_identity: 'host',
                sender_name: 'Aziz Karimov',
                body: 'Uy ishi shu faylda',
                to_identity: null,
                created_at: now(),
                file: {
                  name: 'uy_ishi.pdf',
                  size: 184_320,
                  mime: 'application/pdf',
                  url: 'blob:mock/uy_ishi.pdf',
                  expires_in_s: 3600,
                },
              },
              {
                id: 'c2',
                lesson_id: 'l1',
                sender_identity: 'guest_1',
                sender_name: 'Ali Valiyev',
                body: 'Savolim bor edi',
                to_identity: null,
                created_at: now(),
              },
              {
                id: 'c1',
                lesson_id: 'l1',
                sender_identity: 'host',
                sender_name: 'Aziz Karimov',
                body: 'Darsga xush kelibsiz!',
                to_identity: null,
                created_at: new Date(Date.now() - 60_000).toISOString(),
              },
            ]
          : [],
    }),
  ),

  // ── Chat moderatsiyasi va fayl ulashish ──────────────────────────────────
  // Xabar tarixdan BUTUNLAY o'chadi (qabrtosh yo'q); jonli xonadagilarga
  // server `chat_deleted` data-xabarini yuboradi.
  http.delete(`${B}/lessons/:id/chat/:messageID`, () => new HttpResponse(null, { status: 204 })),

  // Fayl yuklash (ikkala yo'l ham bir xil `ChatMessage` qaytaradi).
  // Server cheklovlari mock'da ham qo'llanadi: 20 MB va turlar allowlist'i —
  // aks holda mock ustida yozilgan UI real backendda 400 bilan yiqilardi.
  ...['lessons/:id', 'rooms/:lessonID'].map((p) =>
    http.post(`${B}/${p}/chat/upload`, async ({ request }) => {
      const form = await request.formData()
      const file = form.get('file')
      if (!file || typeof file === 'string') {
        return HttpResponse.json({ code: 'BAD_REQUEST', message: 'file is required' }, { status: 400 })
      }
      if (file.size > 20 * 1024 * 1024) {
        return HttpResponse.json({ code: 'BAD_REQUEST', message: 'file is too large' }, { status: 400 })
      }
      const ext = file.name.split('.').pop()?.toLowerCase() || ''
      const allowed = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'pdf', 'docx', 'xlsx', 'pptx', 'doc', 'xls', 'ppt', 'txt', 'csv']
      if (!allowed.includes(ext)) {
        return HttpResponse.json({ code: 'BAD_REQUEST', message: 'unsupported file type' }, { status: 400 })
      }
      const mime = ext === 'pdf' ? 'application/pdf' : ['jpg', 'jpeg'].includes(ext) ? 'image/jpeg' : `application/${ext}`
      return ok(
        {
          id: 'cf' + Date.now(),
          lesson_id: 'l1',
          sender_identity: 'host',
          sender_name: 'Aziz Karimov',
          body: form.get('body') || '',
          to_identity: form.get('to') || null,
          created_at: now(),
          // `url` presigned va 1 soatlik — bazada saqlanmaydi.
          file: { name: file.name, size: file.size, mime, url: `blob:mock/${file.name}`, expires_in_s: 3600 },
        },
        201,
      )
    }),
  ),

  // ── So'rovnomalar ────────────────────────────────────────────────────────
  http.get(`${B}/lessons/:id/polls`, () => ok(polls)),
  http.post(`${B}/lessons/:id/polls`, async ({ request }) => {
    const body = await request.json()
    const p = {
      id: 'p' + (polls.length + 1),
      lesson_id: 'l1',
      question: body.question,
      options: body.options,
      is_active: true,
      created_at: now(),
      // Yuborilmasa YOPIQ tomon (server default'i ham shunday).
      results_visibility: body.results_visibility || 'mentor_only',
      results_published_at: null,
    }
    polls = [p, ...polls]
    return ok(p, 201)
  }),
  http.post(`${B}/lessons/:id/polls/:pollID/publish`, ({ params }) => {
    const p = polls.find((x) => x.id === params.pollID)
    if (!p) return HttpResponse.json({ code: 'NOT_FOUND', message: 'topilmadi' }, { status: 404 })
    // mentor_only'da e'lon qilib bo'lmaydi — real backend kabi 400.
    if (p.results_visibility !== 'public') {
      return HttpResponse.json(
        { code: 'BAD_REQUEST', message: 'poll results visibility is mentor_only' },
        { status: 400 },
      )
    }
    p.results_published_at = p.results_published_at || now() // idempotent
    return ok(pollResultsOf(p))
  }),
  http.post(`${B}/polls/:id/close`, ({ params }) => {
    const p = polls.find((x) => x.id === params.id)
    if (!p) return HttpResponse.json({ code: 'NOT_FOUND', message: 'topilmadi' }, { status: 404 })
    p.is_active = false
    p.closed_at = now()
    return ok(pollResultsOf(p))
  }),
  http.post(`${B}/polls/:id/vote`, () => new HttpResponse(null, { status: 204 })),
  // Yopiq natija 403 — bo'sh natija EMAS («0 ovoz» bilan «ko'rsatilmaydi»ni
  // farqlab bo'lmasa klient noto'g'ri diagramma chizardi).
  http.get(`${B}/polls/:id/results`, ({ params }) => {
    const p = polls.find((x) => x.id === params.id)
    if (!p) return HttpResponse.json({ code: 'NOT_FOUND', message: 'topilmadi' }, { status: 404 })
    if (p.results_visibility !== 'public' || !p.results_published_at) {
      return HttpResponse.json(
        { code: 'FORBIDDEN', message: 'poll results are not published yet' },
        { status: 403 },
      )
    }
    return ok(pollResultsOf(p))
  }),

  // Reaksiya — server tarqatadi, hech nima saqlamaydi.
  http.post(`${B}/rooms/:lessonID/reaction`, async ({ request }) => {
    const body = await request.json().catch(() => ({}))
    if (!REACTION_SET.includes(body.emoji)) {
      return HttpResponse.json({ code: 'BAD_REQUEST', message: 'unsupported reaction' }, { status: 400 })
    }
    return new HttpResponse(null, { status: 204 })
  }),
  http.post(`${B}/rooms/:lessonID/hand`, () => new HttpResponse(null, { status: 204 })),

  // Yozuvlar — retention (30 kun) namunalari: yaqinda o'chadigan va o'chgan.
  http.get(`${B}/lessons/:id/recordings`, () => ok(recordings)),
  http.get(`${B}/recordings/:id/download`, ({ params }) => {
    const r = recordings.find((x) => x.id === params.id)
    if (r && r.status === 'expired') {
      return HttpResponse.json(
        { code: 'BAD_REQUEST', message: 'recording has expired and was deleted' },
        { status: 400 },
      )
    }
    return ok({ url: 'blob:mock/recording.mp4', expires_in_s: 3600, duration_sec: 3600, size_bytes: 524288000 })
  }),

  http.get(`${B}/notifications`, () => HttpResponse.json({ data: [], total: 0, page: 1, limit: 50, total_pages: 0 })),
  http.get(`${B}/notifications/unread-count`, () => ok({ count: 0 })),

  // Admin users CRUD (backend: GET/POST /users, DELETE /users/:id — admin-only RBAC).
  http.get(`${B}/users`, () =>
    HttpResponse.json({ data: users, total: users.length, page: 1, limit: 100, total_pages: 1 }),
  ),
  http.post(`${B}/users`, async ({ request }) => {
    const body = await request.json()
    const u = {
      id: 'u' + (users.length + 1),
      full_name: body.full_name,
      email: body.email,
      color: '#19d3a2',
      role: body.role || 'mentor',
      is_active: true,
      created_at: now(),
      updated_at: now(),
    }
    users = [...users, { id: u.id, full_name: u.full_name, email: u.email, color: u.color }]
    return ok(u, 201)
  }),
  http.delete(`${B}/users/:id`, ({ params }) => {
    users = users.filter((u) => u.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),
]
