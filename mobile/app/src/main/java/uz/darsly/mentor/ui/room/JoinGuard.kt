package uz.darsly.mentor.ui.room

/**
 * Xonaga ulanish urinishlarini boshqaruvchi kichik holat mashinasi.
 *
 * ## Nega kerak
 * Avval [RoomViewModel] da qorovul shunday edi:
 * `if (_state.value.connecting || session != null) return`.
 * `connect()` xato bersa `connecting` `false` bo'lardi, lekin `session` maydoni
 * **null qilinmasdi** — natijada qorovul qayta urinishni butunlay bloklardi:
 * ustoz "Qayta urinish"ni bosaverardi, ilova esa jimgina hech narsa qilmasdi.
 * Bundan tashqari LiveKit `Room` (PeerConnection + audio qurilma moduli) "Chiqish"
 * bosilmaguncha xotirada osilib qolardi.
 *
 * Bu yerda holat ATAYLAB alohida klassda: LiveKit/Android'siz, ya'ni JVM unit
 * testlarida to'liq sinaladi.
 */
class JoinGuard {

    enum class State {
        /** Ulanmagan — yangi urinish mumkin. */
        IDLE,

        /** Ulanish jarayonida — takroriy urinish rad etiladi. */
        CONNECTING,

        /** Ulangan va sessiya ushlab turilibdi — qayta ulanish kerak emas. */
        ATTACHED,
    }

    @Volatile
    var state: State = State.IDLE
        private set

    /** Yangi urinish boshlash mumkinmi? Mumkin bo'lsa holatni band qiladi. */
    @Synchronized
    fun tryBegin(): Boolean {
        if (state != State.IDLE) return false
        state = State.CONNECTING
        return true
    }

    /** Ulanish muvaffaqiyatli — sessiya ushlab turilibdi. */
    @Synchronized
    fun onAttached() {
        state = State.ATTACHED
    }

    /**
     * Ulanish muvaffaqiyatsiz — resurslar bo'shatildi.
     * KRITIK: holat [State.IDLE] ga qaytadi, aks holda "Qayta urinish" o'lik tugma bo'ladi.
     */
    @Synchronized
    fun onFailed() {
        state = State.IDLE
    }

    /** Darsdan chiqildi — sessiya bo'shatildi. */
    @Synchronized
    fun onReleased() {
        state = State.IDLE
    }

    val isAttached: Boolean get() = state == State.ATTACHED
}
