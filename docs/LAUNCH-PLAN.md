# Jonly (Darsly) — Ishga tushirish rejasi

> Manba: 2026-08-15 AI audit + kodni to'g'ridan-to'g'ri tekshirish (har band kod bilan
> solishtirildi). Bu hujjat `BACKLOG.md` havola qiladigan, lekin yo'q bo'lgan
> `LAUNCH-PLAN.md` o'rnini to'ldiradi.
>
> **Asosiy tamoyil:** audit uch xil narsani bitta ro'yxatga qo'shib yuborgan —
> (1) *ishga tushish bloklovchilari* (deploy, backup, crash, real sinov),
> (2) *xavfsizlik qatlamlari* (cookie, lockout), (3) *sayqal/zamonaviylashtirish*
> (haptika, skeleton, illyustratsiya). Bularni **bir xil ko'rish — xato**.
> Reja shu uchta sinfni alohida fazalarga ajratadi.

---

## 0. Auditning haqqoniyligi (tekshirilgan)

Auditdagi aniq da'volar kod bilan solishtirildi. Natija: **~95% aniq.**

| Da'vo | Holat | Isbot |
|---|---|---|
| Landing'da soxta statistika + soxta ustoz | ✅ To'g'ri | `Landing.jsx:13-17` (540+/4.9/1200+), `:98` "Aziz Karimov" |
| Inline `style={{}}` ~266 marta | ✅ To'g'ri (aslida **287**) | `grep style={{ → 287` |
| `LiveRoom.jsx` 1297 qator | ✅ Aniq | `wc -l → 1297` |
| `RoomViewModel.kt` 1333 qator | ✅ Aniq | `wc -l → 1333` |
| Haptika 0 ta | ✅ To'g'ri | `HapticFeedback → 0` |
| `res/drawable` 2 ta fayl (faqat launcher) | ✅ To'g'ri | launcher variantlari |
| `enableEdgeToEdge()` yo'q | ✅ To'g'ri | `MainActivity.kt` da chaqiruv yo'q |
| Animatsiya ~3 faylda | ✅ To'g'ri | 3 fayl |
| `prefers-reduced-motion` / `prefers-color-scheme` yo'q | ✅ To'g'ri | 0 ta |
| `aria-*` ~46 ta | ✅ To'g'ri (aslida **56**) | 56 ta |
| Login account-lockout yo'q | ✅ To'g'ri | lockout faqat `joinlink.go` da |
| Refresh token `localStorage` da | ✅ To'g'ri | `api/api.jsx:19-27` |
| DB backup skripti yo'q | ✅ To'g'ri | `deploy/` da pg_dump yo'q |
| Skeleton-loading yo'q | ✅ To'g'ri | web + mobil 0 ta |
| Sentry kodi backend'da bor, DSN bo'sh | ✅ To'g'ri | `app.go:71 sentry.Init`, `.env` bo'sh |

**Auditda eskirgan/noto'g'ri bandlar (tuzatildi):**

- **Caddyfile eski IP (`194.163.139.242`)** — ❌ eskirgan. Fayl allaqachon
  `169.58.104.245` ga yangilangan (`deploy/server/Caddyfile:5,8,61,77,92`). Bu ish **bajarilgan**.
- **Yuklama testi qilinmagan** deyilmaydi, lekin audit F5 `load-test-report.md`ni
  "yo'q" deydi — fayl haqiqatan yo'q, ammo `BACKLOG.md` "300/300 ishtirokchi" o'tganini
  yozadi. Ya'ni **test bajarilgan, hisobot fayli o'chirilgan**. Bu hujjat-haqiqat
  tafovuti (F5), test yetishmasligi emas.

**Xulosa:** auditga ishonish mumkin. Fundament (tokenlar, xavfsizlik mulohazasi,
test qamrovi) mustahkam; yetishmayotgani — *operatsion oxirgi 10%* va *sayqal*.

---

## 1. "Qancha yaxshilaydi?" — halol baho

Savolga sinf bo'yicha javob:

**A. P0 bloklovchilar (deploy, backup, Sentry DSN, crash, real video sinov, soxta statistika).**
Bu "yaxshilash" emas — bu **"bor" bilan "yo'q" orasidagi farq**. Hozir mahsulot
production'da hech qayerda ishlamayapti va halokat bo'lsa ma'lumot butunlay yo'qoladi.
Bularsiz pilot ham boshlab bo'lmaydi. **Ta'sir: hal qiluvchi. Shart.**

**B. Xavfsizlik qatlamlari (HttpOnly cookie, account-lockout).**
Real, lekin *cheklangan pilot* (3-5 ishonchli ustoz) uchun halokatli emas.
Ochiq bozorga chiqishdan oldin shart. **Ta'sir: o'rta. Pilotdan keyin, ommaviy launchdan oldin.**

