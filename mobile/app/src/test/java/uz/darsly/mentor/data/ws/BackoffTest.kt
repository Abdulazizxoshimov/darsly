package uz.darsly.mentor.data.ws

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Qayta ulanish kechikishi — web klienti bilan bir xil bo'lishi shart
 * (`frontend/src/lib/ws.js`: `min(15000, 1000 * 2^retries)`).
 */
class BackoffTest {

    @Test
    fun `ketma-ketlik web bilan bir xil`() {
        assertEquals(1_000L, Backoff.delayMs(0))
        assertEquals(2_000L, Backoff.delayMs(1))
        assertEquals(4_000L, Backoff.delayMs(2))
        assertEquals(8_000L, Backoff.delayMs(3))
        assertEquals(15_000L, Backoff.delayMs(4)) // 16s → chegara
    }

    @Test
    fun `chegaradan oshmaydi`() {
        // Uzoq dars davomida internet bir necha marta uzilsa ham kechikish
        // 15 soniyadan oshmasligi kerak — aks holda dars qayta ulanmay qoladi.
        listOf(5, 10, 20, 31, 32, 63, 1000, Int.MAX_VALUE).forEach { attempt ->
            assertEquals("urinish=$attempt", Backoff.MAX_MS, Backoff.delayMs(attempt))
        }
    }

    @Test
    fun `manfiy yoki nol urinish bazaviy qiymat beradi`() {
        assertEquals(Backoff.BASE_MS, Backoff.delayMs(0))
        assertEquals(Backoff.BASE_MS, Backoff.delayMs(-5))
    }

    @Test
    fun `hech qachon manfiy yoki nol qaytmaydi`() {
        // Manfiy kechikish cheksiz siklga olib kelardi (Long to'lib ketishi).
        (0..64).forEach { assertTrue("urinish=$it", Backoff.delayMs(it) > 0) }
    }
}
