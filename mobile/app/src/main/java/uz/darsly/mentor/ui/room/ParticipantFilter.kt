package uz.darsly.mentor.ui.room

/**
 * Ishtirokchilar panelidagi qidiruvning **sof** mantiqi.
 *
 * ## Nega kerak
 * Ommaviy darsda (200–300 kishi) ustoz shovqin qilayotgan aniq bir o'quvchini
 * mute qilishi yoki chiqarishi kerak. Qidiruvsiz bu amal 300 qatorli ro'yxatni
 * telefon ekranida ko'z bilan varaqlashga aylanadi — ya'ni amalda imkonsiz.
 *
 * ## Nega Compose'dan ajratilgan
 * Filtr qoidalari (bo'sh so'rov, katta-kichik harf, ortiqcha bo'shliq) — aynan
 * jimgina buzilib ketadigan turdagi mantiq. Bu yerda ular JVM testida
 * qulflangan, UI esa faqat natijani chizadi.
 */
object ParticipantFilter {

    /**
     * Qidiruv maydoni shu SONDAN boshlab ko'rsatiladi.
     *
     * Kichik ro'yxatda (odatiy 5–8 kishilik guruh) maydon foyda bermaydi:
     * ustoz ismni baribir bir qarashda ko'radi, maydon esa panel tepasidan joy
     * oladi va klaviaturani ochib ro'yxatni yopib qo'yadi.
     */
    const val SEARCH_MIN_COUNT = 10

    /** Ro'yxat shunchalik kattami — qidiruv maydoni kerakmi. */
    fun shouldShowSearch(count: Int): Boolean = count >= SEARCH_MIN_COUNT

    /**
     * Ismi [query] ni o'z ichiga olgan qatorlar.
     *
     * Qoidalar:
     *  · bo'sh (yoki faqat bo'shliqli) so'rov → ro'yxat O'ZGARMAYDI, ya'ni
     *    maydonni tozalash "hech kim topilmadi" holatiga tushirmaydi;
     *  · katta-kichik harf farqsiz (`lowercase()` — Locale'ga bog'liq emas,
     *    turkcha "i" tuzog'i yo'q);
     *  · **ichki moslik**: "vali" so'rovi "Alijon Valiyev" ni ham topadi, chunki
     *    ustoz ko'pincha familiya bo'yicha qidiradi;
     *  · tartib saqlanadi — ro'yxat sakramaydi.
     */
    fun <T> filter(items: List<T>, query: String, name: (T) -> String): List<T> {
        val needle = query.trim().lowercase()
        if (needle.isEmpty()) return items
        return items.filter { name(it).lowercase().contains(needle) }
    }

    /** [filter] ning ishtirokchi qatori uchun tayyor ko'rinishi. */
    fun roster(items: List<RosterEntry>, query: String): List<RosterEntry> =
        filter(items, query) { it.name }
}
