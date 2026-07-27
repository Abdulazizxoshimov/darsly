# Darsly — API Test Coverage

Avtomatlashtirilgan API funksional testlar (real HTTP + testify + real PostgreSQL/Redis).
Joylashuv: `backend/internal/apitests/` (mavjud `newTestServer` harness ustiga qurilgan).
Ishga tushirish:
```
cd backend && TEST_DB_HOST=localhost TEST_DB_PORT=5442 TEST_REDIS_HOST=localhost TEST_REDIS_PORT=6399 \
  go test ./internal/apitests/... -count=1
```
LiveKit-bog'liq success (room-token, recording-start, admit-token) uchun qo'shimcha:
`TEST_LIVEKIT_URL=ws://localhost:7880 TEST_LIVEKIT_KEY=devkey TEST_LIVEKIT_SECRET=...` (aks holda skip).

## Natija: **to'liq suite `ok` (0 FAIL)** — ~46 test, 2 skip (LiveKit yo'q).

## Qamrov (success + bad)

| Domen | ✅ Success | ❌ Bad |
|---|---|---|
| **Auth** | register→201+token, login→200, refresh→200, logout→204, /me→200 | zaif parol→400, dublikat email→409, noto'g'ri parol→401, token'siz→401, yaroqsiz token→401 |
| **User** | GET/PUT /users/me, parol o'zgartirish (yangi bilan login, eski→401) | IDOR GET /users/&lt;other&gt;→403, self role-escalation bloklanadi, noto'g'ri joriy parol→400 |
| **Lesson** | create→201+slug, list, GET(owner), PATCH, DELETE→404 | student create→403 (RBAC), bo'sh title→400, mavjud emas→404, begona mentor→403 |
| **Joinlink** | preview, join→waiting_room+request_id, join→next_step=join+token, parolli→200 | yaroqsiz slug→404, noto'g'ri parol→401, qulflangan→403 |
| **Waitingroom** | join→request, mentor ro'yxat, status=pending, reject→rejected, admit→token(LK) | begona admit→403, mavjud emas→404, student admit→403, token'siz→401, **2-admit→409 (TOCTOU)**, qayta reject→409 |
| **Room token** | owner→token+ws_url+role=host (LK) | begona→403, mavjud emas→404, token'siz→401, student→403, end: owner→204/begona→403 |
| **Recording** | start→201, list(owner) (LK) | student→403, begona→403, token'siz→401, mavjud emas→404, download: →404/400/403, stop: →404/403/400 |
| **Notification** | list+total, unread-count, mark-read→count kamayadi, filtr, read-all→count=0 | cross-user mark-read xavfsiz no-op (A ning count'i o'zgarmaydi) |
| **E2E zanjir** | register→login→dars→link→kutish→admit→token; parolli variant; to'g'ridan-to'g'ri join; 3 ketma-ket guest + reject | parolsiz/noto'g'ri parol→401 |

## Test paytida topilgan HAQIQIY buglar (tuzatildi)

1. **500-niqoblash (correctness)** — `room.HostToken` va `recording.StartRecording` da `livekit.Enabled()` tekshiruvi mavjudlik/egalikdan OLDIN turgani sabab, LiveKit o'chiq bo'lsa yo'q/begona dars **404/403 o'rniga 500** qaytarardi. Tartib to'g'irlandi (mavjudlik/egalik → keyin LiveKit). `internal/usecase/room/room.go`, `internal/usecase/recording/recording.go`.

## Ma'lum kuzatuvlar (bug emas)
- **Admit LiveKit'siz**: `Admit` avval status→admitted qiladi, keyin token (LiveKit o'chiq→500). Guest DB'da admitted, token yo'q. Prod'da LiveKit doim yoqilgan → muammo emas; agar himoya kerak bo'lsa Admit boshida video-enabled tekshiruvi qo'shiladi.
- **Notification cross-user**: `MarkRead` `WHERE id AND user_id` bilan scope qilingan → begona ID uchun 204 (xavfsiz no-op), leak yo'q.
- **Rate-limit izolyatsiyasi**: testlar Redis `rl:*` kalitlarini har test boshida tozalaydi (`clearRateLimits`).
