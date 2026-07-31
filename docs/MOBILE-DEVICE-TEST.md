# Darsly Mobile — QURILMA SINOVI: holat va qolgan ishlar

> Sana: **2026-07-26 (23:35)** · Qurilma: **HONOR ABR-LX1, Android 15 (API 35)**, adb `AY3SUT5109002441`
> Bu fayl — qurilma sinovining EMPIRIK yozuvi: o'lchov usullari, platforma
> cheklovlari va takrorlanmasligi kerak bo'lgan xatolar. Ochiq ishlar: `docs/BACKLOG.md`

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
| ~~2~~ ✅ | ~~🟡A fon rejimi~~ — §5e | FGS `appops` bilan bloklandi → "Ulanmadi" + "Qayta urinish"; spinner yo'q |
| ~~3~~ ✅ | ~~🟢L "Orqaga"~~ | Jonli darsda tasdiq oynasi; "Chiqish" → FGS va proyeksiya bo'shashi |
| ~~4~~ ✅ | ~~C-3 fon rejimi~~ | PDF/GeoGebra ochib 10+ daqiqa — ulashish uzilmasligi (R0 da 28 daqiqa tasdiqlangan) |
| ~~5~~ ✅ | ~~C-11 Wi-Fi ↔ 4G~~ — **§5d da LTE'da BAJARILDI** | ~2.3 s ichida tiklandi, dars tugamadi |
| ~~6~~ ✅ | ~~C-13/C-14~~ | MediaProjection dialogini **bekor qilish**, ruxsatlarni **rad etish** → crash yo'q |
| ~~7~~ ✅ | ~~B-2/B-4~~ | Pull-to-refresh imosi; **aviarejimda** offline chizig'i va keshlangan ro'yxat |
| ~~8~~ ✅ | ~~M8 Telegram~~ | "Ulashish" → Telegram chooser'i; havola o'quvchida ochilishi |
| ~~9~~ ✅ | ~~M8 nusxa olish~~ — §5e | Tizim buferi oynasi havola bilan chiqdi (Android 13+ o'zi ko'rsatadi) |
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

---

### ✅ 5c. Ikkala gipoteza ham BAJARILDI (2026-07-28) — qurilmada qayta sinash kerak

Yuqoridagi ikki gipoteza tekshirildi va ikkalasi ham **to'g'ri chiqdi**; kod yozildi.

**1-gipoteza tasdiqlandi.** SDK artefaktini (`livekit-android-2.27.0`) ochib ko'rildi:
`RoomOptions.reconnectPolicy` mavjud (`ReconnectPolicy.getNextRetryDelay(ReconnectContext)`).
SDK default'i "internet yomon" holatiga mo'ljallangan — u KUTADI, chunki tez urinish zaif
kanalni battar bo'g'adi. Tarmoq ALMASHUVI esa boshqa hodisa: yangi interfeys odatda darhol
tayyor va kutish sof yo'qotish.

→ `data/livekit/LessonReconnectPolicy.kt`: dastlabki uch urinish **~1.7 s ichida**
(200 ms · 500 ms · 1 s), keyin backoff o'sadi (2→4→6→8→10 s), umumiy oyna **60 s**.
7 ta JVM testi bilan qotirilgan.

**2-gipoteza tasdiqlandi va kengaytirildi.** Faqat `NetworkCallback` yetarli emas: har qanday
tarmoq o'zgarishida uzib-ulash zarar keltiradi (bir Wi-Fi nuqtadan boshqasiga o'tish WebRTC
uchun shaffof). Shuning uchun qaror TRANSPORT o'zgarishiga bog'landi.

→ `data/livekit/NetworkMonitor.kt` (Android qismi) + `NetworkSwitch.kt` (sof qaror):
Wi-Fi↔LTE almashganda **majburiy** `disconnect()` + `connect()`; bir xil transport ichida
tegilmaydi; internet umuman yo'qolganda kutiladi (majburiy urinish shu zahoti yiqilardi).
8 ta JVM testi.

