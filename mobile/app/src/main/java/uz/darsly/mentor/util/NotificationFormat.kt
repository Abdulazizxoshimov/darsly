package uz.darsly.mentor.util

import uz.darsly.mentor.data.api.Notification

/**
 * Bildirishnomalar ro'yxatining **sof** formatlash va birlashtirish mantiqi.
 *
 * Compose'dan ajratilgani ataylab: "hozirgina / 5 daqiqa oldin" matni va jonli
 * (WebSocket) kelgan xabarni ro'yxatga qo'shish qoidalari JVM unit testida
 * sinaladi — emulyator kerak emas.
 */
object NotificationFormat {

    /** Bildirishnoma turlari — `backend/internal/entity/notification.go` konstantalari. */
    const val TYPE_LESSON_REMINDER = "lesson_reminder"
    const val TYPE_WAITING_ROOM = "waiting_room"
    const val TYPE_SYSTEM = "system"

    /** Tur yorlig'i. Notanish tur o'zi ko'rsatilmaydi — umumiy nom beriladi. */
    fun typeLabel(type: String): String = when (type) {
        TYPE_LESSON_REMINDER -> "Dars eslatmasi"
        TYPE_WAITING_ROOM -> "Kutish xonasi"
        TYPE_SYSTEM -> "Tizim"
        else -> "Bildirishnoma"
    }

    /**
     * "Hozirgina" · "5 daqiqa oldin" · "3 soat oldin" · "Kecha 14:30" · "27-iyul, 14:30".
     *
     * Bildirishnomalar uchun **nisbiy** vaqt ishlatiladi (darslar ro'yxatidagi
     * mutlaq vaqtdan farqli): "5 daqiqa oldin" foydalanuvchi savoliga ("yangimi?")
     * to'g'ridan-to'g'ri javob beradi, "14:32" esa uni hisoblashga majbur qiladi.
     * Bir kundan oshganda mutlaq sanaga o'tiladi — "312 soat oldin" ma'nosiz.
     */
    fun relativeTime(
        iso: String?,
        nowMillis: Long = System.currentTimeMillis(),
    ): String? {
        val then = LessonFormat.epochOrNull(iso) ?: return null
        val diff = nowMillis - then
        // Kelajakdagi vaqt (qurilma soati orqada) — "−3 daqiqa oldin" demaymiz.
        if (diff < 0) return LessonFormat.scheduleLabel(iso)
        val minutes = diff / 60_000
        val hours = minutes / 60
        return when {
            minutes < 1 -> "Hozirgina"
            minutes < 60 -> "$minutes daqiqa oldin"
            hours < 24 -> "$hours soat oldin"
            else -> LessonFormat.scheduleLabel(iso)
        }
    }

    /**
     * Jonli (WS) kelgan bildirishnomani ro'yxatga qo'shadi.
     *
     * ## Nega alohida funksiya
     * Uch holat bor va uchalasi ham xato qilish oson:
     *  1. **Dublikat** — WS xabari va `GET /notifications` javobi bir vaqtda
     *     kelishi mumkin (ustoz ekranni ochgan lahzada eslatma yuborilsa).
     *     ID bo'yicha tekshiruv bo'lmasa bitta bildirishnoma ikki marta chiqardi.
     *  2. **Tartib** — yangi xabar eng tepada bo'lishi kerak (backend ham
     *     `created_at DESC` beradi).
     *  3. **Eski nusxa** — dublikat topilsa YANGI nusxa qoladi: WS xabari
     *     serverning eng so'nggi holati.
     */
    fun merge(existing: List<Notification>, incoming: Notification): List<Notification> =
        listOf(incoming) + existing.filterNot { it.id == incoming.id }

    /** O'qilmaganlar soni — nishon (badge) uchun. */
    fun unreadCount(items: List<Notification>): Int = items.count { it.isUnread }

    /**
     * Nishondagi matn: 99 dan oshsa "99+".
     *
     * Uch xonali son doira nishonga sig'maydi va maketni buzadi; aniq son esa
     * bu yerda muhim emas — "juda ko'p" degan ma'no yetarli.
     */
    fun badgeLabel(count: Int): String? = when {
        count <= 0 -> null
        count > 99 -> "99+"
        else -> count.toString()
    }
}
