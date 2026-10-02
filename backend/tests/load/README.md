# Load testlar

Bu papka ikki xil yukni o'lchaydi — ularni ARALASHTIRMANG:

| Nima o'lchanadi | Vosita | Bo'g'iq joy |
|---|---|---|
| **HTTP token berish tezligi** (join/s) | k6 skriptlari, `livekit_load` | backend, DB, Redis, WS-hub |
| **SFU media fan-out** (1 pub → N sub video) | `sfu_media_load` (Go) | LiveKit SFU CPU/bandwidth |

> ⚠️ **Audit topilmasi (2026-08):** avvalgi suite faqat *token berish tezligini*
> isbotlagan. Haqiqiy bo'g'iq — SFU FAN-OUT — hech qachon ~50 passiv ulanishdan
> nariga yuklanmagan (`livekit_load` default `n=20`, nil track callback = obuna
> YO'Q, dekodlash YO'Q). Real xona sig'imi **~60–100 ishtirokchi/xona (200 EMAS)**,
> va u ham hali TASDIQLANMAGAN. `sfu_media_load` aynan shu bo'shliqni yopadi.

---

## 1. LiveKit guruh darsi — token yo'li (Go, eski)

Backend (`:8087`) va LiveKit (`:7880`) ishlab turgan holda:

```bash
cd backend
go run ./tests/load/livekit_load -n 20
go run ./tests/load/livekit_load -n 50 -base http://localhost:8087
```

N ta guest'ni backend joinlink orqali token oldirib LiveKit xonasiga ulaydi va
serverdan ACTIVE ishtirokchi sonini tekshiradi. **Diqqat:** subscriber'lar media
OLMAYDI (passiv ulanish) — bu SFU fan-out'ni o'lchamaydi. Buning uchun 4-bo'lim.

## 2. API load (k6)

`k6` o'rnatilgan bo'lsa. Avval parolsiz, waiting-room OFF, **jonli** dars yarating
va uning `join_slug`ini oling:

```bash
k6 run -e BASE=http://localhost:8087 -e SLUG=<join_slug> tests/load/api_load.js
```

## 3. Sig'im scenario'lari (k6)

- `scenario_a_big_group.js` — bitta darsga 200→2000 join/s.
- `scenario_b_parallel_lessons.js` — 60 darsga taqsimlangan yuk (`slugs.json` kerak).
- `scenario_c_spike.js` — 30s da 0→1000 join/s keskin sakrash.
- **`scenario_d_degradation_ramp.js` (YANGI, L-4/L-5)** — 50→3000 join/s uzluksiz
  rampa. Maqsad: pass/fail EMAS, **tizzani (knee)** topish — latency p95/p99 va
  xato nisbati qaysi join/s'da sakraydi. Thresholds ATAYIN yumshoq.

```bash
k6 run -e BASE=http://localhost:8087 -e SLUG=<slug> tests/load/scenario_d_degradation_ramp.js
# tizzani aniq kesish uchun:
k6 run --out json=ramp.json -e BASE=... -e SLUG=... tests/load/scenario_d_degradation_ramp.js
```

Barcha scenario'lar server `TRUSTED_PROXIES=0.0.0.0/0` bilan ishga tushirilishini
talab qiladi (har iteratsiya unikal `X-Forwarded-For` → per-IP rate-limit chetlab
o'tiladi, sof server sig'imi o'lchanadi).

---

## 4. SFU MEDIA fan-out — `sfu_media_load` (Go, YANGI, KRITIK)

Har xonada **1 ta haqiqiy publisher** (audio + video, belgilangan bitrate) va
**N ta haqiqiy subscriber** — auto-subscribe YOQILGAN, `OnTrackSubscribed`
callback RTP paketlarini o'qib **haqiqatan media OLADI** (kadr/bayt sanaydi).
Bu SFU'ning asosiy yukini — 1 manbani N ta obunachiga uzatishni — modellaydi.

Token to'g'ridan-to'g'ri LiveKit API kalit/sekret bilan mint qilinadi (backend
rate-limit'iga bog'lanmaydi). LiveKit `:7880` ishlab turishi kifoya (backend shart emas).

### L-1 — bitta xona media sig'imi
```bash
cd backend
go run ./tests/load/sfu_media_load -n 60 -dur 60s
go run ./tests/load/sfu_media_load -n 100 -dur 90s -video-kbps 800    # sig'im chetini sinash
```
Chiqish: `ulangan=`, `media=` (haqiqatan qabul qilayotgan subscriber soni),
`paket/s`, `Mbit/s`, va yakuniy jami paket/bayt.

### L-2 — ko'p xona agregat (~1000 ishtirokchi)
```bash
go run ./tests/load/sfu_media_load -rooms 16 -n 80 -dur 120s   # 16×80 = 1280 sub
```
Agregat SFU CPU/bandwidth uchun. **Lekin bitta jarayonda 1000+ WebRTC peer OG'IR**
(pastdagi ogohlantirish). Amaliyroq — xonalarni jarayon/mashinaga bo'ling:
```bash
# turli terminal yoki mashinalarda:
go run ./tests/load/sfu_media_load -rooms 1 -n 80 -room-offset 0 -dur 120s
go run ./tests/load/sfu_media_load -rooms 1 -n 80 -room-offset 1 -dur 120s
# ... room-offset 2..15
```
SFU'ni bir vaqtda `docker stats` / LiveKit Prometheus bilan kuzating (5-bo'lim).

