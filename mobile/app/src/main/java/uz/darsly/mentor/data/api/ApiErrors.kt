package uz.darsly.mentor.data.api

import com.squareup.moshi.Moshi
import retrofit2.HttpException
import java.io.IOException

/**
 * Backend xato kodlarini o'zbekcha matnga xaritalaydi.
 *
 * MANBA (taxmin emas — kod'dan olingan):
 *  · `backend/internal/pkg/errors/errors.go` → NOT_FOUND, CONFLICT, FORBIDDEN,
 *    UNAUTHORIZED, VALIDATION_ERROR, BAD_REQUEST, INTERNAL_ERROR
 *  · `backend/api/middleware/auth.go`        → UNAUTHORIZED, TOKEN_EXPIRED, TOKEN_INVALID
 *  · `backend/api/middleware/rbac.go`        → AUTHZ_UNAVAILABLE, FORBIDDEN
 *  · `backend/api/middleware/rate_limit.go`, `ratelimit_redis.go` → RATE_LIMITED
 *
 * Matnlar **web frontend bilan bir xil** bo'lishi shart:
 * `frontend/src/api/api.jsx` → `ERROR_UZ` (o'sha faylga TEGILMAGAN, faqat o'qilgan).
 * Ustoz web'da va ilovada bir xil xatoni bir xil matn bilan ko'rishi kerak.
 */
object ApiErrors {

    /** Tarmoq umuman yo'q / serverga yetib bo'lmadi (frontend: `TypeError` sharti). */
    const val NETWORK = "Serverga ulanib bo‘lmadi"

    private const val FALLBACK = "Xatolik yuz berdi"

    private val UZ: Map<String, String> = mapOf(
        "UNAUTHORIZED" to "Email yoki parol noto‘g‘ri",
        "FORBIDDEN" to "Bu amalga ruxsatingiz yo‘q",
        "NOT_FOUND" to "Topilmadi",
        "BAD_REQUEST" to "So‘rovda xatolik",
        "CONFLICT" to "Bu ma’lumot allaqachon mavjud",
        "INTERNAL_ERROR" to "Serverda xatolik yuz berdi",
        "VALIDATION_ERROR" to "Kiritilgan ma’lumotlar noto‘g‘ri",
        "RATE_LIMITED" to "Juda ko‘p urinish — biroz kuting",
        "AUTHZ_UNAVAILABLE" to "Xizmat vaqtincha ishlamayapti — birozdan so‘ng urinib ko‘ring",
        "TOKEN_EXPIRED" to "Sessiya tugadi — qaytadan kiring",
        "TOKEN_INVALID" to "Sessiya yaroqsiz — qaytadan kiring",
    )

    private val moshi: Moshi = Moshi.Builder().build()

    /** Kod bo'yicha matn; noma'lum/bo'sh kod uchun `null`. */
    fun uz(code: String?): String? = code?.let { UZ[it] }

    /**
     * Ixtiyoriy istisnoni foydalanuvchiga ko'rsatiladigan matnga aylantiradi.
     *
     * Tartib: **maydon darajasidagi validatsiya tafsiloti** → konvertdagi `code` →
     * server `message` → HTTP status bo'yicha zaxira. Tarmoq istisnosi ([IOException])
     * alohida matn oladi.
     */
    fun humanError(t: Throwable, fallback: String = FALLBACK): String {
        if (t is HttpException) {
            val parsed = runCatching {
                val body = t.response()?.errorBody()?.string()
                if (body.isNullOrBlank()) null else moshi.adapter(ApiError::class.java).fromJson(body)
            }.getOrNull()

            // 🟡D: validatsiya xatolarida server AYNIQ maydonni aytadi
            // ("title: must be at least 2 characters"), umumiy "Kiritilgan ma'lumotlar
            // noto'g'ri" esa buni yo'qotardi. Tanish maydon+qoidani o'zbekchaga
            // o'giramiz; tanimasak — bu ikki kod uchun `uz()` da o'zbekcha matn bor,
            // ya'ni inglizcha xom matn baribir ekranga chiqmaydi.
            if (parsed?.code == "VALIDATION_ERROR" || parsed?.code == "BAD_REQUEST") {
                fieldError(parsed.message)?.let { return it }
            }

            uz(parsed?.code)?.let { return it }
            parsed?.message?.takeIf { it.isNotBlank() }?.let { return it }

            return when (t.code()) {
                400 -> uz("BAD_REQUEST")!!
                401 -> uz("UNAUTHORIZED")!!
                403 -> uz("FORBIDDEN")!!
                404 -> uz("NOT_FOUND")!!
                409 -> uz("CONFLICT")!!
                422 -> uz("VALIDATION_ERROR")!!
                429 -> uz("RATE_LIMITED")!!
                503 -> uz("AUTHZ_UNAVAILABLE")!!
                in 500..599 -> uz("INTERNAL_ERROR")!!
                else -> fallback
            }
        }
        if (t is IOException) return NETWORK
        return t.message?.takeIf { it.isNotBlank() } ?: fallback
    }

