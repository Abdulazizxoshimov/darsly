package uz.darsly.mentor.data.repo

import android.content.Context
import android.media.MediaMetadataRetriever
import android.util.Log
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.livekit.RecorderPipeline
import uz.darsly.mentor.data.livekit.RecordingFiles
import java.io.File
import java.time.Instant
import java.util.concurrent.atomic.AtomicBoolean
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Yuklanmay qolgan lokal yozuvlarni ILOVA OCHILGANDA avtomatik qayta yuklaydi
 * (tugma kerak emas).
 *
 * ## Nega kerak (aniqlangan muammo, 2026-08-16)
 * Yozuv telefonda tugagach [LocalRecordingRepository.upload] bilan serverga
 * ketadi, LEKIN ilova O'LDIRILSA (masalan dars o'rtasida yangilansa yoki OS
 * process'ni yopsa) yuklash yarim yo'lda uziladi va yozuv **jimgina qoladi**:
 * fayl telefonda, server esa bilmaydi (qator "recording" holatida turib
 * qoladi). Muvaffaqiyatsiz yuklash faylni O'CHIRMAYDI, shuning uchun fayl
 * saqlanadi va bu resumer keyingi ochilishda uni topib yuklaydi. Fayl nomi
 * `<lessonId>_<epochMs>.mp4` ([RecordingFiles]), recording_id esa arxiv
 * endpointidan olinadi (u "recording" holatini ham id bilan qaytaradi).
 *
 * ## Xavfsizlik (S4)
 * - Faol yozuvga tegmaydi: resumer kirishda (dars boshlanishidan oldin) ishlaydi.
 * - Bir vaqtda faqat bitta yurish ([running] qulfi) — ikki marta yuklamaydi.
 * - `ready` → o'chiriladi; `recording` → yuklanadi; `failed`/`expired` → server
 *   voz kechgan, o'chiriladi; qolgani ([Action.SKIP]) faqat [MAX_AGE_MS]
 *   gacha saqlanadi — o'quvchilar yozuvi telefonda ABADIY yotmasligi kerak.
 * - Logoutda ([clearAll]) hammasi o'chadi: keyingi kirgan ustoz oldingisining
 *   darsini ko'rmasin.
 */
