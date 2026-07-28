package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Data-channel shakli — SERVER bilan shartnoma
 * (`backend/internal/usecase/roomstate/roomstate.go`, `chat.go`).
 * U o'zgarsa signallar jimgina yo'qoladi, shuning uchun shakl shu yerda qotirilgan.
 */
class RoomDataParserTest {

    private val parser = RoomDataParser()

    @Test
    fun `qol kotarish xabarini oqiydi`() {
        val s = parser.parse("""{"kind":"hand","identity":"u1","name":"Ali","raised":true,"at":1730000000000}""")
        assertEquals(RoomSignal.Hand("u1", "Ali", raised = true, at = 1730000000000L), s)
    }

    @Test
    fun `qol tushirish xabarini oqiydi`() {
        val s = parser.parse("""{"kind":"hand","identity":"u1","name":"","raised":false,"at":1}""")
        assertEquals(RoomSignal.Hand("u1", "u1", raised = false, at = 1L), s)
    }

    @Test
    fun `ismsiz qol identity bilan almashtiriladi`() {
        // Server qo'l tushirishda ismni bo'sh yuboradi — UI'da bo'sh qator chiqmasin.
        val s = parser.parse("""{"kind":"hand","identity":"u9","raised":true,"at":5}""") as RoomSignal.Hand
        assertEquals("u9", s.name)
    }

    @Test
    fun `lower_all tanib olinadi`() {
        assertEquals(RoomSignal.LowerAll, parser.parse("""{"kind":"hand","act":"lower_all"}"""))
    }

    @Test
    fun `identitysiz qol xabari etiborsiz`() {
        assertNull(parser.parse("""{"kind":"hand","raised":true}"""))
    }

    @Test
    fun `reaksiyani oqiydi`() {
        val s = parser.parse("""{"kind":"reaction","emoji":"👍","name":"Ali","identity":"u1"}""")
        assertEquals(RoomSignal.Reaction("👍", "Ali", "u1"), s)
    }

    @Test
    fun `chat xabari kindsiz shaklda tanib olinadi`() {
        val raw = """{"id":"m1","lesson_id":"l1","sender_identity":"u1","sender_name":"Ali",""" +
            """"body":"Salom","created_at":"2026-07-27T10:00:00Z"}"""
        val s = parser.parse(raw) as RoomSignal.Chat
        assertEquals("Ali", s.senderName)
        assertEquals("Salom", s.body)
        assertNull(s.toIdentity)
    }

    @Test
    fun `shaxsiy chat xabari to_identity bilan keladi`() {
        val raw = """{"id":"m2","sender_identity":"u1","sender_name":"Ali","body":"Maxfiy","to_identity":"mentor1"}"""
        val s = parser.parse(raw) as RoomSignal.Chat
        assertEquals("mentor1", s.toIdentity)
    }

    @Test
    fun `notanish tur va buzuq JSON null qaytaradi`() {
        // Oldinga moslik: yangi signal turi qo'shilsa eski ilova YIQILMASLIGI kerak.
        assertNull(parser.parse("""{"kind":"wb","act":"stroke"}"""))
        assertNull(parser.parse("""{"kind":"poll","action":"open"}"""))
        assertNull(parser.parse("buzuq json"))
        assertNull(parser.parse(""))
        assertNull(parser.parse("""{}"""))
    }

    @Test
    fun `baytlardan oqish UTF8 emojini buzmaydi`() {
        val bytes = """{"kind":"reaction","emoji":"🎉","name":"Ali","identity":"u1"}"""
            .toByteArray(Charsets.UTF_8)
        val s = parser.parse(bytes) as RoomSignal.Reaction
        assertEquals("🎉", s.emoji)
    }
}

class HandQueueTest {

    private fun raise(id: String, name: String, at: Long) =
        RoomSignal.Hand(id, name, raised = true, at = at)

    @Test
    fun `navbat kotarilgan vaqt boyicha tartiblanadi`() {
        var q = HandQueue.apply(emptyList(), raise("u2", "Vali", 200))
        q = HandQueue.apply(q, raise("u1", "Ali", 100))
        q = HandQueue.apply(q, raise("u3", "Guli", 300))
        // Kelish tartibi u2→u1→u3, lekin NAVBAT `at` bo'yicha: u1→u2→u3.
        assertEquals(listOf("u1", "u2", "u3"), q.map { it.identity })
    }

    @Test
    fun `takroriy kotarish orinni buzmaydi`() {
        var q = HandQueue.apply(emptyList(), raise("u1", "Ali", 100))
        q = HandQueue.apply(q, raise("u2", "Vali", 200))
        val same = HandQueue.apply(q, raise("u1", "Ali", 999))
        assertTrue("o'zgarmagan ro'yxat aynan qaytishi kerak", same === q)
        assertEquals(listOf("u1", "u2"), same.map { it.identity })
    }

    @Test
    fun `ism ozgarsa yangilanadi lekin orin saqlanadi`() {
        var q = HandQueue.apply(emptyList(), raise("u1", "Ali", 100))
        q = HandQueue.apply(q, raise("u2", "Vali", 200))
        q = HandQueue.apply(q, raise("u1", "Ali Valiyev", 999))
        assertEquals(listOf("u1", "u2"), q.map { it.identity })
        assertEquals("Ali Valiyev", q.first().name)
        assertEquals("eski `at` saqlanishi kerak", 100L, q.first().at)
    }

    @Test
    fun `qol tushirilsa royxatdan chiqadi`() {
        var q = HandQueue.apply(emptyList(), raise("u1", "Ali", 100))
        q = HandQueue.apply(q, RoomSignal.Hand("u1", "Ali", raised = false, at = 0))
        assertTrue(q.isEmpty())
    }

    @Test
    fun `yoq qolni tushirish royxatni ozgartirmaydi`() {
        val q = HandQueue.apply(emptyList(), raise("u1", "Ali", 100))
        val same = HandQueue.apply(q, RoomSignal.Hand("yoq", "X", raised = false, at = 0))
        assertTrue(same === q)
    }

    @Test
    fun `lower_all hammasini tozalaydi va bosh royxatga tegmaydi`() {
        var q = HandQueue.apply(emptyList(), raise("u1", "Ali", 100))
        q = HandQueue.apply(q, RoomSignal.LowerAll)
        assertTrue(q.isEmpty())
        assertTrue(HandQueue.apply(q, RoomSignal.LowerAll) === q)
    }

    @Test
    fun `replaceAll serverdan kelgan holatni tartiblaydi`() {
        val q = HandQueue.replaceAll(
            listOf(RaisedHand("u2", "Vali", 200), RaisedHand("u1", "Ali", 100)),
        )
        assertEquals(listOf("u1", "u2"), q.map { it.identity })
    }
}

class ReactionFeedTest {

    @Test
    fun `eng yangi reaksiya boshida turadi`() {
        var f = ReactionFeed.add(emptyList(), RoomReaction(1, "👍", "Ali"))
        f = ReactionFeed.add(f, RoomReaction(2, "🎉", "Vali"))
        assertEquals("🎉", f.first().emoji)
    }

    @Test
    fun `royxat MAX dan oshmaydi`() {
        var f = emptyList<RoomReaction>()
        repeat(20) { f = ReactionFeed.add(f, RoomReaction(it.toLong(), "👍", "A$it")) }
        assertEquals(ReactionFeed.MAX, f.size)
        // Eng yangisi saqlanadi, eskisi tushib qoladi.
        assertEquals("A19", f.first().name)
    }
}
