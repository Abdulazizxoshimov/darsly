# Darsly Mobile — QURILMA SINOVI: holat va qolgan ishlar

> Sana: **2026-07-26 (23:35)** · Qurilma: **HONOR ABR-LX1, Android 15 (API 35)**, adb `AY3SUT5109002441`
> Bu fayl — qurilma ulanganda davom ettirish uchun. Kod holati: `docs/MOBILE-STATUS.md`

---

## 1. Ish boshlash (har safar)

```bash
export PATH="$PATH:$HOME/Android/Sdk/platform-tools"
adb devices -l                       # qurilma ko'rinishi kerak
adb shell svc power stayon true      # quvvatda ekran o'chmasin (R0 dagi USB uzilishi muammosi)
cd mobile && ./gradlew assembleDebug && adb install -r app/build/outputs/apk/debug/app-debug.apk
for p in CAMERA RECORD_AUDIO POST_NOTIFICATIONS; do adb shell pm grant uz.darsly.mentor android.permission.$p; done
```

⚠️ **Wi-Fi adb bu tarmoqda ISHLAMAYDI.** `adb tcpip 5555` yoqiladi, lekin kompyuter
telefonga yeta olmaydi (`No route to host`, ARP javob bermaydi) — router mijozlar
orasidagi **kiruvchi** trafikni bloklaydi (telefon→kompyuter ping o'tadi, teskarisi yo'q).
Shuning uchun **USB kabel ulangan qolishi kerak**; ekran qulflanmasligi `stayon` bilan hal qilingan.

⚠️ **Telefon qulflangan bo'lsa men ocha olmayman** — barmoq izi/PIN foydalanuvchida.
Uzoq sinovdan oldin qulfdan chiqarib, **quvvat tugmasiga tegmang**.

⚠️ **Ekran ulashish sinovida telefondagi hamma narsa "o'quvchi"ga uzatiladi** va
skrinshotlarga tushadi. Sinovdan oldin **DND** yoqing yoki bildirishnomalarni tozalang.

---

## 2. Asboblar

### 2.1 "O'quvchi" o'lchov klienti (kB/s bo'yicha)

`scratchpad/listener/` (Go, `server-sdk-go v2`). Xonaga guest bo'lib ulanadi va **har trek
bo'yicha** soniyalab bayt/sek yozadi. Nega quloq emas: Opus DTX tufayli **jimlik ≈0.1 kB/s**,
ovoz esa 2–8 kB/s — ya'ni "ovoz o'tdimi" savoliga raqam bilan javob beradi.

```bash
cd <scratchpad>/listener
./listener -slug <join_slug> -name "Oquvchi" -sec 40 2>/dev/null | grep -vE "^20[0-9]{2}/"
```

> Manba kodni saqlab qo'ying — u repo'ga kirmagan (ataylab: sinov asbobi, mahsulot emas).

### 2.2 Nazorat ostidagi ovoz manbai

`scratchpad/darsly_tone.wav` — 120 s, har 2 soniyada 440↔880 Hz. Telefonga:

```bash
adb push darsly_tone.wav /sdcard/Music/
adb shell content call --uri content://media --method scan_file --arg /sdcard/Music/darsly_tone.wav
# so'ng musiqa pleyeridan qo'lda ochish ishonchliroq (media tugmalari HONOR'da toggle qiladi)
```

### 2.3 Matn bo'yicha bosish (koordinata qotirmasdan)

Xona ekranida xato kartasi/indikator chiqsa tugmalar **suriladi** — qotirilgan koordinata
noto'g'ri joyga uradi (bugun shu sabab bir necha marta adashildi). To'g'ri usul:

```bash
adb shell uiautomator dump /sdcard/ui.xml && adb pull /sdcard/ui.xml
# <node> larni parse qilib, text/content-desc bo'yicha bounds markazini bosish
```

---

## 3. ✅ BUGUN QURILMADA TASDIQLANGANI

