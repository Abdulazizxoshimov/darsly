package uz.darsly.mentor.data.livekit

import io.livekit.android.audio.AudioBufferCallback
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Test
import java.nio.ByteBuffer

/**
 * ⭐ C-6 ning yuragi — mikrofon namunalarini o'chirish.
 *
 * ENG QIMMAT MEZON: ustoz **mute** bosganda uning ovozi qurilmadan chiqmasligi kerak.
 * Bu maxfiylik masalasi: agar nollash ishlamasa, "mute" holatidagi ustozning gaplari
 * o'quvchilarga ekran-audio treki orqali yetib boradi.
 *
 * Test haqiqiy `ByteBuffer` ustida ishlaydi va delegat sifatida soxta callback oladi
 * (haqiqiy `ScreenAudioCapturer` — final, Android'ga bog'langan; interfeys esa emas).
 */
class ScreenAudioMixerTest {

    /** Ekran ovozini modellaydi: buferdagi har baytga `add` qo'shadi (SDK mikser'i kabi). */
    private class FakeCapturer(private val add: Byte) : AudioBufferCallback {
        var lastBytesRead = -1
            private set
        var calls = 0
            private set

        override fun onBuffer(
            buffer: ByteBuffer,
            audioFormat: Int,
            channelCount: Int,
            sampleRate: Int,
            bytesRead: Int,
            captureTimeNs: Long,
        ): Long {
            calls++
            lastBytesRead = bytesRead
            // Ishlab chiqarish kodidagi kabi `duplicate().clear()` — absolyut kirish
            // buferning LIMIT'iga bo'ysunadi, shuning uchun to'liq sig'imga ochamiz.
            val view = buffer.duplicate()
            view.clear()
            for (i in 0 until minOf(bytesRead, view.capacity())) {
                view.put(i, (view.get(i) + add).toByte())
            }
            return captureTimeNs
        }
    }

    private fun buffer(vararg values: Int): ByteBuffer =
        ByteBuffer.allocateDirect(values.size).apply {
            values.forEachIndexed { i, v -> put(i, v.toByte()) }
        }

    private fun ByteBuffer.dump(size: Int) = ByteArray(size) { get(it) }

    @Test
    fun `mute holatida mikrofon namunalari ochiriladi va faqat ekran ovozi qoladi`() {
        // Mikrofon 50 (ustoz gapiryapti), ekran ovozi +5.
        val buf = buffer(50, 50, 50, 50)
        val capturer = FakeCapturer(add = 5)
        val mixer = ScreenAudioMixer(capturer) { true }

        mixer.onBuffer(buf, 2, 1, 48_000, 4, 123L)

        assertArrayEquals(
            "ustoz ovozi (50) chiqmasligi kerak — faqat ekran ovozi (5)",
            byteArrayOf(5, 5, 5, 5),
            buf.dump(4),
        )
    }

    @Test
    fun `mikrofon yoniqda ikkisi ham qoladi`() {
        val buf = buffer(50, 50, 50, 50)
        val mixer = ScreenAudioMixer(FakeCapturer(add = 5)) { false }

        mixer.onBuffer(buf, 2, 1, 48_000, 4, 123L)

        assertArrayEquals(
            "mute emas — ustoz ovozi + ekran ovozi",
            byteArrayOf(55, 55, 55, 55),
            buf.dump(4),
        )
    }

    @Test
    fun `bufer position va limit ozgarmaydi`() {
        // SDK callback'dan keyin AYNI buferni o'z position/limit'i bilan ishlatadi.
        // `duplicate()` ishlatilmasa, bu yerda pozitsiya surilib ketardi va
        // kodlanadigan audio buzilardi.
        val buf = buffer(9, 9, 9, 9)
        buf.position(1)
        buf.limit(3)

        ScreenAudioMixer(FakeCapturer(add = 1)) { true }.onBuffer(buf, 2, 1, 48_000, 4, 1L)

        assertEquals(1, buf.position())
        assertEquals(3, buf.limit())
    }

    @Test
    fun `faqat oqilgan baytlar tozalanadi`() {
        // `bytesRead` buferdan kichik bo'lishi mumkin — qolgan qismga tegilmaydi.
        val buf = buffer(7, 7, 7, 7)
        ScreenAudioMixer(FakeCapturer(add = 0)) { true }.onBuffer(buf, 2, 1, 48_000, 2, 1L)
        assertArrayEquals(byteArrayOf(0, 0, 7, 7), buf.dump(4))
    }

    @Test
    fun `delegat har doim chaqiriladi va captureTime qaytariladi`() {
        val capturer = FakeCapturer(add = 1)
        val mixer = ScreenAudioMixer(capturer) { true }
        val ts = mixer.onBuffer(buffer(0, 0), 2, 1, 48_000, 2, 777L)
        assertEquals(1, capturer.calls)
        assertEquals(2, capturer.lastBytesRead)
        assertEquals("captureTimeNs SDK ga qaytishi kerak", 777L, ts)
    }

    @Test
    fun `nol bytesRead bilan yiqilmaydi`() {
        val buf = buffer(3, 3)
        ScreenAudioMixer(FakeCapturer(add = 0)) { true }.onBuffer(buf, 2, 1, 48_000, 0, 1L)
        assertArrayEquals("hech narsa o'qilmagan — tegilmaydi", byteArrayOf(3, 3), buf.dump(2))
    }

    @Test
    fun `katta buferda ham ishlaydi va qayta ishlatiladi`() {
        // Ichki nol-bufer o'sishi kerak (48 kHz stereo 20 ms ≈ 3840 bayt).
        val size = 3840
        val buf = ByteBuffer.allocateDirect(size).apply {
            for (i in 0 until size) put(i, 99.toByte())
        }
        val mixer = ScreenAudioMixer(FakeCapturer(add = 0)) { true }
        mixer.onBuffer(buf, 2, 2, 48_000, size, 1L)
        assertEquals(0, buf.get(0).toInt())
        assertEquals(0, buf.get(size - 1).toInt())
        // Ikkinchi chaqiruv — bufer qayta ishlatiladi, natija bir xil.
        for (i in 0 until size) buf.put(i, 42.toByte())
        mixer.onBuffer(buf, 2, 2, 48_000, size, 2L)
        assertEquals(0, buf.get(1000).toInt())
    }
}
