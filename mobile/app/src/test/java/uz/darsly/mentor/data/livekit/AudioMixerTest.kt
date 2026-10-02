package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.nio.ByteBuffer
import java.nio.ByteOrder

/**
 * Audio aralashtirgich. Ikki xil haqiqat: (1) ovoz manbai bo'lmasa ham audio
 * trek UZLUKSIZ bo'lishi kerak (aks holda muxer boshlanmay butun yozuv yo'qoladi
 * yoki trek buzuq `[0][0][0][0]` chiqadi); (2) ovoz bo'lsa treklar to'g'ri
 * qo'shilib klipping bilan chegaralanishi kerak.
 */
class AudioMixerTest {

    private val RATE = 48000
    private var now = 0L
    private fun mixer() = AudioMixer(RATE, 1) { now }

    private fun pcm(vararg samples: Int): ByteBuffer {
        val bb = ByteBuffer.allocate(samples.size * 2).order(ByteOrder.LITTLE_ENDIAN)
        for (s in samples) bb.putShort(s.toShort())
        bb.flip()
        return bb
    }

    private fun shortsOf(b: ByteArray): ShortArray {
        val sb = ByteBuffer.wrap(b).order(ByteOrder.LITTLE_ENDIAN).asShortBuffer()
        return ShortArray(sb.remaining()).also { sb.get(it) }
    }

    @Test fun noSourceStillEmitsSilence_soMuxerStarts() {
        // BUG tuzatildi: manba UMUMAN yo'q (mikrofon o'chiq + ishtirokchi yo'q).
        // Avval `poll` doim null qaytarardi → audio encoder init bo'lmasdi.
        now = 0L
        val m = mixer()
        m.start {}
        now = 1_000_000_000L // 1s o'tdi
        val buf = m.poll(4096) // 2048 sample so'raladi
        assertTrue("jimlik chiqishi kerak", buf != null && buf.isNotEmpty())
        assertTrue("nol = jimlik", shortsOf(buf!!).all { it.toInt() == 0 })
    }

    @Test fun silenceIsPacedToRealTime_neverRunsAhead() {
        now = 0L
        val m = mixer()
        m.start {}
        // Vaqt o'tmagan (elapsed=0) → jimlik yo'q.
        assertNull("boshda vaqt yo'q", m.poll(4096))
        // 10ms o'tdi → ~480 sample kvota.
        now = 10_000_000L
        val first = m.poll(4096)!! // min(2048, 480) = 480 sample
        assertEquals(480, shortsOf(first).size)
        // Darhol yana (vaqt o'zgармаган) → kvota tugadi → null (oldinга ketmaydi).
        assertNull("real-vaqtdan oshmaydi", m.poll(4096))
    }

    @Test fun singleSourcePlaysThrough() {
        now = 0L
        val m = mixer()
        m.start {}
        m.feed("t1", pcm(1000, 1000, 1000), 16, RATE, 1, 3)
        val out = shortsOf(m.poll(1000)!!) // avail=3, maxSamples=500 → n=3
        assertEquals(3, out.size)
        assertTrue(out.all { it.toInt() == 1000 })
    }

    @Test fun twoSourcesSum() {
        now = 0L
        val m = mixer()
        m.start {}
        m.feed("t1", pcm(1000, 1000), 16, RATE, 1, 2)
        m.feed("t2", pcm(2000, 2000), 16, RATE, 1, 2)
        val out = shortsOf(m.poll(1000)!!)
        assertEquals(2, out.size)
        assertTrue("SUM = 3000", out.all { it.toInt() == 3000 })
    }

    @Test fun sumClipsInsteadOfWrapping() {
        now = 0L
        val m = mixer()
        m.start {}
        m.feed("t1", pcm(30000, -30000), 16, RATE, 1, 2)
        m.feed("t2", pcm(30000, -30000), 16, RATE, 1, 2)
        val out = shortsOf(m.poll(1000)!!)
        assertEquals(32767, out[0].toInt())   // +60000 → clip +32767
        assertEquals(-32768, out[1].toInt())  // -60000 → clip -32768
    }

    @Test fun stereoSourceDownmixesToMono() {
        now = 0L
        val m = mixer()
        m.start {}
        // 2 kanal: (L=1000,R=3000) → mono 2000; (L=0,R=0) → 0
        m.feed("t1", pcm(1000, 3000, 0, 0), 16, RATE, 2, 2)
        val out = shortsOf(m.poll(1000)!!)
        assertEquals(2, out.size)
        assertEquals(2000, out[0].toInt())
        assertEquals(0, out[1].toInt())
    }

    @Test fun stoppedMixerReturnsNull() {
        now = 1_000_000_000L
        val m = mixer()
        // start chaqirilmagan → running=false.
        assertNull(m.poll(4096))
    }
}
