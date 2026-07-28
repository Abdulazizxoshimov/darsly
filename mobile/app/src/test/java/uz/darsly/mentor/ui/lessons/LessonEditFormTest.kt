package uz.darsly.mentor.ui.lessons

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.Lesson

/**
 * Darsni tahrirlash — `PATCH` so'rovining **diff** mantiqi.
 *
 * Bu yerdagi eng muhim mezon: **tegilmagan maydon yuborilmasin**. `PATCH`
 * semantikasida yuborilgan maydon ustiga yoziladi, ya'ni "hammasini yuborish"
 * ustoz formani ochib turganda web'dan kiritilgan o'zgarishni bekor qilardi
 * (poyga) va parolni tasodifan almashtirardi.
 */
class LessonEditFormTest {

    private fun lesson(
        title: String = "Algebra",
        description: String? = null,
        scheduledAt: String? = "2026-07-27T09:30:00Z",
        durationMin: Int = 60,
        hasPasscode: Boolean = false,
        waitingRoom: Boolean = false,
        recording: Boolean = false,
    ) = Lesson(
        id = "l1",
        title = title,
        description = description,
        scheduledAt = scheduledAt,
        durationMin = durationMin,
        hasPasscode = hasPasscode,
        isWaitingRoomEnabled = waitingRoom,
        isRecordingEnabled = recording,
    )

    // ─── fromLesson ───────────────────────────────────────────────────────────

    @Test
    fun `fromLesson formani darsning holati bilan toldiradi`() {
        val input = LessonForm.fromLesson(
            lesson(title = "Geometriya", description = "8-sinf", durationMin = 45, waitingRoom = true),
        )
        assertEquals("Geometriya", input.title)
        assertEquals("8-sinf", input.description)
        assertEquals("45", input.duration)
        assertTrue(input.waitingRoom)
    }

    @Test
    fun `mavjud darsning yozuv sozlamasi formaga tushadi`() {
        assertFalse(LessonForm.fromLesson(lesson(recording = false)).recording)
        assertTrue(LessonForm.fromLesson(lesson(recording = true)).recording)
    }

    @Test
    fun `parol maydoni bosh keladi chunki server hashni qaytarmaydi`() {
        // `entity.Lesson.PasscodeHash` — `json:"-"`. Formaga qo'yadigan qiymat
        // umuman yo'q; ekran buni `has_passcode` bayrog'i bilan tushuntiradi.
        val input = LessonForm.fromLesson(lesson(hasPasscode = true))
        assertEquals("", input.passcode)
    }

    @Test
    fun `vaqti belgilanmagan dars ham formaga tushadi`() {
        val input = LessonForm.fromLesson(lesson(scheduledAt = null))
        assertNull(input.scheduledAtMillis)
    }

    // ─── toUpdateRequest: diff ────────────────────────────────────────────────

    @Test
    fun `ozgarish bolmasa null qaytadi`() {
        val original = lesson()
        val req = LessonForm.toUpdateRequest(original, LessonForm.fromLesson(original))
        // Tarmoqqa umuman chiqilmaydi — bo'sh PATCH `updated_at` ni bekorga surardi.
        assertNull(req)
    }

    @Test
    fun `faqat ozgargan maydon yuboriladi`() {
        val original = lesson(title = "Algebra", durationMin = 60)
        val input = LessonForm.fromLesson(original).copy(title = "Algebra 2")
        val req = LessonForm.toUpdateRequest(original, input)!!

        assertEquals("Algebra 2", req.title)
        // ⭐ Asosiy mezon: qolgan hamma narsa `null` = "tegilmadi".
        assertNull(req.durationMin)
        assertNull(req.description)
        assertNull(req.scheduledAt)
        assertNull(req.isWaitingRoomEnabled)
        assertNull(req.isRecordingEnabled)
        assertFalse(req.removePasscode)
    }

    @Test
    fun `tavsifni tozalash bosh satr yuboradi null emas`() {
        // Go tomonida `null` va maydonning yo'qligi bir xil (`*string` = nil =
        // "tegilmadi"). Tavsifni haqiqatan o'chirish uchun `""` kerak.
        val original = lesson(description = "eski tavsif")
        val input = LessonForm.fromLesson(original).copy(description = "  ")
        val req = LessonForm.toUpdateRequest(original, input)!!
        assertEquals("", req.description)
    }

    @Test
    fun `davomiylik ozgarsa son sifatida yuboriladi`() {
        val original = lesson(durationMin = 60)
        val input = LessonForm.fromLesson(original).copy(duration = "90")
        assertEquals(90, LessonForm.toUpdateRequest(original, input)!!.durationMin)
    }

