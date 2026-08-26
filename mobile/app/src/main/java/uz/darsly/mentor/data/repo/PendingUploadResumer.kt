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
 * ketadi, LEKIN u app-scoped korutinada ishlaydi — ilova O'LDIRILSA (masalan
 * dars o'rtasida yangilansa yoki OS process'ni yopsa) yuklash yarim yo'lda
 * uziladi va yozuv **jimgina qoladi**: fayl telefonda, server esa bilmaydi
 * (qator "recording" holatida turib qoladi). Muvaffaqiyatsiz yuklash faylni
 * O'CHIRMAYDI ([LocalRecordingRepository] faqat `onSuccess`da o'chiradi),
 * shuning uchun fayl saqlanadi va bu resumer keyingi ochilishda uni topib
 * yuklaydi. Fayl nomi = `<lessonId>.mp4` (LessonSession), recording_id esa
 * arxiv endpointidan olinadi (u "recording" holatini ham id bilan qaytaradi).
 *
 * ## Xavfsizlik
 * - Faol yozuvga tegmaydi: resumer kirishda (dars boshlanishidan oldin) ishlaydi.
 * - Bir vaqtda faqat bitta yurish ([running] qulfi) — ikki marta yuklamaydi.
 * - Faqat `ready` (aniq yuklangan) faylni o'chiradi; `recording` ni yuklaydi;
 *   qolgan holatlarga TEGMAYDI; xatoda fayl saqlanadi (keyin qayta urinadi).
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

    fun resume(scope: CoroutineScope) {
        if (!running.compareAndSet(false, true)) return
        scope.launch(Dispatchers.IO) {
            try {
                val dir = File(ctx.filesDir, "recordings")
                val onDisk = dir.listFiles { f -> f.isFile }?.toList().orEmpty()
                // Nomzod tanlash/tartib/dedup — sof [selectUploads] da (JVM testida).
                val files = selectUploads(onDisk.map { PendingFile(it.name, it.length()) })
                    .mapNotNull { p -> onDisk.firstOrNull { it.name == p.name } }
                if (files.isEmpty()) return@launch
                Log.i(TAG, "resume: ${files.size} ta yuklanmagan fayl topildi")
                for (file in files) runCatching { resumeOne(file) }
                    .onFailure { Log.e(TAG, "resume: ${file.name} xato", it) }
            } finally {
                running.set(false)
            }
        }
    }

    private suspend fun resumeOne(file: File) {
        val lessonId = file.name.removeSuffix(".mp4")
        val rec = runCatching { api.lessonArchive(lessonId).data?.recording }.getOrNull()
        Log.i(TAG, "resume: ${file.name} (${file.length()} bayt) → status=${rec?.status ?: "yo'q"}")
        when (decide(rec?.status)) {
            Action.UPLOAD -> {
                val id = rec!!.id
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
                Log.i(TAG, "resume: ${file.name} allaqachon yuklangan — yetim fayl o'chirildi")
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

    /** Diskda topilgan yozuv fayli (nomi = `<lessonId>.mp4`) va o'lchami. */
    data class PendingFile(val name: String, val sizeBytes: Long)

    companion object {
        private const val TAG = "UploadResumer"

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
         */
        fun selectUploads(files: List<PendingFile>): List<PendingFile> =
            files.filter { it.name.endsWith(".mp4") && RecorderPipeline.isUsableOutput(true, it.sizeBytes) }
                .sortedBy { it.name }
                .distinctBy { it.name.removeSuffix(".mp4") }

        /**
         * Arxiv statusiga qarab qaror. `recording` — hali yuklanmagan → yuklaymiz;
         * `ready` — aniq yuklangan → yetim faylni o'chiramiz; qolgani (processing/
         * failed/archived/expired/null/xato) — TEGMAYMIZ (fayl saqlanadi).
         */
        fun decide(status: String?): Action = when (status) {
            "recording" -> Action.UPLOAD
            "ready" -> Action.DELETE
            else -> Action.SKIP
        }
    }
}
