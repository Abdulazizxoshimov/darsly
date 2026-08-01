#!/usr/bin/env bash
# Serverda ishlaydi (deploy/server/ ichida). Idempotent:
#  - .env yo'q bo'lsa — kuchli sirlar bilan yasaydi (bor bo'lsa TEGMAYDI, faqat
#    yetishmayotgan YANGI kalitlarni qo'shadi — ensure_env).
#  - livekit.yaml HAR SAFAR shablondan qayta yasaladi (sirlar .env dan olinadi).
#    Sababi: shablonga TURN/webhook kabi yangi bloklar qo'shilganda eski serverdagi
#    livekit.yaml o'zi yangilanmasa, mobil klient uchun TURN paydo bo'lmaydi.
#    Eski fayl livekit.yaml.bak ga saqlanadi.
#  - ufw kerakli portlarni ochadi (SSH policy'ga tegmaydi).
#
# Yozib olish (Egress) DOIM sozlanadi: livekit.yaml ga `redis:` bloki qo'shiladi
# (egress ↔ LiveKit shu Redis orqali gaplashadi) va egress oddiy
# `docker compose up -d` bilan ko'tariladi.
set -euo pipefail
cd "$(dirname "$0")"

# IP env orqali beriladi: server almashganda skript tahrirlanmasin.
#   SERVER_IP=169.58.104.245 ./remote-setup.sh
# (2026-07-31: eski Contabo 194.163.139.242 to'lov tugab o'chdi.)
IP="${SERVER_IP:-169.58.104.245}"
APP_HOST="app.${IP}.sslip.io"
LK_HOST="livekit.${IP}.sslip.io"
FILES_HOST="files.${IP}.sslip.io"   # MinIO presigned URL'lar uchun ochiq host (Caddy)

gen() { openssl rand -hex "$1"; }

# ensure_env KEY VALUE — .env da kalit YO'Q bo'lsa qo'shadi. Bor bo'lsa tegmaydi
# (serverdagi mavjud sirlar/qo'lda tuzatilgan qiymatlar buzilmasin).
ensure_env() {
  grep -q "^$1=" .env || { printf '%s=%s\n' "$1" "$2" >> .env; echo "   + .env: $1 qo'shildi"; }
}

if [[ ! -f .env ]]; then
  echo ">> .env yasalyapti (kuchli sirlar)..."
  DB_PASSWORD="$(gen 16)"
  REDIS_PASSWORD="$(gen 16)"
  MINIO_ACCESS_KEY="darsly$(gen 4)"
  MINIO_SECRET_KEY="$(gen 20)"
  JWT_SECRET="$(gen 32)"                 # 64 hex belgi
  LIVEKIT_API_SECRET="$(gen 24)"         # 48 hex belgi
  METRICS_TOKEN="$(gen 24)"              # /metrics scrape tokeni (M16)
  GRAFANA_PASSWORD="$(gen 12)"           # Grafana admin (SSH tunnel ortida)

  cat > .env <<EOF
# AVTOMATIK YASALGAN — serverda qoladi, git'ga tushmaydi.
APP_PORT=8080
APP_ENV=production
LOG_LEVEL=info
FRONTEND_BASE_URL=https://${APP_HOST}
# Ochiq ro'yxatdan o'tish O'CHIQ: mentor hisoblarini admin ochadi (users CRUD,
# seed-admin bilan kiriladi). Ochiq registratsiya spam/begona hisoblar manbai edi.
ALLOW_OPEN_REGISTRATION=false
SEED_ADMIN_EMAIL=admin@darsly.uz
SEED_ADMIN_PASSWORD=$(gen 16)   # xavfsizlik K-1: qattiq yozilgan parol o'rniga generatsiya

DB_HOST=postgres
DB_PORT=5432
DB_NAME=darsly
DB_USER=darsly
DB_PASSWORD=${DB_PASSWORD}
DB_MAX_CONNS=50
DB_MIN_CONNS=5
# Ichki docker tarmog'ida postgres TLS yo'q → sslmode=disable (APP_ENV=production
# aks holda pool 'sslmode=require' ishlatib "server refused TLS connection" beradi).
# DATABASE_URL ham pool, ham migrator uchun DSN'ni to'liq override qiladi.
DATABASE_URL=postgres://darsly:${DB_PASSWORD}@postgres:5432/darsly?sslmode=disable

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_DB=0

MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY}
MINIO_SECRET_KEY=${MINIO_SECRET_KEY}
MINIO_BUCKET=darsly
MINIO_USE_SSL=false
# Presigned URL'lar SigV4 imzosiga host header ham kiradi → ichki 'minio:9000' ga
# imzolangan link brauzerda ochilmaydi. Shu sabab imzo ochiq host bilan qo'yiladi
# (Caddy files.<IP>.sslip.io → darsly-minio:9000, Host header saqlanadi).
MINIO_PUBLIC_ENDPOINT=${FILES_HOST}
MINIO_PUBLIC_USE_SSL=true

