# Load testlar

## 1. LiveKit guruh darsi (Go) — 20+ ishtirokchi

Backend (`:8087`) va LiveKit (`:7880`) ishlab turgan holda:

```bash
cd backend
go run ./tests/load/livekit_load -n 20
# yoki ko'proq:
go run ./tests/load/livekit_load -n 50 -base http://localhost:8087
```

N ta guest'ni bitta darsga joinlink orqali token oldirib, haqiqiy LiveKit
xonasiga ulaydi va serverdan ACTIVE ishtirokchilar sonini tekshiradi.

## 2. API load (k6)

`k6` o'rnatilgan bo'lsa. Avval parolsiz, waiting-room OFF, **jonli** dars
yarating va uning `join_slug`ini oling, so'ng:

```bash
k6 run -e BASE=http://localhost:8087 -e SLUG=<join_slug> tests/load/api_load.js
```

25 ta virtual foydalanuvchi bir vaqtda `/joinlink/:slug` ga uriladi;
p95 < 800ms va xato < 1% chegaralari tekshiriladi.
