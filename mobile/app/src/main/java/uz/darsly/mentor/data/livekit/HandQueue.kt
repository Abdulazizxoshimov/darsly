package uz.darsly.mentor.data.livekit

/** Qo'l ko'targan o'quvchi. `at` — ko'tarilgan vaqt (Unix ms), navbat shu bo'yicha. */
data class RaisedHand(val identity: String, val name: String, val at: Long)

/** Ekranda ko'rsatiladigan reaksiya (o'tkinchi — saqlanmaydi). */
data class RoomReaction(val id: Long, val emoji: String, val name: String)

/**
 * Qo'l navbati — **sof** holat o'zgartirishlari.
 *
 * ## Nega tartib `at` bo'yicha, kelish tartibi bo'yicha emas
 * Navbat tartibi mahsulot qoidasi: "kim birinchi so'radi". Uni SERVER belgilaydi
 * (`roomstate.State` ham shu bo'yicha tartiblaydi) va uch klientda bir xil
 * ko'rinishi kerak. Xabarlar kelish tartibiga tayanish esa qayta ulanish yoki
 * kech qo'shilishda boshqa natija berardi — ustoz va o'quvchi turli navbat
 * ko'rgan bo'lardi.
 *
 * Teng `at` da `identity` bo'yicha — tartib deterministik bo'lsin.
 */
object HandQueue {

    fun apply(current: List<RaisedHand>, signal: RoomSignal): List<RaisedHand> = when (signal) {
        is RoomSignal.LowerAll -> if (current.isEmpty()) current else emptyList()

        is RoomSignal.Hand -> if (signal.raised) {
            val existing = current.find { it.identity == signal.identity }
            // TAKRORIY ko'tarish navbatdagi o'rinni buzmaydi: eski `at` saqlanadi.
            // (Server ham shunday qiladi — ikkalasi kelishmasa navbat "sakrab" ketardi.)
            val at = existing?.at ?: signal.at
            if (existing != null && existing.name == signal.name) {
                current
            } else {
                (current.filter { it.identity != signal.identity } +
                    RaisedHand(signal.identity, signal.name, at)).sortedWith(ORDER)
            }
        } else {
            if (current.none { it.identity == signal.identity }) current
            else current.filter { it.identity != signal.identity }
        }

        else -> current
    }

    /** Serverdan kelgan to'liq holat (`GET /rooms/:id/state`) bilan almashtirish. */
    fun replaceAll(hands: List<RaisedHand>): List<RaisedHand> = hands.sortedWith(ORDER)

    private val ORDER = compareBy<RaisedHand>({ it.at }, { it.identity })
}

/**
 * Oxirgi reaksiyalar ro'yxati — cheklangan uzunlikda.
 *
 * Cheklov MAJBURIY: reaksiyalar to'xtovsiz kelishi mumkin va ro'yxat cheksiz
 * o'ssa 90 daqiqalik darsda xotira ham, bildirishnoma matni ham shishib ketardi.
 */
object ReactionFeed {
    const val MAX = 5

    fun add(current: List<RoomReaction>, reaction: RoomReaction): List<RoomReaction> =
        (listOf(reaction) + current).take(MAX)
}
