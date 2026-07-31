package uz.darsly.mentor.data.api

import com.squareup.moshi.Moshi

/**
 * NEGA tizimdan chiqarildik — login ekraniga olib o'tiladigan yagona xabar.
 *
 * ## Nega umuman kerak
 * Backend `SESSION_REVOKED` kodini beradi va u aksariyat hollarda **«boshqa
 * qurilmada kirildi»** degani: bitta akkaunt = bitta faol sessiya (PRODUCT.md
 * №1) — muvaffaqiyatli `POST /auth/login` shu foydalanuvchining boshqa BARCHA
 * sessiyalarini tugatadi. Bu holatda ustozga umumiy «Sessiya tugadi» deyish
 * ADASHTIRADI: u internetni yoki parolni ayblaydi va akkaunti ulashilganini
 * (masalan hamkasbi bilan) bilmay qoladi.
 *
 * Matnlar web bilan **bir xil**: `frontend/src/lib/logoutReason.js` (o'sha
 * faylga tegilmagan, faqat o'qilgan). Ustoz web'da va ilovada bir xil sababni
 * bir xil so'z bilan ko'rishi kerak.
 *
 * ## Nega alohida tur, `String?` emas
 * Sabab uch joydan keladi (401 javob tanasi, refresh javobi, qo'lda logout) va
 * har birida kodni matnga o'girish takrorlanardi. Turlangan qiymat esa `when`
 * bilan qopланadi va yangi kod qo'shilganda kompilyator eslatadi.
 */
enum class LogoutReason(val message: String?) {

    /** Sabab noma'lum (yoki oddiy chiqish) — login ekranida hech nima yozilmaydi. */
    UNKNOWN(null),

    /**
     * Sessiya SERVERDA tugatilgan. Backend buni uch holatda beradi: boshqa
     * qurilmadan kirish, `logout` va parol tiklash. Uchalasida ham to'g'ri
     * harakat bir xil (qayta kirish), lekin eng ko'p uchraydigani — birinchisi.
     */
    REVOKED(
        "Boshqa qurilmada kirildi. Bitta hisobdan bir vaqtda faqat bitta " +
            "qurilmada foydalanish mumkin.",
    ),

    /** Token buzuq/muddati o'tgan va yangilab bo'lmadi. */
    EXPIRED("Sessiya muddati tugadi — qaytadan kiring."),
    ;

    companion object {

        const val CODE_SESSION_REVOKED = "SESSION_REVOKED"
        const val CODE_TOKEN_INVALID = "TOKEN_INVALID"
        const val CODE_TOKEN_EXPIRED = "TOKEN_EXPIRED"

        /** Xato konvertidagi `code` → sabab. Notanish kod — [UNKNOWN]. */
        fun ofCode(code: String?): LogoutReason = when (code) {
            CODE_SESSION_REVOKED -> REVOKED
            CODE_TOKEN_INVALID, CODE_TOKEN_EXPIRED -> EXPIRED
            else -> UNKNOWN
        }

        /**
         * Xato javobining TANASIDAN sabab. Buzuq/bo'sh JSON — [UNKNOWN]
         * (parse xatosi hech qachon ilovani yiqitmasligi kerak).
         */
        fun ofBody(json: String?): LogoutReason {
            val raw = json?.trim()?.takeIf { it.isNotEmpty() } ?: return UNKNOWN
            val parsed = runCatching { adapter.fromJson(raw) }.getOrNull() ?: return UNKNOWN
            return ofCode(parsed.code)
        }

        private val adapter by lazy {
            Moshi.Builder().build().adapter(ApiError::class.java)
        }
    }
}
