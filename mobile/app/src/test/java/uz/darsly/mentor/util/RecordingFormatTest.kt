package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import uz.darsly.mentor.data.api.Recording

/**
 * Yozuvlar ekranidagi matnlar.
 *
 * Bu funksiyalar foydalanuvchi ko'radigan yagona haqiqat: hajm noto'g'ri
 * hisoblansa yoki "0 B" chiqsa, ustoz yozuv buzilgan deb o'ylaydi.
 */
class RecordingFormatTest {

    private fun rec(
        id: String = "r1",
        status: String = RecordingFormat.STATUS_READY,
        startedAt: String? = null,
        createdAt: String? = null,
    ) = Recording(id = id, status = status, startedAt = startedAt, createdAt = createdAt)

    // ─── Hajm ─────────────────────────────────────────────────────────────────

    @Test
    fun `hajm ikkilik prefikslar bilan hisoblanadi`() {
        assertEquals("512 B", RecordingFormat.sizeLabel(512))
        assertEquals("1 KB", RecordingFormat.sizeLabel(1024))
        assertEquals("1 MB", RecordingFormat.sizeLabel(1024L * 1024))
        assertEquals("1,5 MB", RecordingFormat.sizeLabel(1024L * 1024 * 3 / 2))
        assertEquals("2,5 GB", RecordingFormat.sizeLabel((2.5 * 1024 * 1024 * 1024).toLong()))
    }

    @Test
    fun `nol hajm null qaytaradi`() {
        // "0 B" — yozuv bo'sh degan yolg'on ma'no. `processing` holatida server
        // hajmni hali bilmaydi, shuning uchun hech narsa ko'rsatilmaydi.
        assertNull(RecordingFormat.sizeLabel(0))
        assertNull(RecordingFormat.sizeLabel(-1))
    }

    @Test
    fun `juda katta hajm ham togri birlikda`() {
        // 1024 dan katta bo'linishlar zanjiri: chegara noto'g'ri yozilsa
        // "1024 GB" chiqardi.
        assertEquals("1 TB", RecordingFormat.sizeLabel(1024L * 1024 * 1024 * 1024))
    }

    // ─── Davomiylik ───────────────────────────────────────────────────────────

    @Test
    fun `davomiylik togri formatlanadi`() {
        assertEquals("45 soniya", RecordingFormat.durationLabel(45))
        assertEquals("12 daqiqa", RecordingFormat.durationLabel(12 * 60))
        assertEquals("1 soat 30 daqiqa", RecordingFormat.durationLabel(90 * 60))
        assertEquals("2 soat", RecordingFormat.durationLabel(120 * 60))
    }

    @Test
    fun `nol davomiylik null qaytaradi`() {
        assertNull(RecordingFormat.durationLabel(0))
        assertNull(RecordingFormat.durationLabel(-5))
    }

    @Test
    fun `chegarada soniya daqiqaga otadi`() {
        assertEquals("59 soniya", RecordingFormat.durationLabel(59))
        assertEquals("1 daqiqa", RecordingFormat.durationLabel(60))
    }

    // ─── Status ───────────────────────────────────────────────────────────────

    @Test
    fun `notanish status ozi korsatiladi`() {
        assertEquals("Tayyor", RecordingFormat.statusLabel(RecordingFormat.STATUS_READY))
        assertEquals("aborted", RecordingFormat.statusLabel("aborted"))
    }

    // ─── Tartib ───────────────────────────────────────────────────────────────

    @Test
    fun `eng yangi yozuv birinchi`() {
        val list = listOf(
            rec("eski", startedAt = "2026-07-20T10:00:00Z"),
            rec("yangi", startedAt = "2026-07-27T10:00:00Z"),
            rec("orta", startedAt = "2026-07-25T10:00:00Z"),
        )
        assertEquals(
            listOf("yangi", "orta", "eski"),
            RecordingFormat.sortForDisplay(list).map { it.id },
        )
    }

    @Test
    fun `startedAt yoq bolsa createdAt ishlatiladi`() {
        val list = listOf(
            rec("a", startedAt = null, createdAt = "2026-07-27T10:00:00Z"),
            rec("b", startedAt = "2026-07-20T10:00:00Z"),
        )
        assertEquals(listOf("a", "b"), RecordingFormat.sortForDisplay(list).map { it.id })
    }

    @Test
    fun `vaqtsiz yozuv oxirida qoladi`() {
        val list = listOf(
            rec("vaqtsiz"),
            rec("vaqtli", startedAt = "2026-07-20T10:00:00Z"),
        )
        assertEquals(listOf("vaqtli", "vaqtsiz"), RecordingFormat.sortForDisplay(list).map { it.id })
    }

    // ─── `expired` holati ────────────────────────────────────────────────────

    @Test
    fun `expired holati ozbekcha tushuntiriladi`() {
        assertEquals("O'chirilgan", RecordingFormat.statusLabel(RecordingFormat.STATUS_EXPIRED))
    }

    @Test
    fun `expired backend konstantasi bilan bir xil`() {
        assertEquals("expired", RecordingFormat.STATUS_EXPIRED)
    }
}
