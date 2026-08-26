package uz.darsly.mentor.ui.room

/**
 * Lokal yozuv tugmasi va to'xtatish orkestratsiyasining **sof** qarorlari.
 *
 * ## Nega ajratildi
 * Yozuv EKRANni yozadi — ekran ulashilmagan bo'lsa boshlab bo'lmaydi. Bu shart
 * `RoomViewModel.toggleRecording` ichida edi va sinovsiz: agar u tushib qolsa,
 * ustoz ekran ulashmasdan "Record" bosardi va yozuv jimgina yaroqsiz (bo'sh
 * ekranli) chiqardi. To'xtatishda esa yozuv ID yoki fayl yo'q bo'lsa yuklamaslik
 * (va bo'sh faylni o'chirish) kerak — aks holda `null` ID bilan yuklash urinishi
 * yoki yetim fayl qolardi.
 */
object RecordControl {

    /** [toggle] natijasi — ViewModel qaysi amalni bajarishini aytadi. */
    enum class Action {
        /** Yozuv ketyapti — to'xtatib yuklaymiz. */
        STOP,

        /** Ekran ulashilgan — yozuvni boshlaymiz. */
        START,

        /** Ekran ulashilmagan — boshlab bo'lmaydi, ustozga hint. */
        HINT_NO_SCREEN,
    }

    fun toggle(recording: Boolean, screenSharing: Boolean): Action = when {
        recording -> Action.STOP
        screenSharing -> Action.START
        else -> Action.HINT_NO_SCREEN
    }

    /**
     * To'xtatilgan yozuvni yuklash mumkinmi.
     *
     * Yozuv ID (`local-start` bergani) va fayl IKKALASI ham bo'lishi shart. Biri
     * yo'q bo'lsa yuklash ma'nosiz — bunday holda fayl (bo'lsa) yetim qolmasligi
     * uchun o'chiriladi (chaqiruvchi buni [deleteOrphan] bilan biladi).
     */
    fun canUpload(recordingId: String?, hasFile: Boolean): Boolean =
        recordingId != null && hasFile

    /** Yuklab bo'lmaydigan, lekin diskda qolgan faylni o'chirish kerakmi. */
    fun deleteOrphan(recordingId: String?, hasFile: Boolean): Boolean =
        !canUpload(recordingId, hasFile) && hasFile

    /** Yozuv davomiyligi (sekund) — hech bo'lmasa 1 (0-sekundli fayl bema'ni). */
    fun durationSec(startMs: Long, nowMs: Long): Int =
        ((nowMs - startMs) / 1000).toInt().coerceAtLeast(1)
}
