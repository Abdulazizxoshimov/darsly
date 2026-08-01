# Deploy checklist — audit tuzatishlaridan keyin

> Yangi server: **169.58.104.245** (2026-07-31 dan; eski Contabo o'chgan).
> Serverda `telegram/Tg-bot` ning ikki boti systemd'da ishlaydi — ULARGA TEGMANG.
> Jonly hamma narsasi `darsly-` prefiks + `darsly_net` tarmog'ida.

## 0. Server tirikligini tasdiqlash

```bash
ping -c 3 169.58.104.245
ssh root@169.58.104.245 'uptime && docker ps --format "{{.Names}}\t{{.Status}}"'
```

Yetib bo'lmasa — Contabo panelidan holatni tekshiring (o'chgan / to'lov / suspend).

⚠️ Serverda **tgbot** va **tgbot-akademiya** (systemd) ishlaydi. Ularga TEGMANG.

## 1. Kod va konfiguratsiyani yuklash

```bash
# Frontend build (VITE_API_URL BO'SH — SPA va API bir originda)
cd frontend && npm ci && npm run build && rsync -a --delete dist/ ../deploy/server/www/

# Backend manbasi + deploy konfiguratsiyasi
# ⚠️ YO'L MUHIM: docker-compose.yml da backend build konteksti `../../backend`,
# ya'ni serverda tuzilma AYNAN /opt/darsly/{backend,deploy/server} bo'lishi shart.
# Avval `deploy/server` /opt/darsly/server ga tushardi va build
# "path /opt/backend not found" bilan yiqilardi (2026-07-31 da ko'rindi).
cd .. && rsync -a --delete --exclude .git --exclude node_modules \
  backend root@169.58.104.245:/opt/darsly/

# ⚠️ deploy/server uchun --delete BILAN EHTIYOT BO'LING: serverda GENERATSIYA
# QILINGAN va git'da YO'Q fayllar bor — ular o'chib ketadi:
#   .env (barcha sirlar!) · livekit.yaml · dl/*.apk · observability/metrics_token
# 2026-08-01 da aynan shu bo'ldi: .env o'chdi. Tiklash mumkin bo'ldi
# (qiymatlar ishlab turgan konteynerdan olindi), lekin konteynerlar ham
# o'chgan bo'lsa sirlar butunlay yo'qolardi va DB parolini bilmay qolardik.
rsync -a --exclude .git --exclude '.env' --exclude 'livekit.yaml' \
  --exclude 'dl/' --exclude 'observability/metrics_token' \
  deploy/server root@169.58.104.245:/opt/darsly/deploy/
```

**Sirlarni tiklash (agar .env yo'qolsa):**
```bash
ssh root@169.58.104.245 'cd /opt/darsly/deploy/server && \
  docker inspect darsly-backend --format "{{range .Config.Env}}{{println .}}{{end}}" \
  | grep -vE "^(PATH|HOSTNAME|HOME|TERM)=" | grep "=" > .env'
# Keyin: SERVER_IP=... ./remote-setup.sh  (yetishmagan kalitlarni qo'shadi)
```

## 2. Sirlar va konfiguratsiya

```bash
# SERVER_IP majburiy emas (default yangi server), lekin server almashsa shu bilan beriladi
ssh root@169.58.104.245 'cd /opt/darsly/deploy/server && SERVER_IP=169.58.104.245 ./remote-setup.sh'
```

`remote-setup.sh` endi qo'shimcha ravishda:
- `METRICS_TOKEN` va `GRAFANA_PASSWORD` yasaydi (mavjud `.env` ga ham **qo'shadi**,
  eski sirlarga tegmasdan);
- `observability/metrics_token` faylini yozadi (Prometheus `credentials_file`),
  `chmod 600`.

**Tekshirish:** `grep -c '^METRICS_TOKEN=' .env` → `1` bo'lishi kerak.

## 3. Ko'tarish

```bash
# ⚠️ Build 10+ daqiqa oladi. SSH uzilsa build ham to'xtaydi — serverda FONDA
# ishga tushiring (2026-07-31 da SSH timeout bilan bir marta uzilgan):
ssh root@169.58.104.245 'cd /opt/darsly/deploy/server && \
  nohup docker compose up -d --build > /root/jonly-build.log 2>&1 &'
# Kuzatish: ssh ... 'tail -f /root/jonly-build.log'
```

⚠️ Backend qayta ishga tushadi → **jonli darslar uziladi**. Dars bo'lmagan
vaqtda bajaring.

## 4. Deploydan keyingi tekshiruv (SHART)

```bash
# a) Sog'liq
curl -sS https://app.169.58.104.245.sslip.io/api/v1/app-config | head -c 200

# b) /metrics tokensiz YOPIQ bo'lishi kerak
curl -s -o /dev/null -w '%{http_code}\n' https://app.169.58.104.245.sslip.io/metrics   # kutilgan: 404 yoki 401

# c) /swagger production'da YO'Q
curl -s -o /dev/null -w '%{http_code}\n' https://app.169.58.104.245.sslip.io/swagger/index.html  # kutilgan: 404

# d) SPA xavfsizlik sarlavhalari (M5)
curl -sI https://app.169.58.104.245.sslip.io/ | grep -iE 'content-security-policy|strict-transport|x-frame'

# e) Prometheus maqsadni ko'ryaptimi
ssh root@169.58.104.245 'docker exec darsly-prometheus wget -qO- localhost:9090/api/v1/targets' \
  | grep -o '"health":"[a-z]*"'    # kutilgan: "up"
```

⚠️ **(d) eng muhimi:** `Permissions-Policy` da `camera=(self)` bo'lishi shart.
Backend middleware'idagi qiymat (`camera=()`) hujjatga tushsa **kamera butunlay
o'chadi** va dars ishlamaydi.

## 5. Grafana

Tashqariga chiqarilmagan (`127.0.0.1:3030`). Kirish:

```bash
ssh -L 3030:localhost:3030 root@169.58.104.245
# brauzerda http://localhost:3030 · admin / (GRAFANA_PASSWORD .env dan)
```

## 6. Qolgan operatsion ishlar

- [ ] **Sentry DSN** — backend (`SENTRY_DSN` .env) va mobil
      (`./gradlew assembleRelease -PdarslySentryDsn=...`). Berilmaguncha
      crash-telemetriya kodda tayyor, lekin **jim**.
- [ ] **DB backup** — hozir RPO ∞. Minimal variant: `pg_dump` cron + MinIO'ga
      yuklash + tiklashni BIR MARTA sinab ko'rish (sinalmagan backup — backup emas).
- [ ] **TURN TLS/443** — ataylab kechiktirilgan (443 Caddy'da band). Tartib:
      `deploy/server/livekit.yaml.example`.

## Mahalliy tekshirilgan (deploy oldidan qayta sinash shart emas)

| Nima | Usul | Natija |
|---|---|---|
| `Caddyfile` sintaksisi | `caddy validate` | ✅ Valid |
| `docker-compose.yml` | `docker compose config` | ✅ (faqat `GRAFANA_PASSWORD` talab qiladi — to'g'ri xulq) |
| `prometheus.yml` | `promtool check config` | ✅ Valid |
| `alerts.yml` | `promtool check rules` | ✅ 6 qoida |
| Alert metrika nomlari | kod bilan solishtirildi | ✅ hammasi mavjud (`darsly_*` prefiksi bilan) |
| `/metrics` + `/swagger` xulqi | `api/router_ops_test.go` | ✅ 4 test |
| `remote-setup.sh` | `bash -n` | ✅ sintaksis toza |
