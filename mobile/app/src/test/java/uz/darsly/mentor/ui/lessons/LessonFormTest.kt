package uz.darsly.mentor.ui.lessons

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.ZoneId

/**
 * M7 — dars yaratish formasi (mezon B-7: validatsiya xatolari tushunarli).
 *
 * Chegaralar `backend/internal/entity/lesson.go` bilan bir xil bo'lishi shart:
 * agar klient serverdan yumshoqroq bo'lsa, ustoz inglizcha server xatosini ko'radi;
 * qattiqroq bo'lsa — haqiqatda ruxsat etilgan darsni yarata olmaydi.
 */
class LessonFormTest {

    private val tashkent: ZoneId = ZoneId.of("Asia/Tashkent")
    private val now = 1_774_000_000_000L // 2026-03-20 atrofi

    private fun input(
        title: String = "Algebra 7-sinf",
        description: String = "",
        duration: String = "60",
        passcode: String = "",
        scheduledAtMillis: Long? = null,
    ) = LessonForm.Input(
        title = title,
        description = description,
        duration = duration,
        passcode = passcode,
        scheduledAtMillis = scheduledAtMillis,
    )

    @Test
    fun `togri forma xatosiz`() {
        val errors = LessonForm.validate(input(), now)
        assertTrue(errors.isValid)
        assertNull(errors.title)
        assertNull(errors.scheduleWarning)
    }

    @Test
    fun `qisqa sarlavha rad etiladi`() {
        val errors = LessonForm.validate(input(title = "A"), now)
        assertFalse(errors.isValid)
        assertTrue("xato o'zbekcha bo'lsin", errors.title!!.contains("Sarlavha"))
    }

    @Test
    fun `bosh sarlavha rad etiladi`() {
        assertFalse(LessonForm.validate(input(title = "   "), now).isValid)
    }

    @Test
    fun `sarlavha chegaralari backend bilan bir xil`() {
        assertTrue(LessonForm.validate(input(title = "ab"), now).isValid)
        assertTrue(LessonForm.validate(input(title = "a".repeat(255)), now).isValid)
        assertFalse(LessonForm.validate(input(title = "a".repeat(256)), now).isValid)
    }

    @Test
    fun `tavsif 2000 belgidan oshmaydi`() {
        assertTrue(LessonForm.validate(input(description = "x".repeat(2000)), now).isValid)
        assertFalse(LessonForm.validate(input(description = "x".repeat(2001)), now).isValid)
    }

    @Test
    fun `davomiylik chegaralari`() {
        assertFalse(LessonForm.validate(input(duration = "4"), now).isValid)
        assertTrue(LessonForm.validate(input(duration = "5"), now).isValid)
        assertTrue(LessonForm.validate(input(duration = "1440"), now).isValid)
        assertFalse(LessonForm.validate(input(duration = "1441"), now).isValid)
        // Bo'sh = server default'i (60), xato emas.
        assertTrue(LessonForm.validate(input(duration = ""), now).isValid)
    }

    @Test
    fun `raqam bolmagan davomiylik rad etiladi`() {
        val errors = LessonForm.validate(input(duration = "bir soat"), now)
        assertFalse(errors.isValid)
        assertNotNull(errors.duration)
    }

    @Test
    fun `parol chegaralari`() {
        assertTrue("bo'sh parol = parolsiz dars", LessonForm.validate(input(passcode = ""), now).isValid)
        assertFalse(LessonForm.validate(input(passcode = "123"), now).isValid)
        assertTrue(LessonForm.validate(input(passcode = "1234"), now).isValid)
        assertTrue(LessonForm.validate(input(passcode = "x".repeat(20)), now).isValid)
        assertFalse(LessonForm.validate(input(passcode = "x".repeat(21)), now).isValid)
    }

    @Test
    fun `otgan vaqt ogohlantiradi lekin bloklamaydi`() {
        val errors = LessonForm.validate(input(scheduledAtMillis = now - 3_600_000), now)
        assertNotNull(errors.scheduleWarning)
        assertTrue("ogohlantirish yaratishni to'smasligi kerak", errors.isValid)
    }

    @Test
    fun `kelajakdagi vaqt ogohlantirmaydi`() {
        assertNull(LessonForm.validate(input(scheduledAtMillis = now + 3_600_000), now).scheduleWarning)
    }

    // ── So'rovga aylantirish ──────────────────────────────────────────────────

    @Test
    fun `sorovda bosh matnlar null boladi`() {
        val req = LessonForm.toRequest(input(description = "  ", passcode = "  ", duration = ""))
        assertNull("bo'sh tavsif JSON'ga tushmasligi kerak", req.description)
        assertNull(req.passcode)
        assertNull(req.scheduledAt)
        assertEquals(60, req.durationMin)
        assertEquals("Algebra 7-sinf", req.title)
    }

    @Test
    fun `sorov maydonlari toliq kochiriladi`() {
        val req = LessonForm.toRequest(
            LessonForm.Input(
                title = "  Geometriya  ",
                description = "Uchburchaklar",
                duration = "90",
                passcode = "1234",
                waitingRoom = true,
                recording = true,
                scheduledAtMillis = 1_774_000_000_000L,
            ),
        )
        assertEquals("Geometriya", req.title)
        assertEquals("Uchburchaklar", req.description)
        assertEquals(90, req.durationMin)
        assertEquals("1234", req.passcode)
        assertTrue(req.isWaitingRoomEnabled)
        assertTrue(req.isRecordingEnabled)
        assertNotNull(req.scheduledAt)
    }

