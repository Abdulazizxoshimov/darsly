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

    /** Bildirishnoma (dars eslatmasi va boshqalar). */
    data class Notification(
        val id: String?,
        val title: String?,
        val body: String?,
    ) : RealtimeEvent

    /** Notanish tur — ilova yangilanmagan bo'lsa ham ishlashda davom etadi. */
    data class Unknown(val type: String) : RealtimeEvent
}
