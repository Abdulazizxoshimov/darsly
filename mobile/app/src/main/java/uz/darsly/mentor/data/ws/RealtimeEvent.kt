package uz.darsly.mentor.data.ws

import uz.darsly.mentor.data.api.RoomToken

/**
 * Serverdan real-time keladigan hodisalar (M27 va undan keyingi ishlar).
 *
 * ## Protokol (backend manbasi — taxmin emas)
 * `internal/infrastructure/websocket/notify.go` · `waitingroom.go`:
 * ```json
 * { "type": "waiting_room.request", "room": "", "payload": {...}, "created_at": "..." }
 * ```
 * Web klienti ham **aynan shu** konvertni o'qiydi (`frontend/src/lib/ws.js`) — ikkala
 * klient bir xil protokolda bo'lishi shart (mezon D-8 / M29).
 *
 * ## Nega `Unknown` bor
 * Backend yangi hodisa qo'shsa, **eski ilova yiqilmasligi** kerak. Notanish tur jimgina
 * `Unknown` bo'lib o'tadi va jurnalga tushadi.
 */
sealed interface RealtimeEvent {

    /** Kutish xonasiga yangi so'rov (mentorga). */
    data class WaitingRoomRequest(
        val requestId: String,
        val lessonId: String,
        val requesterName: String,
        val createdAt: String?,
    ) : RealtimeEvent

    /** So'rov qabul qilindi — guestga LiveKit tokeni bilan (mobil ilova uchun kelmaydi). */
    data class WaitingRoomAdmitted(val token: RoomToken?) : RealtimeEvent

    /** So'rov rad etildi. */
    data object WaitingRoomRejected : RealtimeEvent

    /**
     * Bildirishnoma (dars eslatmasi va boshqalar).
     *
     * Backend WS payload'i sifatida **butun** `entity.Notification` ni yuboradi
     * (`usecase/notification/notification.go:42`), shuning uchun jonli kelgan
     * xabarni ro'yxatga qo'shish uchun qo'shimcha `GET /notifications` kerak
     * emas — [toModel] shu ishni bajaradi.
     *
     * Maydonlar `null` bo'la oladi: eski server yoki kutilmagan payload shakli
     * "bildirishnoma keldi" faktini yo'qotmasligi kerak.
     */
    data class Notification(
        val id: String?,
        val title: String?,
        val body: String?,
        val type: String? = null,
        val lessonId: String? = null,
        val createdAt: String? = null,
    ) : RealtimeEvent {

        /**
         * Ro'yxatga qo'shsa bo'ladigan modelga aylantiradi.
         *
         * `id` bo'lmasa `null` — identifikatorsiz yozuvni ro'yxatga qo'shib
         * bo'lmaydi (o'qilgan deb belgilash ham, dublikatni aniqlash ham
         * imkonsiz). Bunday holatda ekran shunchaki serverdan yangilaydi.
         *
         * Jonli kelgan bildirishnoma **doim o'qilmagan**: server uni endigina
         * yaratdi, ya'ni `read_at` bo'sh.
         */
        fun toModel(): uz.darsly.mentor.data.api.Notification? {
            val realId = id?.takeIf { it.isNotBlank() } ?: return null
            return uz.darsly.mentor.data.api.Notification(
                id = realId,
                type = type ?: "system",
                title = title.orEmpty(),
                body = body.orEmpty(),
                lessonId = lessonId,
                readAt = null,
                createdAt = createdAt,
            )
        }
    }

    /** Notanish tur — ilova yangilanmagan bo'lsa ham ishlashda davom etadi. */
    data class Unknown(val type: String) : RealtimeEvent
}