JWT_SECRET=${JWT_SECRET}
JWT_ACCESS_TTL=15m
# 14 kun (avval 30). Refresh brauzerda localStorage'da yashaydi — XSS uni
# o'g'irlasa token TTL tugagunicha ishlaydi. Backend default'i bilan mos
# (`internal/pkg/config/config.go`).
JWT_REFRESH_TTL=336h

EMAIL_ENABLED=false
SMTP_FROM=noreply@darsly.uz

LIVEKIT_HOST=wss://${LK_HOST}
# Klientlarga beriladigan manzil oshkora — backend ichki yo'lga o'tsa ham klientlar buzilmaydi.
LIVEKIT_CLIENT_WS_URL=wss://${LK_HOST}
LIVEKIT_API_KEY=darslykey
LIVEKIT_API_SECRET=${LIVEKIT_API_SECRET}
LIVEKIT_WEBHOOK_API_KEY=darslykey

TRUSTED_PROXIES=172.16.0.0/12,10.0.0.0/8,192.168.0.0/16
RATE_LIMIT_RPS=30
RATE_LIMIT_BURST=60

# Dars avto-yakuni (PRODUCT.md «Dars hayoti»): 4 soatlik texnik limit va
# xona bo'shagach 20 daqiqadan keyin yakunlash. Mentor "Yakunlash"ni bosmasa ham
# dars osilib qolmaydi (egress/CPU bekorga band bo'lmaydi).
LESSON_MAX_DURATION=4h
LESSON_EMPTY_GRACE=20m
LESSON_SWEEP_INTERVAL=1m

# Yozuvlar retention (PRODUCT.md №5): 30 kun saqlanadi, o'chishdan 3 kun oldin
# mentorga bildirishnoma. Busiz 145 GB disk bir necha oyda to'lardi.
RECORDING_RETENTION_DAYS=30
RECORDING_RETENTION_WARN_DAYS=3
RECORDING_RETENTION_INTERVAL=1h

# Kuzatuv (M16). METRICS_TOKEN bo'sh bo'lsa production'da /metrics UMUMAN
# yoqilmaydi (api/router.go) — ya'ni endpoint hech qachon himoyasiz qolmaydi.
METRICS_TOKEN=${METRICS_TOKEN}
GRAFANA_PASSWORD=${GRAFANA_PASSWORD}
EOF
  echo ">> .env yasaldi."
else
  echo ">> .env allaqachon bor — tegilmadi (sirlar saqlanadi)."
  # Eski .env da yangi kalitlar bo'lmasligi mumkin — yetishmayotganini qo'shamiz
  # (mavjud sirlarga TEGMASDAN). Busiz eski serverda Grafana ko'tarilmasdi va
  # Prometheus 401 olardi.
  grep -q '^METRICS_TOKEN=' .env || echo "METRICS_TOKEN=$(gen 24)" >> .env
  grep -q '^GRAFANA_PASSWORD=' .env || echo "GRAFANA_PASSWORD=$(gen 12)" >> .env
  # Keyin qo'shilgan sozlamalar mavjud serverga ham yetib borishi kerak.
  ensure_env MINIO_PUBLIC_ENDPOINT "${FILES_HOST}"
  ensure_env MINIO_PUBLIC_USE_SSL   true
  ensure_env LIVEKIT_WEBHOOK_API_KEY darslykey
  # Mobil ilova versiya nazorati (GET /api/v1/app-config).
  # DIQQAT: APK_URL bo'sh bo'lsa FORCE_UPDATE dialogida yuklab olish tugmasi CHIQMAYDI —
  # ya'ni kill-switch yoqilsa foydalanuvchi boshi berk ko'chada qoladi. Shu sabab URL
  # oldindan to'ldiriladi (fayl hali qo'yilmagan bo'lsa ham — /download/ katalogi tayyor).
  ensure_env APP_ANDROID_MIN_VERSION    1.0.0
  ensure_env APP_ANDROID_LATEST_VERSION 1.0.0
  ensure_env APP_ANDROID_APK_URL        "https://${APP_HOST}/download/darsly-mentor.apk"
  ensure_env APP_ANDROID_FORCE_UPDATE   false
  # Dars avto-yakuni va yozuvlar retention'i (2-hafta). Mavjud serverda ham
  # yoqilishi kerak: busiz osilgan darslar va cheksiz o'sadigan yozuv arxivi qoladi.
  ensure_env LESSON_MAX_DURATION            4h
  ensure_env LESSON_EMPTY_GRACE             20m
  ensure_env LESSON_SWEEP_INTERVAL          1m
  ensure_env RECORDING_RETENTION_DAYS       30
  ensure_env RECORDING_RETENTION_WARN_DAYS  3
  ensure_env RECORDING_RETENTION_INTERVAL   1h
fi

