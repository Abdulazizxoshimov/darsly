# Darsly Mobile — Qabul Mezonlari (R1)

> Har blok uchun **tekshiriladigan** mezonlar. QA shu ro'yxat bo'yicha ishlaydi.
> Qoida: mezon **kuzatiladigan xatti-harakat** bo'lishi shart — "kod yozildi" mezon emas.
> Har band uchun: ✅ tasdiqlandi (dalil bilan) · ❌ ishlamadi · ⬜ sinalmadi (sabab bilan)
>
> Sana: 2026-07-26. (Manba reja hujjati arxivlangan — git tarixida.)

---

## Umumiy qoidalar (har blokka tegishli)

| # | Mezon |
|---|-------|
| G-1 | `./gradlew assembleDebug` muvaffaqiyatli, `lint` kritik xatosiz |
| G-2 | `./gradlew testDebugUnitTest` — barcha testlar o'tadi |
| G-3 | **Testlar vakuum emas** — asosiy testni buzib (tuzatishni vaqtincha qaytarib) FAIL bo'lishi ko'rsatilgan |
| G-4 | Mavjud ishlaydigan funksiya buzilmagan — ayniqsa **ekran ulashish** |
| G-5 | `backend/`, `frontend/`, `deploy/`, `services/` fayllariga tegilmagan |
| G-6 | Sirlar kodda yo'q; `.gitignore` to'g'ri |
| G-7 | Kod izohlari o'zbekcha, mavjud uslubga mos |

---

## 1-blok · Asos va sessiya

