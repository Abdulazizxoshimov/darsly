package uz.darsly.mentor.util

/**
 * Chatda fayl ulashishning **sof** qoidalari (Android'siz, JVM testida).
 *
 * ## Nega klientda ham tekshiriladi
 * Server baribir tekshiradi (`backend/internal/usecase/chat/file.go`) va u
 * yagona haqiqat manbai. Lekin 20 MB'lik faylni sekin mobil internetda
 * YUKLAB BO'LIB, keyin 400 olish — ustoz uchun bir necha daqiqa yo'qotilgan
 * vaqt va "ilova buzuq" degan taassurot. Shuning uchun aniq-ravshan rad
 * javoblari (hajm, kengaytma) so'rov YUBORILMASDAN oldin beriladi.
 *
 * ## Nega mazmun sniff'i takrorlanmaydi
 * Server kengaytma allowlist'idan tashqari mazmunni ham tekshiradi
 * (`http.DetectContentType`) — bu XAVFSIZLIK tekshiruvi va uni klientda
 * takrorlash hech nima bermaydi (hujumchi klientni chetlab o'tadi), lekin
 * noto'g'ri "bu fayl yaroqsiz" degan yolg'on rad javoblarini keltirib
 * chiqarardi. Shu sabab bu yerda faqat FOYDALANUVCHI xatolari ushlanadi.
 */
object ChatUpload {

    /** Server chegarasi (`chat.MaxChatFileBytes`) — 20 MB. */
    const val MAX_BYTES: Long = 20L * 1024 * 1024

    /**
     * Ruxsat etilgan kengaytmalar — `allowedChatFiles` bilan bir xil.
     * Backend ro'yxati o'zgarsa bu ham yangilanishi kerak.
     */
    val ALLOWED_EXTENSIONS: Set<String> = setOf(
        "jpg", "jpeg", "png", "gif", "webp",
        "pdf",
        "docx", "xlsx", "pptx", "doc", "xls", "ppt",
        "txt", "csv",
    )

    /** Fayl tanlash oynasiga beriladigan MIME filtri. */
    const val PICKER_MIME = "*/*"

    /** Fayl nomi noma'lum bo'lsa (ba'zi provayderlar `DISPLAY_NAME` bermaydi). */
    const val FALLBACK_NAME = "fayl"

    /**
     * Nomdagi yo'lni tashlaydi (`../../etc/x.png` → `x.png`) va bo'sh nomni
     * [FALLBACK_NAME] bilan almashtiradi.
     *
     * Server ham shunday qiladi, lekin bizga toza nom TANLASHDAN keyin darhol
     * kerak: u tasdiqlash oynasida ko'rsatiladi.
     */
    fun sanitizeName(raw: String?): String {
        val cleaned = raw.orEmpty()
            .replace('\\', '/')
            .substringAfterLast('/')
            .trim()
        return cleaned.takeIf { it.isNotEmpty() && it != "." } ?: FALLBACK_NAME
    }

    /** Kichik harfli kengaytma (nuqtasiz). Kengaytmasiz nom uchun bo'sh satr. */
    fun extensionOf(name: String): String {
        val base = sanitizeName(name)
        val dot = base.lastIndexOf('.')
        if (dot <= 0 || dot == base.lastIndex) return ""
        return base.substring(dot + 1).lowercase()
    }

    fun isAllowed(name: String): Boolean = extensionOf(name) in ALLOWED_EXTENSIONS

    /**
     * Tanlangan fayl yuborilishi mumkinmi. `null` — mumkin, aks holda
     * foydalanuvchiga ko'rsatiladigan o'zbekcha sabab.
     *
     * Tartib ataylab: avval bo'shlik, keyin hajm, keyin tur. Bir vaqtning
     * o'zida ikki muammo bo'lsa ham bitta ANIQ jumla ko'rsatiladi.
     */
    fun validate(name: String, size: Long): String? = when {
        size <= 0 -> "Fayl bo'sh — boshqasini tanlang"
        size > MAX_BYTES -> "Fayl juda katta (${sizeLabel(size)}). Eng ko'pi 20 MB"
        !isAllowed(name) ->
            "Bu turdagi fayl yuborilmaydi. Rasm, PDF, Office hujjati yoki matn fayli tanlang"
        else -> null
    }

    /** "12,4 MB" — [RecordingFormat.sizeLabel] bilan bir xil ko'rinish. */
    fun sizeLabel(bytes: Long): String = RecordingFormat.sizeLabel(bytes) ?: "0 B"

    /**
     * Kartochka ustidagi belgi. Turi bo'yicha guruhlanadi — ustoz ro'yxatda
     * "rasmmi yoki hujjatmi" ni o'qimasdan ajratsin.
     */
    fun icon(name: String): String = when (extensionOf(name)) {
        "jpg", "jpeg", "png", "gif", "webp" -> "🖼"
        "pdf" -> "📕"
        "doc", "docx" -> "📄"
        "xls", "xlsx", "csv" -> "📊"
        "ppt", "pptx" -> "📽"
        else -> "📎"
    }
}
