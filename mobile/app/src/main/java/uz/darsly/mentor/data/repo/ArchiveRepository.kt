package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.LessonArchiveDto
import java.io.IOException
import javax.inject.Inject

/**
 * Dars arxivi — video + chat + materiallar (`GET /lessons/:id/archive`).
 *
 * YAGONA so'rov: vaqt tayanchi (video t=0) ham, chat ham bir o'qishdan keladi.
 * Backend arxivni to'liq beradi; pleyer `recording.url` ni (presigned) stream
 * qiladi va o'sha havola yuklab olish uchun ham ishlaydi.
 */
class ArchiveRepository @Inject constructor(private val api: DarslyApi) {

    suspend fun archive(lessonId: String): Result<LessonArchiveDto> = runCatching {
        api.lessonArchive(lessonId).data
            ?: throw IllegalStateException("Server bo'sh javob qaytardi")
    }

    companion object {
        /** @see LessonsRepository.isOffline */
        fun isOffline(t: Throwable): Boolean = t is IOException
    }
}
