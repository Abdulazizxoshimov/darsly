package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
import org.junit.Test

/** M8 — kadr buferlari qayta ishlatiladi; bo'sh bufer qolmasa kadr tashlanadi. */
class FramePoolTest {

    @Test
    fun `buferlar qayta ishlatiladi`() {
        val pool = FramePool(frameBytes = 16, capacity = 2)
        val a = pool.acquire()
        val b = pool.acquire()
        assertNotNull(a); assertNotNull(b)
        assertNull("sig'im tugadi — kadr tashlanadi, yangi ajratilmaydi", pool.acquire())
        pool.release(a!!)
        assertSame("aynan qaytarilgan bufer qayta beriladi", a, pool.acquire())
    }

    @Test
    fun `begona olchamli massiv havzaga tushmaydi`() {
        val pool = FramePool(frameBytes = 16, capacity = 1)
        pool.acquire()
        pool.release(ByteArray(8))
        assertEquals(0, pool.available)
    }

    @Test
    fun `default sigim navbat chegarasidan katta`() {
        // Navbatda MAX_PENDING_FRAMES + ishlov berilayotgan kadr — hammasiga bufer yetsin.
        val pool = FramePool(frameBytes = 4)
        assertEquals(RecorderPipeline.MAX_PENDING_FRAMES + 2, pool.capacity)
    }
}
