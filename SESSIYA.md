# SESSIYA — 2026-08-03/04 · Yozuv sifati, server sig'imi, sinov to'plami

> **Bu hujjat keyingi sessiyaga TOPSHIRIQ.** Yangi sessiyani boshlashdan oldin
> shuni o'qitib yuboring. Maqsad — nima qilinganini, **qanday qarorlar
> qabul qilinganini** va nima ochiq qolganini aniq bilib turish.
>
> Boshlang'ich nuqta: `fa4182b` · Yakuniy: `2df0ccb` · 10 ta commit, 43 fayl.

---

## 0. Bir jumlada

Asoschi Zoom va Jonly yozuvlarini taqqoslashni so'radi. Tahlil natijasida yozuv
sifati **uch qatlamda** tuzatildi, so'ng emulyator + brauzer + server bilan
uchdan-uchga o'lchandi va **serverning CPU chegarasi** aniqlandi. Yo'l-yo'lakay
uchta haqiqiy nuqson topildi va tuzatildi.

**Asosiy natija:** tik telefondan ulashilganda yozuv **324×718 → 458×1022**
(+41% chiziqli), mayda formulalar endi o'qiladi. Lekin to'liq parite
(576×1280) uchun **server kuchli bo'lishi kerak** — hozirgi 4 yadro yetmaydi.

---

## 1. QARORLAR (savol-javob orqali kelishilgan)

Bular allaqachon KELISHILGAN — qayta so'ramang, faqat amal qiling.

### 1.1. «Noaniq joyda Zoom qanday qilgan bo'lsa, shunday qilamiz»
Loyihaning umumiy qoidasi (avvalgi sessiyalardan). Shu sessiyada ham ikki
marta qo'llandi: yozuv kompozitsiyasi va ulashishda ilovaning fonga o'tishi.

### 1.2. Yozuv sifati — «kamida Zoom kabi»
Asoschi: *«video sifati juda yaxshi bo'lishi kerak»* — sababi: **APK matematika
ustoziga beriladi**, ya'ni formulalar va mayda belgilar o'qilishi shart.
Shundan kelib chiqib **rezolyutsiya fps'dan ustun** qo'yildi (pastda 1.5).

### 1.3. Zoom modeli o'rganildi va QAYTARILDI
Asoschi so'radi: *«zoomni yechimni ko'rib chiq»*. Natija — Zoom bu muammoni
hal qilmagan, **undan chetlab o'tgan**: uning bulutli yozuvi bir nechta
ALOHIDA fayl chiqaradi (*Active Speaker*, *Gallery View*, **Shared Screen**,
*Audio only*). Asoschi bergan 1280×800 fayl — aynan «Shared Screen».

**Qaror:** ikki fayl chiqarish (Zoom'ning aynan o'zi) qilinMADI — o'rniga
**kvadrat kadr** tanlandi, u xuddi shu natijani BITTA fayl bilan beradi.

### 1.4. Kvadrat kadr — asosiy texnik qaror
Kadr endi **kvadrat** (default 1024×1024). Sabab: egress shabloni ekranni
`object-fit: contain` bilan chizadi, ya'ni kvadratda **har qanday
orientatsiyaning uzun tomoni to'liq ishlatiladi**; ortiqcha maydonni transkod
kesadi.

Bu ikkita ildiz sababni **chetlab o'tadi** (ikkalasini ham tuzatishga hojat
qolmaydi):
1. **Vaqt** — yozuv birinchi trekda (mikrofon) boshlanadi, ekran ulashish
   kadr tanlanayotganda xonada hali yo'q;
2. **LiveKit Android SDK yolg'on o'lcham aytadi** — bytecode'dan tasdiqlangan
   (pastda 2.2).

### 1.5. fps 25 → 15 (CPU byudjeti rezolyutsiyaga berildi)
Avval Zoom pariteti uchun 25 fps qo'yilgan edi. Serverda o'lchov shuni
ko'rsatdiki, 25 fps bilan egress mashinani to'ldiradi va **yozuv yo'qoladi**.

**Qaror:** dars yozuvida talab — **matn o'qilishi** (rezolyutsiya), fps esa
faqat skroll silliqligiga ta'sir qiladi. Shuning uchun fps 15 ga tushirildi.
Ikkalasi ham env bilan sozlanadi (`RECORDING_CANVAS`, `RECORDING_FPS`).

### 1.6. «Yozuvni yo'qotish sifatdan beqiyos qimmat»
Default sozlama eng **xavfsiz** nuqtada, eng yuqori sifatda emas. Sinov
paytida bir yozuv haqiqatan yo'qoldi va bu qoidani mustahkamladi.