### L-3 — reconnect storm (blip / SFU restart)
```bash
go run ./tests/load/sfu_media_load -n 60 -reconnect -reconnect-cycles 3 -dur 60s
# tarqoq qayta ulanish (thundering herd yumshoqroq):
go run ./tests/load/sfu_media_load -n 60 -reconnect -reconnect-stagger 20ms
```
N ta media klientni ulaydi, keyin HAMMASINI bir vaqtda uzadi + qayta ulaydi,
va **tiklanish vaqtini** o'lchaydi: `qayta_ulanish=` (hammasi qayta ulanguncha)
va `media_qaytdi(80%)=` (80% subscriber yana media olguncha).

### Muhim flaglar
`-rooms`, `-n`, `-dur`, `-warmup`, `-report`, `-video-kbps`, `-fps`,
`-publish-video`, `-batch`/`-batch-delay` (to'lqin-to'lqin ulanish),
`-room-offset` (ko'p jarayonda xona nomi to'qnashmasin),
`-lk`/`-lk-key`/`-lk-secret`.

### ⚠️ KLIENT O'ZI BO'G'IQ JOY (halol ogohlantirish)
Bitta yuk JARAYONI **~150 dan ortiq WebRTC peer'ni** ko'tara olmaydi:
ICE/DTLS handshake'lari o'zaro raqobatga tushadi (bu `livekit_load` da empirik
hujjatlangan — 150 dan faqat ~35 tasi ulanardi, server CPU'si esa 1% da turardi).
Ya'ni yuqori sonlarda o'lchov SERVERNI emas, VOSITANI o'lchab qo'yadi. Shuning
uchun 1000 media klientga yetish uchun **bir necha jarayon/mashinadan** foydalaning
(L-2). `-batch`/`-batch-delay` bitta jarayon ichida ham bosimni yumshatadi.

---

## 5. Nimani o'lchash (yon-ma-yon kuzatuv)

Yuk ketayotganda quyidagilarni PARALLEL kuzatib, birinchi to'yingan qatlamni toping:

- **SFU (LiveKit):** `docker stats` (services/livekit), LiveKit Prometheus
  (`livekit_room_participant`, CPU, `livekit_*_bytes`). Media testda birinchi
  navbatda shu.
- **Backend API:** k6 chiqishidagi `join_latency` p95/p99 va `join_errors`
  (scenario D), yoki `/metrics` Prometheus.
- **PostgreSQL:** `SELECT count(*) FROM pg_stat_activity;` — faol ulanishlar pool
  chegarasiga tegdimi; `pg_stat_activity` da `wait_event`; sekin so'rovlar.
- **Redis:** `redis-cli -p 6399 INFO clients` (connected_clients), `INFO stats`
  (instantaneous_ops_per_sec), `SLOWLOG GET`.
- **RabbitMQ:** `:15682` management UI yoki `rabbitmqctl list_queues name messages`
  — navbat o'sib ketsa consumer orqada qolayapti.
- **WS-hub:** faol WebSocket ulanishlar soni (waitingroom push); backend log/metrics.
- **Tizim:** `docker stats`, `ss -s` (socket/FD soni), UDP drop (`nstat -az | grep -i udp`).

### Kutilayotgan shift (halol)
- **~60–100 media ishtirokchi/xona** — loyiha xotirasidagi baho. **200 EMAS.**
  Hali uchdan-uchga TASDIQLANMAGAN; `sfu_media_load` bu chegarani o'lchash uchun.
- 1 SFU jarayonining jami sig'imi ~ (xonalar × ishtirokchi) — CPU va yuqori
  bo'lganda upstream bandwidth bilan chegaralanadi.

### Real-scale (TURN / yomon internet) ogohlantirishlar — CLAUDE.md dan
Production/uzoq masofa sinovida media TURN relay orqali o'tadi. UFW da OCHIQ bo'lishi shart:
`3478/udp` (TURN allokatsiya), **`30000-40000/udp` (TURN relay — MAJBURIY, osongina
o'tkazib yuboriladi)**, `50000-60000/udp` (LiveKit media), `7881/tcp` (RTC-TCP fallback),
`80,443/tcp` (Caddy+ACME). `30000-40000/udp` yopiq bo'lsa LiveKit "Starting TURN server"
deb yozadi, lekin media JIMGINA o'tmaydi. Relay yo'lini majburlash uchun
klientda `iceTransportPolicy:'relay'`.

---

## Bu papkada NIMA ISHGA TUSHIRILDI (halol holat)

- `go build ./tests/load/...` va `go vet ./tests/load/sfu_media_load/...` — **TOZA** (tekshirildi).
- k6 skriptlari — **sintaksis tekshirildi** (ESM parse), lekin bu muhitda k6 O'RNATILMAGAN,
  shuning uchun to'liq k6 yugurishi bajarilmadi.
- **`sfu_media_load` real LiveKit'ga qarshi ishga TUSHIRILMADI** (bu muhitda SFU
  ko'tarilmagan; 60–100 real media klient uchun `services/livekit` kerak). Yuqoridagi
  aniq buyruqlar bilan SFU ko'tarilgan muhitda ishga tushiring.
- **1000 concurrent media HALI TASDIQLANMAGAN.** Yuqoridagi ~60–100/xona shift —
  baho, o'lchov emas.
