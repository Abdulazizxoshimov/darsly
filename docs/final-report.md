# Darsly — Yakuniy hisobot (hand-off)

10-agentli orkestratsiya yakuni. Sana: 2026-07-24.

## Nima qurildi

**Backend** (`backend/`, Go + Gin Clean Architecture) — ALLAQACHON qurilgan edi; bu siklda **audit + xavfsizlik hardening**:
- 🔴 Parol reset/deactivate endi Redis sessiyalarni o'ldiradi (o'g'irlangan sessiya reset'dan keyin yashamaydi).
- IDOR yopildi (`GET /users/:id` — faqat self/mentor).
- CORS bo'sh-allowlist xavfi yopildi.
- Refresh-reuse → butun sessiya-oila bekor qilinadi (token-theft himoyasi).
- CSP qattiqlashtirildi (`unsafe-eval` olib tashlandi, `connect-src *` cheklandi).
- Rate-limit Redis o'chsa in-memory fallback (brute-force himoyasi saqlanadi).
- Admin parol-reset (`PUT /users/:id/password`) endi `:id`ni to'g'ri ishlatadi.
- `go build ./... && go vet ./... && go test ./...` — YASHIL.

**Frontend** (`frontend/`) — noldan qurildi: **React 18 + Vite + vanilla CSS + `.jsx` + TanStack Query v5 + react-router v7 + livekit-client**:
- Auth (register/login/refresh/forgot/reset), guard, 401→login.
- Mentor: dars CRUD, jadval, havola nusxalash. Guest: join-link (parol/kutish-xona).
- Jonli xona (LiveKit): gallery/speaker, ekran ulashish, host-panel (mute/mute-all/kick/allow-speak),
  chat (host persist + guest data-channel), reaksiya, qo'l-ko'tarish, real-time so'rovnoma,
  yozib olish, reconnect indikatori, adaptiveStream+dynacast+simulcast.
- Yozuvlar (presigned download), bildirishnoma (WS push), profil.
- MSW mock (backend'siz ishlash), Playwright smoke.

**Infra** (`devops`): docker-compose jiraflow/collab ifloslanishi tozalandi, portlar mos,
LiveKit prod compose (TURN/wss), CI (backend+frontend), `docs/DEPLOYMENT.md`, Makefile.

## Sifat darajasi (avtomatik tasdiqlangan)

| Tekshiruv | Natija |
|---|---|
| Backend `go build/vet/test` | ✅ yashil |
| Frontend `npm run build` | ✅ toza (85 kB gzip + LiveRoom lazy chunk) |
| API kontrakt (frontend↔backend, 40+ endpoint) | ✅ KONTRAKT TOZA |
| Xavfsizlik review (JWT/RBAC/token/SQL/CORS) | ✅ 3 tuzatish + 4 hardening |
| Playwright smoke (auth→dashboard→dars; join) | ✅ 3/3 |
| Har view loading/error/empty | ✅ |

## ⚠️ Cheklovlar — TIRIK inson testi kerak (release blokerlari)

Mahsulot yadrosi (video-qo'ng'iroq) **hech qachon tirik LiveKit SFU + brauzer**da ishlatilmagan —
bu muhitda (SFU/kamera yo'q) avtomatik tasdiqlab bo'lmaydi. Quyidagi 5 stsenariy inson tomonidan
yashil bo'lmaguncha acceptance-criteria 100% emas:

1. **2 brauzer**: mentor xona ochadi + guest join → video ikki tomon ko'rinadimi.
2. **Waiting-room ON**: guest so'rov → mentorga real-time chiqadimi → admit → guest avtomatik kiradimi.
3. **Xona ichida**: mute-all/kick/allow-speak, chat, reaksiya, poll, REC start/stop.
4. **Wi-Fi uzib-ulash** → auto-reconnect chalkashlik bermaydimi.
5. **Auth**: token muddati tugaganda silent refresh + 401→login redirect.

## Ma'lum, ataylab qoldirilgan

- **Fail-open Redis** (sessiya-tekshiruv + rate-limit degrade): ataylab availability tanlovi
  (access TTL 15m bilan cheklangan). Fail-closed'ga o'tkazish — mahsulot egasi qarori.
- **Dizaynda bor, backend endpoint yo'q**: whiteboard/doska, breakout xonalar, subtitr/captions,
  chat DM — MVP'dan tashqari. Ular uchun avval backend domeni kerak.
- Backend `email.go`/`redis.go`'da "jiraflow" satrlari (kosmetik, kod).

## Ishga tushirish

```bash
# 1. Infra + backend
cd backend && make docker-up && make run            # API :8087 (+ migratsiyalar)
# 2. LiveKit (video)
cd services/livekit && docker compose up -d          # :7880 + TURN :3478
# 3. Frontend
cd frontend && npm install && npm run dev            # :3000

# Backend'siz frontend (mock):
cd frontend && VITE_USE_MOCK=true npm run dev
```

Sinov: `http://localhost:3000` — ro'yxatdan o'ting (mentor), dars yarating, havolani ikkinchi
brauzerda oching (guest), yuqoridagi 5 stsenariyni bosib chiqing.

## Verdikt

**Inson testiga TAYYOR.** Barcha kod yozilgan, ulangan, kontrakt-toza, build/test/security yashil.
Keyingi qadam — foydalanuvchi tomonidan tirik 2-brauzer + LiveKit sinovi (yuqoridagi 5 stsenariy).
