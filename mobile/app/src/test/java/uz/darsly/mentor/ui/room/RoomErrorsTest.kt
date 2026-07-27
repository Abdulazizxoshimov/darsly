package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.IOException

/**
 * 🟡A — xonani boshlashdagi kutilmagan istisnolar.
 *
 * Eng muhim kafolat: **matn hech qachon bo'sh bo'lmaydi**. UI "Qayta urinish"
 * tugmasini `error != null` bo'lganda ko'rsatadi; bo'sh xabar = abadiy spinner.
 */
class RoomErrorsTest {

    /**
     * Haqiqiy sinf API 31 da paydo bo'lgan, `minSdk 26` da import qilib bo'lmaydi —
     * [RoomErrors] uni **nomi** bo'yicha taniydi, shuning uchun testda o'sha nomli
     * soxta sinf yetarli (aynan shu mexanizm sinaladi).
     */
    private class ForegroundServiceStartNotAllowedException(msg: String) : IllegalStateException(msg)

    @Test
    fun `fon rejimi xatosi tushunarli maslahat beradi`() {
        val msg = RoomErrors.startFailure(
            ForegroundServiceStartNotAllowedException("startForegroundService() not allowed"),
        )
        assertTrue("ekran ochiq turishi kerakligi aytilsin", msg.contains("ekranda ochiq"))
        assertTrue("inglizcha SDK matni chiqmasin", !msg.contains("startForegroundService"))
    }

    @Test
    fun `oralgan istisno ham taniladi`() {
        // LiveKit/Android ba'zan asl xatoni o'rab qaytaradi.
        val wrapped = RuntimeException(
            "servis ishga tushmadi",
            ForegroundServiceStartNotAllowedException("denied"),
        )
        assertTrue(RoomErrors.startFailure(wrapped).contains("ekranda ochiq"))
    }

    @Test
    fun `ruxsat xatosi alohida matn oladi`() {
        val msg = RoomErrors.startFailure(SecurityException("permission denied"))
        assertTrue(msg.contains("Ruxsat"))
    }

    @Test
    fun `notanish istisno ham bosh bolmagan matn beradi`() {
        listOf(
            IOException("boom"),
            IllegalStateException(),
            RuntimeException(null as String?),
        ).forEach { t ->
            val msg = RoomErrors.startFailure(t)
            assertTrue("bo'sh xabar = abadiy spinner: $t", msg.isNotBlank())
        }
    }

    @Test
    fun `ozini sabab qilib korsatgan istisno siklga tushmaydi`() {
        // `initCause(this)` mumkin emas, lekin ikki istisno bir-birini ko'rsatishi mumkin.
        val a = RuntimeException("a")
        val b = RuntimeException("b", a)
        a.initCause(b)
        assertEquals("Darsni boshlab bo'lmadi — qaytadan urinib ko'ring", RoomErrors.startFailure(b))
    }
}