@Singleton
class PendingUploadResumer @Inject constructor(
    @ApplicationContext private val ctx: Context,
    private val api: DarslyApi,
    private val repo: LocalRecordingRepository,
) {
    /** Yetim fayl bilan nima qilish (sof qaror — JVM testida qotirilgan). */
    enum class Action { UPLOAD, DELETE, SKIP }

    private val running = AtomicBoolean(false)

    private val dir: File get() = File(ctx.filesDir, "recordings")

    fun resume(scope: CoroutineScope) {
        if (!running.compareAndSet(false, true)) return
        scope.launch(Dispatchers.IO) {
            try {
                val onDisk = dir.listFiles { f -> f.isFile }?.toList().orEmpty()
                val now = System.currentTimeMillis()
                // Nomzod tanlash/tartib/dedup — sof [selectUploads] da (JVM testida).
                val files = selectUploads(onDisk.map { PendingFile(it.name, it.length()) })
                    .mapNotNull { p -> onDisk.firstOrNull { it.name == p.name } }
                if (files.isEmpty()) return@launch
                Log.i(TAG, "resume: ${files.size} ta yuklanmagan fayl topildi")
                for (file in files) runCatching { resumeOne(file, now) }
                    .onFailure { Log.e(TAG, "resume: ${file.name} xato", it) }
            } finally {
                running.set(false)
            }
        }
    }

    /**
     * Logout — barcha lokal yozuv fayllari o'chiriladi (S4).
     *
     * Tokenlar allaqachon yo'q, yuklab bo'lmaydi; boshqa ustoz kirsa uning
     * telefonida oldingi darsning yozuvi turmasligi kerak.
     */
    fun clearAll(scope: CoroutineScope) {
        scope.launch(Dispatchers.IO) {
            val removed = dir.listFiles { f -> f.isFile }.orEmpty().count { it.delete() }
            if (removed > 0) Log.i(TAG, "logout: $removed ta lokal yozuv o'chirildi")
        }
    }

    private suspend fun resumeOne(file: File, nowMs: Long) {
        val lessonId = RecordingFiles.lessonIdOf(file.name) ?: return
        val rec = runCatching { api.lessonArchive(lessonId).data?.recording }.getOrNull()
        Log.i(TAG, "resume: ${file.name} (${file.length()} bayt) → status=${rec?.status ?: "yo'q"}")
        when (decide(rec?.status, ageMs = nowMs - file.lastModified())) {
            Action.UPLOAD -> {
                val id = rec?.id ?: return
                val durationSec = durationSec(file)
                val endedAt = Instant.ofEpochMilli(file.lastModified()).toString()
                repo.upload(id, file, durationSec, endedAt)
                    .onSuccess {
                        file.delete()
                        Log.i(TAG, "resume: ${file.name} YUKLANDI (rec=$id)")
                    }
                    .onFailure { Log.e(TAG, "resume: ${file.name} yuklash xato — saqlanadi", it) }
            }
            Action.DELETE -> {
                file.delete()
                Log.i(TAG, "resume: ${file.name} o'chirildi (status=${rec?.status ?: "yo'q"})")
            }
            Action.SKIP -> Log.i(TAG, "resume: ${file.name} — hozircha tegmaymiz")
        }
    }

    private fun durationSec(file: File): Int {
        val r = MediaMetadataRetriever()
        return try {
            r.setDataSource(file.absolutePath)
            val ms = r.extractMetadata(MediaMetadataRetriever.METADATA_KEY_DURATION)?.toLongOrNull() ?: 0L
            (ms / 1000).toInt().coerceAtLeast(1)
        } catch (t: Throwable) {
            1 // buzuq fayl — davomiylikni o'lchab bo'lmadi; server baribir xom nusxani saqlaydi
        } finally {
            runCatching { r.release() }
        }
    }

    /** Diskda topilgan yozuv fayli va o'lchami. */
    data class PendingFile(val name: String, val sizeBytes: Long)

    companion object {
        private const val TAG = "UploadResumer"

        /** Noma'lum holatdagi fayl telefonda eng ko'pi shuncha saqlanadi (7 kun). */
        const val MAX_AGE_MS = 7L * 24 * 60 * 60 * 1000

        /**
         * Qayta yuklashga NOMZOD fayllarni tanlaydi, tartiblaydi va dublikatni
         * yechadi — **sof** (JVM testida qulflangan).
         *
         *  · faqat `.mp4` va BO'SH EMAS (0 bayt — buzuq, yuklash serverni yaroqsiz
         *    fayl bilan to'ldirardi; [RecorderPipeline.isUsableOutput] bilan bir qoida);
         *  · nom bo'yicha BARQAROR tartib — diskning tasodifiy tartibiga tayanmaymiz,
         *    aks holda urinishlar har ochilishda boshqacha bo'lib, xatoni takrorlash
         *    qiyinlashardi;
         *  · har dars (lessonId) BIR MARTA — bitta darsni ikki marta yuklamaymiz.
         *    Bir darsda bir necha segment bo'lsa ENG YANGISI (nomdagi vaqt tamg'asi
         *    bo'yicha) tanlanadi; eskilari keyingi yurishlarda yoki yosh chegarasi
         *    bilan tozalanadi.
         */
        fun selectUploads(files: List<PendingFile>): List<PendingFile> =
            files.filter { RecordingFiles.lessonIdOf(it.name) != null && RecorderPipeline.isUsableOutput(true, it.sizeBytes) }
                .sortedWith(compareBy<PendingFile> { RecordingFiles.lessonIdOf(it.name) }.thenByDescending { it.name })
                .distinctBy { RecordingFiles.lessonIdOf(it.name) }

        /**
         * Arxiv statusi va fayl yoshiga qarab qaror.
         *
         *  · `recording` — hali yuklanmagan → yuklaymiz;
         *  · `ready` — aniq yuklangan → yetim faylni o'chiramiz;
         *  · `failed` / `expired` — server voz kechgan, fayl endi hech qachon
         *    yuklanmaydi → o'chiramiz;
         *  · qolgani (processing/archived/null/xato) — TEGMAYMIZ, lekin
         *    [MAX_AGE_MS] dan eski bo'lsa o'chiramiz (S4: o'quvchi yozuvi
         *    telefonda abadiy qolmasin).
         */
        fun decide(status: String?, ageMs: Long = 0L): Action = when (status) {
            "recording" -> Action.UPLOAD
            "ready", "failed", "expired" -> Action.DELETE
            else -> if (ageMs > MAX_AGE_MS) Action.DELETE else Action.SKIP
        }
    }
}