### 1.7. Ulashishda ilova fonga o'tadi (Zoom xulqi)
Asoschining yozuvida o'quvchilar dars materiali o'rniga Jonly interfeysini
ko'rgan. Zoom ulashish boshlanishi bilan ilovani yig'adi. **Qaror:** biz ham
shunday qilamiz; sozlama «Ko'proq» panelida, **default YONIQ**.
Emulyatorda ISHLASHI TASDIQLANDI.

### 1.8. Hamma tekshiruv server va serverdagi APK bilan
Asoschi: *«hamma checkni localda run qilmay»*. Shu sessiyada APK **serverdan
yuklab olinib** o'rnatildi (`sha256` bilan solishtirildi), barcha o'lchovlar
serverga qarshi olindi.

### 1.9. Reliz sanasi: 2026-yil 15-avgust
O'zgarmadi.

---

## 2. NIMA QILINDI

### 2.1. Video tahlili (boshlanish nuqtasi)

Asoschi ikki fayl berdi. `ffprobe` va kadrlar bilan o'lchandi:

| | Zoom | Jonly (o'sha payt) |
|---|---|---|
| Kadr | 1280×800 (**manba nisbati**) | 1280×720 (**qotib qolgan**) |
| Kontent pikseli | 1 024 000 | 230 400 (**22%**) |
| Kadr/sek | 25 | 15 |
| Ovoz | 48 kHz stereo | 44.1 kHz mono |

Uch nuqson: 78% kadr qora yo'l · yozuvda **Jonly interfeysining o'zi**
ko'rinardi · boshida ~15 s qora.

### 2.2. ⭐ LiveKit Android SDK yolg'on o'lcham aytadi (bytecode dalili)

Bu sessiyaning eng muhim texnik topilmasi. `livekit-android 2.27.0` bytecode'i
o'qildi:

```
LocalParticipant$publishVideoTrack$5 → AddTrackRequest.setWidth/Height
                                     ← LocalVideoTrack.getDimensions()
                                     ← options.captureParams (XOM qiymat)

LocalScreencastVideoTrack.startCapture():
    tik qurilmada capturer'ga (576,1280) beradi   ← HAQIQIY kadr
    lekin captureParams ga QAYTA YOZMAYDI
```

**Ya'ni tik telefonda kadr 576×1280 oqadi, `TrackInfo` esa 1280×576 deydi.**
Keyin ham yangilanmaydi (`UpdateLocalVideoTrack` protobuf bor, lekin SDK unga
murojaat qilmaydi). `TrackInfo.layers` ham yordam bermaydi.

⚠️ **Xulosa: `TrackInfo.width/height` ni Android publisher uchun ishonchli deb
bo'lmaydi.** Kelajakda kimdir "kadrni manbaga moslaymiz" desa, shuni eslatng.

### 2.3. Yozuv sifati — uch qatlam

| Qatlam | Fayl | Nima qiladi |
|---|---|---|
| 1. Manba | `ScreenCaptureSize.kt` | Ulashish o'lchami qurilma **nisbatidan** — qora yo'l manbada tug'ilmaydi |
| 2. Kompozitsiya | `livekit/egress.go` | Kadr **kvadrat**; layout `speaker` → **`single-speaker`**; 48 kHz |
| 3. Kafolat | `worker/videofilter.go`, `transcode_analyze.go` | `cropdetect` bilan qora yo'l kesiladi; boshidagi **qora VA jim** qism tashlanadi |

`single-speaker` qarori egress konteyneridagi **kompilyatsiya qilingan shablon
JS'i** o'qib tasdiqlandi: `speaker` yon karusel ustunini chizadi (kadrning
~1/4 i), `single-speaker` esa faqat birinchi trekni.

Boshni kesish faqat **qora VA jim** bo'lganda ishlaydi — ustozning «salom,
hozir ekranni ulashaman» degani yo'qolmasin.

### 2.4. ⭐ Server CPU chegarasi (o'lchangan)

Serverda (4 yadro; postgres/redis/minio/caddy/prometheus/grafana/loki **va ikki
Telegram bot** bilan birga):

| Kanvas | fps | Piksel/s | Natija |
|---|---|---|---|
| 1280×720 | 15 | 13.8 M | ishlagan (eski holat) |
| 1280×1280 | 25 | 41.0 M | ❌ «pipeline frozen» — **YOZUV YO'QOLDI** |
| 1024×1024 | 25 | 26.2 M | ishlagan, lekin egress **1027% CPU**, yuk 10–13 |
| **1024×1024** | **15** | **15.7 M** | ✅ tanlangan default |

Yakuniy sozlama eski bilan deyarli bir xil CPU yeydi, lekin **+41% rezolyutsiya**
beradi.

### 2.5. Yozuv sifati — yakuniy o'lchov

| | Kadr | Mayda formulalar |
|---|---|---|
| Avval | 324×718 | xira |
| **Hozir (1024)** | **458×1022** | **o'qiladi** |
| 1280 bilan (kuchli server) | 576×1280 | to'liq parite |

Qora yo'l yo'q · 48 kHz stereo · o'quvchi **576×1280** oqim oladi.

### 2.6. Kechikish (o'lchangan, ±9 ms aniqlik)

| | Qiymat |
|---|---|
| Shishadan-shishagacha | min **189** · o'rtacha **456** · maks **629** ms |
| Tarmoq RTT | **226 ms** (ping bilan 128 ms) |
| Jitter-bufer | **395 ms** |
| Kadr yo'qotish | **0** |

Usul: ustoz ekranida xost soatiga HTTP orqali sinxronlangan ms-soat, o'quvchi
brauzerida kadr surati. (`adb shell date` bilan aniqlik ±53 ms edi — yaroqsiz.)

⚠️ **Bu raqam EMULYATORNIKI.** Bufer emulyator kadrlarni 2–18 fps oralig'ida
notekis bergani uchun shishgan. Oldingi **haqiqiy** lokal o'lchovda 85 ms,
bufer 6 ms chiqqan edi.

**Server: Fransiya (Lauterbourg, Contabo), ping RTT 128 ms.** Zoom'ning
O'zbekistonga eng yaqin markazi ham Yevropada, ya'ni masofa bo'yicha
Zoom'dan kam emasmiz.

### 2.7. Tuzatilgan nuqsonlar

1. **Arxivdagi chat sinxroni** — kesish chat sakrashini buzardi. Nol nuqta endi
   `lesson.started_at` emas, **video t=0** (`Recording.PlaybackZero`).
   Bu egress kechikishi sabab **avvaldan** mavjud siljishni ham yopdi.
   Migratsiya: `000031_recording_content_offset`.
2. **Web pleyerdagi qat'iy `aspect-ratio: 16/9`** — tik yozuvda qora yo'llarni
   UI darajasida qaytarardi.
3. **«O'tgan darslarni ham ko'rsatish» tumbleri ta'sir qilmasdi** — tezkor
   darslar (sanasi yo'q) jadvalda abadiy qolardi. «O'tgan» endi = sanasi
   o'tgan **YOKI** holati `ended`.
4. **Transkod fps'ni manbadan yuqoriga ko'tarardi** — 15 fps manbani 25 ga
   cho'zib, kadrlarni takrorlardi.
5. **`APP_ANDROID_APK_URL`** rebranddan qolgan `darsly-mentor.apk` ga ishora
   qilardi (fayllar `jonly-*.apk`) — ilova ichidagi yangilanish 404 olardi.
6. **Deploy checklist'dagi `/metrics`–`/swagger` tekshiruvi yolg'on tinchlik
   berardi** — Caddy SPA fallback'i har qanday yo'lga 200 qaytaradi.

### 2.8. Sinov to'plami (yangi)

`tests/manual/` — hammasi repozitoriyda, vaqtinchalik papkada emas:

```
README.md            ikki stsenariy, o'lchov mezonlari, tuzoqlar
emulator-lesson.sh   ustoz oqimi (kirish → dars → ulashish → sahifa)
student-latency.mjs  o'quvchi + kechikish + getStats
clock/               xost soatiga sinxronlangan o'lchagich + etalon slayd
```

`backend/tests/load/livekit_load` ga **`-slug`** bayrog'i qo'shildi: busiz
vosita o'z bo'sh darsini yasardi va o'quvchilar hech narsaga obuna bo'lmasdi
— SFU yuki o'lchanmasdi.

### 2.9. Sinalgan funksiyalar (emulyator + server)

**Ishlaydi ✅:** kirish/chiqish · darslar · tezkor dars · jadval · xona
(mikrofon, kamera, ulashish, ishtirokchilar, chat + fayl, reaksiyalar,
so'rovnoma, «Ko'proq») · yozuvlar (yiqilgan yozuv aniq xabar bilan) ·
bildirishnomalar · kabinet · qora ro'yxat · havola ulashish · «bitta qurilma»
siyosati · **serverga qarshi 17/17** to'liq oqim sinovi (`frontend/final-sweep.mjs`).

---

## 3. OCHIQ QOLGAN — asoschidan javob kutilmoqda

1. **Server quvvatini oshirasizmi?** 1280 kanvas + 25 fps (to'liq parite,
   576×1280) uchun hozirgi 4 yadro yetmaydi. Bu **server sotib olish
   qarori**, kod ishi emas. (backlog №29)
2. **Jonli darsni ro'yxatdan yakunlash tugmasi** kerakmi? Hozir faqat xona
   ichidan yakunlanadi. (№30)
3. **Xonaga qayta kirilganda ikkinchi yozuv boshlanadi** — bitta darsdan ikki
   fayl. Shundayligicha qoldiraymi (dars qolgan qismi yozilishi uchun) yoki
   boshqacha? (№31)

---

## 4. KEYINGI QADAM (kelishilgan, hali bajarilmagan)

### 4.1. ⏳ «Bitta ustozga bu server yetadimi» sinovi

Asoschi so'radi, men stsenariyni yozdim, **emulyator to'xtatilgani uchun
o'tkazilmadi**. To'liq tartib: `tests/manual/README.md` → 1-STSENARIY.

Qisqasi: ustoz emulyatorda ekran ulashadi + yozuv ketadi + `-n` o'quvchi
qo'shiladi (**10 → 25 → 50**), har bosqichda `recordings.status` tekshiriladi.

**Qaror mezoni:** yozuv `ready` + yuk < 8 → yetadi. `failed` yoki yuk > 10 →
server kuchaytirilsin.

⚠️ Sinovdan keyin darsni **albatta yakunlang** — aks holda jonli qolib, yozuv
CPU yeyaveradi (bu sessiyada shunday bo'ldi).

### 4.2. ⏳ Zoom bilan kechikishni bir xil usulda o'lchash

Asoschi so'radi: *«bu raqamlar Zoom bilan qanchalik farq qiladi»*. Halol javob
— **bilmayman**, chunki Zoom shu sharoitda o'lchanmagan. Nashr etilgan
raqamlar boshqa mamlakat/tarmoq/usulda olingan.

Yagona to'g'ri yo'l: **aynan shu usul bilan** Zoom'ni o'lchash (bir tomonda
soat sahifasi, ikkinchi tomonda kadr surati, bir xil mashina va tarmoq).
Asboblar tayyor, Zoom uchun hech narsa qo'shish shart emas. ~30 daqiqa.

---

## 5. AMALIY MA'LUMOT

- **Server:** `169.58.104.245` (Fransiya, Contabo). Parol —
  `deploy/server/SERVER-CREDENTIALS.txt` (git'da yo'q).
  ⚠️ Serverda asoschining **ikki Telegram boti** ham ishlaydi — tegmang.
- **APK (v1.3.0, versionCode 5):**
  `https://app.169.58.104.245.sslip.io/download/jonly-arm64.apk` (16 MB)
  · brauzer to'xtatib qo'ysa `.bin` variantidan foydalaning
  · ⚠️ **emulyator uchun `jonly-mentor.apk`** (universal, x86_64 bor)
- **Emulyator:** AVD `darsly34`, 1080×2400, x86_64.
  ⚠️ **`-gpu host` SHART** — dasturiy GPU bilan 6 fps chiqadi va kechikish
  o'lchovi emulyatorning sekinligini o'lchab qo'yadi.
- **Admin:** `admin@darsly.uz`, parol serverdagi `.env` da (`SEED_ADMIN_PASSWORD`).
  ⚠️ **Bitta faol sessiya**: API orqali login emulyatordagi sessiyani bekor qiladi.
- **Sozlamalar:** `RECORDING_CANVAS` (default 1024), `RECORDING_FPS` (default 15).

---

## 6. TUZOQLAR (bu sessiyada vaqt yo'qotgan joylar)

- **APK ABI** — `jonly-arm64.apk` emulyatorga o'rnatilmaydi (emulyator x86_64).
- **Klaviatura** — `adb input text` dan keyin klaviatura ochiq qoladi va
  keyingi `tap` tugmaga emas, klaviaturaga tushadi.
- **Dars yakunlash dialogi** — tugma joyi ulashish yoniq/o'chiq holatda
  **siljiydi**; bosishdan oldin `screencap` bilan tekshiring.
- **Kvadrat kadrning yon ta'siri** — LiveKit shabloni ekranni butun holicha
  (`contain`), **kamerani esa to'ldirib** (`cover`) chizadi. Ya'ni faqat
  kamerali darsda kadr kvadrat **kesiladi**. Asosiy stsenariy ekran ulashish
  bo'lgani uchun qabul qilingan.
- **Aralash darsda kesish ishlamaydi** — ham ulashish, ham faqat-kamera davri
  bo'lsa, kesish qarori namunalarning **birlashmasini** oladi va hech narsa
  kesilmaydi (aks holda kamera qirqilardi). Natija kvadrat qoladi:
  rezolyutsiya saqlanadi, kompozitsiya emas.
- **`docker stats` bir zumlik o'lchov** — faoliyatdan keyin darhol o'qilsa
  yolg'on yuqori qiymat beradi. Bir necha marta o'lchang.
