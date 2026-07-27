# Darsly — Acceptance Criteria (PM)

> Har bir band **yopilishi** kerak. PM 100% tasdiqlamaguncha loyiha "tayyor" emas.
> Format: [ ] ochiq, [x] yopilgan.

## A. Auth va profil
- [ ] Ro'yxatdan o'tish (full_name/email/parol) → avtomatik login → dashboard
- [ ] Login/logout ishlaydi; sahifa yangilanganda sessiya saqlanadi (token localStorage)
- [ ] Access token muddati tugasa refresh bilan avtomatik yangilanadi (foydalanuvchi sezmaydi)
- [ ] 401 → login sahifasiga yo'naltiriladi
- [ ] Profil tahrirlash (ism/timezone/til) va parol o'zgartirish ishlaydi
- [ ] Xato holatlar o'zbekcha tushunarli (noto'g'ri parol, email band, va h.k.)

## B. Dars (mentor)
- [ ] Dars yaratish (nom, tavsif, vaqt, davomiylik, parol, recording/waiting-room toggle)
- [ ] Darslar ro'yxati (kartalar) — loading/empty/error holatlari bor
- [ ] Jadval — kunlar bo'yicha guruhlangan
- [ ] Join havolasini nusxalash
- [ ] Darsni tahrirlash/o'chirish

## C. Join-link (guest)
- [ ] `/r/:slug` — dars preview (nom, mentor, parol/waiting-room holati)
- [ ] Ism kiritib qo'shilish; parolli darsda parol so'raladi
- [ ] Noto'g'ri parol → tushunarli xato; yaroqsiz slug → "yaroqsiz havola" ekrani
- [ ] Waiting-room yoqilgan bo'lsa → kutish xonasiga o'tadi
- [ ] Waiting-room o'chiq bo'lsa → to'g'ridan-to'g'ri xonaga

## D. Kutish xonasi
- [ ] Guest kutish ekranida (dars nomi, "ustoz tasdiqini kutmoqda")
- [ ] Mentor kutayotgan so'rovlarni ko'radi (real-time WS push)
- [ ] Mentor admit → guest avtomatik xonaga kiradi (WS yoki polling)
- [ ] Mentor reject → guest "rad etildi" ekranini ko'radi

## E. Jonli xona (LiveKit)
- [ ] Host kamera/mikrofon bilan ulanadi; video ko'rinadi
- [ ] Gallery va speaker view; ekran ulashilsa katta sahnaga o'tadi
- [ ] Ulanish holati indikatori (Yaxshi / Qayta ulanmoqda…)
- [ ] Internet uzilib qayta ulanganda foydalanuvchi chalkashmaydi (auto-reconnect)
- [ ] Mic/cam/screenshare toggle ishlaydi
- [ ] Host: mute-all, bitta ishtirokchini mute/kick, so'zlashga ruxsat/bekor
- [ ] Chat (host saqlaydi, guest data-channel) — real-time
- [ ] Reaksiya (uchuvchi emoji), qo'l-ko'tarish
- [ ] So'rovnoma: host yaratadi/yopadi, guest ovoz beradi, natija ko'rinadi
- [ ] Yozib olish start/stop (host); REC indikatori
- [ ] Chiqish/yakunlash

## F. Yozuvlar / bildirishnoma
- [ ] Yozuvlar ro'yxati; tayyor yozuvni yuklab olish (presigned URL)
- [ ] Bildirishnomalar ro'yxati, o'qildi/barchasi-o'qildi, qo'ng'iroqda o'qilmagan soni
- [ ] Real-time bildirishnoma push (WS)

## G. Chekka holatlar / sifat
- [ ] Har ekran loading/error/empty holatlarga ega
- [ ] Yaroqsiz link, noto'g'ri parol, tarmoq uzilishi — mos ishlov
- [ ] Production'da `console.log` yo'q
- [ ] Xavfsizlik: student mentor endpoint'iga kira olmaydi; parol-reset sessiyani o'ldiradi

## H. Non-func
- [ ] Frontend build toza (`npm run build`)
- [ ] Backend build+test toza (`go build ./... && go test ./...`)
- [ ] MSW mock bilan frontend backend'siz ishlaydi
- [ ] Playwright smoke test o'tadi
- [ ] Past-internet: adaptiveStream+dynacast+simulcast yoqilgan