→ UI: xona ekranida **"Mobil internetga o'tildi"** yozuvi chiqadi va u aloqa yozuvidan
ustun turadi — ustoz "nima bo'ldi" ni bilmasa ilovani yopib qayta ochadi va dars
haqiqatan uziladi.

#### Qayta sinash protokoli (C-11)

> ⚠️ Bu testlar QARORNI qotiradi, qurilmadagi haqiqiy almashuvni EMAS.
> Quyidagi qadamlar **LTE qamrovi bor** joyda bajarilishi shart — avvalgi sinov
> WCDMA (1/5 signal, ping 344–683 ms) da o'tgan va adolatli emas edi.

| # | Qadam | Kutilgan natija |
|---|---|---|
| 1 | Wi-Fi'da darsni boshlang, ekranni ulashing | Ulashish ketmoqda, o'quvchi ko'ryapti |
| 2 | Wi-Fi'ni o'chiring (LTE yoqilgan holda) | ≤2 s ichida **"Mobil internetga o'tildi"** chiqadi |
| 3 | Kuting | **≤10 s** ichida dars tiklanadi; "Dars tugadi" kartasi CHIQMAYDI |
| 4 | Ekran ulashish holatini tekshiring | Ulashish davom etadi (qayta yoqish shart emas) |
| 5 | Wi-Fi'ni qayta yoqing | "Wi-Fi'ga o'tildi" → yana ≤10 s ichida tiklanadi |
| 6 | O'quvchi tomonini tekshiring | Uzilish ≤10 s, keyin video/ekran qaytadi |

