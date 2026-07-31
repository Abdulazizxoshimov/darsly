package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.CreatePollReq
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.Poll
import uz.darsly.mentor.data.api.PollResults
import javax.inject.Inject

/**
 * Dars ichidagi so'rovnomalar (№7).
 *
 * ## Ikki auth yo'li — ataylab
 * Yaratish/yopish/e'lon qilish — **JWT** (mentor), natijani o'qish esa
 * **room-token** bilan (`GET /polls/{id}/results?token=`). Bu backend qarori:
 * natija endpointi xonadagi hamma uchun ochiq va "xonada bo'lganlik isboti"
 * sifatida LiveKit tokenini talab qiladi. Mentor host tokeni bilan doim 200
 * oladi — ya'ni u jonli natijani KO'RISH uchun so'rovnomani yopishi shart emas.
 */
class PollsRepository @Inject constructor(
    private val api: DarslyApi,
) {

    /** `hs.Success(items)` — sahifalanmagan ro'yxat. */
    suspend fun list(lessonId: String): Result<List<Poll>> =
        runCatching { api.polls(lessonId).data.orEmpty() }

    suspend fun create(lessonId: String, req: CreatePollReq): Result<Poll> = runCatching {
        api.createPoll(lessonId, req).data ?: error("Server bo'sh javob qaytardi")
    }

    /**
     * Natijani o'quvchilarga ochadi (idempotent).
     *
     * `mentor_only` so'rovnomada server **400** beradi — rejim yaratishda
     * tanlangan va o'zgarmas. UI bu tugmani o'sha holatda umuman ko'rsatmaydi
     * ([uz.darsly.mentor.ui.room.PollForm.canPublish]).
     */
    suspend fun publish(lessonId: String, pollId: String): Result<PollResults> = runCatching {
        api.publishPoll(lessonId, pollId).data ?: error("Server bo'sh javob qaytardi")
    }

    /** Ovoz berishni to'xtatadi. Natijani OCHMAYDI — bu alohida amal. */
    suspend fun close(pollId: String): Result<PollResults> = runCatching {
        api.closePoll(pollId).data ?: error("Server bo'sh javob qaytardi")
    }

    /** Jonli natija (mentor host-tokeni bilan). */
    suspend fun results(pollId: String, roomToken: String): Result<PollResults> = runCatching {
        api.pollResults(pollId, roomToken).data ?: error("Server bo'sh javob qaytardi")
    }
}
