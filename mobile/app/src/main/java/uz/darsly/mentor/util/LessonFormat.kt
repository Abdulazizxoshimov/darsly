package uz.darsly.mentor.util

import uz.darsly.mentor.data.api.Lesson
import java.time.Instant
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/**
 * Darslar ro'yxatining **sof** (Android'siz) formatlash va tartiblash mantiqi (M6).
 *
 * Compose'dan ajratilgani ataylab: shu tufayli sana matni, bo'lim taqsimoti va
 * ulashish havolasi JVM unit testida sinaladi (emulyator kerak emas).
 *
 * `java.time` — `minSdk 26` da to'g'ridan-to'g'ri mavjud (desugaring shart emas).
 */
object LessonFormat {

    /** Ro'yxat bo'limlari — ustoz avval "hozir nima bo'layotganini" ko'radi. */
    enum class Section { LIVE, UPCOMING, PAST }

    private val UZ_MONTHS = arrayOf(
        "yanvar", "fevral", "mart", "aprel", "may", "iyun",
        "iyul", "avgust", "sentabr", "oktabr", "noyabr", "dekabr",
    )

    private val TIME = DateTimeFormatter.ofPattern("HH:mm")

    fun sectionOf(lesson: Lesson): Section = when (lesson.status) {
        "live" -> Section.LIVE
        "ended", "cancelled" -> Section.PAST
        else -> Section.UPCOMING
    }

    fun sectionTitle(section: Section): String = when (section) {
        Section.LIVE -> "Jonli"
        Section.UPCOMING -> "Rejalashtirilgan"
        Section.PAST -> "Tugagan"
    }

    /** Status yorlig'i — backend qiymatlari (`lesson.go` konstantalari). */
    fun statusLabel(status: String): String = when (status) {
        "live" -> "Jonli"
        "scheduled" -> "Rejalashtirilgan"
        "ended" -> "Tugagan"
        "cancelled" -> "Bekor qilingan"
        else -> status
    }

    /**
     * Ro'yxatni ekran tartibiga soladi:
     *  1. Jonli darslar (eng muhimi — ustoz bir bosishda kiradi)
     *  2. Rejalashtirilganlar — **eng yaqini birinchi**; vaqti belgilanmaganlari oxirida
     *  3. Tugaganlar — eng yangisi birinchi
     */
    fun sortForDisplay(lessons: List<Lesson>): List<Lesson> {
        val bySection = lessons.groupBy { sectionOf(it) }
        val live = bySection[Section.LIVE].orEmpty()
            .sortedByDescending { epochOrNull(it.createdAt) ?: 0L }
        val upcoming = bySection[Section.UPCOMING].orEmpty()
            .sortedWith(
                compareBy<Lesson> { epochOrNull(it.scheduledAt) ?: Long.MAX_VALUE }
                    .thenByDescending { epochOrNull(it.createdAt) ?: 0L },
            )
        val past = bySection[Section.PAST].orEmpty()
            .sortedByDescending { epochOrNull(it.scheduledAt) ?: epochOrNull(it.createdAt) ?: 0L }
        return live + upcoming + past
    }

    /**
     * "Bugun 14:30" · "Ertaga 09:00" · "27-iyul, 14:30" · "27-iyul 2027, 14:30".
     * Vaqti yo'q dars uchun `null` (UI "vaqti belgilanmagan" deb yozadi).
     */
    fun scheduleLabel(
        iso: String?,
        zone: ZoneId = ZoneId.systemDefault(),
        today: LocalDate = LocalDate.now(zone),
    ): String? {
        val instant = parseOrNull(iso) ?: return null
        val local = instant.atZone(zone)
        val date = local.toLocalDate()
        val time = TIME.format(local)
        return when (date) {
            today -> "Bugun $time"
            today.plusDays(1) -> "Ertaga $time"
            today.minusDays(1) -> "Kecha $time"
            else -> {
                val month = UZ_MONTHS[date.monthValue - 1]
                val year = if (date.year == today.year) "" else " ${date.year}"
                "${date.dayOfMonth}-$month$year, $time"
            }
        }
    }

    /** "60 daqiqa" · "1 soat 30 daqiqa" — ustozga tanish shakl. */
    fun durationLabel(minutes: Int): String? {
        if (minutes <= 0) return null
        val h = minutes / 60
        val m = minutes % 60
        return when {
            h == 0 -> "$m daqiqa"
            m == 0 -> "$h soat"
            else -> "$h soat $m daqiqa"
        }
    }

    /**
     * O'quvchiga yuboriladigan havola (M8).
     *
     * Shakl web frontend bilan **bir xil** bo'lishi shart:
     * `frontend/src/views/Dashboard.jsx` → `${origin}/r/${join_slug}` (o'sha faylga tegilmagan,
     * faqat o'qilgan). Farq bo'lsa ustoz yuborgan havola ochilmaydi.
     */
    fun joinUrl(baseUrl: String, joinSlug: String?): String? {
        val slug = joinSlug?.trim()?.takeIf { it.isNotBlank() } ?: return null
        return baseUrl.trimEnd('/') + "/r/" + slug
    }

    /** Telegramga yuboriladigan matn: sarlavha + vaqt + havola. */
    fun shareText(lesson: Lesson, baseUrl: String, zone: ZoneId = ZoneId.systemDefault()): String {
        val url = joinUrl(baseUrl, lesson.joinSlug)
        return buildString {
            append(lesson.title)
            scheduleLabel(lesson.scheduledAt, zone)?.let { append("\n").append(it) }
            if (lesson.hasPasscode) append("\nParol bilan himoyalangan")
            url?.let { append("\n").append(it) }
        }
    }

    /** ISO-8601 → epoch millis; noto'g'ri/bo'sh qiymat uchun `null` (crash yo'q). */
    fun epochOrNull(iso: String?): Long? = parseOrNull(iso)?.toEpochMilli()

    private fun parseOrNull(iso: String?): Instant? {
        val raw = iso?.trim()?.takeIf { it.isNotBlank() } ?: return null
        // Go `time.Time` RFC3339 (`Z` yoki `+05:00`) beradi; ikkalasi ham qo'llab-quvvatlanadi.
        return runCatching { OffsetDateTime.parse(raw).toInstant() }
            .recoverCatching { Instant.parse(raw) }
            .getOrNull()
    }
}
