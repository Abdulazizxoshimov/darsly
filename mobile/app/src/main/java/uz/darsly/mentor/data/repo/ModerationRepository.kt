package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.MuteAllReq
import uz.darsly.mentor.data.api.RemoveParticipantReq
import uz.darsly.mentor.data.api.LowerHandReq
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
 */
class ModerationRepository @Inject constructor(
    private val api: DarslyApi,
) {

    /** Host'dan tashqari hammani mute qiladi. */
    /** [allowSelfUnmute] null bo'lsa bayroq o'zgarmaydi (faqat mute qilinadi). */
    suspend fun muteAll(lessonId: String, allowSelfUnmute: Boolean? = null): Result<Unit> =
        runCatching { api.muteAll(lessonId, MuteAllReq(allowSelfUnmute)) }.map { }

    suspend fun mute(lessonId: String, identity: String): Result<Unit> =
        runCatching { api.muteParticipant(lessonId, identity) }.map { }

    /**
     * Ishtirokchini darsdan chiqaradi.
     *
     * Server buni Redis'dagi ban ro'yxatiga ham yozadi — aks holda chiqarilgan
     * odam eski tokeni bilan darhol qaytib kelardi (backend `room.EnforceJoin`).
     */
    /** [scope]: "lesson" — shu darsdan; "mentor" — doimiy qora ro'yxat. */
    suspend fun remove(lessonId: String, identity: String, scope: String = "lesson"): Result<Unit> =
        runCatching { api.removeParticipant(lessonId, identity, RemoveParticipantReq(scope)) }.map { }

    /** O'quvchiga vaqtincha media publish (so'zlash) ruxsatini beradi. */
    suspend fun allowSpeak(lessonId: String, identity: String): Result<Unit> =
        runCatching { api.allowSpeak(lessonId, identity) }.map { }

    suspend fun revokeSpeak(lessonId: String, identity: String): Result<Unit> =
        runCatching { api.revokeSpeak(lessonId, identity) }.map { }

    suspend fun lowerHand(lessonId: String, identity: String): Result<Unit> =
        runCatching { api.lowerHand(lessonId, LowerHandReq(identity)) }.map { }

    suspend fun lowerAllHands(lessonId: String): Result<Unit> =
        runCatching { api.lowerAllHands(lessonId) }.map { }
}
