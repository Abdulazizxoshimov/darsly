package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.WaitingRoomRequest
import uz.darsly.mentor.data.ws.RealtimeEvent

/**
 * Kutish xonasi ro'yxati (M27).
 *
 * Bu ro'yxat **ikki manbadan** to'ladi (REST surati + WebSocket) va ular
 * bir-birini quvib yetadi. Shu sababli bu yerdagi testlar mahsulot xatti-
 * harakatini himoya qiladi, sintaksisni emas:
 *  · bitta o'quvchi ikki marta ko'rinmasin;
 *  · boshqa darsning o'quvchisi bu darsga kiritilmasin;
 *  · eng uzoq kutgan birinchi bo'lsin.
 */
class WaitingRoomStateTest {

    private val lessonId = "lesson-1"

    private fun req(
        id: String,
        lesson: String = lessonId,
        name: String = "O'quvchi $id",
        status: String = "pending",
        createdAt: String? = null,
    ) = WaitingRoomRequest(
        id = id,
        lessonId = lesson,
        requesterName = name,
        status = status,
        createdAt = createdAt,
    )

    // ─── WS hodisasini modelga aylantirish ────────────────────────────────────

    @Test
    fun `WS hodisasi request_id maydonidan ID oladi`() {
        // REST `id`, WS esa `request_id` yuboradi (`waitingroom.go:68`).
        // Bu farq AYNAN shu yerda yopiladi — boshqa joyda takrorlanmaydi.
        val event = RealtimeEvent.WaitingRoomRequest(
            requestId = "r1",
            lessonId = lessonId,
            requesterName = "Ali",
            createdAt = "2026-07-27T09:00:00Z",
        )
        val model = WaitingRoomState.toRequest(event)
        assertEquals("r1", model.id)
        assertEquals("Ali", model.requesterName)
        assertEquals("pending", model.status)
    }

    // ─── Qo'shish ─────────────────────────────────────────────────────────────

    @Test
    fun `yangi sorov navbat oxiriga qoshiladi`() {
        val existing = listOf(req("a"))
        val result = WaitingRoomState.add(existing, req("b"), lessonId)
        assertEquals(listOf("a", "b"), result.map { it.id })
    }

    @Test
    fun `dublikat ikki marta chiqmaydi`() {
        // ⭐ Backend mentor ulanganda kutayotganlarni QAYTA yuboradi
        // (`DeliverPendingSnapshot`), ya'ni dublikat kutilmaydigan holat emas —
        // u deyarli kafolatlangan.
        val existing = listOf(req("a"), req("b"))
        val result = WaitingRoomState.add(existing, req("a"), lessonId)
        assertEquals(2, result.size)
    }

    @Test
    fun `dublikat royxatdagi joyini saqlaydi`() {
        // Qatorlar sakrab ketsa ustoz "Kiritish"ni NOTO'G'RI qatorda bosishi
        // mumkin — boshqa o'quvchini darsga qo'shib yuborardi.
        val existing = listOf(req("a"), req("b"), req("c"))
        val result = WaitingRoomState.add(existing, req("b", name = "Yangi nom"), lessonId)
        assertEquals(listOf("a", "b", "c"), result.map { it.id })
        assertEquals("Yangi nom", result[1].requesterName)
    }

    @Test
    fun `boshqa darsning sorovi qabul qilinmaydi`() {
        // ⭐ WS kanali USTOZGA bog'langan, xonaga emas. Filtrsiz ustoz A darsida
        // turib B darsining o'quvchisini kiritib yuborardi.
        val existing = listOf(req("a"))
        val result = WaitingRoomState.add(existing, req("b", lesson = "boshqa-dars"), lessonId)
        assertEquals(listOf("a"), result.map { it.id })
    }

    @Test
    fun `dars IDsi bosh bolsa sorov qabul qilinadi`() {
        // Eski server dars ID'sini yubormasligi mumkin. Jimgina tashlab
        // yuborishdan ko'ra ko'rsatgan yaxshiroq: ustoz o'zi qaror qiladi.
        val result = WaitingRoomState.add(emptyList(), req("a", lesson = ""), lessonId)
        assertEquals(1, result.size)
    }

    // ─── Olib tashlash ────────────────────────────────────────────────────────

