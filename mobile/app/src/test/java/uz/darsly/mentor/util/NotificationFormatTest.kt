package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test
import uz.darsly.mentor.data.api.Notification

/** Bildirishnomalar ro'yxatining formatlash va birlashtirish mantiqi. */
class NotificationFormatTest {

    /** 2026-07-27T12:00:00Z */
    private val now = 1_785_153_600_000L

    private fun at(offsetMillis: Long): String =
        java.time.Instant.ofEpochMilli(now - offsetMillis).toString()

    private fun n(id: String, createdAt: String? = null, read: Boolean = false) = Notification(
        id = id,
        title = "Dars",
        createdAt = createdAt,
        readAt = if (read) "2026-07-27T11:00:00Z" else null,
    )

    // ─── Nisbiy vaqt ──────────────────────────────────────────────────────────

    @Test
    fun `nisbiy vaqt togri hisoblanadi`() {
        assertEquals("Hozirgina", NotificationFormat.relativeTime(at(30_000), now))
        assertEquals("5 daqiqa oldin", NotificationFormat.relativeTime(at(5 * 60_000), now))
        assertEquals("3 soat oldin", NotificationFormat.relativeTime(at(3 * 3_600_000), now))
    }

    @Test
    fun `bir kundan oshsa mutlaq sanaga otiladi`() {
        // "312 soat oldin" ma'nosiz — bir kundan keyin sana o'qilishi osonroq.
        val label = NotificationFormat.relativeTime(at(50L * 3_600_000), now)
        assertNotNull(label)
        assertEquals(false, label!!.contains("oldin"))
    }

    @Test
    fun `chegarada daqiqa soatga otadi`() {
        // AYNAN 60 daqiqa — chegara noto'g'ri qo'yilsa "60 daqiqa oldin" chiqardi.
        assertEquals("1 soat oldin", NotificationFormat.relativeTime(at(60 * 60_000), now))
        assertEquals("59 daqiqa oldin", NotificationFormat.relativeTime(at(59 * 60_000), now))
    }

    @Test
    fun `kelajakdagi vaqt manfiy korsatilmaydi`() {
        // Qurilma soati orqada bo'lsa: "−3 daqiqa oldin" ilova buzuq degan
        // taassurot beradi. Bunday holda mutlaq sanaga tushamiz.
        val future = java.time.Instant.ofEpochMilli(now + 600_000).toString()
        val label = NotificationFormat.relativeTime(future, now)
        assertEquals(false, label.orEmpty().startsWith("-"))
    }

    @Test
    fun `vaqtsiz yoki buzuq qiymat null qaytaradi`() {
        assertNull(NotificationFormat.relativeTime(null, now))
        assertNull(NotificationFormat.relativeTime("", now))
        assertNull(NotificationFormat.relativeTime("bu sana emas", now))
    }

    // ─── Birlashtirish ────────────────────────────────────────────────────────

    @Test
    fun `yangi bildirishnoma royxat boshiga qoshiladi`() {
        val existing = listOf(n("a"), n("b"))
        val merged = NotificationFormat.merge(existing, n("c"))
        assertEquals(listOf("c", "a", "b"), merged.map { it.id })
    }

    @Test
    fun `dublikat ikki marta chiqmaydi`() {
        // ⭐ Eng muhim mezon: backend WS orqali SNAPSHOT ham yuboradi
        // (`DeliverPendingSnapshot`), ya'ni bitta xabar ikki yo'ldan kelishi
        // deyarli kafolatlangan.
        val existing = listOf(n("a"), n("b"))
        val merged = NotificationFormat.merge(existing, n("a"))
        assertEquals(2, merged.size)
        assertEquals(listOf("a", "b"), merged.map { it.id })
    }

    @Test
    fun `dublikatda yangi nusxa qoladi`() {
        val existing = listOf(n("a", read = true))
        val merged = NotificationFormat.merge(existing, n("a", read = false))
        assertEquals(true, merged.first().isUnread)
    }

    // ─── Nishon ───────────────────────────────────────────────────────────────

    @Test
    fun `oqilmaganlar sanaladi`() {
        val items = listOf(n("a"), n("b", read = true), n("c"))
        assertEquals(2, NotificationFormat.unreadCount(items))
    }

    @Test
    fun `bosh readAt oqilmagan hisoblanadi`() {
        // Backend `read_at` ni FAQAT o'qilganda yuboradi (`omitempty`), lekin
        // bo'sh satr yuborilsa ham "o'qilmagan" bo'lishi kerak.
        assertEquals(true, Notification(id = "x", readAt = "").isUnread)
        assertEquals(true, Notification(id = "x", readAt = null).isUnread)
        assertEquals(false, Notification(id = "x", readAt = "2026-07-27T10:00:00Z").isUnread)
    }

    @Test
    fun `nishon yorligi chegarada 99 plus boladi`() {
        assertNull(NotificationFormat.badgeLabel(0))
        assertNull(NotificationFormat.badgeLabel(-3))
        assertEquals("1", NotificationFormat.badgeLabel(1))
        assertEquals("99", NotificationFormat.badgeLabel(99))
        assertEquals("99+", NotificationFormat.badgeLabel(100))
    }

    @Test
    fun `notanish tur umumiy nom oladi`() {
        assertEquals("Dars eslatmasi", NotificationFormat.typeLabel("lesson_reminder"))
        assertEquals("Bildirishnoma", NotificationFormat.typeLabel("poll_started"))
    }
}
