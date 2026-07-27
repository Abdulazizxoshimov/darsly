# Darsly Frontend — Arxitektura talablari (manba spetsifikatsiya)

> Foydalanuvchi bergan rasmiy spetsifikatsiya. `frontend-agent` bunga **qat'iy** amal qiladi.

## Stack
- **React 18 + Vite + TanStack Query v5**
- **Vanilla CSS** (Tailwind YO'Q) — barcha ranglar/o'lchamlar `styles.css` da CSS custom property (`:root { --bg: … }`) sifatida, hardcode qilinmaydi.
- `react-router-dom` **v7**
- `livekit-client` (jonli xona)
- Fayllar **`.jsx`** (JavaScript, TypeScript emas)

## Papka strukturasi (qat'iy)
```
src/
  api/        # api.jsx (fetch client) + adapters (backend response → UI model)
  views/      # sahifa darajasidagi komponentlar
  panels/     # katta side/detail panellar (chat, participants, polls)
  components/ # qayta ishlatiluvchi UI (Button, Input, Card, Avatar, Modal, Toast…)
  store/      # app.jsx (AppContext) + data.jsx (TanStack Query hook'lari)
  lib/        # ws.js (WebSocket client) va util'lar
  test/       # MSW mock (server + handlers)
  styles.css  # CSS custom property'lar + barcha uslublar
e2e/          # Playwright smoke test
```

## Dizayn tizimi (CSS custom property sifatida)
- Fon: `--bg: #0a0e14`; karta: `--card: #12171f`; ko'tarilgan: `#161c26`
- Chegara: `--border: #232b37`, `--border-strong: #262f3b`
- Accent gradient: `--accent: #6c5ce7` → `--accent-light: #8b7cf6`, hover `#7d6ff2`
- Matn: `--text: #eef1f6`, `--text-2: #97a3b3`, `--text-3: #6b7688`
- Holat: xato `--danger: #ef4444`, muvaffaqiyat `--success: #34d399`, ogohlantirish `--warning: #f59e0b`
- Shrift: **Manrope**. Radius: kartalar 14–16px, tugma/input 8–9px.

## `src/api/api.jsx` — asosiy client
- `fetch` wrapper, base URL `.env` dan (`VITE_API_URL`).
- Har so'rovga JWT avtomatik (`Authorization: Bearer`).
- **401 → avtomatik logout + login sahifasiga**; refresh token bilan bir marta yangilashga urinish.
- Backend `http_status`/`errors.go` formatidagi xatoni parse qiladi: `{ "code": "...", "message": "..." }`. Response konverti: `{ "data": … }`, ro'yxat: `{ data, total, page, limit, total_pages }`.
- Xato kodini o'zbekcha matnga map qiladi.

## `src/store/data.jsx` — TanStack Query hook'lari (domen bo'yicha)
`useLessons()`, `useCreateLesson()`, `useLesson(id)`, `useJoinLink(slug)`, `useSubmitJoin()`, `useLiveKitToken(lessonId)`, `useWaitingRoomRequests(lessonId)`, `useAdmitRequest()`, `useRecordings(lessonId)`, `useNotifications()`, `useProfile()` … — har birida `isLoading/isError` UI'da to'g'ri ko'rsatiladi (skeleton), mutation'dan keyin `invalidateQueries`.

## `src/lib/ws.js` — WebSocket
- `/api/v1/ws?token=<jwt>` (mentor authed) va `/api/v1/ws/waitingroom?request_id=` (guest).
- Avtomatik qayta ulanish (exponential backoff).
- Event turlari (backend konverti `{type, room?, payload, created_at}`): `notification`, `waiting_room.request`, `waiting_room.admitted`, `waiting_room.rejected`.
- Event'larni `AppContext` orqali global holatga tarqatadi.

## `src/views/liveroom.jsx` — LiveKit (eng muhim)
- Backend'dan olingan token bilan ulanadi (`{ token, ws_url, room_name, identity, role }`).
- Ulanish holati: `connecting → connected → reconnecting → disconnected` — har biri uchun mos UI (dizayndagi "Ulanish sifati" indikatori).
- Video grid LiveKit `RemoteParticipant`/`LocalParticipant` track'lariga dinamik bog'lanadi (gallery + speaker view).
- Mic/camera/screenshare/hand-raise/reactions — LiveKit SDK metodlariga to'g'ridan-to'g'ri; alohida state boshqarilmaydi.
- Internet uzilsa LiveKit avto-reconnect → "Qayta ulanmoqda…" holati.

## Sifat talablari
- Har API chaqiruvi `try/catch`, xato o'zbekcha tushunarli.
- Production kodda `console.log` yo'q.
- MSW mock (`src/test/`) — backend'siz ham frontend ishlab chiqish/test.
- `e2e/smoke.spec.js` — Playwright: login → dars yaratish → link orqali kirish → live room.
- Har ekran uchun loading/error/empty holatlar.

## Backend mustaqilligi
Real backend ulanganda faqat `api.jsx` base URL + response format moslashtiriladi — boshqa hech narsa o'zgarmaydi.