    // ─── Maydon darajasidagi validatsiya (🟡D) ────────────────────────────────
    //
    // MANBA: `backend/internal/pkg/validator/validator.go`
    //   · `ValidationError.Error()` → "field: message" lar "; " bilan qo'shiladi
    //   · `fieldMessage(e)` → aniq inglizcha shablonlar (quyida hammasi qoplangan)
    // Backend shu fayllarni o'zgartirsa bu xarita ham yangilanishi kerak; tanimagan
    // shablon xavfsiz tarzda umumiy matnga tushadi (yolg'on tarjima qilinmaydi).

    private val FIELD_UZ: Map<String, String> = mapOf(
        "title" to "Dars nomi",
        "description" to "Tavsif",
        "duration_min" to "Davomiylik",
        "passcode" to "Parol",
        "scheduled_at" to "Boshlanish vaqti",
        "recurrence_rule" to "Takrorlanish qoidasi",
        "email" to "Email",
        "password" to "Parol",
        "full_name" to "Ism",
        "status" to "Holat",
    )

    /** Raqamli maydonlar: ular uchun "belgi" emas, sof son haqida gapiriladi. */
    private val NUMERIC_FIELDS = setOf("duration_min")

    /**
     * "title: must be at least 2 characters" → "Dars nomi: kamida 2 belgi bo'lishi kerak".
     * Bironta bo'lak tanilmasa `null` — chala tarjima ko'rsatishdan ko'ra umumiy matn yaxshiroq.
     */
    internal fun fieldError(message: String?): String? {
        val raw = message?.trim()?.takeIf { it.isNotEmpty() } ?: return null
        val parts = raw.split("; ").map { it.trim() }.filter { it.isNotEmpty() }
        if (parts.isEmpty()) return null

        val translated = parts.map { part ->
            val idx = part.indexOf(": ")
            if (idx <= 0) return null
            val field = part.substring(0, idx)
            val rule = part.substring(idx + 2)
            val label = FIELD_UZ[field] ?: return null
            val ruleUz = ruleMessage(rule, numeric = field in NUMERIC_FIELDS) ?: return null
            "$label: $ruleUz"
        }
        return translated.joinToString("; ")
    }

    private fun ruleMessage(rule: String, numeric: Boolean): String? {
        // "kamida 2 belgi" · "kamida 5" (raqamli maydonda "belgi" so'zi noto'g'ri bo'lardi)
        val unit = if (numeric) "" else " belgi"
        // "2000 belgidan oshmasligi" — qo'shimcha so'zga QO'SHIB yoziladi, ajratib emas.
        val unitFrom = if (numeric) " dan" else " belgidan"
        return when {
            rule == "field is required" -> "to'ldirilishi shart"
            rule == "invalid email format" -> "email formati noto'g'ri"
            rule == "invalid UUID format" -> "noto'g'ri identifikator"
            rule == "invalid URL format" -> "havola formati noto'g'ri"
            rule.startsWith("must be at least ") && rule.endsWith(" characters") ->
                "kamida ${rule.numberBetween("must be at least ", " characters") ?: return null}$unit bo'lishi kerak"
            rule.startsWith("must be at most ") && rule.endsWith(" characters") ->
                "${rule.numberBetween("must be at most ", " characters") ?: return null}$unitFrom oshmasligi kerak"
            rule.startsWith("must be exactly ") && rule.endsWith(" characters") ->
                "aniq ${rule.numberBetween("must be exactly ", " characters") ?: return null}$unit bo'lishi kerak"
            rule.startsWith("must be greater than or equal to ") ->
                "kamida ${rule.removePrefix("must be greater than or equal to ")} bo'lishi kerak"
            rule.startsWith("must be less than or equal to ") ->
                "ko'pi bilan ${rule.removePrefix("must be less than or equal to ")} bo'lishi kerak"
            rule.startsWith("must be greater than ") ->
                "${rule.removePrefix("must be greater than ")} dan katta bo'lishi kerak"
            rule.startsWith("must be less than ") ->
                "${rule.removePrefix("must be less than ")} dan kichik bo'lishi kerak"
            rule.startsWith("must be one of: ") ->
                "quyidagilardan biri bo'lishi kerak: ${rule.removePrefix("must be one of: ")}"
            else -> null
        }
    }

    /** Shablon ichidagi sonni ajratadi; son bo'lmasa `null` (tarjima qilinmaydi). */
    private fun String.numberBetween(prefix: String, suffix: String): String? =
        removePrefix(prefix).removeSuffix(suffix).takeIf { it.isNotEmpty() && it.all(Char::isDigit) }
}
