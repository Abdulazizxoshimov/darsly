# CI/CD — GitOps oqimi (GitHub → ghcr → server)

> Real-kompaniya oqimи: **feature-branch → PR → CI (yashил) → master'ga merge →
> avtomatik build (ghcr image) → serverga deploy**. QA testlari CI'да, hamda
> har kecha jadval bo'yicha ishlaydi.

## Oqim (bir qarashda)

```
 dasturchi                       GitHub                          Server (169.58...)
 ─────────                       ──────                          ─────────────────
 git checkout -b feat/x
 ...kod...
 git push origin feat/x   ──►  CI (ci.yml): backend+frontend
                                +e2e+mobil testlar  ──►  ❌ qizil bo'lsa merge YO'Q
 PR ochish (→ master)     ──►  Branch protection: CI yashил shart
 PR merge → master        ──►  Deploy (deploy.yml):
                                 1) backend Docker image quradi
                                 2) ghcr.io ga push (:sha, :latest)
                                 3) frontend build (dist)
                                 4) SSH ──────────────────────►  docker compose pull backend
                                    frontend dist → www              docker compose up -d
                                                                     /ready 200 tekshiradi
 har kecha 02:00 UTC      ──►  CI (schedule): to'liq regressiya
```

## Workflow'lar (`.github/workflows/`)

| Fayl | Qachon | Nima qiladi |
|---|---|---|
| **`ci.yml`** | har push/PR + **har kecha (cron)** | backend (build+vet+lint+**test -race**), frontend (lint+vitest+build), e2e (Playwright), mobil (unit+lint) |
| **`deploy.yml`** | master'ga push (+ qo'lда) | backend image → ghcr → serverga (image pull + frontend dist + `compose up`) + `/ready` tekshiruvi |

## Nega bunday (qarorlar)

- **Registry (ghcr.io):** server endi **build qilmaydi** — tayyor image'ni tortadi. Deploy tez,
  takrorlanuvchi, va build CPU'si serverга tushmaydi (jonli darsга xalaqit bermaydi).
  Rollback = eski `:sha` teg (image saqlanadi).
- **Frontend statik:** SPA — image emas, `dist` artifact serverning `www` siga jo'natiladi
  (Caddy `/srv` dan xizmat qiladi; CSP/kesh qoidalari `Caddyfile` da o'zgarmaydi).
- **`backend` xizmatida `image:` + `build:` ikkalasi:** serverда `pull` image'ni oladi;
  lokal dev'да `docker compose up --build` hali ham quradi. `BACKEND_TAG` = commit sha.
- **Branch protection asosiy kafolat:** kod master'ga faqat CI yashил PR orqali tushadi,
  shuning uchun `deploy.yml` test'ni qayta yugurtirmaydi (PR'да o'tган).

---

## ⚙️ BIR MARTALIK SOZLASH (siz qilasiz — men qila olmayman)

### 1. GitHub Secrets (Settings → Secrets and variables → Actions → New secret)
| Secret | Qiymat |
|---|---|
| `SERVER_HOST` | `169.58.104.245` |
| `SERVER_SSH_KEY` | Deploy uchun SSH **maxfiy** kaliti (quyida yasaladi) |
| `GHCR_TOKEN` | `read:packages` huquqli PAT (server private image tortishi uchun) |

### 2. Deploy SSH kaliti (parolsiz, faqat deploy uchun)
```bash
# Lokalда yasang:
ssh-keygen -t ed25519 -f deploy_key -N "" -C "github-actions-deploy"
# Public'ni serverga qo'shing:
ssh-copy-id -i deploy_key.pub root@169.58.104.245   # yoki qo'lда authorized_keys ga
# Maxfiy'ni (deploy_key faylining ICHINI) GitHub'ga `SERVER_SSH_KEY` sifatida joylang.
rm deploy_key deploy_key.pub   # lokalда saqlamang
```

### 3. GHCR_TOKEN (PAT)
GitHub → Settings → Developer settings → **Personal access tokens (classic)** →
`read:packages` belgilab yarating → `GHCR_TOKEN` sifatida joylang.
*(Image'ni public qilsangiz bu shart emas: Packages → darsly-backend → Package settings → Change visibility → Public.)*

### 4. Branch protection (Settings → Branches → Add rule, `master`)
- ✅ Require a pull request before merging
- ✅ Require status checks to pass → tanlang: `backend`, `frontend`, `e2e`, `mobile`
- ✅ Require branches to be up to date before merging

### 5. Server tayyor (bir marta)
- `/opt/darsly/deploy/server/` da `.env`, `livekit.yaml` allaqachon bor (qo'lда deploy'дan).
- Docker'ga ghcr login (private image bo'lsa): CI o'zi `deploy.yml` da qiladi (`GHCR_TOKEN`).

---

## 👩‍🔬 QA jamoa uchun — testlar qayerga qo'shiladi

| Qatlam | Joy | Buyruq (CI shuni yugurtiradi) |
|---|---|---|
| Backend (unit/integration) | `backend/**/_test.go` (`testutil` faqe'lari bilan) | `go test -race ./...` |
| Frontend (unit/komponent) | `frontend/src/**/*.test.js(x)` (vitest) | `npm test` |
| Frontend E2E (brauzer) | `frontend/e2e/**` (Playwright + MSW mock) | `npm run test:e2e` |
| Mobil (JVM unit) | `mobile/app/src/test/**` (sof mantiq: MediaTuning, ScreenAudioPolicy, AudioMixer…) | `./gradlew :app:testDebugUnitTest` |

QA yangi test qo'shsa — PR ochadi, CI avtomatik yugurtiradi. **Har kecha** (cron) butun batareya
qayta ishlaydi → tashqi regressiya (image/modul yangilanishi) ertalab ko'rinadi.

### Kelajak (kengaytirish)
- **Yuklama testи** (`backend/tests/load/`) — jadval bo'yicha alohida workflow (100–200 soxta ishtirokchi)
  staging'да; hozir manual. MVP oldidan avtomatlashtiriladi.
- **Deploy'дan keyingi smoke** — `deploy.yml` `/ready` tekshiradi; keyinroq to'liq E2E staging'да.

---

## Rollback (deploy buzilса)
```bash
ssh root@169.58.104.245 'cd /opt/darsly/deploy/server && \
  export BACKEND_TAG=<oldingi-yaxshi-sha> && docker compose up -d backend'
```
Eski image'lar ghcr'да saqlanadi (`:sha` teglari), shuning uchun har qanday oldingi versiyaga qaytish mumkin.
