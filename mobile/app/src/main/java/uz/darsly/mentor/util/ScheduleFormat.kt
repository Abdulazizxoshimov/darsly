package uz.darsly.mentor.util

import uz.darsly.mentor.data.api.Lesson
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

/**
 * Jadval ko'rinishining **sof** mantiqi: darslarni kunlar bo'yicha guruhlash.
 *
 * ## Nega darslar ro'yxatidan alohida ekran
 * "Darslar" ro'yxati holat bo'yicha guruhlangan (jonli / rejalashtirilgan /
 * tugagan) — u "hozir nima qilay?" degan savolga javob beradi. Jadval esa
 * **vaqt** bo'yicha: "shanba kuni nechta darsim bor?". Zoom'da ham ikkalasi
 * alohida ("Meet & Chat" va "Meetings" → "Upcoming").
 *
 * ## Nega sof funksiyalar
 * Kun chegaralari — sana mantiqidagi eng ko'p xato qilinadigan joy: mintaqa
 * (UTC+5 da yarim tundan keyingi dars UTC'da "kecha" bo'ladi), yil chegarasi,
 * vaqti belgilanmagan darslar. Bularning hammasi JVM testida sinaladi.
 */
object ScheduleFormat {

    /** Bitta kun va o'sha kundagi darslar (vaqt bo'yicha o'sish tartibida). */
    data class Day(
        val date: LocalDate,
        val lessons: List<Lesson>,
    )

    private val UZ_WEEKDAYS = arrayOf(
        "dushanba", "seshanba", "chorshanba", "payshanba",
        "juma", "shanba", "yakshanba",
    )

    private val UZ_MONTHS = arrayOf(
        "yanvar", "fevral", "mart", "aprel", "may", "iyun",
        "iyul", "avgust", "sentabr", "oktabr", "noyabr", "dekabr",
    )

    /**
     * Vaqti belgilangan darslarni kunlar bo'yicha guruhlaydi.
     *
     * · Faqat **vaqti bor** darslar kiradi — vaqtsiz darsni kalendarga qo'yib
     *   bo'lmaydi; ular alohida ro'yxatda ([undated]).
     * · Bekor qilingan darslar TASHLANADI: jadval "nima bo'ladi" degan savolga
     *   javob beradi, bekor qilingani esa bo'lmaydi.
     * · Kunlar o'sish tartibida, kun ichidagi darslar ham.
     * · [fromDate] dan oldingi kunlar tashlanadi (default — bugun): jadval
     *   kelajakka qaraydi. `null` berilsa hamma kunlar qoladi (tarix ko'rinishi).
     */
    fun groupByDay(
        lessons: List<Lesson>,
        zone: ZoneId = ZoneId.systemDefault(),
        fromDate: LocalDate? = LocalDate.now(zone),
    ): List<Day> = lessons
        .filter { it.status != "cancelled" }
        .mapNotNull { lesson ->
            val millis = LessonFormat.epochOrNull(lesson.scheduledAt) ?: return@mapNotNull null
            val date = Instant.ofEpochMilli(millis).atZone(zone).toLocalDate()
            if (fromDate != null && date.isBefore(fromDate)) return@mapNotNull null
            date to lesson
        }
        .groupBy({ it.first }, { it.second })
        .map { (date, items) ->
            Day(date, items.sortedBy { LessonFormat.epochOrNull(it.scheduledAt) ?: 0L })
        }
        .sortedBy { it.date }

    /**
     * Vaqti belgilanmagan darslar — jadvalning pastida alohida bo'lim.
     *
     * Ularni jimgina tashlab yuborish mumkin emas: ustoz "darsim yo'qoldi" deb
     * o'ylardi. Bekor qilinganlar bu yerda ham tashlanadi.
     */
    fun undated(lessons: List<Lesson>): List<Lesson> = lessons
        .filter { it.status != "cancelled" && LessonFormat.epochOrNull(it.scheduledAt) == null }
        .sortedByDescending { LessonFormat.epochOrNull(it.createdAt) ?: 0L }

    /** "Bugun, 27-iyul" · "Ertaga, 28-iyul" · "shanba, 1-avgust" · "1-yanvar 2027". */
    fun dayTitle(date: LocalDate, today: LocalDate = LocalDate.now()): String {
        val month = UZ_MONTHS[date.monthValue - 1]
        val dayMonth = "${date.dayOfMonth}-$month"
        // `DayOfWeek.value`: dushanba = 1 … yakshanba = 7.
        val weekday = UZ_WEEKDAYS[date.dayOfWeek.value - 1]
        return when {
            date == today -> "Bugun, $dayMonth"
            date == today.plusDays(1) -> "Ertaga, $dayMonth"
            date.year != today.year -> "$dayMonth ${date.year}"
            else -> "$weekday, $dayMonth"
        }
    }

    /**
     * Kun sarlavhasi ostidagi qisqa xulosa: "3 dars · 2 soat 30 daqiqa".
     *
     * Umumiy davomiylik ustozning haqiqiy savoliga javob beradi: "bu kun
     * qanchalik band?".
     */
    fun daySummary(day: Day): String {
        val totalMinutes = day.lessons.sumOf { it.durationMin.coerceAtLeast(0) }
        val count = "${day.lessons.size} dars"
        val duration = LessonFormat.durationLabel(totalMinutes) ?: return count
        return "$count · $duration"
    }

    /** Kun ichidagi dars uchun soat: "14:30". Vaqti yo'q bo'lsa `null`. */
    fun timeLabel(lesson: Lesson, zone: ZoneId = ZoneId.systemDefault()): String? {
        val millis = LessonFormat.epochOrNull(lesson.scheduledAt) ?: return null
        val time = Instant.ofEpochMilli(millis).atZone(zone).toLocalTime()
        return "%02d:%02d".format(time.hour, time.minute)
    }
}