    @Test
    fun `qaror qabul qilingan sorov olib tashlanadi`() {
        val existing = listOf(req("a"), req("b"))
        assertEquals(listOf("b"), WaitingRoomState.remove(existing, "a").map { it.id })
    }

    @Test
    fun `mavjud bolmagan IDni olib tashlash zararsiz`() {
        val existing = listOf(req("a"))
        assertEquals(1, WaitingRoomState.remove(existing, "yoq").size)
    }

    // ─── Birlashtirish ────────────────────────────────────────────────────────

    @Test
    fun `surat va jonli sorovlar birlashadi`() {
        val snapshot = listOf(req("a"), req("b"))
        val live = listOf(req("c"))
        val merged = WaitingRoomState.merge(snapshot, live)
        assertEquals(listOf("a", "b", "c"), merged.map { it.id })
    }

    @Test
    fun `surat olinayotganda kelgan sorov yoqolmaydi`() {
        // O'quvchi ustoz ekranni ochgan AYNAN o'sha lahzada kirsa: WS xabari
        // REST javobidan oldin kelishi mumkin. Jonli so'rov tashlansa u
        // ro'yxatda umuman ko'rinmasdi.
        val snapshot = listOf(req("eski"))
        val live = listOf(req("yangi"))
        assertTrue(WaitingRoomState.merge(snapshot, live).any { it.id == "yangi" })
    }

    @Test
    fun `hal qilingan sorovlar korsatilmaydi`() {
        // Ularning tugmalari 409 berardi va ustoz "ishlamayapti" deb o'ylardi.
        val snapshot = listOf(req("a"), req("b", status = "admitted"), req("c", status = "rejected"))
        assertEquals(listOf("a"), WaitingRoomState.merge(snapshot, emptyList()).map { it.id })
    }

    @Test
    fun `dublikat birlashtirishda ham ikkilanmaydi`() {
        val snapshot = listOf(req("a"))
        val live = listOf(req("a"), req("b"))
        assertEquals(listOf("a", "b"), WaitingRoomState.merge(snapshot, live).map { it.id })
    }

    // ─── Tartib ───────────────────────────────────────────────────────────────

    @Test
    fun `eng uzoq kutgan birinchi`() {
        val list = listOf(
            req("yangi", createdAt = "2026-07-27T09:05:00Z"),
            req("eski", createdAt = "2026-07-27T09:00:00Z"),
        )
        assertEquals(listOf("eski", "yangi"), WaitingRoomState.sortForDisplay(list).map { it.id })
    }

    @Test
    fun `vaqti nomalum sorov oxirida qoladi`() {
        // Nol qo'yilsa u ro'yxat BOSHIGA chiqardi va ustoz uni eng uzoq kutgan
        // o'quvchi deb o'ylardi — noto'g'ri navbat.
        val list = listOf(
            req("vaqtsiz", createdAt = null),
            req("vaqtli", createdAt = "2026-07-27T09:00:00Z"),
        )
        assertEquals(listOf("vaqtli", "vaqtsiz"), WaitingRoomState.sortForDisplay(list).map { it.id })
    }

    // ─── Matnlar ──────────────────────────────────────────────────────────────

    @Test
    fun `kutish vaqti oqiladigan matnga aylanadi`() {
        val now = 1_785_153_600_000L // 2026-07-27T12:00:00Z
        fun label(offsetMillis: Long) = WaitingRoomState.waitingLabel(
            req("a", createdAt = java.time.Instant.ofEpochMilli(now - offsetMillis).toString()),
            now,
        )
        assertEquals("Hozirgina keldi", label(20_000))
        assertEquals("3 daqiqa kutmoqda", label(3 * 60_000))
        assertEquals("2 soatdan ortiq kutmoqda", label(2 * 3_600_000))
    }

    @Test
    fun `vaqti nomalum bolsa taxmin qilinmaydi`() {
        assertNull(WaitingRoomState.waitingLabel(req("a", createdAt = null)))
    }

    @Test
    fun `sarlavha birlik va koplikni ajratadi`() {
        assertEquals("1 o'quvchi kutmoqda", WaitingRoomState.title(1))
        assertEquals("3 o'quvchi kutmoqda", WaitingRoomState.title(3))
    }
}
