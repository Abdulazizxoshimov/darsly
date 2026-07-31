// Ildiz build fayli — plaginlar faqat e'lon qilinadi, qo'llanmaydi.

// ⚠️ JavaPoet 1.13 MAJBURIY (Hilt Gradle plagini uchun).
//
// Hilt 2.52 ning `AggregateDepsTask` i JavaPoet 1.13 API'siga tayanadi
// (`ClassName.canonicalName()`), lekin plagin klasspathiga boshqa transitiv
// bog'liqlik orqali 1.10 tushib qolishi mumkin — o'shanda build
// `NoSuchMethodError: ClassName.canonicalName()` bilan yiqiladi.
//
// Bu xato INKREMENTAL buildda ko'rinmaydi (task o'tkazib yuboriladi) va faqat
// `clean` dan keyin — ya'ni odatda CI'da yoki reliz yig'ishda — chiqadi.
// 2026-07-31 da aynan shunday yuz berdi.
buildscript {
    dependencies {
        classpath("com.squareup:javapoet:1.13.0")
    }
}

plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.android) apply false
    alias(libs.plugins.kotlin.compose) apply false
    alias(libs.plugins.ksp) apply false
}
