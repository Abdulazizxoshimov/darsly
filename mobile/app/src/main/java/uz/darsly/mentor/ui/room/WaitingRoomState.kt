package uz.darsly.mentor.ui.room

import uz.darsly.mentor.data.api.WaitingRoomRequest
import uz.darsly.mentor.data.ws.RealtimeEvent
import uz.darsly.mentor.util.LessonFormat

/**
 * Kutish xonasi ro'yxatining **sof** mantiqi (M27).
 *
 * ## Nega alohida va nega testlanadi
 * Bu ro'yxat **ikki manbadan** to'ladi va ular bir-birini quvib yetadi:
 *  · `GET /lessons/:id/waitingroom` — ekran ochilgandagi suratga olish;
 *  · WebSocket `waiting_room.request` — jonli kelgan yangi so'rovlar.
 *
 * Backend WS orqali **snapshot ham** yuboradi (`DeliverPendingSnapshot` —
 * mentor ulanganda barcha kutayotganlarni qayta yuboradi), ya'ni bitta so'rov
 * ikki yo'ldan kelishi deyarli kafolatlangan. Dublikatga qarshi himoya
 * bo'lmasa ustoz bitta o'quvchini ikki marta ko'rardi va ikkinchi "Qabul
 * qilish" 409 bilan qaytardi.
 *
 * Bundan tashqari WS xabari **boshqa darsning** so'rovi bo'lishi mumkin: kanal
 * ustozga bog'langan, xonaga emas. Filtrsiz ustoz A darsida turib B darsining
 * o'quvchisini qabul qilib yuborardi.
 */
object WaitingRoomState {

    /**
     * WS hodisasini model obyektiga aylantiradi.
     *
     * WS payload'ida `id` emas, `request_id` keladi (`waitingroom.go:68`) —
     * shu farq AYNAN shu joyda, bir marta yopiladi.
     *
     * `status` majburan `pending`: WS orqali faqat yangi (hal qilinmagan)
     * so'rovlar keladi.
     */
    fun toRequest(event: RealtimeEvent.WaitingRoomRequest): WaitingRoomRequest =
        WaitingRoomRequest(
            id = event.requestId,
            lessonId = event.lessonId,
            requesterName = event.requesterName,
            status = "pending",
            createdAt = event.createdAt,
        )

    /**
     * Yangi so'rovni ro'yxatga qo'shadi.
     *
     * · **Boshqa darsniki** bo'lsa e'tiborsiz qoldiriladi ([lessonId] mos kelmasa);
     *   so'rovda dars ID'si bo'lmasa (eski server) qabul qilinadi — jimgina
     *   tashlab yuborishdan ko'ra ko'rsatgan yaxshiroq.
     * · **Dublikat** bo'lsa yangi nusxa eskisining O'RNIGA qo'yiladi, lekin
     *   ro'yxatdagi **joyi saqlanadi**: ustoz ro'yxatni o'qiyotganda qatorlar
     *   sakrab ketmasligi kerak (u "Qabul qilish"ni noto'g'ri qatorda bosishi mumkin).
     * · Yangi so'rov OXIRIGA qo'shiladi — navbat tartibi: birinchi kelgan
     *   birinchi qabul qilinadi.
     */
    fun add(
        existing: List<WaitingRoomRequest>,
        incoming: WaitingRoomRequest,
        lessonId: String,
    ): List<WaitingRoomRequest> {
        if (incoming.lessonId.isNotBlank() && incoming.lessonId != lessonId) return existing
        val index = existing.indexOfFirst { it.id == incoming.id }
        return if (index >= 0) {
            existing.toMutableList().also { it[index] = incoming }
        } else {
            existing + incoming
        }
    }

    /** Qaror qabul qilingan so'rovni ro'yxatdan olib tashlaydi. */
    fun remove(existing: List<WaitingRoomRequest>, requestId: String): List<WaitingRoomRequest> =
        existing.filterNot { it.id == requestId }

    /**
     * Serverdan kelgan suratni ro'yxat bilan birlashtiradi.
     *
     * Server **haqiqat manbai**, lekin surat olingandan keyin (so'rov ketayotgan
     * paytda) WS orqali kelgan so'rovlar yo'qolmasligi kerak — shuning uchun
     * ular oxiriga qo'shiladi. Aks holda o'quvchi ustoz ekranini ochgan aynan
     * o'sha lahzada kirsa, u ro'yxatda umuman ko'rinmasdi.
     *
     * Faqat `pending` qoladi: hal qilingan so'rovni ko'rsatish ma'nosiz va
     * uning tugmalari 409 berardi.
     */
    fun merge(
        snapshot: List<WaitingRoomRequest>,
        live: List<WaitingRoomRequest>,
    ): List<WaitingRoomRequest> {
        val known = snapshot.map { it.id }.toSet()
        return (snapshot + live.filterNot { it.id in known })
            .filter { it.status == "pending" }
    }

    /**
     * Ro'yxat tartibi: eng uzoq kutgani birinchi.
     *
     * Vaqti noma'lum so'rov OXIRDA qoladi (`Long.MAX_VALUE`) — u "yangi kelgan"
     * deb hisoblanadi. Buning aksi (nol qo'yish) uni ro'yxat boshiga chiqarardi
     * va ustoz eng uzoq kutgan o'quvchi deb o'ylardi.
     */
    fun sortForDisplay(requests: List<WaitingRoomRequest>): List<WaitingRoomRequest> =
        requests.sortedBy { LessonFormat.epochOrNull(it.createdAt) ?: Long.MAX_VALUE }

    /**
     * "3 daqiqa kutmoqda" · "Hozirgina keldi".
     *
     * Kutish vaqti ustoz uchun qaror mezoni: uzoq kutgan o'quvchi darsdan
     * qolayotgan bo'ladi. Vaqt noma'lum bo'lsa `null` — taxmin qilmaymiz.
     */
    fun waitingLabel(
        request: WaitingRoomRequest,
        nowMillis: Long = System.currentTimeMillis(),
    ): String? {
        val since = LessonFormat.epochOrNull(request.createdAt) ?: return null
        val minutes = (nowMillis - since) / 60_000
        return when {
            minutes < 1 -> "Hozirgina keldi"
            minutes < 60 -> "$minutes daqiqa kutmoqda"
            else -> "${minutes / 60} soatdan ortiq kutmoqda"
        }
    }

    /** Sarlavhadagi son: 0 bo'lsa panel umuman ko'rsatilmaydi. */
    fun title(count: Int): String = when (count) {
        1 -> "1 o'quvchi kutmoqda"
        else -> "$count o'quvchi kutmoqda"
    }
}
