package uz.darsly.mentor.data.repo

import retrofit2.HttpException
import retrofit2.Response
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.LowerHandReq
import uz.darsly.mentor.data.api.MuteAllReq
import uz.darsly.mentor.data.api.RemoveParticipantReq
import javax.inject.Inject

/**
 * Ustozning moderatsiya amallari.
 *
 * Alohida repozitoriy, chunki bu ETTITA amal bitta mahsulot qoidasiga
 * bo'ysunadi: **hammasi darsga tegishli va hammasi natijada ro'yxatni
 * yangilashni talab qiladi**. Ular `RoomViewModel` ichida yetti alohida
 * `Net.api` chaqiruvi bo'lib yotardi.
 *
 * Har biri `Result` qaytaradi — xatoni ko'rsatish qarori (snackbar? banner?)
 * UI qatlamining ishi, repozitoriyniki emas.
 *
 * ## `Response<Unit>` tuzog'i (H2)
 * Endpointlar 204 tanasiz qaytaradi, shuning uchun API'da `Response<Unit>`.
 * Lekin `Response` bilan Retrofit **4xx/5xx da ham muvaffaqiyat** qaytaradi —
 * avval `runCatching { api.mute(...) }.map { }` 403/404/500 ni ham "bajarildi"
 * deb ko'rsatardi: ustoz "mute qildim" deb o'ylar, o'quvchi esa gapiraverardi.
 * Endi har javob [ok] darvozasidan o'tadi va xato `HttpException` sifatida
 * (`ApiErrors.humanError` tanish shakli) qaytadi — `DarslyApi.unblock`
 * izohidagi qoida bilan bir xil.
 */
class ModerationRepository @Inject constructor(
    private val api: DarslyApi,
) {

    /** Host'dan tashqari hammani mute qiladi. [allowSelfUnmute] null bo'lsa bayroq o'zgarmaydi. */
    suspend fun muteAll(lessonId: String, allowSelfUnmute: Boolean? = null): Result<Unit> =
        ok { api.muteAll(lessonId, MuteAllReq(allowSelfUnmute)) }

    suspend fun mute(lessonId: String, identity: String): Result<Unit> =
        ok { api.muteParticipant(lessonId, identity) }

    /**
     * Ishtirokchini darsdan chiqaradi.
     *
     * Server buni Redis'dagi ban ro'yxatiga ham yozadi — aks holda chiqarilgan
     * odam eski tokeni bilan darhol qaytib kelardi (backend `room.EnforceJoin`).
     *
     * [scope]: "lesson" — shu darsdan; "mentor" — doimiy qora ro'yxat.
     */
    suspend fun remove(lessonId: String, identity: String, scope: String = "lesson"): Result<Unit> =
        ok { api.removeParticipant(lessonId, identity, RemoveParticipantReq(scope)) }

    /** O'quvchiga vaqtincha media publish (so'zlash) ruxsatini beradi. */
    suspend fun allowSpeak(lessonId: String, identity: String): Result<Unit> =
        ok { api.allowSpeak(lessonId, identity) }

    suspend fun revokeSpeak(lessonId: String, identity: String): Result<Unit> =
        ok { api.revokeSpeak(lessonId, identity) }

    suspend fun lowerHand(lessonId: String, identity: String): Result<Unit> =
        ok { api.lowerHand(lessonId, LowerHandReq(identity)) }

    suspend fun lowerAllHands(lessonId: String): Result<Unit> =
        ok { api.lowerAllHands(lessonId) }

    /** Tarmoq istisnosi ham, muvaffaqiyatsiz HTTP status ham — `Result.failure`. */
    private suspend inline fun ok(call: () -> Response<Unit>): Result<Unit> = runCatching {
        val resp = call()
        if (!resp.isSuccessful) throw HttpException(resp)
    }
}
