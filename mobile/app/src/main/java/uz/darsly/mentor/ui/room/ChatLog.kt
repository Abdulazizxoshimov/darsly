package uz.darsly.mentor.ui.room

/**
 * Xona chat ro'yxatining **sof** o'zgartirishlari (M?/№15 · №6).
 *
 * ## Nega ajratildi
 * Chat ro'yxati bir vaqtda IKKI manbadan to'ladi: bizning optimistik
 * qo'shishimiz (`appendOwnMessage`) va serverning data-channel echo'si
 * (`RoomSignal.Chat`). Ikkalasi bir xabarni ikki marta ko'rsatmasligi
 * (ID bo'yicha dedup), ro'yxat cheksiz o'smasligi ([MAX] bilan kesish) va
 * o'chirilgan xabar darhol ketishi kerak. Bu qoidalar `RoomViewModel` ichida
 * uch joyda takrorlangan va sinovsiz edi — endi bitta sof joyda.
 */
object ChatLog {

    /**
     * Xotirada saqlanadigan eng ko'p xabar.
     *
     * Cheklov MAJBURIY: 90 daqiqalik darsda chat cheksiz o'ssa ham ro'yxat, ham
     * Compose render'i og'irlashadi. To'liq tarix serverda qoladi.
     */
    const val MAX = 200

    /** [add] natijasi: yangi ro'yxat va o'qilmaganlar o'zgarishi. */
    data class Result(val chat: List<ChatMessageUi>, val unreadDelta: Int)

    /**
     * Xabar qo'shadi (o'zimizniki ham, serverdan kelgani ham).
     *
     *  · ID allaqachon bo'lsa — hech nima o'zgarmaydi (dublikat kesiladi);
     *  · o'qilmaganlar faqat O'ZGANING xabarida ortadi (`self == false`);
     *  · ro'yxat [MAX] gacha kesiladi (oxiridan — eng yangilari qoladi).
     */
    fun add(current: List<ChatMessageUi>, msg: ChatMessageUi): Result {
        if (current.any { it.id == msg.id }) return Result(current, 0)
        val delta = if (msg.self) 0 else 1
        return Result((current + msg).takeLast(MAX), delta)
    }

    /**
     * Xabarni ID bo'yicha o'chiradi (moderatsiya). Topilmasa AYNI ro'yxat qaytadi —
     * ortiqcha rekompozitsiya yo'q.
     */
    fun delete(current: List<ChatMessageUi>, id: String): List<ChatMessageUi> {
        val left = current.filterNot { it.id == id }
        return if (left.size == current.size) current else left
    }
}
