package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.Poll

/**
 * So'rovnoma formasi va tugma qoidalari (№7).
 *
 * Eng muhimi — «E'lon qilish» qachon KO'RINISHI: `mentor_only` so'rovnomada
 * server 400 beradi va ko'rsatilgan tugma ustozni boshi berk ko'chaga olib
 * borardi ("bosdim, xato chiqdi, nima qilay?").
 */
class PollFormTest {

    private fun poll(
        id: String = "p1",
        isActive: Boolean = true,
        visibility: String = PollForm.MENTOR_ONLY,
        publishedAt: String? = null,
        createdAt: String? = null,
    ) = Poll(
        id = id,
        question = "Savol?",
        options = listOf("A", "B"),
        isActive = isActive,
        resultsVisibility = visibility,
        resultsPublishedAt = publishedAt,
        createdAt = createdAt,
    )

    // ─── Forma ────────────────────────────────────────────────────────────────

    @Test
    fun `bosh savol rad etiladi`() {
        assertNotNull(PollForm.questionError(""))
        assertNotNull(PollForm.questionError("   "))
        assertNull(PollForm.questionError("2 + 2 nechchi?"))
    }

    @Test
    fun `juda uzun savol rad etiladi`() {
        assertNull(PollForm.questionError("a".repeat(PollForm.MAX_QUESTION)))
        assertNotNull(PollForm.questionError("a".repeat(PollForm.MAX_QUESTION + 1)))
    }

    @Test
    fun `bosh variantlar hisobga olinmaydi`() {
        // Formada doim bo'sh maydon turadi — u XATO emas, shunchaki bo'sh.
        assertEquals(listOf("A", "B"), PollForm.cleanOptions(listOf("A", " B ", "", "   ")))
    }

    @Test
    fun `kamida ikkita variant kerak`() {
        assertNotNull(PollForm.optionsError(listOf("A", "")))
        assertNotNull(PollForm.optionsError(listOf("", "")))
        assertNull(PollForm.optionsError(listOf("A", "B")))
    }

    @Test
    fun `kopi bilan onta variant`() {
        assertNull(PollForm.optionsError((1..PollForm.MAX_OPTIONS).map { "V$it" }))
        assertNotNull(PollForm.optionsError((1..PollForm.MAX_OPTIONS + 1).map { "V$it" }))
    }

    @Test
    fun `takrorlangan variantlar rad etiladi`() {
        // Bir xil variantlarda ovozlar ikkiga bo'linadi va diagramma yolg'on
        // ko'rsatadi — server buni tekshirmaydi, shuning uchun biz tekshiramiz.
        assertNotNull(PollForm.optionsError(listOf("Ha", "ha")))
        assertNotNull(PollForm.optionsError(listOf("A", "B", " A ")))
    }

    @Test
    fun `juda uzun variant rad etiladi`() {
        assertNotNull(PollForm.optionsError(listOf("a".repeat(PollForm.MAX_OPTION + 1), "B")))
    }

    @Test
    fun `sorov yaratishda bosh variantlar tushmaydi`() {
        val req = PollForm.request(" 2+2? ", listOf("3", "", " 4 "), publicResults = true)
        assertEquals("2+2?", req.question)
        assertEquals(listOf("3", "4"), req.options)
        assertEquals(PollForm.PUBLIC, req.resultsVisibility)
    }

    @Test
    fun `default rejim yopiq tomonda`() {
        // Yopiq tomon ATAYLAB: rejim yaratilgandan keyin o'zgarmaydi, ya'ni
        // tasodifan "hammaga" qilib yuborilgan so'rovnomani qaytarib bo'lmaydi.
        val req = PollForm.request("Savol?", listOf("A", "B"), publicResults = false)
        assertEquals(PollForm.MENTOR_ONLY, req.resultsVisibility)
        assertEquals("mentor_only", PollForm.MENTOR_ONLY)
        assertEquals("public", PollForm.PUBLIC)
    }

    @Test
    fun `canSubmit ikkala tekshiruvni ham qamraydi`() {
        assertFalse(PollForm.canSubmit("", listOf("A", "B")))
        assertFalse(PollForm.canSubmit("Savol?", listOf("A", "")))
        assertTrue(PollForm.canSubmit("Savol?", listOf("A", "B")))
    }

    // ─── E'lon qilish qoidasi ────────────────────────────────────────────────

    @Test
    fun `mentor_only sorovnomada elon qilib bolmaydi`() {
        // Server 400 beradi — tugma umuman ko'rinmasligi kerak.
        assertFalse(PollForm.canPublish(poll(visibility = PollForm.MENTOR_ONLY)))
    }