| # | Nima | Dalil |
|---|---|---|
| 1 | **Mentor roli bilan to'liq oqim** (avval faqat `admin` sinalgandi) | `spike.mentor@darsly.uz` bilan login → ro'yxat → xona → ekran ulashish ishladi |
| 2 | **Tenant izolyatsiyasi** | Mentor ro'yxatda faqat o'z darsini ko'rdi; admin yaratgan dars ko'rinmadi |
| 3 | RBAC (curl) | `GET/POST /lessons` 200 · **`POST /lessons/:id/token` 200** · `GET /users` **403** |
| 4 | **B-3** bo'sh holat | "Hali dars yo'q" + tugma to'g'ri chiqdi |
| 5 | **B-5** uchdan-uchga | Ilovadan yaratilgan dars serverda paydo bo'ldi (`join_slug` bilan) |
| 6 | **🟢I** | Bo'sh formada "Yaratish" **o'chiq** (kul rang), matn kiritilgach yondi |
| 7 | **🟢H** | Dialog yopib qayta ochilganda forma **bo'sh** (eski matn qolmadi) |
| 8 | **M6** ko'rinish | "Jonli"/"Rejalashtirilgan" bo'limlari, davomiylik, "Ulashish", snackbar |
| 9 | **M12** | Holat qatorida "Kamera: yoniq (old)", tugmada "Orqa kameraga" |
| 10 | **B-1 🔴 muammosi JONLI TASDIQLANDI** | Tizim dialogi haqiqatan **"A single app"** ni tanlab qo'ygan; bizning tushuntirish dialogimiz undan oldin chiqdi va "Entire screen" tanlash mumkin bo'ldi |
| 11 | **M16** | Aloqa yomonlashganda ekranda **"Aloqa yo'q"** qizil yozuvi chiqdi |
| 12 | **⭐ C-6 to'liq tasdiqlandi** | Batafsil §4 — mute'dan keyin video ovozi davom etadi, ustoz ovozi sizmaydi (nazorat sinovi bilan) |
| 13 | **B-6 maxfiylik** ✅ | Serverdan `POST /end` → "Dars tugadi. Ekran ulashish to'xtatildi", hamma media o'chdi, `dumpsys media_projection` **bo'sh**, FGS **0** |
| 14 | **⭐ C-3 fon rejimi** ✅ | Ilova **fonda** (`com.android.settings` old planda), ekran aylantirilganda o'quvchiga **32–184 kB/s** oqim ketdi. Statik ekranda ~0 kB/s — bu **to'g'ri** xulq (H.264 harakatsiz kadr uzatmaydi), nosozlik emas |
| 15 | **🟢L "Orqaga"** ✅ | Jonli darsda "Darsdan chiqasizmi?" · "Darsda qolish" → xonada qoladi · "Chiqish" → ro'yxat, FGS **0** |
| 16 | **C-13** ✅ | MediaProjection dialogida **Deny** → crash yo'q, "Ekran: o'chiq", tugma qayta tayyor |
| 17 | **C-14** ✅ | `RECORD_AUDIO` rad etildi → crash yo'q, "Mikrofonga ruxsat berilmadi" + "Ruxsat so'rash"/"Sozlamalar" |
| 18 | **B-4 offline** ✅ | Tarmoq uzilgach qizil "Internet yo'q — saqlangan ro'yxat" + "Oxirgi yangilanish: Bugun 23:13" + keshdagi ro'yxat; tarmoq qaytgach yo'qoldi |
| 19 | **B-2 pull-to-refresh** ✅ | Pastga tortish yangi darsni ro'yxatga keltirdi |
| 20 | **M8 ulashish** ✅ | Tizim chooser'i **Telegram** bilan ochildi |
| 21 | **A-2 token shifrlash** ✅ | `run-as` bilan `darsly_session.xml`: faqat Tink keyset + shifrlangan blob, **`eyJ` (JWT) topilmadi** |
| 22 | **A-1 sessiya saqlanishi** ✅ | `force-stop` → qayta ochish → login so'ralmadi, to'g'ridan-to'g'ri darslar ro'yxati |
| 23 | **A-5 jim refresh** ✅ | Access token eskirganda ilova o'zi `POST /auth/refresh` qilib so'rovni qayta yubordi — foydalanuvchi sezmadi |
| 24 | **M16 jonli** ✅ | Wi-Fi uzilishi bilan "Qayta ulanmoqda…" + indikator chiqdi |
| 25 | **M12 bloklanishi** ✅ | Kamera o'chiq bo'lganda "Orqa kameraga" tugmasi o'chiq |

---

## 4. ✅ C-6 HAL BO'LDI — qurilmada o'lchov bilan tasdiqlangan (22:55)

### 4.0 Yakuniy natija

