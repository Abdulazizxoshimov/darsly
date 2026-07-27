# Darsly — "Hammasi 10/10" Mukammallik Rejasi

**Maqsad:** past-kechikish **7→10**, Zoom-funksionallik **58%→100%**, past-internet **4→10**, masshtab **4→10**.
**Asos:** 3 senior-agent auditi (latency / Zoom-parity / scale). Har vazifa aniq topilmaga bog'langan.

---
## ✅ BAJARILGAN (2026-07-23, sinovdan o'tgan)
- **A1** WebSocket Redis pub/sub fan-out — ko'p-instans real-time (fanout_test: A→B yetadi, double-delivery yo'q).
- **A3** Dockerfile Go 1.26.
- **B1** Ishtirokchilar ro'yxati API (`GET /lessons/:id/participants`).
- **B2** Host control — `mute-all`, `mute/:identity`, `remove/:identity` (kick) — LiveKit RoomService.
- **C5** Ochiq MinIO endpoint (presigned URL alohida public-client bilan).
- **C1-C3** Deployment config: `livekit.prod.yaml` (TURN-TLS/443, external_ip), `Caddyfile` (wss), `docs/DEPLOYMENT.md`. *(deployment — real domen/sert bilan sinaladi)*
- **D1** `EnsureRoom`ni join hot-path'dan chiqarish (Redis "room-ready" flag).
- **D2** HTTP gzip compression (WS/metrics istisno).
- **D3** Mentor-name Redis cache (join hot-path DB o'qishini kamaytiradi).
- **A4** Async email — RabbitMQ QueuedSender + `worker/email.go` (SMTP request oqimidan ajratilgan).
- **B4** Chat — domen (persist + history + LiveKit `SendData` broadcast), `POST/GET /lessons/:id/chat`, migratsiya 000007, testlar.
- **E4** Metrikalar — `darsly_ws_active_connections` gauge + room-tokens/waitingroom-decisions/recordings/notifications counterlar.
- **B8** Polls — domen (create/vote/results/close), guest ovoz LiveKit room-token orqali (`livekit.VerifyToken`), 1 kishi 1 ovoz (upsert), migratsiya 000008, testlar.

Holat: 56 test, 15 paket, build/vet/gofmt toza.

## 🔜 QOLGAN
- **A2** WithTx refactor (Register — hozir kompensatsiya/saga bilan ishlaydi, past-prioritet)
- **B3** co-host · **B6** recurring (RRULE expand) · **B7** breakout · **B9** student reminder (enrollment kerak) · **B5** reaksiya (asosan frontend)
- **C1-C3** deployment sinovi (real domen/sert) · **C4** audio-only fallback (frontend+token)
- **D4-D6** DB tuning/HTTP2/region · **E1-E3,E5** massiv scale infra (multi-node, replica, load/chaos)
---

**Belgilar:** `[KOD]` = sandboxda bajariladi · `[DEPLOY]` = real domen/sertifikat/infra kerak · `10/10 mezoni` = qachon "bajarildi".

---

## FAZA A — Masshtab & Real-time poydevori → **scale 10/10**
> Eng kritik arxitektura blocker'i. Busiz ko'p-instans real-time ishlamaydi.

### A1. WebSocket Hub'ni Redis pub/sub bilan ko'p-instansga ulash `[KOD]` — **KRITIK**
- **Muammo:** `Hub` butunlay in-memory; `cache.Publish/Subscribe` interfeysi bor-u ishlatilmaydi. Ko'p-instansda mentorning "yangi guest" signali yo'qoladi.
- **Yechim:** Har instans startda Redis kanaliga (`ws:fanout`) `Subscribe` qiladi. `Send(userID,msg)` → xabarni `{target:userID, msg}` sifatida Redis'ga `Publish`. Har instans o'z lokal `clients[userID]`iga yetkazadi. `newMsg` allaqachon serializable. Idempotent — o'z instansi ham subscribe orqali oladi (yoki lokalni ham darhol yuborib, o'zini skip qilish uchun instans-ID qo'shiladi).
- **10/10 mezoni:** 2 Hub + bitta Redis bilan test — instans A'dagi userga instans B'dan yuborilgan xabar yetadi. Yangi `hub_multiinstance_test.go`.

### A2. Ko'p-yozuvli oqimlarni `WithTx` ga o'rash `[KOD]`
- **Muammo:** `WithTx` yozilgan, 0 chaqiruv; `auth.Register` qo'lda saga-kompensatsiya.
- **Yechim:** Repolarni `Querier` (pool|tx) qabul qiladigan qilib moslash; `Register` (user+refresh) bitta tranzaksiyada. Kelajakdagi breakout/enrollment ham.
- **10/10 mezoni:** Register atomik; kompensatsiya kodi olib tashlanadi; integration test rollback'ni tekshiradi.

### A3. Dockerfile Go 1.26 `[KOD]`
- **Muammo:** `Dockerfile` `golang:1.22`, `go.mod` `go 1.26` — Docker build sinishi mumkin.
- **10/10 mezoni:** `docker build` toza o'tadi; CI'da docker-build bosqichi.

### A4. Async ish navbati (RabbitMQ) `[KOD]`
- **Muammo:** `mq` ulanadi-yu `_ = mq`; email sinxron (SMTP kutish request latency'ga ta'sir).
- **Yechim:** email worker + queued sender; recording.finished event; og'ir ishlar (RRULE expand, breakout) navbatga.
- **10/10 mezoni:** email async; worker ishlaydi; test bor.

### A5. Multi-instans deploy tayyorligi `[DEPLOY]`
- docker-compose `replicas`, nginx (WS Redis-backed bo'lgach sticky shart emas, lekin belt-and-suspenders), health/ready probe.
- **10/10 mezoni:** 3 instans LB ortida, real-time hamma instansda ishlaydi.

---

## FAZA B — Zoom "yig'ilish ichidagi" funksiyalar → **Zoom 100%**
> Backend mas'uliyatidagi yetishmayotgan funksiyalar. Media/screen-share/virtual-fon frontend+LiveKit-native — backend shart emas.

### B1. Ishtirokchilar ro'yxati API `[KOD]` — **eng arzon g'alaba**
- Infra tayyor (`livekit.ListParticipants`). `GET /lessons/:id/participants` (host) — kim ulangan, mute/video holati, rol.
- **10/10 mezoni:** host real-vaqt ishtirokchilar ro'yxatini oladi; test.

### B2. Host boshqaruv endpointlari `[KOD]`
- **Muammo:** host token'da `RoomAdmin` grant bor, lekin xavfsiz server-side API yo'q (faqat `EndLesson`).
- **Yechim:** LiveKit `RoomService` orqali: `POST /lessons/:id/participants/:identity/mute` (MutePublishedTrack), `/remove` (RemoveParticipant), `/mute-all`, `disable-video`. Ownership + host tekshiruvi.
- **10/10 mezoni:** host boshqa ishtirokchini server orqali mute/remove qiladi; test.

### B3. Co-host / moderator roli `[KOD]`
- Rol modeli `host | co-host | participant`; `POST /lessons/:id/participants/:identity/promote`; co-host token grant'i host-lite (mute/admit, lekin end-lesson yo'q). Waiting-room admit co-host'ga ham.
- **10/10 mezoni:** host co-host tayinlaydi; co-host admit/mute qila oladi; test.

### B4. Chat (guruh + shaxsiy) `[KOD]`
- **Yechim:** efemer yetkazish LiveKit data-channel (frontend); backend **tarixni** saqlaydi (`chat_messages` jadval: lesson_id, sender, body, private_to, created_at) + `GET /lessons/:id/chat` (tarix) + `POST` (persist + WS/data-channel broadcast). Shaxsiy (DM) `private_to` bilan.
- **10/10 mezoni:** xabar saqlanadi, tarix qaytadi, real-time yetkaziladi; test.

### B5. Qo'l ko'tarish + reaksiyalar `[KOD]`
- Holatni (`hand_raised`) participant-state'da saqlash + WS broadcast; reaksiyalar efemer (data-channel) yoki qisqa persist.
- **10/10 mezoni:** qo'l ko'tarilgani host ro'yxatida ko'rinadi; test.

### B6. Takrorlanuvchi darslarni expand qilish `[KOD]`
- **Muammo:** `RecurrenceRule` (RRULE) faqat string; instanslar yaratilmaydi.
- **Yechim:** `rrule-go` bilan keyingi N ta instansni materializatsiya (worker orqali) yoki ko'rinish vaqtida generatsiya; har instans o'z `join_slug`i.
- **10/10 mezoni:** haftalik dars kelasi hafta instansini avtomatik yaratadi; test.

### B7. Breakout rooms `[KOD]`
- Sub-xonalar yaratish, ishtirokchilarni taqsimlash (yangi token + room move), taymer, "hammaga qaytish" broadcast.
- **10/10 mezoni:** host 3 breakout yaratadi, ishtirokchilarni ko'chiradi, qaytaradi; test.

### B8. So'rovnoma (polls) / viktorina `[KOD]`
- `polls` + `poll_votes` jadval; create/vote/results; real-time natija (WS).
- **10/10 mezoni:** poll yaratish→ovoz→natija oqimi; test.

### B9. O'quvchilarga ham eslatma `[KOD]`
- **Muammo:** reminder faqat mentorga. O'quvchilar guest — hisob yo'q.
- **Yechim:** enrollment (ixtiyoriy ro'yxatdan o'tgan o'quvchilar) yoki email-list; ularga ham WS/email eslatma.
- **10/10 mezoni:** ro'yxatdagi o'quvchi eslatma oladi; test.

---

## FAZA C — Past-internet media transporti → **past-internet 10/10**
> Backend/WS qatlami allaqachon chidamli (~8/10). Blocker — media deployment konfiguratsiyasi. `[DEPLOY]` ustun.

### C1. TURN over TLS (443) — **KRITIK** `[DEPLOY+KOD]`
- **Muammo:** TURN faqat UDP/3478; TLS-TURN(443) o'chirilgan. Simmetrik NAT / UDP-bloklangan / mobil-korporativ tarmoq faqat 443'ni o'tkazadi.
- **Yechim:** `livekit.yaml` `turn: {tls_port:5349, domain, external_tls:true}` yoki alohida `coturn` (turns://:443); sertifikat (Let's Encrypt). 443'ni ochish.
- **10/10 mezoni:** faqat 443/tcp ochiq tarmoqni simulyatsiya qilib (firewall) klient media ulanadi (force-relay test).

### C2. `use_external_ip` / `node_ip` `[DEPLOY]`
- **Muammo:** `false` → internetdagi klient media portlariga yeta olmaydi.
- **10/10 mezoni:** LAN'dan tashqaridagi klient ulanadi.

### C3. `wss://` signaling `[DEPLOY]`
- Reverse-proxy (Caddy/nginx) + TLS; `LIVEKIT_HOST=wss://`; HTTPS frontend mixed-content'siz.
- **10/10 mezoni:** signaling 443/wss ustidan; mobil-proksi uzmaydi.

### C4. Adaptive / low-bandwidth rejim `[KOD+FRONTEND]`
- Simulcast/dynacast/adaptive-stream (LiveKit default yoniq — tasdiqlash); **audio-only fallback** juda past tarmoqda; backend token'da layer preferensiyalari.
- **10/10 mezoni:** past bandwidth'da avtomatik past sifatga/audio-only'ga tushadi.

### C5. Ochiq MinIO endpoint / CDN `[DEPLOY+KOD]`
- **Muammo:** presigned URL ichki `MINIO_ENDPOINT`ga imzolanadi — tashqi klient yeta olmaydi.
- **Yechim:** `MINIO_PUBLIC_ENDPOINT` sozlamasi (presign uchun ochiq domen+TLS); yozuvlar oldiga CDN.
- **10/10 mezoni:** tashqi tarmoqdan yozuv yuklab olinadi.

### C6. WS deadline zaxirasi `[KOD]`
- ping 50s / read 60s → read 75s (jitterli mobil tarmoqda noto'g'ri uzilishni kamaytirish).
- **10/10 mezoni:** yuqori-jitter simulyatsiyada uzilish kamayadi.

---

## FAZA D — Kechikishni minimallashtirish → **latency 10/10**

### D1. `EnsureRoom`ni join hot-path'dan chiqarish `[KOD]`
- **Muammo:** har join'da sinxron LiveKit `CreateRoom` — SFU'ga RTT qo'shadi.
- **Yechim:** xonani `HostToken` (dars live bo'lganda) yoki `lesson.Create` da bir marta yaratish; join'da Redis "room-ready" flag bilan tekshirish.
- **10/10 mezoni:** join hot-path'da LiveKit API chaqiruvi yo'q.

### D2. HTTP compression `[KOD]`
- gzip/br middleware (yoki nginx `gzip on`).
- **10/10 mezoni:** JSON javoblar siqiladi.

### D3. `toPublic` mentor-name cache `[KOD]`
- Har join'da `userRepo.GetByID(mentor)` o'rniga Redis cache (yoki lesson'da denormalizatsiya).
- **10/10 mezoni:** join'da ortiqcha DB o'qishi yo'q.

### D4. DB & Redis tuning `[KOD]`
- Per-query `statement_timeout`, prepared statements, indekslarni `EXPLAIN` bilan tekshirish, pool tuning; Redis pipelining hot-path'da.
- **10/10 mezoni:** yuk ostida p95 barqaror.

### D5. HTTP/2 + keep-alive `[DEPLOY]`
- Reverse-proxy HTTP/2; keep-alive tuning.

### D6. Region-aware (ilg'or) `[DEPLOY]`
- Foydalanuvchini eng yaqin LiveKit node'ga; edge signaling.
- **10/10 mezoni:** p95 media-connect < 1s, join API < 150ms (SLO test).

---

## FAZA E — Massiv masshtab & operatsiya → **scale 10/10 (barqaror)**
- **E1.** LiveKit multi-node + Redis, region routing `[DEPLOY]`
- **E2.** Egress autoscale (worker pool) `[DEPLOY]`
- **E3.** DB read-replica + PgBouncer `[DEPLOY]`
- **E4.** Observability: WS + biznes metrikalari (faol ulanish, xona a'zolari, drop, token, admit), OpenTelemetry tracing, Grafana dashboard + alert `[KOD]`
- **E5.** Load/SLO test (k6 + livekit-cli), chaos (node/DB/Redis uzilishi), DR `[KOD+DEPLOY]`
- **10/10 mezoni:** 100 dars × 30 ishtirokchi (3000 concurrent) N-instansda linear scale; SLO'lar ta'minlanadi.

---

## FAZA F — Muhandislik silliqlash `[KOD]`
- **F1.** JiraFlow merosini tozalash (module string, DB default `jiraflow`, grafana `jiraflow-logs.json`, izohlar).
- **F2.** DRY (`ownedLesson` umumiy helper), casbin `model.conf` embed, magic-number konstanta.
- **F3.** Testlar: WS Hub (concurrency+multi-instance), room/user usecase; coverage gate CI.
- **F4.** Hujjatlar: README, ARCHITECTURE.md, runbook; swagger regen; govulncheck CI.

---

## Ketma-ketlik va bog'liqliklar

```
A (scale poydevor) ──► B (Zoom funksiyalar) ──► D (latency)
     │                      │
     │                      └── B chat/host-control WS Hub'ga tayanadi → A1 avval
     └── A1 (WS Redis) hamma real-time funksiya uchun poydevor

C (past-internet media) — mustaqil, DEPLOY — parallel boradi (real domen kerak)
E (massiv scale) — A,B,C,D dan keyin
F (silliqlash) — davomida parallel
```

## Nima sandboxda, nima deploymentда
| Faza | Sandbox `[KOD]` | Deployment `[DEPLOY]` |
|------|-----------------|------------------------|
| A | A1,A2,A3,A4 | A5 |
| B | B1–B9 hammasi | — |
| C | C4,C5(kod),C6 | C1,C2,C3,C5(infra) |
| D | D1,D2,D3,D4 | D5,D6 |
| E | E4,E5(test) | E1,E2,E3,E5(infra) |
| F | hammasi | — |

## Tavsiya qilingan boshlanish
1. **A1 (WS Redis pub/sub)** — hamma real-time funksiya (chat, host-control, breakout) va masshtab uchun poydevor. **Birinchi shu.**
2. **B1+B2+B3 (ishtirokchilar ro'yxati + host control + co-host)** — Zoom-to'liqlikka eng katta sakrash, infra qisman tayyor.
3. **B4 (chat)** — MVP uchun kritik.
4. **C (past-internet media)** — deployment bosqichida, real domen bilan.

## Baho maqsadlari
| O'lcham | Hozir | Faza A–B dan keyin | Faza C–D dan keyin | Faza E dan keyin |
|---------|-------|--------------------|--------------------|------------------|
| Latency | 7 | 8 | **10** | 10 |
| Zoom-funksionallik | 58% | **95%** | 98% | **100%** |
| Past-internet | 4 | 5 | **10** | 10 |
| Masshtab | 4 | **8** | 9 | **10** |
