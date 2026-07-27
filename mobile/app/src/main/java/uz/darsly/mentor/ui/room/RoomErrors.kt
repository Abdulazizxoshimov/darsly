package uz.darsly.mentor.ui.room

/**
 * Xonani boshlashdagi **kutilmagan** istisnolarni ustozga tushunarli matnga aylantiradi (🟡A).
 *
 * Nega alohida obyekt: [RoomViewModel] Android'ga bog'langan va JVM testida yasab
 * bo'lmaydi. Xato matni tanlash mantiqi esa sof — u shu yerda, testlar ostida.
 *
 * MUHIM KAFOLAT: hech qachon `null` yoki bo'sh satr qaytarmaydi. UI "Qayta urinish"
 * tugmasini AYNAN `error != null` bo'lganda ko'rsatadi — bo'sh xabar abadiy spinner
 * degani bo'lardi.
 */
object RoomErrors {

    /**
     * Android 12+ (API 31) da fon rejimidan foreground servis ishga tushirilsa
     * otiladi. Sinf API 31 da paydo bo'lgani uchun tipi bilan emas, **nomi** bilan
     * taqqoslanadi — `minSdk 26` da kompilyatsiya muammosi bo'lmasin.
     */
    private const val FGS_NOT_ALLOWED = "ForegroundServiceStartNotAllowedException"

    fun startFailure(t: Throwable): String = when {
        t.isOrCausedBy(FGS_NOT_ALLOWED) ->
            "Darsni ilova ekranda ochiq turganda boshlang — Android fon rejimidan " +
                "boshlashga ruxsat bermadi"

        t.isOrCausedBy("SecurityException") ->
            "Ruxsat yetarli emas — kamera, mikrofon va bildirishnoma ruxsatlarini tekshiring"

        else -> "Darsni boshlab bo'lmadi — qaytadan urinib ko'ring"
    }

    /** Istisno o'zi yoki uning sababi shu nomdagi sinfmi (o'ralgan istisnolar uchun). */
    private fun Throwable.isOrCausedBy(simpleName: String): Boolean {
        var current: Throwable? = this
        // Sabab zanjiri o'zini ko'rsatishi mumkin (`initCause(this)`) — 8 qadam bilan cheklaymiz.
        repeat(8) {
            val node = current ?: return false
            if (node::class.java.simpleName == simpleName) return true
            current = node.cause?.takeIf { it !== node }
        }
        return false
    }
}