| Tekshiruv | O'lchov | Xulosa |
|---|---|---|
| Ekran audiosi capture'i | Dinamik **mute** holatida mikrofon treki tonni **5 kB/s** bilan uzatdi | ✅ akustik yo'l emas, haqiqiy `AudioPlaybackCapture` |
| **Mute → video ovozi davom etadimi** | Mute bosilgach oqim **~5 kB/s** da davom etdi (t=12…40 s) | ✅ Zoom xulqi |
| **Maxfiylik: ustoz ovozi sizadimi** | Mute + baland ovozda gapirish → **0.1–0.2 kB/s** (30 s, hech qachon oshmadi) | ✅ chiqmaydi |
| **Nazorat** (test o'zi yaroqlimi) | Mikrofon yoniq + gapirish → **1.0–3.8 kB/s** cho'qqilar | ✅ mikrofon yo'li ishlaydi, farq 6.6× |
| Ilova UI'si rostmi | "Mikrofon: o'chiq · Ekran audiosi: yoniq" | ✅ |

### 4.1 Dizayn o'lchov tufayli O'ZGARDI

Dastlabki reja — **"ikki trekni navbatlashtirish"** (mikrofon va `screen_share_audio`
treklarini almashtirib turish) — **qurilmada yiqildi**:

> O'quvchi `SCREEN_SHARE_AUDIO` trekiga **umuman obuna bo'lmadi** (`OnTrackSubscribed`
> chaqirilmadi), chunki trek e'lon paytida **mute** edi. Keyin uni unmute qilish ham
> obunani tug'dirmadi → mute'dan keyin o'quvchida **jimlik**.

Ya'ni LiveKit mute holatidagi audio trekni obunachiga bermaydi va bu keyinchalik
tuzalmaydi. Kech qo'shilgan o'quvchi uchun ham shu muammo bo'lardi.

**Ishlaydigan dizayn — bitta trek, mazmun nazorati:**

| Ustoz niyati | Mikrofon treki (server) | ADM buferi | O'quvchi eshitadi |
|---|---|---|---|
| Mikrofon YONIQ | ovozli | mikrofon + ekran ovozi | ustoz + video |
| Mikrofon O'CHIQ | **ovozli qoladi** | **mikrofon nollangan** + ekran ovozi | **faqat video** |

Ustozning ovozi **kodlashdan oldin** o'chiriladi. Qo'shimcha yutuq: ovoz mikrofon
trekida ketgani uchun **web tomonda o'zgarish kerak emas** (u bu trekni allaqachon o'ynatadi).

### 4.2 Ma'lum cheklov (4-blokda yopiladi)

Trek serverda "unmuted" bo'lgani uchun **web'dagi ishtirokchilar ro'yxati ustozni mute
deb ko'rsatmaydi**. To'g'ri holat data-channel orqali yuborilishi kerak (M29 / D-8).
Ilovaning o'z ekrani rost gapiradi. Ekran ulashilmayotganda oddiy, haqiqiy mute ishlaydi —
ya'ni chetlanish faqat ulashish davomida amal qiladi.

### 4.3 Qolgan mayda savol

