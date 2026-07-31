# Jonly — Frontend

**React 18 + Vite + TanStack Query v5 + vanilla CSS + `.jsx`** (Tailwind/TS YO'Q).
Backend (`github.com/zoom/darsly`) bilan ishlaydi. Dizayn: `../Darsly Platform Design System(1)/`.
Talablar: `../docs/darsly-frontend-prompt.md`. API: `../docs/api-contract.md`.

## Ishga tushirish

```bash
npm install
npm run dev        # http://localhost:3000  (/api,/ws → backend :8087 proxy)
```

Backend kerak: `cd ../backend && make docker-up && make run` (:8087).
Jonli xona uchun LiveKit: `cd ../services/livekit && docker compose up -d` (:7880).

**Backend'siz ishlash (MSW mock):**
```bash
VITE_USE_MOCK=true npm run dev
```

Build: `npm run build` → `dist/`. E2E: `npm run test:e2e` (Playwright, MSW mock).

## Struktura

```
src/
  api/        api.jsx (fetch client: JWT, 401→refresh, o'zbekcha xato) + <domen>.jsx
  store/      app.jsx (AppContext: auth + realtime), data.jsx (TanStack Query hook'lari)
  lib/        ws.js (WebSocket reconnect), toast.js, format.js, roomSession.js, queryClient.js
  components/ Button, Field, Card, Avatar, Badge, Modal, Spinner, Toaster, RequireAuth, AppShell
  views/      Landing, Auth, ForgotPassword, ResetPassword, Dashboard, Schedule, Recordings,
              Notifications, Profile, Join, WaitingRoom, LiveRoom, NotFound
  panels/     ChatPanel, ParticipantsPanel, PollsPanel
  livekit/    useRoom (livekit-client ulanish), Stage, ParticipantTile, Controls,
              ReactionsOverlay, messaging (data-channel)
  test/       MSW (handlers.js, browser.js)
  styles.css  CSS custom property'lar (:root --bg/--accent/…) + barcha uslublar
e2e/          Playwright smoke.spec.js
```

## Xususiyatlar

- **Auth**: register/login/refresh (avto), forgot/reset-password, guard, 401→login.
- **Dars (mentor)**: CRUD, jadval, havola nusxalash.
- **Join (guest)**: preview, parol, kutish xonasi (WS push + polling fallback).
- **Jonli xona (LiveKit)**: gallery/speaker view, ekran ulashish, host-panel (mute/mute-all/kick/allow-speak),
  chat (host persist + guest data-channel), reaksiya, qo'l-ko'tarish, real-time so'rovnoma,
  yozib olish, reconnect indikatori. adaptiveStream+dynacast+simulcast (past-internet).
- **Yozuvlar/bildirishnoma/profil**: to'liq.
- Har view'da loading/error/empty; xatolar o'zbekcha; production'da `console.log` yo'q.

## Holat

Build toza, Playwright smoke 3/3, API kontrakt backend bilan mos. Jonli video-qo'ng'iroq
tirik LiveKit + brauzer bilan sinalishi kerak (inson testi).
