package uz.darsly.mentor.ui.blocklist

import uz.darsly.mentor.data.api.BlocklistEntry
import uz.darsly.mentor.util.LessonFormat

/**
 * Qora ro'yxat ekranining **sof** mantiqi (tartib, qidiruv, ko'rsatiladigan ism).
 *
 * Compose'dan ajratilgan — JVM testida qulflanadi.
 */
object BlocklistFormat {

    /**
     * Eng YANGI ban tepada.
     *
     * Sabab aniq: ustoz bu ekranga deyarli har doim "hozirgina xato bosdim"
     * degan holatda kiradi. Alifbo tartibi uni ro'yxat o'rtasidan qidirishga
     * majburlardi.
     */
    fun sortForDisplay(items: List<BlocklistEntry>): List<BlocklistEntry> =
        items.sortedByDescending { LessonFormat.epochOrNull(it.createdAt) ?: 0L }

    /**
     * Ro'yxatda ko'rsatiladigan ism.
     *
     * `display_name` — ban'ning HAQIQIY kaliti (moslik shu bo'yicha ketadi),
     * shuning uchun u birinchi. Bo'sh bo'lsa (eski yozuv) `identity` ga
     * tushamiz; u ham bo'sh bo'lsa — bo'sh qator o'rniga ochiq matn.
     */
    fun nameOf(entry: BlocklistEntry): String =
        entry.displayName.trim().takeIf { it.isNotEmpty() }
            ?: entry.identity.trim().takeIf { it.isNotEmpty() }
            ?: "Noma'lum ishtirokchi"
}