**C. Sayqal / zamonaviylashtirish (skeleton, haptika, illyustratsiya, animatsiya,
inline-style refaktor, LiveRoom/RoomViewModel bo'lish, a11y).**
"Ishlaydi" bilan "premium his qiladi" farqi. **Lekin 3-5 ustozli pilotda ustozning
asl savoli bu emas** — "haqiqiy sinfda, real internetda video uzilmasdan ishlaydimi?"
Sayqal chiroyli, ammo qo'ng'iroq uzilib qolsa hech narsani qutqarmaydi.
**Ta'sir: real, lekin marginal — pilotdan KEYIN, 30 kun ichida. ROI keyinroq yuqori.**

**Bitta istisno — F1 (soxta statistika):** C sinfida emas. Bu ~30 daqiqalik ish,
lekin *ishonch* og'irligi eng yuqori (yolg'on statistika aniqlansa — ishonch qaytmaydi,
ba'zi yurisdiksiyalarda chalg'ituvchi reklama). **P0 ga ko'chiriladi.**

> **Bir jumlada:** P0 sinfini to'liq bajaring — u shart. B sinfini pilot davomida.
> C sinfini (sayqal/zamonaviylashtirish) pilotdan keyin — chunki u mahsulotning asl
> riskini (real darsda video ishlashi) kamaytirmaydi, faqat taassurotni yaxshilaydi.

---

## 2. FAZA 0 — Pilotgacha majburiy (🔴 P0)

**Maqsad:** 3-5 ta ishonchli ustoz bilan cheklangan pilot boshlash imkoniyati.
**Taxminiy muddat: 1-2 hafta.**

| # | Ish | Qabul mezoni | Taxmin |
|---|---|---|---|
| **P0-1** | Yangi serverga Docker + compose o'rnatish, `DEPLOY-CHECKLIST.md` bo'yicha to'liq deploy | `https://app.169.58.104.245.sslip.io` (yoki domen) ochilib, login ishlaydi | 1 kun |
| **P0-2** | 2 brauzer + 2 real qurilma/kamera bilan uchdan-uchga video sinov (odam bilan) | Ovoz+video ikki tomonlama ishladi, TURN relay orqali ham | 0.5 kun |
| **P0-3** | DB backup: `pg_dump` cron → MinIO + **tiklashni real sinash** | Backup fayldan bo'sh bazaga to'liq tiklandi, vaqt o'lchandi | 0.5 kun |
| **P0-4** | Sentry DSN'ni backend `.env` + mobil `-PdarslySentryDsn` ga ulash (kod tayyor) | Qasddan xato → Sentry'da ko'rindi (backend va mobil) | 2 soat |
| **P0-5** | Mobil nativ WebRTC SIGABRT crash — chastotani o'lchash (Sentry NDK), so'ng `livekit-android` yangilash yoki `disconnect()` xulqini tuzatish | Wi-Fi↔4G 20 marta almashtirib qulashsiz | 1-3 kun (noaniq) |
| **P0-6** | **F1 — Landing soxta statistika + soxta testimonial**ni olib tashlash/almashtirish | `Landing.jsx` da o'ylab topilgan raqam/ism yo'q; "Yangi platforma — birinchi ustozlarga maxsus shartlar" kabi halol framing | 0.5 kun |

**P0-6 aniq yechim** (`Landing.jsx:13-17`, `94-101`): STATS massivini olib tashlash
yoki `count > 20` sharti bilan yashirish; soxta "Aziz Karimov" kartochkasini xususiyat
ro'yxati / mahsulot skrinshoti / halol "erta foydalanuvchi" framing bilan almashtirish.

**Faza 0 tugagach — "cheklangan pilotga tayyor" (3-5 ustoz), ochiq bozorga emas.**

---

## 3. FAZA 1 — Pilot davomida qattiqlashtirish (🟠 P1)

**Maqsad:** ommaviy launchga ishonchli asos. Pilot ketayotganda parallel.
**Taxminiy muddat: 1-2 hafta (pilot bilan ustma-ust).**

