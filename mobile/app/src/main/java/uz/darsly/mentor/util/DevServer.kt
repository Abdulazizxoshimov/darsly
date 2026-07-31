package uz.darsly.mentor.util

import android.content.Context
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import uz.darsly.mentor.BuildConfig

/**
 * DEV: server manzilini ILOVA ICHIDAN almashtirish (faqat debug build).
 *
 * Muammo: dev'da API manzili `BuildConfig` ga qotirilgan — tarmoq/IP har
 * o'zgarganda (hotspot ↔ uy Wi-Fi ↔ ofis) APK qayta build qilinardi. Bu
 * 2026-07-30 kuni bir kunda UCH marta ro'y berdi.
 *
 * Yechim: login ekranidagi (debug'dagina ko'rinadigan) «Server» tugmasi manzilni
 * SharedPreferences'ga yozadi; OkHttp interceptor'i HAR so'rovda host/port/sxemani
 * shu qiymatga almashtiradi — restart ham kerak emas. LiveKit signaling manzili
 * backend'dan `LIVEKIT_CLIENT_WS_URL=auto` bilan keladi, ya'ni u ham avtomatik
 * yangi hostga ergashadi.
 *
 * Release build'da BUTUNLAY o'chiq: [override] doim `null` qaytaradi va UI
 * tugmasi chizilmaydi — foydalanuvchini boshqa serverga burab bo'lmaydi.
 */
object DevServer {

    private const val PREFS = "dev_server"
    private const val KEY = "api_base_url"

    /** Joriy amaldagi bazaviy manzil (ko'rsatish uchun). */
    fun currentBase(ctx: Context): String =
        override(ctx)?.toString()?.trimEnd('/') ?: BuildConfig.API_BASE_URL.trimEnd('/')

    /** O'rnatilgan override (debug'da va yaroqli bo'lsa), aks holda `null`. */
    fun override(ctx: Context): HttpUrl? {
        if (!BuildConfig.DEBUG) return null
        val raw = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY, null) ?: return null
        return raw.toHttpUrlOrNull()
    }

    /**
     * Manzilni saqlaydi. Bo'sh/`null` — override o'chiriladi (BuildConfig'ga qaytadi).
     * Yaroqsiz URL saqlanmaydi va `false` qaytadi.
     */
    fun set(ctx: Context, url: String?): Boolean {
        val prefs = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val trimmed = url?.trim().orEmpty()
        if (trimmed.isEmpty()) {
            prefs.edit().remove(KEY).apply()
            return true
        }
        val parsed = trimmed.toHttpUrlOrNull() ?: return false
        prefs.edit().putString(KEY, parsed.toString()).apply()
        return true
    }
}
