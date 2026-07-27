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

android {
    namespace = "uz.darsly.mentor"
    compileSdk = 35

    defaultConfig {
        applicationId = "uz.darsly.mentor"
        minSdk = 26
        targetSdk = 35
        versionCode = 2
        // DIQQAT (M42): bu qiymat `GET /api/v1/app-config` dagi `min_version` bilan
        // solishtiriladi. Serverda hozir min_version=1.0.0 — spike'dagi "0.1.0-spike"
        // qolsa ilova o'zini bloklab qo'yardi. R1 bloki = 1.0.0.
        versionName = "1.0.0"
    }

    buildFeatures {
        compose = true
        buildConfig = true
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
