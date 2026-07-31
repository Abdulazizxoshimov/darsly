package uz.darsly.mentor.data.livekit

/** Qo'l ko'targan o'quvchi. `at` — ko'tarilgan vaqt (Unix ms), navbat shu bo'yicha. */
data class RaisedHand(val identity: String, val name: String, val at: Long)

/**
 * Ekranda ko'rsatiladigan reaksiya (o'tkinchi — saqlanmaydi).
 *
 * [at] — kelgan vaqt (Unix ms). `0` bo'lsa reaksiya muddatsiz (eski chaqiruv
 * joylari va testlar uchun); real oqimda u DOIM to'ldiriladi va shu tufayli
 * eskirgan reaksiya ekrandan o'zi ketadi ([ReactionFeed.TTL_MS]).
 */
data class RoomReaction(val id: Long, val emoji: String, val name: String, val at: Long = 0L)

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
 * Oxirgi reaksiyalar ro'yxati — cheklangan uzunlikda VA cheklangan umrda.
 *
 * Uzunlik cheklovi MAJBURIY: reaksiyalar to'xtovsiz kelishi mumkin va ro'yxat
 * cheksiz o'ssa 90 daqiqalik darsda xotira ham, bildirishnoma matni ham
 * shishib ketardi.
 *
 * ## Nega umr cheklovi ham kerak
 * Reaksiya — **o'tkinchi** signal ("hozir qarsak chalindi"), lekin avval u
 * ekranda YANGISI kelguncha turardi. Amalda bu shuni anglatardi: darsning
 * 5-daqiqasida kelgan 👍 soat oxirigacha sahna burchagida osilib turardi va
 * ustoz uni HOZIRGI reaksiya deb o'qirdi. Bu shunchaki chiroyli emaslik emas,
 * yolg'on ma'lumot: server tomonda reaksiya umuman saqlanmaydi.
 */
object ReactionFeed {
    const val MAX = 5

    /**
     * Reaksiya ekranda qancha turadi.
     *
     * 6 soniya — ustoz ekran ulashishdan qaytib qarashga ulguradigan, lekin
     * "hozir" degan ma'noni yo'qotmaydigan oraliq (web'dagi animatsiya umri
     * ham shu tartibda).
     */
    const val TTL_MS = 6_000L

    fun add(current: List<RoomReaction>, reaction: RoomReaction): List<RoomReaction> =
        (listOf(reaction) + prune(current, reaction.at)).take(MAX)

    /**
     * Muddati o'tganlarini olib tashlaydi.
     *
     * Vaqtsiz (`at <= 0`) yozuvlar TEGILMAYDI: ular vaqt manbai bo'lmagan
     * joydan kelgan va ularni "cheksiz eski" deb hisoblash butun ro'yxatni
     * jimgina tozalab yuborardi.
     *
     * O'zgarish bo'lmasa AYNI ro'yxat qaytadi — ortiqcha rekompozitsiya yo'q
     * (`HandQueue.apply` bilan bir naqsh).
     */
    fun prune(current: List<RoomReaction>, now: Long): List<RoomReaction> {
        if (current.isEmpty()) return current
        val left = current.filter { it.at <= 0L || now - it.at < TTL_MS }
        return if (left.size == current.size) current else left
    }
}
