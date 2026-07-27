package uz.darsly.mentor.data.livekit

import io.livekit.android.audio.AudioBufferCallback
import java.nio.ByteBuffer

/**
 * Ekran audiosi mikser'i — mikrofon namunalarini **o'chirib** faqat ekran ovozini
 * qoldirish imkonini beradi (C-6).
 *
 * ## Nima uchun shunday qilinishi kerak (SDK cheklovi)
 * `livekit-android 2.27.0` da audio bufer callback'i **ADM darajasida** (bitta
 * `AudioBufferCallbackDispatcher`, `audioModule(...)` ga uzatiladi) — trek darajasida
 * EMAS. Telefonda audio kirishi ham bitta. Ya'ni:
 *
 *  · `LocalAudioTrack.setAudioBufferCallback` qaysi trekda chaqirilsa ham, **global**
 *    ADM buferini o'zgartiradi;
 *  · e'lon qilingan HAR BIR lokal audio trek AYNAN o'sha buferni kodlaydi.
 *
 * Shu sababli "ekran audiosi uchun mustaqil manba" olish mumkin emas. Lekin buferning
 * MAZMUNINI boshqarish mumkin: mikrofon namunalarini nolga tenglashtirib,
 * [ScreenAudioCapturer] ustiga faqat ekran ovozini mikslasa — natija "ekran ovozi,
 * ustozning ovozisiz" bo'ladi. Ustozning ovozi **qurilmadan chiqmaydi**: u kodlashdan
 * oldin o'chiriladi.
 *
 * [delegate] ataylab interfeys: haqiqiy `ScreenAudioCapturer` (final, Android'ga
 * bog'langan) o'rniga testda soxta callback beriladi va mikslash mantiqi JVM'da sinaladi.
 */
class ScreenAudioMixer(
    private val delegate: AudioBufferCallback,
    /** `true` — mikrofon namunalari o'chiriladi (ustoz "mute" bosgan holat). */
    private val silenceMic: () -> Boolean,
) : AudioBufferCallback {

    /** Nolga tenglashtirish uchun qayta ishlatiladigan bufer (har 10 ms da yasamaymiz). */
    private var zeros = ByteArray(0)


    override fun onBuffer(
        buffer: ByteBuffer,
        audioFormat: Int,
        channelCount: Int,
        sampleRate: Int,
        bytesRead: Int,
        captureTimeNs: Long,
    ): Long {
        if (silenceMic() && bytesRead > 0) {
            if (zeros.size < bytesRead) zeros = ByteArray(bytesRead)
            // `duplicate()` — asl buferning position/limit'iga TEGMAYDI (SDK keyin
            // o'sha qiymatlarga tayanadi), lekin mazmuni bir xil xotira.
            //
            // DIQQAT: `duplicate()` LIMIT'ni ham meros oladi. Agar bufer position/limit
            // bilan kelgan bo'lsa (SDK shunday qiladi), `bytesRead` limitdan katta
            // bo'lib `BufferOverflowException` beradi — audio oqimida, ya'ni darsni
            // uzadi. Shuning uchun avval `clear()` bilan to'liq sig'imga ochamiz va
            // sig'imdan oshib ketmaymiz.
            val view = buffer.duplicate()
            view.clear()
            view.put(zeros, 0, minOf(bytesRead, view.capacity()))
        }
        // Ekran ovozi shu bufer USTIGA mikslanadi (MixerAudioBufferCallback shunday ishlaydi).
        return delegate.onBuffer(buffer, audioFormat, channelCount, sampleRate, bytesRead, captureTimeNs)
    }
}