    @Test
    fun `bir xil vaqt qayta yuborilmaydi`() {
        val original = lesson(scheduledAt = "2026-07-27T09:30:00Z")
        // Forma vaqtni millisekundda saqlaydi; qayta RFC3339 ga o'girilganda
        // AYNAN o'sha lahza chiqishi kerak, aks holda har saqlashda "o'zgardi"
        // deb hisoblanardi va tugma hech qachon o'chmasdi.
        val req = LessonForm.toUpdateRequest(original, LessonForm.fromLesson(original))
        assertNull(req)
    }

    @Test
    fun `vaqt ozgarsa RFC3339 formatida yuboriladi`() {
        val original = lesson(scheduledAt = "2026-07-27T09:30:00Z")
        val input = LessonForm.fromLesson(original)
            .copy(scheduledAtMillis = 1785150000000L) // 2026-07-27T11:00:00Z
        assertEquals("2026-07-27T11:00:00Z", LessonForm.toUpdateRequest(original, input)!!.scheduledAt)
    }

    @Test
    fun `vaqtni null qilish sorov yasamaydi chunki backend uni qollamaydi`() {
        // Ma'lum cheklov (`LessonForm.toUpdateRequest` izohi): backend
        // `ScheduledAt` ni faqat `!= nil` bo'lganda qo'llaydi, ya'ni "o'chir"
        // signali yo'q. Bu test cheklovni QOTIRADI: kimdir kelajakda vaqtni
        // tozalash tugmasini qo'shsa, u jimgina ishlamasligi shu yerda ko'rinadi.
        val original = lesson(scheduledAt = "2026-07-27T09:30:00Z")
        val input = LessonForm.fromLesson(original).copy(scheduledAtMillis = null)
        assertNull(LessonForm.toUpdateRequest(original, input))
    }

    // ─── toUpdateRequest: parolning uch holati ────────────────────────────────

    @Test
    fun `parol tegilmasa yuborilmaydi`() {
        val original = lesson(hasPasscode = true)
        val input = LessonForm.fromLesson(original).copy(title = "Yangi nom")
        val req = LessonForm.toUpdateRequest(original, input)!!
        assertNull(req.passcode)
        assertFalse(req.removePasscode)
    }

    @Test
    fun `yangi parol yuboriladi`() {
        val original = lesson(hasPasscode = true)
        val input = LessonForm.fromLesson(original).copy(passcode = "1234")
        val req = LessonForm.toUpdateRequest(original, input)!!
        assertEquals("1234", req.passcode)
        assertFalse(req.removePasscode)
    }

    @Test
    fun `parolni olib tashlash yangi parolni bekor qiladi`() {
        // Backend'da `RemovePasscode` ustunlik qiladi (`lesson.go:119-128`), ya'ni
        // ikkalasini birga yuborish yozilgan parolni JIMGINA yo'qotardi. Shuning
        // uchun klient ham ikkalasini birga yubormaydi.
        val original = lesson(hasPasscode = true)
        val input = LessonForm.fromLesson(original).copy(passcode = "1234")
        val req = LessonForm.toUpdateRequest(original, input, removePasscode = true)!!
        assertTrue(req.removePasscode)
        assertNull(req.passcode)
    }

    @Test
    fun `parolsiz darsda olib tashlash sorov yasamaydi`() {
        // Bo'sh amal: server holati o'zgarmaydi, ya'ni PATCH yuborish ma'nosiz.
        val original = lesson(hasPasscode = false)
        val input = LessonForm.fromLesson(original)
        assertNull(LessonForm.toUpdateRequest(original, input, removePasscode = true))
    }

    // ─── Almashtirgichlar ─────────────────────────────────────────────────────

    @Test
    fun `kutish xonasi ochirilsa false yuboriladi`() {
        // `false` — haqiqiy qiymat, `null` bilan chalkashmasligi kerak: `takeIf`
        // noto'g'ri yozilsa `false` tushib qolib, o'chirish ishlamasdi.
        val original = lesson(waitingRoom = true)
        val input = LessonForm.fromLesson(original).copy(waitingRoom = false)
        assertEquals(false, LessonForm.toUpdateRequest(original, input)!!.isWaitingRoomEnabled)
    }

    @Test
    fun `yozib olishni ochirish PATCH da yuboriladi`() {
        // Yozuv default yoniq, lekin ustoz uni o'chira oladi — va bu o'zgarish
        // serverga YETIB BORISHI kerak, aks holda tugma jimgina ishlamasdi.
        val original = lesson(recording = true)
        val input = LessonForm.fromLesson(original).copy(recording = false)
        assertEquals(false, LessonForm.toUpdateRequest(original, input)!!.isRecordingEnabled)
    }

    @Test
    fun `tegilmagan yozib olish sozlamasi yuborilmaydi`() {
        val original = lesson(recording = true)
        val input = LessonForm.fromLesson(original).copy(title = "Yangi nom")
        assertNull(LessonForm.toUpdateRequest(original, input)!!.isRecordingEnabled)
    }
}
