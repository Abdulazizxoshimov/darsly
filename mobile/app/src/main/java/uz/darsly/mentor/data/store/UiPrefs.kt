package uz.darsly.mentor.data.store

import android.content.Context
import android.content.SharedPreferences

/**
 * Interfeys uchun kichik, maxfiy bo'lmagan sozlamalar (B-1).
 *
 * Bu yerda **sir yo'q** — shuning uchun oddiy `SharedPreferences` (tokenlar
 * [SecureTokenStore] da qoladi). Ikkala bayroq ham ekran ulashish xulqiga
 * tegishli: ustoz nimani ko'rishi va ilova o'zini qanday tutishi.
 */
interface UiPrefs {
    /** "Butun ekran" tushuntirishi ko'rsatilsinmi (ustoz "eslatmang" desa `false`). */
    var screenShareTipEnabled: Boolean

    /**
     * Ulashish boshlanganda ilova o'zini fonga olsinmi (Zoom xulqi, №25).
     *
     * Default YONIQ: 2026-08-01 dagi yozuvda aynan bu yo'qligi tufayli
     * o'quvchilar dars materiali o'rniga Jonly interfeysini ko'rgan. Ya'ni
     * to'g'ri xulq — asosiy yo'l, o'chirish esa istisno (masalan ustoz
     * planshetda bo'lib, ulashish uchun ilovani ochiq ushlab turmoqchi).
     */
    var autoBackgroundOnShare: Boolean
}

class PrefsUiPrefs(private val prefs: SharedPreferences) : UiPrefs {

    override var screenShareTipEnabled: Boolean
        get() = prefs.getBoolean(KEY_TIP, true)
        // ATAYLAB `commit()`, `apply()` emas (lint `ApplySharedPref` shuni taklif qiladi).
        //
        // Sabab: bu qiymat FAQAT ustoz "Boshqa eslatilmasin" ni belgilaganda yoziladi —
        // ya'ni sekundda bir marta ham emas, ana shundan keyin darhol MediaProjection
        // dialogi ochiladi va tizim ilovani fonga tashlashi/o'ldirishi mumkin. `apply()`
        // navbatga qo'yadi, `commit()` esa diskka yozib qaytadi. Bloklash narxi bir
        // martalik va ahamiyatsiz; yo'qolgan tanlov esa ustozni har ulashishda
        // bezovta qiladi. (1-blok QA'si `TokenStore.clear()` uchun ham aynan shuni so'ragan.)
        @Suppress("ApplySharedPref")
        set(value) {
            prefs.edit().putBoolean(KEY_TIP, value).commit()
        }

    override var autoBackgroundOnShare: Boolean
        get() = prefs.getBoolean(KEY_AUTO_BG, true)
        // `commit()` — yuqoridagi bilan bir xil sabab, hatto kuchliroq: bu
        // tanlov dars ichida, ulashishdan bir necha soniya oldin qilinadi va
        // shundan keyin ilova O'ZINI FONGA OLADI. `apply()` navbati fon
        // rejimidagi jarayon o'ldirilsa yozilmay qolishi mumkin — ustoz esa
        // sozlamani o'zgartirdim deb o'ylab qoladi.
        @Suppress("ApplySharedPref")
        set(value) {
            prefs.edit().putBoolean(KEY_AUTO_BG, value).commit()
        }

    companion object {
        private const val PREFS_NAME = "darsly_ui"
        private const val KEY_TIP = "screen_share_tip_enabled"
        private const val KEY_AUTO_BG = "auto_background_on_share"

        fun create(context: Context): UiPrefs = PrefsUiPrefs(
            context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE),
        )
    }
}
