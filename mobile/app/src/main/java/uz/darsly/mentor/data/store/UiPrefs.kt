package uz.darsly.mentor.data.store

import android.content.Context
import android.content.SharedPreferences

/**
 * Interfeys uchun kichik, maxfiy bo'lmagan sozlamalar (B-1).
 *
 * Bu yerda **sir yo'q** — shuning uchun oddiy `SharedPreferences` (tokenlar
 * [SecureTokenStore] da qoladi). Hozircha bitta bayroq: ekran ulashish
 * tushuntirishini ko'rsatishni davom etish kerakmi.
 */
interface UiPrefs {
    /** "Butun ekran" tushuntirishi ko'rsatilsinmi (ustoz "eslatmang" desa `false`). */
    var screenShareTipEnabled: Boolean
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

    companion object {
        private const val PREFS_NAME = "darsly_ui"
        private const val KEY_TIP = "screen_share_tip_enabled"

        fun create(context: Context): UiPrefs = PrefsUiPrefs(
            context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE),
        )
    }
}
