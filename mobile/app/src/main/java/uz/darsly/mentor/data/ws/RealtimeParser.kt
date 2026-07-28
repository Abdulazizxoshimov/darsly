package uz.darsly.mentor.data.ws

import com.squareup.moshi.JsonAdapter
import com.squareup.moshi.Moshi
import com.squareup.moshi.Types
import uz.darsly.mentor.data.api.RoomToken

/**
 * WS xabarini turlangan hodisaga aylantiradi — **sof** funksiya (Android'siz, JVM testida).
 *
 * ## Nega alohida va nega Android'siz
 * Real-time xatolarining eng yomon turi — "xabar keldi, lekin noto'g'ri o'qildi va ilova
 * jim yiqildi". Parse mantiqi soketdan ajratilgani uchun har bir hodisa turi, buzilgan
 * JSON va **notanish tur** testlar bilan qoplanadi.
 *
 * Dastlab `org.json` ishlatilgandi — u **Android sinfi**, JVM testida stub bo'lib
 * `NullPointerException` berdi va butun parser sinovsiz qolardi. Endi Moshi bilan
 * oddiy `Map` ga o'qiladi: bitta implementatsiya ham ilovada, ham testda.
 */
class RealtimeParser(moshi: Moshi = Moshi.Builder().build()) {

    private val mapAdapter: JsonAdapter<Map<String, Any?>> = moshi.adapter(
        Types.newParameterizedType(Map::class.java, String::class.java, Any::class.java),
    )

    /**
     * @return hodisa yoki `null` — xabar umuman o'qib bo'lmasa (buzilgan JSON, `type` yo'q).
     *         `null` ilova uchun "e'tiborsiz qoldir" degani, crash emas.
     */
    fun parse(raw: String): RealtimeEvent? {
        val root = runCatching { mapAdapter.fromJson(raw) }.getOrNull() ?: return null
        val type = (root["type"] as? String)?.takeIf { it.isNotBlank() } ?: return null
        val payload = root["payload"] as? Map<*, *>

        return when (type) {
            TYPE_WAITING_REQUEST -> {
                // ID'siz so'rovni qabul qilib bo'lmaydi — ko'rsatish chalg'ituvchi bo'lardi.
                val id = payload.str("request_id") ?: return null
                RealtimeEvent.WaitingRoomRequest(
                    requestId = id,
                    lessonId = payload.str("lesson_id").orEmpty(),
                    requesterName = payload.str("requester_name") ?: DEFAULT_GUEST_NAME,
                    createdAt = payload.str("created_at"),
                )
            }

            TYPE_WAITING_ADMITTED -> RealtimeEvent.WaitingRoomAdmitted(token = payload.toRoomToken())

            TYPE_WAITING_REJECTED -> RealtimeEvent.WaitingRoomRejected

            TYPE_NOTIFICATION -> RealtimeEvent.Notification(
                id = payload.str("id"),
                title = payload.str("title"),
                // Backend `body` yoki `message` yuborishi mumkin — ikkalasi ham qabul qilinadi.
                body = payload.str("body") ?: payload.str("message"),
                // Payload = butun `entity.Notification`, shuning uchun bu maydonlar
                // ham keladi va bildirishnomani ro'yxatga qayta so'rovsiz qo'shishga
                // yetadi (`RealtimeEvent.Notification.toModel`).
                type = payload.str("type"),
                lessonId = payload.str("lesson_id"),
                createdAt = payload.str("created_at"),
            )

            else -> RealtimeEvent.Unknown(type)
        }
    }

    /** Bo'sh satr "qiymat yo'q" bilan bir xil — UI'da bo'sh qator chiqmasin. */
    private fun Map<*, *>?.str(key: String): String? =
        (this?.get(key) as? String)?.trim()?.takeIf { it.isNotEmpty() }

    private fun Map<*, *>?.toRoomToken(): RoomToken? {
        val token = str("token") ?: return null
        return RoomToken(
            token = token,
            wsUrl = str("ws_url").orEmpty(),
            roomName = str("room_name").orEmpty(),
            identity = str("identity").orEmpty(),
            role = str("role").orEmpty(),
        )
    }

    private companion object {
        // Manba: backend/internal/infrastructure/websocket/{notify,waitingroom}.go
        const val TYPE_WAITING_REQUEST = "waiting_room.request"
        const val TYPE_WAITING_ADMITTED = "waiting_room.admitted"
        const val TYPE_WAITING_REJECTED = "waiting_room.rejected"
        const val TYPE_NOTIFICATION = "notification"

        const val DEFAULT_GUEST_NAME = "O'quvchi"
    }
}
