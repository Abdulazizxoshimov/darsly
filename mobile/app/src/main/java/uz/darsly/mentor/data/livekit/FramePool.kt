package uz.darsly.mentor.data.livekit

import java.util.concurrent.ArrayBlockingQueue

/**
 * Qayta ishlatiladigan kadr buferlari (M8).
 *
 * ## Nega kerak
 * Har kadr uchun yangi 1.38 MB (`1280×720` I420) massiv ajratish 24 fps da
 * ~30 MB/s axlat degani — WebRTC ushlash oqimida GC pauzalari jonli
 * ulashishning o'zini titratardi. Endi buferlar bir marta ajratiladi va
 * encoder ularni qaytarib beradi.
 *
 * ## Qoida
 * Bo'sh bufer qolmagan bo'lsa kadr TASHLANADI ([acquire] `null`): bu encoder
 * navbatga yetib ulgurmayotganini bildiradi va jonli yozuvda "eski kadr"
 * "yangi ajratish" dan yaxshi. Sig'im navbat chegarasidan katta
 * ([RecorderPipeline.MAX_PENDING_FRAMES] + ishlov berilayotgan kadrlar).
 */
class FramePool(private val frameBytes: Int, val capacity: Int = RecorderPipeline.MAX_PENDING_FRAMES + 2) {
    private val free = ArrayBlockingQueue<ByteArray>(capacity).apply {
        repeat(capacity) { add(ByteArray(frameBytes)) }
    }

    /** Bo'sh bufer yoki `null` (hammasi band — kadr tashlanadi). */
    fun acquire(): ByteArray? = free.poll()

    /** Buferni qaytaradi. Begona (boshqa o'lchamli) massiv qabul qilinmaydi. */
    fun release(buf: ByteArray) {
        if (buf.size == frameBytes) free.offer(buf)
    }

    val available: Int get() = free.size
}