| # | Ish | Qabul mezoni |
|---|---|---|
| P1-1 | 3 pilot ustoz bilan 90 daqiqalik **real** dars (R1 darvozasi) | 3/3 uzilishsiz, ustoz feedback yig'ildi |
| P1-2 | Refresh token'ni `localStorage` → **HttpOnly cookie** | `document.cookie` da token yo'q; mobil API varianti Keystore bilan ko'rildi |
| P1-3 | Login account-lockout (5 xato → 15 daq blok, IP'dan mustaqil; `joinlink.go` M7 naqshi bor) | Turli IP'dan bitta akkauntga ketma-ket → bloklandi |
| P1-4 | TURN TLS/443 yoqish (`livekit.yaml.example` tartibi) | Faqat 443 ochiq tarmoqdan ulanib video ishladi |
| P1-5 | Kuzatuv (Grafana/observability) deploy (`deploy/server/observability/` tayyor) | Dashboard'da real metrikalar |
| P1-6 | 2-brauzerli E2E, real LiveKit media (ovoz/video legi) CI'da | CI'da real SFU testi yashil |
| P1-7 | Play Store majburiylari: maxfiylik siyosati, Data Safety, **akkaunt o'chirish** | Play Console tekshiruvidan o'tdi |
| P1-8 | Mobil real domen (`darsly.uz`/`jonly.uz`), prod build shu domenga | Prod build haqiqiy domenga ulanadi |

---

## 4. FAZA 2 — Sayqal + zamonaviylashtirish (🟡 P2, pilotdan keyin 30 kun)

**Maqsad:** "ishlaydi" → "premium his qiladi". Arxitektura o'zgarmaydi — mavjud kod ustiga.
**Eng yuqori ROI uchtaligi (audit bilan mos): haptika+mikroanimatsiya → skeleton → illyustratsiya.**

### 4A. Eng yuqori ta'sirli (arzon, sezilarli)
- **Haptika (mobil):** asosiy harakatlarga `LocalHapticFeedback` (tugma, dars boshlash, xato).
- **Skeleton-ekranlar (web + mobil):** spinner o'rniga kontent-shaklli skelet (darslar, arxiv,
  bildirishnoma). *Sekin internet auditoriyasi uchun funksional* — sub'ektiv tezroq.
- **Bo'sh/xato holat illyustratsiyalari:** 6-8 ta SVG (bo'sh darslar, bo'sh bildirishnoma,
  tarmoq xatosi, umumiy xato). Bir manba — web SVG, mobil vector drawable.
- **Mikro-animatsiya:** web `transition` + mobil `AnimatedVisibility`/`animateContentSize`;
  bosilish scale 0.96.

### 4B. Kod-sifat / barqarorlik
- **`LiveRoom.jsx` (1297) bo'lish** — kutish/jonli/yozib olinmoqda/xato holatlarini alohida
  komponentlarga; vizual QA yengillashadi.
- **`RoomViewModel.kt` (1333) refaktori** — "god ViewModel"ni mas'uliyati aniq bo'laklarga;
  har regressiya xavfini kamaytiradi (mobil-arch memo naqshi bor).
- **287 inline `style={{}}` → dizayn tokeni klasslari** — bosqichma-bosqich, `LiveRoom.jsx` dan.
- **`enableEdgeToEdge()` (mobil)** — Android 15+ da majburiy bo'lib bormoqda.
- **`prefers-reduced-motion` + (ixtiyoriy) `prefers-color-scheme` light** administrativ sahifalarga.

### 4C. Accessibility + halollik-framing
- Asosiy interaktiv elementlarga `aria-label`, audio/video holat ikonkalariga matn alternativi.
- Marketing tili ("Zoom'ga to'liq muqobil") mahsulot real holatidan oldinga chiqmasin.
- "Iliqlik" copywriting: dars tugagach qisqa ijobiy xabar, bo'sh jadvalda quruq "Dars yo'q" o'rniga iliq ohang.

### 4D. Infra/hujjat (P2)
- CD (git push → staging avtomatik deploy).
- Runbook: tiklash/deploy/rollback/incident.
- F5: `docs/` dagi yo'q havolalar (`load-test-report.md` va h.k.) tiklansin yoki o'chirilsin.
- B5 Redis fail-open ongli qaror + alert; B6 migratsiya rollback siyosati; B7 sessiya teskari indeks.

---

## 5. Tavsiya etilgan tartib (bir qatorda)

```
FAZA 0 (shart):  P0-1 deploy → P0-2 real video → P0-6 F1 → P0-4 Sentry → P0-3 backup → P0-5 crash
FAZA 1 (pilot):  P1-1 real dars → P1-2 cookie → P1-3 lockout → P1-4 TURN → P1-5..8
FAZA 2 (keyin):  haptika+skeleton+illyustratsiya → refaktorlar → a11y → infra/hujjat
```

**Minimal "boshlash" to'plami (faqat 🔴):** P0-1, P0-2, P0-3, P0-4, P0-5, P0-6 — 6 band.
Shular bajarilgach: **cheklangan pilotga (3-5 ustoz) tayyor**, ochiq bozorga hali emas.
