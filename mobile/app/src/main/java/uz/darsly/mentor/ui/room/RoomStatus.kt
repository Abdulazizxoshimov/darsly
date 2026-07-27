package uz.darsly.mentor.ui.room

/**
 * Aloqa sifati va darsning tugash sababi — **sof** mantiq (M16 · B-6).
 *
 * ## Nega LiveKit turlari bu yerda ishlatilmaydi
 * SDK enum'lari (`ConnectionQuality`, `DisconnectReason`) Android artefaktida keladi va
 * JVM unit testida yaratib bo'lmaydi. Shuning uchun chegarada (`RoomViewModel` ning
 * hodisa kuzatuvchisida) SDK qiymati **nomi bo'yicha** shu yerdagi turlarga
 * o'giriladi — natijada butun matn tanlash mantiqi testlar ostida qoladi.
 *
 * SDK ro'yxati (livekit-android 2.27.0 dan tekshirilgan):
 *  · `ConnectionQuality`: EXCELLENT · GOOD · POOR · LOST · UNKNOWN
 *  · `DisconnectReason`: CLIENT_INITIATED · ROOM_DELETED · ROOM_CLOSED · PARTICIPANT_REMOVED ·
 *    SERVER_SHUTDOWN · CONNECTION_TIMEOUT · MEDIA_FAILURE · SIGNAL_CLOSE · JOIN_FAILURE ·
 *    DUPLICATE_IDENTITY · STATE_MISMATCH · MIGRATION · USER_UNAVAILABLE · USER_REJECTED ·
 *    SIP_TRUNK_FAILURE · UNKNOWN_REASON
 */
object RoomStatus {

    /** Aloqa sifati (M16 indikatori). */
    enum class LinkQuality { EXCELLENT, GOOD, POOR, LOST, UNKNOWN }

    /** Dars nima uchun tugadi (B-6 — resurslarni bo'shatish va rost xabar). */
    enum class EndReason {
        /** Ustozning o'zi chiqdi — xabar kerak emas. */
        SELF,

        /** Xona serverda yopildi (`/lessons/:id/end` yoki boshqa qurilmadan). */
        HOST_ENDED,

        /** Ishtirokchi chiqarib yuborildi. */
        REMOVED,

        /** Internet uzildi / media o'tmadi. */
        NETWORK,

        /** Server tomonidagi nosozlik. */
        SERVER,

        UNKNOWN,
    }

    fun qualityOf(sdkName: String?): LinkQuality = when (sdkName) {
        "EXCELLENT" -> LinkQuality.EXCELLENT
        "GOOD" -> LinkQuality.GOOD
        "POOR" -> LinkQuality.POOR
        "LOST" -> LinkQuality.LOST
        else -> LinkQuality.UNKNOWN
    }

    /**
     * Ekrandagi aloqa yozuvi. `null` — ko'rsatishga arzimaydi (hammasi joyida).
     *
     * Zoom xulqi: sifat yaxshi bo'lganda indikator bezovta qilmaydi, faqat
     * yomonlashganda ko'rinadi. Qayta ulanish esa HAR DOIM ko'rsatiladi —
     * ustoz jim qolgan darsni "buzilib qoldi" deb o'ylamasligi kerak.
     */
    fun linkLabel(quality: LinkQuality, reconnecting: Boolean): String? = when {
        reconnecting -> "Qayta ulanmoqda…"
        quality == LinkQuality.LOST -> "Aloqa yo'q"
        quality == LinkQuality.POOR -> "Aloqa yomon — video sifati pasayishi mumkin"
        else -> null
    }

    /** Indikator ogohlantiruvchi rangda bo'lsinmi. */
    fun isWarning(quality: LinkQuality, reconnecting: Boolean): Boolean =
        reconnecting || quality == LinkQuality.LOST || quality == LinkQuality.POOR

    fun endReasonOf(sdkName: String?): EndReason = when (sdkName) {
        "CLIENT_INITIATED" -> EndReason.SELF
        "ROOM_DELETED", "ROOM_CLOSED" -> EndReason.HOST_ENDED
        "PARTICIPANT_REMOVED" -> EndReason.REMOVED
        "CONNECTION_TIMEOUT", "MEDIA_FAILURE", "SIGNAL_CLOSE" -> EndReason.NETWORK
        "SERVER_SHUTDOWN", "JOIN_FAILURE", "STATE_MISMATCH", "DUPLICATE_IDENTITY" -> EndReason.SERVER
        else -> EndReason.UNKNOWN
    }

    /**
     * Dars tugagani haqidagi xabar. `null` — ustozning o'zi chiqqan, xabar ortiqcha.
     *
     * Har bir matn **nima bo'lgani va endi nima qilish kerakligini** aytadi:
     * "uzildi" degan quruq so'z ustozga yordam bermaydi.
     */
    fun endMessage(reason: EndReason): String? = when (reason) {
        EndReason.SELF -> null
        EndReason.HOST_ENDED -> "Dars yakunlandi. Ekran ulashish to'xtatildi."
        EndReason.REMOVED -> "Siz xonadan chiqarildingiz."
        EndReason.NETWORK ->
            "Aloqa uzildi va qayta ulanib bo'lmadi. Internetni tekshirib darsni qayta boshlang."
        EndReason.SERVER -> "Server bilan aloqa uzildi. Birozdan so'ng qayta urinib ko'ring."
        EndReason.UNKNOWN -> "Dars uzildi. Qayta boshlash mumkin."
    }
}
