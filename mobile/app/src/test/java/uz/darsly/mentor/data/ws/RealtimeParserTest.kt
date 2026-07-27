package uz.darsly.mentor.data.ws

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Real-time protokoli — mobil va web **bir xil** o'qishi shart (M29 / D-8).
 *
 * Xabar shakllari backend manbasidan olingan (taxmin emas):
 *  · `websocket/notify.go` → konvert `{type, room, payload, created_at}`
 *  · `websocket/waitingroom.go` → `waiting_room.request|admitted|rejected`
 *  · `usecase/waitingroom/waitingroom.go:68` → request payload maydonlari
 */
class RealtimeParserTest {

    private val parser = RealtimeParser()

    @Test
    fun `kutish xonasi sorovi oqiladi`() {
        val raw = """
            {"type":"waiting_room.request","room":"","created_at":"2026-07-26T20:00:00Z",
             "payload":{"request_id":"req-1","lesson_id":"les-1",
                        "requester_name":"Ali","created_at":"2026-07-26T20:00:00Z"}}
        """.trimIndent()

        val e = parser.parse(raw) as RealtimeEvent.WaitingRoomRequest
        assertEquals("req-1", e.requestId)
        assertEquals("les-1", e.lessonId)
        assertEquals("Ali", e.requesterName)
    }

    @Test
    fun `ismsiz sorovda ham korsatiladigan nom boladi`() {
        // Guest ism kiritmagan bo'lishi mumkin — ro'yxatda bo'sh qator chiqmasin.
        val raw = """{"type":"waiting_room.request","payload":{"request_id":"r","requester_name":""}}"""
        val e = parser.parse(raw) as RealtimeEvent.WaitingRoomRequest
        assertEquals("O'quvchi", e.requesterName)
    }

    @Test
    fun `request_id yoq sorov tashlanadi`() {
        // ID'siz so'rovni qabul qilib bo'lmaydi — uni ko'rsatish foydasiz va chalg'ituvchi.
        assertNull(parser.parse("""{"type":"waiting_room.request","payload":{"lesson_id":"l"}}"""))
    }

    @Test
    fun `qabul qilindi hodisasi token bilan oqiladi`() {
        val raw = """
            {"type":"waiting_room.admitted","payload":
              {"token":"jwt","ws_url":"wss://lk","room_name":"lesson_1","identity":"g1","role":"participant"}}
        """.trimIndent()
        val e = parser.parse(raw) as RealtimeEvent.WaitingRoomAdmitted
        assertEquals("jwt", e.token?.token)
        assertEquals("wss://lk", e.token?.wsUrl)
    }

    @Test
    fun `rad etildi hodisasi payloadsiz ham ishlaydi`() {
        assertTrue(parser.parse("""{"type":"waiting_room.rejected"}""") is RealtimeEvent.WaitingRoomRejected)
    }

    @Test
    fun `bildirishnoma body yoki message dan oqiladi`() {
        val withBody = parser.parse(
            """{"type":"notification","payload":{"id":"n1","title":"Dars","body":"5 daqiqada"}}""",
        ) as RealtimeEvent.Notification
        assertEquals("5 daqiqada", withBody.body)

        // Backend `message` nomi bilan yuborsa ham yo'qotmaymiz.
        val withMessage = parser.parse(
            """{"type":"notification","payload":{"id":"n2","title":"Dars","message":"boshlanmoqda"}}""",
        ) as RealtimeEvent.Notification
        assertEquals("boshlanmoqda", withMessage.body)
    }

    @Test
    fun `notanish tur ilovani yiqitmaydi`() {
        // ⭐ Eng muhim moslik testi: backend yangi hodisa qo'shsa, eski ilova ishlashda
        // davom etishi kerak (foydalanuvchida avtomatik yangilanish yo'q).
        val e = parser.parse("""{"type":"lesson.updated","payload":{"id":"x"}}""")
        assertEquals(RealtimeEvent.Unknown("lesson.updated"), e)
    }

    @Test
    fun `buzuq yoki bosh xabar null qaytaradi`() {
        assertNull(parser.parse("bu JSON emas"))
        assertNull(parser.parse("{}"))
        assertNull(parser.parse("""{"type":""}"""))
        assertNull(parser.parse(""))
    }

    @Test
    fun `payload buzuq bolsa ham hodisa yoqolmaydi`() {
        // Notifikatsiya tanasi kutilmagan shaklda kelsa ham "bildirishnoma keldi"
        // fakti yo'qolmasligi kerak (UI ro'yxatni yangilaydi).
        val e = parser.parse("""{"type":"notification","payload":{"id":123}}""")
        assertTrue(e is RealtimeEvent.Notification)
    }
}
