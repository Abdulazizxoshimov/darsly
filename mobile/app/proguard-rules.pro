# R8 / ProGuard qoidalari — RELIZ uchun (M13).
#
# Avval `isMinifyEnabled = false` edi, ya'ni reliz APK'si:
#   · obfuskatsiyasiz (butun kod, endpoint nomlari, mantiq ochiq o'qiladi),
#   · qisqartirilmagan (APK keraksiz kattaligicha — sekin internetda muhim).
#
# R8 yoqilganda esa asosiy xavf REFLECTIYA: Moshi modellarni, LiveKit/WebRTC
# esa JNI klasslarini nom bo'yicha topadi. Ular qisqartirilsa ilova KOMPILYATSIYA
# BO'LADI, lekin ishga tushganda yiqiladi — shuning uchun quyidagilar aniq
# saqlanadi. Har blok nima uchun kerakligi bilan izohlangan.

# ── LiveKit / WebRTC ─────────────────────────────────────────────────────────
# JNI orqali C++ tomondan nom bilan chaqiriladi — obfuskatsiya JNI bog'lanishini
# uzadi (`UnsatisfiedLinkError` yoki jimgina media ishlamasligi).
-keep class livekit.org.webrtc.** { *; }
-keep class io.livekit.android.** { *; }
-keep class livekit.** { *; }
-dontwarn livekit.org.webrtc.**
-dontwarn io.livekit.android.**

# ── Moshi (JSON) ─────────────────────────────────────────────────────────────
# Codegen adapterlari `<model>JsonAdapter` nomi bo'yicha topiladi.
-keep class **JsonAdapter { *; }
-keepnames @com.squareup.moshi.JsonClass class *
-keepclassmembers @com.squareup.moshi.JsonClass class * { <init>(...); }
-keepclassmembers class * {
    @com.squareup.moshi.FromJson <methods>;
    @com.squareup.moshi.ToJson <methods>;
}
-dontwarn com.squareup.moshi.**

# API model va servis interfeyslari — maydon nomlari JSON kalitlari bilan mos
# bo'lishi SHART (obfuskatsiya ularni `a`, `b` ga aylantirardi).
-keep class uz.darsly.mentor.data.api.** { *; }
-keep interface uz.darsly.mentor.data.api.** { *; }

# ── Retrofit / OkHttp ────────────────────────────────────────────────────────
# Retrofit servis metodlarining generik imzolari reflectiya bilan o'qiladi.
-keepattributes Signature, InnerClasses, EnclosingMethod
-keepattributes RuntimeVisibleAnnotations, RuntimeVisibleParameterAnnotations
-keep,allowobfuscation interface retrofit2.http.*
-dontwarn retrofit2.**
-dontwarn okhttp3.**
-dontwarn okio.**

# ── Kotlin ───────────────────────────────────────────────────────────────────
-keepattributes *Annotation*
-dontwarn kotlinx.coroutines.**

# ── Crash hisobotlari uchun ──────────────────────────────────────────────────
# Satr raqamlarisiz stack-trace deyarli foydasiz bo'ladi: R8 obfuskatsiya
# qilganda ham manba satrlari saqlanadi va mapping fayli bilan tiklanadi.
# (`app/build/outputs/mapping/release/mapping.txt` — RELIZ bilan birga saqlansin!)
-keepattributes SourceFile, LineNumberTable
-renamesourcefileattribute SourceFile

# ── Tink / androidx.security.crypto ──────────────────────────────────────────
# Token Keystore'da AES-GCM bilan saqlanadi (`androidx.security.crypto` →
# Google Tink).
#
# Faqat `-dontwarn`, `-keep` EMAS. Butun `com.google.crypto.tink.**` ni saqlash
# R8'ni `KeysDownloader` ni ham ushlab qolishga majbur qiladi, u esa ilovada
# UMUMAN yo'q ixtiyoriy kutubxonalarga (Google HTTP client, Joda-Time) havola
# qiladi va build yana yiqiladi. Bizga kerak bo'lgan qism (AEAD/Keyset)
# `androidx.security.crypto` orqali to'g'ridan-to'g'ri chaqiriladi, ya'ni R8
# uni o'zi saqlaydi — reflectiya yo'q.
-dontwarn com.google.errorprone.annotations.**
-dontwarn javax.annotation.**
-dontwarn com.google.api.client.**
-dontwarn org.joda.time.**
