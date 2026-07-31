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
    fun `fayl ilovasi boglangan chat xabari oqiladi`() {
        val raw = """{"id":"m3","sender_identity":"u1","sender_name":"Ali","body":"Uy ishi",""" +
            """"file":{"name":"uy_ishi.pdf","size":184320,"mime":"application/pdf",""" +
            """"url":"https://minio/x?sig=1","expires_in_s":3600}}"""
        val s = parser.parse(raw) as RoomSignal.Chat
        assertEquals("uy_ishi.pdf", s.file?.name)
        assertEquals(184320L, s.file?.size)
        assertEquals("https://minio/x?sig=1", s.file?.url)
    }

    @Test
    fun `izohsiz fayl xabari YOQOLMAYDI`() {
        // ⭐ REGRESSIYA: `body` avval MAJBURIY edi va izohsiz yuborilgan fayl
        // jimgina tashlanardi — fayl serverga tushardi, lekin ustoz ekranida
        // hech nima ko'rinmasdi (faqat tarixni qayta yuklaganda paydo bo'lardi).
        val raw = """{"id":"m4","sender_identity":"u1","sender_name":"Ali","body":"",""" +
            """"file":{"name":"rasm.png","size":10,"mime":"image/png","url":"https://x"}}"""
        val s = parser.parse(raw) as RoomSignal.Chat
        assertEquals("", s.body)
        assertEquals("rasm.png", s.file?.name)
    }

    @Test
    fun `na matn na fayl bolsa xabar etiborsiz`() {
        assertNull(parser.parse("""{"id":"m5","sender_name":"Ali"}"""))
    }

    @Test
    fun `chat_deleted hodisasi ID beradi`() {
        val raw = """{"kind":"chat_deleted","id":"m1","lesson_id":"l1","deleted_by":"mentor1"}"""
        assertEquals(RoomSignal.ChatDeleted("m1"), parser.parse(raw))
    }

    @Test
    fun `IDsiz chat_deleted etiborsiz`() {
        // ID'siz hodisa bilan qiladigan ish yo'q — noto'g'ri xabarni
        // o'chirgandan ko'ra hech narsa qilmagan yaxshi.
        assertNull(parser.parse("""{"kind":"chat_deleted","lesson_id":"l1"}"""))
    }

    @Test
    fun `poll_published natijasi bilan keladi`() {
        val raw = """{"kind":"poll_published","results":{"poll":{"id":"p1","question":"Q",""" +
            """"options":["A","B"]},"counts":[3,7],"total":10}}"""
        val s = parser.parse(raw) as RoomSignal.PollPublished
        assertEquals("p1", s.pollId)
        assertEquals(listOf(3, 7), s.counts)
        assertEquals(10, s.total)
    }

    @Test
    fun `poll_published totalsiz kelsa ovozlar yigindisi olinadi`() {
        val raw = """{"kind":"poll_published","results":{"poll":{"id":"p2"},"counts":[1,2,3]}}"""
        val s = parser.parse(raw) as RoomSignal.PollPublished
        assertEquals(6, s.total)
    }

    @Test
    fun `natijasiz poll_published etiborsiz`() {
        assertNull(parser.parse("""{"kind":"poll_published"}"""))
        assertNull(parser.parse("""{"kind":"poll_published","results":{"counts":[1]}}"""))
    }

    @Test
    fun `notanish tur va buzuq JSON null qaytaradi`() {
        // Oldinga moslik: yangi signal turi qo'shilsa eski ilova YIQILMASLIGI kerak.
        assertNull(parser.parse("""{"kind":"wb","act":"stroke"}"""))
        assertNull(parser.parse("""{"kind":"kelajakdagi_tur","action":"open"}"""))
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

    // ─── O'tkinchilik (TTL) ──────────────────────────────────────────────────
    //
    // ⭐ REGRESSIYA: reaksiya ekranda YANGISI kelguncha turardi. Ya'ni darsning
    // 5-daqiqasidagi 👍 soat oxirigacha sahna burchagida osilib turardi va
    // ustoz uni HOZIRGI reaksiya deb o'qirdi — serverda esa u umuman
    // saqlanmaydi (o'tkinchi signal).

    @Test
    fun `eskirgan reaksiya royxatdan chiqadi`() {
        val old = RoomReaction(1, "👍", "Ali", at = 1_000L)
        val feed = listOf(old)
        assertTrue(ReactionFeed.prune(feed, 1_000L + ReactionFeed.TTL_MS).isEmpty())
    }

    @Test
    fun `hali muddati otmagan reaksiya qoladi`() {
        val fresh = RoomReaction(1, "👍", "Ali", at = 1_000L)
        val left = ReactionFeed.prune(listOf(fresh), 1_000L + ReactionFeed.TTL_MS - 1)
        assertEquals(listOf(fresh), left)
    }

    @Test
    fun `yangi reaksiya qoshilganda eskilari tozalanadi`() {
        val old = RoomReaction(1, "👍", "Ali", at = 1_000L)
        val new = RoomReaction(2, "🎉", "Vali", at = 1_000L + ReactionFeed.TTL_MS + 1)
        val f = ReactionFeed.add(listOf(old), new)
        assertEquals(listOf("🎉"), f.map { it.emoji })
    }

    @Test
    fun `vaqtsiz reaksiyalar tegilmaydi`() {
        // `at = 0` — vaqt manbai bo'lmagan joydan kelgan. Ularni "cheksiz eski"
        // deb hisoblash butun ro'yxatni jimgina tozalab yuborardi.
        val timeless = listOf(RoomReaction(1, "👍", "Ali"))
        assertTrue(ReactionFeed.prune(timeless, Long.MAX_VALUE) === timeless)
    }

    @Test
    fun `ozgarish bolmasa AYNI royxat qaytadi`() {
        // Ortiqcha rekompozitsiya bo'lmasin (`HandQueue.apply` bilan bir naqsh).
        val feed = listOf(RoomReaction(1, "👍", "Ali", at = 1_000L))
        assertTrue(ReactionFeed.prune(feed, 1_500L) === feed)
        assertTrue(ReactionFeed.prune(emptyList(), 1_500L).isEmpty())
    }
}
