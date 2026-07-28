package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.Net
import uz.darsly.mentor.data.api.WaitingRoomRequest
import java.io.IOException
import retrofit2.HttpException

/**
 * Kutish xonasi — ustoz tomoni (M27).
 *
 * O'quvchi `POST /joinlink/:slug` bilan kirishga urinadi → backend so'rov yaratadi
 * va ustozga **WebSocket** orqali darhol yuboradi. Bu repozitoriy ikki ish qiladi:
 * ekran ochilganda **boshlang'ich ro'yxatni** oladi (WS'dan oldin kelganlar
 * yo'qolmasin) va qarorlarni yuboradi.
 *
 * ## Nega 409 xato emas
 * `admit`/`reject` backend'da atomik (`TransitionFromPending` — `WHERE
 * status='pending'`). Ikkinchi qaror **409** oladi. Bu poyga natijasi: ustoz ikki
 * marta bosdi yoki ikki qurilmadan qaror qilindi. Foydalanuvchi nuqtai nazaridan
 * ish **bajarilgan**, shuning uchun 409 muvaffaqiyat deb hisoblanadi va
 * so'rov ro'yxatdan olib tashlanadi — aks holda ekranda hal qilingan so'rov
 * "xato" bilan osilib qolardi.
 */
class WaitingRoomRepository(private val api: DarslyApi) {

    suspend fun pending(lessonId: String): Result<List<WaitingRoomRequest>> = runCatching {
        api.waitingRoom(lessonId).data.orEmpty()
    }

    /**
     * So'rovni qabul qilish. Javobdagi guest tokeni **ataylab tashlanadi**: u
     * o'quvchiga WS orqali server tomonidan yuboriladi, ustozga esa keraksiz.
     */
    suspend fun admit(requestId: String): Result<Unit> =
        runCatching { api.admitWaitingRoom(requestId) }.map { }.recoverAlreadyDecided()

    suspend fun reject(requestId: String): Result<Unit> =
        runCatching { api.rejectWaitingRoom(requestId) }.recoverAlreadyDecided()

    /** 409 (`CONFLICT`) — so'rov allaqachon hal qilingan. Bu xato emas. */
    private fun Result<Unit>.recoverAlreadyDecided(): Result<Unit> = recoverCatching { t ->
        if (t is HttpException && t.code() == HTTP_CONFLICT) Unit else throw t
    }

    companion object {
        private const val HTTP_CONFLICT = 409

        /** @see LessonsRepository.isOffline */
        fun isOffline(t: Throwable): Boolean = t is IOException

        fun create(): WaitingRoomRepository = WaitingRoomRepository(Net.api)
    }
}