    @Test
    fun `rfc3339 shakli Go time RFC3339 bilan mos`() {
        // Go `time.RFC3339` soniyalarni TALAB qiladi va faqat UTC `Z` yoki ofset qabul qiladi.
        assertEquals("2026-07-27T09:30:00Z", LessonForm.rfc3339Utc(1785144600000L))
        assertTrue(LessonForm.rfc3339Utc(1785144600000L).matches(Regex("""\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z""")))
    }

    @Test
    fun `sana va vaqt qurilma mintaqasida qoshiladi`() {
        // DatePicker 2026-07-27 ni UTC yarim tunida qaytaradi.
        val dateUtcMidnight = 1785110400000L // 2026-07-27T00:00:00Z
        val millis = LessonForm.combineDateAndTime(dateUtcMidnight, hour = 14, minute = 30, zone = tashkent)
        // Toshkentda 14:30 = UTC 09:30 (sana surilmaydi!)
        assertEquals("2026-07-27T09:30:00Z", LessonForm.rfc3339Utc(millis))
    }

    @Test
    fun `manfiy ofsetli mintaqada sana surilib ketmaydi`() {
        // 🟡F: bu test AYNAN mutatsiyani o'ldirish uchun. Toshkent (UTC+5) da
        // `.atZone(UTC)` ni `.atZone(zone)` ga almashtirsangiz natija BIR XIL chiqadi
        // (UTC yarim tun +5 da o'sha kunning 05:00 i) — ya'ni Toshkentdagi test
        // xatoni ushlamaydi. Manfiy ofsetda esa UTC yarim tun **oldingi kun**ga
        // tushadi va noto'g'ri implementatsiya darsni bir kun oldinga suradi.
        val newYork = ZoneId.of("America/New_York") // iyulda UTC-4
        val dateUtcMidnight = 1785110400000L // 2026-07-27T00:00:00Z
        val millis = LessonForm.combineDateAndTime(dateUtcMidnight, hour = 14, minute = 30, zone = newYork)
        assertEquals("2026-07-27T18:30:00Z", LessonForm.rfc3339Utc(millis))
        assertEquals(27, java.time.Instant.ofEpochMilli(millis).atZone(newYork).dayOfMonth)
    }

    @Test
    fun `erta soat tanlansa sana orqaga surilmaydi`() {
        // Eng xavfli chekka holat: UTC+5 da 02:00 → UTC'da OLDINGI kun 21:00.
        // Sana tanlagichdan kelgan kun saqlanishi kerak, aks holda ustoz
        // 27-iyulga qo'ygan dars serverda 26-iyulda ko'rinardi.
        val dateUtcMidnight = 1785110400000L // 2026-07-27T00:00:00Z
        val millis = LessonForm.combineDateAndTime(dateUtcMidnight, hour = 2, minute = 0, zone = tashkent)
        assertEquals("2026-07-26T21:00:00Z", LessonForm.rfc3339Utc(millis))
        // …lekin qurilma mintaqasida bu haqiqatan 27-iyul 02:00.
        assertEquals(
            "27",
            java.time.Instant.ofEpochMilli(millis).atZone(tashkent).dayOfMonth.toString(),
        )
    }

    @Test
    fun `yangi darsda yozib olish DEFAULT O'CHIQ`() {
        // ⭐ Mahsulot qarori (2026-08-15): yozuv default O'CHIQ. Har dars
        // avtomatik yozilishi shart emas — ustoz kerak bo'lganda o'zi yoqadi.
        assertFalse(LessonForm.Input().recording)
        assertFalse(LessonForm.toRequest(input()).isRecordingEnabled)
    }

    @Test
    fun `ustoz yozib olishni ochira oladi`() {
        // Majburiy EMAS: o'chirilgan bo'lsa so'rovda ham `false` ketadi va
        // backend `EnsureRecording` uni hurmat qiladi.
        val req = LessonForm.toRequest(input().copy(recording = false))
        assertFalse(req.isRecordingEnabled)
    }

    // ─── Ovoz sozlamalari (№11) ───────────────────────────────────────────────

    @Test
    fun `ovoz sozlamalari default qiymatlari server bilan bir xil`() {
        // Backend berilmagan maydonga `true` qo'yadi (api-contract.md:173).
        // Klient default'i undan farq qilsa, forma ko'rsatgan holat bilan
        // haqiqiy dars holati ajralib ketardi.
        assertTrue(LessonForm.MUTE_ON_ENTRY_DEFAULT)
        assertTrue(LessonForm.ALLOW_SELF_UNMUTE_DEFAULT)
        assertTrue(LessonForm.Input().muteOnEntry)
        assertTrue(LessonForm.Input().allowSelfUnmute)
    }

    @Test
    fun `ovoz sozlamalari sorovga tushadi`() {
        val req = LessonForm.toRequest(input())
        assertTrue(req.muteOnEntry)
        assertTrue(req.allowSelfUnmute)
    }

    @Test
    fun `maruza rejimi sorovga false bolib tushadi`() {
        // "O'quvchi o'zi ocholmasin" — `false` haqiqiy qiymat, tushib qolmasligi
        // kerak (aks holda toggle jimgina ishlamasdi).
        val req = LessonForm.toRequest(
            input().copy(muteOnEntry = true, allowSelfUnmute = false),
        )
        assertTrue(req.muteOnEntry)
        assertFalse(req.allowSelfUnmute)
    }

    @Test
    fun `kirganda mikrofon yoniq bolsin tanlovi sorovga tushadi`() {
        val req = LessonForm.toRequest(input().copy(muteOnEntry = false))
        assertFalse(req.muteOnEntry)
    }
}
