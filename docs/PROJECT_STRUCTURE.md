# Darsly — Loyiha strukturasi

## 1. Umumiy
```
zoom/
├── backend/    # Go + Gin Clean Architecture (module github.com/zoom/darsly) — QURILGAN
├── frontend/   # React + Vite (vanilla CSS, .jsx) — QURILMOQDA
├── services/livekit/  # LiveKit SFU + Egress + coturn
├── deployments/       # Docker Compose, nginx/Caddy, CI
└── docs/              # rejalar, kontrakt, acceptance criteria
```

## 2. Backend (mavjud — `backend/CLAUDE.md` to'liq bayon qiladi)
Qatlam: `Handler → Usecase → Repository interface → Postgres impl`.
Yangi domen tartibi: entity → repository interface → postgres impl → storage.go → usecase → usecase.go → handler → wire.go → handler.go → router.go → casbin policy → migration.
Domenlar: auth, user, lesson, joinlink, room, waitingroom, recording, notification, chat, poll.

## 3. Frontend — React + Vite
```
src/
  api/        # api.jsx (fetch client), <domen>.jsx (endpoint funksiyalari), adapters
  views/      # sahifa komponentlari (Landing, Auth, Dashboard, Schedule, Join,
              #   WaitingRoom, LiveRoom, Recordings, Notifications, Profile)
  panels/     # katta panellar (ChatPanel, ParticipantsPanel, PollsPanel)
  components/ # Button, Input, Card, Avatar, Modal, Toast, Spinner, Badge, StatusBadge
  store/      # app.jsx (AppContext: auth+realtime), data.jsx (TanStack Query hook'lari)
  lib/        # ws.js (WebSocket + reconnect), livekit.js (SDK yordamchilari), format.js
  test/       # MSW: server.js + handlers.js
  styles.css  # CSS custom property'lar + barcha uslublar
  main.jsx, App.jsx
e2e/          # Playwright smoke.spec.js
```
Batafsil talablar: `docs/darsly-frontend-prompt.md`. API: `docs/api-contract.md`. Dizayn: `Darsly Platform Design System(1)/`.

## 4. Portlar (dev)
| Servis | Port |
|---|---|
| Frontend (Vite) | 3000 |
| Backend API | 8087 |
| PostgreSQL | 5442 |
| Redis | 6399 |
| MinIO | 9020/9021 |
| RabbitMQ | 5682/15682 |
| LiveKit | 7880/7881 |

Frontend `.env`: `VITE_API_URL` (dev'da bo'sh → Vite proxy `/api`,`/ws` → :8087).
