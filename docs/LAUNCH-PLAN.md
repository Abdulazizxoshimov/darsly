# Jonly — launch rejasi

> **RELIZ SANASI: 2026-yil 15-avgust** (asoschi belgilagan).
> Telegram-bot integratsiyasi alohida ko'rib chiqiladi — u bilan birga yana bitta
> funksiya qo'shiladi (asoschi qarori kutilmoqda).

> Manba: docs/PRODUCT.md (asoschi qarorlari). Maqsad: 2–4 hafta ichida pilot
> mentorlar bilan real darslar. Ish raqamlari PRODUCT.md'dagi «Qurilishi kerak» jadvalidan.

## 1-hafta — «Zoom modeli» yadrosi (eng katta xavf oldinga)

Mavzu: o'quvchi gapira oladigan/kamera yoqa oladigan darsga o'tish.

- **Backend (№11):** guest token `canPublish=true`; dars sozlamalari: kirganda mute,
  «o'zi ochishiga ruxsat» bayrog'i; `Mute All` API; yakka mute (bor) kengaytirish;
  ruxsat o'zgarishlarini LiveKit orqali jonli qo'llash
- **Backend (№3):** yakunlangan dars havolasi join qilmasin (preview'da «dars tugagan»)
- **Backend (№4):** ban tanlovi — bir darslik / mentor bo'yicha doimiy (qora ro'yxat jadvali)
- **Web (№11,12,16):** o'quvchida mikrofon/kamera tugmalari; galereya sahifalash (9-16/sahifa);
  dars-oldi kutish sahifasi (avto-kirish)
- **Mobil (№11):** mentorda «Hammani o'chirish» + ruxsat tumbleri; ishtirokchi paneli yangilanishi
- Chiqish mezoni: planshet(mentor) + 2 brauzer(o'quvchi) — o'quvchi gapirdi, mentor o'chirdi,
  ruxsat berdi, yana gapirdi; galereya ishladi

## 2-hafta — dars hayoti + kontent vositalari

- **Backend (№1):** bitta faol sessiya (yangi login eskisini tugatadi)
- **Backend (№2):** 4 soatlik limit + xona bo'shagach 15–30 daq avto-yakun (webhook room_finished)
- **Backend (№5):** yozuvlar 30 kunlik retention worker (ogohlantirish → o'chirish)
- **№6:** chat xabarini o'chirish (mentor) — backend+web+mobil
- **№7:** poll ikki rejim + «E'lon qilish»
- **№14:** emoji-reaksiyalar (data-channel + UI)
- **№15:** chatda fayl ulashish (MinIO, hajm limiti, presigned)
- **№13:** sifat profillari — ovoz>ekran>kamera (simulcast/dynacast, past internet rejimi)
- Chiqish mezoni: to'liq dars stsenariysi boshdan-oxir chekka holatlar bilan

## 3-hafta — masshtab, sifat, chiqish

- **№10:** yuklama testi 100–300 ishtirokchi (tests/load kengaytirish; 4 CPU serverda o'lchash,
  natijaga qarab server tavsiyasi)
- **№8:** Telegram-bot — mentor eslatmalari (dars 15 daq qolganda)
- To'liq regressiya: planshet + web + API sweep (mavjud avtomatik to'plamlar + qo'lda)
- Staging serverni yangi kod bilan yangilash; jonly.uz domen + HTTPS + TURN tekshiruvi
- Pilot onboarding: 2-3 mentor bilan haqiqiy dars, kuzatuv, tuzatishlar
- Chiqish mezoni: pilot mentor mustaqil ravishda to'liq dars o'tdi

## Relizgacha qolgan ishlar (2026-07-31 auditi)

Audit natijasi: yadro tayyor, lekin quyidagilar ochiq — `docs/BACKLOG.md` da ham.

**Tuzatilmoqda (asoschi «hammasini tuzat» dedi):**
1. Kutish xonasi defaulti uch joyda ZID (web ON, mobil OFF, DB DEFAULT TRUE) —
   PRODUCT.md «o'chiq» deydi; + web'da «hammasini kiritish» tugmasi yo'q
2. Chatdagi rasm production'da CSP tufayli ko'rinmaydi (Caddyfile `img-src`)
3. Mobil: dars formasida ovoz sozlamalari yo'q, qora ro'yxat ekrani yo'q
4. Ishtirokchilar panelida qidiruv yo'q (300 kishilik darsda zarur)
5. Ovoz siyosati klientlararo tarqaladi — server javobiga ko'chirilsin

**Qaror kutmoqda:** Telegram-bot (№8) — reliz doirasi.

**Deploy oldidan:** server tiklash · jonly.uz + Caddyfile domenlari · DB backup
(tiklash sinovi bilan) · Sentry DSN · serverda yuklama va TURN tekshiruvi ·
2 brauzer + real kamera bilan qo'lda uchdan-uchga sinov (A0.1).

## Doimiy qoidalar
- Har o'zgarish testlar bilan; har hafta oxirida planshetda jonli regressiya
- Noaniq UX savolida — Zoom andozasi (PRODUCT.md qoidasi)
- docs/PRODUCT.md ga zid ish qilinmaydi; ziddiyat chiqsa asoschidan so'raladi
