package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.DarslyApi
import javax.inject.Inject
import uz.darsly.mentor.data.api.Recording
import uz.darsly.mentor.data.api.RecordingDownload
import java.io.IOException

/**
 * Dars yozuvlari (LiveKit Egress → MinIO).
 *
 * Backend allaqachon tayyor edi (`/lessons/:id/recording/start`,
 * `/recordings/:id/stop|download`), mobil ilovada esa faqat dars yaratishdagi
 * "Yozib olish" belgisi bor edi — yozuvni **ko'rish yoki yuklab olish** imkoni
 * yo'q edi.
 *
 * ## Yozuv holatlari haqida
 * `start` → `recording` · `stop` → `processing` · LiveKit webhook'i kelgach →
 * `ready` yoki `failed`. Ya'ni to'xtatgandan keyin fayl **darhol tayyor emas** —
 * UI buni aytishi shart, aks holda ustoz "yuklab olish ishlamayapti" deb o'ylaydi.
 */
class RecordingsRepository @Inject constructor(private val api: DarslyApi) {

    suspend fun list(lessonId: String): Result<List<Recording>> = runCatching {
        api.recordings(lessonId).data.orEmpty()
    }

    suspend fun start(lessonId: String): Result<Recording> = runCatching {
        api.startRecording(lessonId).data
            ?: throw IllegalStateException("Server bo'sh javob qaytardi")
    }

    suspend fun stop(recordingId: String): Result<Unit> = runCatching { api.stopRecording(recordingId) }

    /**
     * Vaqtinchalik yuklab olish havolasi (presigned URL).
     *
     * Havola **muddatli** ([RecordingDownload.expiresInS]) — shuning uchun u
     * saqlanmaydi va har bosishda qaytadan so'raladi. Saqlangan havola bir
     * necha soatdan keyin 403 berardi va bu "yozuv yo'qolgan"dek ko'rinardi.
     */
    suspend fun downloadUrl(recordingId: String): Result<RecordingDownload> = runCatching {
        api.recordingDownload(recordingId).data
            ?: throw IllegalStateException("Server bo'sh javob qaytardi")
    }

    companion object {
        /** @see LessonsRepository.isOffline */
        fun isOffline(t: Throwable): Boolean = t is IOException

    }
}
