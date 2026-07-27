package uz.darsly.mentor.data.ws

/**
 * Real-time kanalning yagona nusxasi (process darajasida).
 *
 * Nega singleton: soket ekranlarga emas, **sessiyaga** bog'langan. Ustoz darslar
 * ro'yxatidan xonaga o'tganda kanal uzilmasligi kerak, aks holda o'sha lahzada kelgan
 * kutish so'rovi yo'qolardi.
 *
 * TODO(R2): `Net` bilan birga AppContainer/Hilt ga ko'chirish.
 */
object Realtime {
    val client: RealtimeClient by lazy { RealtimeClient() }
}
