# Jonly — mahsulot konstitutsiyasi

> Manba: asoschi bilan 43-savollik intervyu (2026-07-30). Bu hujjat barcha texnik va
> dizayn qarorlarning asosi. O'zgartirish faqat asoschi qarori bilan.

## Bir jumlada

**Yakka repetitorlar uchun, past internetda ham sifatli ishlaydigan, o'zbek tilidagi
jonli video-dars platformasi — birinchi relizda «Zoom qila oladigan narsani qilish» darajasida.**

## Asosiy qarorlar

### Auditoriya va pozitsiya
- **Kim uchun:** yakka repetitorlar (o'quv markazi/tashkilot tushunchasi YO'Q)
- **Guruh hajmi:** birinchi relizda 100–300 ishtirokchi; istiqbolda minglab
  (D-intervyuda model Zoom'ga o'zgardi — o'quvchi ham gapira oladi; masshtab
  «kirganda mute» + simulcast bilan boshqariladi)
- **Geografiya:** faqat O'zbekiston (birinchi yil)
- **Til:** faqat o'zbek
- **Raqobat ustunligi (1-reliz):** Zoom pariteti + past internetli hududlarda sifat
  (TURN, past bitreyt profillari, barqaror qayta ulanish)

### Biznes-model
- **Hozircha bepul.** Tariflar/narx/to'lov — birinchi relizdan KEYIN.
  Hozirgi ustuvorlik: asosiy funksiyalar bexato va kuchli ishlashi.

### Rollar
- **O'quvchi:** havola bilan kiradi; **ixtiyoriy akkaunt** (keyingi bosqichda tarix/davomat uchun)
- **Mentor qo'shish:** faqat super-admin (ochiq registratsiya YOPIQ)
- **Sessiya siyosati:** bitta akkaunt = **bitta faol sessiya** (akkaunt ulashishga qarshi) — ⚠️ hali qurilmagan
- **Mentor asosiy qurilmasi: telefon/planshet** — mobil tajriba birinchi darajali!
- iOS: kerak emas (Android yetadi). Student mobil ilova: keyinroq.