**Yiqilsa nima yozish kerak:** ilova ichidagi **Diagnostika (debug)** panelini oching —
u har bir tarmoq hodisasini va qayta ulanish urinishini yozadi (logcat'dan qulayroq).

---

## ✅ 5d. C-11 BAJARILDI (2026-07-28, 02:23) — LTE'da tasdiqlandi

**Qurilma:** HONOR ABR-LX1 · Android 15 · **LTE (Uztelecom), RSRP −95 dBm, signal 4/5**,
Band 3 / 20 MHz / carrier aggregation. Ya'ni §5b dagi "sinov adolatli emas edi"
(WCDMA, 1/5, −106 dBm) sharti bartaraf etilgan.

**Natija:** Wi-Fi → LTE almashuvida dars **TUGAMADI**, ~2.3 soniyada tiklandi.
Teskari yo'nalishda (LTE → Wi-Fi) uzilish umuman bo'lmadi.

### Ilova diagnostikasidan olingan aniq ketma-ketlik

```
tarmoq: Internet yo'q
tarmoq: Mobil internetga o'tildi
tarmoq almashdi — majburiy qayta ulanish
uzildi: CLIENT_INITIATED
uzilish qayta ulanish uchun — dars davom etadi
qayta ulanish 1-urinish muvaffaqiyatsiz          ← DNS hali tayyor emas
uzildi: JOIN_FAILURE
uzilish qayta ulanish uchun — dars davom etadi
qayta ulandi (2-urinish)                          ← ~2.3 s
```

### Sinov davomida topilgan VA tuzatilgan uchta nosozlik

Birinchi urinish **yiqildi** va sabab kutilmagan bo'lib chiqdi — C-11 uchun yozilgan
tuzatishning o'zi darsni o'ldirardi. Uchala nosozlik ham faqat qurilmada ko'rindi:

| # | Nosozlik | Sabab | Tuzatish |
|---|---|---|---|
| 1 | Soxta "Mobil internetga o'tildi" va bekorga qayta ulanish | `registerNetworkCallback` **hamma** tarmoqlar uchun signal beradi; Wi-Fi va LTE bir vaqtda yoqiq bo'lsa oqim WIFI↔CELLULAR bo'lib tebranadi | `registerDefaultNetworkCallback` — faqat DEFAULT tarmoq |
| 2 | Qayta ulanish DNS'da yiqilib, **hech narsa qayta urinmasdi** | Transport signali tarmoq hali ishlamayotgan paytda keladi → `UnknownHostException`; bitta urinishdan keyin to'xtardi | `NET_CAPABILITY_VALIDATED` gating + backoff bilan 7 urinish |
| 3 | **Majburiy qayta ulanish darsni o'ldirardi** | `room.disconnect()` → `Disconnected(CLIENT_INITIATED)` → mavjud handler buni "ustoz chiqdi" deb tushunib MediaProjection va foreground servisni bo'shatardi | `intentionalReconnect` bayrog'i — bizning uzilishimiz jimgina o'tadi |

> Uchinchisi eng muhim: avtomatik testlar buni **umuman topa olmasdi**, chunki u ikki
> mustaqil to'g'ri komponentning o'zaro ta'siridan tug'iladi.

### Yo'l-yo'lakay topilgan boshqa nosozliklar (o'sha sessiyada tuzatildi)

| # | Nosozlik | Tuzatish |
|---|---|---|
| 4 | Boshqaruv paneli **telefonda sig'masdi** — 7-tugma ("Chiqish") yozuvi vertikal cho'zilib ekrandan chiqib ketardi (planshetda muammo ko'rinmasdi) | `BoxWithConstraints` — tugma o'lchami mavjud endan hisoblanadi |
| 5 | Bildirishnomadagi `✋ 1 · 👍 Ali` matni **pardada ko'rinmasdi** — Android ikkita doimiy bildirishnomani avtomatik guruhlab, faqat "Darsly Mentor" ko'rsatardi | Guruh o'zimiz e'lon qilinadi; dars bildirishnomasi — SARLAVHA (`setGroupSummary`) |
| 6 | **O'lik sessiyada boshi berk ko'cha**: foydalanuvchi DB'dan o'chirilgan, token esa hali yaroqli → `GET /users/me` 404 → ekranda "Topilmadi / Qayta urinish", chiqish tugmasi yo'q. Qayta urinish har safar aynan o'sha 404 ni qaytaradi; yagona chora ilova ma'lumotini tozalash edi | Profil 404 → **avtomatik chiqish** va kirish ekrani (`Session.forceLogout()`). 500/503 kabi vaqtinchalik xatolar sessiyani o'ldirmaydi — 2 test bilan qotirilgan |

> ✅ №6 endi **qurilmada ham tasdiqlandi** — §5e ga qarang.

**§5b dan qolgan gipotezalar:**
3. ~~Uzilgach ekran ulashishni **avtomatik tiklash**~~ — **§5f da bajarildi** (Android 14+
   cheklovi tufayli "bir bosishlik tiklash" shaklida);
4. Zaif tarmoqda (WCDMA/1-signal) takrorlash — muammo qanchalik tarmoqqa bog'liqligini ajratish.

---

## ✅ 5e. §5d dan qolgan ish yopildi (2026-07-28, 09:35–09:50)

Qurilma: o'sha HONOR ABR-LX1 · Android 15. Debug APK qayta yig'ilib o'rnatildi.
Sinov hisobi admin API bilan yasaldi va **sinov oxirida o'chirildi** (`devicetest@darsly.uz`,
`deadsession@darsly.uz`, `Nusxa olish sinovi` darsi — hammasi tozalandi).

### ✅ №6 (o'lik sessiya) — qurilmada tasdiqlandi

| Qadam | Natija |
|---|---|
| `deadsession@darsly.uz` (mentor) bilan kirildi, "Kabinet" ochildi | Profil normal ko'rindi |
| Admin `DELETE /users/:id` bilan foydalanuvchi o'chirildi (token hali yaroqli) | 204 |
| Ilovada "Darslar" → "Kabinet" ga qayta kirildi | **Kirish ekraniga avtomatik chiqarildi** — "Topilmadi / Qayta urinish" halqasi YO'Q |

### ✅ 🟡A fon rejimi (§5 № 2) — qurilmada tasdiqlandi

Ikki holat sinaldi:

1. **"Boshlash" bosilib darhol HOME** — dars odatdagidek boshlandi. Android bosishdan
   keyingi imtiyoz oynasida FGS'ga ruxsat beradi, ya'ni bu yo'l bilan xato holatiga
   umuman tushib bo'lmaydi (spinnerda qotish ham yo'q).
