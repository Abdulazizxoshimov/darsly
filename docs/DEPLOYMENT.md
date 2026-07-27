# Darsly — Production Deployment (past-internet uchun)

"Yomon internetdagi o'quvchi ham video bilan qo'shila olishi" talabini yopadigan
deployment qatlami. Backend kodi tayyor — quyidagilar **infratuzilma** sozlamasi.

## 1. LiveKit media transporti (KRITIK)

`services/livekit/livekit.prod.yaml` ni ishlating:
- **`use_external_ip: true`** (yoki `node_ip: <ochiq IP>`) — aks holda internetdagi klient media portlariga yeta olmaydi.
- **TURN yoqilgan** (`turn.enabled: true`):
  - `udp_port: 3478` — tez yo'l (UDP ochiq tarmoqlar).
  - `tls_port: 5349` + **443 ustidan TURNS** — simmetrik NAT / UDP-bloklangan / korporativ-mobil firewall faqat 443'ni o'tkazadi. **Bu — yomon internet uchun eng muhim yo'l.**
- UDP diapazon `50000-60000` ochilgan (firewall/security-group'da).

## 2. `wss://` signaling

Mobil/korporativ proksilar shifrsiz `ws://`ni uzadi; HTTPS frontend `ws://`ni bloklaydi (mixed-content).
`services/livekit/Caddyfile` (avtomatik Let's Encrypt TLS):
- `wss://livekit.darsly.uz` → LiveKit 7880
- `https://api.darsly.uz` → backend 8080 (WebSocket ham shu orqali)
- `https://files.darsly.uz` → MinIO 9000

Backend `.env`: `LIVEKIT_HOST=wss://livekit.darsly.uz`.

## 3. Ochiq MinIO (yozuv yuklab olish)

Presigned URL imzosi host'ni qamraydi — tashqi klient yeta oladigan domenga imzolanishi shart:
```
MINIO_PUBLIC_ENDPOINT=files.darsly.uz
MINIO_PUBLIC_USE_SSL=true
```
(Backend'da C5 — alohida presign-client sifatida amalga oshirilgan.)

## 4. TrustedProxies (real IP)

Caddy/LB ortida:
```
TRUSTED_PROXIES=<LB/proksi CIDR>   # masalan 10.0.0.0/8
```

## 5. Ko'p-instans (horizontal scale)

WebSocket real-time Redis fan-out orqali ishlaydi (A1 bajarilgan) — sticky-session shart emas.
`docker compose` da `app` replicas'ni oshiring; hamma instans bir Redis'ga ulanadi.

## 6. Masshtab (1000-10000 talaba) — INFRA talablari

Backend **kodi** stateless va scale-ready, lekin quyidagi infra bo'lmasa 1000+ talabada
to'siladi (audit P1). Bular kod emas, deploy sozlamalari:

### 6.1 PgBouncer (MAJBURIY 3+ instansda)
Har instans `DB_MAX_CONNS=50` (default). PostgreSQL default `max_connections=100` →
atigi ~2 instans poolni to'ldiradi. 10000 talaba uchun:
- **PgBouncer** (transaction pooling) app ↔ PostgreSQL orasiga qo'ying.
- App poolini kichraytiring: `DB_MAX_CONNS=20`, PgBouncer ↔ PG'ga 100-200 real conn.
- Yoki PostgreSQL `max_connections`ni `instans_soni × DB_MAX_CONNS + zaxira`dan katta qiling.

### 6.2 Multi-node LiveKit (MAJBURIY 1000+ subscriber'da)
Bitta node 10000 subscriber × ~1.5 Mbps ≈ **15 Gbps** egress'ni ko'tara olmaydi.
- Bir nechta `livekit-server` node + **umumiy Redis** + LB (Caddy/NLB).
- Node soni ≈ kutilayotgan_subscriber / ~1500.
- `backend`: `LIVEKIT_HOST=wss://sfu.darsly.uz` (LB manzili) — LiveKit node-routing'ni
  Redis orqali ichki bajaradi, backend kodi o'zgarmaydi.
- Har node'da `use_external_ip: true` yoki `node_ip` to'g'ri.

### 6.3 Frontend SDK kontrakti (past-internet uchun HAL QILUVCHI)
Simulcast/Dynacast/AdaptiveStream **LiveKit server'da default yoqilgan**, lekin haqiqiy
foyda **klient SDK**'da yoqilishiga bog'liq. Frontend'da MAJBURIY:
- **Subscriber**: `adaptiveStream: true`, `dynacast: true` — tarmoqqa qarab avto-sifat.
- **Publisher (host)**: `simulcast: true` + sifat-qatlamlar.
Bularsiz 10000 past-internetli talaba host'ning full-res oqimini oladi va uziladi.

### 6.4 Redis HA
Redis hozir SPOF. Prod'da **Sentinel yoki Cluster**; `REDIS_POOL_SIZE`ni yuklama bo'yicha
sozlang (default 0 = go-redis 10×GOMAXPROCS).

### 6.5 Reminder worker — leader-election (ixtiyoriy)
Har instans reminder worker ishga tushiradi. `ClaimReminder` atomik → **dublikat push yo'q**,
lekin N× ortiqcha DB skani. Ko'p instansda alohida cron-worker yoki leader-election afzal.

### 6.6 Ataylab qoldirilgan
- **GetBySlug kesh**: join hot-path'да lesson kesh qilinmadi — `entity.Lesson.PasscodeHash`
  `json:"-"` bo'lgani uchun Redis JSON round-trip'ida yo'qoladi (passcode-bypass xavfi) +
  stale `is_locked`/`status` xavfsizlik oynasi. GetBySlug — indekslangan unique lookup (arzon);
  qimmat SFU chaqiruvi allaqachon singleflight bilan hal qilingan.
- **WS fanout bitta kanal**: hozir dars ichidagi chat/video LiveKit data-channel orqali ketadi,
  WS fanout faqat mentor bildirishnomalari + waiting-room (past hajm). 10k+ ulanish + yuqori
  xabar tezligida `ws:fanout:<hash(userID)%N>` shard qilinsin.

## Tekshirish (past-internet simulyatsiyasi)

1. Klient tarmog'ida UDP'ni bloklab (faqat 443/tcp) test qiling — TURNS orqali media ulanishi kerak.
2. `livekit-cli` yoki brauzer `chrome://webrtc-internals` da ICE nomzodlari `relay` (TURN) ekanini tasdiqlang.

## Xulosa

Bu 4 sozlama (external-ip, TURNS/443, wss, ochiq MinIO) qo'llanilganda backend
yomon internetdagi foydalanuvchiga past-kechikishli video beradi. Backend kodi
buni to'liq qo'llab-quvvatlaydi; qolgani — DNS + sertifikat + firewall sozlamasi.
