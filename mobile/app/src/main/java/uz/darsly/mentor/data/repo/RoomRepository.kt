package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.RoomParticipantDto
import uz.darsly.mentor.data.api.RoomStateResp
import uz.darsly.mentor.data.api.RoomToken
import javax.inject.Inject

/**
 * Xonaga kirish va uning holati (`/lessons/{id}/…`).
 *
 * ## Nega bu repozitoriy paydo bo'ldi
 *
 * `RoomViewModel` `DarslyApi` ga **14 marta** to'g'ridan-to'g'ri murojaat
 * qilardi — boshqa hamma ViewModel repozitoriy orqali ishlagani holda.
 * Ya'ni `ui/ → repo/ → api/` qoidasi aynan eng murakkab ekranda buzilgan edi,
 * va istisno bilan qoida — qoida emas.
 *
 * Amaliy oqibati: xona mantiqini sinash uchun butun `DarslyApi` interfeysini
 * mocklash kerak bo'lardi. Endi ViewModel uchta tor repozitoriyga bog'lanadi
 * va ularning har birini alohida almashtirish mumkin.
 */
class RoomRepository @Inject constructor(
    private val api: DarslyApi,
) {

    /** Host tokeni — xonaga ulanish uchun. */
    suspend fun hostToken(lessonId: String): Result<RoomToken> = runCatching {
        api.hostToken(lessonId).data ?: error("Token bo'sh keldi")
    }

    /**
     * Xonaning joriy holati (ko'tarilgan qo'llar).
     *
     * Data-channel faqat KELAJAKDAGI signallarni beradi — ustoz kech qo'shilsa
     * yoki ilovani qayta ochsa, o'shangacha ko'tarilgan qo'llar unga yetib
     * bormasdi. Shu sabab holat serverdan alohida olinadi.
     */
    suspend fun state(lessonId: String, roomToken: String): Result<RoomStateResp?> =
        runCatching { api.roomState(lessonId, roomToken).data }

    /** Xonadagi ishtirokchilar — moderatsiya paneli uchun (LiveKit'dan, server orqali). */
    suspend fun participants(lessonId: String): Result<List<RoomParticipantDto>> =
        runCatching { api.participants(lessonId).data.orEmpty() }

    /**
     * Emoji reaksiya (№14) — room-token bilan.
     *
     * Server hech nima saqlamaydi, faqat xonaga tarqatadi (ustozning O'Z
     * ekranida ham u serverdan qaytib keladi — optimistik ko'rsatish shart
     * emas). Ruxsatsiz emoji 400, tez-tez bosilsa 429 beradi; ikkalasi ham
     * ustozga ko'rsatiladi.
     */
    suspend fun sendReaction(lessonId: String, roomToken: String, emoji: String): Result<Unit> =
        runCatching {
            val resp = api.sendReaction(lessonId, uz.darsly.mentor.data.api.SendReactionReq(roomToken, emoji))
            if (!resp.isSuccessful) throw retrofit2.HttpException(resp)
        }

    /** Darsni yakunlaydi: xona yopiladi, hamma uziladi, yozuv to'xtaydi. */
    suspend fun endLesson(lessonId: String): Result<Unit> =
        runCatching { api.endLesson(lessonId) }
}