2. **FGS majburan bloklandi** — `adb shell cmd appops set uz.darsly.mentor START_FOREGROUND deny`.
   Natija: **"Ulanmadi" kartasi + "Qayta urinish"**, abadiy spinner emas. Appop qaytarilib
   "Qayta urinish" bosilgach xona to'liq tiklandi (sarlavha, REC, ishtirokchilar).

> Usul eslab qolinsin: `appops … START_FOREGROUND deny` — 🟡A ning xato shoxini
> qurilmada ishonchli qo'zg'atadigan yagona amaliy yo'l.

### ✅ M8 nusxa olish (§5 № 9) — qurilmada tasdiqlandi

"Nusxa olish" ikonkasi → Android 13+ tizim buferi oynasi havola bilan chiqdi
(`https://app.…/r/b2h-dra2-xzs`, "Edit / Share" tugmalari bilan). Ilova o'z snackbar'ini
ko'rsatmaydi va **ko'rsatmasligi to'g'ri** — tizimniki bilan ikkilanib qolardi.

### Qurilmada topilgan va shu sessiyada tuzatilgan nosozliklar

| # | Nosozlik | Tuzatish |
|---|---|---|
| 7 | §6 №1 tasdiqlandi: jonli darsda **"Davom etish"** ikki qatorga bo'linib tugmani baland qilardi (telefon 360dp; planshetda ko'rinmasdi) | `LessonsScreen.LessonCard` — `BoxWithConstraints`: tor kartada "Ulashish" yozuvi olib tashlanadi (ikonka qoladi), matn `maxLines = 1`. Qurilmada bir qatorga sig'di |
| 8 | Karta meta qatorida **"1 soat"** ikki qatorga bo'linardi | Sana `weight(1f, fill = false)` + ellipsis, davomiylik `maxLines = 1` |
| 9 | Ulanmagan holatda sarlavha ostida inglizcha **"disconnected"** (SDK enum nomi) ko'rinardi | `RoomStatus.connLabel()` — o'zbekcha yozuv; noma'lum qiymat ham "ulanmagan" beradi. 1 test qo'shildi (jami **347**) |

### Eskirgan deb yopilgan §6 bandlari

- §6 №2 (`RoomScreen.Row2` da uzun UUID qatorlari) — M20 qayta qurilishida bu qatorlar
  butunlay olib tashlangan, ekranda UUID yo'q.
- §6 №3 (xato kartasi tugmalarni suradi) — yangi maketda xato kartasi TEPADA, boshqaruv
  paneli PASTGA qotirilgan; FGS bloklangan sinovda tugmalar joyidan qimirlamadi.

---

## ✅ 5f. Ekran ulashishni tiklash (§5b · 3-gipoteza) — 2026-07-28, 10:00–10:20

