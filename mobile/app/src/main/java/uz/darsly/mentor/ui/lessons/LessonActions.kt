package uz.darsly.mentor.ui.lessons

/**
 * Dars kartasida qaysi amal ochiq — **sof** mantiq (JVM testida).
 *
 * ## Nega kerak
 * Avval karta va uning asosiy tugmasi HAR QANDAY dars uchun xona ekranini ochardi.
 * Yakunlangan darsni bosganda ham "dars o'tish oynasi" ochilib, keyin serverdan
 * `lesson is not active` (400) qaytib xato kartasi chiqardi. Ya'ni interfeys
 * qila olmaydigan ishni taklif qilardi va xatoni foydalanuvchiga tashlardi.
 *
 * Server allaqachon to'g'ri ishlaydi (`room.HostToken`: `ended`/`cancelled` →
 * 400), shuning uchun bu qoida uni TAKRORLAYDI — ekranda esa boshidanoq
 * to'g'ri tugma turadi.
 */
object LessonActions {

    enum class Primary {
        /** Dars hali boshlanmagan — xona ochiladi. */
        START,

        /** Dars jonli — xonaga qaytiladi. */
        RESUME,

        /** Dars yakunlangan — xona emas, yozuvlar ro'yxati. */
        RECORDINGS,

        /** Bekor qilingan dars — ochiladigan hech narsa yo'q. */
        NONE,
    }

    fun primary(status: String): Primary = when (status) {
        "live" -> Primary.RESUME
        "ended" -> Primary.RECORDINGS
        "cancelled" -> Primary.NONE
        else -> Primary.START
    }

    fun label(p: Primary): String = when (p) {
        Primary.START -> "Boshlash"
        Primary.RESUME -> "Davom etish"
        Primary.RECORDINGS -> "Yozuvlar"
        Primary.NONE -> "Bekor qilingan"
    }

    /** Kartaning O'ZI bosilganda xona ochiladimi. */
    fun canOpenRoom(status: String): Boolean {
        val p = primary(status)
        return p == Primary.START || p == Primary.RESUME
    }

    /**
     * Dars tahrirlanadimi. Tugagan/bekor qilingan dars — FAQAT O'QILADI (o'chirish
     * mumkin, tahrirlash yo'q): tahrirlash maydonlari sessiyani boshqaradi, sessiya
     * esa allaqachon tugagan — o'zgartirish ma'nosiz va chalg'ituvchi (2026-08-15).
     */
    fun isEditable(status: String): Boolean = status != "ended" && status != "cancelled"
}
