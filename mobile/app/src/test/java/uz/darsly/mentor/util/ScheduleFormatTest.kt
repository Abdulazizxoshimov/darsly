package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.Lesson
import java.time.LocalDate
import java.time.ZoneId

/**
 * Jadval — kunlar bo'yicha guruhlash.
 *
 * Sana mantiqi bu loyihada allaqachon bir marta xato bergan (Material3
 * `DatePicker` ning UTC yarim tuni). Shuning uchun bu yerda mintaqa chekka
 * holatlari **manfiy ofsetli** mintaqa bilan ham tekshiriladi: UTC+5 da ikkala
 * (to'g'ri va noto'g'ri) implementatsiya bir xil natija berishi mumkin.
 */
class ScheduleFormatTest {

    private val tashkent: ZoneId = ZoneId.of("Asia/Tashkent") // UTC+5
    private val newYork: ZoneId = ZoneId.of("America/New_York") // UTC−4

    private fun lesson(
        id: String,
        scheduledAt: String?,
        status: String = "scheduled",
        durationMin: Int = 60,
        createdAt: String? = null,
    ) = Lesson(
        id = id,
        title = "Dars $id",
        scheduledAt = scheduledAt,
        status = status,
        durationMin = durationMin,
        createdAt = createdAt,
    )

    // ─── Guruhlash ────────────────────────────────────────────────────────────

