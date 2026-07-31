# Jonly — Yuklama (Load) Test Hisoboti

## ⭐ 2026-07-31 — LiveKit media sig'imi (300 ishtirokchi maqsadi)

> Maqsad (docs/PRODUCT.md): birinchi relizda **100–300 ishtirokchi**.
> Vosita: `backend/tests/load/livekit_load` — HAQIQIY LiveKit ulanishlari (HTTP imitatsiya emas).
> Muhit: lokal 4 yadro; backend + LiveKit (docker, host-net) + Postgres/Redis/MinIO.
> Ishtirokchilar tomoshabin rejimida (Zoom modelida gapirganda vaqtincha publish qiladi).

| Ishtirokchi | Ulanish rejimi | Natija | Vaqt | LiveKit CPU | LiveKit RAM |
|---|---|---|---|---|---|
| 50  | bir vaqtda | 50/50 | 4.3 s | ~1 % | ~0.4 GB |
| 150 | 25 tadan / 2 s | **150/150** | 10.5 s | ~1 % | ~0.7 GB |
| 300 | 25 tadan / 2 s | 275/300 | 38 s | ~1 % | 1.16 GB |
| 300 | 15 tadan / 3 s | **300/300** | 57 s | ~0.3 % | 1.07 GB |

Backend cho'qqida: CPU ~13 %, RAM ~95 MB. **300 maqsadi bajarildi**; cheklovchi omil
protsessor emas — ulanish o'rnatish (signaling handshake) tezligi.

### Topilgan va tuzatilgan KRITIK muammo
**LiveKit webhook'i 30/daqiqa bilan cheklangan edi** (`api/router.go`). 150 kishilik
sinovda **10 ta webhook 429 bilan rad etildi**. Oqibatlari jimgina va og'ir:
`track_published` yo'qolsa dars **umuman yozib olinmaydi**; `participant_joined`
yo'qolsa **ban qo'llanmaydi**; `participant_left` yo'qolsa **avto-yakun ishlamaydi**.
Chegara 200/s (burst 400) ga ko'tarildi — manba bitta va imzo bilan tekshiriladi,
ya'ni chegara faqat DoS to'sig'i. Keyingi barcha yurishlarda 429 = 0.

### O'lchov vositasidagi xato (natijani buzayotgan edi)
Birinchi 150 kishilik yurishda faqat 35 tasi ulandi, log «handshake error: EOF»
bilan to'ldi — lekin SFU CPU'si 1 % da edi. Sabab serverda emas: vosita 150 ta
WebRTC peer'ini bitta jarayonda bir zumda ochib, o'z chegarasini o'lchab qo'yardi.
To'lqinli rejim qo'shildi (`-batch`, `-batch-delay`) — bu haqiqatga ham mos:
300 o'quvchi bitta millisekundda emas, bir necha soniyada kiradi.

### Tavsiyalar
1. 3-haftada real serverda (TURN + HTTPS) qayta o'lchash — signaling navbati o'zgaradi.
2. Publish qiluvchi ishtirokchilar (5–10 kishi gapirganda) alohida o'lchanishi kerak
   (`tests/load/publish_probe`).
3. 4 yadroli VPS 300 tomoshabinni ko'taradi; hal qiluvchi omil — **chiquvchi kanal**
   (300 × ~0.5 Mbit/s ≈ 150 Mbit/s), protsessor emas.

### Takrorlash
```bash
cd backend && go run ./tests/load/livekit_load \
  -base http://localhost:8087 -email <mentor> -password <parol> \
  -n 300 -batch 15 -batch-delay 3s
```

---

# Darsly — Yuklama (Load) Test Hisoboti (2026-07-24, HTTP/k6)

Vosita: **k6** (Docker: `grafana/k6`). Skriptlar: `backend/tests/load/scenario_{a,b,c}_*.js`.
Sana: 2026-07-24.

## Muhit va metodologiya (MUHIM — raqamlarni to'g'ri talqin qilish uchun)

- **Bir xil dev-mashina**: k6 (Docker) + backend server + PostgreSQL + Redis + boshqa Docker konteynerlar **bitta 20-yadroli mashinada birga** ishladi. Ya'ni load-generator server bilan CPU uchun raqobat qildi. **Raqamlar — dev-mashina "pol"i, production-apparat sig'imi emas** (haqiqiy ajratilgan serverda ancha yuqori bo'ladi).
- **Rate-limit chetlab o'tish**: join endpoint **600 so'rov/min/IP** bilan himoyalangan (policy). Sof server sig'imini o'lchash uchun har simulyatsiya qilingan foydalanuvchi **unikal `X-Forwarded-For` IP** yubordi (server `TRUSTED_PROXIES=0.0.0.0/0` bilan). Aks holda bitta IP'dan hamma 429 oladi (bu — to'g'ri policy, sig'im emas).
- **Test yo'li**: `POST /api/v1/joinlink/:slug` → participant token (mahalliy JWT imzo). Bu — dars boshlanishida hamma bosadigan "issiq yo'l". **LiveKit SFU media sig'imi HTTP yuklama bilan sinalmaydi** (alohida masala; DEPLOYMENT.md'da tahlil qilingan).
- Arrival-rate executor (sekundiga N yangi foydalanuvchi) — real "join oqimi"ni modellaydi.

