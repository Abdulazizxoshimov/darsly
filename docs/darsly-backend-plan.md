# Darsly Backend — reja va HOZIRGI HOLAT

> Backend ALLAQACHON qurilgan (`go build ./...` = 0). Backend-agentlar noldan yozmaydi —
> **audit + kamchilik tuzatish** rejimida ishlaydi.

## Domenlar (barchasi mavjud)
| Domen | Holat | agent |
|---|---|---|
| auth (register/login/refresh/logout/forgot/reset/me) | ✅ | backend-agent-1 |
| user (CRUD, me, password, RBAC) | ✅ | backend-agent-1 |
| lesson (CRUD, join_slug, passcode, lock) | ✅ | backend-agent-1 |
| joinlink (preview/join, passcode lockout) | ✅ | backend-agent-1 |
| room (LiveKit token, host controls, end) | ✅ | backend-agent-2 |
| waitingroom (WS push, admit/reject, atomik) | ✅ | backend-agent-2 |
| recording (Egress→MinIO, webhook, presigned) | ✅ | backend-agent-2 |
| notification (list/unread/read, reminder worker) | ✅ | backend-agent-2 |
| chat (host persist + LiveKit broadcast) | ✅ | backend-agent-2 |
| poll (create/close/vote/results) | ✅ | backend-agent-2 |

## Ma'lum kamchiliklar (audit'da tasdiqlash + tuzatish)
1. **🔴 HIGH — Parol reset/deactivate Redis sessiyalarni tozalamaydi** → o'g'irlangan sessiya reset'dan keyin yashaydi. `auth.go:246`, `jwt.go:111,140`. (backend-agent-1)
2. **🟡 IDOR — `GET /users/:id`** har autentifikatsiyalangan foydalanuvchiga ochiq (self-check yo'q). `policy.csv:11`, `user.go:52`. (backend-agent-1 / security)
3. **🟡 Redis fail-open** — sessiya-tekshiruv va rate-limit Redis o'chsa ochiladi. (security qaror qiladi — hozircha hujjatlashtiriladi)
4. **🟡 CORS bo'sh-allowlist** har origin'ni credentials bilan aks ettiradi (`FRONTEND_BASE_URL` bo'sh bo'lsa). (backend-agent-1)
5. **WithTx** yozilgan-u ishlatilmaydi — `Register` va ko'p-yozuvli oqimlar atomik emas. (backend-agent-1, past ustuvorlik)

## Infra (LiveKit past-internet)
`services/livekit/livekit.prod.yaml`: TURN (3478 + TLS 443), `use_external_ip`, wss. `devops-agent` tekshiradi.
