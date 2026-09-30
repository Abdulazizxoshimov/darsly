package uz.darsly.mentor.data.livekit

import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.VideoTrack
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

/**
 * Lokal yozuvchi chegarasi — [LocalRecorder] ning JVM'da almashtiriladigan yuzi.
 *
 * `MediaCodec`/`MediaMuxer` JVM testida yaratilmaydi, shuning uchun yozuvning
 * EGALIGI ([LocalRecordingController]) kodekdan ajratilgan va soxta yozuvchi
 * bilan sinaladi. [LocalRecorder] esa faqat SDK bilan gaplashadi.
 */
interface Recorder {
    /** @return muvaffaqiyat (`false` — qo'llab-quvvatlanmadi, fayl yaroqsiz). */
    fun start(screenVideo: VideoTrack, videoBitrate: Int, localAudio: AudioTrack?, remoteAudio: List<AudioTrack>): Boolean
    fun addRemoteAudio(track: AudioTrack)
    fun removeRemoteAudio(track: AudioTrack)

    /** Yozuvni yakunlaydi (`moov` yoziladi). Idempotent; BLOKLAYDI — asosiy oqimdan chaqirilmasin. */
    fun stop()
}

/** Yozuvchi yasovchi — sessiya har yozuv uchun YANGI yozuvchi va YANGI fayl oladi. */
fun interface RecorderFactory {
    fun create(outputFile: File): Recorder
}

/**
 * Bir sessiyadagi lokal yozuvning egasi: boshlash, to'xtatish, o'quvchi
 * ovozini ulash va — eng muhimi — **bo'shatishda yakunlash**.
 *
 * ## Nega alohida (C2 · H3 · H4)
 * Avval bu holat `LessonSession` ichida uchta maydon edi va `release()` uni
 * UMUMAN to'xtatmasdi: ustoz ilovani recents'dan surib tashlasa yoki sessiya
 * majburan yopilsa ikki `MediaCodec` va `HandlerThread` oqib qolardi, mikser
 * jimlik bilan faylni o'stirardi (~40 MB/soat) va fayl `moov`siz — ya'ni
 * buzuq — qolardi. Endi [stop] har qanday bo'shatish yo'lidan chaqiriladi va
 * ikki marta chaqirilsa ham zararsiz.
 *
 * Fayl nomi har yozuvda YANGI ([RecordingFiles.newFile]): bitta `<lessonId>.mp4`
 * yo'li ikkinchi Record bosilganda birinchi yuklanayotgan faylni kesib
 * tashlardi (H4).
 */
class LocalRecordingController(
    private val dir: File,
    private val factory: RecorderFactory,
    private val now: () -> Long = System::currentTimeMillis,
) {
    private var recorder: Recorder? = null
    private var file: File? = null

    val isRecording: Boolean get() = recorder != null

    /**
     * @return yozuv fayli yoki `null` (yozuvchi boshlanmadi). Allaqachon yozilayotgan
     *   bo'lsa joriy fayl qaytadi — ikkinchi yozuvchi ochilmaydi.
     */
    fun start(
        lessonId: String,
        screenVideo: VideoTrack,
        bitrate: Int,
        localAudio: AudioTrack?,
        remoteAudio: List<AudioTrack>,
    ): File? {
        file?.let { return it }
        dir.mkdirs()
        val out = RecordingFiles.newFile(dir, lessonId, now())
        val rec = factory.create(out)
        if (!rec.start(screenVideo, bitrate, localAudio, remoteAudio)) return null
        recorder = rec
        file = out
        return out
    }

    /** Ovoz manbai (mikrofon yoki o'quvchi) yozuvga ulanadi; yozuv yo'q bo'lsa hech nima. */
    fun onAudio(track: AudioTrack, added: Boolean) {
        val rec = recorder ?: return
        if (added) rec.addRemoteAudio(track) else rec.removeRemoteAudio(track)
    }

    /**
     * Yozuvni yakunlaydi va yuklashga YAROQLI faylni qaytaradi (bo'sh/buzuq → `null`,
     * qaror sof [RecorderPipeline.isUsableOutput]).
     *
     * `Dispatchers.Default` da: [Recorder.stop] kodek va muxer'ni to'xtatguncha
     * bloklaydi (M3 — avval bu asosiy oqimda edi).
     */
    suspend fun stop(): File? {
        val rec = recorder ?: return null
        val out = file
        recorder = null
        file = null
        withContext(Dispatchers.Default) { rec.stop() }
        return out?.takeIf { RecorderPipeline.isUsableOutput(it.exists(), it.length()) }
    }
}

/**
 * Yozuv fayllarining nomlash qoidasi — **sof** (JVM testida).
 *
 * `<lessonId>_<epochMs>.mp4`. Ajratuvchi `_`: dars ID'si UUID (faqat `-` va
 * hex), demak `_` undan keyingi birinchi belgi. Eski `<lessonId>.mp4` nomlari
 * ham o'qiladi ([lessonIdOf]) — yangilanishdan oldin qolgan fayllar
 * yuklanishda davom etsin.
 */
object RecordingFiles {
    private const val EXT = ".mp4"
    private const val SEP = '_'

    fun newFile(dir: File, lessonId: String, nowMs: Long): File = File(dir, fileName(lessonId, nowMs))

    fun fileName(lessonId: String, nowMs: Long): String = "$lessonId$SEP$nowMs$EXT"

    /** Fayl nomidan dars ID'si; `.mp4` bo'lmasa `null`. */
    fun lessonIdOf(fileName: String): String? {
        if (!fileName.endsWith(EXT)) return null
        val stem = fileName.removeSuffix(EXT)
        return stem.substringBefore(SEP).takeIf { it.isNotBlank() }
    }

    /** Nomdagi boshlanish vaqti (eski nomda yo'q → `null`). */
    fun startedAtOf(fileName: String): Long? =
        fileName.removeSuffix(EXT).substringAfter(SEP, "").toLongOrNull()
}
