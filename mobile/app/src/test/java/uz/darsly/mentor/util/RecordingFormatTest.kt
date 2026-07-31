package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
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

    // ─── Amallar ──────────────────────────────────────────────────────────────

    @Test
    fun `faqat tayyor yozuvni yuklab olish mumkin`() {
        assertTrue(RecordingFormat.canDownload(rec(status = RecordingFormat.STATUS_READY)))
        assertFalse(RecordingFormat.canDownload(rec(status = RecordingFormat.STATUS_PROCESSING)))
        assertFalse(RecordingFormat.canDownload(rec(status = RecordingFormat.STATUS_RECORDING)))
        assertFalse(RecordingFormat.canDownload(rec(status = RecordingFormat.STATUS_FAILED)))
    }

    @Test
    fun `faqat faol yozuvni toxtatish mumkin`() {
        assertTrue(RecordingFormat.canStop(rec(status = RecordingFormat.STATUS_RECORDING)))
        assertFalse(RecordingFormat.canStop(rec(status = RecordingFormat.STATUS_READY)))
    }

    @Test
    fun `tayyorlanmoqda holatida sabab tushuntiriladi`() {
        // Ustoz uchun eng chalkash holat: u "To'xtatish"ni bosgan, lekin fayl
        // hali yo'q. Tushuntirish bo'lmasa "ishlamayapti" degan xulosa chiqadi.
        val hint = RecordingFormat.statusHint(RecordingFormat.STATUS_PROCESSING)
        assertTrue(hint != null && hint.isNotBlank())
        assertNull(RecordingFormat.statusHint(RecordingFormat.STATUS_READY))
    }

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

    // ─── Havola muddati ───────────────────────────────────────────────────────

    @Test
    fun `havola muddati oqiladigan matnga aylanadi`() {
        assertEquals("Havola 30 soniya amal qiladi", RecordingFormat.expiryLabel(30))
        assertEquals("Havola 15 daqiqa amal qiladi", RecordingFormat.expiryLabel(15 * 60))
        assertEquals("Havola 2 soat amal qiladi", RecordingFormat.expiryLabel(2 * 3600))
        assertNull(RecordingFormat.expiryLabel(0))
    }

    // ─── Saqlash muddati (retention · PRODUCT.md №5) ─────────────────────────
    //
    // NEGA MUHIM: yozuv 30 kundan keyin O'CHADI. Ustoz buni bilmasa uni
    // "abadiy turadi" deb o'ylab yuklab olmaydi va bir kun kelib yo'qotadi —
    // jimgina, hech qanday xabarsiz.

    /** Sana hisobini barqaror qilish uchun qat'iy zona va "hozir". */
    private val zone: java.time.ZoneId = java.time.ZoneId.of("Asia/Tashkent")
    private val now: java.time.Instant = java.time.Instant.parse("2026-07-30T09:00:00Z")

    private fun retention(iso: String?) = RecordingFormat.retentionLabel(iso, zone, now)

    @Test
    fun `muddat kunlar bilan korsatiladi`() {
        // 30-iyul (Toshkent) → 2-avgust = 3 kun.
        assertEquals("3 kundan keyin o'chadi", retention("2026-08-02T09:00:00Z"))
        assertEquals("30 kundan keyin o'chadi", retention("2026-08-29T09:00:00Z"))
    }

    @Test
    fun `bugun va ertaga alohida aytiladi`() {
        assertEquals("Bugun o'chadi", retention("2026-07-30T18:00:00Z"))
        assertEquals("Ertaga o'chadi", retention("2026-07-31T09:00:00Z"))
    }

    @Test
    fun `otgan muddat korsatilmaydi`() {
        // Fon ishchisi hali ishlab ulgurmagan bo'lishi mumkin —
        // "-1 kundan keyin o'chadi" ilova buzuq degan taassurot berardi.
        assertNull(retention("2026-07-29T09:00:00Z"))
    }

    @Test
    fun `muddatsiz yozuvda hech nima yozilmaydi`() {
        // `expired` yozuvda va `RECORDING_RETENTION_DAYS=0` da maydon kelmaydi.
        assertNull(retention(null))
        assertNull(retention(""))
        assertNull(retention("buzuq-sana"))
    }

    @Test
    fun `uch kun qolganda ogohlantiriladi`() {
        // Chegara serverdagi `recording_expiring` bildirishnomasi bilan bir xil:
        // ustoz xabarnomani ko'rib kirsa, ro'yxatda AYNI o'sha yozuvlar ajralib
        // tursin.
        assertTrue(RecordingFormat.isExpiringSoon("2026-08-02T09:00:00Z", zone, now))
        assertTrue(RecordingFormat.isExpiringSoon("2026-07-30T18:00:00Z", zone, now))
        assertFalse(RecordingFormat.isExpiringSoon("2026-08-03T09:00:00Z", zone, now))
        assertFalse(RecordingFormat.isExpiringSoon(null, zone, now))
    }

    // ─── `expired` holati ────────────────────────────────────────────────────

    @Test
    fun `muddati otgan yozuvni yuklab bolmaydi`() {
        // Server 400 `BAD_REQUEST` beradi — tugmani ko'rsatish ustozni
        // boshi berk ko'chaga olib borardi.
        assertFalse(RecordingFormat.canDownload(rec(status = RecordingFormat.STATUS_EXPIRED)))
        assertFalse(RecordingFormat.canStop(rec(status = RecordingFormat.STATUS_EXPIRED)))
    }

    @Test
    fun `expired holati ozbekcha tushuntiriladi`() {
        assertEquals("O'chirilgan", RecordingFormat.statusLabel(RecordingFormat.STATUS_EXPIRED))
        val hint = RecordingFormat.statusHint(RecordingFormat.STATUS_EXPIRED)
        assertNotNull(hint)
        // Sabab aytilishi shart: "O'chirilgan" o'zi "men o'chirdimmi?" degan
        // savol tug'diradi — javob "yo'q, muddati tugagan".
        assertTrue(hint!!, hint.contains("muddat"))
    }

    @Test
    fun `expired backend konstantasi bilan bir xil`() {
        assertEquals("expired", RecordingFormat.STATUS_EXPIRED)
    }
}
