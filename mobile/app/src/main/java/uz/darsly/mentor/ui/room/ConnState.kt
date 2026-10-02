package uz.darsly.mentor.ui.room

/**
 * Xona ulanish holati (M7).
 *
 * Avval `RoomUiState.connState` **satr** edi (`"connected"`) va uch faylda
 * (`RoomViewModel`, `ReconnectController`, `RoomScreen`) literal bilan
 * solishtirilardi — bitta imlo xatosi kompilyatsiyada emas, darsda
 * ("tugmalar o'chiq qoldi") ko'rinardi.
 *
 * SDK ro'yxati (`Room.State`, livekit-android 2.27.0): CONNECTING · CONNECTED ·
 * DISCONNECTED · RECONNECTING. Notanish nom [DISCONNECTED] ga tushadi — SDK
 * yangi qiymat qo'shsa ilova yiqilmaydi.
 */
enum class ConnState {
    DISCONNECTED,
    CONNECTING,
    CONNECTED,
    RECONNECTING,
    ;

    /** Xonadamiz (ulangan yoki vaqtincha uzilgan) — dars davom etyapti. */
    val inRoom: Boolean get() = this == CONNECTED || this == RECONNECTING

    companion object {
        fun fromSdk(name: String?): ConnState = when (name) {
            "CONNECTING" -> CONNECTING
            "CONNECTED" -> CONNECTED
            "RECONNECTING" -> RECONNECTING
            else -> DISCONNECTED
        }
    }
}
