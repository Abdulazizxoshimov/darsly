# Runbook — Jonly (Darsly) production

Tiklash / deploy / rollback / incident tartibi (I6). Yangi odam shu hujjatni o'qib
deploy qila olishi kerak. Server: `169.58.104.245` (kirish: `SERVER-CREDENTIALS.txt`).

> ⚠️ Serverda `telegram/Tg-bot` ning ikki systemd boti ishlaydi — ularga TEGMA.
> Jonly'ning hammasi `darsly-` prefiks + `darsly_net` tarmog'ida.

---

## 1. Birinchi deploy

`deploy/DEPLOY-CHECKLIST.md` to'liq tartib. Qisqacha:
1. Frontend build (`VITE_API_URL` bo'sh) → `deploy/server/www/`.
2. Manba (backend + deploy) serverga rsync.
3. `cd /root/darsly/deploy/server && ./remote-setup.sh` (sir + `.env` + `livekit.yaml`
   + UFW portlar + **backup cron** yasaydi).
4. `docker compose up -d --build`.
5. Tekshirish: `curl -sk https://app.169.58.104.245.sslip.io/api/v1/health`.

## 2. Yangilash (kod o'zgargach)
```bash
# lokal: frontend build → www/ → rsync
cd /root/darsly/deploy/server
docker compose up -d --build backend caddy   # faqat o'zgarganini
docker compose ps                             # hammasi healthy?
docker compose logs -f backend                # migratsiya + start loglari
```

## 3. Rollback
- **Kod:** oldingi git commit'ga qayting, qayta build + `up -d --build backend`.
- **Sxema:** `migrations/MIGRATIONS.md` → `down` qo'lda (ehtiyot: ma'lumot yo'qolishi mumkin).
- **Ma'lumotlar bazasi:** `deploy/server/BACKUP.md` → real tiklash tartibi.

## 4. DB backup / tiklash
- Avtomatik: kunlik cron 03:17 UTC (`backup.sh` → MinIO `darsly-backups`).
- Tiklash va **sinov**: `deploy/server/BACKUP.md`.

## 5. Kuzatuv (metrikalar + loglar)
- Grafana faqat `127.0.0.1:3030` — SSH tunnel bilan:
  `ssh -L 3030:localhost:3030 root@169.58.104.245`, so'ng `http://localhost:3030`.
- Parol: `.env` dagi `GRAFANA_PASSWORD`.
- Loglar: Loki (Grafana ichida). Metrikalar: Prometheus (`alerts.yml`).
- Xato/crash telemetriyasi (Sentry): **DSN qo'yilganda** yoqiladi — hozircha
  o'chiq, backend loglar Grafana'da.

## 6. Incident — tez tekshiruv
```bash
docker compose ps                    # qaysi servis unhealthy?
docker compose logs --tail=100 backend
docker stats --no-stream             # CPU/RAM (egress og'ir)
df -h                                # disk (yozuvlar + backup)
ufw status | grep -E '443|3478|30000'  # TURN portlari ochiqmi
```
Video o'tmasa: **30000-40000/udp** (TURN relay) ochiqligini tekshiring —
`remote-setup.sh:131-138` va `CLAUDE.md` ogohlantirishiga qarang.

---

## 7. Redis fail-open siyosati (B5) — ONGLI QAROR

**Hozirgi xulq — CHEKLANGAN fail-open** (`internal/pkg/token/jwt.go:143-180`).
Access token'ni tekshirishda Redis uzilib qolsa:
- token **`redisOutageGrace` ichida** (default 5 daqiqa) yaratilgan bo'lsa → vaqtincha qabul qilinadi;
- undan **eski** token bo'lsa → **rad etiladi** (fail-CLOSED).

Ya'ni Redis uzilganda jonli, yangi sessiyalar davom etadi, lekin bekor qilingan
eski tokenlar QAYTA JONLANMAYDI. Bu — to'liq fail-open (hamma tiriladi) bilan
to'liq fail-closed (platforma to'xtaydi) orasidagi ongli murosalar.

**Nega bunday:** to'liq fail-closed Redis yiqilishida har bir foydalanuvchini dars
o'rtasida chiqarib yuboradi; to'liq fail-open esa o'g'irlangan/bekor qilingan
tokenlarni tiriltiradi. Cheklangan (5 daqiqa) variant ikkalasining eng yomonini
oldini oladi.

**Shart (aks holda bu qaror yaroqsiz):** Redis uzilishi **ko'rinishi** kerak.
- Prometheus'da `up == 0` alerti bor (`observability/alerts.yml:76`, umumiy
  instance-down). ⚠️ Redis aynan scrape qilinishini tekshiring — kerak bo'lsa
  `redis_exporter` qo'shing, aks holda alert Redis'ni qamramaydi.
- Redis persistence (AOF) yoqilgan (`docker-compose.yml`: `--appendonly yes`) —
  restart'da bekor-qilingan sessiyalar ro'yxati yo'qolmaydi.

**Qachon qayta ko'riladi:** yuqori xavfli (to'lov/admin) amallar qo'shilsa —
o'sha yo'llar uchun grace'ni 0 ga tushirish yoki qo'shimcha tekshiruv.