# ── Kuzatuv sirlari (M16) ────────────────────────────────────────────────────
# Prometheus scrape tokenini FAYL sifatida beradi (`credentials_file`), chunki
# uni prometheus.yml ichiga yozish sirni git-tracked konfiguratsiyaga
# olib kirardi. Fayl faqat serverda yashaydi.
echo ">> Prometheus scrape tokeni yozilyapti..."
MT="$(grep '^METRICS_TOKEN=' .env | cut -d= -f2-)"
[[ -n "$MT" ]] || { echo "!! .env da METRICS_TOKEN yo'q"; exit 1; }
mkdir -p observability
printf '%s' "$MT" > observability/metrics_token
chmod 600 observability/metrics_token

# ── livekit.yaml — HAR SAFAR shablondan qayta yasaladi ────────────────────────
# LiveKit config fayli ${ENV} ni KENGAYTIRMAYDI (empirik tasdiqlangan), shu sabab
# sir/domen/parol shu yerda sed bilan o'rniga qo'yiladi.
echo ">> livekit.yaml shablondan yasalyapti (recording: doim yoqilgan)..."
LK_SECRET="$(grep '^LIVEKIT_API_SECRET=' .env | cut -d= -f2-)"
RD_PASSWORD="$(grep '^REDIS_PASSWORD=' .env | cut -d= -f2-)"
[[ -n "$LK_SECRET" ]] || { echo "!! .env da LIVEKIT_API_SECRET yo'q"; exit 1; }
if [[ -f livekit.yaml ]]; then cp -f livekit.yaml livekit.yaml.bak; fi   # rollback uchun

sed -e "s|__LIVEKIT_SECRET__|${LK_SECRET}|" \
    -e "s|__TURN_DOMAIN__|${LK_HOST}|" \
    -e "s|__APP_HOST__|${APP_HOST}|" \
    -e "s|__REDIS_PASSWORD__|${RD_PASSWORD}|" \
    livekit.yaml.example > livekit.yaml

# `redis:` bloki HAR DOIM yoqiladi — egress uchun majburiy.
#
# Avval bu `ENABLE_RECORDING=1` ortida edi va serverda hech qachon berilmagan,
# natijada LiveKit egress bilan umumiy Redis'siz ishlagan va yozib olish
# JIMGINA ishlamagan (UI'da tugma bor edi, orqasida hech narsa yo'q).
# Mahsulot qoidasi o'zgardi: yozib olish default yoniq va avtomatik
# boshlanadi, ya'ni egress konfiguratsiyasi ixtiyoriy bo'la olmaydi.
sed -i 's|^#RECORDING# ||' livekit.yaml
if grep -q '__[A-Z_]*__' livekit.yaml; then
  echo "!! livekit.yaml da almashtirilmagan placeholder qoldi"; exit 1
fi
echo ">> livekit.yaml tayyor (TURN: enabled, domain=${LK_HOST})."

echo ">> UFW portlari ochilyapti (faqat ALLOW qo'shadi, SSH'ga tegmaydi)..."
ufw allow 80/tcp            >/dev/null 2>&1 || true   # HTTP (ACME)
ufw allow 443/tcp           >/dev/null 2>&1 || true   # HTTPS
ufw allow 7881/tcp          >/dev/null 2>&1 || true   # LiveKit RTC over TCP (fallback)
ufw allow 3478/udp          >/dev/null 2>&1 || true   # TURN/STUN — allokatsiya so'rovi shu portga keladi
# TURN RELAY diapazoni — MAJBURIY va OSON O'TKAZIB YUBORILADI.
# TURN ikki bosqichda ishlaydi: (1) klient 3478'ga allokatsiya so'rovi yuboradi,
# (2) server relay uchun ALOHIDA port ajratadi va media SHU port orqali oqadi.
# LiveKit default relay diapazoni 30000-40000 (log: "turn.relay_range_start").
# Bu diapazon yopiq bo'lsa TURN loglarda "Starting TURN server" deb ko'rinadi,
# lekin media O'TMAYDI — jimgina ishlamaydigan holat. Empirik aniqlangan (2026-07-25).
ufw allow 30000:40000/udp   >/dev/null 2>&1 || true   # TURN relay
ufw allow 50000:60000/udp   >/dev/null 2>&1 || true   # LiveKit media (UDP)
# TLS TURN (TURNS) hozir YOQILMAGAN. Kelajakda livekit.yaml da `tls_port: 5349`
# ochilsa, shu qatorni ham yoqing:
# ufw allow 5349/tcp        >/dev/null 2>&1 || true   # TURNS (TLS)
# Caddy (darsly_net konteyneri) → host.docker.internal:7880 (host-network LiveKit signaling).
# Default DROP bo'lgani uchun docker→host ulanish bloklanadi; faqat RFC1918 docker
# diapazoniga ochamiz (public internetga 7880 OCHILMAYDI — signaling faqat wss orqali).
ufw allow from 172.16.0.0/12 to any port 7880 proto tcp >/dev/null 2>&1 || true
echo ">> UFW holati:"; ufw status | grep -E '80|443|7881|3478|50000' || true

echo ">> Tayyor. Endi: docker compose up -d --build"
echo ">> (egress konteyneri ham shu buyruq bilan ko'tariladi — alohida profil kerak emas)"
exit 0
