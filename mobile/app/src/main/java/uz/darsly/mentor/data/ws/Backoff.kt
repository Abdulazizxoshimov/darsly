package uz.darsly.mentor.data.ws

/**
 * Qayta ulanish kechikishi — **sof** hisob (JVM testida sinaladi).
 *
 * Web klienti bilan bir xil formula (`frontend/src/lib/ws.js`):
 * `min(15000, 1000 * 2^urinish)` — 1s, 2s, 4s, 8s, 15s, 15s…
 *
 * Nega cheklov muhim: ustozning interneti dars davomida bir necha marta uzilishi mumkin.
 * Cheklovsiz eksponensial o'sish yarim soatlik pauzaga olib kelardi va dars qayta
 * ulanmasdan qolardi. 15 soniya — ustoz sezadigan, lekin serverni urmaydigan chegara.
 */
object Backoff {

    const val BASE_MS = 1_000L
    const val MAX_MS = 15_000L

    /** Jitter ulushi: kechikish `[d, d + d·JITTER)` oralig'ida tarqaladi. */
    const val JITTER = 0.25

    /** @param attempt 0 dan boshlanadi (birinchi uzilishdan keyingi urinish). */
    fun delayMs(attempt: Int): Long {
        if (attempt <= 0) return BASE_MS
        // 2^attempt tez o'sadi — 63 dan katta siljish `Long` ni buzadi, shuning uchun
        // chegaradan oshgan urinishlar darhol maksimumga tushadi.
        if (attempt >= 32) return MAX_MS
        val exp = BASE_MS shl attempt
        return if (exp > MAX_MS || exp < 0) MAX_MS else exp
    }

    /**
     * Jitter'li kechikish (M1).
     *
     * Server deploy'dan keyin qayta ko'tarilganda BARCHA mentorlar bir vaqtda
     * uzilib, bir xil jadval bilan qaytib uradi ("thundering herd"). Tasodifiy
     * qo'shimcha ularni vaqt bo'ylab yoyadi. [unit] — `[0, 1)` oralig'idagi
     * tasodifiy son; test uchun oshkora parametr.
     */
    fun jitteredMs(attempt: Int, unit: Double): Long {
        val base = delayMs(attempt)
        val u = unit.coerceIn(0.0, 1.0)
        return base + (base * JITTER * u).toLong()
    }
}
