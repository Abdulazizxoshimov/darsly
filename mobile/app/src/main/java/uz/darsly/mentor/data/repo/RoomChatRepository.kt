package uz.darsly.mentor.data.repo

import android.net.Uri
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.toRequestBody
import uz.darsly.mentor.data.api.ChatMessageDto
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.SendRoomChatReq
import javax.inject.Inject

/**
 * Dars ichidagi chat.
 *
 * Xabarlar SERVER orqali ketadi (LiveKit data-channel orqali emas): shunda
 * ular saqlanadi, tarixda qoladi va kech kirgan ham ko'radi. Bundan tashqari
 * server yuboruvchini tokendan aniqlaydi — klient payload'iga ishonilmaydi.
 */
class RoomChatRepository @Inject constructor(
    private val api: DarslyApi,
    private val attachments: ChatAttachments,
) {

    /** Tarix — server eng yangidan eskiga beradi. */
    suspend fun history(lessonId: String, roomToken: String): Result<List<ChatMessageDto>> =
        runCatching { api.roomChat(lessonId, roomToken).data.orEmpty() }

    /** `to` bo'sh bo'lsa — hammaga, aks holda shaxsiy. */
    suspend fun send(
        lessonId: String,
        roomToken: String,
        body: String,
        to: String,
    ): Result<ChatMessageDto?> = runCatching {
        api.sendRoomChat(lessonId, SendRoomChatReq(roomToken, body, to)).data
    }

    /**
     * Xabarni o'chirish (moderatsiya, №6) — faqat dars egasi.
     *
     * **404 xato EMAS**: u "allaqachon o'chirilgan" degani (server atomik
     * `WHERE deleted_at IS NULL` bilan ishlaydi). Ikki marta bosilgan yoki
     * ikki qurilmadan kelgan buyruqda foydalanuvchi nuqtai nazaridan ish
     * BAJARILGAN — uni xato deb ko'rsatish "o'chmadi" degan yolg'on bo'lardi.
     * Aynan shu qoida kutish xonasidagi 409 uchun ham amal qiladi.
     */
    suspend fun delete(lessonId: String, messageId: String): Result<Unit> = runCatching {
        val resp = api.deleteChatMessage(lessonId, messageId)
        if (!resp.isSuccessful && resp.code() != 404) {
            throw retrofit2.HttpException(resp)
        }
    }

    /**
     * Fayl ulashish (№15) — JWT yo'li (`POST /lessons/{id}/chat/upload`).
     *
     * URI o'qish va chegaralar [ChatAttachments] da; bu yerda faqat multipart
     * yig'iladi. `Content-Type` sifatida `application/octet-stream` yuboriladi:
     * server klientnikini ATAYLAB e'tiborga olmaydi va kanonik MIME'ni o'zi
     * aniqlaydi (kengaytma + mazmun sniff), ya'ni bu yerda taxmin qilish
     * faqat yolg'on aniqlik bo'lardi.
     */
    suspend fun upload(
        lessonId: String,
        uri: Uri,
        body: String = "",
        to: String = "",
    ): Result<ChatMessageDto?> {
        val file = attachments.read(uri).getOrElse { return Result.failure(it) }
        return runCatching {
            val part = MultipartBody.Part.createFormData(
                "file",
                file.name,
                file.bytes.toRequestBody(OCTET_STREAM),
            )
            api.uploadChatFile(
                lessonId = lessonId,
                file = part,
                body = body.takeIf { it.isNotBlank() }?.toRequestBody(TEXT_PLAIN),
                to = to.takeIf { it.isNotBlank() }?.toRequestBody(TEXT_PLAIN),
            ).data
        }
    }

    private companion object {
        val OCTET_STREAM = "application/octet-stream".toMediaTypeOrNull()
        val TEXT_PLAIN = "text/plain".toMediaTypeOrNull()
    }
}
