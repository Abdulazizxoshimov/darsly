# Jonly — Kechikish va ekran-ulashish sifati (2026-07-31)

> Asoschi bahosi (`docs/PRODUCT.md`, «Ekran ulashish overlay'i va SIFAT maqsadi»):
> **kechikish juda ko'p**, **ekran ulashish sifati past**. Bu hisobot shu ikki
> gapni raqamga aylantiradi, sababini topadi va kiritilgan o'zgarishlarni
> oldin/keyin o'lchovi bilan ko'rsatadi.

## Bir qatorda

Zaif internetdagi o'quvchi uchun ekrandagi **matn o'qib bo'lmas darajada edi** —
sabab bitreyt emas, **qatlam narvonining tuzilishi**. Tuzatilgandan keyin
o'sha o'quvchida matn qirralarining **0.34 → 0.83** i saqlanadi (1.0 = asl
keskinlik), yaxshi kanaldagi o'quvchida esa kechikish **154 → 85 ms** ga tushdi.

| O'lchov (1080p ekran ulashish) | OLDIN | KEYIN |
|---|---|---|
| Zaif kanal — matn keskinligi | **0.34** (o'qilmaydi) | **0.83** |
| Zaif kanal — PSNR | 17.0 dB | 28.8 dB |
| Zaif kanal — o'lcham | 640×360 | 1920×1080 @ 3 fps |
| O'rta kanal (~800 kbps) | **qatlam yo'q** → 640×360 ga tushardi | 1920×1080 @ 8 fps, keskinlik 0.94 |
| Yaxshi kanal — kechikish p50 / p95 | 106 / 148 ms | **85 / 113 ms** |
| Yaxshi kanal — jitter-bufer | 43–72 ms | **5 ms** |
| Yaxshi kanal — sifat | 39.3 dB / 0.97 | 38.5 dB / 0.96 (o'zgarishsiz) |

---

## 1. O'lchov usuli

Vosita: **`frontend/tools/media-probe/`** (yangi). Nega mavjud vositalar yetmadi:
`tests/load/livekit_load` faqat ULANISHni, `publish_probe` esa faqat audio trekni
o'lchaydi — ikkalasi ham "ekrandagi matn o'qiladimi" degan savolga javob bermaydi.

Zond ikki brauzer konteksti ochadi:

- **Nashr qiluvchi** — canvas'ga slayd/kod chizadi (mayda monospace matn, ingichka
  chiziqli jadval, 1 px panjara, 10–20 px shriftlar), `captureStream` bilan uni
  ekran-ulashish trekiga aylantiradi va LiveKit'ga e'lon qiladi. Har kadrga
  **marker** yoziladi (yuqori chapdagi oq/qora bloklar): kadr raqami + `Date.now()`.
- **Obunachi** — dekodlangan kadrni oladi, markerdan kadr raqamini o'qiydi,
  **asl kadrni qayta chizadi** va ikkisini solishtiradi.

O'lchanadigan kattaliklar:

| Kattalik | Ma'nosi |
|---|---|
| **PSNR (dB)** | umumiy sodiqlik; matn uchun 30 dB dan past = ko'zga tashlanadi |
| **keskinlik** | Tenengrad gradient energiyasi nisbati (dekod/asl). 1.0 = asl. Bu PSNR ushlamaydigan narsani ushlaydi: **bir tekis xiralashuv** PSNR'ni ko'p tushirmaydi, lekin matnni o'qib bo'lmas qiladi |
| **kechikish** | "shishadan-shishagacha": kadr chizilgan paytdan ekranda ko'ringungacha |
| **TTFF** | qo'shilish tugmasidan ekranni ko'rgungacha |
| ICE yo'li, RTT, jitter, jitter-bufer, paket yo'qotishi, enkoder/dekoder vaqti | |

```bash
# frontend dev-server (3000) va LiveKit (7880) ishlab turishi kerak
cd frontend && node tools/media-probe/run.mjs --secs 12 --out /tmp/probe.json
node tools/media-probe/run.mjs --only shipped     # yetkazilgan sozlamaning o'zi
```

`shipped-*` stsenariylari `src/livekit/mediaTuning.js` ni **import qiladi** —
ya'ni hisobotdagi "keyin" ustuni kod bilan jimgina ajralib keta olmaydi.

**Muhit:** lokal, 20 yadro, LiveKit 1.13.4 (host-net), `livekit-client@2.21.0`,
Chromium 149 (dasturiy enkoderlar). ICE yo'li `host/host`, RTT ~1 ms — ya'ni
quyidagi kechikish raqamlari **tarmoqsiz pol**: real serverda RTT ustiga qo'shiladi.

---

## 2. Topilgan sabablar

### 2.1 ⭐ Asosiy sabab — narvonda faqat ikki pog'ona bor edi

`mediaTuning.js` ekran uchun **bitta** qo'shimcha qatlam berardi
(`screenShareSimulcastLayers: [SCREEN_LOW]`). SDK kodidan o'qildi
(`publishUtils`, taxmin emas): bitta preset berilsa **atigi ikki** qatlam yasaladi.
Natijada narvon shunday edi:

```
1920×1080 @ 15 fps — 1 500 kbps
   ... 200 va 1500 orasida HECH NARSA ...
  640×360 @  3 fps —   200 kbps
```

SFU o'quvchiga sig'adigan **eng yuqori** qatlamni beradi. Demak 800 kbps li
o'quvchi 1.5 Mbps ni ko'tara olmay to'g'ridan-to'g'ri eng pastkisiga tushardi:
kanalining uchdan ikki qismi ishlatilmay qolardi va matn yo'qolardi.

### 2.2 Pastki qatlam matnni O'LCHAM bilan o'ldirardi, bitreyt bilan emas

1080p → 360p = **3× kichraytirish**: 14 px shrift ~5 px ga aylanadi. Bu qaytarib
bo'lmaydigan yo'qotish. O'lchov buni tasdiqladi — o'sha 360p qatlamga bitreyt va
fps qo'shish natijani deyarli o'zgartirmadi:

| pastki qatlam (manba 1080p) | PSNR | keskinlik |
|---|---|---|
| 640×360 @ 3 fps / 200 kbps (eski) | 17.0 dB | 0.34 |
| 640×360 @ 10 fps / 250 kbps | 17.0 dB | 0.35 |
| 640×360 @ 15 fps / 400 kbps | 19.5 dB | 0.14 |
| 960×540 @ 5 fps / 300 kbps | 22.4 dB | 0.28 |
| 1280×720 @ 5 fps / 400 kbps | 23.5 dB | 0.39 |
| **1920×1080 @ 3 fps / 300 kbps** | **28.9 dB** | **0.83** |

Bir xil ~750 kbps byudjetda o'rta qatlam uchun ham xuddi shunday:

| o'rta qatlam | PSNR | keskinlik |
|---|---|---|
| 1280×720 @ 15 fps | 24.3 dB | 0.42 |
| 1600×900 @ 12 fps | 25.8 dB | 0.49 |
| **1920×1080 @ 8 fps** | **33.7 dB** | **0.93** |

**Xulosa:** slayd va kod uchun **piksel > kadr**. Kanal torayganda o'lcham emas,
kadr chastotasi qurbon bo'lishi kerak.

### 2.3 Trek ustuvorligi ("ekran > kamera") Chrome'da UMUMAN ishlamasdi

SDK kodi (`encodingsFromPresets`):

```js
canSetPriority = (browser === 'Firefox' && os !== 'iOS') || idx === 0
```

Chrome faqat **0-indeksdagi** (eng past) qatlamning ustuvorligini oladi. Kodda
`priority: 'high'` esa eng YUQORI qatlamga qo'yilgan edi — Chrome uni jimgina
tashlab yuborardi. Ya'ni hujjatlashtirilgan "ovoz > ekran > kamera" qoidasi
amalda **hech qachon qo'llanmagan**.

### 2.4 `maintain-resolution` kameraga ham tushardi

`degradationPreference: 'maintain-resolution'` **xona darajasidagi**
`publishDefaults` da edi, ya'ni kameraga ham qo'llanardi: zaif tarmoqda ustozning
yuzi kichraymasdan **muzlab qolardi**. Mobil ilova esa kamerada `BALANCED`
ishlatardi — ikki platforma bir-biriga zid xulq ko'rsatardi (garchi ikkala fayl
ham "biz bir xilmiz" deb yozilgan bo'lsa ham).

### 2.5 Jitter-bufer kechikishning yarmini yeyardi

Lokal RTT ~1 ms bo'lsa ham yuqori qatlamda jitter-bufer **43–72 ms** ushlab
turardi. Chrome ekran-ulashishda (past fps, katta kadrlar) buferni o'zicha
kattalashtiradi; LiveKit'da `room.playout_delay` bloki umuman yo'q edi.

---

## 3. Kodek: VP9/AV1 TEKSHIRILDI va RAD ETILDI (o'lchov bilan)

Vazifada "VP9/AV1 past bitreytda matn keskinligini oshiradi" degan taxmin bor edi.
Ekran-ulashish kontentida bu **tasdiqlanmadi** (1280×720 manba, bitta qatlam,
bir xil bitreyt chegarasi):

| chegara | VP8 | VP9 (L1T3) | AV1 (L1T3) | H.264 |
|---|---|---|---|---|
| 200 kbps | 29.1 dB · 14 fps · 114 ms | 26.8 dB · **1 fps** · 105 ms | 27.3 dB · 3 fps · 97 ms | 36.2 dB · 15 fps · 57 ms |
| 400 kbps | 39.4 dB · 106 ms | 38.9 dB · **234 ms** | 40.3 dB · 190 ms | 39.2 dB · 73 ms |
| 800 kbps | 43.5 dB · 86 ms | 43.6 dB · **200 ms** | 42.6 dB · 129 ms | 39.2 dB · 50 ms |
| 1500 kbps | 43.8 dB · 56 ms · **346 kbps sarfladi** | 44.0 dB · 86 ms · **1499 kbps sarfladi** | 42.9 dB · 80 ms | 39.1 dB · 52 ms |

VP8 saqlanadi, uch sabab bilan:

1. **VP9 past bitreytda kadr chastotasini 1 fps ga tushiradi** — ekran muzlagandek
   ko'rinadi. Bu aynan eng muhim (zaif kanal) holatda sodir bo'ladi.
2. **VP9/AV1 kechikishni 2–4 barobar oshiradi** (dasturiy enkod). Arzon Android
   telefonda (asosiy auditoriya) bu yanada yomonroq.
3. **VP9 kanalni to'liq yeydi**: 1.5 Mbps ruxsatdan 1499 kbps oldi, VP8 esa 346
   kbps bilan bir xil sifatga chiqdi. 300 o'quvchili darsda bu to'g'ridan-to'g'ri
   server chiqish kanali (`300 × 1.15 Mbps ≈ 350 Mbit/s` ortiqcha).

Qo'shimcha: SVC (`L3T3_KEY`) da simulcast O'CHADI, ya'ni yuqoridagi uch pog'onali
narvon yo'qoladi — bu esa topilgan asosiy muammoni qaytarib olib keladi.

**H.264 qiziq natija ko'rsatdi** (eng past kechikish, 200 kbps da eng yaxshi matn),
lekin sifat shifti 39 dB da qotib qoladi (VP8/VP9 43–45 dB ga chiqadi) va TTFF
yomonroq (779–1427 ms). U **kelajakdagi nomzod** sifatida qayd etiladi — arzon
Android'da apparat enkoderi borligi uni jozibador qiladi, lekin qaror haqiqiy
qurilmalarda o'lchovni talab qiladi.

---

## 4. Kiritilgan o'zgarishlar

### 4.1 Web — `frontend/src/livekit/mediaTuning.js`

Uch pog'onali narvon, **uchalasi ham manba o'lchamida**, farq faqat fps/bitreytda:

| rid | o'lcham | fps | bitreyt |
|---|---|---|---|
| `f` | manba (≤1080p) | 15 | 1 500 kbps |
| `h` | manba | 8 | 800 kbps |
| `q` | manba | 3 | 300 kbps |

Presetlar `1920×1080` deb yozilgan, lekin bu **cheklov emas, ustki chegara**:
SDK `scaleResolutionDownBy = max(1, min(manba)/min(preset))` deb hisoblaydi, ya'ni
1080p va undan kichik ekranda kichraytirish umuman bo'lmaydi; 1440p/4K da esa
past ikki qatlam 1080p ga tushadi — aynan kerakli xulq.

Qo'shimcha tuzatishlar shu faylda:
- `priority` endi narvonning **birinchi** presetida (Chrome cheklovi — §2.3);
- `degradationPreference` xona darajasidan olib tashlandi, o'rniga
  `SCREEN_PUBLISH` (`maintain-resolution`) va `CAMERA_PUBLISH` (`balanced`)
  — chaqiruv joylari: `Controls.jsx`, `useRoom.js`;
- `SCREEN_CAPTURE.resolution` oshkora 1080p (avval SDK default'iga tayanardi);
- eskirgan izohlar tuzatildi (SDK 2.6.4 → 2.21.0, `AudioPresets.speech` 24 kbps).

### 4.2 Mobil — `mobile/.../data/livekit/MediaTuning.kt`

Xuddi shu siyosat, lekin **SDK cheklovi bilan**: Android
`LocalParticipant.computeVideoEncodings` `scaleDownBy == 1.0` bo'lgan qatlamni
**tashlab yuboradi** ("Discarding duplicate encoding with a scale down == 1.0",
2.27.0 — o'qib tekshirilgan). Shuning uchun mobilda qatlamlar manba o'lchamida
bo'la olmaydi:

| rid | o'lcham (1280×720 manbadan) | fps | bitreyt |
|---|---|---|---|
| `f` | 1280×720 (1.0×) | 15 | 1 500 kbps |
| `h` | 960×540 (1.33×) — **YANGI** | 8 | 800 kbps |
| `q` | 640×360 (2.0×) | 3 | 200 kbps |

Ya'ni mobilda ham **yetishmayotgan o'rta pog'ona yopildi** va kichraytirish
1.33× dan oshmaydi. Bu SDK farqi, mahsulot qarori emas — siyosat bir xil:
avval kadr chastotasi qurbon bo'ladi.

### 4.3 Server — barcha LiveKit konfiguratsiyalari

```yaml
room:
  playout_delay:
    enabled: true
    min: 0
    max: 500
```

Fayllar: `services/livekit/livekit{,.local,.prod,.standalone}.yaml`,
`deploy/server/livekit.yaml.example`.

O'lchangan ta'siri (boshqa hech narsa o'zgarmagan holda):

| qatlam | kechikish p50/p95 oldin | keyin | jitter-bufer |
|---|---|---|---|
| yuqori | 154 / 180 ms | **85 / 113 ms** | 72 → 5 ms |
| o'rta | 68 / 104 ms | 63 / 142 ms | 5 → 7 ms |
| past | 288 / 1118 ms | **171 / 384 ms** | 113 → 117 ms |

Sifat (PSNR, keskinlik) o'zgarmadi — sof yutuq. `max: 500` ataylab 250 emas:
yaxshi tarmoqda natija bir xil (o'lchandi), yomon tarmoqda esa buferga joy qoladi.

---

## 5. Yakuniy oldin/keyin (yetkazilgan sozlama bilan o'lchangan)

Manba 1920×1080 @ 15 fps, VP8, aylanuvchi kod/slayd (eng og'ir holat).

| Kanal | Sozlama | O'lcham | Bitreyt | PSNR | Keskinlik | Kechikish p50/p95 | TTFF |
|---|---|---|---|---|---|---|---|
| yaxshi | **oldin** | 1920×1080 | 1525 kbps | 39.3 dB | 0.97 | 106 / 148 ms | 351 ms |
| yaxshi | **keyin** | 1920×1080 | 1489 kbps | 38.5 dB | 0.96 | **85 / 113 ms** | 403 ms |
| o'rta | **oldin** | *qatlam yo'q — 640×360 ga tushardi* | | | | | |
| o'rta | **keyin** | 1920×1080 @ 8 | 761 kbps | 33.9 dB | 0.94 | 63 / 142 ms | 367 ms |
| zaif | **oldin** | 640×360 @ 3 | 139 kbps | 17.0 dB | **0.34** | 57 / 65 ms | 248 ms |
| zaif | **keyin** | 1920×1080 @ 3 | 338 kbps | **28.8 dB** | **0.83** | 171 / 384 ms | 370 ms |

### Halol narxi

1. **Zaif qatlamda kechikish oshdi** (57 → 171 ms). Sabab: to'liq 1080p kadrni
   past bitreytda kodlash. Bu **ongli savdo**: o'qib bo'lmaydigan 57 ms dan
   o'qiladigan 171 ms afzal (`docs/PRODUCT.md`: matn o'qilishi mahsulot o'zagi).
   Har xil fps sinaldi (3/5/8 fps) — kechikish 175–256 ms oralig'ida qoldi, ya'ni
   u kadrlar orasidagi masofadan emas, enkoderdan keladi.
2. **Zaif qatlam bitreyti 139 → 338 kbps.** Ovoz bilan birga ~390 kbps —
   hali ham kuchsiz 4G chegarasida.
3. **Nashr qiluvchi protsessori ~0.20 → ~0.40 yadro** (uch qatlam, VP8 dasturiy
   enkoder, 1080p). `dynacast: true` bo'lgani uchun obunachisi yo'q qatlam
   to'xtatiladi, ya'ni amaldagi narx odatda bundan past.

---

## 6. Qolgan tavsiyalar

1. **Serverda qayta o'lchash (eng muhim).** Bu raqamlar `host/host` ICE va ~1 ms
   RTT bilan olingan. Real serverda: (a) RTT qo'shiladi; (b) bir qism klient
   **TURN relay** orqali ketadi. Zondni server manzili bilan ishga tushirish
   yetarli:
   `node tools/media-probe/run.mjs --lk wss://livekit.<host> --key darslykey --secret <sir>`
   va `ice` maydonida `relay` uchraydimi — shuni tekshirish.
2. **TURN TLS (443)** hali yoqilmagan (`deploy/server/livekit.yaml.example`).
   Korporativ/mehmonxona tarmoqlarida media umuman ulanmaydi — kechikish emas,
   **ishlamaslik** muammosi.
3. **`rtc.congestion_control.allow_pause`** (default `true`) tekshirilsin: kanal
   eng past qatlamga ham yetmaganda LiveKit videoni butunlay TO'XTATADI. Dars
   uchun degradatsiya to'xtashdan afzal bo'lishi mumkin, lekin buni cheklangan
   kanal emulyatsiyasi bilan o'lchash kerak (bu muhitda emulyatsiya yo'q edi).
4. **H.264 nomzodligi** haqiqiy Android qurilmalarida o'lchansin (apparat
   enkoderi CPU va issiqlikni sezilarli kamaytirishi mumkin).
5. **Mobil top qatlami 720p, web'da 1080p.** Telefon ekrani portret bo'lgani
   uchun bu hozircha to'g'ri, lekin planshetdan ulashilganda 1080p'ga ko'tarish
   qaraladi (avval arzon qurilmada issiqlik o'lchansin).
6. **Statik slayd holati** alohida qaralsin: o'lchovda statik kontentda VP8
   36.7 dB, VP9 esa 30.6 dB berdi — ya'ni statik slaydda ham VP9 yutmaydi.
