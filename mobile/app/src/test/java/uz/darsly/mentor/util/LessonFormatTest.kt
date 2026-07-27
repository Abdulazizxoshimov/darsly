package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import uz.darsly.mentor.data.api.Lesson
import java.time.LocalDate
import java.time.ZoneId

/**
 * M6 — ro'yxat formatlash va tartiblash.
 *
 * Bu testlar sof funksiyalarni sinaydi, lekin ular ustozning ekranida ko'rinadigan
 * AYNI matn va AYNI tartib: "Ertaga 09:00", jonli dars birinchi, va eng muhimi —
 * web bilan bir xil join havolasi (`/r/<slug>`), aks holda ustoz yuborgan havola
 * o'quvchida ochilmaydi.
 */
class LessonFormatTest {

    private val tashkent: ZoneId = ZoneId.of("Asia/Tashkent")
    private val today: LocalDate = LocalDate.of(2026, 7, 26)

    private fun lesson(
        id: String = "1",
        status: String = "scheduled",
        scheduledAt: String? = null,
        createdAt: String? = null,
        slug: String? = "abc123",
        hasPasscode: Boolean = false,
    ) = Lesson(
        id = id,
        title = "Algebra $id",
        scheduledAt = scheduledAt,
        createdAt = createdAt,
        joinSlug = slug,
        status = status,
        hasPasscode = hasPasscode,
        durationMin = 60,
    )

    // ── Bo'limlar ─────────────────────────────────────────────────────────────

    @Test
    fun `status boyicha bolim aniqlanadi`() {
        assertEquals(LessonFormat.Section.LIVE, LessonFormat.sectionOf(lesson(status = "live")))
        assertEquals(LessonFormat.Section.UPCOMING, LessonFormat.sectionOf(lesson(status = "scheduled")))
        assertEquals(LessonFormat.Section.PAST, LessonFormat.sectionOf(lesson(status = "ended")))
        assertEquals(LessonFormat.Section.PAST, LessonFormat.sectionOf(lesson(status = "cancelled")))
    }

    @Test
    fun `notanish status kelsa ilova yiqilmaydi`() {
        // Backend keyinchalik yangi status qo'shsa (masalan "paused") — ilova eski
        // versiyada ham ishlashi kerak: noma'lum status "kelayotgan" deb qaraladi.
        assertEquals(LessonFormat.Section.UPCOMING, LessonFormat.sectionOf(lesson(status = "paused")))
        assertEquals("paused", LessonFormat.statusLabel("paused"))
    }

    @Test
    fun `jonli dars birinchi keyin eng yaqin rejalashtirilgan`() {
        val list = listOf(
            lesson(id = "ended", status = "ended", scheduledAt = "2026-07-20T10:00:00Z"),
            lesson(id = "kech", scheduledAt = "2026-07-30T10:00:00Z"),
            lesson(id = "jonli", status = "live"),
            lesson(id = "yaqin", scheduledAt = "2026-07-27T10:00:00Z"),
        )
        assertEquals(
            listOf("jonli", "yaqin", "kech", "ended"),
            LessonFormat.sortForDisplay(list).map { it.id },
        )
    }

    @Test
    fun `vaqti belgilanmagan dars rejalashtirilganlar oxirida turadi`() {
        val list = listOf(
            lesson(id = "vaqtsiz", scheduledAt = null, createdAt = "2026-07-26T08:00:00Z"),
            lesson(id = "vaqtli", scheduledAt = "2026-07-29T10:00:00Z"),
        )
        assertEquals(
            listOf("vaqtli", "vaqtsiz"),
            LessonFormat.sortForDisplay(list).map { it.id },
        )
    }

    // ── Sana matni ────────────────────────────────────────────────────────────

    @Test
    fun `bugun ertaga va kecha alohida yoziladi`() {
        // 2026-07-26T09:00Z = Toshkentda 14:00 (UTC+5) — mintaqa hisobga olinishi shart.
        assertEquals(
            "Bugun 14:00",
            LessonFormat.scheduleLabel("2026-07-26T09:00:00Z", tashkent, today),
        )
        assertEquals(
            "Ertaga 09:30",
            LessonFormat.scheduleLabel("2026-07-27T04:30:00Z", tashkent, today),
        )
        assertEquals(
            "Kecha 18:00",
            LessonFormat.scheduleLabel("2026-07-25T13:00:00Z", tashkent, today),
        )
    }

    @Test
    fun `uzoq sana ozbekcha oy nomi bilan`() {
        assertEquals(
            "30-iyul, 15:00",
            LessonFormat.scheduleLabel("2026-07-30T10:00:00Z", tashkent, today),
        )
        // Boshqa yil bo'lsa yil ham ko'rsatiladi.
        assertEquals(
            "5-yanvar 2027, 15:00",
            LessonFormat.scheduleLabel("2027-01-05T10:00:00Z", tashkent, today),
        )
    }

    @Test
    fun `ofset bilan kelgan vaqt ham oqiladi`() {
        // Go `time.Time` DB'dan `+05:00` ofseti bilan ham qaytishi mumkin.
        assertEquals(
            "Bugun 14:00",
            LessonFormat.scheduleLabel("2026-07-26T14:00:00+05:00", tashkent, today),
        )
    }

    @Test
    fun `bosh yoki buzuq sana null qaytaradi`() {
        assertNull(LessonFormat.scheduleLabel(null, tashkent, today))
        assertNull(LessonFormat.scheduleLabel("", tashkent, today))
        assertNull(LessonFormat.scheduleLabel("kecha", tashkent, today))
        assertNull(LessonFormat.epochOrNull("0000-99-99"))
    }

    @Test
    fun `davomiylik soat va daqiqaga bolinadi`() {
        assertEquals("45 daqiqa", LessonFormat.durationLabel(45))
        assertEquals("1 soat", LessonFormat.durationLabel(60))
        assertEquals("1 soat 30 daqiqa", LessonFormat.durationLabel(90))
        assertNull(LessonFormat.durationLabel(0))
    }

    // ── Join havolasi (M8) ────────────────────────────────────────────────────

    @Test
    fun `join havolasi web bilan bir xil shaklda`() {
        // Manba: frontend/src/views/Dashboard.jsx → `${origin}/r/${join_slug}`
        assertEquals(
            "https://app.example.uz/r/abc123",
            LessonFormat.joinUrl("https://app.example.uz", "abc123"),
        )
        // Oxiridagi ortiqcha `/` ikkilanmaydi.
        assertEquals(
            "https://app.example.uz/r/abc123",
            LessonFormat.joinUrl("https://app.example.uz/", "abc123"),
        )
    }

    @Test
    fun `slug yoq bolsa havola yasalmaydi`() {
        assertNull(LessonFormat.joinUrl("https://app.example.uz", null))
        assertNull(LessonFormat.joinUrl("https://app.example.uz", "  "))
    }

    @Test
    fun `ulashish matnida sarlavha vaqt parol va havola bor`() {
        val text = LessonFormat.shareText(
            lesson(scheduledAt = "2026-07-30T10:00:00Z", hasPasscode = true),
            "https://app.example.uz",
            tashkent,
        )
        assertEquals(
            listOf(
                "Algebra 1",
                LessonFormat.scheduleLabel("2026-07-30T10:00:00Z", tashkent)!!,
                "Parol bilan himoyalangan",
                "https://app.example.uz/r/abc123",
            ),
            text.lines(),
        )
    }
}