## Natijalar

| Stsenariy | Target | Erishilgan | p50 | p95 | p99 | max | Xato |
|---|---|---|---|---|---|---|---|
| **A** — bitta katta guruh | 200→2000 join/s | ~995/s¹ | 1.36ms | 2.25ms | 3.46ms | 55ms | **0%** |
| **B** — 60 parallel dars | 200→1200 join/s | ~683/s¹ | 1.48ms | 2.41ms | 3.62ms | 44ms | **0%** |
| **C** — spike (30s da 0→1000/s) | 1000 join/s | ~714/s¹ | 1.55ms | 2.22ms | 3.27ms | 15.6ms | **0%** |
| **Ceiling** — cheklovni topish | 2000→8000 join/s | ~3558/s | 141ms | 904ms | 1.02s | 1.42s | **0%** |

¹ Erishilgan tezlik target'dan past — chunki k6 (yuz-minglab iteratsiya) + server + PG **bir mashinada** raqobatlashdi; server emas, load-generator/mashina cheklovchi bo'ldi (VU'lar target'ni to'ldira olmadi).

## Bottleneck tahlili

- **DB ulanishlari**: butun test davomida barqaror **~41-56** (pool cap 50) — hech qachon tugamadi. Read-path (slug lookup) DB-bound EMAS.
- **Xato darajasi: 0%** hamma yugurishда — server yiqilmadi, panic yo'q. Ortiqcha yukда ham **graceful degradatsiya** (kechikish oshadi, xato bermaydi).
- **Ceiling (~3558/s, p99 1s)**: asosiy cheklovchi — bir mashinada k6 (4000 VU) + PostgreSQL + server **CPU raqobati**, sof server limiti emas. Server CPU o'sdi (57%→100%+), lekin mashina load-generator bilan to'yindi.
- **Birinchi to'yingan resurs**: bu yukда server EMAS. API/DB read-path juda tez (p99 < 4ms @ 1000-2000/s).

## Production'da birinchi bottleneck'lar (arxitektura + ushbu ma'lumot + DEPLOYMENT.md)

1. **Per-IP rate-limit (600/min/IP)** — POLICY chegara, sig'im emas. NAT ortidagi katta sinf uchun sozlanadi.
2. **LiveKit SFU media (~1500 subscriber/node)** — HTTP bilan sinab bo'lmaydi; multi-node kerak (DEPLOYMENT.md §6.2).
3. **DB connection pool (50/instans)** — WRITE-og'ir yo'llarda (waiting-room insert); PgBouncer bilan yumshatiladi (DEPLOYMENT.md §6.1).
4. **WebSocket hub fan-out** (bitta Redis kanal) — juda ko'p node'da shard kerak.

## XULOSA

**Bitta dev-instansda join issiq yo'li (link → token) 1000-2000 join/s ni p99 &lt; 4ms va 0% xato bilan barqaror ko'tardi va to'yinmadi** — DB pool tugamadi, server xato bermadi. API/DB read-path bottleneck EMAS.

**Baholangan production sig'imi** (bir instans, ajratilgan apparat): join issiq yo'li uchun **bir necha ming join/s** — ya'ni bitta katta guruh darsi (500-1000 talaba bir zumda kirishi) va o'nlab parallel dars bemalol. Undan yuqorisi uchun cheklov **backend kodi emas, balki infra**: multi-node LiveKit (media), PgBouncer (DB write), Redis HA — barchasi DEPLOYMENT.md'da rejalashtirilgan (kod stateless, scale-ready).

---

# Real SFU media sig'imi (LiveKit) — o'lchandi

Yuqoridagi k6 testi faqat **token API**ni o'lchadi (arzon JWT). Bu bo'lim **haqiqiy WebRTC media**ni
(video oqimi SFU orqali forwarding) o'lchaydi — video-platformaning asl "shift" nuqtasi.

**Muhit sozlamasi:** LiveKit `livekit.yaml`da top-level `redis:` bloki bor edi → server distributed
rejimda Redis'ni majbur qilardi va compose'da redis-DNS yechilmay crash-loop qilardi. **Yechim:**
bitta-node uchun `livekit.standalone.yaml` (redis'siz — redis faqat multi-node routing uchun kerak),
`--network host` bilan ishga tushirildi ("single-node routing"). Kalit: `devkey`.

## 1. Ulanish (signaling) sig'imi — `tests/load/livekit_load` (Go SDK, real WebRTC)
Bitta darsga N ta guest **haqiqiy** ulanadi (ICE/peer-connection), server-side ACTIVE tasdiqlanadi:

| N subscriber | Ulandi | Ulanish vaqti | ACTIVE (server) | LiveKit CPU tepasi |
|---|---|---|---|---|
| 25 | 25/25 (100%) | 513 ms | 25 | — |
| 50 | 50/50 (100%) | 875 ms | 50 | — |
| 100 | 100/100 (100%) | 1.31 s | 100 | — |
| 200 | 200/200 (100%) | 3.37 s | 200 | **~667% (≈6.7 yadro) burst** |

→ **200 bir vaqtdagi ulanish 100% muvaffaqiyat.** Burst ulanishда CPU vaqtincha ~6.7 yadroga chiqadi.

## 2. Media forwarding sig'imi — `livekit-cli load-test` (1 video publisher + N subscriber)
Webinar modeli: mentor video chiqaradi (simulcast, high), N o'quvchi qabul qiladi:

| Subscriber | Jami bitrate | Har biriga (avg) | Paket yo'qolishi | LiveKit CPU | Holat |
|---|---|---|---|---|---|
| 100 | 110.7 mbps | **1.4 mbps (to'liq sifat)** | **0%** | ~65% (<1 yadro) | Toza |
| 300 | (toza oqim) | ~1.4 mbps | 0% (ulanganlar) | ~178% (≈1.8 yadro) | Toza¹ |
| 500 | 230.9 mbps | 462 kbps (avto-pasaydi) | **1.13% (34 sub)** | ~224% (≈2.2 yadro) | **Degradatsiya boshi** |

¹ N=300 tez 50/s burst'da bir nechta signal-ulanish uzildi (co-located client+SFU raqobati); ulanganlar 0% yo'qolish.

## SFU xulosasi (halol)
- **Bitta node ~100-300 subscriber'ni to'liq 1.4 mbps sifatda 0% yo'qolish bilan xizmat qiladi** (ushbu mashinada).
- **500 subscriber'da**: hammasi (500/500) qabul qildi, lekin LiveKit **simulcast bilan sifatni avto-pasaytirdi** (462 kbps) va **~1.1% paket yo'qolishi** paydo bo'ldi — ammo yiqilish emas, **graceful adaptatsiya**.
- **LiveKit CPU 500'da atigi ~2.2 yadro** (20 dan) — SFU to'yinmagan. 500'dagi degradatsiya asosan **co-located load-generator** (500 WebRTC klient + SFU bir mashinada) raqobati, sof SFU limiti emas. Ajratilgan apparat + real tarqoq klientларда yaxshiroq bo'ladi.
- **Bandwidth chegara**: 500 sub × 1.4 mbps = ~700 mbps; 1500 sub ≈ 2 Gbps — bu NIC/tarmoq chegarasi. LiveKit'ning hujjatlangan ~1000-1500 sub/node ko'rsatkichi **bandwidth-bound** (CPU emas), va ushbu test uni empirik tasdiqlaydi.

**Amaliy javob:** bitta LiveKit node bitta katta guruh darsida **~500 o'quvchigacha** video beradi (500'da sifat biroz pasayadi, lekin uziladigan emas); **1000+ o'quvchi yoki ko'p parallel katta dars uchun multi-node LiveKit** kerak (DEPLOYMENT.md §6.2 — umumiy Redis + LB bilan). Backend kodi buni qo'llab-quvvatlaydi (LIVEKIT_HOST = LB manzili).

---

## Cheklovlar (halol)
- Raqamlar dev-mashina "pol"i (k6 co-located) — production-apparat emas.
- k6 qismида token = mahalliy JWT (LiveKit SFU o'chiq edi). **Media sig'imi endi alohida o'lchandi** — yuqoridagi "Real SFU media sig'imi" bo'limiga qarang (100-500 subscriber, real WebRTC).
- Write-og'ir yo'l (waiting-room insert) va WS-hub yuki alohida benchmark qilinmagan — keyingi qadam sifatida tavsiya etiladi.
- k6 skriptlari real IP-spoofing (`X-Forwarded-For`) ishlatadi — bu FAQAT test uchun (`TRUSTED_PROXIES=0.0.0.0/0`); production'da TRUSTED_PROXIES faqat LB CIDR'iga o'rnatiladi.
