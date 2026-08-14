package uz.darsly.mentor.util

import uz.darsly.mentor.data.api.Recording

/**
 * Yozuvlar ro'yxatining **sof** formatlash mantiqi (Android'siz, JVM testida).
 *
 * Bu yerdagi har bir funksiya foydalanuvchi ko'radigan matnni yasaydi, shuning
 * uchun chekka holatlar (nol hajm, tugamagan yozuv, notanish status) test bilan
 * qotirilgan: "0 B" yoki "-1 daqiqa" kabi matn ilova buzuq degan taassurot beradi.
 */
object RecordingFormat {

    /** Statuslar — `backend/internal/entity/recording.go` konstantalari. */
    const val STATUS_RECORDING = "recording"
    const val STATUS_PROCESSING = "processing"
    const val STATUS_READY = "ready"
    const val STATUS_FAILED = "failed"

    /**
     * Saqlash muddati (retention) tugagan: MinIO'dagi fayl O'CHIRILGAN, qator
     * esa tarix uchun qolgan (PRODUCT.md №5). `expires_at` endi qaytmaydi.
     */
    const val STATUS_EXPIRED = "expired"

    fun statusLabel(status: String): String = when (status) {
        STATUS_RECORDING -> "Yozilmoqda"
        STATUS_PROCESSING -> "Tayyorlanmoqda"
        STATUS_READY -> "Tayyor"
        STATUS_FAILED -> "Xatolik"
        STATUS_EXPIRED -> "O'chirilgan"
        else -> status
    }

    /**
     * Bayt → "12,4 MB". Ikkilik prefikslar (1024) — fayl menejerlari shunday sanaydi.
     *
     * Nol yoki noma'lum hajm uchun `null`: "0 B" deb yozish yozuv bo'sh degan
     * yolg'on ma'no beradi, holbuki hajm hali hisoblanmagan bo'lishi mumkin
     * (`processing` holatida server uni bilmaydi).
     */
    fun sizeLabel(bytes: Long): String? {
        if (bytes <= 0) return null
        val units = listOf("B", "KB", "MB", "GB", "TB")
        var value = bytes.toDouble()
        var unit = 0
        while (value >= 1024 && unit < units.lastIndex) {
            value /= 1024
            unit++
        }
        // Baytlarda kasr ma'nosiz; kattaroq birliklarda bitta kasr yetarli.
        return if (unit == 0) "${bytes} ${units[0]}" else "${format1(value)} ${units[unit]}"
    }

    /** "45 soniya" · "12 daqiqa" · "1 soat 30 daqiqa". Nol/manfiy uchun `null`. */
    fun durationLabel(seconds: Int): String? {
        if (seconds <= 0) return null
        if (seconds < 60) return "$seconds soniya"
        return LessonFormat.durationLabel(seconds / 60)
    }

    /**
     * Ro'yxat tartibi: eng yangi yozuv birinchi.
     *
     * `started_at` yo'q bo'lsa `created_at` ga tushamiz — ikkalasi ham bo'lmasa
     * yozuv oxirida qoladi (tartibsiz "sakrash" bo'lmasligi uchun barqaror).
     */
    fun sortForDisplay(recordings: List<Recording>): List<Recording> =
        recordings.sortedByDescending {
            LessonFormat.epochOrNull(it.startedAt) ?: LessonFormat.epochOrNull(it.createdAt) ?: 0L
        }

    /** Bitta kasrli son, vergul bilan (o'zbek tilida o'nlik ajratkich — vergul). */
    private fun format1(value: Double): String {
        val rounded = Math.round(value * 10) / 10.0
        val whole = rounded.toLong()
        val frac = Math.round((rounded - whole) * 10)
        return if (frac == 0L) "$whole" else "$whole,$frac"
    }
}
