package uz.darsly.mentor.data.store

import android.content.SharedPreferences
import uz.darsly.mentor.data.api.TokenPair

/**
 * Token saqlagich (M1).
 *
 * Ilova jarayoni o'lganda ham sessiya saqlanishi kerak — aks holda ustoz har safar
 * qayta login qiladi. Interfeys ataylab **Android'siz** (`SharedPreferences` faqat
 * implementatsiyada) — shu tufayli single-flight refresh mantiqi va saqlash mantiqi
 * JVM unit testlarida emulyatorsiz sinaladi.
 */
interface TokenStore {
    /** Saqlangan juftlik yoki `null` (tizimga kirilmagan). */
    fun read(): TokenPair?

    /** Juftlikni saqlaydi (login yoki refresh natijasi). */
    fun save(pair: TokenPair)

    /** To'liq tozalash (logout yoki refresh muvaffaqiyatsizligi). */
    fun clear()
}

/** Faqat xotirada. Test uchun va shifrlangan saqlagich yasalmasa zaxira sifatida. */
class InMemoryTokenStore(initial: TokenPair? = null) : TokenStore {
    @Volatile private var pair: TokenPair? = initial

    override fun read(): TokenPair? = pair
    override fun save(pair: TokenPair) { this.pair = pair }
    override fun clear() { pair = null }
}

/**
 * `SharedPreferences` ustidagi saqlagich.
 *
 * Ishlab chiqarishda [SecureTokenStore] bu klassni **EncryptedSharedPreferences** bilan
 * yasaydi; testlarda oddiy fake `SharedPreferences` beriladi — ya'ni test ishlab
 * chiqarish kodining AYNI o'zini sinaydi.
 *
 * Xotira keshi bor: har HTTP so'rovda `Authorization` header uchun token o'qiladi,
 * shifrlangan preferens'ni har safar deshifrlash keraksiz yuk bo'lardi.
 */
class PrefsTokenStore(private val prefs: SharedPreferences) : TokenStore {

    // `null` = hali o'qilmagan; `Holder(null)` = o'qildi, token yo'q.
    @Volatile private var cache: Holder? = null

    private class Holder(val pair: TokenPair?)

    override fun read(): TokenPair? {
        cache?.let { return it.pair }
        return synchronized(this) {
            cache?.pair ?: run {
                val access = prefs.getString(KEY_ACCESS, null)
                val refresh = prefs.getString(KEY_REFRESH, null)
                val pair = if (access.isNullOrBlank() || refresh.isNullOrBlank()) {
                    null
                } else {
                    TokenPair(access, refresh)
                }
                cache = Holder(pair)
                pair
            }
        }
    }

    /**
     * `apply()` (asinxron) ATAYLAB: yozish tez-tez (har refresh rotatsiyasida) va
     * ba'zan main thread'dagi korutinadan chaqiriladi. Yozuv diskka tushmasdan
     * jarayon o'lsa ham xavf kichik: backend'da 60 soniyalik grace oynasi bor,
     * ya'ni eski refresh token o'sha oyna ichida ayni juftlikni qaytaradi.
     */
    override fun save(pair: TokenPair) {
        synchronized(this) {
            prefs.edit()
                .putString(KEY_ACCESS, pair.accessToken)
                .putString(KEY_REFRESH, pair.refreshToken)
                .apply()
            cache = Holder(pair)
        }
    }

    /**
     * Bu yerda esa `commit()` (sinxron) — tozalash **kafolatlangan** bo'lishi kerak.
     * Logoutdan (yoki refresh o'lganidan) so'ng darhol jarayon o'ldirilsa,
     * `apply()` da shifrlangan tokenlar diskda qolib ketishi mumkin edi.
     * Logout kamdan-kam va foydalanuvchi baribir kutadi — narxi ahamiyatsiz.
     */
    @Suppress("ApplySharedPref")
    override fun clear() {
        synchronized(this) {
            prefs.edit().remove(KEY_ACCESS).remove(KEY_REFRESH).commit()
            cache = Holder(null)
        }
    }

    private companion object {
        const val KEY_ACCESS = "access_token"
        const val KEY_REFRESH = "refresh_token"
    }
}
