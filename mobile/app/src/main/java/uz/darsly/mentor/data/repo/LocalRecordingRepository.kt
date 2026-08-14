package uz.darsly.mentor.data.repo

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.asRequestBody
import uz.darsly.mentor.data.api.CompleteRecordingReq
import uz.darsly.mentor.data.api.DarslyApi
import java.io.File
import java.io.IOException
import java.util.concurrent.TimeUnit
import javax.inject.Inject

/**
 * Client-side (lokal) yozuvni serverga yuklash — «Zoom local recording».
 *
 * Telefon darsni O'ZI yozadi (`LocalRecorder`), bu repo esa faylni serverga
 * yetkazadi: `local-start` (yozuv qatori) → `upload-url` (presigned PUT) →
 * to'g'ridan MinIO'ga PUT → `complete` (server faylni tekshirib `ready` qiladi).
 */
class LocalRecordingRepository @Inject constructor(private val api: DarslyApi) {

    // Auth-header'siz ALOHIDA client: presigned PUT MinIO imzosiga tayanadi,
    // ilovaning Authorization interceptori imzoni buzmasligi kerak.
    private val putClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(60, TimeUnit.MINUTES) // katta fayl (~1 GB) sekin tarmoqda
            .build()
    }

    /** Yozuv qatorini yaratadi, `recording_id` qaytaradi. */
    suspend fun localStart(lessonId: String): Result<String> = runCatching {
        api.localStartRecording(lessonId).data?.id
            ?: throw IllegalStateException("Server bo'sh javob qaytardi")
    }

    /** Faylni yuklaydi: upload-url → PUT → complete. */
    suspend fun upload(recordingId: String, file: File, durationSec: Int, endedAtIso: String?): Result<Unit> = runCatching {
        val url = api.recordingUploadUrl(recordingId).data?.url
            ?: throw IllegalStateException("upload-url bo'sh")
        val body = file.asRequestBody("video/mp4".toMediaType())
        val req = Request.Builder().url(url).put(body).build()
        putClient.newCall(req).execute().use { resp ->
            if (!resp.isSuccessful) throw IOException("MinIO PUT muvaffaqiyatsiz: ${resp.code}")
        }
        api.completeRecording(recordingId, CompleteRecordingReq(durationSec, endedAtIso))
    }

    companion object {
        fun isOffline(t: Throwable): Boolean = t is IOException
    }
}
