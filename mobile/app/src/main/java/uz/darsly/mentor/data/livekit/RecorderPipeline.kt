package uz.darsly.mentor.data.livekit

import java.util.concurrent.atomic.AtomicBoolean

/**
 * [LocalRecorder] konveyerining **sof** qarorlari — MediaCodec/muxer/VideoSink'siz
 * JVM testida qulflanadigan yagona joy.
 *
 * ## Nega ajratildi
 * Konveyerdagi har bir noto'g'ri qaror JIMGINA buzuq yozuv beradi: fayl bor,
 * lekin pleyerda ochilmaydi (muxer trek qo'shilishidan oldin boshlanib ketgan),
 * yoki ovoz videodan siljib qolgan (PTS birligi noto'g'ri), yoki fayl `moov`siz
 * (kodek-konfiguratsiya buferi sample sifatida yozilgan). Bularni real kodek
 * bilan sinash imkonsiz (emulyator yo'q), shuning uchun QARORLAR kodekdan
 * ajratildi va shu yerda sinaladi. [LocalRecorder] endi faqat SDK bilan gaplashib,
 * qaror uchun shu obyektga murojaat qiladi.
 */
object RecorderPipeline {

    /**
     * Video kadr navbatining eng katta uzunligi.
     *
     * Encoder sekin bo'lsa (past telefon) navbat cheksiz o'sib xotirani yeb
     * qo'yardi. Cheklovdan oshsa eng eski kadr tashlanadi — jonli yozuvda
     * "eskirgan kadr" "xotira tugashi"dan yaxshiroq.
     */
    const val MAX_PENDING_FRAMES = 4

    /**
     * Muxer FAQAT ikkala trek (video + audio) qo'shilgach boshlanishi kerak.
     *
     * Agar muxer bitta trek bilan boshlanib, keyin `addTrack` chaqirilsa
     * MediaMuxer istisno otadi va butun yozuv yo'qoladi. Shuning uchun ikkala
     * `onOutputFormatChanged` kelgunча muxer kutadi.
     */
    fun muxerReady(videoTrackIndex: Int, audioTrackIndex: Int): Boolean =
        videoTrackIndex >= 0 && audioTrackIndex >= 0

    /** Navbat to'ldi — eng eski kadr tashlansinmi. */
    fun shouldDropOldest(queueSize: Int): Boolean = queueSize > MAX_PENDING_FRAMES

    /**
     * Sample vaqt tamg'asi — MIKROSEKUNDDA (MediaMuxer aynan shu birlikni kutadi).
     *
     * `nanoTime` (nanosekund) ni 1000 ga bo'lamiz. 1_000_000 ga bo'lish (millisekund)
     * yoki bo'lmaslik (nanosekund) A/V ni sekundlar bilan siljitardi.
     */
    fun ptsMicros(nowNanos: Long, startNanos: Long): Long = (nowNanos - startNanos) / 1000

    /** [drainAction] natijasi. */
    enum class DrainAction {
        /** Kodek-konfiguratsiya buferi — muxerga YOZILMAYDI, faqat bo'shatiladi. */
        SKIP_CONFIG,

        /** Haqiqiy kadr — muxer boshlangan va o'lcham musbat: yoziladi. */
        WRITE,

        /** Muxer hali boshlanmagan yoki bo'sh bufer — yozmasdan bo'shatiladi. */
        RELEASE_ONLY,
    }

    /**
     * Kodek chiqish buferi bilan nima qilish.
     *
     *  · `isCodecConfig` — SPS/PPS kabi konfiguratsiya buferi. Uni sample deb
     *    yozish faylni buzardi (pleyer boshini o'qiy olmaydi).
     *  · muxer hali boshlanmagan bo'lsa yozish istisno otardi — kutamiz.
     *  · o'lcham 0 bo'lsa yozadigan narsa yo'q.
     */
    fun drainAction(isCodecConfig: Boolean, size: Int, muxerStarted: Boolean): DrainAction = when {
        isCodecConfig -> DrainAction.SKIP_CONFIG
        muxerStarted && size > 0 -> DrainAction.WRITE
        else -> DrainAction.RELEASE_ONLY
    }

    /**
     * Yakunlangan yozuv fayli SERVERGA yuborishga yaroqlimi.
     *
     * Bo'sh (0 bayt) fayl — buzuq yozuv (muxer hech qachon boshlanmagan yoki
     * darhol uzilgan). Uni yuklash serverni yaroqsiz `moov`siz fayl bilan
     * to'ldirardi. [LessonSession.stopLocalRecording] va [PendingUploadResumer]
     * shu darvozadan o'tkazadi.
     */
    fun isUsableOutput(exists: Boolean, lengthBytes: Long): Boolean = exists && lengthBytes > 0
}

/**
 * Yozuvning **hayot sikli** — `start → recording → stopped` — sof, atomik.
 *
 * [LocalRecorder] ni ikki marta boshlash (ikkinchi `start` no-op, `true`
 * qaytaradi — chaqiruvchi egress fallback'iga o'tmasligi kerak) va ikki marta
 * to'xtatish (ikkinchi `stop` hech nima qilmaydi — aks holda allaqachon
 * bo'shatilgan encoder/muxer ustidan qayta ishlab crash bo'lardi) xavfsiz
 * bo'lishi shart. Bir vaqtda video oqimi (kadr keldi) va UI oqimi (to'xtat)
 * bu bayroqni o'qigani uchun [AtomicBoolean].
 */
class RecorderLifecycle {
    private val running = AtomicBoolean(false)

    /** `start()` boshlanishi kerakmi. `false` — allaqachon ishlayapti (no-op). */
    fun beginStart(): Boolean = !running.getAndSet(true)

    /** `start()` ichida istisno bo'ldi — holatni orqaga qaytaramiz. */
    fun failStart() = running.set(false)

    /** `stop()` bajarilishi kerakmi. `false` — allaqachon to'xtagan. */
    fun beginStop(): Boolean = running.getAndSet(false)

    fun isRunning(): Boolean = running.get()
}