### Dars hayoti
- Maksimal davomiylik: **texnik limit ~4 soat** (avto-yakun) — ⚠️ hali qurilmagan
- **Avto-yakun:** xona bo'shagach 15–30 daqiqadan keyin dars o'zi yakunlanadi — ⚠️ hali qurilmagan
- Reopen YO'Q: yakunlangan dars qayta ochilmaydi
- Kechikish: mentor xohlasa qulflaydi (mavjud qulf yetadi)
- Takrorlanuvchi darslar: kerak, lekin 1-relizdan keyin
- **Havola muddati:** dars yakunlangach join ishlamaydi (faqat ma'lumot) — ⚠️ hali qurilmagan

### Xavfsizlik
- Kutish xonasi: default **o'chiq** (hozirgidek)
- **Ban:** chiqarishda mentor tanlaydi — «shu darsdan» yoki «doimiy (mening hamma darslarimdan)» — ⚠️ hozir faqat bir darslik

### Yozuvlar (video)
- Avto-yozuv: dars boshlanishi bilan (qurilgan ✅)
- Ko'rish: ilovada **faqat mentor**. Tarqatish Telegram orqali va **qaror mentorniki**:
  dars tugagach bot guruhlar ro'yxatini beradi, mentor qaysi biriga yuborishni tanlaydi
  yoki umuman yubormaydi. **Arxiv guruhiga esa avtomatik** saqlanadi (2026-08-01).
- **Saqlash: server 30 kun** → keyin serverdan o'chadi, lekin **Telegramda ABADIY**
  qoladi (2026-08-01 qarori — «Dars arxivi va Telegram saqlash» bo'limiga qara).
  Eski qoida «30 kundan keyin butunlay yo'qoladi» BEKOR QILINDI.
- Sifat: kichik hajm + yuqori sifat balansi (CRF qayta kodlash — qurilgan ✅)
- «Yozilmoqda» banneri o'quvchiga: KERAK EMAS (REC indikator yetadi)

### Chat va Poll
- Chat tarixi: ilovada faqat mentor ko'radi (qurilgan ✅). Telegramga yuborish —
  **mentor tanlaydi**: har dars oxirida bot «chat ham ketsinmi?» deb so'raydi (2026-08-01).
- O'quvchilar o'rtasida shaxsiy chat: YO'Q (umumiy + mentorga shaxsiy)
- **Mentor xabar o'chira oladi** (moderatsiya) — ⚠️ hali qurilmagan
- **Poll ikki rejimda yaratiladi:** «faqat mentor uchun» / «hammaga ko'rinadigan»;
  natija o'quvchilarga faqat mentor **«E'lon qilish»** tugmasini bosganda chiqadi — ⚠️ hali qurilmagan

### Bildirishnomalar
- Mentorga: ilova push + **Telegram-bot** — ⚠️ bot hali qurilmagan
- O'quvchiga: hozircha YO'Q (mentor o'zi Telegram guruhida eslatadi);
  ro'yxatdan o'tganlarga keyingi bosqichda

### Operatsion
- **Launch: 2–4 hafta.** Pilot foydalanuvchilar aniq (asoschining tanish mentorlari)
- Support: Telegram
- Server: 4 CPU VPS (bitta mentor + o'quvchilari uchun yetarli deb qabul qilingan)
- Backup: keyinroq (ongli risk)
- Domen: jonly.uz — asoschi shu hafta oladi

### Dars o'tish tajribasi (D-intervyu, 2026-07-31)
- **MODEL O'ZGARDI: webinar emas, Zoom modeli.** O'quvchi mikrofon va kamerani ERKIN yoqa oladi.
  Umumiy qoida: noaniq joyda **Zoom qanday qilgan bo'lsa, shunday qilamiz** (asoschi ko'rsatmasi).
- Ovoz nazorati (Zoom andozasi): kirganda mute · «Hammani o'chirish» · «o'zi ochishiga ruxsat» tumbleri ·
  yakka mute (bor) — ⚠️ qurilishi kerak
- O'quvchilar bir-birini ko'radi (galereya, sahifalab — bir sahifada 9-16) — ⚠️ katta UI ishi
- Sifat degradatsiyasi: **ovoz > ekran > kamera** (Zoom kabi; LiveKit simulcast/dynacast sozlamalari)
- Mentor sahnasi: planshetdan ekran ulashish (notes/slayd) — asosiy stsenariy; PiP kamera bilan
- Qo'l ko'tarish: faqat belgi bo'lib qoladi (mikrofon baribir erkin)
- **Reaksiyalar (emoji) — 1-relizda** ⚠️ · **Fayl ulashish (chatda PDF/rasm) — 1-relizda** ⚠️
- Dars boshlanmagan: o'quvchiga «kutish sahifasi», boshlanganda avto-kirish — ⚠️
- Ko-host: keyinroq. Ishtirokchilar paneli: Zoom standarti (holatlar, qo'l tepada, qidiruv)

### Ekran ulashish overlay'i va SIFAT maqsadi (asoschi, 2026-07-31)

**Strategik maqsad:** Zoom'ni ortda qoldirish — sekin, lekin izchil. Birinchi reliz
foydalanuvchida **chuqur iz** qoldirishi kerak, ya'ni kamida Zoom darajasida
(tezlik va sifat bo'yicha), imkon bo'lsa undan yaxshiroq.

**Ikki o'lchov birinchi darajali:**
1. **Kechikish (latency)** — hozir "juda ko'p" deb baholandi; minimallashtirish shart.
2. **Ekran ulashish sifati** — hozir past; past internetli hududlarda ham yuqori
   sifat kerak (matn/slayd o'qilishi).

**Overlay talabi (screen-share-safe):**
- Ekran ulashilayotganda, foydalanuvchi BOSHQA ilovada bo'lsa ham, ekranda
  kichik suzuvchi panel turadi (Document PiP — alohida OS oynasi).
- Panelda IKKI mustaqil element: (a) **Chat tugmasi** yangi xabar badge'i bilan,
  (b) **Qo'l ko'targanlar indikatori** — chatdan BUTUNLAY ajratilgan, o'z badge'i
  va animatsiyasi bilan.
- Chat tugmasi bosilganda ekranning ~1/4 qismini egallovchi yon panel ochiladi
  (xabarlar tarixi + real-time yangi xabarlar + kim yozgani).
- Qo'l ko'targanlar ro'yxati chat panelining ICHIDA emas — overlay'ning o'zida
  (dropdown/mini-ro'yxat).
- Panel ochilganda badge nolga tushadi (o'qilgan deb belgilanadi).
- **Muhim cheklov:** butun ekran ulashilsa hech qanday web-overlay yashira olmaydi
  (OS darajasidagi cheklov) — shuning uchun ulashish boshlanishida ogohlantirish:
  «Overlay o'quvchilarga ko'rinmasligi uchun butun ekranni emas, bitta oyna/tabni
  ulashing».

### Dars arxivi va Telegram saqlash (asoschi, 2026-08-01)

**Maqsad:** Zoom kabi — dars tugagach nafaqat video, balki **chat ham** saqlanadi;
video esa Telegram guruhiga yuboriladi (o'quvchilarga tarqatish + ikkinchi nusxa).

**Dars sahifasi (ilovada):** o'tgan darsni bosganda — chapda video pleyer, o'ngda
chat tarixi (vaqt belgilari bilan). Chatdagi vaqtni bosganda video o'sha daqiqaga
o'tadi. Pastda «Materiallar» — darsda ulashilgan fayllar (serverda qoladi).

**Video saqlash zanjiri (ishonchlilik birinchi o'rinda):**
1. Yozuv serverga tushadi (hozirgidek, egress → MinIO)
2. Bot Telegramga yuklaydi; muvaffaqiyat **tasdiqlangach** server nusxasi qoladi
3. **Server nusxasi 30 kun** turadi → keyin o'chadi (Telegram = doimiy arxiv)
4. 30 kundan eski darsni ochganda: Telegramdan yuklab olinadi (30-60 s),
   ijro etiladi va **1 kun keshda** turadi, keyin yana o'chadi
5. Yuklash uzilsa: 3 marta ortib boruvchi oraliqda qayta urinish + mentorga
   bildirishnoma; **server nusxasi O'CHIRILMAYDI** (video hech qachon yo'qolmaydi)

**Telegram cheklovi va yechimi:** oddiy bot 50 MB gacha yuboradi, 1 soatlik dars
esa ~75–250 MB. Shuning uchun serverda **Local Bot API Server** (Telegram'ning
rasmiy dasturi, Docker) ishlaydi — chegara 2 GB. Kerak: `api_id`/`api_hash`
(my.telegram.org) + bot tokeni (@BotFather).

**Guruh siyosati (2026-08-01, aniqlashtirilgan):**
- **Arxiv guruhi** — mentorning shaxsiy «Video darslar» guruhi: video u yerga
  HAR DOIM avtomatik saqlanadi (bu ikkinchi nusxa, ishonchlilik uchun).
- **O'quvchilar guruhi** — dars tugagach bot mavjud guruhlar ro'yxatini beradi,
  mentor qaysi biriga yuborishni tanlaydi **yoki umuman yubormaydi**. Qaror mentorniki.
- **Chat fayli** (TXT + HTML) — mentor tanlaydi: bot «chat ham ketsinmi?» deb so'raydi.
- **Darsdagi fayllar** Telegramga yuborilmaydi (guruh shishmasin) — ilovada qoladi.

**Maxfiylik:** guruh a'zolari videoni forward qila oladi — bu qabul qilingan
(mentor o'z guruhini biladi).

### Yozuv sifati: Zoom bilan taqqoslash (2026-08-01, o'lchangan)

Asoschi ikki faylni berdi: Zoom yozuvi va Jonly yozuvi. Ikkalasi ochib, kadrlari
ko'rildi va o'lchandi. **Farq bitreytda emas — KOMPOZITSIYADA.**

| | Zoom | Jonly (hozir) |
|---|---|---|
| Kadr | 1280×800 (**manba nisbati**) | 1280×720 (**qotib qolgan**) |
| Kontent pikseli | 1 024 000 | 230 400 (**22%**) |
| Kadr/sek | 25 | 15 |
| Ovoz | 48 kHz stereo | 44.1 kHz mono |

**Uchta asosiy nuqson:**
1. **78% kadr behuda.** Tik (portrait) telefon ekrani 16:9 kadrga solingan →
   chap va o'ngda qora yo'llar. Zoom esa kadrni MANBA nisbatiga moslaydi
   (1280×800 = 16:10 planshet ekrani) va kontent butun kadrni to'ldiradi.
2. **Yozuvda ilovaning O'ZI ko'rinadi** — «Ekraningiz ulashilmoqda»,
   ishtirokchilar paneli, tugmalar. Ustoz butun telefon ekranini ulashib,
   Jonly ichida qolgan. Zoom bunda ilovani **avtomatik yig'ib** (floating
   tugmaga) foydalanuvchini kontent ilovasiga chiqaradi.
3. **Boshida ~15 s qora ekran** va kamera plitkasi bo'sh avatar bo'lib turadi.

**Qaror (asoschi): Zoom andozasi.** Ya'ni:
- yozuv kadri manba nisbatiga moslashadi (tik manba → tik yozuv), qora yo'l yo'q;
- ekran ulashilganda kontent butun kadrni egallaydi (kamera kichik yoki yo'q);
- 25 fps, 48 kHz stereo ovoz;
- ulashish boshlanishida ilova avtomatik fonga o'tadi (Zoom kabi), efir-ramka
  va suzuvchi panel qoladi.
Kamchiliklar chiqsa — keyingi relizda qayta ko'riladi.

#### Bajarildi (2026-08-03) — o'lchov bilan tasdiqlangan

Uch bosqichda tuzatildi. Har biri alohida qatlamda, chunki bittasi ishlamay
qolsa keyingisi qutqaradi:

| Bosqich | Qayerda | Nima qiladi |
|---|---|---|
| 1. Manba | `ScreenCaptureSize.kt` (mobil) | Ulashish manbasi qurilma **nisbatidan** hisoblanadi. Avval qotib qolgan 1280×720 edi va `MediaProjection` tik telefonni o'sha kadrga qora yo'l bilan solardi — ya'ni qora yo'l MANBADA tug'ilardi. |
| 2. Kompozitsiya | `egress.go` (backend) | Yozuv kadri xonadagi haqiqiy trekdan (`roomVideoSize` → `pickRecordingSize`) olinadi; layout `speaker` → **`single-speaker`** (yon karusel ustuni kadrning ~1/4 ini yeyardi); 25 fps; 48 kHz. |
| 3. Kafolat | `videofilter.go` + `transcode_analyze.go` | Qayta kodlashda `cropdetect` bilan qolgan qora yo'l kesiladi va boshidagi **qora VA jim** qism tashlanadi. Bu — oxirgi to'siq: yuqoridagi ikkalasi adashsa ham yakuniy faylda qora yo'l qolmaydi. |

**Asoschining haqiqiy yozuvi ustida o'lchandi** (`video_2026-08-01_17-24-41.mp4`,
quvurni to'liq yurgizib):

| | Oldin | Keyin |
|---|---|---|
| Kadr | 1280×720 | 320×718 (**tik — manba nisbati**) |
| Kadr/sek | 15 | 25 |
| Ovoz | 44.1 kHz mono | 48 kHz stereo |
| Kontent kadrning necha % i | 25% | **100%** |

Kesilgan kadr ko'z bilan ham tekshirildi: telefon ekrani to'liq, hech qayeri
qirqilmagan. Qaror mantiqi 20+ chekka holat testi ostida
(`videofilter_test.go`), quvurning o'zi esa nuqsonni sun'iy takrorlovchi
haqiqiy-ffmpeg testi ostida (`TestTranscodePipeline_RemovesBarsAndDeadStart`).

#### ⚠️ Qolgan bo'shliq — asoschi qarori kerak

**Yozuv birinchi `track_published` da boshlanadi va u odatda MIKROFON.** Ya'ni
kadr o'lchami tanlanayotgan paytda xonada hali video trek yo'q → bazaviy
1280×720 olinadi. Ekran ulashish keyinroq kelib, o'sha kadrga qora yo'l bilan
tushadi. 3-bosqich (kesish) kompozitsiyani baribir tuzatadi, lekin
**rezolyutsiyani qaytara olmaydi** — piksellar egress bosqichida yo'qolgan.

O'lchovda ko'ringan farq (kamera O'CHIQ, telefon ekrani ulashilgan):
- kadr to'g'ri tanlansa: **576×1280**
- hozirgi holda (kesish bilan): **324×720** — ya'ni chiziqli ~44% past

Kamera YONIQ bo'lsa muammo yo'q: telefon kamerasi tik trek beradi va kadr
o'shanga to'g'ri moslanadi.

**Yechim va uning narxi.** Yozuvni birinchi *video* trekda boshlash — u holda
kadr har doim to'g'ri. Lekin ustoz kamerasiz kirib, ulashishdan oldin gapirsa
(«salom, hozir ekranni ulashaman»), **o'sha gap yozuvga tushmaydi**. Bu aynan
`leadingDeadSeconds` ataylab saqlab qolayotgan narsa — ya'ni bu ovoz yo'qotish
bilan rezolyutsiya o'rtasidagi savdo, texnik emas, MAHSULOT qarori.

Uchinchi yo'l yo'q: egress boshlangach kadr o'lchamini o'zgartirib bo'lmaydi,
qayta boshlash esa bitta darsdan ikkita fayl qoldiradi.

## Qurilishi kerak (javoblardan kelib chiqqan yangi ishlar)

| # | Ish | Manba savol | Hajm |
|---|-----|------------|------|
| 1 | Bitta faol sessiya (yangi login eskisini o'chiradi) | 14 | backend O |
| 2 | 4 soatlik texnik limit + xona bo'shagach 15-30 daq avto-yakun | 15, 19 | backend O |
| 3 | Yakunlangan dars havolasi join qilmasin | 21 | backend K |
| 4 | Ban tanlovi: bir darslik / mentor bo'yicha doimiy | 22 | backend+UI O |
| 5 | Yozuvlar 30 kunlik retention (ogohlantirish + avto-o'chirish) | 24 | backend O |
| 6 | Chat xabarini o'chirish (mentor) | 29 | backend+UI K |
| 7 | Poll ikki rejim + «E'lon qilish» | 30 | backend+UI O |
| 8 | Telegram-bot (mentor eslatmalari) | 31 | yangi servis O |
| 9 | O'quvchi ixtiyoriy akkaunti | 11 | katta — keyingi bosqich |
| 10 | 100-300 tomoshabinga yuklama testi (mavjud tests/load kengaytiriladi) | 2 | QA O |
| 11 | **Zoom modeli: o'quvchi publish (mic+kamera), kirganda mute, Mute All, ruxsat tumbleri** | D1-D5 | backend+web KATTA |
| 12 | Galereya ko'rinishi sahifalash bilan (o'quvchilar bir-birini ko'radi) | D11 | web O |
| 13 | Sifat profillari: ovoz>ekran>kamera degradatsiyasi (simulcast/dynacast) | D6 | backend+klient O |
| 14 | Emoji-reaksiyalar | D7 | backend K + UI K |
| 15 | Chatda fayl ulashish (MinIO orqali) | D8 | backend O + UI K |
| 16 | Dars oldi kutish sahifasi (avto-kirish bilan) | D10 | web K |
| 17 | **Kechikishni minimallashtirish** (Zoom darajasi yoki undan yaxshi) | 2026-07-31 | KATTA |
| 18 | **Ekran ulashish sifati past internetda** (matn o'qiladigan) | 2026-07-31 | KATTA |
| 19 | Overlay: kompakt PiP + chat badge + alohida qo'l indikatori + 1/4 panel | 2026-07-31 | web O |
| 20 | **Dars arxiv sahifasi** — video + chat yonma-yon, vaqt bo'yicha sakrash, materiallar | 2026-08-01 | backend K + web O + mobil O |
| 21 | **Chat transkripti** (TXT + HTML eksport) | 2026-08-01 | backend K |
| 22 | **Telegram saqlash**: Local Bot API Server, guruh tanlash, qayta urinish, 30 kun + 1 kunlik kesh | 2026-08-01 | yangi servis KATTA |
| 23 | ~~Yozuv kadri manba nisbatiga moslashsin (qora yo'llar yo'qolsin)~~ ✅ 2026-08-03 | 2026-08-01 | backend O |
| 24 | ~~Yozuv 25 fps + 48 kHz stereo (Zoom pariteti)~~ ✅ 2026-08-03 | 2026-08-01 | backend K |
| 25 | ~~Ulashishda ilova avtomatik fonga o'tsin (Zoom kabi)~~ ✅ 2026-08-03 | 2026-08-01 | mobil K |
| 26 | **Yozuv boshi kesilganda kadr o'lchami** — ustoz kamerasiz kirsa yozuv 16:9 boshlanadi (yuqoridagi «Qolgan bo'shliq») | 2026-08-03 | qaror kerak |

(K = kichik, O = o'rta)

## Keyinga surilgan (ataylab)
Monetizatsiya/tariflar/to'lov · o'quv markazi rejimi · takrorlanuvchi darslar · davomat
hisoboti · whiteboard/breakout/test (hammasi kerak, tartibi launchdan keyin) · iOS ·
student ilova · SMS · backup · rus/ingliz tillari