Mikrofon treki nutq uchun sozlangan (AEC/NS/AGC yoqilgan) — musiqa/video ovozi sifati
shundan pasayishi mumkin. Kerak bo'lsa ulashish davomida `LocalAudioTrack.applyOptions`
bilan ishlov berishni o'chirib ko'rish mumkin (o'lchov: tonni quloq bilan solishtirish).

---

### 4.4 Uslubiy saboqlar (o'lchovni buzgan omillar — takrorlamang)

Bugungi birinchi o'lchovlar **ziddiyatli** chiqdi va sababi asbobda emas, tajriba
sharoitida edi. Har biri kelgusi audio sinovlarida ham amal qiladi:

1. **Akustik yo'l.** Telefon ovozni **dinamikdan** chaladi, mikrofon esa uni eshitadi —
   "mikrofon trekida ovoz bor" degani "ekran audiosi ishlayapti" degani EMAS.
   → Ovoz balandligini **0** ga tushiring (`mutedState:streamVolume` bo'lishi kerak);
   `AudioPlaybackCapture` oqimni mikserdan oladi, shuning uchun capture baribir ishlaydi
   (bugun aynan shu bilan isbotlandi).
2. **Media tugmalari HONOR'da toggle.** `KEYCODE_MEDIA_PLAY` musiqani **to'xtatib**
   qo'ydi va keyingi o'lchov "jimlik" ko'rsatdi. → Har o'lchovdan oldin
   `dumpsys media_session | grep state=` bilan **PLAYING** ekanini tasdiqlang.
3. **Aloqa sifati.** O'lchov o'rtasida ilova "Aloqa yo'q" ko'rsatdi (kamera 200–390 kB/s
   yeyayotgandi). → Audio sinovida **kamerani o'chiring**; indikatorga qarab natijani
   qabul qiling.
4. **Nazorat sinovisiz xulosa yo'q.** "Ovoz chiqmadi" natijasi mikrofon buzilgani uchun
   ham bo'lishi mumkin. Shuning uchun maxfiylik testidan keyin **mikrofonni yoqib** ayni
   sharoitda takrorlash shart — bugun shu 6.6× farqni ko'rsatdi.
5. **Qotirilgan koordinatalar ishlamaydi.** Xona ekranida xato kartasi chiqsa tugmalar
   suriladi. → `uiautomator dump` + matn bo'yicha bosish (§2.3).

---

## 5. ⬜ QURILMADA HALI SINALMAGAN (C-6 dan tashqari)

| # | Ish | Qanday |
|---|---|---|
| ~~1~~ ✅ | ~~B-6 maxfiylik~~ | Web'dan `POST /lessons/:id/end` → telefonda "Dars yakunlandi" kartasi, bildirishnoma yo'qolishi, `adb shell dumpsys media_projection` da **faol sessiya yo'qligi** |
| 2 | **🟡A fon rejimi** | Ilovani fonga tashlab dars boshlash → abadiy spinner EMAS, xato + "Qayta urinish" |
| ~~3~~ ✅ | ~~🟢L "Orqaga"~~ | Jonli darsda tasdiq oynasi; "Chiqish" → FGS va proyeksiya bo'shashi |
| ~~4~~ ✅ | ~~C-3 fon rejimi~~ | PDF/GeoGebra ochib 10+ daqiqa — ulashish uzilmasligi (R0 da 28 daqiqa tasdiqlangan) |
| ~~5~~ 🔴 | ~~C-11 Wi-Fi ↔ 4G~~ — **YIQILDI, §5b** (LTE bor joyda takrorlansin) | Dars tugamasligi; "Qayta ulanmoqda…" chiqib yo'qolishi |
| ~~6~~ ✅ | ~~C-13/C-14~~ | MediaProjection dialogini **bekor qilish**, ruxsatlarni **rad etish** → crash yo'q |
| ~~7~~ ✅ | ~~B-2/B-4~~ | Pull-to-refresh imosi; **aviarejimda** offline chizig'i va keshlangan ro'yxat |
| ~~8~~ ✅ | ~~M8 Telegram~~ | "Ulashish" → Telegram chooser'i; havola o'quvchida ochilishi |
| 9 | **M8 nusxa olish** | "Nusxa olish" tugmasi → snackbar (Android 13+ da tizim o'zi ko'rsatadi) |
| ~~10~~ ✅ | ~~A-1/A-2~~ | `force-stop` → qayta ochish → kirgan holat; `run-as` bilan prefs faylida token **ochiq matnda yo'qligi** |

---

## 5b. 🔴 C-11 BAJARILMADI — tarmoq almashuvida dars TUGADI

**Mezon:** "Wi-Fi ↔ 4G almashganda dars **tugamaydi**, qayta ulanadi."

**Kuzatilgan:** ekran ulashilayotgan darsda Wi-Fi o'chirildi → "Qayta ulanmoqda…" ~90 soniya
turdi → LiveKit taslim bo'ldi → **"Dars tugadi. Dars uzildi"** kartasi. Ustoz qaytadan kirib,
ekranni **qaytadan ulashishi** kerak.

**Yaxshi tomoni:** uzilish **toza** kechdi — proyeksiya va FGS bo'shatildi, xabar tushunarli,
"Qayta boshlash" tugmasi bor. Ya'ni B-6 yo'li tarmoq uzilishida ham ishlaydi.

**MUHIM KONTEKST — sinov adolatli emas edi:** telefonda **LTE yo'q**, zaxira sifatida
**WCDMA (3G), signal 1/5, rscp −106 dBm**, ping 344–683 ms. Ya'ni bu "4G ga o'tish" emas,
"juda kuchsiz 3G ga tushish" bo'ldi. Foydalanuvchi ham shu hududda internet yomonligini
aytdi. **Xulosa: C-11 ni LTE bor joyda qayta sinash kerak.**

**Keyingi qadamlar (tekshirilmagan gipotezalar):**
1. LiveKit qayta ulanish siyosati/vaqtini sozlash mumkinmi (`ConnectOptions`/`RoomOptions`) —
   SDK'ni `javap` bilan aniqlash;
2. Tarmoq qaytganda **oshkora** qayta ulanish: `ConnectivityManager.NetworkCallback` →
   `room.reconnect()`;
3. Uzilgach ekran ulashishni **avtomatik tiklash** (hozir ustoz qo'lda qayta ulashadi);
4. LTE bor joyda takrorlash — muammo qanchalik tarmoqqa bog'liqligini ajratish.

---

## 5c. Kesh fayli haqida eslatma (maxfiylik)

`darsly_lessons_cache.xml` **shifrlanmagan** (ataylab): dars sarlavhalari va **`join_slug`**
ochiq matnda. Ilova xotirasi boshqa ilovalarga yopiq va release build'da `run-as` ishlamaydi,
lekin root qilingan qurilmada slug o'qilishi mumkin. Slug — darsga kirish kaliti.
Agar zarur deb topilsa, keshni ham `EncryptedSharedPreferences` ga o'tkazish arzon
(`PrefsLessonsCache` allaqachon `SharedPreferences` interfeysini oladi).

---

## 6. Bugun topilgan mayda kamchiliklar (kodda tuzatilishi kerak)

| # | Muammo | Joy |
|---|---|---|
| 1 | **"Davom etish"** tugmasi ikki qatorga bo'linib ketadi (kartada baland ko'rinadi) | `LessonsScreen.LessonCard` — `maxLines = 1` yoki qisqaroq matn |
| 2 | **B-7 (ma'lum edi, tasdiqlandi):** `Xona`/`Identity` qatorlarida uzun UUID satrlari maketni buzadi | `RoomScreen.Row2` — `ellipsize`/`maxLines` |
| 3 | Xona ekranida xato kartasi chiqqanda tugmalar suriladi (avtomatlashtirishga xalaqit, foydalanuvchiga ham "sakrash" effekti) | `RoomScreen` — boshqaruv tugmalari tepada, xabarlar pastda bo'lishi mumkin |
| 4 | O'lchovda `lvl=127..0` — RFC 6464 audio darajasi to'g'ri o'qilmadi (bayt/sek ishladi) | `scratchpad/listener` (asbob, mahsulot emas) |

---

## 7. TOZALASH RO'YXATI (sinov tugagach)

- [ ] **Telefonda ekran ulashish to'xtatilganini tekshirish** (bildirishnoma paneli) — bugun
      qurilma uzilganda ilova hali ishlab turgan bo'lishi mumkin
- [ ] `adb shell rm -f /sdcard/Music/darsly_tone.wav` (10 MB) va `darsly_qa_tone.wav` (52 MB, R0 dan)
- [ ] Media ovozi qaytarilsin — sinovda **15 pog'ona pasaytirilgan**
- [ ] Vaqtinchalik hisob: `spike.mentor@darsly.uz` — `DELETE /api/v1/users/:id` (admin bilan)
- [ ] Sinov darslari: `Can spike sinovi`, `Mentor RBAC sinovi` — `DELETE /api/v1/lessons/:id`
- [ ] `adb uninstall uz.darsly.mentor` (yakunda, toza o'rnatish sinovi uchun)
- [ ] Kodda qolgan diagnostika: `LessonSession.dumpPublications()` va `C6:` log'lari —
      C-6 yopilgach olib tashlansin (audio oqimidagi hisoblagich **allaqachon olib tashlangan**)

---

## 8. Kirish ma'lumotlari (sinov muhiti)

| | |
|---|---|
| Server | `https://app.194.163.139.242.sslip.io` |
| Admin (role=**admin**) | `admin@darsly.uz` · parol: `deploy/server/SERVER-CREDENTIALS.txt` |
| Mentor (role=**mentor**) | sinov uchun admin API bilan yasaladi, parol shu yerda saqlanmaydi |
| Boshqa hisoblar | `mentor@darsly.uz`, `student@darsly.uz` — **parollari noma'lum** (seed skripti yo'q) |

> `admin` roli `mentor` huquqlarini meros oladi (`policy.csv`), shuning uchun admin bilan
> sinash mentor bilan sinashni **almashtirmaydi** — bugun aynan shu sabab mentor hisobi yasaldi.