Qurilma: HONOR ABR-LX1 · Android 15 · Wi-Fi ↔ LTE (ikkala yo'nalish ham).

### Platforma cheklovi — "avtomatik" so'zining chegarasi

**Android 14 (API 34) dan boshlab har bir yozib olish sessiyasi uchun YANGI rozilik
majburiy**: eski `createScreenCaptureIntent()` natijasini qayta ishlatib bo'lmaydi.
Ya'ni "ustoz hech narsa bosmasdan ulashish tiklanadi" 14+ da **prinsipial mumkin emas**.
Shuning uchun tiklash ikki xil qilib yozildi (`data/livekit/ScreenSharePlan.kt`, 5 test):

| Holat | Qaror |
|---|---|
| Ustoz ulashmagan / o'zi to'xtatgan | `NONE` — tiklanmaydi (aks holda to'xtatilgan ulashish qaytib kelardi) |
| Uzilish trekka tegmagan | `NONE` — ketayotgan oqim uzilmaydi |
| Android ≤ 13, rozilik saqlangan | `REUSE_TOKEN` — jimgina qayta boshlanadi |
| Android 14+ | `ASK_CONSENT` — bir bosishlik taklif |

### Uch kanal — chunki ustoz odatda ilovada emas

1. **Ekranda karta**: "Ekran ulashish uzildi · Davom ettirish / Keyinroq". "Davom ettirish"
   tizim oynasini darhol ochadi ("Butun ekran" tushuntirishi TAKRORLANMAYDI — bu tiklash).
2. **Bildirishnoma + titrash**: `Ekran ulashish uzildi — davom ettirish uchun bosing`.
   Ustoz PDF/GeoGebra ichida bo'lsa kartani ko'rmaydi; usiz u ulashish o'lganini faqat
   o'quvchilar aytganda bilardi.
3. Ulashish qaytgach bildirishnoma matni odatdagi holatiga qaytadi.

### 🔴 Yo'l-yo'lakay topilgan JIDDIY nosozlik: interfeys yolg'on gapirardi

Birinchi o'lchovda qayta ulanishdan keyin ekranda **"Ekraningiz ulashilmoqda"** va
**EFIRDA** yozuvlari turaverdi — aslida ekran treki yo'q edi. Sabab: `_screenShareOn`
BAYROQ edi (faqat ustoz to'xtatganda o'chardi), qayta ulanishda esa trek SERVERDA
yo'qoladi. Ya'ni ustoz "ulashyapman" deb ishonib dars o'tardi, o'quvchilar esa hech
narsa ko'rmasdi — bu C-11 ning o'zidan ham yomonroq holat.

→ `LessonSession.reconcileScreenShare()`: holat endi bayroqdan emas, **e'lon qilingan
trekdan** (`getTrackPublication(SCREEN_SHARE)`) olinadi. Tiklash qarori ham shundan
boshlanadi.

### Qurilmadagi o'lchov (ikkala yo'nalish)

| Qadam | Natija |
|---|---|
| Wi-Fi'da ulashish boshlandi | "Ekraningiz ulashilmoqda" · EFIRDA |
| Wi-Fi o'chirildi (LTE bor) | "Mobil internetga o'tildi" → 2-urinishda qayta ulandi (§5d bilan bir xil) |
| Qayta ulangach | Sahna **rost** holatga qaytdi (ulashish yo'q) va **tiklash kartasi** chiqdi |
| "Davom ettirish" → tizim oynasi → Allow | Ulashish tiklandi (EFIRDA qaytdi) |
| Ilova FONDA, Wi-Fi qayta yoqildi | Bildirishnoma matni `Ekran ulashish uzildi — davom ettirish uchun bosing` ga o'zgardi (`dumpsys notification` bilan tasdiqlandi) |
| Ilovaga qaytib "Davom ettirish" | Ulashish yana tiklandi |

Jami testlar: **352** (yangi: `ScreenSharePlanTest` 5 ta).

### ⚠️ Bir marta kuzatilgan NATIV crash (tuzatilmagan)

Bir o'lchovda (uch takrordan birida) tarmoq almashuvi paytida ilova nativ darajada
yiqildi — bizning Kotlin kodimizda emas, LiveKit ichidagi WebRTC'da:

```
pid: 31279, tid: 18542, name: network_thread  >>> uz.darsly.mentor <<<
signal 6 (SIGABRT) · libjingle_peerconnection_so.so
Abort message: 'libc++ Hardening assertion this->has_value() failed:
                optional operator-> called on a disengaged value'
```

Keyingi ikki takrorda qaytarilmadi. Belgilab qo'yildi: agar takrorlansa —
`livekit-android` versiyasini yangilash yoki `disconnect()` o'rniga SDK'ning o'z
qayta ulanishini kutish varianti sinaladi.

✅ **TELEMETRIYA QO'SHILDI (audit · 2026-07-28).** Bu yiqilish endi ko'rinmas
emas: `sentry-android` NDK handler'i bilan ulandi (`DarslyApp.initCrashReporting`).
Nativ SIGABRT'ni faqat NDK handler tutadi — Java darajasidagi handler u yerda
hech nima ko'rmaydi. DSN reliz jarayonida beriladi
(`-PdarslySentryDsn=...`); berilmasa SDK butunlay o'chiq qoladi.

Shundan keyin "SDK'ni yangilash" qarori taxmin emas, **haqiqiy takrorlanish
chastotasi** asosida qabul qilinadi.

---

## 5c. Kesh fayli haqida eslatma (maxfiylik)

✅ **TUZATILDI (audit · 2026-07-28).** Kesh endi `EncryptedSharedPreferences`
bilan shifrlanadi (`LessonsCache.create` → `SecureTokenStore.openEncryptedPrefs`).

Avvalgi holat: `darsly_lessons_cache.xml` shifrlanmagan edi — dars sarlavhalari
va **`join_slug`** ochiq matnda yotardi. Slug esa darsga kirish kaliti, ya'ni
root qilingan qurilmada yoki zaxira nusxada u sizib chiqishi mumkin edi.

Shifrlash ochilmasa (OEM Keystore nosozligi) kesh UMUMAN yozilmaydi
(`NoopLessonsCache`) — shifrlanmagan holatga qaytilmaydi.

---

## 6. Mayda kamchiliklar

| # | Muammo | Holat |
|---|---|---|
| ~~1~~ ✅ | ~~**"Davom etish"** tugmasi ikki qatorga bo'linib ketadi~~ | Tuzatildi (§5e №7), qurilmada tasdiqlandi |
| ~~2~~ ✅ | ~~**B-7:** `Xona`/`Identity` qatorlarida uzun UUID satrlari maketni buzadi~~ | Eskirdi — M20 qayta qurilishida qatorlar olib tashlangan |
| ~~3~~ ✅ | ~~Xona ekranida xato kartasi chiqqanda tugmalar suriladi~~ | Eskirdi — xato kartasi tepada, panel pastga qotirilgan (§5e da tekshirildi) |
| 4 | O'lchovda `lvl=127..0` — RFC 6464 audio darajasi to'g'ri o'qilmadi (bayt/sek ishladi) | `scratchpad/listener` (asbob, mahsulot emas) — qoladi |

---

## 7. TOZALASH RO'YXATI (sinov tugagach)

- [ ] **Telefonda ekran ulashish to'xtatilganini tekshirish** (bildirishnoma paneli) — bugun
      qurilma uzilganda ilova hali ishlab turgan bo'lishi mumkin
- [ ] `adb shell rm -f /sdcard/Music/darsly_tone.wav` (10 MB) va `darsly_qa_tone.wav` (52 MB, R0 dan)
- [ ] Media ovozi qaytarilsin — sinovda **15 pog'ona pasaytirilgan**
- [ ] Vaqtinchalik hisob: `spike.mentor@darsly.uz` — `DELETE /api/v1/users/:id` (admin bilan)
- [x] §5e hisoblari va darsi (`devicetest@`, `deadsession@`, `Nusxa olish sinovi`) — o'chirildi
- [ ] ⚠️ §5f: `Ulashishni tiklash sinovi` darsi **yetim qoldi** — egasi (`sharetest@darsly.uz`)
      undan OLDIN o'chirilgani uchun API endi uni o'chirishga ruxsat bermaydi
      (`you do not own this lesson`). Faqat DB'dan tozalanadi.
      **Saboq: avval darsni, keyin foydalanuvchini o'chiring.**
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
