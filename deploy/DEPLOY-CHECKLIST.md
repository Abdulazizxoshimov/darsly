# Deploy checklist — audit tuzatishlaridan keyin

> Bu ro'yxat **server qaytgach** bajariladi. 2026-07-28 holatiga
> `194.163.139.242` yetib bo'lmaydi (ICMP/22/443 — hammasi jim, yo'l upstream'da
> uziladi). Deploy artefaktlari mahalliy tekshirildi, quyida faqat bajarish qoladi.

## 0. Server tirikligini tasdiqlash

```bash
ping -c 3 194.163.139.242
ssh root@194.163.139.242 'uptime && docker ps --format "{{.Names}}\t{{.Status}}"'
```

Yetib bo'lmasa — Contabo panelidan holatni tekshiring (o'chgan / to'lov / suspend).

⚠️ Serverda **tgbot** va **eduverse** ham bor. Ularга TEGMANG: darsly hamma
narsasi `darsly-` prefiks va `darsly_net` tarmog'ida.

## 1. Kod va konfiguratsiyani yuklash

```bash
# Frontend build (VITE_API_URL BO'SH — SPA va API bir originda)
cd frontend && npm ci && npm run build && rsync -a --delete dist/ ../deploy/server/www/

# Backend manbasi + deploy konfiguratsiyasi
cd .. && rsync -a --delete --exclude .git backend deploy/server root@194.163.139.242:/opt/darsly/
```

## 2. Sirlar va konfiguratsiya

```bash
ssh root@194.163.139.242 'cd /opt/darsly/deploy/server && ./remote-setup.sh'
```

`remote-setup.sh` endi qo'shimcha ravishda:
- `METRICS_TOKEN` va `GRAFANA_PASSWORD` yasaydi (mavjud `.env` ga ham **qo'shadi**,
  eski sirlarga tegmasdan);
- `observability/metrics_token` faylini yozadi (Prometheus `credentials_file`),
  `chmod 600`.

**Tekshirish:** `grep -c '^METRICS_TOKEN=' .env` → `1` bo'lishi kerak.

## 3. Ko'tarish

```bash
ssh root@194.163.139.242 'cd /opt/darsly/deploy/server && docker compose up -d --build'
```

⚠️ Backend qayta ishga tushadi → **jonli darslar uziladi**. Dars bo'lmagan
vaqtda bajaring.

## 4. Deploydan keyingi tekshiruv (SHART)

```bash
# a) Sog'liq
curl -sS https://app.194.163.139.242.sslip.io/api/v1/app-config | head -c 200

# b) /metrics tokensiz YOPIQ bo'lishi kerak
curl -s -o /dev/null -w '%{http_code}\n' https://app.194.163.139.242.sslip.io/metrics   # kutilgan: 404 yoki 401

# c) /swagger production'da YO'Q
curl -s -o /dev/null -w '%{http_code}\n' https://app.194.163.139.242.sslip.io/swagger/index.html  # kutilgan: 404

# d) SPA xavfsizlik sarlavhalari (M5)
curl -sI https://app.194.163.139.242.sslip.io/ | grep -iE 'content-security-policy|strict-transport|x-frame'

# e) Prometheus maqsadni ko'ryaptimi
ssh root@194.163.139.242 'docker exec darsly-prometheus wget -qO- localhost:9090/api/v1/targets' \
  | grep -o '"health":"[a-z]*"'    # kutilgan: "up"
```

⚠️ **(d) eng muhimi:** `Permissions-Policy` da `camera=(self)` bo'lishi shart.
Backend middleware'idagi qiymat (`camera=()`) hujjatga tushsa **kamera butunlay
o'chadi** va dars ishlamaydi.

## 5. Grafana

Tashqariga chiqarilmagan (`127.0.0.1:3030`). Kirish:

```bash
ssh -L 3030:localhost:3030 root@194.163.139.242
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
