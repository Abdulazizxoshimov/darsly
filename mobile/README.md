# Darsly Mentor — Android ilova (R1 · 2-blok)

Ustoz (mentor) uchun native Android ilova.

- **R0 spike** ✅ — `MediaProjection` orqali ekran ulashish LiveKit ustida ishlashi tasdiqlangan.
- **R1 · 1-blok** ✅ — asos va sessiya: M1 (shifrlangan token saqlash), M2 (single-flight
  jim token yangilash), M3 (logout), M42 (majburiy yangilanish), yagona xato modeli, B-2 tuzatishi.
- **R1 · 3-blok** 🔄 — jonli xona: B-4/B-5 (ekran audiosi rost holati + resurs bo'shatish),
  B-6 (dars tugaganda MediaProjection bo'shatiladi), B-1 ("Butun ekran" tushuntirishi),
  M12 (kamera almashtirish), M16 (qayta ulanish + aloqa sifati). Qoldi: **C-6** (ekran
  audiosini alohida trekka chiqarish) — qurilmada spike talab qiladi.
- **R1 · 2-blok** ✅ — darslar: M6 (ro'yxat + bo'limlar + pull-to-refresh + offline kesh),
  M7 (dars yaratish + o'zbekcha validatsiya), M8 (join havolasini native "Ulashish"),
  `LessonsRepository` qatlami.

<!-- Keyingi bloklar: M11–M16 (jonli xona), M22–M27 (boshqaruv) — roadmap §5. -->

- Paket: `uz.darsly.mentor` · Ilova nomi: **Darsly Mentor** · **`1.0.0` (versionCode 2)**
- `minSdk 26` (Android 8.0) · `targetSdk 35` · `compileSdk 35`
- Kotlin 2.0.21 · Jetpack Compose (Material 3) · AGP 8.7.3 · Gradle 8.11.1
- LiveKit: `io.livekit:livekit-android:2.27.0`
- Tokenlar: `androidx.security:security-crypto:1.1.0` (EncryptedSharedPreferences) — §9
- Backend API: `BuildConfig.API_BASE_URL` · o'quvchiga yuboriladigan havola bazasi:
  **`BuildConfig.WEB_BASE_URL`** (hozir bir xil host, lekin ataylab alohida — API
  `api.*` subdomeniga ko'chsa join havolalari buzilmasin)

> ⚠️ **`versionName` — bu shunchaki yorliq emas.** U `GET /api/v1/app-config` dagi
> `min_version` bilan solishtiriladi (§9.3). Versiyani tushirib yubormang: serverdagi
> `min_version` dan past qiymat ilovani o'z-o'zini bloklashga majbur qiladi.

---

## 1. Muhitni tayyorlash

### Android SDK

Loyiha SDK'ni `local.properties` dagi `sdk.dir` orqali topadi (bu fayl **git'ga tushmaydi**).

```bash
# 1) commandline-tools
mkdir -p ~/Android/Sdk/cmdline-tools
curl -Lo /tmp/cmdline-tools.zip \
  https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip
unzip -q /tmp/cmdline-tools.zip -d /tmp/cmdline-tools-x
mv /tmp/cmdline-tools-x/cmdline-tools ~/Android/Sdk/cmdline-tools/latest

# 2) paketlar + litsenziyalar
export ANDROID_HOME=~/Android/Sdk
yes | $ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager --licenses
$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager \
  "platform-tools" "platforms;android-35" "build-tools;35.0.0"

# 3) loyihaga bog'lash
echo "sdk.dir=$HOME/Android/Sdk" > mobile/local.properties
```

### Java

**JDK o'rnatish SHART EMAS.** `settings.gradle.kts` da `foojay-resolver-convention`
plagini bor — Gradle kerakli **JDK 17 toolchain'ini o'zi yuklab oladi** (`~/.gradle/jdks`).
Bu mashinada faqat **JRE 21** bor edi (`javac` yo'q), toolchain shuni hal qildi.
Foydalanuvchining global muhitiga (`.bashrc`, `JAVA_HOME`) **tegilmagan**.

### Test hisobi (ixtiyoriy)

Login formasini oldindan to'ldirish uchun `mobile/local.properties` ga qo'shing
(**bu fayl gitignore'da — sirlar kodga tushmaydi**):

```properties
darsly.testEmail=<sizning-test-emailingiz>
darsly.testPassword=<parol>
```

> Haqiqiy hisob ma'lumotlarini bu faylga YOZMANG — `README.md` git'ga tushadi.
> Test hisobi ma'lumotlari jamoa ichida alohida uzatiladi.

---

## 2. Qurish

```bash
cd mobile
./gradlew assembleDebug
```

APK: `mobile/app/build/outputs/apk/debug/app-debug.apk` (~66 MB — debug, siqilmagan,
4 ta ABI uchun WebRTC native kutubxonalari bilan).

```bash
./gradlew lint          # statik tahlil (hisobot: app/build/reports/lint-results-debug.html)
./gradlew clean         # tozalash
```

### 2.1 Testlar

```bash
./gradlew testDebugUnitTest      # 139 ta JVM unit test (tarmoqsiz, ~25 s)
```

Hisobot: `app/build/reports/tests/testDebugUnitTest/index.html`

| Test sinfi | Nimani qoplaydi |
|---|---|
| `TokenAuthenticatorTest` | ⭐ **single-flight**: N ta parallel 401 → serverga 1 ta refresh (MockWebServer bilan sanaladi), cheksiz sikl yo'qligi, 4xx → toza logout, 5xx va tarmoq uzilishida sessiya saqlanishi |
| `SessionStateTest` | Sessiya holati; kech ulangan obunachi ham logoutni ko'radi |
| `TokenStoreTest` | Saqlash / o'qish / tozalash; ilova qayta ochilganda sessiya saqlanishi |
| `SemverTest` | `1.10.0 > 1.9.0`, pre-release, bo'sh/buzuq qiymat |
| `UpdateDeciderTest` | Majburiy / yumshoq yangilanish, `force_update`, fail-open |
| `ApiErrorsTest` | 11 ta backend kodi → o'zbekcha matn (web `ERROR_UZ` bilan aynan bir xil) |
| `ApiFieldErrorsTest` | Maydon darajasidagi validatsiya: `"title: must be at least 2 characters"` → "Dars nomi: kamida 2 belgi bo'lishi kerak"; tanimagan shablon **tarjima qilinmaydi** |
| `JoinGuardTest` | Ulanish xatosidan keyin qayta urinish bloklanmasligi |
| `RoomErrorsTest` | Kutilmagan istisnolar (fon rejimida FGS taqiqi) → tushunarli matn; **hech qachon bo'sh emas** (bo'sh xabar = abadiy spinner) |
| `RoomStatusTest` | M16 aloqa indikatori (yaxshi bo'lsa ko'rinmaydi, qayta ulanishda har doim) va B-6 dars tugash sabablari → o'zbekcha xabar. SDK enum'lari **nomi bo'yicha** o'giriladi |
| `UiPrefsTest` | B-1 "Boshqa eslatilmasin" tanlovi ilova qayta ochilganda saqlanishi |
| `ScreenAudioPolicyTest` | Ekran audiosi qachon oqadi va **oqmasa nega**: Android <10, ruxsat yo'q, ulashish yo'q, mikrofon o'chiq. UI matni shu yerdan |
| `RoomUiStateTest` | Tizim "Orqaga" tugmasi qachon tasdiq so'raydi: ulangan/ulanayotgan/qayta ulanayotgan yoki ekran ulashilayotgan bo'lsa — ha; xonaga kirmagan bo'lsa — yo'q |
| `LessonsRepositoryTest` | ⭐ Offline: tarmoq yiqilsa kesh **tegilmaydi**; sahifalash (`total` qoplanguncha), 500 ta chegara oshsa `truncated`; yaratish so'rovining tanasi |
| `LessonsCacheTest` | Kesh diskda saqlanishi; **buzilgan JSON** ilovani yiqitmasligi |
| `LessonFormatTest` | Bo'limlar va tartib (jonli → eng yaqin → tugagan), o'zbekcha sana, join havolasi web bilan aynan bir xil (`/r/<slug>`) |
| `LessonFormTest` | Yaratish formasi chegaralari (backend `validate` teglari bilan bir xil); sana+vaqtni **mintaqa bilan** birlashtirish |

**Jonli integratsiya testi** (odatda `Assume` bilan o'tkazib yuboriladi — CI tarmoqqa
bog'liq bo'lmasligi uchun). Haqiqiy serverga qarshi login → eskirgan access →
single-flight refresh → `app-config` → logout → sessiya o'lgach toza logout:

```bash
DARSLY_LIVE=1 \
DARSLY_LIVE_EMAIL=<email> DARSLY_LIVE_PASSWORD=<parol> \
  ./gradlew testDebugUnitTest --tests '*LiveAuthFlowTest'
```

Ixtiyoriy: `DARSLY_BASE_URL=https://...` bilan boshqa serverga yo'naltirish.
Sirlar **faqat muhit o'zgaruvchilaridan** o'qiladi, kodda va git'da saqlanmaydi.

---

## 3. Qurilmaga o'rnatish

### 3.1 Telefonda USB debugging'ni yoqish

1. **Sozlamalar → Telefon haqida** → **Build number** ustiga **7 marta** bosing
   ("Siz endi dasturchisiz" chiqadi).
2. **Sozlamalar → Tizim → Dasturchi parametrlari** → **USB debugging** ni yoqing.
3. Telefonni USB bilan ulang → telefonda "Allow USB debugging?" → **Allow**
   ("Always allow from this computer" ni belgilang).

Xiaomi/Redmi (MIUI) uchun qo'shimcha: **USB orqali o'rnatish** (Install via USB) ni ham yoqing.

### 3.2 O'rnatish

```bash
adb devices -l                 # qurilma ko'rinishi kerak
adb install -r mobile/app/build/outputs/apk/debug/app-debug.apk
```

`INSTALL_FAILED_UPDATE_INCOMPATIBLE` chiqsa: `adb uninstall uz.darsly.mentor` va qayta o'rnating.

### 3.3 ⭐ Wi-Fi orqali adb (`adb tcpip 5555`) — QURILMA SINOVIDAN OLDIN YOQING

**Muammo:** ba'zi qurilmalarda (aniqlangan: **HONOR / MagicOS**, shuningdek MIUI va
Samsung'ning ayrim buildlarida) ekran qulflanganda USB rejimi avtomatik "faqat
quvvatlash" (charge only) holatiga qaytadi. Natijada **adb ulanishi uziladi** — aynan
uzoq davom etadigan sinovlarda (90 daqiqalik dars, fon rejimi, ekran qulflash testi)
`logcat` o'rtada to'xtaydi va sinovni boshidan takrorlashga to'g'ri keladi.

**Yechim — USB hali ulangan va ishlayotgan paytda TCP rejimini yoqib qo'ying:**

```bash
adb devices -l                    # qurilma ko'rinayotganiga ishonch hosil qiling
adb tcpip 5555                    # qurilma adb'ni 5555-portda tinglashga o'tadi
adb shell ip route | awk '{print $9}'   # qurilma IP manzili (yoki: Sozlamalar → Wi-Fi)
adb connect <QURILMA-IP>:5555
adb devices -l                    # endi ikkita yozuv: USB + <IP>:5555
# Endi USB kabelni uzsa ham bo'ladi:
adb -s <QURILMA-IP>:5555 logcat --pid=$(adb -s <QURILMA-IP>:5555 shell pidof -s uz.darsly.mentor)
```

Eslatmalar:
- Kompyuter va telefon **bitta Wi-Fi tarmoqda** bo'lishi shart.
- Qurilma qayta yuklansa rejim USB'ga qaytadi — `adb tcpip 5555` ni takrorlang.
- Android 11+ da muqobil: **Dasturchi parametrlari → Wireless debugging** (juftlash kodi bilan),
  bu USB'ni umuman talab qilmaydi.
- Ekran qulflanganda sinash uchun `adb shell input keyevent 26` (power) qulay —
  ulanish TCP orqali bo'lgani uchun uzilmaydi.

---

## 4. ⭐ Ekran ulashishni sinash (R0 asosiy testi)

### Tayyorgarlik
- Web'da (`https://app.194.163.139.242.sslip.io`) mentor hisobi bilan kiring va **dars yarating**.
- **Ikkinchi qurilma/brauzerda** o'sha darsning join havolasini oching (o'quvchi roli) —
  ekran ulashilganini **shu yerda** ko'rasiz.

### Qadamlar
1. Ilovani oching → **Login** (mentor email/parol).
2. **Darslar** ro'yxatidan darsni bosing.
3. Ruxsat so'rovlari chiqadi: **Kamera**, **Mikrofon**, (Android 13+) **Bildirishnoma** → hammasiga **Allow**.
4. Xona ekranida `Holat: connected` bo'lishini kuting. `Mikrofon: yoniq`, `Kamera: yoniq`.
5. **"EKRANNI ULASHISH"** tugmasini bosing.
6. Tizim dialogi: *"Darsly Mentor ekranni yozib olishni boshlaysizmi?"* → **Boshlash / Start now**.
7. Kutilgan natija:
   - Ilovada `Ekran: ULASHILMOQDA`, diagnostika jurnalida `EKRAN ULASHISH BOSHLANDI (720p/15fps)`
   - Bildirishnoma panelida "Dars davom etmoqda" doimiy bildirishnomasi
   - **Brauzerdagi o'quvchi ustoz ekranini ko'radi**

### Fon rejimi testi (M15)
Ekran ulashilayotgan holda **Home** tugmasini bosing, PDF yoki GeoGebra oching.
**Kutilgan:** brauzerdagi o'quvchi yangi ilovani ko'rishda davom etadi, ulashish uzilmaydi.

### Ekran audiosi testi (M14)
Ekran ulashilayotgan holda telefonda video (masalan mahalliy video pleyer) o'ynating.
**Kutilgan:** o'quvchi videoning ovozini ham eshitadi.
⚠️ Cheklovlar §6 da — Android 10+ talab qilinadi va manba ilova capture'ga ruxsat bergan bo'lishi kerak.

---

## 5. `adb logcat` — nimani kuzatish

```bash
# Faqat shu ilova (eng qulay usul)
adb logcat --pid=$(adb shell pidof -s uz.darsly.mentor)

# Yoki teglar bo'yicha
adb logcat -s LiveKit:V DarslyMentor:V ActivityManager:I AndroidRuntime:E
```

### ✅ Muvaffaqiyat belgilari
| Log | Ma'nosi |
|---|---|
| `Screen capture service is connected` | LiveKit'ning `ScreenCaptureService` bog'landi |
| `Starting foreground service` / `startForeground` (ActivityManager) | FGS ko'tarildi |
| LiveKit `VERBOSE` da `publishing track ... SCREEN_SHARE` | Ekran track'i e'lon qilindi |
| Ilova ichidagi diagnostika: `EKRAN ULASHISH BOSHLANDI` | Oqim oxirigacha o'tdi |

### ❌ Xato belgilari va sababi
| Log | Sabab / yechim |
|---|---|
| `SecurityException: Media projections require a foreground service of type ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION` | **R-1 riski ro'yobga chiqdi.** FGS `MediaProjection`dan keyin ko'tarilgan. `LessonService.start(ctx, withProjection=true)` `setScreenShareEnabled`dan OLDIN chaqirilishini tekshiring |
| `ForegroundServiceStartNotAllowedException` | FGS fon rejimidan ishga tushirilgan (Android 12+). Faqat ekran ochiq holda boshlang |
| `Failed to bind ScreenCaptureService` | Manifest merger buzilgan — `io.livekit...ScreenCaptureService` merged manifest'da bormi tekshiring |
| WS `403` / `Origin` | **BE-1 bloker** — backend WS `Origin` tekshiruvi native klientni rad etyapti |
| Media ulanmaydi, ICE `failed` | **BE-3** — deploy'da TURN o'chirilgan. 4G/CGNAT ortida media o'tmaydi |
| `AudioRecord.startRecording failed` | Ekran audiosi: `RECORD_AUDIO` yo'q yoki Android < 10 |

Merged manifest'ni tekshirish:
```bash
grep -A4 'ScreenCaptureService' \
  app/build/intermediates/merged_manifests/debug/processDebugManifest/AndroidManifest.xml
```
`android:foregroundServiceType="mediaProjection|microphone"` bo'lishi kerak.

---

## 5b. Emulyatorda tasdiqlangan natija (2026-07-25)

R0 spike **Android 14 (API 34) emulyatorida uchdan-uchga sinovdan o'tdi** — bu FGS
qoidalari eng qattiq versiya (risk R-1).

```bash
export ANDROID_HOME=~/Android/Sdk
$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager "emulator" "system-images;android-34;google_apis;x86_64"
$ANDROID_HOME/cmdline-tools/latest/bin/avdmanager create avd -n darsly34 \
  -k "system-images;android-34;google_apis;x86_64" -d pixel_6
$ANDROID_HOME/emulator/emulator -avd darsly34 -no-window -gpu swiftshader_indirect &

adb install -r app/build/outputs/apk/debug/app-debug.apk
for p in CAMERA RECORD_AUDIO POST_NOTIFICATIONS; do adb shell pm grant uz.darsly.mentor android.permission.$p; done
# Tizim MediaProjection dialogini avtomatik tasdiqlash (FAQAT test uchun):
adb shell appops set uz.darsly.mentor PROJECT_MEDIA allow
adb shell am start -n uz.darsly.mentor/.MainActivity
```

**Olingan dalillar:**

| Tekshiruv | Natija |
|---|---|
| Login → `POST /auth/login` | ✅ token olindi |
| Darslar ro'yxati | ✅ yuklandi |
| `POST /lessons/:id/token` → LiveKit `wss://` | ✅ `Holat: connected` |
| Mikrofon + kamera publish | ✅ `track e'lon qilindi: MICROPHONE / CAMERA` |
| **Ekran ulashish (API 34)** | ✅ `track e'lon qilindi: SCREEN_SHARE`, **SecurityException YO'Q** |
| WebRTC virtual displey | ✅ `DisplayDeviceInfo{"WebRTC_ScreenCapture" ... 720 x 1280 ...}` |
| **Ekran audiosi** | ✅ `ekran audiosi YOQILDI` (`AudioRecord` xatosiz ishga tushdi) |
| `LessonService` FGS tipi | ✅ `types=0xE0` = MEDIA_PROJECTION\|CAMERA\|MICROPHONE |
| LiveKit `ScreenCaptureService` FGS tipi | ✅ `types=0xA0` = MEDIA_PROJECTION\|MICROPHONE (manifest override ishladi) |
| **Fon rejimi** (HOME bosildi) | ✅ ikkala FGS ham `isForeground=true`, virtual displey saqlandi, uzilish yo'q |
| Server tomoni (`GET /lessons/:id/participants`) | ✅ ishtirokchi `active: true` |

⚠️ **Emulyator nimani ISBOTLAMAYDI:** haqiqiy video sifati/o'qilishi, real tarmoq
(4G/TURN), MIUI/EMUI kabi ishlab chiqaruvchi cheklovlari, batareya/qizish,
`appops` o'rniga **haqiqiy foydalanuvchi dialogi**. Bular real qurilma talab qiladi (§4).

---

## 6. Ma'lum cheklovlar (halol ro'yxat)

| # | Cheklov | Izoh |
|---|---|---|
| 1 | **Ekran audiosi Android 10 (API 29)+ talab qiladi** | SDK `ScreenAudioCapturer` `@RequiresApi(Q)`. `minSdk 26` bo'lgani uchun Android 8/9 da bu funksiya **prinsipial ishlamaydi** |
| 2 | **Ekran audiosi mikrofon track'iga mikslanadi** | Alohida track emas — `LocalAudioTrack.setAudioBufferCallback`. Ya'ni **mikrofon yoniq bo'lishi shart**. Mikrofon o'chirilsa audio to'xtaydi va UI shuni **aytadi** (`ScreenAudioPolicy`), resurs bo'shatiladi. **Qaror:** keyingi blokda alohida trekka o'tkaziladi (Zoom xulqi) |
| 3 | Manba ilova capture'ga ruxsat bermasa — jimlik | `AudioAttributes.ALLOW_CAPTURE_BY_ALL` qo'ymagan ilovalar (YouTube va boshqalar) tizim tomonidan jim qaytariladi. Bizning xatomiz emas |
| 4 | ~~Token faqat xotirada~~ | ✅ **1-blokda yopildi** — `EncryptedSharedPreferences`, §9 |
| 5 | ~~Token yangilash yo'q~~ | ✅ **1-blokda yopildi** — single-flight refresh, §9.2 |
| 6 | Video sahna ko'rsatilmaydi | Hozircha faqat holat matni. `TODO(M20)` `VideoTrackView`/gallery |
| 6b | Darslarni **tahrirlash/o'chirish** yo'q | 2-blok qamrovi: ro'yxat + yaratish + ulashish. `TODO(M9)` |
| 6c | Ro'yxatda **500 tadan ko'p** dars bo'lsa qolgani ko'rsatilmaydi | 10 × 50 sahifa chegarasi. Jim qirqilmaydi — ustozga xabar chiqadi (`LessonsRepository.MAX_PAGES`) |
| 7 | Kutish xonasi (WS) yo'q | `TODO(M27)` — BE-1 backend tomonda ✅, mobil tomon keyingi blokda |
| 8 | Release build imzolanmagan, R8 o'chiq | `TODO` imzo kaliti + proguard (tarqatishdan oldin) |
| 9 | Room hali servis ichida emas | Process-singleton (`LessonSessionHolder`) + FGS. Activity o'lsa ham process tirik, lekin `TODO` egalikni servisga to'liq ko'chirish |
| 10 | `EncryptedSharedPreferences` shifrlash qatlami JVM testida sinalmaydi | Testlar `PrefsTokenStore` mantiqini qoplaydi; Keystore/Tink qismi **haqiqiy qurilma** talab qiladi (§10 QA ro'yxati) |
| 11 | `RoomViewModel` uchun unit test yo'q | LiveKit `Room`/`Application`siz JVM'da yaratib bo'lmaydi. Ajratilgan `JoinGuard` sinaladi, qolgani — qurilma sinovi |
| 12 | ~~Repository qatlami yarim~~ | ✅ **2-blokda yopildi** — `LessonsRepository` (tarmoq + offline kesh). Qolgani: `RoomViewModel` va `UpdateGate` hamon `Net.api` ni to'g'ridan-to'g'ri chaqiradi |
| 13 | 2-blok UI'si **qurilmada sinalmagan** | Bo'limlar ko'rinishi, pull-to-refresh imosi, aviarejimda offline chizig'i, Telegram "Ulashish" oynasi — adb'ga qurilma ulanmagani uchun faqat kod darajasida tekshirilgan |

---

## 7. Tuzilma

```
mobile/
├── settings.gradle.kts          # repolar, foojay toolchain resolver, JitPack (faqat davidliu)
├── build.gradle.kts             # ildiz
├── gradle/libs.versions.toml    # versiya katalogi
├── local.properties             # sdk.dir + test hisobi (GITIGNORE)
└── app/
    ├── build.gradle.kts         # minSdk 26 / targetSdk 35, BuildConfig.API_BASE_URL, test deps
    └── src/
        ├── main/
        │   ├── AndroidManifest.xml       # FGS ruxsatlari + ScreenCaptureService tipini kengaytirish
        │   ├── res/xml/data_extraction_rules.xml  # sessiya fayli backup/D2D dan chiqarilgan
        │   └── java/uz/darsly/mentor/
        │       ├── DarslyApp.kt          # Application: Session.init, LiveKit log darajasi
        │       ├── MainActivity.kt       # NavHost (login → lessons → room) + UpdateGate
        │       ├── data/api/
        │       │   ├── Models.kt         # Envelope, TokenPair, AppConfig, Lesson, ...
        │       │   ├── DarslyApi.kt      # REST kontrakt + AuthRefreshApi (alohida)
        │       │   ├── Net.kt            # OkHttp/Retrofit: 2 ta klient (oddiy + bare)
        │       │   ├── TokenAuthenticator.kt  # ⭐ M2 single-flight refresh
        │       │   ├── Session.kt        # sessiya holati (StateFlow) + TokenStore fasadi
        │       │   ├── AuthRepository.kt # login / logout (M1 · M3)
        │       │   └── ApiErrors.kt      # xato kodi → o'zbekcha matn
        │       ├── data/store/
        │       │   ├── TokenStore.kt     # interfeys + PrefsTokenStore + InMemoryTokenStore
        │       │   ├── SecureTokenStore.kt   # EncryptedSharedPreferences + recovery
        │       │   ├── LessonsCache.kt   # darslar offline keshi (Moshi + prefs)
        │       │   └── UiPrefs.kt        # interfeys sozlamalari (B-1 tushuntirish bayrog'i)
        │       ├── data/repo/
        │       │   └── LessonsRepository.kt   # tarmoq + kesh + sahifalash (M6 · M7)
        │       ├── data/livekit/         # LessonSession (⭐ ekran ulashish), LessonSessionHolder
        │       ├── service/              # LessonService (FGS), LessonNotifications
        │       ├── util/
        │       │   ├── Semver.kt         # semver solishtirish (M42)
        │       │   ├── LessonFormat.kt   # bo'lim/tartib/sana matni + join havolasi
        │       │   └── Share.kt          # native "Ulashish" + klipbord (M8)
        │       └── ui/
        │           ├── login/ theme/
        │           ├── lessons/          # LessonsScreen/ViewModel + CreateLesson* + LessonForm
        │           ├── room/JoinGuard.kt # ulanish qorovuli (qayta urinish mumkinligi)
        │           ├── room/RoomErrors.kt    # kutilmagan istisno → o'zbekcha matn
        │           ├── room/RoomStatus.kt    # aloqa sifati + dars tugash sababi (M16 · B-6)
        │           └── update/           # UpdateDecider + UpdateGate (M42)
        └── test/java/uz/darsly/mentor/   # 139 ta JVM unit test (§2.1)
```

---

## 8. Keyingi qadam

Ochiq ishlar: `docs/BACKLOG.md`. Backend blokerlari (BE-1…BE-5) **yopilgan**.
Keyingi blok — **3-blok, jonli xona** (eng kattasi): B-1 "Entire screen" ogohlantirishi,
B-4 ekran audiosi holatining rostgo'yligi, C-6 (ekran audiosi alohida trekmi — **qaror kerak**),
B-6 MediaProjection'ni bo'shatish, M12 kamera almashtirish, M16 qayta ulanish, M18 adaptiv sifat.

---

## 9. Sessiya: token saqlash, yangilash, versiya nazorati

### 9.1 Token saqlash (M1)

Tokenlar `EncryptedSharedPreferences` (`androidx.security:security-crypto:1.1.0`) ichida,
kalit **Android Keystore**da (`AES256_GCM` master key). Fayl: `darsly_session.xml`.

**Nega aynan 1.1.0:** avvalgi `1.1.0-alpha06` — 2023-yil aprelidagi pre-release edi.
`1.1.0` esa **barqaror** reliz (2025-07-30, Tink 1.8.0). Tanlov `maven-metadata.xml`
va rasmiy reliz yozuvlari bo'yicha tekshirildi, so'ng `assembleDebug` + unit testlar
bilan tasdiqlandi.

⚠️ **Halol ogohlantirish:** bu kutubxonaning **barcha API'lari 1.1.0-beta01 dan boshlab
deprecated** — Google to'g'ridan-to'g'ri Keystore ishlatishni tavsiya qiladi. Barqaror
reliz bo'lgani uchun hozircha ishlatilyapti; almashtirish keyingi bloklarga rejalashtirilgan.

**Android 15 / OEM xatarlari qanday qoplangan:** `security-crypto` da native kod yo'q
(Tink — sof Java), shuning uchun 16 KB sahifa muammosi tegishli emas. Haqiqiy xatar —
ba'zi OEM'larda Keystore keyset'i buzilib `create()` istisno tashlashi va ilova
**ishga tushmayoq crash** bo'lishi. `SecureTokenStore` buni qoplaydi:
xato → keyset+fayl o'chiriladi → **bir marta** qayta urinish → u ham bo'lmasa xotiradagi
saqlagich (ilova ishlaydi, faqat sessiya saqlanmaydi). Ustozni dars oldida crash bilan
qoldirgandan ko'ra qayta login qildirish yaxshiroq.

Qo'shimcha: `android:allowBackup="false"` + `data_extraction_rules.xml` — sessiya fayli
bulutli zaxira va qurilmadan-qurilmaga ko'chirishdan chiqarilgan.

### 9.2 Jim token yangilash — single-flight (M2)

Access TTL 15 daqiqa. 401 kelganda OkHttp **`Authenticator`** ishga tushadi
(`Interceptor` emas — sabab `TokenAuthenticator.kt` sinf izohida batafsil).

Kafolatlar:
- Bir vaqtda N ta 401 bo'lsa ham serverga **BITTA** `POST /auth/refresh` ketadi
  (Kotlin `Mutex` + qulf ichida "boshqa oqim yangiladimi?" tekshiruvi).
- Qayta urinish **faqat bir marta** (`priorResponse` qorovuli).
- Refresh 4xx → **toza logout**: tokenlar o'chadi, `Session.loggedIn` `false` bo'ladi,
  UI login ekraniga qaytadi. Keyingi 401'da token yo'qligi uchun refresh umuman
  yuborilmaydi — **cheksiz sikl prinsipial ravishda mumkin emas**.
- Refresh 5xx → sessiya saqlanadi (server tiklanishi mumkin).
- Refresh paytida tarmoq uzilsa → `IOException` uzatiladi, UI "Serverga ulanib bo'lmadi"
  deydi va foydalanuvchi **tizimdan chiqarilmaydi**.

Backend tomonda 60 soniyalik grace oynasi bor (`docs/api-contract.md`), lekin klient
poygasi baribir yo'q qilingan — grace zaxira, kafolat emas.

### 9.3 Majburiy yangilanish (M42)

Ishga tushganda `GET /api/v1/app-config` so'raladi:

| Shart | Natija |
|---|---|
| `versionName` < `min_version` | **Bloklovchi** dialog (orqaga tugmasi ham yopmaydi), `apk_url` ga havola |
| `force_update: true` | Bloklovchi dialog (versiyadan qat'i nazar) |
| `versionName` < `latest_version` | Yopiladigan eslatma ("Keyinroq" tugmasi bilan) |
| Endpoint javob bermadi / qiymat buzuq | **Hech narsa** — ilova ishlashda davom etadi |

Oxirgi qator ataylab: **fail-open**. Aks holda serverning bir daqiqalik nosozligi
barcha ustozlarni darsdan mahrum qilardi.

Solishtirish `util/Semver.kt` orqali — **satr solishtirish emas**
(`"1.10.0" < "1.9.0"` bo'lib chiqardi va butun guruhni noto'g'ri bloklardi).

### 9.4 Xato modeli

Backend barcha xatolarni `{"code","message"}` konvertida qaytaradi (401/403/429 ham).
`ApiErrors` 11 ta kodni o'zbekcha matnga xaritalaydi; matnlar `frontend/src/api/api.jsx`
dagi `ERROR_UZ` bilan **aynan bir xil** — ustoz web'da va ilovada bir xil xabarni ko'radi.
Tarmoq xatosi alohida matn oladi: "Serverga ulanib bo'lmadi".

---

## 10. QA nimani qurilmada tekshirishi kerak

JVM testlari qoplay olmaydigan, **haqiqiy qurilma** talab qiladigan holatlar:

1. **Sessiya saqlanishi:** login → ilovani to'liq yopish (recents'dan swipe) → qayta ochish
   → login ekrani **ko'rinmasligi** kerak. Qurilmani qayta yuklab ham takrorlang.
2. **Keystore:** `adb shell pm clear uz.darsly.mentor` dan keyin ilova crash qilmasdan
   ochilishi va login ishlashi.
3. **Logout:** "Chiqish" → login ekrani; qayta ochilganda ham kirilmagan holat.
   Aviarejimda "Chiqish" bosilganda ham lokal chiqish bajarilishi.
4. **Jim yangilash:** 15+ daqiqa ilovani ochiq qoldiring (yoki access TTL ni kutib),
   so'ng darslar ro'yxatini yangilang — **qayta login so'ralmasligi** kerak.
5. **Sessiya o'lishi:** web'da o'sha hisobdan logout qiling → ilovada biror amal bajaring
   → **darhol login ekraniga** qaytishi (darslar ekranida xato bilan qolib ketmasligi).
6. **B-2 / ruxsatlar:** darsga kirishda mikrofonni **rad eting** → tushunarli xabar +
   "Ruxsat so'rash"/"Sozlamalar" tugmalari; ruxsat bergach dars boshlanishi.
   `logcat` da `SecurityException ... FOREGROUND_SERVICE_MICROPHONE` **bo'lmasligi** kerak.
7. **Kamerasiz dars:** faqat kamerani rad eting → dars ovoz + ekran ulashish bilan ishlashi.
8. **Fon rejimi:** ekran ulashilayotganda HOME → PDF ochish → ulashish davom etishi
   (§4). Android 10 (API 29) qurilmada ham servis o'lmasligi.
9. **Ulanish xatosi:** aviarejimda darsga kiring → o'zbekcha xato + **"Qayta urinish"**
   tugmasi; internetni yoqib bosganda dars **haqiqatan** boshlanishi.
10. **Majburiy yangilanish:** serverda `min_version` ni `2.0.0` ga qo'ying → ilova
    bloklovchi dialog ko'rsatishi va orqaga tugmasi bilan yopilmasligi; `force_update`
    va yumshoq eslatma variantlarini ham tekshiring. Serverni o'chirib qo'yganda esa
    ilova **odatdagidek ishlashi** (fail-open).
11. **Ekran ulashish regressiyasi:** §4 dagi to'liq oqim (bu blok LiveKit kodiga tegmagan,
    lekin tasdiqlash shart).
