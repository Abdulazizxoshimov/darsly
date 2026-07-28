// DIQQAT: `java.util.Properties` deb yozib bo'lmaydi — Kotlin DSL'da `java`
// Gradle'ning JavaPluginExtension'iga bog'lanadi. Shuning uchun import.
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.ksp)
}

// Test login ma'lumotlari sirlar hisoblanadi — kodga yozilmaydi.
// mobile/local.properties (gitignore'da) ichida ixtiyoriy ravishda beriladi:
//   darsly.testEmail=...
//   darsly.testPassword=...
val localProps = Properties().apply {
    val f = rootProject.file("local.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
fun localProp(key: String, default: String = "") = localProps.getProperty(key) ?: default

// ── Reliz imzosi ─────────────────────────────────────────────────────────────
// Kalit va parollar `mobile/keystore.properties` da (gitignore'da), kalitning
// o'zi `.jks` (u ham gitignore'da). Fayl bo'lmasa imzo konfiguratsiyasi UMUMAN
// yaratilmaydi va `assembleRelease` imzosiz APK beradi — ya'ni boshqa mashinada
// yoki CI'da qurish buzilmaydi, faqat tarqatib bo'lmaydi.
//
// NEGA DEBUG APK TARQATILMAYDI: debug variantida `TEST_EMAIL`/`TEST_PASSWORD`
// (haqiqiy sinov hisobi) `BuildConfig` ga yoziladi. Ochiq `/download/` manzilida
// turgan debug APK'dan bu parollarni ajratib olish jiddiy mehnat talab qilmaydi.
// Relizda ular bo'sh satr.
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
        versionCode = 4
        // DIQQAT (M42): bu qiymat `GET /api/v1/app-config` dagi `min_version` bilan
        // solishtiriladi. Serverda hozir min_version=1.0.0 — spike'dagi "0.1.0-spike"
        // qolsa ilova o'zini bloklab qo'yardi. R1 bloki = 1.0.0.
        // 1.1.0 — profil · bildirishnomalar · yozuvlar · jadval · dars tahrirlash ·
        //         parol tiklash · kutish xonasi.
        // 1.2.0 — yozib olish default yoniq va avtomatik boshlanadi; ekran
        //         ulashishda simulcast (past internet uchun past qatlam).
        versionName = "1.2.0"
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
            buildConfigField("String", "API_BASE_URL", "\"https://app.194.163.139.242.sslip.io\"")
            // WEB_BASE_URL — o'quvchiga yuboriladigan `/r/<slug>` havolasining bazasi.
            // Hozir API bilan bir xil host, lekin ATAYLAB alohida: API alohida
            // subdomenga (`api.*`) ko'chirilsa, join havolalari jimgina buzilmasin.
            buildConfigField("String", "WEB_BASE_URL", "\"https://app.194.163.139.242.sslip.io\"")
            buildConfigField("String", "TEST_EMAIL", "\"${localProp("darsly.testEmail")}\"")
            buildConfigField("String", "TEST_PASSWORD", "\"${localProp("darsly.testPassword")}\"")
        }
        release {
            // Imzo kaliti bo'lsa — imzolanadi; bo'lmasa APK imzosiz chiqadi va
            // qurish baribir muvaffaqiyatli tugaydi (boshqa mashina / CI uchun).
            releaseSigning?.let { signingConfig = signingConfigs.getByName("release") }
            isMinifyEnabled = false // TODO(R1): R8 + proguard qoidalari (LiveKit/Moshi uchun)
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            buildConfigField("String", "API_BASE_URL", "\"https://app.194.163.139.242.sslip.io\"")
            buildConfigField("String", "WEB_BASE_URL", "\"https://app.194.163.139.242.sslip.io\"")
            buildConfigField("String", "TEST_EMAIL", "\"\"")
            buildConfigField("String", "TEST_PASSWORD", "\"\"")
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

    // M1 — token EncryptedSharedPreferences'da (androidx.security:security-crypto 1.1.0).
    implementation(libs.androidx.security.crypto)
    implementation(libs.androidx.datastore.preferences)

    // ── Testlar ───────────────────────────────────────────────────────────────
    testImplementation(libs.junit)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.kotlinx.coroutines.test)
}
