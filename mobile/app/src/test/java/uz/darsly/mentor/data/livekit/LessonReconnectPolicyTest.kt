package uz.darsly.mentor.data.livekit

import io.livekit.android.room.network.ReconnectContext
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Qayta ulanish siyosati — C-11 (tarmoq almashuvi) uchun eng muhim qaror.
 *
 * DIQQAT: bu testlar siyosat MANTIG'INI qotiradi, qurilmadagi haqiqiy
 * Wi-Fi↔LTE almashuvini EMAS — u faqat real qurilmada tekshiriladi.
 */
class LessonReconnectPolicyTest {

    private val policy = LessonReconnectPolicy()

    private fun ctx(retry: Int, elapsedSec: Int = 0) =
        ReconnectContext(retryCount = retry, elapsedTime = elapsedSec.seconds)

    @Test
    fun `birinchi urinish deyarli darhol boladi`() {
        // Tarmoq almashuvida yangi interfeys odatda TAYYOR — kutish sof yo'qotish.
        val first = policy.getNextRetryDelay(ctx(0))
        assertNotNull(first)
        assertTrue("birinchi urinish 500ms dan tez bo'lishi kerak", first!! <= 200.milliseconds)
    }

    @Test
    fun `dastlabki uch urinish bir soniya ichida sigadi`() {
        val total = (0..2).sumOf { policy.getNextRetryDelay(ctx(it))!!.inWholeMilliseconds }
        assertTrue("uch urinish ~1.7s ichida bo'lishi kerak, edi: $total ms", total <= 1_800)
    }

    @Test
    fun `keyingi urinishlar sekinlashadi`() {
        // Uch tez urinish yordam bermasa — muammo interfeys emas, KANAL.
        // U yerda tez urinish faqat zarar keltiradi.
        val early = policy.getNextRetryDelay(ctx(1))!!
        val late = policy.getNextRetryDelay(ctx(6))!!
        assertTrue("kechki urinish uzoqroq kutishi kerak", late > early)
    }

    @Test
    fun `urinishlar tugagach null qaytadi`() {
        val beyond = LessonReconnectPolicy.RETRY_DELAYS.size
        assertNull(policy.getNextRetryDelay(ctx(beyond)))
    }

    @Test
    fun `umumiy vaqt tugagach urinish toxtaydi`() {
        // Cheksiz "qayta ulanmoqda" ekrani — yolg'on umid. Vaqt tugasa rost
        // xabar berish kerak (ustoz qayta kirishi mumkin).
        assertNull(policy.getNextRetryDelay(ctx(retry = 0, elapsedSec = 61)))
    }

    @Test
    fun `umumiy oyna ichida urinish davom etadi`() {
        assertNotNull(policy.getNextRetryDelay(ctx(retry = 1, elapsedSec = 30)))
    }

    @Test
    fun `maksimal oyna 60 soniya`() {
        assertEquals(60.seconds, LessonReconnectPolicy.MAX_RECONNECT)
    }
}