    @Test
    fun `public sorovnomada elon qilish mumkin`() {
        assertTrue(PollForm.canPublish(poll(visibility = PollForm.PUBLIC)))
    }

    @Test
    fun `allaqachon elon qilingan sorovnomada tugma yoq`() {
        val published = poll(visibility = PollForm.PUBLIC, publishedAt = "2026-07-30T10:00:00Z")
        assertFalse(PollForm.canPublish(published))
        assertTrue(PollForm.isPublished(published))
    }

    @Test
    fun `yopilgan sorovnomani ham elon qilsa boladi`() {
        // Yopish ≠ e'lon qilish: ovoz berish to'xtaydi, natija esa hali yopiq.
        // Aynan shu ketma-ketlik odatiy: "yopdim → endi natijani ko'rsataman".
        assertTrue(PollForm.canPublish(poll(isActive = false, visibility = PollForm.PUBLIC)))
    }

    @Test
    fun `faqat ochiq sorovnomani yopish mumkin`() {
        assertTrue(PollForm.canClose(poll(isActive = true)))
        assertFalse(PollForm.canClose(poll(isActive = false)))
    }

    @Test
    fun `korinuvchanlik izohi uch holatni farqlaydi`() {
        val mentorOnly = PollForm.visibilityHint(poll(visibility = PollForm.MENTOR_ONLY))
        val notYet = PollForm.visibilityHint(poll(visibility = PollForm.PUBLIC))
        val done = PollForm.visibilityHint(
            poll(visibility = PollForm.PUBLIC, publishedAt = "2026-07-30T10:00:00Z"),
        )
        assertEquals(3, setOf(mentorOnly, notYet, done).size)
    }

    // ─── Natija ko'rinishi ───────────────────────────────────────────────────

    @Test
    fun `foiz hisoblanadi`() {
        assertEquals(50, PollForm.percent(5, 10))
        assertEquals(100, PollForm.percent(7, 7))
        assertEquals(33, PollForm.percent(1, 3))
    }

    @Test
    fun `ovoz bolmasa foiz nol`() {
        // 0/0 → NaN bo'lardi va diagramma to'liq to'lib ko'rinardi.
        assertEquals(0, PollForm.percent(0, 0))
        assertEquals(0, PollForm.percent(0, 10))
    }

    @Test
    fun `royxatda ochiq sorovnomalar tepada`() {
        val list = listOf(
            poll("eski-yopiq", isActive = false, createdAt = "2026-07-30T12:00:00Z"),
            poll("ochiq-1", isActive = true, createdAt = "2026-07-30T10:00:00Z"),
            poll("ochiq-2", isActive = true, createdAt = "2026-07-30T11:00:00Z"),
        )
        assertEquals(
            listOf("ochiq-2", "ochiq-1", "eski-yopiq"),
            PollForm.sortForDisplay(list).map { it.id },
        )
    }
}

/**
 * Reaksiya to'plami — SERVER bilan shartnoma.
 *
 * `allowedReactions` (backend) bilan farq qilsa ustoz bosgan emoji jimgina
 * 400 olardi: ekranda hech nima ko'rinmasdi va sabab ham aytilmasdi.
 */
class ReactionsTest {

    @Test
    fun `toplam backend royxati bilan bir xil`() {
        // Manba: `backend/internal/usecase/roomstate/roomstate.go: allowedReactions`.
        assertEquals(setOf("👍", "👏", "❤️", "😂", "😮", "🎉", "✋"), Reactions.ALLOWED.toSet())
        assertEquals("dublikat bo'lmasin", Reactions.ALLOWED.size, Reactions.ALLOWED.toSet().size)
    }

    @Test
    fun `yurak variatsiya selektori bilan`() {
        // ❤️ = U+2764 U+FE0F. Selektorsiz "❤" BOSHQA satr va server 400 berardi —
        // bu aynan ko'zga tashlanmaydigan xato turi.
        val heart = Reactions.ALLOWED.first { it.startsWith("❤") }
        assertEquals("❤️", heart)
    }

    @Test
    fun `royxatdan tashqari emoji yuborilmaydi`() {
        // Klient darajasidagi qorovul: server baribir tekshiradi, lekin
        // ma'nosiz so'rov umuman ketmasligi kerak.
        assertFalse(Reactions.isAllowed("💩"))
        assertFalse(Reactions.isAllowed(""))
        assertFalse(Reactions.isAllowed("salom"))
        assertTrue(Reactions.isAllowed("👍"))
    }
}
