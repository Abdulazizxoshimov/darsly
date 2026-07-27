package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/**
 * QA 🟡2 — ulanish xatosidan keyin QAYTA URINISH ishlashi kerak.
 *
 * Avvalgi qorovul `if (connecting || session != null) return` edi va `connect()`
 * xato berganda `session` null qilinmasdi → "Qayta urinish" o'lik tugmaga aylanardi.
 */
class JoinGuardTest {

    @Test
    fun `birinchi urinish ruxsat etiladi`() {
        val g = JoinGuard()
        assertTrue(g.tryBegin())
        assertEquals(JoinGuard.State.CONNECTING, g.state)
    }

    @Test
    fun `ulanish jarayonida takroriy urinish rad etiladi`() {
        val g = JoinGuard()
        assertTrue(g.tryBegin())
        assertFalse("ikkita parallel ulanish bo'lmasligi kerak", g.tryBegin())
    }

    @Test
    fun `ulangandan keyin qayta ulanmaydi`() {
        val g = JoinGuard()
        g.tryBegin()
        g.onAttached()
        assertTrue(g.isAttached)
        assertFalse("allaqachon ulangan — qayta ulanish kerak emas", g.tryBegin())
    }

    /** ★ ASOSIY TEST: xatodan keyin qayta urinish MUMKIN bo'lishi shart. */
    @Test
    fun `ulanish xatosidan keyin qayta urinish mumkin`() {
        val g = JoinGuard()
        assertTrue(g.tryBegin())

        g.onFailed() // `connect()` istisno tashladi, resurslar bo'shatildi

        assertEquals(JoinGuard.State.IDLE, g.state)
        assertTrue("xatodan keyin qayta urinish bloklanmasligi kerak", g.tryBegin())
    }

    @Test
    fun `ketma-ket bir necha xato ham blokladagi qoldirmaydi`() {
        val g = JoinGuard()
        repeat(5) {
            assertTrue("$it-urinish bloklandi", g.tryBegin())
            g.onFailed()
        }
        assertTrue(g.tryBegin())
    }

    @Test
    fun `chiqishdan keyin qayta ulanish mumkin`() {
        val g = JoinGuard()
        g.tryBegin()
        g.onAttached()
        g.onReleased()
        assertEquals(JoinGuard.State.IDLE, g.state)
        assertFalse(g.isAttached)
        assertTrue(g.tryBegin())
    }

    @Test
    fun `parallel chaqiruvlarda faqat bittasi otadi`() {
        // Ruxsat natijasi va "Qayta urinish" bosilishi bir vaqtda kelishi mumkin.
        val g = JoinGuard()
        val threads = 16
        val start = CountDownLatch(1)
        val done = CountDownLatch(threads)
        val winners = AtomicInteger()

        repeat(threads) {
            Thread {
                start.await()
                if (g.tryBegin()) winners.incrementAndGet()
                done.countDown()
            }.start()
        }
        start.countDown()
        assertTrue(done.await(10, TimeUnit.SECONDS))

        assertEquals("faqat bitta ulanish urinishi boshlanishi kerak", 1, winners.get())
    }
}