    @Test
    fun `bir kundagi darslar bitta guruhga tushadi`() {
        val lessons = listOf(
            lesson("a", "2026-07-27T09:00:00Z"),
            lesson("b", "2026-07-27T12:00:00Z"),
            lesson("c", "2026-07-28T09:00:00Z"),
        )
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 1, 1))
        assertEquals(2, days.size)
        assertEquals(2, days[0].lessons.size)
        assertEquals(1, days[1].lessons.size)
    }

    @Test
    fun `kun ichida vaqt bilan tartiblanadi`() {
        val lessons = listOf(
            lesson("kech", "2026-07-27T15:00:00Z"),
            lesson("erta", "2026-07-27T06:00:00Z"),
        )
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 1, 1))
        assertEquals(listOf("erta", "kech"), days[0].lessons.map { it.id })
    }

    @Test
    fun `kunlar osish tartibida`() {
        val lessons = listOf(
            lesson("kelasi", "2026-08-01T09:00:00Z"),
            lesson("bugun", "2026-07-27T09:00:00Z"),
        )
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 1, 1))
        assertEquals(listOf("bugun", "kelasi"), days.map { it.lessons.first().id })
    }

    @Test
    fun `kun chegarasi qurilma mintaqasida hisoblanadi`() {
        // ⭐ UTC 21:00 → Toshkentda ERTASI kun 02:00. Guruhlash UTC'da qilinsa
        // dars noto'g'ri kunga tushardi va ustoz uni jadvalda topa olmasdi.
        val lessons = listOf(lesson("a", "2026-07-27T21:00:00Z"))
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 1, 1))
        assertEquals(LocalDate.of(2026, 7, 28), days[0].date)
    }

    @Test
    fun `manfiy ofsetli mintaqada ham togri kun`() {
        // Nyu-Yorkda (UTC−4) UTC ertalabki 02:00 — OLDINGI kun 22:00.
        // Bu holat UTC+5 bilan tekshirilmasdi, ya'ni noto'g'ri implementatsiya
        // sezilmay qolardi.
        val lessons = listOf(lesson("a", "2026-07-28T02:00:00Z"))
        val days = ScheduleFormat.groupByDay(lessons, newYork, fromDate = LocalDate.of(2026, 1, 1))
        assertEquals(LocalDate.of(2026, 7, 27), days[0].date)
    }

    @Test
    fun `otgan kunlar sukut boyicha tashlanadi`() {
        val lessons = listOf(
            lesson("otgan", "2026-07-20T09:00:00Z"),
            lesson("kelasi", "2026-07-30T09:00:00Z"),
        )
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 7, 27))
        assertEquals(1, days.size)
        assertEquals("kelasi", days[0].lessons.first().id)
    }

    @Test
    fun `bugungi kun tashlanmaydi`() {
        // Chegara `isBefore` bo'lishi kerak, `!isAfter` emas: aks holda bugungi
        // darslar jadvaldan yo'qolardi — eng ko'p kerak bo'ladigan kun.
        val lessons = listOf(lesson("bugun", "2026-07-27T09:00:00Z"))
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 7, 27))
        assertEquals(1, days.size)
    }

    @Test
    fun `fromDate null bolsa hamma kunlar qoladi`() {
        val lessons = listOf(
            lesson("otgan", "2020-01-01T09:00:00Z"),
            lesson("kelasi", "2030-01-01T09:00:00Z"),
        )
        assertEquals(2, ScheduleFormat.groupByDay(lessons, tashkent, fromDate = null).size)
    }

    @Test
    fun `bekor qilingan dars jadvalda korinmaydi`() {
        val lessons = listOf(
            lesson("bekor", "2026-07-28T09:00:00Z", status = "cancelled"),
            lesson("normal", "2026-07-28T10:00:00Z"),
        )
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 1, 1))
        assertEquals(listOf("normal"), days[0].lessons.map { it.id })
    }

    @Test
    fun `vaqti belgilanmagan dars kunlarga tushmaydi`() {
        val lessons = listOf(lesson("vaqtsiz", null))
        assertTrue(ScheduleFormat.groupByDay(lessons, tashkent, fromDate = null).isEmpty())
    }

    // ─── Vaqtsiz darslar ──────────────────────────────────────────────────────

    @Test
    fun `vaqtsiz darslar alohida royxatda`() {
        // Ularni jimgina tashlab yuborish mumkin emas — ustoz "darsim yo'qoldi"
        // deb o'ylardi.
        val lessons = listOf(
            lesson("vaqtsiz", null, createdAt = "2026-07-01T09:00:00Z"),
            lesson("vaqtli", "2026-07-28T09:00:00Z"),
        )
        assertEquals(listOf("vaqtsiz"), ScheduleFormat.undated(lessons).map { it.id })
    }

    @Test
    fun `bekor qilingan vaqtsiz dars ham korinmaydi`() {
        val lessons = listOf(lesson("bekor", null, status = "cancelled"))
        assertTrue(ScheduleFormat.undated(lessons).isEmpty())
    }

    // ─── Sarlavhalar ──────────────────────────────────────────────────────────

    @Test
    fun `kun sarlavhasi bugun va ertaga uchun maxsus`() {
        val today = LocalDate.of(2026, 7, 27)
        assertEquals("Bugun, 27-iyul", ScheduleFormat.dayTitle(today, today))
        assertEquals("Ertaga, 28-iyul", ScheduleFormat.dayTitle(today.plusDays(1), today))
    }

    @Test
    fun `boshqa kun hafta kuni bilan korsatiladi`() {
        val today = LocalDate.of(2026, 7, 27) // dushanba
        // 2026-08-01 — shanba.
        assertEquals("shanba, 1-avgust", ScheduleFormat.dayTitle(LocalDate.of(2026, 8, 1), today))
    }

    @Test
    fun `boshqa yil sana yil bilan korsatiladi`() {
        val today = LocalDate.of(2026, 7, 27)
        assertEquals("1-yanvar 2027", ScheduleFormat.dayTitle(LocalDate.of(2027, 1, 1), today))
    }

    @Test
    fun `kun xulosasi dars soni va umumiy davomiylikni beradi`() {
        val day = ScheduleFormat.Day(
            LocalDate.of(2026, 7, 27),
            listOf(
                lesson("a", "2026-07-27T09:00:00Z", durationMin = 60),
                lesson("b", "2026-07-27T12:00:00Z", durationMin = 90),
            ),
        )
        assertEquals("2 dars · 2 soat 30 daqiqa", ScheduleFormat.daySummary(day))
    }

    @Test
    fun `davomiyligi nol bolsa faqat soni korsatiladi`() {
        val day = ScheduleFormat.Day(
            LocalDate.of(2026, 7, 27),
            listOf(lesson("a", "2026-07-27T09:00:00Z", durationMin = 0)),
        )
        assertEquals("1 dars", ScheduleFormat.daySummary(day))
    }

    @Test
    fun `soat qurilma mintaqasida korsatiladi`() {
        // UTC 09:30 → Toshkentda 14:30.
        assertEquals("14:30", ScheduleFormat.timeLabel(lesson("a", "2026-07-27T09:30:00Z"), tashkent))
        // Bir xonali soat nol bilan to'ldiriladi — ustun tekis tursin.
        assertEquals("06:05", ScheduleFormat.timeLabel(lesson("a", "2026-07-27T01:05:00Z"), tashkent))
    }

    // ─── «O'tgan darslarni ham ko'rsatish» tumbleri ──────────────────────────
    //
    // 2026-08-04 da emulyatorda topilgan nuqson: tumbler faqat SANALI bo'limga
    // ta'sir qilardi. Tezkor darsning sanasi yo'q, shuning uchun ular jadvalda
    // abadiy qolib, ekranni to'ldirib yuborardi.

    @Test
    fun `tumbler ochiq bolsa vaqtsiz TUGAGAN dars korinmaydi`() {
        val lessons = listOf(
            lesson("tugagan", null, status = "ended"),
            lesson("kutilmoqda", null, status = "scheduled"),
        )
        val got = ScheduleFormat.undated(lessons, includePast = false)
        assertEquals(listOf("kutilmoqda"), got.map { it.id })
    }

    @Test
    fun `tumbler yoniq bolsa vaqtsiz tugagan dars ham korinadi`() {
        val lessons = listOf(
            lesson("tugagan", null, status = "ended"),
            lesson("kutilmoqda", null, status = "scheduled"),
        )
        val got = ScheduleFormat.undated(lessons, includePast = true)
        assertEquals(setOf("tugagan", "kutilmoqda"), got.map { it.id }.toSet())
    }

    // Sanasi KELAJAKDA, lekin allaqachon o'tkazilgan dars ham "o'tgan" hisoblanadi —
    // aks holda ustoz uni yana o'tkazishi kerakdek ko'rinib turardi.
    @Test
    fun `kelajak sanali TUGAGAN dars tumbler ochiqda yashiriladi`() {
        val lessons = listOf(
            lesson("tugagan", "2026-08-01T09:00:00Z", status = "ended"),
            lesson("kelgusi", "2026-08-01T12:00:00Z", status = "scheduled"),
        )
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = LocalDate.of(2026, 1, 1))
        assertEquals(listOf("kelgusi"), days.flatMap { it.lessons }.map { it.id })
    }

    @Test
    fun `tumbler yoniqda kelajak sanali tugagan dars ham qoladi`() {
        val lessons = listOf(
            lesson("tugagan", "2026-08-01T09:00:00Z", status = "ended"),
            lesson("kelgusi", "2026-08-01T12:00:00Z", status = "scheduled"),
        )
        val days = ScheduleFormat.groupByDay(lessons, tashkent, fromDate = null)
        assertEquals(setOf("tugagan", "kelgusi"), days.flatMap { it.lessons }.map { it.id }.toSet())
    }
}