| # | Mezon | Qanday tekshiriladi |
|---|-------|---------------------|
| A-1 | Login → ilova **yopilib qayta ochilganda** kirgan holatda | Qurilmada: login → `force-stop` → qayta ochish → darslar ro'yxati chiqadi |
| A-2 | Token `EncryptedSharedPreferences` da, ochiq matnda emas | `adb shell run-as` bilan prefs faylini o'qib, token matni **ko'rinmasligini** tasdiqlash |
| A-3 | **Single-flight:** N ta parallel 401 → serverga **atigi 1 ta** refresh | MockWebServer'da so'rovlarni sanash; `Mutex` olib tashlanganda test FAIL berishi |
| A-4 | Refresh muvaffaqiyatsiz → **toza logout**, cheksiz sikl yo'q | Refresh'ni majburan buzish → login ekraniga qaytadi, takroriy so'rov bo'roni yo'q |
| A-5 | Access muddati tugagach foydalanuvchi **hech narsa sezmaydi** | 15 daqiqa kutib (yoki TTL'ni qisqartirib) amal bajarish → uzilishsiz ishlaydi |
| A-6 | Logout → server sessiyasi ham bekor | `POST /auth/logout` chaqirilgani; eski token bilan so'rov → 401 |
| A-7 | Xato kodlari o'zbekcha matn beradi, `undefined`/inglizcha chiqmaydi | Har kod uchun test: `RATE_LIMITED`, `FORBIDDEN`, `TOKEN_EXPIRED`, `VALIDATION_ERROR`… |
| A-8 | Xato kodlari ro'yxati **backend manbasi** bilan mos | `grep` bilan solishtirish; web `ERROR_UZ` bilan izchillik |
| A-9 | Tarmoq yo'q holati alohida, tushunarli xabar | Aviarejimda amal bajarish |
| A-10 | `min_version` dan past versiya → **bloklovchi** dialog | `/app-config` da `min_version` ni ko'tarib sinash |
| A-11 | Semver to'g'ri solishtiriladi (`1.10.0 > 1.9.0`) | Unit test: string solishtirish emas |
| A-12 | `/app-config` javob bermasa ilova **ishlashda davom etadi** | Endpoint'ni bloklab sinash — fail-open |
| A-13 | **B-2:** ruxsat natijasi kutiladi, `SecurityException` yo'q | `adb logcat` da xona ochilganda `SecurityException` **bo'lmasligi** |

---

## 2-blok · Darslar

| # | Mezon |
|---|-------|
| B-1 | Darslar ro'yxati yuklanadi; jonli/rejalashtirilgan/tugagan farqlanadi |
| B-2 | Pull-to-refresh ishlaydi |
| B-3 | Bo'sh holat, xato holati, yuklanish holati — uchalasi ham ko'rsatiladi |
| B-4 | Offline: keshlangan ro'yxat ko'rinadi + "internet yo'q" belgisi |
| B-5 | Dars yaratish → **web'da ham** ko'rinadi (uchdan-uchga) |
| B-6 | Join havolasi native "Ulashish" bilan Telegramga yuboriladi |
| B-7 | Yaratishda validatsiya xatolari tushunarli ko'rsatiladi |

---

## 3-blok · Xona: media (⭐ eng muhim)

| # | Mezon | Izoh |
|---|-------|------|
| C-1 | Ekran ulashish → web'dagi o'quvchi **to'liq va o'qiladigan** ko'radi | `object-fit: contain` deploy qilindi — endi kesilmaydi |
| C-2 | **B-1 🔴:** "Entire screen" tanlash bo'yicha ogohlantirish bor | Dialog sukut bo'yicha "A single app" tanlaydi — ustoz PDF'ga o'tsa o'quvchi hech narsa ko'rmaydi |
| C-3 | Ustoz boshqa ilovaga o'tdi → ulashish **davom etadi** | HONOR'da 28 daqiqa tasdiqlangan (R0) |
| C-4 | Ekran qulflandi → dars uzilmaydi; o'quvchiga "vaqtincha to'xtatildi" ko'rsatiladi | R0 da qulflanganda muzlagan kadr ko'rinardi |
| C-5 | **B-4:** mikrofon o'chirilganda UI **yolg'on gapirmaydi** | Hozir "Ekran audiosi: yoniq" deb ko'rsatadi, aslida o'chgan |
| C-6 | **Mahsulot qarori:** ekran audiosi mikrofondan mustaqilmi? | Alohida trek qilinsa — ustoz mikrofonni o'chirib video ko'rsata oladi |
| C-7 | Ekran audiosi Android 10+ da ishlaydi; 8–9 da **crash yo'q** + ogohlantirish | SDK `@RequiresApi(Q)` |
| C-8 | **B-6:** dars yakunlanganda MediaProjection **bo'shatiladi** | Maxfiylik: hozir server yakunlasa ham ilova ushlab turadi |
| C-9 | **B-5:** mikrofon o'chirilganda `AudioRecord` bo'shatiladi | Resurs/batareya oqishi |
| C-10 | Old/orqa kamera almashadi |  |
| C-11 | Wi-Fi ↔ 4G almashganda dars **tugamaydi**, qayta ulanadi |  |
| C-12 | Sifat rejimi: matn uchun 720p/15fps, video uchun 24fps |  |
| C-13 | MediaProjection dialogini **bekor qilish** → crash yo'q, tugma holatiga qaytadi | R0 da sinalmagan |
| C-14 | Har bir ruxsatni **rad etish** → crash yo'q, tushunarli xabar | R0 da sinalmagan |

---

## 4-blok · Xona boshqaruvi

| # | Mezon |
|---|-------|
| D-1 | Kutish xonasi so'rovi **3 soniyada** ilovada ko'rinadi (WS) |
| D-2 | Qabul → o'quvchi darsga kiradi; Rad → rad etilgan ekranini ko'radi |
| D-3 | WS uzilib qayta ulanadi (backoff bilan), kutayotganlar ro'yxati tiklanadi |
| D-4 | Ishtirokchilar ro'yxati real vaqtda yangilanadi |
| D-5 | So'zlashga ruxsat → o'quvchi gapira oladi; bekor qilish ishlaydi |
| D-6 | Ustoz **o'zini** nishonga ololmaydi (backend 400 qaytaradi) — UI buni ko'rsatmaydi |
| D-7 | Darsni yakunlash → hamma chiqadi, dars `ended` bo'ladi |
| D-8 | **Protokol mosligi:** ilovadan yozilgan chat web'da ko'rinadi; web'dagi ✋/👍 ilovada ko'rinadi |
| D-9 | `wb` (doska) xabarlari ilovani **yiqitmaydi** (jim tashlanadi) |

---

## 5-blok · Sayqal va reliz

| # | Mezon |
|---|-------|
| E-1 | Onboarding: kamera/mikrofon/bildirishnoma/overlay ruxsatlari tushuntiriladi |
| E-2 | Batareya optimizatsiyasini o'chirish so'raladi (busiz fon rejimi jim o'ladi) |
| E-3 | Planshet landscape + telefon portret layoutlari ishlaydi |
| E-4 | Sentry crash hisoboti keladi |
| E-5 | Ekran ulashish telemetriyasi: boshlandi/xato/tugadi |
| E-6 | Imzolangan APK quriladi; QR bilan o'rnatiladi |
| E-7 | `versionCode` oshgan, `/app-config` bilan mos |

---

## R1 RELIZ DARVOZASI

Quyidagilarsiz R1 **chiqarilmaydi**:

1. **3 ta real ustoz 90 daqiqalik haqiqiy darsni faqat mobil qurilmadan** uzilishsiz o'tkazdi
2. Ustoz dars o'rtasida PDF/GeoGebra ochdi — dars va ekran ulashish davom etdi
3. Web'dagi o'quvchi ekranni **matn o'qiladigan sifatda va to'liq** ko'rdi
4. Video ko'rsatilganda o'quvchi **ovozni ham** eshitdi
5. Kutish xonasidagi o'quvchi 3 soniyada qabul qilindi
6. Wi-Fi ↔ 4G almashtirilganda dars tugamadi
7. **0 ta kritik crash**; ekran ulashish muvaffaqiyati **≥ 95%** (telemetriya bilan o'lchangan)

### Qurilma matritsasi (minimal)

| Qurilma | Android | Nega |
|---|---|---|
| HONOR ABR-LX1 | 15 | ✅ R0 da sinaldi — eng qattiq FGS qoidalari + agressiv ROM |
| Samsung planshet | 13–14 | Asosiy maqsad qurilma, S Pen |
| Xiaomi/Redmi | 13–14 | MIUI fon o'ldirishi |
| Arzon/eski qurilma | 8–10 | `minSdk 26` chegarasi, ekran audiosi yo'q |

---

## R0 da yopilgan (takrorlanmaydi)

✅ Ekran ulashish Android 15 + HONOR'da ishlaydi (14.27 fps, matn o'qiladi)
✅ Fon rejimi MagicOS'da 28+ daqiqa saqlanadi
✅ **4G/mobil internet ishlaydi** (TURN'siz ham — LiveKit ochiq IP'li SFU)
✅ TURN relay uchdan-uchga tasdiqlangan (relay-only rejimda 1.5 MB media)
✅ Backend: WS Origin, refresh grace, sessiya forking, `sid` DoS, CGNAT rate-limit, UUID 500'lar
