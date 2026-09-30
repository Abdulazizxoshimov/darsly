// DIQQAT: `java.util.Properties` deb yozib bo'lmaydi — Kotlin DSL'da `java`
// Gradle'ning JavaPluginExtension'iga bog'lanadi. Shuning uchun import.
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.ksp)
    alias(libs.plugins.hilt)
}

// Test login ma'lumotlari sirlar hisoblanadi — kodga yozilmaydi.
// mobile/local.properties (gitignore'da) ichida ixtiyoriy ravishda beriladi:
//   darsly.testEmail=...
//   darsly.testPassword=...
//   darsly.devPrefill=true   ← S3: login formasini oldindan to'ldirish uchun
//                              ALOHIDA opt-in. `debug` build'ning o'zi yetarli
//                              EMAS: aks holda har debug APK'da haqiqiy sinov
//                              hisobi kompilyatsiya qilinardi.
val localProps = Properties().apply {
    val f = rootProject.file("local.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
fun localProp(key: String, default: String = "") = localProps.getProperty(key) ?: default
val devPrefill = localProp("darsly.devPrefill") == "true"
fun devCred(key: String) = if (devPrefill) localProp(key) else ""

// ── Muhit manzillari (M13) ───────────────────────────────────────────────────
// Avval staging IP (`app.194.163.139.242.sslip.io`) reliz variantiga ham
// QOTIRIB yozilgan edi — ya'ni real domenga o'tish uchun kodni tahrirlash
// kerak bo'lardi va tasodifan staging'ga qarab turgan APK tarqalishi mumkin edi.
//
// Endi qiymat quyidagi tartibda olinadi:
//   1) Gradle xossasi  -PdarslyApiUrl=https://...   (CI/reliz uchun)
//   2) local.properties: darsly.apiUrl=https://...  (dasturchi mashinasi)
//   3) default — staging (hozircha yagona ishlaydigan muhit)
//
// Real domen paydo bo'lganda default shu yerda BIR JOYDA o'zgaradi.
// 2026-07-31: eski Contabo (194.163.139.242) to'lov tugab o'chdi, yangi server olindi.
val stagingBase = "https://app.169.58.104.245.sslip.io"
fun envUrl(gradleKey: String, localKey: String): String =
    (project.findProperty(gradleKey) as String?)
        ?: localProps.getProperty(localKey)
        ?: stagingBase

// ── Reliz imzosi ─────────────────────────────────────────────────────────────
// Kalit va parollar `mobile/keystore.properties` da (gitignore'da), kalitning
// o'zi `.jks` (u ham gitignore'da). Fayl bo'lmasa imzo konfiguratsiyasi UMUMAN
// yaratilmaydi va `assembleRelease` imzosiz APK beradi — ya'ni boshqa mashinada
// yoki CI'da qurish buzilmaydi, faqat tarqatib bo'lmaydi.
//
// NEGA DEBUG APK TARQATILMAYDI: debug variantida `darsly.devPrefill=true` bo'lsa
// `TEST_EMAIL`/`TEST_PASSWORD` (haqiqiy sinov hisobi) `BuildConfig` ga yoziladi.
// Ochiq `/download/` manzilida turgan debug APK'dan bu parollarni ajratib olish
// jiddiy mehnat talab qilmaydi. Relizda ular HAR DOIM bo'sh satr.
val keystoreProps = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
val releaseSigning = keystoreProps.getProperty("storeFile")?.takeIf {
    rootProject.file(it).exists()
}

android {
    namespace = "uz.darsly.mentor"
    compileSdk = 35

    defaultConfig {
        applicationId = "uz.darsly.mentor"
        minSdk = 26
        targetSdk = 35
        versionCode = 14
        // DIQQAT (M42): bu qiymat `GET /api/v1/app-config` dagi `min_version` bilan
        // solishtiriladi. Serverda hozir min_version=1.0.0 — spike'dagi "0.1.0-spike"
        // qolsa ilova o'zini bloklab qo'yardi. R1 bloki = 1.0.0.
        // 1.1.0 — profil · bildirishnomalar · yozuvlar · jadval · dars tahrirlash ·
        //         parol tiklash · kutish xonasi.
        // 1.2.0 — yozib olish default yoniq va avtomatik boshlanadi; ekran
        //         ulashishda simulcast (past internet uchun past qatlam).
        // 1.3.0 — Zoom pariteti (docs/PRODUCT.md «Yozuv sifati»): ulashish
        //         manbasi qurilma nisbatidan (qora yo'l yo'q) va ulashish
        //         boshlanishida ilova o'zini fonga oladi.
        // 1.4.0 — «Arxiv» bo'limi: o'tgan darslar video (ilova ichida pleyer) +
        //         Telegram-uslub chat paneli; bildirishnomalar endi o'ng-tepadagi
        //         qo'ng'iroqda (alohida bo'lim emas).
        // 1.7.0 — Kabinet qayta tuzildi (sozlamalar-ro'yxati), kirganda mic/kamera
        //         o'chiq + recording default o'chiq, dars o'chirish 3-nuqta menyusiga,
        //         hisob o'chirish so'rovi (admin tasdiqlaydi), LoginReq bug tuzatildi.
        versionName = "1.7.5"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    signingConfigs {
        releaseSigning?.let { path ->
            create("release") {
                storeFile = rootProject.file(path)
                storePassword = keystoreProps.getProperty("storePassword")
                keyAlias = keystoreProps.getProperty("keyAlias")
                keyPassword = keystoreProps.getProperty("keyPassword")
                // APK Signature Scheme v2/v3 — Android 7+ o'rnatishni tezlashtiradi
                // va v1 (jar) imzosini ham qoldiradi (minSdk 26 uchun v1 shart emas,
                // lekin ba'zi eski o'rnatuvchilar undan foydalanadi).
                enableV1Signing = true
                enableV2Signing = true
                enableV3Signing = true
            }
        }
    }

    buildTypes {
        debug {
            isMinifyEnabled = false
            // Test serveri (CLAUDE.md · deploy/server — sslip.io + Caddy HTTPS)
            buildConfigField("String", "API_BASE_URL", "\"${envUrl("darslyApiUrl", "darsly.apiUrl")}\"")
            // WEB_BASE_URL — o'quvchiga yuboriladigan `/r/<slug>` havolasining bazasi.
            // Hozir API bilan bir xil host, lekin ATAYLAB alohida: API alohida
            // subdomenga (`api.*`) ko'chirilsa, join havolalari jimgina buzilmasin.
            buildConfigField("String", "WEB_BASE_URL", "\"${envUrl("darslyWebUrl", "darsly.webUrl")}\"")
            // S3: faqat `darsly.devPrefill=true` bo'lsa (yuqorida) — debug'ning o'zi emas.
            buildConfigField("String", "TEST_EMAIL", "\"${devCred("darsly.testEmail")}\"")
            buildConfigField("String", "TEST_PASSWORD", "\"${devCred("darsly.testPassword")}\"")
            // Debug'da crash-hisobotlari default O'CHIQ: dasturchi mashinasidagi
            // yiqilishlar production statistikasini ifloslantirmasin.
            buildConfigField("String", "SENTRY_DSN", "\"${localProp("darsly.sentryDsn")}\"")
        }
        release {
            // Imzo kaliti bo'lsa — imzolanadi; bo'lmasa APK imzosiz chiqadi va
            // qurish baribir muvaffaqiyatli tugaydi (boshqa mashina / CI uchun).
            releaseSigning?.let { signingConfig = signingConfigs.getByName("release") }
            // R8 YOQILDI (M13). Qoidalar `proguard-rules.pro` da — LiveKit/WebRTC
            // JNI klasslari va Moshi modellari reflectiya bilan topiladi, ular
            // qisqartirilsa ilova ISHGA TUSHGANDA yiqilardi (kompilyatsiyada emas).
            //
            // `isShrinkResources` ham yoqilgan: ishlatilmagan resurslar APK'dan
            // chiqadi — sekin internetda yuklab olish vaqti muhim.
            //
            // ⚠️ RELIZDA: `build/outputs/mapping/release/mapping.txt` ni SAQLANG.
            // Usiz crash-hisobotlaridagi stack-trace o'qib bo'lmaydigan bo'ladi.
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            buildConfigField("String", "API_BASE_URL", "\"${envUrl("darslyApiUrl", "darsly.apiUrl")}\"")
            buildConfigField("String", "WEB_BASE_URL", "\"${envUrl("darslyWebUrl", "darsly.webUrl")}\"")
            buildConfigField("String", "TEST_EMAIL", "\"\"")
            buildConfigField("String", "TEST_PASSWORD", "\"\"")
            // Relizda DSN reliz jarayonidan keladi:
            //   ./gradlew assembleRelease -PdarslySentryDsn=https://...@sentry.io/...
            // Berilmasa SDK o'chiq qoladi (qarang: `DarslyApp.initCrashReporting`).
            buildConfigField(
                "String",
                "SENTRY_DSN",
                "\"${(project.findProperty("darslySentryDsn") as String?) ?: localProp("darsly.sentryDsn")}\"",
            )
        }
    }

    compileOptions {
        // JVM MAQSADI: 17.
        // Muhitda faqat JDK 21 o'rnatilgan, shuning uchun toolchain 21 (pastda),
        // lekin bytecode 17 ga kompilyatsiya qilinadi — Android/D8 uchun eng
        // sinovdan o'tgan maqsad. AGP 8.7.3 JDK 21 bilan quradi.
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        // JDK 17 — Android uchun eng sinovdan o'tgan toolchain.
        // Mahalliy muhitda topilmasa foojay-resolver (settings.gradle.kts) yuklab oladi.
        jvmToolchain(17)
    }

    // JAVA toolchain — Kotlin'niki bilan bir xil (17).
    //
    // Nega alohida kerak: `jvmToolchain(17)` FAQAT Kotlin taskilariga ta'sir
    // qiladi. Hilt esa annotatsiya protsessori uchun `javac` ishlatadigan
    // alohida task yaratadi (`hiltJavaCompileDebug`) va u tizim JDK'siga
    // tushardi. Bu muhitda tizimda JRE 21 (javac'siz) o'rnatilgan —
    // natijada `does not provide the required capabilities: [JAVA_COMPILER]`.
    java {
        toolchain { languageVersion = JavaLanguageVersion.of(17) }
    }

    // ── APK hajmi: ABI bo'yicha bo'lish ──────────────────────────────────────
    //
    // LiveKit/WebRTC har bir protsessor arxitekturasi uchun alohida native
    // kutubxona olib keladi. Universal APK'da ularning HAMMASI yotadi:
    //   x86_64 15.3 MB · x86 12.0 MB · armeabi-v7a 6.5 MB · arm64-v8a 11.6 MB
    // ya'ni 62 MB APK'ning 45 MB'i shu, va uning 27 MB'i (x86/x86_64) faqat
    // EMULYATORGA kerak — haqiqiy telefon yoki planshetga hech qachon emas.
    //
    // Sinov qurilmalari (Galaxy Tab S9, Redmi Note 11) va amaldagi barcha
    // zamonaviy Android qurilmalari — `arm64-v8a`. Shu sabab alohida arm64 APK
    // yig'iladi: ~28 MB, ya'ni ikki barobardan ko'proq kichik. Sekin internetda
    // (serverdan o'lchangani ~300 KB/s) bu 3.5 daqiqa o'rniga 1.5 daqiqa.
    //
    // `isUniversalApk = true` — universal variant ham saqlanadi: `armeabi-v7a`
    // li eski qurilma uchrasa, u ishlashda davom etadi. Ya'ni kichraytirish
    // hech kimni qamrovdan chiqarmaydi, shunchaki to'g'ri faylni tanlash
    // imkonini beradi.
    splits {
        abi {
            isEnable = true
            reset()
            include("arm64-v8a", "armeabi-v7a")
            isUniversalApk = true
        }
    }

    packaging {
        resources.excludes += setOf(
            "/META-INF/{AL2.0,LGPL2.1}",
            "META-INF/DEPENDENCIES",
            "META-INF/INDEX.LIST",
        )
    }

    lint {
        // Spike: build lint xatosidan to'xtamasin, lekin hisobot yozilsin.
        abortOnError = false
        warningsAsErrors = false
    }

    testOptions {
        unitTests {
            // JVM testlarida android.jar stub'lari xato o'rniga default qiymat qaytarsin.
            // (Bizning testlar Android API'ga tegmaydi, lekin bu xavfsiz zaxira.)
            isReturnDefaultValues = true
        }
    }
}

// Jonli integratsiya testi (LiveAuthFlowTest) muhit o'zgaruvchilari bilan yoqiladi.
// Sirlar KODDA emas — faqat shu yerdan test JVM'iga uzatiladi. Berilmasa test
// `Assume` orqali o'tkazib yuboriladi, ya'ni odatiy qurish tarmoqqa bog'liq emas.
tasks.withType<Test>().configureEach {
    listOf("DARSLY_LIVE", "DARSLY_LIVE_EMAIL", "DARSLY_LIVE_PASSWORD", "DARSLY_BASE_URL")
        .forEach { key ->
            providers.environmentVariable(key).orNull?.let { environment(key, it) }
        }
}

dependencies {
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.navigation.compose)

    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons.extended)
    debugImplementation(libs.androidx.compose.ui.tooling)

    // Tarmoq
    implementation(libs.retrofit)
    implementation(libs.retrofit.converter.moshi)
    implementation(libs.okhttp)
    implementation(libs.okhttp.logging)
    implementation(libs.moshi)
    // Moshi adapterlari KSP bilan generatsiya qilinadi (kotlin-reflect kerak emas).
    ksp(libs.moshi.codegen)

    // Video (SFU)
    implementation(libs.livekit.android)

    // Arxiv yozuvini ilova ichida o'ynatish (ExoPlayer + PlayerView).
    implementation(libs.androidx.media3.exoplayer)
    implementation(libs.androidx.media3.ui)

    // M1 — token EncryptedSharedPreferences'da (androidx.security:security-crypto 1.1.0).
    // Crash-hisobotlari (M15). `sentry-android` NDK handler'ini ham olib keladi —
    // ma'lum yiqilish (`libjingle_peerconnection_so.so` · SIGABRT) nativ qatlamda
    // va uni faqat shu tutadi. DSN bo'sh bo'lsa SDK butunlay o'chiq qoladi.
    // Hilt — bog'liqlik injeksiyasi (izohi `libs.versions.toml` da).
    implementation(libs.hilt.android)
    ksp(libs.hilt.compiler)
    implementation(libs.hilt.navigation.compose)

    implementation(libs.sentry.android)
    implementation(libs.androidx.security.crypto)

    // ── Testlar ───────────────────────────────────────────────────────────────
    testImplementation(libs.junit)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.kotlinx.coroutines.test)
}
