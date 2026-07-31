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
- Ko'rish: **faqat mentor**; o'quvchilarga tarqatishni mentor o'zi hal qiladi (Telegram)
- **Saqlash: 30 kun**, keyin avto-o'chirish (ogohlantirish bilan) — ⚠️ hali qurilmagan
- Sifat: kichik hajm + yuqori sifat balansi (CRF qayta kodlash — qurilgan ✅)
- «Yozilmoqda» banneri o'quvchiga: KERAK EMAS (REC indikator yetadi)

### Chat va Poll
- Chat tarixi: faqat mentor ko'radi (web modal — qurilgan ✅)
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

(K = kichik, O = o'rta)

## Keyinga surilgan (ataylab)
Monetizatsiya/tariflar/to'lov · o'quv markazi rejimi · takrorlanuvchi darslar · davomat
hisoboti · whiteboard/breakout/test (hammasi kerak, tartibi launchdan keyin) · iOS ·
student ilova · SMS · backup · rus/ingliz tillari
