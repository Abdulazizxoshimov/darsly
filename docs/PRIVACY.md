# Maxfiylik siyosati — Jonly (Darsly)

> **Kuchga kirish sanasi:** [SANA]
> **Operator:** [TASHKILOT/JISMONIY SHAXS NOMI]
> **Aloqa:** [ALOQA EMAIL]
>
> ⚠️ Bu — kod ma'lumot oqimlariga asoslangan **asos matn**. Play Store'ga qo'yishdan
> oldin operator nomi, sana, aloqa manzili to'ldirilishi va yuridik ko'rikdan
> o'tishi kerak. Play Store uchun bu hujjat **ochiq URL**da joylashtiriladi
> (masalan `https://[domen]/privacy`).

## 1. Biz qanday ma'lumot yig'amiz

**Siz taqdim etadigan:**
- **Hisob ma'lumotlari:** to'liq ism, email manzil, parol (shifrlangan — bcrypt).
- **Profil:** til, vaqt mintaqasi.
- **Dars kontenti:** dars nomlari, jadval, yaratgan darslaringiz.

**Xizmatdan foydalanishda avtomatik:**
- **Texnik ma'lumot:** IP manzil va qurilma/brauzer turi (`User-Agent`) — sessiya
  xavfsizligi va suiiste'molга qarshi himoya uchun.
- **Video/audio oqim:** jonli dars davomida ovoz va video real vaqtda uzatiladi
  (LiveKit SFU orqali). Yozib olish **yoqilgan** darslarda yozuv saqlanadi.
- **Chat:** dars davomidagi xabarlar va ulashilgan fayllar.

## 2. Ma'lumotdan qanday foydalanamiz

- Xizmatni ko'rsatish (jonli dars, kutish xonasi, yozib olish, bildirishnomalar).
- Hisobingizni himoya qilish (autentifikatsiya, brute-force'ga qarshi lockout).
- Texnik nosozliklarni tuzatish (server loglari).

Ma'lumotingizni **sotmaymiz** va reklama uchun ishlatmaymiz.

## 3. Ma'lumot qayerda saqlanadi

- **Ma'lumotlar bazasi (PostgreSQL):** hisob, dars, chat metama'lumotlari.
- **Obyekt saqlash (MinIO/S3):** dars yozuvlari, ulashilgan fayllar.
- **Redis:** vaqtinchalik sessiya va kesh ma'lumotlari.
- Serverlar [SERVER JOYLASHUVI/PROVAYDER] da joylashgan.

## 4. Uchinchi tomonlar

- **LiveKit** — real vaqtli video/audio uzatish (media oqimi).
- [SENTRY — agar yoqilsa: xato/nosozlik telemetriyasi]
- Boshqa uchinchi tomon analitikasi yoki reklama tarmog'i **ishlatilmaydi**.

## 5. Ma'lumotni saqlash muddati

- **Dars yozuvlari:** 30 kun (`expires_at`), so'ng avtomatik o'chiriladi.
- **Sessiyalar:** refresh token TTL tugagach bekor bo'ladi.
- **Hisob ma'lumotlari:** siz o'chirmaguningizcha (quyida 6-band).

## 6. Sizning huquqlaringiz — hisobni o'chirish

Ilovada **Kabinet → Akkaunt → "Akkauntni o'chirish"** orqali hisobingizni istalgan
vaqtda o'chirishingiz mumkin. O'chirilganda hisobingiz nofaol qilinadi, barcha
sessiyalar bekor qilinadi va darslaringiz havolalari ishlamay qoladi.

Ma'lumotni ko'rish yoki tuzatish uchun [ALOQA EMAIL] ga murojaat qiling.

## 7. Bolalar

Xizmat [YOSH CHEGARASI, masalan 13] yoshdan kichik bolalar uchun mo'ljallanmagan.

## 8. O'zgarishlar

Ushbu siyosat yangilanganda sana yangilanadi va muhim o'zgarishlar haqida
xabar beramiz.

## 9. Aloqa

Savollar: [ALOQA EMAIL]
