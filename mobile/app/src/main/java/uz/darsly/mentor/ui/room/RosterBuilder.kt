package uz.darsly.mentor.ui.room

/**
 * Moderatsiya paneli ro'yxatini serverdagi ishtirokchilardan yig'ish — **sof**.
 *
 * ## Nega ajratildi
 * `RoomViewModel.refreshRoster` bu ro'yxatni har moderatsiya amalidan keyin
 * qayta yig'adi va uch nozik qoida bor: (1) ustozning O'ZI ro'yxatda ko'rinmasin
 * (o'zini mute/chiqarish tugmasi bema'ni), (2) qo'l ko'targanlar shu paytdagi
 * navbatdan belgilansin, (3) ismsiz o'quvchi `identity` bilan ko'rinsin. Bu
 * qoidalar jimgina buziladigan turdan (masalan `me` filtri tushib qolsa ustoz
 * o'zini chiqarib yuborishi mumkin), shuning uchun JVM testida qulflandi.
 */
object RosterBuilder {

    /**
     * @param items serverdan kelgan ishtirokchilar (istalgan turdan — kirish akssessorlar bilan)
     * @param me ustozning o'z identity'si (ro'yxatdan chiqariladi)
     * @param raised hozir qo'l ko'targanlar identity to'plami
     */
    fun <T> build(
        items: List<T>,
        me: String?,
        raised: Set<String>,
        identity: (T) -> String,
        name: (T) -> String,
        audioMuted: (T) -> Boolean,
    ): List<RosterEntry> =
        items
            .filter { identity(it) != me }
            .map {
                val id = identity(it)
                RosterEntry(
                    identity = id,
                    name = name(it).ifBlank { id },
                    audioMuted = audioMuted(it),
                    handRaised = id in raised,
                )
            }
}
