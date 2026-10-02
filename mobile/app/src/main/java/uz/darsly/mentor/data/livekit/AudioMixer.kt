package uz.darsly.mentor.data.livekit

import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.util.concurrent.ConcurrentHashMap

/**
 * Ko'p manbali audio aralashtirgich (ustoz mikrofoni + o'quvchilar ovozi).
 *
 * Har LiveKit audio treki [feed] orqali xom PCM beradi; ular MONO 16-bit'ga
 * keltiriladi va har trek uchun alohida navbatga (FIFO) yig'iladi. Audio
 * encoder [poll] chaqirganda barcha navbatlar SUM qilinib (klipping bilan)
 * bitta oqim qaytariladi.
 *
 * Soddalik/ishonchlilik uchun MONO: nutq uchun yetarli va aralashtirish
 * kanalma-kanal muvozanatsiz bo'lmaydi. Manba 48 kHz deb qabul qilinadi
 * (WebRTC default); kanal ko'p bo'lsa o'rtacha bilan mono'ga tushiriladi.
 */
class AudioMixer(
    private val sampleRate: Int,
    private val channels: Int,
    private val nowNanos: () -> Long = System::nanoTime,
) {

    private val fifos = ConcurrentHashMap<Any, ByteFifo>()
    @Volatile private var running = false

    // Real-vaqt JIMLIK pacing (Zoom kabi uzluksiz audio trek). Ovoz manbai
    // umuman bo'lmasa ham (mikrofon o'chiq + ishtirokchi yo'q) jimlik chiqadi →
    // audio encoder DOIM initsializatsiya bo'ladi → muxer DOIM boshlanadi va
    // trek VALID qoladi. Aks holda: audio encoder bo'sh qolib `onOutputFormatChanged`
    // yonmaydi → `audioTrackIndex=-1` → muxer boshlanmaydi (VIDEO ham yo'qoladi),
    // yoki trek buzuq `[0][0][0][0]` bo'lib pleyerda ochilmaydi.
    // `emittedSamples` faqat `poll` (bitta encoder oqimi) da o'zgaradi — poyga yo'q.
    private var startNanos = 0L
    @Volatile private var emittedSamples = 0L

    fun start(@Suppress("UNUSED_PARAMETER") onReady: (Boolean) -> Unit) {
        startNanos = nowNanos()
        emittedSamples = 0L
        running = true
    }

    fun stop() {
        running = false
        fifos.clear()
    }

    /** LiveKit trekidan xom PCM — mono 16-bit'ga keltirib navbatga qo'yadi. */
    fun feed(track: Any, data: ByteBuffer, bitsPerSample: Int, srcRate: Int, srcChannels: Int, frames: Int) {
        if (!running || bitsPerSample != 16) return
        val fifo = fifos.getOrPut(track) { ByteFifo() }
        val src = data.duplicate().order(ByteOrder.LITTLE_ENDIAN)
        val shorts = src.asShortBuffer()
        // Mono 16-bit chiqish (frames ta sample).
        val out = ByteBuffer.allocate(frames * 2).order(ByteOrder.LITTLE_ENDIAN)
        var i = 0
        while (i < frames * srcChannels && shorts.remaining() >= srcChannels) {
            val sample: Int = if (srcChannels >= 2) {
                var sum = 0
                for (c in 0 until srcChannels) sum += shorts.get()
                sum / srcChannels
            } else {
                shorts.get().toInt()
            }
            out.putShort(sample.toShort())
            i += srcChannels
        }
        out.flip()
        fifo.write(out.array(), 0, out.limit())
    }

    /**
     * Aralashtirilgan mono 16-bit PCM'ni qaytaradi (eng ko'pi `maxBytes`).
     *
     * Real ovoz bo'lsa — treklar SUM qilinib qaytadi. Ovoz bo'lmasa — real-vaqt
     * bilan pace qilingan JIMLIK qaytadi (audio trek uzluksiz, muxer doim ishlaydi).
     * Real-vaqtga yetib bo'lgan bo'lsa `null` (chaqiruvchi qisqa kutadi).
     */
    fun poll(maxBytes: Int): ByteArray? {
        if (!running) return null
        val maxSamples = maxBytes / 2
        if (maxSamples <= 0) return null

        // Mavjud real audio — jim treklar 0 beradi, ovozli trek to'xtab qolmasin.
        var avail = 0
        fifos.values.forEach { avail = maxOf(avail, it.available() / 2) }
        val n = minOf(maxSamples, avail)
        if (n > 0) {
            val acc = IntArray(n)
            fifos.values.forEach { fifo ->
                val chunk = fifo.read(n * 2) ?: return@forEach
                val sb = ByteBuffer.wrap(chunk).order(ByteOrder.LITTLE_ENDIAN).asShortBuffer()
                var i = 0
                while (i < n && sb.hasRemaining()) {
                    acc[i] += sb.get().toInt()
                    i++
                }
            }
            val outBuf = ByteBuffer.allocate(n * 2).order(ByteOrder.LITTLE_ENDIAN)
            for (i in 0 until n) {
                outBuf.putShort(acc[i].coerceIn(-32768, 32767).toShort())
            }
            emittedSamples += n
            return outBuf.array()
        }

        // Real ovoz yo'q — audio trekni uzluksiz saqlash uchun real-vaqt bilan
        // pace qilingan jimlik. `emittedSamples` real-vaqt kvotasidan oshib
        // ketmasin (aks holda encoder jimlik bilan to'lib PTS oldinга ketardi).
        return pollSilence(maxSamples)
    }

    private fun pollSilence(maxSamples: Int): ByteArray? {
        val elapsedNanos = nowNanos() - startNanos
        if (elapsedNanos <= 0L) return null
        val expected = elapsedNanos * sampleRate / 1_000_000_000L
        val behind = expected - emittedSamples
        if (behind <= 0L) return null // real-vaqtga yetdik — chaqiruvchi kutsin
        val n = minOf(maxSamples.toLong(), behind).toInt()
        if (n <= 0) return null
        emittedSamples += n
        return ByteArray(n * 2) // nol baytlar = 16-bit mono jimlik
    }

    /**
     * Oddiy sinxron bayt-navbati (FIFO). Yozuvchi (sink) va o'quvchi (encoder)
     * alohida oqimlarda — shuning uchun `synchronized`. Cheksiz o'smasligi
     * uchun eski ma'lumot tashlanadi (real vaqtda navbat kichik qoladi).
     */
    private class ByteFifo {
        private val chunks = ArrayDeque<ByteArray>()
        private var headOffset = 0
        private var total = 0

        @Synchronized fun write(b: ByteArray, off: Int, len: Int) {
            if (len <= 0) return
            chunks.addLast(b.copyOfRange(off, off + len))
            total += len
            // ~2 soniyalik buferdan ortig'ini tashlaymiz (48k*2 bayt ≈ 96KB/s mono).
            while (total > MAX_BYTES && chunks.isNotEmpty()) {
                val head = chunks.first()
                val remain = head.size - headOffset
                if (remain <= total - MAX_BYTES) {
                    chunks.removeFirst(); total -= remain; headOffset = 0
                } else break
            }
        }

        @Synchronized fun available(): Int = total

        @Synchronized fun read(len: Int): ByteArray? {
            if (total == 0) return null
            val want = minOf(len, total)
            val out = ByteArray(want)
            var written = 0
            while (written < want && chunks.isNotEmpty()) {
                val head = chunks.first()
                val remain = head.size - headOffset
                val take = minOf(remain, want - written)
                System.arraycopy(head, headOffset, out, written, take)
                written += take
                headOffset += take
                total -= take
                if (headOffset >= head.size) { chunks.removeFirst(); headOffset = 0 }
            }
            return out
        }

        companion object {
            private const val MAX_BYTES = 48000 * 2 * 2 // ~2s mono @48k
        }
    }
}
